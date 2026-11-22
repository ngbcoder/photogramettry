package mesh

import "github.com/ngbcoder/photogramettry/pkg/models"

// Delaunay2D computes a 2D Delaunay triangulation of points (x,y) and returns triangles.
func Delaunay2D(pts [][2]float64) []models.Triangle {
	// Bowyer-Watson simplified.
	type tri struct{ a, b, c int }
	// super triangle
	minX, minY, maxX, maxY := pts[0][0], pts[0][1], pts[0][0], pts[0][1]
	for _, p := range pts {
		if p[0] < minX {
			minX = p[0]
		}
		if p[1] < minY {
			minY = p[1]
		}
		if p[0] > maxX {
			maxX = p[0]
		}
		if p[1] > maxY {
			maxY = p[1]
		}
	}
	dx := maxX - minX
	dy := maxY - minY
	dmax := dx
	if dy > dmax {
		dmax = dy
	}
	midX := (minX + maxX) / 2
	midY := (minY + maxY) / 2
	super := [][2]float64{
		{midX - 20*dmax, midY - dmax},
		{midX, midY + 20*dmax},
		{midX + 20*dmax, midY - dmax},
	}
	allPts := append([][2]float64{}, pts...)
	allPts = append(allPts, super...)
	si := len(pts)
	s1, s2, s3 := si, si+1, si+2

	tris := []tri{{s1, s2, s3}}


	for pi := 0; pi < len(pts); pi++ {
		p := pts[pi]
		// find bad triangles
		var bad []int
		for ti, t := range tris {
			a := allPts[t.a]
			b := allPts[t.b]
			c := allPts[t.c]
			if pointInCircumcircle(a, b, c, p) {
				bad = append(bad, ti)
			}
		}
		// collect boundary edges
		edgeCount := map[[2]int]int{}
		for _, ti := range bad {
			t := tris[ti]
			edges := [][2]int{{t.a, t.b}, {t.b, t.c}, {t.c, t.a}}
			for _, e := range edges {
				key := e
				if key[0] > key[1] {
					key[0], key[1] = key[1], key[0]
				}
				edgeCount[key]++
			}
		}
		// remove bad
		newTris := []tri{}
		for ti, t := range tris {
			badSet := false
			for _, b := range bad {
				if ti == b {
					badSet = true
					break
				}
			}
			if !badSet {
				newTris = append(newTris, t)
			}
		}
		tris = newTris
		// retriangulate
		for edge, cnt := range edgeCount {
			if cnt == 1 {
				tris = append(tris, tri{edge[0], edge[1], pi})
			}
		}
	}

	// remove triangles using super vertices
	out := []models.Triangle{}
	for _, t := range tris {
		if t.a >= len(pts) || t.b >= len(pts) || t.c >= len(pts) {
			continue
		}
		out = append(out, models.Triangle{A: t.a, B: t.b, C: t.c})
	}
	return out
}

func pointInCircumcircle(a, b, c, p [2]float64) bool {
	// ensure CCW orientation
	o := (b[0]-a[0])*(c[1]-a[1]) - (b[1]-a[1])*(c[0]-a[0])
	if o < 0 {
		b, c = c, b
	}
	ax, ay := a[0], a[1]
	bx, by := b[0], b[1]
	cx, cy := c[0], c[1]
	px, py := p[0], p[1]
	d1 := (ax - px)*(by - py) - (bx - px)*(ay - py)
	d2 := (bx - px)*(cy - py) - (cx - px)*(by - py)
	d3 := (cx - px)*(ay - py) - (ax - px)*(cy - py)
	// Not exact circumcircle but orientation; use determinant of 4x4
	_ = d1
	_ = d2
	_ = d3
	// Compute circumcircle determinant
	m := [][]float64{
		{ax, ay, ax*ax + ay*ay, 1},
		{bx, by, bx*bx + by*by, 1},
		{cx, cy, cx*cx + cy*cy, 1},
		{px, py, px*px + py*py, 1},
	}
	det := det4(m)
	return det > 0
}

func det4(m [][]float64) float64 {
	// compute determinant of 4x4
	a := m
	// use expansion
	det := a[0][0]*(a[1][1]*(a[2][2]*a[3][3]-a[2][3]*a[3][2]) - a[1][2]*(a[2][1]*a[3][3]-a[2][3]*a[3][1]) + a[1][3]*(a[2][1]*a[3][2]-a[2][2]*a[3][1])) -
		a[0][1]*(a[1][0]*(a[2][2]*a[3][3]-a[2][3]*a[3][2]) - a[1][2]*(a[2][0]*a[3][3]-a[2][3]*a[3][0]) + a[1][3]*(a[2][0]*a[3][2]-a[2][2]*a[3][0])) +
		a[0][2]*(a[1][0]*(a[2][1]*a[3][3]-a[2][3]*a[3][1]) - a[1][1]*(a[2][0]*a[3][3]-a[2][3]*a[3][0]) + a[1][3]*(a[2][0]*a[3][1]-a[2][1]*a[3][0])) -
		a[0][3]*(a[1][0]*(a[2][1]*a[3][2]-a[2][2]*a[3][1]) - a[1][1]*(a[2][0]*a[3][2]-a[2][2]*a[3][0]) + a[1][2]*(a[2][0]*a[3][1]-a[2][1]*a[3][0]))
	return det
}
