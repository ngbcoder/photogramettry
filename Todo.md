# Todo — PhotoGen Web GUI Photogrammetry Engine

Tracking file. Update checkboxes as work completes. Milestones are intentionally granular.

## Phase 0 — Repository & Tooling
- [ ] Initialize git repository
- [ ] Create GitHub repository (ngbcoder/photogramettry)
- [ ] Add .gitignore (Go, Node, OpenCV, OS)
- [ ] Add Makefile (build, test, lint, dev)
- [ ] Add Dockerfile (OpenCV + Go + Node multi-stage)
- [ ] Add README with dev setup (OpenCV install per OS)
- [ ] Configure golangci-lint
- [ ] Configure ESLint/Prettier for frontend

## Phase 1 — Backend Foundation (Go)
- [ ] Choose HTTP router (net/http ServeMux vs chi) and document
- [ ] Create `cmd/photogen-web/main.go` (server entrypoint, flags: --port, --input, --output)
- [ ] Add graceful shutdown + signal handling
- [ ] Add structured logging (slog) with verbose flag
- [ ] Add OpenCV availability check on startup (fail fast with helpful message)
- [ ] Add CORS + static file serving for frontend build
- [ ] Add health endpoint `/api/health`
- [ ] Add project model (`Project` struct) + `.photogen` JSON persistence
- [ ] Add project endpoints: create/open/save/list recent
- [ ] Add directory listing endpoint (for path picker)
- [ ] Add thumbnail generation endpoint (cached)
- [ ] Add WebSocket (or SSE) event bus for progress/logs
- [ ] Add context/cancellation plumbing for long-running stages
- [ ] Add unit tests for project persistence

## Phase 2 — Frontend Foundation (React + Vite)
- [ ] Scaffold Vite React TS app under `frontend/`
- [ ] Add Tailwind CSS
- [ ] Add routing/layout (sidebar, main panel, status bar)
- [ ] Add API client (fetch wrapper + base URL)
- [ ] Add WebSocket client for events
- [ ] Add global state (zustand): project, images, stage status, logs
- [ ] Add dark/light theme toggle
- [ ] Add reusable components: Button, Card, Modal, ProgressBar, LogConsole
- [ ] Add path picker dialog (server-backed)
- [ ] Add error toast notifications
- [ ] Add loading skeletons

## Phase 3 — 3D Viewport (Three.js) foundation
- [ ] Add three.js + OrbitControls
- [ ] Create Viewport3D component (canvas, resize handling)
- [ ] Add point cloud renderer (BufferGeometry, vertex colors, size control)
- [ ] Add mesh renderer (geometry + material + texture)
- [ ] Add camera frustum helper
- [ ] Add grid/axes helpers
- [ ] Add screenshot export (PNG)
- [ ] Add performance: chunked point loading for large clouds

## Phase 4 — M1: Ingest + Features (first milestone)
### Backend
- [ ] Implement `pkg/ingest/exif.go` (EXIF focal length, dimensions)
- [ ] Implement `pkg/ingest/camera_db.go` (sensor width lookup)
- [ ] Build `ImageFrame` list + intrinsics K matrix
- [ ] Add `/api/ingest` endpoint (input dir -> frames)
- [ ] Add progress events during ingest
- [ ] Implement `pkg/features/detector.go` interface (AKAZE/ORB)
- [ ] Implement `pkg/features/pool.go` bounded worker pool
- [ ] Add `/api/features/run` endpoint
- [ ] Enforce `Mat.Close()` in all feature paths
- [ ] Cache FeatureSet per frame (disk)
- [ ] Add `/api/features/keypoints` endpoint (for overlay)
- [ ] Add export keypoints PLY
- [ ] Unit tests for ingest + features
### Frontend
- [ ] ImageGrid component (thumbnails, metadata)
- [ ] Image detail panel (dims, focal, K)
- [ ] Run ingest button + progress
- [ ] Run features button + progress
- [ ] Keypoint overlay toggle on thumbnail/preview
- [ ] Stage status panel (M1 stages)
- [ ] Log console filtering

