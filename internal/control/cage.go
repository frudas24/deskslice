// Package control handles input protocol and gesture mapping.
package control

import "github.com/frudas24/deskslice/internal/calib"

// CursorProvider can report the current OS cursor position.
type CursorProvider interface {
	CursorPos() (x, y int, ok bool)
}

// ClampPointToRect clamps (x,y) to stay inside rect.
func ClampPointToRect(rect calib.Rect, x, y int) (int, int) {
	rect = calib.Normalize(rect)
	if rect.W <= 0 || rect.H <= 0 {
		return x, y
	}
	minX := rect.X
	minY := rect.Y
	maxX := rect.X + rect.W - 1
	maxY := rect.Y + rect.H - 1
	if x < minX {
		x = minX
	}
	if x > maxX {
		x = maxX
	}
	if y < minY {
		y = minY
	}
	if y > maxY {
		y = maxY
	}
	return x, y
}

// RectCenter returns the center point of rect.
func RectCenter(rect calib.Rect) (int, int) {
	rect = calib.Normalize(rect)
	return rect.X + rect.W/2, rect.Y + rect.H/2
}
