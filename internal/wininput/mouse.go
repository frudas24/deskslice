//go:build windows

// Package wininput defines Windows input injection interfaces.
package wininput

import "github.com/lxn/win"

// MoveAbs moves the cursor to an absolute screen coordinate.
func (w *WinInjector) MoveAbs(x, y int) error {
	dx, dy := mapAbsolute(x, y)
	flags := uint32(win.MOUSEEVENTF_MOVE | win.MOUSEEVENTF_ABSOLUTE | win.MOUSEEVENTF_VIRTUALDESK)
	if err := sendMouseInput(flags, dx, dy, 0); err != nil {
		if win.SetCursorPos(int32(x), int32(y)) {
			return nil
		}
		return err
	}
	win.SetCursorPos(int32(x), int32(y))
	return nil
}

// LeftDown presses the left mouse button.
func (w *WinInjector) LeftDown() error {
	return sendMouseInput(win.MOUSEEVENTF_LEFTDOWN, 0, 0, 0)
}

// LeftUp releases the left mouse button.
func (w *WinInjector) LeftUp() error {
	return sendMouseInput(win.MOUSEEVENTF_LEFTUP, 0, 0, 0)
}

// ClickAt moves the cursor and performs a left click.
func (w *WinInjector) ClickAt(x, y int) error {
	if err := w.MoveAbs(x, y); err != nil {
		return err
	}
	if err := w.LeftDown(); err != nil {
		return err
	}
	return w.LeftUp()
}

// Wheel scrolls by the provided delta.
func (w *WinInjector) Wheel(delta int) error {
	return sendMouseInput(win.MOUSEEVENTF_WHEEL, 0, 0, uint32(delta))
}

// mapAbsolute converts screen coordinates to the WinAPI absolute range.
func mapAbsolute(x, y int) (int32, int32) {
	vx := win.GetSystemMetrics(win.SM_XVIRTUALSCREEN)
	vy := win.GetSystemMetrics(win.SM_YVIRTUALSCREEN)
	vw := win.GetSystemMetrics(win.SM_CXVIRTUALSCREEN)
	vh := win.GetSystemMetrics(win.SM_CYVIRTUALSCREEN)
	if vw <= 1 {
		vw = 2
	}
	if vh <= 1 {
		vh = 2
	}
	dx := (int64(x) - int64(vx)) * 65535 / int64(vw-1)
	dy := (int64(y) - int64(vy)) * 65535 / int64(vh-1)
	return int32(dx), int32(dy)
}
