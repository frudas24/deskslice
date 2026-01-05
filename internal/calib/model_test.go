package calib

import "testing"

// TestNormalizeRect_Positive verifies Normalize keeps positive sizes intact.
func TestNormalizeRect_Positive(t *testing.T) {
	in := Rect{X: 1, Y: 2, W: 3, H: 4}
	out := Normalize(in)
	if out != in {
		t.Fatalf("expected %+v, got %+v", in, out)
	}
}

// TestNormalizeRect_NegativeDims verifies Normalize flips negative sizes.
func TestNormalizeRect_NegativeDims(t *testing.T) {
	in := Rect{X: 10, Y: 20, W: -5, H: -6}
	out := Normalize(in)
	want := Rect{X: 5, Y: 14, W: 5, H: 6}
	if out != want {
		t.Fatalf("expected %+v, got %+v", want, out)
	}
}

// TestContains_Inside verifies a point inside the rect is reported as contained.
func TestContains_Inside(t *testing.T) {
	r := Rect{X: 10, Y: 20, W: 5, H: 4}
	if !Contains(r, 12, 22) {
		t.Fatalf("expected point to be inside rect")
	}
}

// TestContains_Edges verifies edges are treated as inside the rect.
func TestContains_Edges(t *testing.T) {
	r := Rect{X: 10, Y: 20, W: 5, H: 4}
	if !Contains(r, 10, 20) {
		t.Fatalf("expected top-left edge to be inside rect")
	}
	if !Contains(r, 15, 24) {
		t.Fatalf("expected bottom-right edge to be inside rect")
	}
}

// TestContains_Outside verifies points outside the rect are rejected.
func TestContains_Outside(t *testing.T) {
	r := Rect{X: 10, Y: 20, W: 5, H: 4}
	if Contains(r, 9, 20) || Contains(r, 16, 25) {
		t.Fatalf("expected point to be outside rect")
	}
}
