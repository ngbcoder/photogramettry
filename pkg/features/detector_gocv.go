//go:build opencv

package features

import (
	"fmt"

	"gocv.io/x/gocv"

	"github.com/ngbcoder/photogramettry/pkg/models"
)

// GOCVDetector wraps GoCV feature detection.
type GOCVDetector struct {
	algo string
}

// NewDetector creates a GoCV-backed detector.
func NewDetector(algo string) (Detector, error) {
	switch algo {
	case "orb":
		return &GOCVDetector{algo: "orb"}, nil
	case "akaze", "":
		return &GOCVDetector{algo: "akaze"}, nil
	default:
		return nil, fmt.Errorf("unknown detector: %s", algo)
	}
}

func (d *GOCVDetector) Name() string { return d.algo }

func (d *GOCVDetector) Detect(path string) (models.FeatureSet, error) {
	img := gocv.IMRead(path, gocv.IMReadColor)
	if img.Empty() {
		return models.FeatureSet{}, fmt.Errorf("cannot read image: %s", path)
	}
	defer img.Close()

	var kps []gocv.KeyPoint
	var desc gocv.Mat
	if d.algo == "orb" {
		orb := gocv.NewORB()
		defer orb.Close()
		kps, desc = orb.DetectAndCompute(img, gocv.NewMat())
	} else {
		akaze := gocv.NewAKAZE()
		defer akaze.Close()
		kps, desc = akaze.DetectAndCompute(img, gocv.NewMat())
	}
	defer desc.Close()

	pts := make([]models.Point2, len(kps))
	for i, kp := range kps {
		pts[i] = models.Point2{X: float64(kp.X), Y: float64(kp.Y)}
	}
	return models.FeatureSet{Keypoints: pts}, nil
}
