package imgstat_test

import (
	"image"
	"image/color"
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
