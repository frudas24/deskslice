//go:build windows

// Package monitor describes display geometry and enumeration.
package monitor

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/lxn/win"
)

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	procEnumDisplayMonitors = user32.NewProc("EnumDisplayMonitors")
)

// ListMonitors returns the list of available displays using WinAPI.
func ListMonitors() ([]Monitor, error) {
	state := &enumState{}
	callback := syscall.NewCallback(state.enumProc)

	ret, _, err := procEnumDisplayMonitors.Call(0, 0, callback, 0)
	if ret == 0 {
		return nil, fmt.Errorf("EnumDisplayMonitors failed: %w", err)
	}
	if len(state.list) == 0 {
		return nil, fmt.Errorf("no monitors detected")
	}
	return state.list, nil
}

type enumState struct {
	list  []Monitor
	index int
}

func (s *enumState) enumProc(hMonitor win.HMONITOR, hdc win.HDC, rect *win.RECT, lparam uintptr) uintptr {
	var info win.MONITORINFO
	info.CbSize = uint32(unsafe.Sizeof(info))
	if !win.GetMonitorInfo(hMonitor, &info) {
		return 1
	}

	monitorRect := info.RcMonitor
	s.index++
	m := Monitor{
		Index:   s.index,
		X:       int(monitorRect.Left),
		Y:       int(monitorRect.Top),
		W:       int(monitorRect.Right - monitorRect.Left),
		H:       int(monitorRect.Bottom - monitorRect.Top),
		Primary: info.DwFlags&win.MONITORINFOF_PRIMARY != 0,
	}
	s.list = append(s.list, m)
	return 1
}
