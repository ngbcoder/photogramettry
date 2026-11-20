Software Design Document: PhotoGen (Go Photogrammetry Engine)
Status: Draft | Language: Go (1.22+) | Core Dependencies: GoCV (OpenCV bindings), Gonum
|---|
1. Overview & Goals
PhotoGen is an open-source, concurrent photogrammetry engine written in Go. The system processes a collection of overlapping 2D photographs of a physical object or scene and reconstructs high-fidelity 3D point clouds, dense geometry, and textured surface meshes.
Objective
Provide a lightweight, memory-efficient, and easily deployable CLI tool that executes the end-to-end multi-view geometry pipeline—from feature extraction to textured mesh export—leveraging Go's native concurrency model for pipeline stage acceleration.
Target Use Cases
Small-to-medium physical object 3D scanning.
Rapid 3D model prototyping for spatial computing and game development.
Multi-view geometry learning, research, and algorithmic experimentation.
Automated headless 3D asset generation pipelines running in containerized environments.
Non-Goals (v1.0)
Real-time Simultaneous Localization and Mapping (SLAM).
Massive city-scale or aerial drone mapping involving tens of thousands of ultra-high-resolution images.
Native custom GPU CUDA kernel implementation within the Go codebase.
2. System Architecture & Concurrency Model
PhotoGen utilizes Go's native concurrency mechanisms (goroutines, channels, bounded worker pools, and golang.org/x/sync/errgroup) to maximize multi-core CPU utilization during computationally intensive tasks like image resizing, keypoint detection, descriptor extraction, and pairwise feature matching.

+-----------------------------------------------------------------------------------+
|                                 INPUT DATASET                                     |
|                       (Directory of JPEG / PNG Images)                            |
+-----------------------------------------------------------------------------------+
                                          |
                                          v
+-----------------------------------------------------------------------------------+
|                         STAGE 1: IMAGE INGESTION (`pkg/ingest`)                  |
|          Extract EXIF metadata, parse focal length, construct initial K           |
+-----------------------------------------------------------------------------------+
                                          |
                                          v
+-----------------------------------------------------------------------------------+
|                      STAGE 2: FEATURE EXTRACTION (`pkg/features`)                 |
|   +---------------------------------------------------------------------------+   |
|   |                      Bounded Worker Pool (NumCPU)                         |   |
|   |  Worker 1 (Mat) ---> Worker 2 (Mat) ---> Worker 3 (Mat) ... Worker N      |   |
|   +---------------------------------------------------------------------------+   |
|             Explicit lifecycle: Mat.Close() prevents CGO leaks                    |
+-----------------------------------------------------------------------------------+
                                          |
                                          v
+-----------------------------------------------------------------------------------+
|                   STAGE 3: FEATURE MATCHING (`pkg/matcher`)                       |
|   +---------------------------------------------------------------------------+   |
|   |                  Concurrent Pairwise Matching Engine                      |   |
|   |    FLANN / Hamming Distance + Lowe's Ratio Test + RANSAC Filter           |   |
|   +---------------------------------------------------------------------------+   |
+-----------------------------------------------------------------------------------+
                                          |
                                          v
+-----------------------------------------------------------------------------------+
|                 STAGE 4: INCREMENTAL SfM RECONSTRUCTION (`pkg/sfm`)               |
|      Initial Seed Pair -> 3D Triangulation -> PnP Camera Registration             |
|                  -> Global Bundle Adjustment (Sparse Ceres/Gonum)                 |
+-----------------------------------------------------------------------------------+
                                          |
                                          v
+-----------------------------------------------------------------------------------+
|               STAGE 5: DENSE MVS & SURFACE RECONSTRUCTION (`pkg/mvs`)             |
|         Semi-Global Block Matching (SGBM) -> Poisson Surface Reconstruction       |
+-----------------------------------------------------------------------------------+
                                          |
                                          v
