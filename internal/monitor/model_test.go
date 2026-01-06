package monitor

import "testing"

// TestGetMonitorByIndex_Found verifies a monitor is found by index.
func TestGetMonitorByIndex_Found(t *testing.T) {
	list := []Monitor{
		{Index: 1, W: 100, H: 100},
		{Index: 2, W: 200, H: 200},
	}
	m, ok := GetMonitorByIndex(list, 2)
	if !ok || m.Index != 2 {
		t.Fatalf("expected index 2, got ok=%v monitor=%+v", ok, m)
	}
}

// TestGetMonitorByIndex_NotFound verifies missing indexes return false.
func TestGetMonitorByIndex_NotFound(t *testing.T) {
	list := []Monitor{{Index: 1, W: 100, H: 100}}
	_, ok := GetMonitorByIndex(list, 3)
	if ok {
		t.Fatalf("expected not found")
	}
}
