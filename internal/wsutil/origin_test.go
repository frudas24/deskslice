package wsutil

import (
	"net/http"
	"testing"
)

// TestCheckOrigin verifies same-origin, cross-origin, and non-browser cases.
func TestCheckOrigin(t *testing.T) {
	tests := []struct {
		name       string
		origin     string
		host       string
		wantAllow  bool
	}{
		{name: "no origin (non-browser)", origin: "", host: "192.168.1.5:8787", wantAllow: true},
		{name: "same origin", origin: "http://192.168.1.5:8787", host: "192.168.1.5:8787", wantAllow: true},
		{name: "same origin case-insensitive host", origin: "http://LOCALHOST:8787", host: "localhost:8787", wantAllow: true},
		{name: "cross origin", origin: "http://evil.example", host: "192.168.1.5:8787", wantAllow: false},
		{name: "cross origin same port different host", origin: "http://evil.example:8787", host: "192.168.1.5:8787", wantAllow: false},
		{name: "malformed origin", origin: "://bad", host: "192.168.1.5:8787", wantAllow: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &http.Request{Host: tt.host, Header: http.Header{}}
			if tt.origin != "" {
				r.Header.Set("Origin", tt.origin)
			}
			if got := CheckOrigin(r); got != tt.wantAllow {
				t.Fatalf("CheckOrigin(%q, host=%q) = %v, want %v", tt.origin, tt.host, got, tt.wantAllow)
			}
		})
	}
}
