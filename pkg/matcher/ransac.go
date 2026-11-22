package matcher

import (
	"math"
	"math/rand"

	"gonum.org/v1/gonum/mat"

	"github.com/ngbcoder/photogramettry/pkg/models"
)

// RANSACEssential runs RANSAC to estimate Essential matrix and inlier mask.
func RANSACEssential(pair models.MatchPair, ptsA, ptsB []models.Point2, kA, kB *mat.Dense, threshold float64, iters int) ([]byte, *mat.Dense) {
	n := len(pair.Matches)
	mask := make([]byte, n)
	if n < 8 || kA == nil || kB == nil {
		for i := range mask {
			mask[i] = 1
		}
		return mask, nil
	}
	if threshold <= 0 {
		threshold = 1.0
	}
	if iters <= 0 {
		iters = 500
	}

	// normalized points
	normA := make([][3]float64, n)
	normB := make([][3]float64, n)
	kaInv := mat.NewDense(3, 3, nil)
	kbInv := mat.NewDense(3, 3, nil)
	kaInv.Inverse(kA)
	kbInv.Inverse(kB)
	for i, m := range pair.Matches {
		pa := ptsA[m.QueryIdx]
		pb := ptsB[m.TrainIdx]
		vA := mat.NewVecDense(3, []float64{pa.X, pa.Y, 1})
		vB := mat.NewVecDense(3, []float64{pb.X, pb.Y, 1})
		na := mat.NewVecDense(3, nil)
		nb := mat.NewVecDense(3, nil)
		na.MulVec(kaInv, vA)
		nb.MulVec(kbInv, vB)
		normA[i] = [3]float64{na.AtVec(0), na.AtVec(1), na.AtVec(2)}
		normB[i] = [3]float64{nb.AtVec(0), nb.AtVec(1), nb.AtVec(2)}
	}

	bestInliers := 0
	var bestE *mat.Dense
	var bestMask []byte

	rng := rand.New(rand.NewSource(42))
	for iter := 0; iter < iters; iter++ {
		// sample 8
		idxs := randomSample(rng, n, 8)
		E := estimateEssential(idxs, normA, normB)
		if E == nil {
			continue
		}
		// count inliers
		curMask := make([]byte, n)
		inl := 0
		for i := 0; i < n; i++ {
			if sampsonError(E, normA[i], normB[i]) < threshold {
				curMask[i] = 1
				inl++
			}
		}
		if inl > bestInliers {
			bestInliers = inl
			bestE = E
			bestMask = curMask
		}
		if bestInliers == n {
			break
		}
	}

	if bestE == nil {
		for i := range mask {
			mask[i] = 1
		}
		return mask, nil
	}
	// refine mask
	for i := range mask {
		mask[i] = bestMask[i]
	}
	return mask, bestE
}

func randomSample(rng *rand.Rand, n, k int) []int {
	idxs := make([]int, 0, k)
	seen := map[int]bool{}
	for len(idxs) < k {
		r := rng.Intn(n)
		if seen[r] {
			continue
		}
		seen[r] = true
		idxs = append(idxs, r)
	}
	return idxs
}

func estimateEssential(idxs []int, a, b [][3]float64) *mat.Dense {
	// build A (8x9)
	A := mat.NewDense(len(idxs), 9, nil)
	for r, i := range idxs {
		xa, ya := a[i][0]/a[i][2], a[i][1]/a[i][2]
		xb, yb := b[i][0]/b[i][2], b[i][1]/b[i][2]
		A.Set(r, 0, xb*xa)
		A.Set(r, 1, xb*ya)
		A.Set(r, 2, xb)
		A.Set(r, 3, yb*xa)
		A.Set(r, 4, yb*ya)
		A.Set(r, 5, yb)
		A.Set(r, 6, xa)
		A.Set(r, 7, ya)
		A.Set(r, 8, 1)
	}
	// SVD
	var svd mat.SVD
	ok := svd.Factorize(A, mat.SVDThinU|mat.SVDFullV)
	if !ok {
		return nil
	}
	// null space = last column of V (smallest singular value)
	Vmat := mat.NewDense(9, 9, nil)
	svd.VTo(Vmat)
	// last column
	e := make([]float64, 9)
	for i := 0; i < 9; i++ {
		e[i] = Vmat.At(i, 8)
	}
	E := mat.NewDense(3, 3, e)
	// enforce rank 2
	var svd2 mat.SVD
	if !svd2.Factorize(E, mat.SVDThin) {
		return E
	}
	Umat := mat.NewDense(3, 3, nil)
	svd2.UTo(Umat)
	Vtmat := mat.NewDense(3, 3, nil)
	svd2.VTo(Vtmat)
	s := svd2.Values(nil)
	// zero smallest
	if len(s) >= 3 {
		s[2] = 0
	}
	// recompose: U * diag(s) * V^T
	D := mat.NewDense(3, 3, nil)
	for i := 0; i < 3 && i < len(s); i++ {
		D.Set(i, i, s[i])
	}
	tmp := mat.NewDense(3, 3, nil)
	tmp.Mul(Umat, D)
	E2 := mat.NewDense(3, 3, nil)
	E2.Mul(tmp, Vtmat.T())
	return E2
}

func sampsonError(E *mat.Dense, a, b [3]float64) float64 {
	xa, ya := a[0]/a[2], a[1]/a[2]
	xb, yb := b[0]/b[2], b[1]/b[2]
	pa := mat.NewVecDense(3, []float64{xa, ya, 1})
	pb := mat.NewVecDense(3, []float64{xb, yb, 1})
	// e = pb^T E pa
	Ep := mat.NewVecDense(3, nil)
	Ep.MulVec(E, pa)
	e := pb.AtVec(0)*Ep.AtVec(0) + pb.AtVec(1)*Ep.AtVec(1) + pb.AtVec(2)*Ep.AtVec(2)
	// denominators
	ETpb := mat.NewVecDense(3, nil)
	ETpb.MulVec(E.T(), pb)
	den := Ep.AtVec(0)*Ep.AtVec(0) + Ep.AtVec(1)*Ep.AtVec(1) +
		ETpb.AtVec(0)*ETpb.AtVec(0) + ETpb.AtVec(1)*ETpb.AtVec(1)
	if den == 0 {
		return math.Inf(1)
	}
	return (e * e) / den
}

// TriangulateTwoView is a placeholder for SVD triangulation.
func TriangulateTwoView(kA, kB *mat.Dense, r, t *mat.Dense, ptsA, ptsB [][2]float64) [][3]float64 {
	_ = kA
	_ = kB
	_ = r
	_ = t
	_ = ptsA
	_ = ptsB
	return nil
}

var _ = models.MatchPair{}
