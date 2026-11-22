package mesh

import "github.com/ngbcoder/photogramettry/pkg/models"

// GridMesh triangulates a depth/disparity grid into a mesh.
func GridMesh(rows, cols int) models.Mesh {
	m := models.Mesh{}
	// vertices indexed row*cols+col
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			m.Vertices = append(m.Vertices, [3]float64{float64(c), float64(r), 0})
		}
	}
	for r := 0; r < rows-1; r++ {
		for c := 0; c < cols-1; c++ {
			i := r*cols + c
			m.Triangles = append(m.Triangles, models.Triangle{A: i, B: i + 1, C: i + cols})
			m.Triangles = append(m.Triangles, models.Triangle{A: i + 1, B: i + cols + 1, C: i + cols})
		}
	}
	return m
}
