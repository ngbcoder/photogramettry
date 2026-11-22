package sfm

import (
	"gonum.org/v1/gonum/mat"
)

// RecoverPose decomposes Essential matrix into R, t (up to scale), choosing cheirality.
func RecoverPose(E, K *mat.Dense, pts1, pts2 [][2]float64) (*mat.Dense, *mat.Dense) {
	var svd mat.SVD
	if !svd.Factorize(E, mat.SVDThin) {
		return nil, nil
	}
	Um := mat.NewDense(3, 3, nil)
	svd.UTo(Um)
	Vtm := mat.NewDense(3, 3, nil)
	svd.VTo(Vtm)

	W := mat.NewDense(3, 3, []float64{0, -1, 0, 1, 0, 0, 0, 0, 1})
	Wt := mat.NewDense(3, 3, nil)
	Wt.CloneFrom(W.T())

	type cand struct {
		R *mat.Dense
		t *mat.Dense
	}
	var cands []cand
	for _, Wc := range []*mat.Dense{W, Wt} {
		for s := 0; s < 2; s++ {
			sign := 1.0
			if s == 1 {
				sign = -1.0
			}
			R := mat.NewDense(3, 3, nil)
			tmp := mat.NewDense(3, 3, nil)
			tmp.Mul(Um, Wc)
			R.Mul(tmp, Vtm.T())
			t := mat.NewDense(3, 1, []float64{sign * Um.At(0, 2), sign * Um.At(1, 2), sign * Um.At(2, 2)})
			cands = append(cands, cand{R: R, t: t})
		}
	}

	bestScore := -1
	var bestR, bestT *mat.Dense
	for _, c := range cands {
		score := 0
		for i := range pts1 {
			P1 := mat.NewDense(3, 4, nil)
			for r := 0; r < 3; r++ {
				P1.Set(r, r, K.At(r, r))
				P1.Set(r, 3, 0)
			}
			P2 := ProjectionMatrix(K, c.R, c.t)
			X := TriangulatePoint(P1, P2, pts1[i], pts2[i])
			// depth cam1
			if X[2] <= 0 {
				continue
			}
			pt := mat.NewVecDense(4, []float64{X[0], X[1], X[2], 1})
			RT := mat.NewDense(3, 4, nil)
			for r := 0; r < 3; r++ {
				for cc := 0; cc < 3; cc++ {
					RT.Set(r, cc, c.R.At(r, cc))
				}
				RT.Set(r, 3, c.t.At(r, 0))
			}
			cam2 := mat.NewVecDense(3, nil)
			cam2.MulVec(RT, pt)
			if cam2.AtVec(2) > 0 {
				score++
			}
		}
		if score > bestScore {
			bestScore = score
			bestR = c.R
			bestT = c.t
		}
	}
	if bestR == nil {
		bestR = cands[0].R
		bestT = cands[0].t
	}
	return bestR, bestT
}
