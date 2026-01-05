package monitor

// Monitor describes a display and its bounds.
type Monitor struct {
	Index   int
	X       int
	Y       int
	W       int
	H       int
	Primary bool
}
