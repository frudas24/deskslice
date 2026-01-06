//go:build windows

// Package wininput defines Windows input injection interfaces.
package wininput

import "github.com/lxn/win"

// WinInjector injects mouse and keyboard input using WinAPI.
type WinInjector struct{}

// NewInjector returns a Windows input injector.
func NewInjector() (Injector, error) {
	return &WinInjector{}, nil
}

// sendMouseInput dispatches a single mouse input event.
func sendMouseInput(flags uint32, dx, dy int32, data uint32) error {
	input := win.INPUT{
		Type: win.INPUT_MOUSE,
		Mi: win.MOUSEINPUT{
			Dx:        dx,
			Dy:        dy,
			MouseData: data,
			DwFlags:   flags,
		},
	}
	if win.SendInput(1, &input, int32(unsafeSizeofInput())) != 1 {
		return win.GetLastError()
	}
	return nil
}

// sendKeyboardInput dispatches a single keyboard input event.
func sendKeyboardInput(key win.KEYBDINPUT) error {
	input := win.INPUT{
		Type: win.INPUT_KEYBOARD,
		Ki:   key,
	}
	if win.SendInput(1, &input, int32(unsafeSizeofInput())) != 1 {
		return win.GetLastError()
	}
	return nil
}

// unsafeSizeofInput returns the input struct size for SendInput.
func unsafeSizeofInput() uintptr {
	return unsafeSizeofInputValue
}

var unsafeSizeofInputValue = uintptr(win.SizeofINPUT)
