package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"strings"
	"testing"
)

func TestHistoUsesStraightRGBForAlphaPixels(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, G: 0, B: 0, A: 128})

	entries, total := histo(img)
	if total != 1 {
		t.Fatalf("total = %d, want 1", total)
	}
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}
	if got, want := entries[0].px, (px{r: 255, g: 0, b: 0}); got != want {
		t.Fatalf("pixel = %#v, want %#v", got, want)
	}
}

func TestReportRejectsNegativeN(t *testing.T) {
	out, err := captureStdout(func() error {
		return report("test", bytes.NewReader(nil), -1, 0)
	})
	if err == nil {
		t.Fatal("report returned nil error, want failure for negative -n")
	}
	if !strings.Contains(err.Error(), "-n must be >= 0") {
		t.Fatalf("error = %q, want negative -n message", err)
	}
	if out != "" {
		t.Fatalf("stdout = %q, want no output", out)
	}
}

func TestReportPaletteHeaderUsesActualClusterCount(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 12, G: 34, B: 56, A: 255})

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode: %v", err)
	}

	out, err := captureStdout(func() error {
		return report("test.png", bytes.NewReader(buf.Bytes()), defaultTop, defaultK)
	})
	if err != nil {
		t.Fatalf("report: %v", err)
	}
	if !strings.Contains(out, "# palette k=1\n") {
		t.Fatalf("stdout missing actual palette size:\n%s", out)
	}
}

func captureStdout(fn func() error) (string, error) {
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	defer r.Close()

	old := os.Stdout
	os.Stdout = w
	defer func() {
		os.Stdout = old
	}()

	runErr := fn()
	if err := w.Close(); err != nil {
		return "", err
	}
	out, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	return string(out), runErr
}
