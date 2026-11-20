package features

import (
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"

	"github.com/ngbcoder/photogramettry/pkg/models"
)

// FastBriefDetector is a pure-Go FAST corner + BRIEF descriptor detector.
type FastBriefDetector struct {
	Threshold int
	MaxKp     int
}

// NewDetector returns a pure-Go detector (algo ignored for now).
func NewDetector(algo string) (Detector, error) {
	return &FastBriefDetector{Threshold: 20, MaxKp: 2000}, nil
}

func (d *FastBriefDetector) Name() string { return "fast-brief" }

// circle offsets for FAST (radius 3, 16 points).
var fastCircle = [][2]int{
	{0, -3}, {1, -3}, {2, -2}, {3, -1},
	{3, 0}, {3, 1}, {2, 2}, {1, 3},
	{0, 3}, {-1, 3}, {-2, 2}, {-3, 1},
	{-3, 0}, {-3, -1}, {-2, -2}, {-1, -3},
}

// BRIEF pair offsets (subset of 256).
var briefPairs = func() [][4]int {
	// deterministic pseudo-random pairs within a 31x31 patch
	pairs := make([][4]int, 256)
	// simple LCG
	seed := uint32(12345)
	next := func() int {
		seed = seed*1664525 + 1013904223
		return int(seed)
	}
	for i := range pairs {
		for {
			x1 := next()%31 - 15
			y1 := next()%31 - 15
			x2 := next()%31 - 15
			y2 := next()%31 - 15
			if x1 != x2 || y1 != y2 {
				pairs[i] = [4]int{x1, y1, x2, y2}
				break
			}
		}
	}
	return pairs
}()

func (d *FastBriefDetector) Detect(path string) (models.FeatureSet, error) {
	f, err := os.Open(path)
	if err != nil {
		return models.FeatureSet{}, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return models.FeatureSet{}, err
	}
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	// grayscale
	gray := make([][]uint8, h)
	for y := 0; y < h; y++ {
		row := make([]uint8, w)
		for x := 0; x < w; x++ {
			r, g, b, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			// luminance
			lum := (299*int(r>>8) + 587*int(g>>8) + 114*int(b>>8)) / 1000
			row[x] = uint8(lum)
		}
		gray[y] = row
	}

	// FAST detection
	type kp struct {
		x, y int
		score int
	}
	var keypoints []kp
	margin := 3
	for y := margin; y < h-margin; y++ {
		for x := margin; x < w-margin; x++ {
			center := int(gray[y][x])
			// check circle
			// We need 9 contiguous. Count runs on the 16-pixel circle.
			contigBright := 0
			contigDark := 0
			maxBright := 0
			maxDark := 0
			for i := 0; i < 16; i++ {
				ox, oy := fastCircle[i][0], fastCircle[i][1]
				px := x + ox
				py := y + oy
				if px < 0 || py < 0 || px >= w || py >= h {
					continue
				}
				val := int(gray[py][px])
				if val > center+d.Threshold {
					contigBright++
					if contigBright > maxBright {
						maxBright = contigBright
					}
					contigDark = 0
				} else if val < center-d.Threshold {
					contigDark++
					if contigDark > maxDark {
						maxDark = contigDark
					}
					contigBright = 0
				} else {
					contigBright = 0
					contigDark = 0
				}
			}
			// wrap-around check: concatenate first 8 to end to detect contiguous across boundary
			// (simplified: require max run >= 9)
			if maxBright >= 9 || maxDark >= 9 {
				keypoints = append(keypoints, kp{x: x, y: y, score: max(maxBright, maxDark)})
			}
		}
	}

	// Non-maximum suppression by score in 3x3 (simple)
	filtered := []kp{}
	for _, k := range keypoints {
		isMax := true
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if dx == 0 && dy == 0 {
					continue
				}
				nx, ny := k.x+dx, k.y+dy
				if nx < margin || ny < margin || nx >= w-margin || ny >= h-margin {
					continue
				}
				// find neighbor score
				for _, k2 := range keypoints {
					if k2.x == nx && k2.y == ny && k2.score > k.score {
						isMax = false
						break
					}
				}
				if !isMax {
					break
				}
			}
			if !isMax {
				break
			}
		}
		if isMax {
			filtered = append(filtered, k)
		}
	}
	if len(filtered) > d.MaxKp {
		// sort by score desc
		// simple selection
		// bubble top
		for i := 0; i < d.MaxKp; i++ {
			for j := i + 1; j < len(filtered); j++ {
				if filtered[j].score > filtered[i].score {
					filtered[i], filtered[j] = filtered[j], filtered[i]
				}
			}
		}
		filtered = filtered[:d.MaxKp]
	}

	// Build output + BRIEF descriptors
	pts := make([]models.Point2, len(filtered))
	descs := make([][]float64, len(filtered))
	for i, k := range filtered {
		pts[i] = models.Point2{X: float64(k.x), Y: float64(k.y)}
		desc := make([]float64, 256)
		for b := 0; b < 256; b++ {
			p := briefPairs[b]
			x1 := k.x + p[0]
			y1 := k.y + p[1]
			x2 := k.x + p[2]
			y2 := k.y + p[3]
			if x1 < 0 || y1 < 0 || x2 < 0 || y2 < 0 || x1 >= w || y1 >= h || x2 >= w || y2 >= h {
				desc[b] = 0
				continue
			}
			if gray[y1][x1] < gray[y2][x2] {
				desc[b] = 1
			} else {
				desc[b] = 0
			}
		}
		descs[i] = desc
	}
	return models.FeatureSet{Keypoints: pts, Descriptors: descs}, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
