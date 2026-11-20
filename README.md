# PhotoGen Web

Browser-based GUI photogrammetry engine. Go backend (GoCV + Gonum) + React/Three.js frontend.

## Dev setup

Backend:
- Go 1.22+
- OpenCV 4.x (for `-tags opencv` builds). Without OpenCV, builds use stubs.

Frontend:
- Node 20+

Run dev:
```
make dev          # backend
make frontend-dev # frontend (Vite)
```

Build:
```
make build
```

Docker:
```
make docker
```
