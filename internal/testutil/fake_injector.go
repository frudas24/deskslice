package testutil

import "github.com/frudas24/deskslice/internal/wininput"

// Call records a single injected action.
type Call struct {
	Name string
	X    int
	Y    int
	Text string
}

// FakeInjector implements wininput.Injector and records calls for tests.
type FakeInjector struct {
	Calls []Call
}

// Ensure FakeInjector implements the interface.
var _ wininput.Injector = (*FakeInjector)(nil)

// MoveAbs records an absolute move.
func (f *FakeInjector) MoveAbs(x, y int) error {
	f.Calls = append(f.Calls, Call{Name: "MoveAbs", X: x, Y: y})
	return nil
}

// LeftDown records a left mouse down.
func (f *FakeInjector) LeftDown() error {
	f.Calls = append(f.Calls, Call{Name: "LeftDown"})
	return nil
}

// LeftUp records a left mouse up.
func (f *FakeInjector) LeftUp() error {
	f.Calls = append(f.Calls, Call{Name: "LeftUp"})
	return nil
}

// ClickAt records a click at a position.
func (f *FakeInjector) ClickAt(x, y int) error {
	f.Calls = append(f.Calls, Call{Name: "ClickAt", X: x, Y: y})
	return nil
}

// TypeUnicode records typed text.
func (f *FakeInjector) TypeUnicode(text string) error {
	f.Calls = append(f.Calls, Call{Name: "TypeUnicode", Text: text})
	return nil
}

// Enter records an Enter key press.
func (f *FakeInjector) Enter() error {
	f.Calls = append(f.Calls, Call{Name: "Enter"})
	return nil
}

// Wheel records a mouse wheel delta.
func (f *FakeInjector) Wheel(delta int) error {
	f.Calls = append(f.Calls, Call{Name: "Wheel", Y: delta})
	return nil
}
