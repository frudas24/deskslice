package control

import (
	"testing"

	"github.com/frudas24/deskslice/internal/calib"
	"github.com/frudas24/deskslice/internal/monitor"
	"github.com/frudas24/deskslice/internal/session"
	"github.com/frudas24/deskslice/internal/testutil"
)

// TestClampPointToRect_ClampsToEdges verifies clamping stays inside the rect bounds.
func TestClampPointToRect_ClampsToEdges(t *testing.T) {
	r := calib.Rect{X: 10, Y: 20, W: 30, H: 40}
	x, y := ClampPointToRect(r, -5, 999)
	if x != 10 || y != 59 {
		t.Fatalf("expected (10,59), got (%d,%d)", x, y)
	}
}

// TestRunMode_RelMoveCaged verifies relative moves are converted to clamped absolute moves in Run mode.
func TestRunMode_RelMoveCaged(t *testing.T) {
	sess := session.New("pw")
	sess.SetInputEnabled(true)
	sess.SetMode(session.ModeRun)
	sess.SetMonitor(1)
	sess.SetCalib(calib.Calib{
		MonitorIndex: 1,
		PluginAbs:    calib.Rect{X: 100, Y: 200, W: 300, H: 400},
	})

	inj := &testutil.FakeInjector{X: 1, Y: 2, HasXY: true}
	monitors := []monitor.Monitor{{Index: 1, X: 0, Y: 0, W: 1920, H: 1080, Primary: true}}
	server := NewServer(sess, inj, func() ([]monitor.Monitor, error) { return monitors, nil }, nil, nil)

	if err := server.handleRelMove(Message{DX: 5000, DY: 0}); err != nil {
		t.Fatalf("handleRelMove failed: %v", err)
	}
	if len(inj.Calls) != 2 || inj.Calls[0].Name != "MoveAbs" || inj.Calls[1].Name != "MoveAbs" {
		t.Fatalf("expected two MoveAbs calls, got %#v", inj.Calls)
	}
	// First call should cage into the plugin rect center.
	if inj.Calls[0].X != 250 || inj.Calls[0].Y != 400 {
		t.Fatalf("expected cage to center (250,400), got (%d,%d)", inj.Calls[0].X, inj.Calls[0].Y)
	}
	// Second call should clamp to the right edge.
	if inj.Calls[1].X != 399 || inj.Calls[1].Y != 400 {
		t.Fatalf("expected clamped target (399,400), got (%d,%d)", inj.Calls[1].X, inj.Calls[1].Y)
	}
}
