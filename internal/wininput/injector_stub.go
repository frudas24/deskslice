//go:build !windows

// Package wininput defines Windows input injection interfaces.
package wininput

import "errors"

// ErrUnsupported indicates WinAPI input injection is not available.
var ErrUnsupported = errors.New("wininput is only supported on Windows")

// NoopInjector is a placeholder injector for non-Windows builds.
type NoopInjector struct{}

// NewInjector returns a non-functional injector on non-Windows platforms.
func NewInjector() (Injector, error) {
	return &NoopInjector{}, ErrUnsupported
}

// MoveAbs returns ErrUnsupported.
func (n *NoopInjector) MoveAbs(x, y int) error {
	_ = x
	_ = y
	return ErrUnsupported
}

// LeftDown returns ErrUnsupported.
func (n *NoopInjector) LeftDown() error {
	return ErrUnsupported
}

// LeftUp returns ErrUnsupported.
func (n *NoopInjector) LeftUp() error {
	return ErrUnsupported
}

// ClickAt returns ErrUnsupported.
func (n *NoopInjector) ClickAt(x, y int) error {
	_ = x
	_ = y
	return ErrUnsupported
}

// TypeUnicode returns ErrUnsupported.
func (n *NoopInjector) TypeUnicode(text string) error {
	_ = text
	return ErrUnsupported
}

// Enter returns ErrUnsupported.
func (n *NoopInjector) Enter() error {
	return ErrUnsupported
}

// Wheel returns ErrUnsupported.
func (n *NoopInjector) Wheel(delta int) error {
	_ = delta
	return ErrUnsupported
}
