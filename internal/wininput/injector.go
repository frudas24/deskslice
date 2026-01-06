// Package wininput defines Windows input injection interfaces.
package wininput

// Injector defines the input operations used by the control layer.
type Injector interface {
	MoveAbs(x, y int) error
	LeftDown() error
	LeftUp() error
	ClickAt(x, y int) error
	TypeUnicode(text string) error
	Enter() error
	SelectAll() error
	Delete() error
	Wheel(delta int) error
	HWheel(delta int) error
}
