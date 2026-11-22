package sfm

import (
	"gonum.org/v1/gonum/mat"
)

// TriangulatePoint DLT-triangulates a 3D point from two views.
func TriangulatePoint(P1, P2 *mat.Dense, x1, x2 [2]float64) [3]float64 {
	// Build 4x4 matrix A.
	A := mat.NewDense(4, 4, nil)
	for i := 0; i < 4; i++ {
		A.Set(0, i, x1[0]*P1.At(2, i)-P1.At(0, i))
		A.Set(1, i, x1[1]*P1.At(2, i)-P1.At(1, i))
		A.Set(2, i, x2[0]*P2.At(2, i)-P2.At(0, i))
		A.Set(3, i, x2[1]*P2.At(2, i)-P2.At(1, i))
	}
	// SVD null space (last column of V)
	var svd mat.SVD
	if !svd.Factorize(A, mat.SVDThin) {
		return [3]float64{}
	}
	Vm := mat.NewDense(4, 4, nil)
	svd.VTo(Vm)
	X := [4]float64{Vm.At(0, 3), Vm.At(1, 3), Vm.At(2, 3), Vm.At(3, 3)}
	if X[3] != 0 {
		return [3]float64{X[0] / X[3], X[1] / X[3], X[2] / X[3]}
	}
	return [3]float64{X[0], X[1], X[2]}
}

// ProjectionMatrix builds K*[R|t] (3x4).
func ProjectionMatrix(K, R, t *mat.Dense) *mat.Dense {
	P := mat.NewDense(3, 4, nil)
	// RT = R | t
	RT := mat.NewDense(3, 4, nil)
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			RT.Set(i, j, R.At(i, j))
		}
		RT.Set(i, 3, t.At(i, 0))
	}
	P.Mul(K, RT)
	return P
}
