package control

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/frudas24/deskslice/internal/monitor"
	"github.com/frudas24/deskslice/internal/session"
	"github.com/frudas24/deskslice/internal/testutil"
	"github.com/gorilla/websocket"
)

// newAuthedControlServer returns a control server with an authenticated session.
func newAuthedControlServer() *Server {
	sess := session.New("pw")
	sess.Authenticate("pw")
	inj := &testutil.FakeInjector{HasXY: true}
	monitors := []monitor.Monitor{{Index: 1, X: 0, Y: 0, W: 1920, H: 1080, Primary: true}}
	return NewServer(sess, inj, func() ([]monitor.Monitor, error) { return monitors, nil }, nil, nil)
}

// TestServeHTTPRejectsCrossOrigin verifies cross-origin upgrades are rejected.
func TestServeHTTPRejectsCrossOrigin(t *testing.T) {
	srv := newAuthedControlServer()
	ts := httptest.NewServer(srv)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	h := http.Header{}
	h.Set("Origin", "http://evil.example")
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, h)
	if err == nil {
		t.Fatal("expected cross-origin upgrade to fail")
	}
	if resp != nil && resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
}

// TestServeHTTPAcceptsSameOriginAndCloseActive verifies same-origin upgrades succeed and CloseActive closes the connection.
func TestServeHTTPAcceptsSameOriginAndCloseActive(t *testing.T) {
	srv := newAuthedControlServer()
	ts := httptest.NewServer(srv)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	h := http.Header{}
	h.Set("Origin", ts.URL)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, h)
	if err != nil {
		t.Fatalf("same-origin upgrade failed: %v", err)
	}
	defer conn.Close()

	srv.CloseActive()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatal("expected read to fail after CloseActive")
	}
}
