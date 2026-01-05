package session

import "testing"

// TestAuthenticate_Success verifies successful authentication.
func TestAuthenticate_Success(t *testing.T) {
	s := New("secret")
	if !s.Authenticate("secret") {
		t.Fatalf("expected authentication to succeed")
	}
	if !s.IsAuthenticated() {
		t.Fatalf("expected authenticated state")
	}
}

// TestAuthenticate_Fail verifies failed authentication.
func TestAuthenticate_Fail(t *testing.T) {
	s := New("secret")
	if s.Authenticate("nope") {
		t.Fatalf("expected authentication to fail")
	}
	if s.IsAuthenticated() {
		t.Fatalf("expected unauthenticated state")
	}
}

// TestLogout verifies logout clears auth state.
func TestLogout(t *testing.T) {
	s := New("secret")
	s.Authenticate("secret")
	s.Logout()
	if s.IsAuthenticated() {
		t.Fatalf("expected unauthenticated state")
	}
}

// TestInputEnabled_Toggle verifies input enabled toggle.
func TestInputEnabled_Toggle(t *testing.T) {
	s := New("secret")
	s.SetInputEnabled(false)
	if s.InputEnabled() {
		t.Fatalf("expected input disabled")
	}
	s.SetInputEnabled(true)
	if !s.InputEnabled() {
		t.Fatalf("expected input enabled")
	}
}

// TestSnapshot verifies snapshot content.
func TestSnapshot(t *testing.T) {
	s := New("secret")
	s.Authenticate("secret")
	s.SetInputEnabled(false)
	s.SetMode(ModeRun)
	s.SetMonitor(2)
	snap := s.Snapshot()
	if !snap.Authenticated || snap.InputEnabled || snap.Mode != ModeRun || snap.MonitorIndex != 2 {
		t.Fatalf("unexpected snapshot: %+v", snap)
	}
}
