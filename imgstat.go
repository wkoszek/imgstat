// Package imgstat provides image color analysis functions.
package imgstat

import (
	"image"
	"image/color"
	"math"
	"sort"
)

const sobelThresh = 30.0

// Px is an RGB pixel value.
type Px struct{ R, G, B uint8 }

// Entry is a color and its pixel count in an image.
type Entry struct {
	Px    Px
	Count int
}

// Stats holds image analysis metrics.
// Sharpness and EdgeDensity are populated by ConvStats, not CalcStats.
type Stats struct {
	MeanR, MeanG, MeanB float64
	StdR, StdG, StdB    float64
	Luma, StdLuma       float64
	Entropy             float64
	Saturation          float64
	CastR, CastG, CastB float64
	Colorfulness        float64
	DynRange            float64
	Sharpness           float64
	EdgeDensity         float64
}

func pixelAt(img image.Image, x, y int) Px {
	c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
	return Px{c.R, c.G, c.B}
}

// Histo builds a color frequency histogram from img.
// Returns entries sorted by frequency descending and total pixel count.
func Histo(img image.Image) ([]Entry, int) {
	bounds := img.Bounds()
	m := make(map[Px]int)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			m[pixelAt(img, x, y)]++
		}
	}
	es := make([]Entry, 0, len(m))
	for p, c := range m {
		es = append(es, Entry{p, c})
	}
	sort.Slice(es, func(i, j int) bool { return es[i].Count > es[j].Count })
	return es, bounds.Dx() * bounds.Dy()
}

// CalcStats computes scalar statistics from the color histogram.
func CalcStats(entries []Entry, total int) Stats {
	var sumR, sumG, sumB float64
	var sumR2, sumG2, sumB2 float64
	var sumL, sumL2, sumS float64
	var sumRG, sumYB, sumRG2, sumYB2 float64
	var entropy float64
	minL, maxL := math.MaxFloat64, -math.MaxFloat64

	for _, e := range entries {
		w := float64(e.Count) / float64(total)
		r := float64(e.Px.R)
		g := float64(e.Px.G)
		b := float64(e.Px.B)

		sumR += r * w
		sumG += g * w
		sumB += b * w
		sumR2 += r * r * w
		sumG2 += g * g * w
		sumB2 += b * b * w

		luma := 0.299*r + 0.587*g + 0.114*b
		sumL += luma * w
		sumL2 += luma * luma * w
		if luma < minL {
			minL = luma
		}
		if luma > maxL {
			maxL = luma
		}

		rn, gn, bn := r/255, g/255, b/255
		mx := math.Max(rn, math.Max(gn, bn))
		mn := math.Min(rn, math.Min(gn, bn))
		lt := (mx + mn) / 2
		if d := mx - mn; d > 0 {
			if lt < 0.5 {
				sumS += w * d / (mx + mn)
			} else {
				sumS += w * d / (2 - mx - mn)
			}
		}

		// Hasler-Süsstrunk colorfulness (rg = R-G, yb = 0.5(R+G)-B)
		rg := r - g
		yb := 0.5*(r+g) - b
		sumRG += rg * w
		sumYB += yb * w
		sumRG2 += rg * rg * w
		sumYB2 += yb * yb * w

		entropy -= w * math.Log2(w)
	}

	stdRG := math.Sqrt(math.Max(0, sumRG2-sumRG*sumRG))
	stdYB := math.Sqrt(math.Max(0, sumYB2-sumYB*sumYB))
	neutral := (sumR + sumG + sumB) / 3

	return Stats{
		MeanR: sumR, MeanG: sumG, MeanB: sumB,
		StdR:         math.Sqrt(math.Max(0, sumR2-sumR*sumR)),
		StdG:         math.Sqrt(math.Max(0, sumG2-sumG*sumG)),
		StdB:         math.Sqrt(math.Max(0, sumB2-sumB*sumB)),
		Luma:         sumL,
		StdLuma:      math.Sqrt(math.Max(0, sumL2-sumL*sumL)),
		Entropy:      entropy,
		Saturation:   sumS,
		CastR:        sumR - neutral,
		CastG:        sumG - neutral,
		CastB:        sumB - neutral,
		Colorfulness: math.Sqrt(stdRG*stdRG+stdYB*stdYB) + 0.3*math.Sqrt(sumRG*sumRG+sumYB*sumYB),
		DynRange:     (maxL - minL) / 255,
	}
}

