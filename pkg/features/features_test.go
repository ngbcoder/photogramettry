package features

import (
	"os"
	"testing"
)

func TestFastDetector(t *testing.T) {
	dir := "/tmp/opencode/testimgs"
	if _, err := os.Stat(dir + "/img1.jpg"); err != nil {
		t.Skip("no test image")
	}
	det, err := NewDetector("fast")
	if err != nil {
		t.Fatal(err)
	}
	fs, err := det.Detect(dir + "/img1.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if len(fs.Keypoints) == 0 {
		t.Error("no keypoints")
	}
	if len(fs.Descriptors) != len(fs.Keypoints) {
		t.Errorf("descriptor count mismatch: %d vs %d", len(fs.Descriptors), len(fs.Keypoints))
	}
}
