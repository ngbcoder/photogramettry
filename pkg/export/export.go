package export

import (
	"fmt"
	"os"

	"github.com/ngbcoder/photogramettry/pkg/models"
)

// WritePLY writes a point cloud to a PLY file (ASCII).
func WritePLY(path string, cloud []models.Point3D) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	fmt.Fprintf(f, "ply\nformat ascii 1.0\nelement vertex %d\n", len(cloud))
	fmt.Fprintf(f, "property float x\nproperty float y\nproperty float z\n")
	fmt.Fprintf(f, "property uchar red\nproperty uchar green\nproperty uchar blue\n")
	fmt.Fprintf(f, "end_header\n")
	for _, p := range cloud {
		fmt.Fprintf(f, "%f %f %f %d %d %d\n", p.Position[0], p.Position[1], p.Position[2], p.Color[0], p.Color[1], p.Color[2])
	}
	return nil
}

// WriteOBJ writes a mesh to OBJ.
func WriteOBJ(path string, m models.Mesh) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, v := range m.Vertices {
		fmt.Fprintf(f, "v %f %f %f\n", v[0], v[1], v[2])
	}
	for _, t := range m.Triangles {
		fmt.Fprintf(f, "f %d %d %d\n", t.A+1, t.B+1, t.C+1)
	}
	return nil
}
