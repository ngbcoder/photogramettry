package features

import "github.com/ngbcoder/photogramettry/pkg/models"

// NoopDetector returns empty feature sets.
type NoopDetector struct{}

func (NoopDetector) Name() string { return "noop" }

func (NoopDetector) Detect(path string) (models.FeatureSet, error) {
	return models.FeatureSet{FrameID: 0}, nil
}
