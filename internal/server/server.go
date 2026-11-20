package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/ngbcoder/photogramettry/internal/events"
	"github.com/ngbcoder/photogramettry/pkg/features"
	"github.com/ngbcoder/photogramettry/pkg/ingest"
	"github.com/ngbcoder/photogramettry/pkg/matcher"
	"github.com/ngbcoder/photogramettry/pkg/models"
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

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
