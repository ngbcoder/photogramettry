package sfm

import (
	"gonum.org/v1/gonum/mat"
)

// DLT_PnP estimates camera pose (R,t) from 3D-2D correspondences.
func DLT_PnP(K *mat.Dense, pts3D [][3]float64, pts2D [][2]float64) (*mat.Dense, *mat.Dense) {
	n := len(pts3D)
	if n < 6 {
		return nil, nil
	}
	// Build 2n x 12 matrix.
	A := mat.NewDense(2*n, 12, nil)
	for i := 0; i < n; i++ {
		X := pts3D[i]
		x := pts2D[i]
		row := 2 * i
		A.Set(row, 0, X[0])
		A.Set(row, 1, X[1])
		A.Set(row, 2, X[2])
		A.Set(row, 3, 1)
		A.Set(row, 4, 0)
		A.Set(row, 5, 0)
		A.Set(row, 6, 0)
		A.Set(row, 7, 0)
		A.Set(row, 8, -x[0]*X[0])
		A.Set(row, 9, -x[0]*X[1])
		A.Set(row, 10, -x[0]*X[2])
		A.Set(row, 11, -x[0])
		row2 := row + 1
		A.Set(row2, 0, 0)
		A.Set(row2, 1, 0)
		A.Set(row2, 2, 0)
		A.Set(row2, 3, 0)
		A.Set(row2, 4, X[0])
		A.Set(row2, 5, X[1])
		A.Set(row2, 6, X[2])
		A.Set(row2, 7, 1)
		A.Set(row2, 8, -x[1]*X[0])
		A.Set(row2, 9, -x[1]*X[1])
		A.Set(row2, 10, -x[1]*X[2])
		A.Set(row2, 11, -x[1])
	}
	var svd mat.SVD
	if !svd.Factorize(A, mat.SVDThin) {
		return nil, nil
	}
	Vm := mat.NewDense(12, 12, nil)
	svd.VTo(Vm)
	// last column
	P := mat.NewDense(3, 4, nil)
	for i := 0; i < 3; i++ {
		for j := 0; j < 4; j++ {
			P.Set(i, j, Vm.At(i*4+j, 11))
		}
	}
	// M = K^-1 P
	Kinv := mat.NewDense(3, 3, nil)
	Kinv.Inverse(K)
	M := mat.NewDense(3, 4, nil)
	M.Mul(Kinv, P)
	// R = first 3 cols, t = last col
	R := mat.NewDense(3, 3, nil)
	t := mat.NewDense(3, 1, nil)
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			R.Set(i, j, M.At(i, j))
		}
		t.Set(i, 0, M.At(i, 3))
	}
	// Orthogonalize R via SVD
	var svd2 mat.SVD
	if svd2.Factorize(R, mat.SVDThin) {
		Um := mat.NewDense(3, 3, nil)
		svd2.UTo(Um)
		Vtm := mat.NewDense(3, 3, nil)
		svd2.VTo(Vtm)
		R.Mul(Um, Vtm.T())
	}
	return R, t
}
