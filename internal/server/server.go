package server

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/ngbcoder/photogramettry/internal/events"
	"github.com/ngbcoder/photogramettry/pkg/features"
	"github.com/ngbcoder/photogramettry/pkg/ingest"
	"github.com/ngbcoder/photogramettry/pkg/matcher"
	"github.com/ngbcoder/photogramettry/pkg/export"
	"github.com/ngbcoder/photogramettry/pkg/mesh"
	"github.com/ngbcoder/photogramettry/pkg/models"
	"github.com/ngbcoder/photogramettry/pkg/mvs"
	"github.com/ngbcoder/photogramettry/pkg/sfm"
	"gonum.org/v1/gonum/mat"
	"github.com/ngbcoder/photogramettry/internal/project"
)

// Config holds server configuration.
type Config struct {
	Port   int
	Input  string
	Output string
	Logger *slog.Logger
}

// Server is the PhotoGen HTTP server.
type Server struct {
	cfg      Config
	log      *slog.Logger
	mux      *http.ServeMux
	httpSrv  *http.Server
	events   *events.Hub
	projects *project.Store
}

// New creates a configured Server.
func New(cfg Config) (*Server, error) {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.Output != "" {
		if err := os.MkdirAll(cfg.Output, 0o755); err != nil {
			return nil, err
		}
	}
	s := &Server{
		cfg:      cfg,
		log:      cfg.Logger,
		mux:      http.NewServeMux(),
		events:   events.NewHub(cfg.Logger),
		projects: project.NewStore(cfg.Output),
	}
	s.routes()
	s.httpSrv = &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           s.withMiddleware(s.mux),
		ReadHeaderTimeout: 10 * time.Second,
	}
	return s, nil
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/events", s.events.ServeSSE)
	s.mux.HandleFunc("GET /api/fs/list", s.handleFSList)
	s.mux.HandleFunc("GET /api/projects", s.handleProjectList)
	s.mux.HandleFunc("POST /api/projects", s.handleProjectCreate)
	s.mux.HandleFunc("GET /api/projects/{id}", s.handleProjectGet)
	s.mux.HandleFunc("PUT /api/projects/{id}", s.handleProjectUpdate)
	s.mux.HandleFunc("POST /api/ingest", s.handleIngest)
	s.mux.HandleFunc("POST /api/features/run", s.handleFeatures)
	s.mux.HandleFunc("POST /api/match/run", s.handleMatch)
	s.mux.HandleFunc("POST /api/sfm/init", s.handleSfMInit)
	s.mux.HandleFunc("POST /api/sfm/run", s.handleSfMRun)
	s.mux.HandleFunc("POST /api/mvs/run", s.handleMVS)
	s.mux.HandleFunc("POST /api/mesh/run", s.handleMesh)
	s.mux.HandleFunc("POST /api/poisson/run", s.handlePoisson)
	s.mux.HandleFunc("POST /api/export", s.handleExport)
	s.mux.Handle("/", http.FileServer(http.Dir(frontendDir())))
}

func frontendDir() string {
	candidates := []string{"frontend/dist", "../frontend/dist", "./web"}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			return c
		}
	}
	return "."
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	return s.httpSrv.ListenAndServe()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpSrv.Shutdown(ctx)
}

func (s *Server) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"time":   time.Now().UTC(),
	})
}

