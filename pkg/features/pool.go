package features

import (
	"context"
	"runtime"

	"github.com/ngbcoder/photogramettry/pkg/models"
)

// Detector detects keypoints and descriptors for an image.
type Detector interface {
	Name() string
	Detect(path string) (models.FeatureSet, error)
}

// Pool runs detection over paths with bounded concurrency.
type Pool struct {
	detector Detector
	workers  int
}

// NewPool creates a worker pool.
func NewPool(detector Detector, workers int) *Pool {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	return &Pool{detector: detector, workers: workers}
}

// Run processes all paths and returns feature sets (order preserved).
func (p *Pool) Run(ctx context.Context, paths []string) ([]models.FeatureSet, error) {
	type result struct {
		idx int
		fs  models.FeatureSet
		err error
	}
	jobs := make(chan int, len(paths))
	results := make(chan result, len(paths))
	for i := range paths {
		jobs <- i
	}
	close(jobs)

	for w := 0; w < p.workers; w++ {
		go func() {
			for idx := range jobs {
				select {
				case <-ctx.Done():
					results <- result{idx: idx, err: ctx.Err()}
					return
				default:
				}
				fs, err := p.detector.Detect(paths[idx])
				results <- result{idx: idx, fs: fs, err: err}
			}
		}()
	}

	out := make([]models.FeatureSet, len(paths))
	for i := 0; i < len(paths); i++ {
		r := <-results
		if r.err != nil {
			return nil, r.err
		}
		out[r.idx] = r.fs
	}
	return out, nil
}
