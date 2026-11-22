package mesh

import "github.com/ngbcoder/photogramettry/pkg/models"

// PlanarUV computes planar UV coordinates for mesh vertices by normalizing X,Y.
func PlanarUV(m models.Mesh) [][2]float64 {
	uvs := make([][2]float64, len(m.Vertices))
	if len(m.Vertices) == 0 {
		return uvs
	}
	minX, maxX, minY, maxY := m.Vertices[0][0], m.Vertices[0][0], m.Vertices[0][1], m.Vertices[0][1]
	for _, v := range m.Vertices {
		if v[0] < minX {
			minX = v[0]
		}
		if v[0] > maxX {
			maxX = v[0]
		}
		if v[1] < minY {
			minY = v[1]
		}
		if v[1] > maxY {
			maxY = v[1]
		}
	}
	rx := maxX - minX
	ry := maxY - minY
	if rx == 0 {
		rx = 1
	}
	if ry == 0 {
		ry = 1
	}
	for i, v := range m.Vertices {
		uvs[i][0] = (v[0] - minX) / rx
		uvs[i][1] = (v[1] - minY) / ry
	}
	return uvs
}
