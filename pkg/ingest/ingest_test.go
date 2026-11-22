package ingest

import (
	"os"
	"testing"
)

func TestIngestDir(t *testing.T) {
	dir := "/tmp/opencode/testimgs"
	if _, err := os.Stat(dir); err != nil {
		t.Skip("no test images")
	}
	frames, err := IngestDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) == 0 {
		t.Fatal("no frames")
	}
	for _, f := range frames {
		if f.Width == 0 || f.Height == 0 {
			t.Errorf("bad dims for %s", f.Path)
		}
		if f.Intrinsics.FocalLengthX == 0 {
			t.Errorf("missing fx for %s", f.Path)
		}
	}
}
