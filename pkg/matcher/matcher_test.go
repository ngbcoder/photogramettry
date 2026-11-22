package matcher

import (
	"testing"

	"github.com/ngbcoder/photogramettry/pkg/models"
)

func TestMatcherRatio(t *testing.T) {
	a := models.FeatureSet{
		Keypoints:   []models.Point2{{0, 0}, {1, 1}},
		Descriptors: [][]float64{{0, 0}, {1, 1}},
	}
	b := models.FeatureSet{
		Keypoints:   []models.Point2{{0, 0}, {1, 1}},
		Descriptors: [][]float64{{0, 0}, {1, 1}},
	}
	m := NewMatcher(0.75)
	pair := m.Match(0, 1, a, b)
	if len(pair.Matches) == 0 {
		t.Error("expected matches")
	}
}