+-----------------------------------------------------------------------------------+
|                           STAGE 6: EXPORT (`pkg/export`)                          |
|                     Streaming Output: .PLY Point Clouds / .OBJ Mesh               |
+-----------------------------------------------------------------------------------+
Memory Management & CGO Safety
Go's garbage collector is oblivious to allocations made on the C++ heap by OpenCV underlying gocv.Mat structures. Unmanaged allocations inside tight worker loops will rapidly cause out-of-memory crashes. PhotoGen strictly enforces deterministic frame-level resource cleanup using manual .Close() calls wrapped in defer statements, along with worker-level buffer reuse pattern to enforce an upper memory bound during execution.
3. Core Pipeline Specifications
Stage
Package
Primary Inputs
Core Algorithms & Go Libraries
Key Outputs
1. Ingestion
pkg/ingest
Raw Image Files (.jpg, .png)
EXIF extraction via rwcarlsen/goexif; sensor database lookup for pixel size conversion.
ImageFrame struct list containing image dimensions and initial Camera Intrinsics Matrix ($K$).
2. Feature Extraction
pkg/features
Ingested ImageFrame slice
Bounded goroutine pool executing gocv AKAZE or ORB keypoint detection and binary/float descriptor extraction.
Keypoint location arrays and dense feature descriptor matrices (FeatureSet).
3. Feature Matching
pkg/matcher
FeatureSet across all image pairs
Parallel pairwise Brute-Force Hamming or FLANN matching; Lowe's Ratio Test (threshold 0.75); RANSAC Essential Matrix estimation.
Geometric match matrix containing verified MatchPair pairs and robust inlier indexes.
4. Incremental SfM
pkg/sfm
Verified MatchPair graph, $K$ matrices
Five-point focal matrix estimation; SVD Triangulation (gonum/mat); Perspective-n-Point (PnP) pose solver; Sparse Bundle Adjustment (gonum/optimize).
Sparse 3D Point Cloud, calculated 6-DOF Camera Poses ($R, t$), Reprojection Error metrics.
5. Dense MVS & Mesh
pkg/mvs, pkg/mesh
Camera Poses, Sparse Cloud, Original Images
Semi-Global Block Matching (SGBM) for depth maps; Screened Poisson Surface Reconstruction or Delaunay Triangulation; UV projection.
Dense 3D Point Cloud, Triangle Mesh Index, Parameterized Texture Map.
6. Exporter
pkg/export
Sparse/Dense Cloud, Mesh Data
High-throughput buffered streaming writer for ASCII/Binary specifications.
.PLY point cloud files, .OBJ textured mesh files with corresponding .MTL material definitions.

4. Repository & Package Layout
photogen/
├── cmd/
│   └── photogen/
│       └── main.go                 # CLI Entrypoint, flag parsing, context handling
├── pkg/
│   ├── pipeline/
│   │   ├── pipeline.go             # Pipeline orchestrator & stage runner
│   │   └── context.go              # Thread-safe pipeline execution context
│   ├── ingest/
│   │   ├── exif.go                 # EXIF parsing & metadata extraction
│   │   └── camera_db.go            # Sensor width lookup table
│   ├── features/
│   │   ├── detector.go             # Feature detection interface (AKAZE / ORB)
│   │   └── pool.go                 # Concurrency worker pool for GoCV operations
│   ├── matcher/
│   │   ├── matcher.go              # FLANN and Brute-Force feature matchers
│   │   └── ransac.go               # RANSAC Epipolar geometry filtering
│   ├── sfm/
│   │   ├── initialize.go           # Seed pair selection & initial triangulation
│   │   ├── pnp.go                  # Perspective-n-Point pose estimation
│   │   └── bundle_adjustment.go    # Sparse bundle adjustment solver
│   ├── mvs/
│   │   ├── sgbm.go                 # Semi-Global Block Matching depth maps
│   │   └── fused_cloud.go          # Multi-view depth map fusion
│   ├── mesh/
│   │   ├── poisson.go              # Surface mesh reconstruction
│   │   └── texturing.go            # Mesh UV unwrap & image texture baking
│   └── export/
│       ├── ply.go                  # Binary/ASCII PLY point cloud writer
│       └── obj.go                  # OBJ/MTL surface mesh exporter
├── Dockerfile                      # Multi-stage build (OpenCV + GoCV + PhotoGen)
├── Makefile                        # Build, test, and dependency targets
├── go.mod                          # Go module definition
└── go.sum                          # Go module checksums
5. Data Structures & Interfaces
The core system architecture relies on strongly typed contracts to ensure thread safety and explicit memory ownership across pipeline stages.

package photogen

import (
	"context"
	"gonum.org/v1/gonum/mat"
	"gocv.io/x/gocv"
)

// CameraIntrinsics defines intrinsic parameters (K matrix).
type CameraIntrinsics struct {
	FocalLengthX float64     `json:"fx"`
	FocalLengthY float64     `json:"fy"`
	PrincipalX   float64     `json:"cx"`
	PrincipalY   float64     `json:"cy"`
	KMatrix      *mat.Dense  `json:"-"`
}

// ImageFrame represents an ingested input frame.
type ImageFrame struct {
	ID         int               `json:"id"`
	Path       string            `json:"path"`
	Width      int               `json:"width"`
	Height     int               `json:"height"`
	Intrinsics CameraIntrinsics  `json:"intrinsics"`
}

// FeatureSet holds detected keypoints and their descriptors.
type FeatureSet struct {
	FrameID     int           `json:"frame_id"`
	Keypoints   []gocv.Point2f `json:"keypoints"`
	Descriptors gocv.Mat      `json:"-"` // Explicit Close() required
}