## Phase 5 — M2: Feature Matching + Two-View Geometry
### Backend
- [ ] Implement `pkg/matcher/matcher.go` (Hamming/FLANN)
- [ ] Implement Lowe's ratio test (threshold configurable)
- [ ] Implement `pkg/matcher/ransac.go` (Essential matrix + inliers)
- [ ] Add `/api/match/run` endpoint (pairwise)
- [ ] Add match visualization data endpoint
- [ ] Implement two-view triangulation (SVD, gonum/mat)
- [ ] Add `/api/sfm/init` endpoint (seed pair -> cloud + poses)
- [ ] Unit tests for matcher/ransac/triangulation
### Frontend
- [ ] Match graph view (which pairs matched)
- [ ] Inlier visualization (side-by-side images with lines)
- [ ] Two-view cloud in 3D viewport
- [ ] Camera frustums for seed pair
- [ ] Reprojection error display

## Phase 6 — M3: Incremental SfM + Bundle Adjustment
### Backend
- [ ] Implement PnP pose solver (`pkg/sfm/pnp.go`)
- [ ] Incremental registration loop (add images one by one)
- [ ] Track 2D-3D correspondences
- [ ] Implement sparse BA (`pkg/sfm/bundle_adjustment.go`) via gonum
- [ ] Optional Ceres CGO binding if available
- [ ] Add `/api/sfm/run` endpoint (full incremental)
- [ ] Add `/api/sfm/ba` endpoint (rerun BA)
- [ ] Progress events per registered image
- [ ] Unit tests for PnP/BA
### Frontend
- [ ] Incremental cloud growth animation
- [ ] Registered camera count + pose list
- [ ] BA convergence chart (error vs iteration)
- [ ] Pause/resume/cancel controls
- [ ] Reprojection error heatmap

## Phase 7 — M4: Dense MVS + Mesh + Texture
### Backend
- [ ] Implement SGBM depth maps (`pkg/mvs/sgbm.go`)
- [ ] Implement depth map fusion (`pkg/mvs/fused_cloud.go`)
- [ ] Implement Poisson reconstruction (`pkg/mesh/poisson.go`)
- [ ] Implement UV unwrapping + texture baking (`pkg/mesh/texturing.go`)
- [ ] Add `/api/mvs/run` endpoint
- [ ] Add `/api/mesh/run` endpoint
- [ ] Add GLTF export (in addition to PLY/OBJ)
- [ ] Unit tests for mvs/mesh
### Frontend
- [ ] Depth map preview
- [ ] Dense cloud toggle in viewport
- [ ] Mesh view (solid/wireframe/textured)
- [ ] Texture preview
- [ ] Export dialog (format picker, path)
- [ ] Results summary panel (points, tris, error)

## Phase 8 — Export & Project Management
- [ ] Implement `pkg/export/ply.go` (ASCII + binary)
- [ ] Implement `pkg/export/obj.go` (+ MTL)
- [ ] Add `/api/export` endpoint
- [ ] Project save/load with full state
- [ ] Recent projects list
- [ ] Auto-save

## Phase 9 — Quality, Performance, Memory
- [ ] Memory budget config + worker throttling
- [ ] Mat lifecycle audit
- [ ] Benchmark feature extraction
- [ ] Benchmark matching
- [ ] Profile memory (pprof endpoint)
- [ ] Fix leaks found

## Phase 10 — Packaging & Release
- [ ] Cross-compile scripts (linux/darwin/windows)
- [ ] GitHub Actions CI (build + test)
- [ ] GitHub Actions release workflow (binaries)
- [ ] Docker image publish
- [ ] User docs
- [ ] Changelog

## Phase 11 — Polish
- [ ] Keyboard shortcuts
- [ ] Onboarding tour
- [ ] Crash reporting
- [ ] Auto-update check
- [ ] Accessibility pass
