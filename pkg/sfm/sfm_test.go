package sfm

import (
	"testing"

	"gonum.org/v1/gonum/mat"
)

func TestTriangulatePoint(t *testing.T) {
	K := mat.NewDense(3, 3, []float64{100, 0, 50, 0, 100, 50, 0, 0, 1})
	R := mat.NewDense(3, 3, []float64{1, 0, 0, 0, 1, 0, 0, 0, 1})
	tt := mat.NewDense(3, 1, []float64{1, 0, 0})
	P1 := ProjectionMatrix(K, mat.NewDense(3, 3, []float64{1, 0, 0, 0, 1, 0, 0, 0, 1}), mat.NewDense(3, 1, []float64{0, 0, 0}))
	P2 := ProjectionMatrix(K, R, tt)
	X := TriangulatePoint(P1, P2, [2]float64{50, 50}, [2]float64{49, 50})
	if X[2] == 0 {
		t.Error("degenerate triangulation")
	}
}