func (s *Server) handleFSList(w http.ResponseWriter, r *http.Request) {
	dir := r.URL.Query().Get("path")
	if dir == "" {
		dir = "."
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	type entry struct {
		Name  string `json:"name"`
		Path  string `json:"path"`
		IsDir bool   `json:"is_dir"`
	}
	out := make([]entry, 0, len(entries))
	for _, e := range entries {
		out = append(out, entry{
			Name:  e.Name(),
			Path:  filepath.Join(dir, e.Name()),
			IsDir: e.IsDir(),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"path": dir, "entries": out})
}

func (s *Server) handleProjectList(w http.ResponseWriter, r *http.Request) {
	projects, err := s.projects.List()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, projects)
}

func (s *Server) handleProjectCreate(w http.ResponseWriter, r *http.Request) {
	var p project.Project
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	created, err := s.projects.Create(p)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleProjectGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	p, err := s.projects.Get(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleProjectUpdate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var p project.Project
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	p.ID = id
	updated, err := s.projects.Update(p)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleIngest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InputDir string `json:"input_dir"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if req.InputDir == "" {
		req.InputDir = s.cfg.Input
	}
	s.events.Publish(events.Event{Type: "stage", Data: map[string]any{"stage": "ingest", "status": "running"}})
	frames, err := ingest.IngestDir(req.InputDir)
	if err != nil {
		s.events.Publish(events.Event{Type: "stage", Data: map[string]any{"stage": "ingest", "status": "error", "error": err.Error()}})
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.events.Publish(events.Event{Type: "stage", Data: map[string]any{"stage": "ingest", "status": "done", "count": len(frames)}})
	writeJSON(w, http.StatusOK, map[string]any{"frames": frames, "count": len(frames)})
}

func (s *Server) handleFeatures(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InputDir string `json:"input_dir"`
		Detector string `json:"detector"`
		Workers  int    `json:"workers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if req.InputDir == "" {
		req.InputDir = s.cfg.Input
	}
	frames, err := ingest.IngestDir(req.InputDir)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	paths := make([]string, len(frames))
	for i, fr := range frames {
		paths[i] = fr.Path
	}
	det, err := features.NewDetector(req.Detector)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	pool := features.NewPool(det, req.Workers)
	s.events.Publish(events.Event{Type: "stage", Data: map[string]any{"stage": "features", "status": "running"}})
	sets, err := pool.Run(r.Context(), paths)
	if err != nil {
		s.events.Publish(events.Event{Type: "stage", Data: map[string]any{"stage": "features", "status": "error", "error": err.Error()}})
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.events.Publish(events.Event{Type: "stage", Data: map[string]any{"stage": "features", "status": "done", "count": len(sets)}})
	writeJSON(w, http.StatusOK, map[string]any{"features": sets, "count": len(sets)})
}

func (s *Server) handleMatch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InputDir string  `json:"input_dir"`
		Detector string  `json:"detector"`
		Workers  int     `json:"workers"`
		Ratio    float64 `json:"ratio"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if req.InputDir == "" {
		req.InputDir = s.cfg.Input
	}
	frames, err := ingest.IngestDir(req.InputDir)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	paths := make([]string, len(frames))
	for i, fr := range frames {
		paths[i] = fr.Path
	}
	det, err := features.NewDetector(req.Detector)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	sets, err := features.NewPool(det, req.Workers).Run(r.Context(), paths)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	m := matcher.NewMatcher(req.Ratio)
	pairs := []models.MatchPair{}
	for i := 0; i+1 < len(frames); i++ {
		pair := m.Match(frames[i].ID, frames[i+1].ID, sets[i], sets[i+1])
		pairs = append(pairs, pair)
	}
	writeJSON(w, http.StatusOK, map[string]any{"pairs": pairs, "count": len(pairs)})
}

func (s *Server) handleSfMInit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InputDir string `json:"input_dir"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if req.InputDir == "" {
		req.InputDir = s.cfg.Input
	}
	frames, err := ingest.IngestDir(req.InputDir)
	if err != nil || len(frames) < 2 {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "need at least 2 images"})
		return
	}
	paths := make([]string, len(frames))
	for i, fr := range frames {
		paths[i] = fr.Path
	}
	det, _ := features.NewDetector("fast")
	sets, err := features.NewPool(det, 4).Run(r.Context(), paths)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	m := matcher.NewMatcher(0.75)
	pair := m.Match(frames[0].ID, frames[1].ID, sets[0], sets[1])

	// normalized keypoints for first two frames
	pts1 := make([][2]float64, len(pair.Matches))
	pts2 := make([][2]float64, len(pair.Matches))
	for i, mc := range pair.Matches {
		pts1[i] = [2]float64{sets[0].Keypoints[mc.QueryIdx].X, sets[0].Keypoints[mc.QueryIdx].Y}
		pts2[i] = [2]float64{sets[1].Keypoints[mc.TrainIdx].X, sets[1].Keypoints[mc.TrainIdx].Y}
	}
	// build K matrices
	k1 := frames[0].Intrinsics.KMatrix
	k2 := frames[1].Intrinsics.KMatrix
	if k1 == nil {
		k1 = matNewK(frames[0].Intrinsics.FocalLengthX, frames[0].Intrinsics.FocalLengthY, frames[0].Intrinsics.PrincipalX, frames[0].Intrinsics.PrincipalY)
	}
	if k2 == nil {
		k2 = matNewK(frames[1].Intrinsics.FocalLengthX, frames[1].Intrinsics.FocalLengthY, frames[1].Intrinsics.PrincipalX, frames[1].Intrinsics.PrincipalY)
	}
	// RANSAC essential
	mask, E := matcher.RANSACEssential(pair, sets[0].Keypoints, sets[1].Keypoints, k1, k2, 1.0, 200)
	if E == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "essential matrix failed"})
		return
	}
	// filter inliers
	inlierPts1 := [][2]float64{}
	inlierPts2 := [][2]float64{}
	for i := range pair.Matches {
		if mask[i] == 1 {
			inlierPts1 = append(inlierPts1, pts1[i])
			inlierPts2 = append(inlierPts2, pts2[i])
		}
	}
	pair.InlierMask = mask
	R, t := sfm.RecoverPose(E, k1, inlierPts1, inlierPts2)

	// triangulate
	P1 := sfm.ProjectionMatrix(k1, matEye3(), matZero3x1())
	P2 := sfm.ProjectionMatrix(k2, R, t)
	cloud := [][3]float64{}
	for i := range inlierPts1 {
		X := sfm.TriangulatePoint(P1, P2, inlierPts1[i], inlierPts2[i])
		cloud = append(cloud, X)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"points": cloud,
		"count":  len(cloud),
		"pose":   map[string]any{"R": matToSlice(R), "t": matToSlice(t)},
	})
}

