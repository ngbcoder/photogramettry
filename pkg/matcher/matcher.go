package matcher

import (
	"math"

	"github.com/ngbcoder/photogramettry/pkg/models"
)

// Matcher performs brute-force feature matching.
type Matcher struct {
	RatioThreshold float64
}

// NewMatcher creates a matcher.
func NewMatcher(ratio float64) *Matcher {
	if ratio <= 0 {
		ratio = 0.75
	}
	return &Matcher{RatioThreshold: ratio}
}

// euclidean distance between descriptor vectors.
func descDist(a, b []float64) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return math.Inf(1)
	}
	s := 0.0
	for i := range a {
		d := a[i] - b[i]
		s += d * d
	}
	return math.Sqrt(s)
}

// Match matches features between two frames.
func (m *Matcher) Match(frameA, frameB int, a, b models.FeatureSet) models.MatchPair {
	pair := models.MatchPair{FrameIDA: frameA, FrameIDB: frameB}
	nA := len(a.Keypoints)
	nB := len(b.Keypoints)
	if nA == 0 || nB == 0 {
		return pair
	}

	// fallback: if no descriptors, match by index modulo
	hasDesc := len(a.Descriptors) == nA && len(b.Descriptors) == nB && len(a.Descriptors[0]) > 0

	for i := 0; i < nA; i++ {
		// find two nearest in B
		best1, best2 := -1, -1
		bestD1, bestD2 := math.Inf(1), math.Inf(1)
		for j := 0; j < nB; j++ {
			var d float64
			if hasDesc {
				d = descDist(a.Descriptors[i], b.Descriptors[j])
			} else {
				// spatial nearest as fallback
				dx := a.Keypoints[i].X - b.Keypoints[j].X
				dy := a.Keypoints[i].Y - b.Keypoints[j].Y
				d = math.Sqrt(dx*dx + dy*dy)
			}
			if d < bestD1 {
				bestD2 = bestD1
				best2 = best1
				bestD1 = d
				best1 = j
			} else if d < bestD2 {
				bestD2 = d
				best2 = j
			}
		}
		if best1 < 0 {
			continue
		}
		ratio := bestD1 / bestD2
		if best2 >= 0 && ratio < m.RatioThreshold {
			pair.Matches = append(pair.Matches, models.Correspondence{QueryIdx: i, TrainIdx: best1})
		} else if best2 < 0 {
			pair.Matches = append(pair.Matches, models.Correspondence{QueryIdx: i, TrainIdx: best1})
		}
	}
	return pair
}
