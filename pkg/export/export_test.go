package export

import (
	"os"
	"testing"

	"github.com/ngbcoder/photogramettry/pkg/models"
)

func TestWritePLY(t *testing.T) {
	path := t.TempDir() + "/t.ply"
	cloud := []models.Point3D{{Position: [3]float64{1, 2, 3}, Color: [3]uint8{255, 0, 0}}}
	if err := WritePLY(path, cloud); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	if len(b) == 0 {
		t.Error("empty ply")
	}
}

func TestWriteOBJ(t *testing.T) {
	path := t.TempDir() + "/t.obj"
	m := models.Mesh{Vertices: [][3]float64{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}}, Triangles: []models.Triangle{{0, 1, 2}}}
	if err := WriteOBJ(path, m); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	if len(b) == 0 {
		t.Error("empty obj")
	}
}
