// Package monitor describes display geometry and enumeration.
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

// GetMonitorByIndex returns the monitor matching the 1-based index.
func GetMonitorByIndex(list []Monitor, idx int) (Monitor, bool) {
	for _, m := range list {
		if m.Index == idx {
			return m, true
		}
	}
	return Monitor{}, false
}