// MatchPair holds verified feature correspondences between two frames.
type MatchPair struct {
	FrameIDA   int          `json:"frame_id_a"`
	FrameIDB   int          `json:"frame_id_b"`
	Matches    []gocv.DMatch `json:"matches"`
	InlierMask []byte       `json:"inlier_mask"`
}

// Point3D represents a reconstructed landmark in world coordinates.
type Point3D struct {
	ID         uint64     `json:"id"`
	Position   [3]float64 `json:"position"` // X, Y, Z
	Color      [3]uint8   `json:"color"`    // R, G, B
	Normals    [3]float64 `json:"normals"`  // Nx, Ny, Nz
	Error      float64    `json:"reprojection_error"`
}

// ReconstructionState contains the complete internal sparse/dense state.
type ReconstructionState struct {
	Frames     map[int]*ImageFrame     `json:"frames"`
	Poses      map[int]*mat.Dense      `json:"poses"` // 4x4 Transformation Matrices
	Points3D   map[uint64]*Point3D     `json:"points_3d"`
	Matches    map[uint64]*MatchPair   `json:"matches"`
}

// Stage interface implemented by all sequential execution modules.
type Stage interface {
	Name() string
	Run(ctx context.Context, state *ReconstructionState) (*ReconstructionState, error)
}
6. CLI Interface & Configuration
cmd/photogen exposes a command-line interface configured through arguments and flags:

Flag Name
Type
Default
Description
--input, -i
 String
"" (Required)
Path to the directory containing input image files.
--output, -o
 String
"./output"
Destination directory for point clouds, meshes, and logs.
--format, -f
 String
"ply"
Output geometry format. Supported: ply, obj, all.
--detector, -d
 String
"akaze"
Feature detection algorithm. Options: akaze, orb.
--workers, -w
 Int
NumCPU
Concurrent worker pool limit for feature processing.
--max-image-dim
 Int
2400
Downscales inputs if max(width, height) exceeds value (0 = original).
--ratio-test
 Float
0.75
Distance ratio threshold for Lowe's match filtering.
--verbose, -v
 Bool
false
Enables verbose debug logs and per-stage timing breakdown.

7. Memory Management, Risks & Mitigations
CGO / OpenCV Memory Leaks
Risk: OpenCV C++ allocations created via GoCV binding wrappers bypass Go's garbage collector. Creating thousands of transient gocv.Mat instances during pairwise image matching causes rapid system memory depletion.
Mitigation: Enforce strict memory scope rules. Every function that instantiates or receives a gocv.Mat pointer via transformation must explicitly call defer mat.Close(). Worker routines process images inside bounded channels, recycling fixed matrix buffer pools where feasible.
Build & Platform Complexity
Risk: Compiling CGO bindings alongside native C++ OpenCV libraries introduces heavy cross-compilation friction and brittle host environment setups across Unix and Windows environments.
Mitigation: Maintain a dual-stage Dockerfile. Stage 1 compiles C++ OpenCV static binaries and Go dependencies; Stage 2 outputs a minimal, lightweight runtime image. Provide a Makefile automation layer for local development.
Sparse Bundle Adjustment Performance
Risk: Pure Go matrix operations on large sparse Jacobians during Bundle Adjustment can lead to slow execution times during iterative non-linear optimization.
Mitigation: Use Cgo bindings to native Ceres Solver when present on system paths. If pure Go execution is required, utilize optimized block-diagonal linear algebra structures built on gonum/mat combined with Levenberg-Marquardt optimization provided by gonum/optimize.
8. Implementation Milestones
Phase
Milestone Name
Key Technical Deliverables
Target Artifacts
Phase 1
Ingestion, Features & Basic Export
Implement pkg/ingest EXIF reader, build bounded concurrent worker pool in pkg/features for AKAZE/ORB, implement basic .PLY writer in pkg/export.
Functional pipeline executing keypoint extraction on an image set and exporting keypoint visual representations.
Phase 2
Two-View Geometry & Triangulation
Implement pkg/matcher with FLANN/Hamming strategies, Lowe's ratio test, and RANSAC epipolar filtering. Build initial seed pair 3D triangulation in pkg/sfm via SVD (gonum/mat).
Generation of a stable, scaled 3D point cloud from a two-image baseline input.
Phase 3
Incremental SfM & Bundle Adjustment
Implement Perspective-n-Point (PnP) pose registration for progressive image registration. Integrate sparse Bundle Adjustment iteratively to correct visual drift.
Complete sparse 3D point cloud generation and calculated camera poses across arbitrary multi-view image sets.
Phase 4
Dense MVS, Meshing & Texturing
Implement SGBM stereo matching depth map generation in pkg/mvs. Implement Poisson Surface Reconstruction and UV texture projection in pkg/mesh. Export .OBJ/.MTL assets.
Fully textured 3D mesh assets generated directly from raw photo collections via CLI.


