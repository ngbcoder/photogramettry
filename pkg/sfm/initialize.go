package sfm

import "math"

// SelectSeedPair returns indices of two frames with most matches.
func SelectSeedPair(counts [][]int) (int, int) {
	n := len(counts)
	best := -1
	bi, bj := 0, 1
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			c := 0
			if i < len(counts) && j < len(counts[i]) {
				c = counts[i][j]
			}
			if c > best {
				best = c
				bi, bj = i, j
			}
		}
	}
	_ = math.Sqrt
	return bi, bj
}
