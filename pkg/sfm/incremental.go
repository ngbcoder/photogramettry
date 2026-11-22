package sfm

import (
	"gonum.org/v1/gonum/mat"

	"github.com/ngbcoder/photogramettry/pkg/models"
)

// IncrementalResult holds outputs of incremental SfM.
type IncrementalResult struct {
	Poses  []*mat.Dense
	Cloud  [][3]float64
}

// RunIncremental runs a simplified incremental SfM over feature sets.
// It uses seed pair triangulation then PnP for subsequent images.
func RunIncremental(frames int, k *mat.Dense, featureSets []models.FeatureSet, matcher func(i, j int) ([]int, [][2]float64, [][2]float64)) IncrementalResult {
	res := IncrementalResult{}
	if frames < 2 {
		return res
	}
	res.Poses = make([]*mat.Dense, frames)
	// seed: identity and recovered pose from pair 0,1
	res.Poses[0] = mat.NewDense(4, 4, []float64{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1})
	// We'll keep poses as 4x4; convert to 3x4 projection externally. For simplicity store 4x4.
	// Triangulate seed
	idxs, pts1, pts2 := matcher(0, 1)
	E := estimateEssentialFromPts(pts1, pts2) // may be nil
	_ = E
	// Use simple baseline pose for seed
	R := mat.NewDense(3, 3, []float64{1, 0, 0, 0, 1, 0, 0, 0, 1})
	t := mat.NewDense(3, 1, []float64{1, 0, 0})
	res.Poses[1] = toPose4x4(R, t)
	P1 := ProjectionMatrix(k, mat.NewDense(3, 3, []float64{1, 0, 0, 0, 1, 0, 0, 0, 1}), mat.NewDense(3, 1, []float64{0, 0, 0}))
	P2 := ProjectionMatrix(k, R, t)
	for i := range idxs {
		X := TriangulatePoint(P1, P2, pts1[i], pts2[i])
		res.Cloud = append(res.Cloud, X)
	}
	// subsequent images: PnP using correspondences to previous cloud (approx: use matched indices)
	for img := 2; img < frames; img++ {
		idxs2, _, pCur := matcher(img-1, img)
		// Build 3D-2D correspondences from previous triangulated points (if index exists)
		var pts3D [][3]float64
		var pts2D [][2]float64
		for i, mc := range idxs2 {
			if mc < len(res.Cloud) {
				pts3D = append(pts3D, res.Cloud[mc])
				pts2D = append(pts2D, pCur[i])
			}
		}
		var Rn, tn *mat.Dense
		if len(pts3D) >= 6 {
			Rn, tn = DLT_PnP(k, pts3D, pts2D)
		}
		if Rn == nil {
			Rn = mat.NewDense(3, 3, []float64{1, 0, 0, 0, 1, 0, 0, 0, 1})
			tn = mat.NewDense(3, 1, []float64{1, 0, 0})
		}
		res.Poses[img] = toPose4x4(Rn, tn)
	}
	return res
}

func toPose4x4(R *mat.Dense, t *mat.Dense) *mat.Dense {
	P := mat.NewDense(4, 4, nil)
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			P.Set(i, j, R.At(i, j))
		}
		P.Set(i, 3, t.At(i, 0))
	}
	P.Set(3, 3, 1)
	return P
}

func estimateEssentialFromPts(pts1, pts2 [][2]float64) *mat.Dense {
	// placeholder returns nil (not used)
	return nil
}

var _ = models.FeatureSet{}
