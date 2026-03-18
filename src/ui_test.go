package main

import (
	"image/color"
	"testing"
)

func TestClampIndex(t *testing.T) {
	if got := clampIndex(-1, 3); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
	if got := clampIndex(2, 3); got != 2 {
		t.Fatalf("expected 2, got %d", got)
	}
	if got := clampIndex(5, 3); got != 2 {
		t.Fatalf("expected 2, got %d", got)
	}
	if got := clampIndex(0, 0); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}

func TestItoa(t *testing.T) {
	cases := []struct {
		in   int
		want string
	}{
		{0, "0"},
		{7, "7"},
		{-7, "-7"},
		{12345, "12345"},
	}

	for _, tc := range cases {
		if got := itoa(tc.in); got != tc.want {
			t.Fatalf("itoa(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestGridCols(t *testing.T) {
	ui := NewUI(nil, 200, 200)
	if got := ui.gridCols(); got != 1 {
		t.Fatalf("expected 1 col, got %d", got)
	}

	ui.width = 900
	if got := ui.gridCols(); got < 1 || got > 4 {
		t.Fatalf("expected cols in [1,4], got %d", got)
	}
}

func TestMoveGrid(t *testing.T) {
	if got := moveGrid(0, 0, 1, 1, 2); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
	if got := moveGrid(0, 10, 0, 1, 3); got != 1 {
		t.Fatalf("expected 1, got %d", got)
	}
	if got := moveGrid(2, 10, 1, 0, 3); got != 5 {
		t.Fatalf("expected 5, got %d", got)
	}
	if got := moveGrid(8, 10, 1, 0, 3); got != 9 {
		t.Fatalf("expected 9, got %d", got)
	}
	if got := moveGrid(1, 10, 0, -1, 3); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("hello", 0); got != "hello" {
		t.Fatalf("expected original, got %q", got)
	}
	if got := truncate("hello", 2); got != "he" {
		t.Fatalf("expected he, got %q", got)
	}
	if got := truncate("hello", 3); got != "hel" {
		t.Fatalf("expected hel, got %q", got)
	}
	if got := truncate("hello", 4); got != "h..." {
		t.Fatalf("expected h..., got %q", got)
	}
}

func TestScaleColor(t *testing.T) {
	c := color.RGBA{R: 100, G: 150, B: 200, A: 255}
	if got := scaleColor(c, 2.0); got.R != 100 || got.G != 150 || got.B != 200 {
		t.Fatalf("expected clamped scale 1.0, got %+v", got)
	}
	if got := scaleColor(c, 0.1); got.R == 0 || got.G == 0 || got.B == 0 {
		// sanity check: should clamp to 0.2 so channels remain non-zero
		t.Fatalf("expected non-zero channels after clamp, got %+v", got)
	}
}