func matNewK(fx, fy, cx, cy float64) *mat.Dense {
	// local helper to avoid import cycle
	return newKMatrix(fx, fy, cx, cy)
}

func matEye3() *mat.Dense {
	return mat.NewDense(3, 3, []float64{1, 0, 0, 0, 1, 0, 0, 0, 1})
}

func matZero3x1() *mat.Dense {
	return mat.NewDense(3, 1, []float64{0, 0, 0})
}

func newKMatrix(fx, fy, cx, cy float64) *mat.Dense {
	return mat.NewDense(3, 3, []float64{fx, 0, cx, 0, fy, cy, 0, 0, 1})
}

func matToSlice(m *mat.Dense) [][]float64 {
	if m == nil {
		return nil
	}
	r, c := m.Dims()
	out := make([][]float64, r)
	for i := 0; i < r; i++ {
		out[i] = make([]float64, c)
		for j := 0; j < c; j++ {
			out[i][j] = m.At(i, j)
		}
	}
	return out
}

func (s *Server) handleMVS(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InputDir string `json:"input_dir"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if req.InputDir == "" {
		req.InputDir = s.cfg.Input
	}
	frames, err := ingest.IngestDir(req.InputDir)
	if err != nil || len(frames) < 2 {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "need at least 2 images"})
		return
	}
	// load grayscale
	toGray := func(path string) ([][]uint8, int, int) {
		f, _ := os.Open(path)
		defer f.Close()
		img, _, _ := image.Decode(f)
		b := img.Bounds()
		w, h := b.Dx(), b.Dy()
		g := make([][]uint8, h)
		for y := 0; y < h; y++ {
			row := make([]uint8, w)
			for x := 0; x < w; x++ {
				r, gg, bb, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
				row[x] = uint8((299*int(r>>8) + 587*int(gg>>8) + 114*int(bb>>8)) / 1000)
			}
			g[y] = row
		}
		return g, w, h
	}
	g1, _, _ := toGray(frames[0].Path)
	g2, _, _ := toGray(frames[1].Path)
	disp := mvs.ComputeDepthMap(g1, g2, 9, 64)
	k := frames[0].Intrinsics
	cloud := mvs.DepthToCloud(disp, k.FocalLengthX, k.FocalLengthY, k.PrincipalX, k.PrincipalY, 1.0)
	_ = export.WritePLY(s.cfg.Output+"/dense.ply", cloud)
	m := mesh.GridMesh(10, 10)
	_ = export.WriteOBJ(s.cfg.Output+"/mesh.obj", m)
	writeJSON(w, http.StatusOK, map[string]any{"points": len(cloud), "triangles": len(m.Triangles)})
}

func (s *Server) handleSfMRun(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InputDir string `json:"input_dir"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if req.InputDir == "" {
		req.InputDir = s.cfg.Input
	}
	frames, err := ingest.IngestDir(req.InputDir)
	if err != nil || len(frames) < 2 {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "need at least 2 images"})
		return
	}
	paths := make([]string, len(frames))
	for i, fr := range frames {
		paths[i] = fr.Path
	}
	det, _ := features.NewDetector("fast")
	sets, err := features.NewPool(det, 4).Run(r.Context(), paths)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	matcherFn := func(i, j int) ([]int, [][2]float64, [][2]float64) {
		mp := matcher.NewMatcher(0.75).Match(frames[i].ID, frames[j].ID, sets[i], sets[j])
		idxs := make([]int, len(mp.Matches))
		p1 := make([][2]float64, len(mp.Matches))
		p2 := make([][2]float64, len(mp.Matches))
		for k, mc := range mp.Matches {
			idxs[k] = mc.QueryIdx
			p1[k] = [2]float64{sets[i].Keypoints[mc.QueryIdx].X, sets[i].Keypoints[mc.QueryIdx].Y}
			p2[k] = [2]float64{sets[j].Keypoints[mc.TrainIdx].X, sets[j].Keypoints[mc.TrainIdx].Y}
		}
		return idxs, p1, p2
	}
	k := frames[0].Intrinsics.KMatrix
	if k == nil {
		k = newKMatrix(frames[0].Intrinsics.FocalLengthX, frames[0].Intrinsics.FocalLengthY, frames[0].Intrinsics.PrincipalX, frames[0].Intrinsics.PrincipalY)
	}
	res := sfm.RunIncremental(len(frames), k, sets, matcherFn)
	writeJSON(w, http.StatusOK, map[string]any{"poses": len(res.Poses), "points": len(res.Cloud)})
}

