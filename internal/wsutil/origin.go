// Package wsutil provides shared helpers for websocket servers.
package wsutil

import (
	"net/http"
	"net/url"
	"strings"
)

// CheckOrigin allows same-origin browser connections and non-browser clients
// (which send no Origin header). Cross-origin requests are rejected to prevent
// cross-site WebSocket hijacking (CSWSH).
func CheckOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return strings.EqualFold(u.Host, r.Host)
}
