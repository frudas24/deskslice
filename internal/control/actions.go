// Package control handles input protocol and gesture mapping.
package control

// ActionType identifies the kind of input action to execute.
type ActionType string

const (
	// ActMove moves the mouse cursor.
	ActMove ActionType = "move"
	// ActLeftDown presses the left mouse button.
	ActLeftDown ActionType = "left_down"
	// ActLeftUp releases the left mouse button.
	ActLeftUp ActionType = "left_up"
	// ActClick performs a click at a position.
	ActClick ActionType = "click"
	// ActType types unicode text.
	ActType ActionType = "type"
	// ActEnter presses Enter.
	ActEnter ActionType = "enter"
)

// Action describes a normalized input operation to apply.
type Action struct {
	Type ActionType
	X    int
	Y    int
	Text string
}
