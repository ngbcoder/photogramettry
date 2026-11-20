//go:build !opencv

package features

// NewDetector returns a noop detector when OpenCV is not available.
func NewDetector(algo string) (Detector, error) {
	return NoopDetector{}, nil
}
