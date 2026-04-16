package main

import (
	"flag"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"os"
	"sort"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

const (
	defaultTop   = 20
	defaultK     = 6
	sobelThresh  = 30.0
)

type px struct{ r, g, b uint8 }

type entry struct {
	px    px
	count int
}

type stats struct {
	meanR, meanG, meanB float64
	stdR, stdG, stdB    float64
	luma, stdLuma       float64
	entropy             float64
	saturation          float64
	castR, castG, castB float64
	colorfulness        float64
	dynRange            float64
	sharpness           float64
	edgeDensity         float64
}

func histo(img image.Image) ([]entry, int) {
	bounds := img.Bounds()
	m := make(map[px]int)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			m[px{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)}]++
		}
	}
	es := make([]entry, 0, len(m))
	for p, c := range m {
		es = append(es, entry{p, c})
	}
	sort.Slice(es, func(i, j int) bool { return es[i].count > es[j].count })
	return es, bounds.Dx() * bounds.Dy()
}

func calcStats(entries []entry, total int) stats {
	var sumR, sumG, sumB float64
	var sumR2, sumG2, sumB2 float64
	var sumL, sumL2, sumS float64
	var sumRG, sumYB, sumRG2, sumYB2 float64
	var entropy float64
	minL, maxL := math.MaxFloat64, -math.MaxFloat64

	for _, e := range entries {
		w := float64(e.count) / float64(total)
		r := float64(e.px.r)
		g := float64(e.px.g)
		b := float64(e.px.b)

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

	return stats{
		meanR: sumR, meanG: sumG, meanB: sumB,
		stdR:         math.Sqrt(math.Max(0, sumR2-sumR*sumR)),
		stdG:         math.Sqrt(math.Max(0, sumG2-sumG*sumG)),
		stdB:         math.Sqrt(math.Max(0, sumB2-sumB*sumB)),
		luma:         sumL,
		stdLuma:      math.Sqrt(math.Max(0, sumL2-sumL*sumL)),
		entropy:      entropy,
		saturation:   sumS,
		castR:        sumR - neutral,
		castG:        sumG - neutral,
		castB:        sumB - neutral,
		colorfulness: math.Sqrt(stdRG*stdRG+stdYB*stdYB) + 0.3*math.Sqrt(sumRG*sumRG+sumYB*sumYB),
		dynRange:     (maxL - minL) / 255,
	}
}

// convStats makes a second pass to compute Laplacian variance (sharpness)
// and Sobel edge density; these require neighbor access unavailable from histo.
func convStats(img image.Image) (sharpness, edgeDensity float64) {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w < 3 || h < 3 {
		return 0, 0
	}

	luma := make([]float64, w*h)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			luma[(y-bounds.Min.Y)*w+(x-bounds.Min.X)] =
				0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8)
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

func pxDist(a, b px) float64 {
	dr := float64(int(a.r) - int(b.r))
	dg := float64(int(a.g) - int(b.g))
	db := float64(int(a.b) - int(b.b))
	return dr*dr + dg*dg + db*db
}

func kmeans(entries []entry, k int) []entry {
	if k >= len(entries) {
		return entries
	}
	centroids := make([]px, k)
	for i := range centroids {
		centroids[i] = entries[i].px
	}
	assign := make([]int, len(entries))

	for range 20 {
		changed := false
		for i, e := range entries {
			best, bestD := 0, math.MaxFloat64
			for ci, c := range centroids {
				if d := pxDist(e.px, c); d < bestD {
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
			wt := float64(e.count)
			sums[assign[i]][0] += float64(e.px.r) * wt
			sums[assign[i]][1] += float64(e.px.g) * wt
			sums[assign[i]][2] += float64(e.px.b) * wt
			weights[assign[i]] += wt
		}
		for i := range centroids {
			if weights[i] == 0 {
				continue
			}
			centroids[i] = px{
				uint8(sums[i][0] / weights[i]),
				uint8(sums[i][1] / weights[i]),
				uint8(sums[i][2] / weights[i]),
			}
		}
	}

	buckets := make([]int, k)
	for i, e := range entries {
		buckets[assign[i]] += e.count
	}
	result := make([]entry, k)
	for i, c := range centroids {
		result[i] = entry{c, buckets[i]}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].count > result[j].count })
	return result
}

func report(name string, r io.Reader, n, k int) error {
	img, format, err := image.Decode(r)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	entries, total := histo(img)
	if n > len(entries) {
		n = len(entries)
	}
	s := calcStats(entries, total)
	s.sharpness, s.edgeDensity = convStats(img)

	fmt.Printf("# %s %s %dx%d %dpx %d colors\n", name, format, w, h, total, len(entries))
	fmt.Printf("# %6s %3s %3s %3s %7s %6s\n", "hex", "r", "g", "b", "n", "%")
	for _, e := range entries[:n] {
		pct := float64(e.count) * 100 / float64(total)
		fmt.Printf("  %06x %3d %3d %3d %7d %5.1f%%\n",
			int(e.px.r)<<16|int(e.px.g)<<8|int(e.px.b),
			e.px.r, e.px.g, e.px.b, e.count, pct)
	}
	fmt.Printf("# mean   %3.0f %3.0f %3.0f  luma %5.1f±%.1f  sat %.2f  entropy %.2f\n",
		s.meanR, s.meanG, s.meanB, s.luma, s.stdLuma, s.saturation, s.entropy)
	fmt.Printf("# stddev %3.0f %3.0f %3.0f  cast R%+.0f G%+.0f B%+.0f  colorful %.1f  dynrange %.2f\n",
		s.stdR, s.stdG, s.stdB, s.castR, s.castG, s.castB, s.colorfulness, s.dynRange)
	fmt.Printf("# sharp %.1f  edges %.1f%%\n", s.sharpness, s.edgeDensity*100)

	if k > 0 {
		palette := kmeans(entries, k)
		fmt.Printf("# palette k=%d\n", k)
		fmt.Printf("# %6s %3s %3s %3s %6s\n", "hex", "r", "g", "b", "%")
		for _, e := range palette {
			pct := float64(e.count) * 100 / float64(total)
			fmt.Printf("  %06x %3d %3d %3d %5.1f%%\n",
				int(e.px.r)<<16|int(e.px.g)<<8|int(e.px.b),
				e.px.r, e.px.g, e.px.b, pct)
		}
	}
	return nil
}

func main() {
	n := flag.Int("n", defaultTop, "top N colors")
	k := flag.Int("k", defaultK, "k-means palette size (0 to disable)")
	flag.Parse()

	failed := false
	args := flag.Args()
	if len(args) == 0 {
		if err := report("stdin", os.Stdin, *n, *k); err != nil {
			fmt.Fprintln(os.Stderr, err)
			failed = true
		}
	} else {
		for _, name := range args {
			f, err := os.Open(name)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				failed = true
				continue
			}
			if err := report(name, f, *n, *k); err != nil {
				fmt.Fprintln(os.Stderr, err)
				failed = true
			}
			f.Close()
		}
	}

	if failed {
		os.Exit(1)
	}
}
