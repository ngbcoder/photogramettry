package sfm

import (
	"gonum.org/v1/gonum/mat"
)

// BundleAdjuster performs (simplified) bundle adjustment refining 3D points.
type BundleAdjuster struct {
	Iterations int
}

// Run refines points given fixed camera poses and observations.
// observations: map pointIndex -> list of (cameraIndex, pixel x,y)
func (ba *BundleAdjuster) Run(points [][3]float64, poses []*mat.Dense, k *mat.Dense, obs map[int][][3]float64) [][3]float64 {
	if ba.Iterations <= 0 {
		ba.Iterations = 5
	}
	out := make([][3]float64, len(points))
	copy(out, points)
	for it := 0; it < ba.Iterations; it++ {
		for pi := range out {
			o, ok := obs[pi]
			if !ok || len(o) < 2 {
				continue
			}
			// Gauss-Newton: solve for delta minimizing sum of squared reprojection errors.
			// Jacobian 2m x 3, residual 2m.
			m := len(o)
			J := mat.NewDense(2*m, 3, nil)
			r := mat.NewVecDense(2*m, nil)
			for ki, ob := range o {
				ci := int(ob[0])
				px, py := ob[1], ob[2]
				Pose := poses[ci]
				// project
				X := mat.NewVecDense(4, []float64{out[pi][0], out[pi][1], out[pi][2], 1})
				pcam := mat.NewVecDense(3, nil)
				pcam.MulVec(Pose, X)
				if pcam.AtVec(2) == 0 {
					continue
				}
				xp := k.At(0, 0)*pcam.AtVec(0)/pcam.AtVec(2) + k.At(0, 2)
				yp := k.At(1, 1)*pcam.AtVec(1)/pcam.AtVec(2) + k.At(1, 2)
				r.SetVec(2*ki, xp-px)
				r.SetVec(2*ki+1, yp-py)
				// jacobian approx via finite differences
				eps := 1e-6
				for d := 0; d < 3; d++ {
					plus := [3]float64{out[pi][0], out[pi][1], out[pi][2]}
					minus := [3]float64{out[pi][0], out[pi][1], out[pi][2]}
					plus[d] += eps
					minus[d] -= eps
					for _, sign := range []int{1, -1} {
						pt := plus
						if sign < 0 {
							pt = minus
						}
						Xp := mat.NewVecDense(4, []float64{pt[0], pt[1], pt[2], 1})
						pp := mat.NewVecDense(3, nil)
						pp.MulVec(Pose, Xp)
						if pp.AtVec(2) == 0 {
							continue
						}
						xpp := k.At(0, 0)*pp.AtVec(0)/pp.AtVec(2) + k.At(0, 2)
						ypp := k.At(1, 1)*pp.AtVec(1)/pp.AtVec(2) + k.At(1, 2)
						// derivative
						_ = xpp
						_ = ypp
					}
					// finite diff derivative
					Xp := mat.NewVecDense(4, []float64{plus[0], plus[1], plus[2], 1})
					Xm := mat.NewVecDense(4, []float64{minus[0], minus[1], minus[2], 1})
					pp := mat.NewVecDense(3, nil)
					pm := mat.NewVecDense(3, nil)
					pp.MulVec(Pose, Xp)
					pm.MulVec(Pose, Xm)
					if pp.AtVec(2) != 0 && pm.AtVec(2) != 0 {
						dx := (k.At(0, 0)*pp.AtVec(0)/pp.AtVec(2) + k.At(0, 2)) - (k.At(0, 0)*pm.AtVec(0)/pm.AtVec(2) + k.At(0, 2))
						dy := (k.At(1, 1)*pp.AtVec(1)/pp.AtVec(2) + k.At(1, 2)) - (k.At(1, 1)*pm.AtVec(1)/pm.AtVec(2) + k.At(1, 2))
						J.Set(2*ki, d, dx/(2*eps))
						J.Set(2*ki+1, d, dy/(2*eps))
					}
				}
			}
			// solve (J^T J) delta = -J^T r
			Jt := mat.NewDense(3, 2*m, nil)
			Jt.CloneFrom(J.T())
			A := mat.NewDense(3, 3, nil)
			A.Mul(Jt, J)
			// add damping
			for d := 0; d < 3; d++ {
				A.Set(d, d, A.At(d, d)+1e-3)
			}
			b := mat.NewVecDense(3, nil)
			b.MulVec(Jt, r)
			for d := 0; d < 3; d++ {
				b.SetVec(d, -b.AtVec(d))
			}
			delta := mat.NewVecDense(3, nil)
			if err := delta.SolveVec(A, b); err != nil {
				continue
			}
			for d := 0; d < 3; d++ {
				out[pi][d] += delta.AtVec(d)
			}
		}
	}
	return out
}
