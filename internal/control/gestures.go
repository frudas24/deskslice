// Package control handles input protocol and gesture mapping.
package control

import (
	"time"

	"github.com/frudas24/deskslice/internal/calib"
)

const (
	minMoveInterval = 16 * time.Millisecond
	minMoveDelta    = 2
)

// GestureState tracks drag state for touch interactions.
type GestureState struct {
	dragActive  bool
	dragPointer int
	lastMoveAt  time.Time
	lastX       int
	lastY       int
	now         func() time.Time
}

// NewGestureState returns a ready-to-use gesture tracker.
func NewGestureState() *GestureState {
	return &GestureState{now: time.Now}
}

// SetNowFunc overrides the clock used for throttling.
func (g *GestureState) SetNowFunc(fn func() time.Time) {
	if fn != nil {
		g.now = fn
	}
}

// HandleDown processes a pointer down event.
func (g *GestureState) HandleDown(inputEnabled bool, pointerID int, absX, absY int, plugin calib.Rect, scrollRel calib.Rect) []Action {
	if !inputEnabled {
		return nil
	}

	plugin = calib.Normalize(plugin)
	scrollRel = calib.Normalize(scrollRel)
	relX := absX - plugin.X
	relY := absY - plugin.Y

	if calib.Contains(scrollRel, relX, relY) {
		g.dragActive = true
		g.dragPointer = pointerID
		g.lastMoveAt = g.now()
		g.lastX = absX
		g.lastY = absY
		return []Action{{Type: ActLeftDown, X: absX, Y: absY}}
	}

	g.dragActive = false
	return []Action{{Type: ActClick, X: absX, Y: absY}}
}

// HandleMove processes a pointer move event.
func (g *GestureState) HandleMove(inputEnabled bool, pointerID int, absX, absY int) []Action {
	if !inputEnabled {
		return nil
	}
	if !g.dragActive || g.dragPointer != pointerID {
		return nil
	}

	now := g.now()
	if !g.lastMoveAt.IsZero() && now.Sub(g.lastMoveAt) < minMoveInterval {
		return nil
	}
	dx := absX - g.lastX
	dy := absY - g.lastY
	if abs(dx) < minMoveDelta && abs(dy) < minMoveDelta {
		return nil
	}

	g.lastMoveAt = now
	g.lastX = absX
	g.lastY = absY
	return []Action{{Type: ActMove, X: absX, Y: absY}}
}

// HandleUp processes a pointer up event.
func (g *GestureState) HandleUp(inputEnabled bool, pointerID int, absX, absY int) []Action {
	if !inputEnabled {
		return nil
	}
	if !g.dragActive || g.dragPointer != pointerID {
		return nil
	}

	g.dragActive = false
	return []Action{{Type: ActLeftUp, X: absX, Y: absY}}
}

// ActionsForType generates a click+type sequence targeting the chat input.
func ActionsForType(inputEnabled bool, text string, chatAbs calib.Rect) []Action {
	if !inputEnabled || text == "" {
		return nil
	}
	x, y := centerPoint(chatAbs)
	return []Action{
		{Type: ActClick, X: x, Y: y},
		{Type: ActType, Text: text},
	}
}

// ActionsForEnter generates a click+enter sequence targeting the chat input.
func ActionsForEnter(inputEnabled bool, chatAbs calib.Rect) []Action {
	if !inputEnabled {
		return nil
	}
	x, y := centerPoint(chatAbs)
	return []Action{
		{Type: ActClick, X: x, Y: y},
		{Type: ActEnter},
	}
}

// centerPoint returns the center of a rectangle.
func centerPoint(r calib.Rect) (int, int) {
	r = calib.Normalize(r)
	return r.X + r.W/2, r.Y + r.H/2
}

// abs returns the absolute value of an integer.
func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
