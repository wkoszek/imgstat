package imgstat_test

import (
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/wkoszek/imgstat"
)

func TestHistoUsesStraightRGBForAlphaPixels(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, G: 0, B: 0, A: 128})

	entries, total := imgstat.Histo(img)
	if total != 1 {
		t.Fatalf("total = %d, want 1", total)
	}
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}
	if got, want := entries[0].Px, (imgstat.Px{R: 255, G: 0, B: 0}); got != want {
		t.Fatalf("pixel = %#v, want %#v", got, want)
	}
}

func TestCalcStatsEmptyImageReturnsZeroValues(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 0, 0))

	entries, total := imgstat.Histo(img)
	stats := imgstat.CalcStats(entries, total)

	if total != 0 {
		t.Fatalf("total = %d, want 0", total)
	}
	if len(entries) != 0 {
		t.Fatalf("len(entries) = %d, want 0", len(entries))
	}
	if stats != (imgstat.Stats{}) {
		t.Fatalf("stats = %#v, want zero value", stats)
	}
	if math.IsInf(stats.DynRange, 0) || math.IsNaN(stats.DynRange) {
		t.Fatalf("dyn range = %v, want finite zero", stats.DynRange)
	}
}

func TestKmeansZeroClustersReturnsNil(t *testing.T) {
	entries := []imgstat.Entry{{Px: imgstat.Px{R: 1, G: 2, B: 3}, Count: 1}}

	got := imgstat.Kmeans(entries, 0)
	if got != nil {
		t.Fatalf("Kmeans(..., 0) = %#v, want nil", got)
	}
}
