package calib

import (
	"path/filepath"
	"testing"
)

// TestSaveLoad_RoundTrip verifies saving and loading preserves calibration data.
func TestSaveLoad_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "calib.json")
	in := Calib{
		MonitorIndex: 2,
		PluginAbs:    Rect{X: 1, Y: 2, W: 3, H: 4},
		ChatRel:      Rect{X: 5, Y: 6, W: 7, H: 8},
		ScrollRel:    Rect{X: 9, Y: 10, W: 11, H: 12},
	}

	if err := Save(path, in); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	out, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if out != in {
		t.Fatalf("expected %+v, got %+v", in, out)
	}
}

// TestLoad_MissingFile_ReturnsEmpty verifies missing files return zero data.
func TestLoad_MissingFile_ReturnsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")
	out, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if out != (Calib{}) {
		t.Fatalf("expected empty calib, got %+v", out)
	}
}
