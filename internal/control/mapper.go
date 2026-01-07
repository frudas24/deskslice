// Package control handles input protocol and gesture mapping.
package control

import (
	"math"

	"github.com/frudas24/deskslice/internal/calib"
	"github.com/frudas24/deskslice/internal/monitor"
)

// NormToAbsPresetup maps normalized coordinates to absolute screen coords.
func NormToAbsPresetup(xn, yn float64, m monitor.Monitor) (int, int) {
	xn = clamp01(xn)
	yn = clamp01(yn)
	return m.X + normToPixels(xn, m.W), m.Y + normToPixels(yn, m.H)
}

// NormToAbsRun maps normalized coordinates to absolute plugin coordinates.
func NormToAbsRun(xn, yn float64, plugin calib.Rect) (int, int) {
	xn = clamp01(xn)
	yn = clamp01(yn)
	return plugin.X + normToPixels(xn, plugin.W), plugin.Y + normToPixels(yn, plugin.H)
}

func normToPixels(norm float64, span int) int {
	if span <= 1 {
		return 0
	}
	return int(math.Round(norm * float64(span-1)))
}

// clamp01 bounds a float to the [0..1] range.
func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
