# AGENTS.md — PhotoGen Web (current work)

## Status
Browser-based GUI photogrammetry engine in Go (backend) + React/Three.js (frontend). OpenCV removed; feature detection is now pure Go (FAST corners + BRIEF descriptors). Default builds work without CGO.

## What's done
- Backend: HTTP server (health, SSE, projects, FS listing, ingest, features, matching).
- Frontend: App shell, Images panel, Three.js viewport, Tailwind.
- Packages: models, ingest (EXIF), features (pure-Go FAST+BRIEF), matcher (brute-force + Lowe + RANSAC).
- Tooling: Makefile, Dockerfile, README, Todo.md, .gitignore. Repo pushed to GitHub.

## In progress / next
- Wire matching UI and viewport to real data.
- Implement M3: incremental SfM (PnP, triangulation, bundle adjustment) in pure Go.
- Implement M4: dense MVS (SGBM) + Poisson mesh + texturing.
- Add tests.

## Build
- `go build ./...` (no CGO required)
- `make dev` / `make frontend-dev`