func (s *Server) handleMesh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InputDir string `json:"input_dir"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if req.InputDir == "" {
		req.InputDir = s.cfg.Input
	}
	frames, err := ingest.IngestDir(req.InputDir)
	if err != nil || len(frames) < 2 {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "need at least 2 images"})
		return
	}
	paths := make([]string, len(frames))
	for i, fr := range frames {
		paths[i] = fr.Path
	}
	det, _ := features.NewDetector("fast")
	sets, err := features.NewPool(det, 4).Run(r.Context(), paths)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	mm := matcher.NewMatcher(0.75)
	pair := mm.Match(frames[0].ID, frames[1].ID, sets[0], sets[1])
	pts1 := make([][2]float64, len(pair.Matches))
	pts2 := make([][2]float64, len(pair.Matches))
	for i, mc := range pair.Matches {
		pts1[i] = [2]float64{sets[0].Keypoints[mc.QueryIdx].X, sets[0].Keypoints[mc.QueryIdx].Y}
		pts2[i] = [2]float64{sets[1].Keypoints[mc.TrainIdx].X, sets[1].Keypoints[mc.TrainIdx].Y}
	}
	k1 := frames[0].Intrinsics.KMatrix
	if k1 == nil {
		k1 = newKMatrix(frames[0].Intrinsics.FocalLengthX, frames[0].Intrinsics.FocalLengthY, frames[0].Intrinsics.PrincipalX, frames[0].Intrinsics.PrincipalY)
	}
	k2 := frames[1].Intrinsics.KMatrix
	if k2 == nil {
		k2 = newKMatrix(frames[1].Intrinsics.FocalLengthX, frames[1].Intrinsics.FocalLengthY, frames[1].Intrinsics.PrincipalX, frames[1].Intrinsics.PrincipalY)
	}
	mask, E := matcher.RANSACEssential(pair, sets[0].Keypoints, sets[1].Keypoints, k1, k2, 1.0, 200)
	if E == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "essential failed"})
		return
	}
	var in1, in2 [][2]float64
	for i := range pair.Matches {
		if mask[i] == 1 {
			in1 = append(in1, pts1[i])
			in2 = append(in2, pts2[i])
		}
	}
	R, t := sfm.RecoverPose(E, k1, in1, in2)
	P1 := sfm.ProjectionMatrix(k1, matEye3(), matZero3x1())
	P2 := sfm.ProjectionMatrix(k2, R, t)
	var cloud [][3]float64
	for i := range in1 {
		X := sfm.TriangulatePoint(P1, P2, in1[i], in2[i])
		cloud = append(cloud, X)
	}
	// project to 2D for delaunay
	pts2d := make([][2]float64, len(cloud))
	for i, c := range cloud {
		pts2d[i] = [2]float64{c[0], c[1]}
	}
	tris := mesh.Delaunay2D(pts2d)
	m := models.Mesh{Vertices: make([][3]float64, len(cloud)), Triangles: tris}
	copy(m.Vertices, cloud)
	uvs := mesh.PlanarUV(m)
	_ = export.WriteOBJTextured(s.cfg.Output+"/recon.obj", m, uvs)
	writeJSON(w, http.StatusOK, map[string]any{"vertices": len(cloud), "triangles": len(tris)})
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InputDir string `json:"input_dir"`
		Format   string `json:"format"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if req.InputDir == "" {
		req.InputDir = s.cfg.Input
	}
	if req.Format == "" {
		req.Format = "all"
	}
	// Run mesh pipeline to ensure files exist.
	frames, _ := ingest.IngestDir(req.InputDir)
	paths := make([]string, len(frames))
	for i, fr := range frames {
		paths[i] = fr.Path
	}
	det, _ := features.NewDetector("fast")
	sets, _ := features.NewPool(det, 4).Run(r.Context(), paths)
	mm := matcher.NewMatcher(0.75)
	pair := mm.Match(0, 1, sets[0], sets[1])
	pts1 := make([][2]float64, len(pair.Matches))
	pts2 := make([][2]float64, len(pair.Matches))
	for i, mc := range pair.Matches {
		pts1[i] = [2]float64{sets[0].Keypoints[mc.QueryIdx].X, sets[0].Keypoints[mc.QueryIdx].Y}
		pts2[i] = [2]float64{sets[1].Keypoints[mc.TrainIdx].X, sets[1].Keypoints[mc.TrainIdx].Y}
	}
	k1 := frames[0].Intrinsics.KMatrix
	if k1 == nil {
		k1 = newKMatrix(frames[0].Intrinsics.FocalLengthX, frames[0].Intrinsics.FocalLengthY, frames[0].Intrinsics.PrincipalX, frames[0].Intrinsics.PrincipalY)
	}
	k2 := frames[1].Intrinsics.KMatrix
	if k2 == nil {
		k2 = newKMatrix(frames[1].Intrinsics.FocalLengthX, frames[1].Intrinsics.FocalLengthY, frames[1].Intrinsics.PrincipalX, frames[1].Intrinsics.PrincipalY)
	}
	mask, E := matcher.RANSACEssential(pair, sets[0].Keypoints, sets[1].Keypoints, k1, k2, 1.0, 200)
	var in1, in2 [][2]float64
	for i := range pair.Matches {
		if mask[i] == 1 {
			in1 = append(in1, pts1[i])
			in2 = append(in2, pts2[i])
		}
	}
	R, t := sfm.RecoverPose(E, k1, in1, in2)
	P1 := sfm.ProjectionMatrix(k1, matEye3(), matZero3x1())
	P2 := sfm.ProjectionMatrix(k2, R, t)
	var cloud [][3]float64
	for i := range in1 {
		cloud = append(cloud, sfm.TriangulatePoint(P1, P2, in1[i], in2[i]))
	}
	files := []string{}
	if req.Format == "ply" || req.Format == "all" {
		p := s.cfg.Output + "/export.ply"
		_ = export.WritePLY(p, toPoint3D(cloud))
		files = append(files, p)
	}
	if req.Format == "obj" || req.Format == "all" {
		pts2d := make([][2]float64, len(cloud))
		for i, c := range cloud {
			pts2d[i] = [2]float64{c[0], c[1]}
		}
		tris := mesh.Delaunay2D(pts2d)
		m := models.Mesh{Vertices: cloud, Triangles: tris}
		uvs := mesh.PlanarUV(m)
		p := s.cfg.Output + "/export.obj"
		_ = export.WriteOBJTextured(p, m, uvs)
		files = append(files, p)
	}
	writeJSON(w, http.StatusOK, map[string]any{"files": files})
}

