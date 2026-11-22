package mvs

import "testing"

func TestComputeDepthMap(t *testing.T) {
	left := [][]uint8{{10, 10, 10}, {10, 10, 10}, {10, 10, 10}}
	right := [][]uint8{{10, 10, 10}, {10, 10, 10}, {10, 10, 10}}
	disp := ComputeDepthMap(left, right, 1, 4)
	if len(disp) != 3 {
		t.Error("bad depth map")
	}
}
