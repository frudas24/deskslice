package calib

// Rect describes a rectangle using top-left origin and size.
type Rect struct {
	X int
	Y int
	W int
	H int
}

// Calib stores calibration data for the Codex panel and its sub-areas.
type Calib struct {
	MonitorIndex int
	PluginAbs    Rect
	ChatRel      Rect
	ScrollRel    Rect
}

// Normalize returns a rectangle with non-negative width/height.
func Normalize(r Rect) Rect {
	if r.W < 0 {
		r.X += r.W
		r.W = -r.W
	}
	if r.H < 0 {
		r.Y += r.H
		r.H = -r.H
	}
	return r
}

// Contains reports whether a point is inside the rectangle (edges inclusive).
func Contains(r Rect, x, y int) bool {
	if r.W <= 0 || r.H <= 0 {
		return false
	}
	maxX := r.X + r.W
	maxY := r.Y + r.H
	return x >= r.X && x <= maxX && y >= r.Y && y <= maxY
}
