package models

import "gonum.org/v1/gonum/mat"

// CameraIntrinsics defines intrinsic parameters (K matrix).
type CameraIntrinsics struct {
	FocalLengthX float64    `json:"fx"`
	FocalLengthY float64    `json:"fy"`
	PrincipalX   float64    `json:"cx"`
	PrincipalY   float64    `json:"cy"`
	KMatrix      *mat.Dense `json:"-"`
}

// ImageFrame represents an ingested input frame.
type ImageFrame struct {
	ID         int              `json:"id"`
	Path       string           `json:"path"`
	Width      int              `json:"width"`
	Height     int              `json:"height"`
	Intrinsics CameraIntrinsics `json:"intrinsics"`
}

// FeatureSet holds detected keypoints and their descriptors (stub descriptor bytes).
type FeatureSet struct {
	FrameID     int        `json:"frame_id"`
	Keypoints   []Point2   `json:"keypoints"`
	Descriptors [][]float64 `json:"-"`
}

// Point2 is a 2D point.
type Point2 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Correspondence is a matched keypoint index pair.
type Correspondence struct {
	QueryIdx int `json:"query_idx"`
	TrainIdx int `json:"train_idx"`
}

// MatchPair holds verified feature correspondences between two frames.
type MatchPair struct {
	FrameIDA   int              `json:"frame_id_a"`
	FrameIDB   int              `json:"frame_id_b"`
	Matches    []Correspondence `json:"matches"`
	InlierMask []byte           `json:"inlier_mask"`
}

// Point3D represents a reconstructed landmark.
type Point3D struct {
	ID       uint64     `json:"id"`
	Position [3]float64 `json:"position"`
	Color    [3]uint8   `json:"color"`
}
