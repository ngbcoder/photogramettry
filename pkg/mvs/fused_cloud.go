package mvs

import (
	"github.com/ngbcoder/photogramettry/pkg/models"
)

// DepthToCloud converts a disparity map to a 3D point cloud (simple pinhole).
func DepthToCloud(disp [][]int, kFx, kFy, kCx, kCy, baseline float64) []models.Point3D {
	h := len(disp)
	if h == 0 {
		return nil
	}
	w := len(disp[0])
	var cloud []models.Point3D
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			d := disp[y][x]
			if d <= 0 {
				continue
			}
			z := (kFx * baseline) / float64(d)
			X := (float64(x) - kCx) * z / kFx
			Y := (float64(y) - kCy) * z / kFy
			cloud = append(cloud, models.Point3D{
				Position: [3]float64{X, Y, z},
				Color:    [3]uint8{128, 128, 128},
			})
		}
	}
	return cloud
}
