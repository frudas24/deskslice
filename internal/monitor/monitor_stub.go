//go:build !windows

// Package monitor describes display geometry and enumeration.
package monitor

import "fmt"

// ListMonitors returns an error on non-Windows platforms.
func ListMonitors() ([]Monitor, error) {
	return nil, fmt.Errorf("ListMonitors is only supported on Windows")
}
