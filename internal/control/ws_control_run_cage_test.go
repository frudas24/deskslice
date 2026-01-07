package control

import (
	"testing"

	"github.com/frudas24/deskslice/internal/calib"
	"github.com/frudas24/deskslice/internal/monitor"
	"github.com/frudas24/deskslice/internal/session"
	"github.com/frudas24/deskslice/internal/testutil"
)

// TestRunMode_ClickCaged verifies clicks in Run mode move the cursor into the plugin rect first.
func TestRunMode_ClickCaged(t *testing.T) {
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

	if err := server.handleClick(); err != nil {
		t.Fatalf("handleClick failed: %v", err)
	}
	if len(inj.Calls) != 3 {
		t.Fatalf("expected 3 calls, got %#v", inj.Calls)
	}
	if inj.Calls[0].Name != "MoveAbs" || inj.Calls[1].Name != "LeftDown" || inj.Calls[2].Name != "LeftUp" {
		t.Fatalf("unexpected call sequence %#v", inj.Calls)
	}
	if inj.Calls[0].X != 250 || inj.Calls[0].Y != 400 {
		t.Fatalf("expected cage to center (250,400), got (%d,%d)", inj.Calls[0].X, inj.Calls[0].Y)
	}
}