// ConvStats computes Laplacian variance (sharpness) and Sobel edge density.
// It requires a second pass over the image because both metrics need neighbor access.
func ConvStats(img image.Image) (sharpness, edgeDensity float64) {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w < 3 || h < 3 {
		return 0, 0
	}

	luma := make([]float64, w*h)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			p := pixelAt(img, x, y)
			luma[(y-bounds.Min.Y)*w+(x-bounds.Min.X)] =
				0.299*float64(p.R) + 0.587*float64(p.G) + 0.114*float64(p.B)
		}
	}

	var sumL, sumL2 float64
	var edgeCount int
	n := (w - 2) * (h - 2)

	for y := 1; y < h-1; y++ {
		for x := 1; x < w-1; x++ {
			lap := luma[(y-1)*w+x] + luma[(y+1)*w+x] +
				luma[y*w+x-1] + luma[y*w+x+1] - 4*luma[y*w+x]
			sumL += lap
			sumL2 += lap * lap

			gx := -luma[(y-1)*w+x-1] + luma[(y-1)*w+x+1] +
				-2*luma[y*w+x-1] + 2*luma[y*w+x+1] +
				-luma[(y+1)*w+x-1] + luma[(y+1)*w+x+1]
			gy := -luma[(y-1)*w+x-1] - 2*luma[(y-1)*w+x] - luma[(y-1)*w+x+1] +
				luma[(y+1)*w+x-1] + 2*luma[(y+1)*w+x] + luma[(y+1)*w+x+1]
			if math.Sqrt(gx*gx+gy*gy) > sobelThresh {
				edgeCount++
			}
		}
	}

	mean := sumL / float64(n)
	return sumL2/float64(n) - mean*mean, float64(edgeCount) / float64(n)
}

func pxDist(a, b Px) float64 {
	dr := float64(int(a.R) - int(b.R))
	dg := float64(int(a.G) - int(b.G))
	db := float64(int(a.B) - int(b.B))
	return dr*dr + dg*dg + db*db
}

// Kmeans clusters entries into k dominant colors using weighted k-means on the histogram.
func Kmeans(entries []Entry, k int) []Entry {
	if k >= len(entries) {
		return entries
	}
	centroids := make([]Px, k)
	for i := range centroids {
		centroids[i] = entries[i].Px
	}
	assign := make([]int, len(entries))

	for range 20 {
		changed := false
		for i, e := range entries {
			best, bestD := 0, math.MaxFloat64
			for ci, c := range centroids {
				if d := pxDist(e.Px, c); d < bestD {
					best, bestD = ci, d
				}
			}
			if assign[i] != best {
				assign[i] = best
				changed = true
			}
		}
		if !changed {
			break
		}
		sums := make([][3]float64, k)
		weights := make([]float64, k)
		for i, e := range entries {
			wt := float64(e.Count)
			sums[assign[i]][0] += float64(e.Px.R) * wt
			sums[assign[i]][1] += float64(e.Px.G) * wt
			sums[assign[i]][2] += float64(e.Px.B) * wt
			weights[assign[i]] += wt
		}
		for i := range centroids {
			if weights[i] == 0 {
				continue
			}
			centroids[i] = Px{
				uint8(sums[i][0] / weights[i]),
				uint8(sums[i][1] / weights[i]),
				uint8(sums[i][2] / weights[i]),
			}
		}
	}

	buckets := make([]int, k)
	for i, e := range entries {
		buckets[assign[i]] += e.Count
	}
	result := make([]Entry, k)
	for i, c := range centroids {
		result[i] = Entry{c, buckets[i]}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Count > result[j].Count })
	return result
}
