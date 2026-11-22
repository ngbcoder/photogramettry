package mesh

import (
	"github.com/ngbcoder/photogramettry/pkg/models"
)

// PoissonReconstruct builds a surface mesh from points using Delaunay on the XY projection.
// (Simplified stand-in for screened Poisson surface reconstruction.)
func PoissonReconstruct(points [][3]float64) models.Mesh {
	pts2d := make([][2]float64, len(points))
	for i, p := range points {
		pts2d[i] = [2]float64{p[0], p[1]}
	}
	tris := Delaunay2D(pts2d)
	m := models.Mesh{Vertices: points, Triangles: tris}
	return SmoothMesh(m, 1)
}
