package control

import (
	"testing"
	"time"

	"github.com/frudas24/deskslice/internal/calib"
)

// TestDragScroll_StartsOnlyInsideScrollRel verifies drag begins inside scroll area.
func TestDragScroll_StartsOnlyInsideScrollRel(t *testing.T) {
	g := NewGestureState()
	plugin := calib.Rect{X: 100, Y: 100, W: 200, H: 200}
	scroll := calib.Rect{X: 10, Y: 10, W: 50, H: 50}

	actions := g.HandleDown(true, 1, 120, 120, plugin, scroll)
	if len(actions) != 1 || actions[0].Type != ActLeftDown {
		t.Fatalf("expected left_down, got %#v", actions)
	}

	actions = g.HandleDown(true, 2, 180, 180, plugin, scroll)
	if len(actions) != 1 || actions[0].Type != ActClick {
		t.Fatalf("expected click, got %#v", actions)
	}
}

// TestDragScroll_MoveOnlyWhenActiveAndSamePointer verifies drag move logic.
func TestDragScroll_MoveOnlyWhenActiveAndSamePointer(t *testing.T) {
	g := NewGestureState()
	now := time.Unix(0, 0)
	g.SetNowFunc(func() time.Time { return now })

	plugin := calib.Rect{X: 100, Y: 100, W: 200, H: 200}
	scroll := calib.Rect{X: 10, Y: 10, W: 50, H: 50}

	g.HandleDown(true, 1, 120, 120, plugin, scroll)

	now = now.Add(20 * time.Millisecond)
	actions := g.HandleMove(true, 2, 130, 130)
	if len(actions) != 0 {
		t.Fatalf("expected no actions for different pointer, got %#v", actions)
	}

	actions = g.HandleMove(true, 1, 130, 130)
	if len(actions) != 1 || actions[0].Type != ActMove {
		t.Fatalf("expected move, got %#v", actions)
	}
}

// TestDragScroll_UpStopsAndEmitsLeftUp verifies drag termination.
func TestDragScroll_UpStopsAndEmitsLeftUp(t *testing.T) {
	g := NewGestureState()
	plugin := calib.Rect{X: 100, Y: 100, W: 200, H: 200}
	scroll := calib.Rect{X: 10, Y: 10, W: 50, H: 50}

	g.HandleDown(true, 1, 120, 120, plugin, scroll)
	actions := g.HandleUp(true, 1, 120, 120)
	if len(actions) != 1 || actions[0].Type != ActLeftUp {
		t.Fatalf("expected left_up, got %#v", actions)
	}

	actions = g.HandleMove(true, 1, 130, 130)
	if len(actions) != 0 {
		t.Fatalf("expected no actions after drag end, got %#v", actions)
	}
}

// TestTapOutsideScroll_EmitsClick verifies taps outside scroll area click.
func TestTapOutsideScroll_EmitsClick(t *testing.T) {
	g := NewGestureState()
	plugin := calib.Rect{X: 100, Y: 100, W: 200, H: 200}
	scroll := calib.Rect{X: 10, Y: 10, W: 50, H: 50}

	actions := g.HandleDown(true, 1, 180, 180, plugin, scroll)
	if len(actions) != 1 || actions[0].Type != ActClick {
		t.Fatalf("expected click, got %#v", actions)
	}
}

// TestInputDisabled_NoActions verifies the kill switch blocks actions.
func TestInputDisabled_NoActions(t *testing.T) {
	g := NewGestureState()
	plugin := calib.Rect{X: 100, Y: 100, W: 200, H: 200}
	scroll := calib.Rect{X: 10, Y: 10, W: 50, H: 50}

	if actions := g.HandleDown(false, 1, 120, 120, plugin, scroll); len(actions) != 0 {
		t.Fatalf("expected no actions, got %#v", actions)
	}
	if actions := g.HandleMove(false, 1, 130, 130); len(actions) != 0 {
		t.Fatalf("expected no actions, got %#v", actions)
	}
	if actions := g.HandleUp(false, 1, 130, 130); len(actions) != 0 {
		t.Fatalf("expected no actions, got %#v", actions)
	}
}

// TestTypeEnter_EmitsClickTypeEnterSequence verifies type/enter behavior.
func TestTypeEnter_EmitsClickTypeEnterSequence(t *testing.T) {
	chat := calib.Rect{X: 10, Y: 20, W: 100, H: 40}

	actions := ActionsForType(true, "hola", chat)
	if len(actions) != 2 || actions[0].Type != ActClick || actions[1].Type != ActType {
		t.Fatalf("expected click+type, got %#v", actions)
	}

	actions = ActionsForEnter(true, chat)
	if len(actions) != 2 || actions[0].Type != ActClick || actions[1].Type != ActEnter {
		t.Fatalf("expected click+enter, got %#v", actions)
	}
}
