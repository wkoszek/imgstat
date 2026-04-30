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

func TestReportRejectsNegativeN(t *testing.T) {
	out, err := captureStdout(func() error {
		return report("test", bytes.NewReader(nil), -1, 0, false)
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

func TestReportRejectsNegativeK(t *testing.T) {
	out, err := captureStdout(func() error {
		return report("test", bytes.NewReader(nil), 0, -1, false)
	})
	if err == nil {
		t.Fatal("report returned nil error, want failure for negative -k")
	}
	if !strings.Contains(err.Error(), "-k must be >= 0") {
		t.Fatalf("error = %q, want negative -k message", err)
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
		return report("test.png", bytes.NewReader(buf.Bytes()), defaultTop, defaultK, false)
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
	defer func() { os.Stdout = old }()

	runErr := fn()
	w.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	return string(out), runErr
}
