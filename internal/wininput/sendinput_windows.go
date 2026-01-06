//go:build windows

// Package wininput defines Windows input injection interfaces.
package wininput

import (
	"syscall"
	"unsafe"

	"github.com/lxn/win"
)

// WinInjector injects mouse and keyboard input using WinAPI.
type WinInjector struct{}

// NewInjector returns a Windows input injector.
func NewInjector() (Injector, error) {
	return &WinInjector{}, nil
}

type input struct {
	Type uint32
	_    uint32
	Mi   win.MOUSEINPUT
}

// sendMouseInput dispatches a single mouse input event.
func sendMouseInput(flags uint32, dx, dy int32, data uint32) error {
	inp := input{
		Type: win.INPUT_MOUSE,
		Mi: win.MOUSEINPUT{
			Dx:        dx,
			Dy:        dy,
			MouseData: data,
			DwFlags:   flags,
		},
	}
	if win.SendInput(1, unsafe.Pointer(&inp), int32(unsafe.Sizeof(inp))) != 1 {
		return syscall.Errno(win.GetLastError())
	}
	return nil
}

// sendKeyboardInput dispatches a single keyboard input event.
func sendKeyboardInput(key win.KEYBDINPUT) error {
	inp := input{Type: win.INPUT_KEYBOARD}
	*(*win.KEYBDINPUT)(unsafe.Pointer(&inp.Mi)) = key
	if win.SendInput(1, unsafe.Pointer(&inp), int32(unsafe.Sizeof(inp))) != 1 {
		return syscall.Errno(win.GetLastError())
	}
	return nil
}
