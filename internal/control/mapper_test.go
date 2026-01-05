package control

import (
	"testing"

	"github.com/frudas24/deskslice/internal/calib"
	"github.com/frudas24/deskslice/internal/monitor"
)

// TestNormToAbsPresetup_TopLeft verifies the top-left mapping.
func TestNormToAbsPresetup_TopLeft(t *testing.T) {
	m := monitor.Monitor{X: 100, Y: 200, W: 300, H: 400}
	x, y := NormToAbsPresetup(0, 0, m)
	if x != 100 || y != 200 {
		t.Fatalf("expected (100,200), got (%d,%d)", x, y)
	}
}

// TestNormToAbsPresetup_Center verifies center mapping.
func TestNormToAbsPresetup_Center(t *testing.T) {
	m := monitor.Monitor{X: 100, Y: 200, W: 300, H: 400}
	x, y := NormToAbsPresetup(0.5, 0.5, m)
	if x != 250 || y != 400 {
		t.Fatalf("expected (250,400), got (%d,%d)", x, y)
	}
}

// TestNormToAbsPresetup_BottomRight verifies bottom-right mapping.
func TestNormToAbsPresetup_BottomRight(t *testing.T) {
	m := monitor.Monitor{X: 100, Y: 200, W: 300, H: 400}
	x, y := NormToAbsPresetup(1, 1, m)
	if x != 400 || y != 600 {
		t.Fatalf("expected (400,600), got (%d,%d)", x, y)
	}
}

// TestNormToAbsRun_TopLeft verifies run-mode top-left mapping.
func TestNormToAbsRun_TopLeft(t *testing.T) {
	r := calib.Rect{X: 10, Y: 20, W: 30, H: 40}
	x, y := NormToAbsRun(0, 0, r)
	if x != 10 || y != 20 {
		t.Fatalf("expected (10,20), got (%d,%d)", x, y)
	}
}

// TestNormToAbsRun_Center verifies run-mode center mapping.
func TestNormToAbsRun_Center(t *testing.T) {
	r := calib.Rect{X: 10, Y: 20, W: 30, H: 40}
	x, y := NormToAbsRun(0.5, 0.5, r)
	if x != 25 || y != 40 {
		t.Fatalf("expected (25,40), got (%d,%d)", x, y)
	}
}

// TestNormToAbsRun_BottomRight verifies run-mode bottom-right mapping.
func TestNormToAbsRun_BottomRight(t *testing.T) {
	r := calib.Rect{X: 10, Y: 20, W: 30, H: 40}
	x, y := NormToAbsRun(1, 1, r)
	if x != 40 || y != 60 {
		t.Fatalf("expected (40,60), got (%d,%d)", x, y)
	}
}

// TestNormToAbs_ClampOutOfRange verifies normalization clamps out-of-range values.
func TestNormToAbs_ClampOutOfRange(t *testing.T) {
	m := monitor.Monitor{X: 100, Y: 200, W: 300, H: 400}
	x, y := NormToAbsPresetup(-1, 2, m)
	if x != 100 || y != 600 {
		t.Fatalf("expected clamped (100,600), got (%d,%d)", x, y)
	}
}
