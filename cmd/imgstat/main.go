package main

import (
	"flag"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"

	"github.com/wkoszek/imgstat"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

const (
	defaultTop = 20
	defaultK   = 6
)

func report(name string, r io.Reader, n, k int) error {
	if n < 0 {
		return fmt.Errorf("-n must be >= 0")
	}
	if k < 0 {
		return fmt.Errorf("-k must be >= 0")
	}
	img, format, err := image.Decode(r)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	entries, total := imgstat.Histo(img)
	if n > len(entries) {
		n = len(entries)
	}
	s := imgstat.CalcStats(entries, total)
	s.Sharpness, s.EdgeDensity = imgstat.ConvStats(img)

	fmt.Printf("# %s %s %dx%d %dpx %d colors\n", name, format, w, h, total, len(entries))
	fmt.Printf("# %6s %3s %3s %3s %7s %6s\n", "hex", "r", "g", "b", "n", "%")
	for _, e := range entries[:n] {
		pct := float64(e.Count) * 100 / float64(total)
		fmt.Printf("  %06x %3d %3d %3d %7d %5.1f%%\n",
			int(e.Px.R)<<16|int(e.Px.G)<<8|int(e.Px.B),
			e.Px.R, e.Px.G, e.Px.B, e.Count, pct)
	}
	fmt.Printf("# mean   %3.0f %3.0f %3.0f  luma %5.1f±%.1f  sat %.2f  entropy %.2f\n",
		s.MeanR, s.MeanG, s.MeanB, s.Luma, s.StdLuma, s.Saturation, s.Entropy)
	fmt.Printf("# stddev %3.0f %3.0f %3.0f  cast R%+.0f G%+.0f B%+.0f  colorful %.1f  dynrange %.2f\n",
		s.StdR, s.StdG, s.StdB, s.CastR, s.CastG, s.CastB, s.Colorfulness, s.DynRange)
	fmt.Printf("# sharp %.1f  edges %.1f%%\n", s.Sharpness, s.EdgeDensity*100)

	if k > 0 {
		palette := imgstat.Kmeans(entries, k)
		fmt.Printf("# palette k=%d\n", len(palette))
		fmt.Printf("# %6s %3s %3s %3s %6s\n", "hex", "r", "g", "b", "%")
		for _, e := range palette {
			pct := float64(e.Count) * 100 / float64(total)
			fmt.Printf("  %06x %3d %3d %3d %5.1f%%\n",
				int(e.Px.R)<<16|int(e.Px.G)<<8|int(e.Px.B),
				e.Px.R, e.Px.G, e.Px.B, pct)
		}
	}
	return nil
}

func main() {
	n := flag.Int("n", defaultTop, "top N colors")
	k := flag.Int("k", defaultK, "k-means palette size (0 to disable)")
	flag.Parse()

	if *n < 0 {
		fmt.Fprintln(os.Stderr, "-n must be >= 0")
		os.Exit(2)
	}
	if *k < 0 {
		fmt.Fprintln(os.Stderr, "-k must be >= 0")
		os.Exit(2)
	}

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
