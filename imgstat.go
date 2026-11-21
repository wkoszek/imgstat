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
	if total <= 0 {
		return Stats{}
	}

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

// --- Oklab helpers ---------------------------------------------------------

type oklab struct{ L, A, B float64 }

func srgbLinearize(c float64) float64 {
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func pixelToOklab(px Px) oklab {
	r := srgbLinearize(float64(px.R) / 255)
	g := srgbLinearize(float64(px.G) / 255)
	b := srgbLinearize(float64(px.B) / 255)
	l := math.Cbrt(0.4122214708*r + 0.5363325363*g + 0.0514459929*b)
	m := math.Cbrt(0.2119034982*r + 0.6806995451*g + 0.1073969566*b)
	s := math.Cbrt(0.0883024619*r + 0.2817188376*g + 0.6299787005*b)
	return oklab{
		L: 0.2104542553*l + 0.7936177850*m - 0.0040720468*s,
		A: 1.9779984951*l - 2.4285922050*m + 0.4505937099*s,
		B: 0.0259040371*l + 0.7827717662*m - 0.8086757660*s,
	}
}

func oklabChroma(c oklab) float64 { return math.Sqrt(c.A*c.A + c.B*c.B) }

func oklabDist(a, b oklab) float64 {
	dL, dA, dB := a.L-b.L, a.A-b.A, a.B-b.B
	return math.Sqrt(dL*dL + dA*dA + dB*dB)
}

// oklabDistW weights the chromatic (a,b) axes 2x relative to lightness.
func oklabDistW(a, b oklab) float64 {
	dL, dA, dB := a.L-b.L, (a.A-b.A)*2, (a.B-b.B)*2
	return math.Sqrt(dL*dL + dA*dA + dB*dB)
}

func mergeOklab(a, b oklab, wa, wb float64) oklab {
	t := wa + wb
	return oklab{(a.L*wa + b.L*wb) / t, (a.A*wa + b.A*wb) / t, (a.B*wa + b.B*wb) / t}
}

type hCluster struct {
	centroid oklab
	pixels   []Px
	weight   int
}

// PaletteHinton extracts a 5-color palette using Amanda Hinton's algorithm
// (https://amandahinton.com/blog/creating-a-color-palette-from-an-image).
// It works in Oklab, uses k-means with k=14, then merges/filters down to 5.
func PaletteHinton(img image.Image) []Entry {
	const (
		kInit         = 14
		maxSamples    = 90_000
		mergeThresh   = 0.07
		phantomMass   = 0.025
		phantomChroma = 0.05
		targetN       = 5
		rescueDist    = 0.07
		rescueMass    = 0.001
		chromaSelect  = 0.03
		achroThresh   = 0.02
	)

	// 1. Sample pixels (stride-subsample when larger than maxSamples)
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	total := w * h
	capacity := min(total, maxSamples)
	samples := make([]Px, 0, capacity)
	labs := make([]oklab, 0, capacity)

	if total <= maxSamples {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				px := pixelAt(img, x, y)
				samples = append(samples, px)
				labs = append(labs, pixelToOklab(px))
			}
		}
	} else {
		step := float64(total) / float64(maxSamples)
		for f := 0.0; f < float64(total); f += step {
			i := int(f)
			px := pixelAt(img, bounds.Min.X+i%w, bounds.Min.Y+i/w)
			samples = append(samples, px)
			labs = append(labs, pixelToOklab(px))
		}
	}

	n := len(samples)
	if n == 0 {
		return nil
	}

	// 2. Deterministic k-means++ seeding: start at center, pick max-distance next
	k := min(kInit, n)
	centroids := make([]oklab, k)
	centroids[0] = labs[n/2]
	for ci := 1; ci < k; ci++ {
		best, bestD := 0, -1.0
		for i, lab := range labs {
			minD := math.MaxFloat64
			for _, c := range centroids[:ci] {
				if d := oklabDistW(lab, c); d < minD {
					minD = d
				}
			}
			if minD > bestD {
				bestD = minD
				best = i
			}
		}
		centroids[ci] = labs[best]
	}

	// 3. K-means iteration (Oklab, chromatic-weighted distance)
	assign := make([]int, n)
	for range 50 {
		changed := false
		for i, lab := range labs {
			best, bestD := 0, math.MaxFloat64
			for ci, c := range centroids {
				if d := oklabDistW(lab, c); d < bestD {
					bestD = d
					best = ci
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
		counts := make([]int, k)
		for i, lab := range labs {
			ci := assign[i]
			sums[ci][0] += lab.L
			sums[ci][1] += lab.A
			sums[ci][2] += lab.B
			counts[ci]++
		}
		for ci := range centroids {
			if counts[ci] == 0 {
				continue
			}
			centroids[ci] = oklab{
				sums[ci][0] / float64(counts[ci]),
				sums[ci][1] / float64(counts[ci]),
				sums[ci][2] / float64(counts[ci]),
			}
		}
	}

	clusters := make([]hCluster, k)
	for ci := range clusters {
		clusters[ci].centroid = centroids[ci]
	}
	for i, px := range samples {
		ci := assign[i]
		clusters[ci].pixels = append(clusters[ci].pixels, px)
		clusters[ci].weight++
	}

	// 4. Natural merge: collapse pairs within mergeThresh (chromatic-weighted)
	for {
		merged := false
		for i := range clusters {
			if clusters[i].weight == 0 {
				continue
			}
			for j := i + 1; j < len(clusters); j++ {
				if clusters[j].weight == 0 {
					continue
				}
				if oklabDistW(clusters[i].centroid, clusters[j].centroid) >= mergeThresh {
					continue
				}
				wi, wj := float64(clusters[i].weight), float64(clusters[j].weight)
				clusters[i].centroid = mergeOklab(clusters[i].centroid, clusters[j].centroid, wi, wj)
				clusters[i].pixels = append(clusters[i].pixels, clusters[j].pixels...)
				clusters[i].weight += clusters[j].weight
				clusters[j] = hCluster{}
				merged = true
			}
		}
		if !merged {
			break
		}
	}

	compacted := clusters[:0]
	for _, c := range clusters {
		if c.weight > 0 {
			compacted = append(compacted, c)
		}
	}
	clusters = compacted

	// 5. Phantom guard: drop tiny low-chroma clusters
	live := clusters[:0]
	for _, c := range clusters {
		if float64(c.weight)/float64(n) < phantomMass && oklabChroma(c.centroid) < phantomChroma {
			continue
		}
		live = append(live, c)
	}
	clusters = live

	// 6. Closest-pair merge down to targetN
	for len(clusters) > targetN {
		minD := math.MaxFloat64
		mi, mj := 0, 1
		for i := range clusters {
			for j := i + 1; j < len(clusters); j++ {
				if d := oklabDistW(clusters[i].centroid, clusters[j].centroid); d < minD {
					minD = d
					mi, mj = i, j
				}
			}
		}
		wi, wj := float64(clusters[mi].weight), float64(clusters[mj].weight)
		clusters[mi].centroid = mergeOklab(clusters[mi].centroid, clusters[mj].centroid, wi, wj)
		clusters[mi].pixels = append(clusters[mi].pixels, clusters[mj].pixels...)
		clusters[mi].weight += clusters[mj].weight
		clusters = append(clusters[:mj], clusters[mj+1:]...)
	}

	// 7. Rescue pass: add underrepresented regions if still below targetN
	for len(clusters) < targetN {
		bestD, bestIdx := 0.0, -1
		for i, lab := range labs {
			minD := math.MaxFloat64
			for _, c := range clusters {
				if d := oklabDist(lab, c.centroid); d < minD {
					minD = d
				}
			}
			if minD > bestD {
				bestD = minD
				bestIdx = i
			}
		}
		if bestIdx < 0 || bestD < rescueDist {
			break
		}
		newCentroid := labs[bestIdx]
		var newPixels []Px
		newW := 0
		for i, lab := range labs {
			if oklabDist(lab, newCentroid) < rescueDist {
				newPixels = append(newPixels, samples[i])
				newW++
			}
		}
		if float64(newW)/float64(n) < rescueMass {
			break
		}
		clusters = append(clusters, hCluster{centroid: newCentroid, pixels: newPixels, weight: newW})
	}

	// 8. Representative pixel selection per cluster
	result := make([]Entry, len(clusters))
	for ci, c := range clusters {
		var chosen Px
		if oklabChroma(c.centroid) >= chromaSelect {
			// Chromatic: pixel with highest chroma in cluster
			bestC := -1.0
			for _, px := range c.pixels {
				if ch := oklabChroma(pixelToOklab(px)); ch > bestC {
					bestC = ch
					chosen = px
				}
			}
		} else {
			// Grayscale: pixel closest to centroid (avoids warm/cool outliers)
			bestD := math.MaxFloat64
			for _, px := range c.pixels {
				if d := oklabDist(pixelToOklab(px), c.centroid); d < bestD {
					bestD = d
					chosen = px
				}
			}
		}
		result[ci] = Entry{Px: chosen, Count: c.weight}
	}

	// 9. Sort: achromatic by lightness, chromatic by hue angle
	sort.Slice(result, func(i, j int) bool {
		li := pixelToOklab(result[i].Px)
		lj := pixelToOklab(result[j].Px)
		iAchro := oklabChroma(li) < achroThresh
		jAchro := oklabChroma(lj) < achroThresh
		if iAchro != jAchro {
			return iAchro
		}
		if iAchro {
			return li.L < lj.L
		}
		return math.Atan2(li.B, li.A) < math.Atan2(lj.B, lj.A)
	})

	return result
}

// ---------------------------------------------------------------------------

func pxDist(a, b Px) float64 {
	dr := float64(int(a.R) - int(b.R))
	dg := float64(int(a.G) - int(b.G))
	db := float64(int(a.B) - int(b.B))
	return dr*dr + dg*dg + db*db
}

// Kmeans clusters entries into k dominant colors using weighted k-means on the histogram.
func Kmeans(entries []Entry, k int) []Entry {
	if k <= 0 {
		return nil
	}
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
