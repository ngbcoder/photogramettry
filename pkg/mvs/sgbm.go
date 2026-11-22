package mvs

// ComputeDepthMap computes a disparity map using block matching (SAD).
func ComputeDepthMap(left, right [][]uint8, blockSize, maxDisp int) [][]int {
	h := len(left)
	if h == 0 {
		return nil
	}
	w := len(left[0])
	if blockSize < 1 {
		blockSize = 5
	}
	if maxDisp < 1 {
		maxDisp = 64
	}
	half := blockSize / 2
	out := make([][]int, h)
	for y := range out {
		out[y] = make([]int, w)
	}
	for y := half; y < h-half; y++ {
		for x := half; x < w-half; x++ {
			bestDisp := 0
			bestCost := 1 << 30
			for d := 0; d < maxDisp && x-d-half >= 0; d++ {
				cost := 0
				for by := -half; by <= half; by++ {
					for bx := -half; bx <= half; bx++ {
						lv := int(left[y+by][x+bx])
						rv := int(right[y+by][x-d+bx])
						diff := lv - rv
						if diff < 0 {
							diff = -diff
						}
						cost += diff
					}
				}
				if cost < bestCost {
					bestCost = cost
					bestDisp = d
				}
			}
			out[y][x] = bestDisp
		}
	}
	return out
}