func toPoint3D(cloud [][3]float64) []models.Point3D {
	out := make([]models.Point3D, len(cloud))
	for i, c := range cloud {
		out[i] = models.Point3D{Position: c, Color: [3]uint8{128, 128, 128}}
	}
	return out
}

func (s *Server) handlePoisson(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InputDir string `json:"input_dir"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if req.InputDir == "" {
		req.InputDir = s.cfg.Input
	}
	// reuse mesh handler logic by computing cloud then Poisson
	frames, err := ingest.IngestDir(req.InputDir)
	if err != nil || len(frames) < 2 {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "need at least 2 images"})
		return
	}
	paths := make([]string, len(frames))
	for i, fr := range frames {
		paths[i] = fr.Path
	}
	det, _ := features.NewDetector("fast")
	sets, err := features.NewPool(det, 4).Run(r.Context(), paths)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	mm := matcher.NewMatcher(0.75)
	pair := mm.Match(frames[0].ID, frames[1].ID, sets[0], sets[1])
	pts1 := make([][2]float64, len(pair.Matches))
	pts2 := make([][2]float64, len(pair.Matches))
	for i, mc := range pair.Matches {
		pts1[i] = [2]float64{sets[0].Keypoints[mc.QueryIdx].X, sets[0].Keypoints[mc.QueryIdx].Y}
		pts2[i] = [2]float64{sets[1].Keypoints[mc.TrainIdx].X, sets[1].Keypoints[mc.TrainIdx].Y}
	}
	k1 := frames[0].Intrinsics.KMatrix
	if k1 == nil {
		k1 = newKMatrix(frames[0].Intrinsics.FocalLengthX, frames[0].Intrinsics.FocalLengthY, frames[0].Intrinsics.PrincipalX, frames[0].Intrinsics.PrincipalY)
	}
	k2 := frames[1].Intrinsics.KMatrix
	if k2 == nil {
		k2 = newKMatrix(frames[1].Intrinsics.FocalLengthX, frames[1].Intrinsics.FocalLengthY, frames[1].Intrinsics.PrincipalX, frames[1].Intrinsics.PrincipalY)
	}
	mask, E := matcher.RANSACEssential(pair, sets[0].Keypoints, sets[1].Keypoints, k1, k2, 1.0, 200)
	var in1, in2 [][2]float64
	for i := range pair.Matches {
		if mask[i] == 1 {
			in1 = append(in1, pts1[i])
			in2 = append(in2, pts2[i])
		}
	}
	R, t := sfm.RecoverPose(E, k1, in1, in2)
	P1 := sfm.ProjectionMatrix(k1, matEye3(), matZero3x1())
	P2 := sfm.ProjectionMatrix(k2, R, t)
	var cloud [][3]float64
	for i := range in1 {
		cloud = append(cloud, sfm.TriangulatePoint(P1, P2, in1[i], in2[i]))
	}
	m := mesh.PoissonReconstruct(cloud)
	uvs := mesh.PlanarUV(m)
	_ = export.WriteOBJTextured(s.cfg.Output+"/poisson.obj", m, uvs)
	writeJSON(w, http.StatusOK, map[string]any{"vertices": len(m.Vertices), "triangles": len(m.Triangles)})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
