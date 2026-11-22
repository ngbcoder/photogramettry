package mesh

import (
	"testing"

	"github.com/ngbcoder/photogramettry/pkg/models"
)

func TestDelaunay2D(t *testing.T) {
	pts := [][2]float64{{0, 0}, {1, 0}, {0, 1}, {1, 1}}
	tris := Delaunay2D(pts)
	if len(tris) == 0 {
		t.Error("expected triangles")
	}
	for _, tr := range tris {
		if tr.A >= len(pts) || tr.B >= len(pts) || tr.C >= len(pts) {
			t.Errorf("invalid triangle %v", tr)
		}
	}
	_ = models.Mesh{}
}

func TestPlanarUV(t *testing.T) {
	m := models.Mesh{Vertices: [][3]float64{{0, 0, 0}, {1, 1, 0}}}
	uvs := PlanarUV(m)
	if len(uvs) != 2 {
		t.Error("uv count mismatch")
	}
}
