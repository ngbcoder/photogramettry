package mesh

import "github.com/ngbcoder/photogramettry/pkg/models"

// SmoothMesh applies Laplacian smoothing to mesh vertices.
func SmoothMesh(m models.Mesh, iterations int) models.Mesh {
	if iterations <= 0 {
		iterations = 1
	}
	// build adjacency
	adj := make([][]int, len(m.Vertices))
	for _, t := range m.Triangles {
		for _, e := range [][2]int{{t.A, t.B}, {t.B, t.C}, {t.C, t.A}} {
			adj[e[0]] = append(adj[e[0]], e[1])
			adj[e[1]] = append(adj[e[1]], e[0])
		}
	}
	v := make([][3]float64, len(m.Vertices))
	copy(v, m.Vertices)
	for it := 0; it < iterations; it++ {
		nv := make([][3]float64, len(v))
		for i := range v {
			if len(adj[i]) == 0 {
				nv[i] = v[i]
				continue
			}
			sx, sy, sz := 0.0, 0.0, 0.0
			for _, j := range adj[i] {
				sx += v[j][0]
				sy += v[j][1]
				sz += v[j][2]
			}
			n := float64(len(adj[i]))
			nv[i][0] = v[i][0]*0.5 + (sx/n)*0.5
			nv[i][1] = v[i][1]*0.5 + (sy/n)*0.5
			nv[i][2] = v[i][2]*0.5 + (sz/n)*0.5
		}
		v = nv
	}
	m.Vertices = v
	return m
}
