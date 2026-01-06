//go:build windows

// Package wininput defines Windows input injection interfaces.
package wininput

import (
	"unicode/utf16"

	"github.com/lxn/win"
)

// TypeUnicode types Unicode text into the focused window.
func (w *WinInjector) TypeUnicode(text string) error {
	if text == "" {
		return nil
	}
	for _, code := range utf16.Encode([]rune(text)) {
		if err := sendKeyboardInput(win.KEYBDINPUT{WScan: code, DwFlags: win.KEYEVENTF_UNICODE}); err != nil {
			return err
		}
		if err := sendKeyboardInput(win.KEYBDINPUT{WScan: code, DwFlags: win.KEYEVENTF_UNICODE | win.KEYEVENTF_KEYUP}); err != nil {
			return err
		}
	}
	return nil
}

// Enter sends an Enter key press.
func (w *WinInjector) Enter() error {
	if err := sendKeyboardInput(win.KEYBDINPUT{WVk: win.VK_RETURN}); err != nil {
		return err
	}
	return sendKeyboardInput(win.KEYBDINPUT{WVk: win.VK_RETURN, DwFlags: win.KEYEVENTF_KEYUP})
}
