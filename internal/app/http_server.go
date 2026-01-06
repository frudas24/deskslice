// Package app wires HTTP, signaling, and pipeline state together.
package app

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/frudas24/deskslice/internal/calib"
	"github.com/frudas24/deskslice/internal/web"
)

// RegisterRoutes wires API and static handlers onto the mux.
func (a *App) RegisterRoutes(mux *http.ServeMux, staticDir string) {
	if staticDir == "" {
		staticDir = filepath.Join("internal", "web", "static")
	}

	mux.HandleFunc("/login", a.handleLogin)
	mux.HandleFunc("/logout", a.handleLogout)
	mux.HandleFunc("/api/monitors", a.handleMonitors)
	mux.HandleFunc("/api/state", a.handleState)
	mux.Handle("/ws/signal", a.Signaling())
	mux.Handle("/ws/control", a.Control())
	mux.HandleFunc("/favicon.ico", handleFavicon)
	if stream := a.PreviewStream(); stream != nil {
		mux.HandleFunc("/mjpeg/desktop", stream.Handler)
	}

	mux.Handle("/", staticFileServer(staticDir))
}

type loginRequest struct {
	Password string `json:"password"`
}

type stateResponse struct {
	Mode          string       `json:"mode"`
	MonitorIndex  int          `json:"monitor"`
	InputEnabled  bool         `json:"inputEnabled"`
	VideoMode     string       `json:"videoMode"`
	Calib         calibStatus  `json:"calib"`
	CalibData     *calib.Calib `json:"calibData,omitempty"`
	Authenticated bool         `json:"authenticated"`
}

type calibStatus struct {
	Plugin bool `json:"plugin"`
	Chat   bool `json:"chat"`
	Scroll bool `json:"scroll"`
}

// handleLogin authenticates the session.
func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !a.session.Authenticate(req.Password) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

// handleLogout clears authentication state.
func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	a.session.Logout()
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

// handleMonitors returns the list of monitors.
func (a *App) handleMonitors(w http.ResponseWriter, _ *http.Request) {
	if !a.requireAuth(w) {
		return
	}
	list, err := a.ListMonitors()
	if err != nil {
		http.Error(w, "failed to list monitors", http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(list)
}

// handleState returns current session state and calibration status.
func (a *App) handleState(w http.ResponseWriter, _ *http.Request) {
	if !a.requireAuth(w) {
		return
	}
	snap := a.session.Snapshot()
	resp := stateResponse{
		Mode:          snap.Mode,
		MonitorIndex:  snap.MonitorIndex,
		InputEnabled:  snap.InputEnabled,
		VideoMode:     snap.VideoMode,
		Calib:         buildCalibStatus(snap.Calib),
		Authenticated: snap.Authenticated,
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// requireAuth returns false and writes an error if the session is not authenticated.
func (a *App) requireAuth(w http.ResponseWriter) bool {
	if !a.session.IsAuthenticated() {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	}
	return true
}

// buildCalibStatus summarizes whether calibration rectangles are present.
func buildCalibStatus(c calib.Calib) calibStatus {
	plugin := calib.Normalize(c.PluginAbs)
	chat := calib.Normalize(c.ChatRel)
	scroll := calib.Normalize(c.ScrollRel)
	return calibStatus{
		Plugin: plugin.W > 0 && plugin.H > 0,
		Chat:   chat.W > 0 && chat.H > 0,
		Scroll: scroll.W > 0 && scroll.H > 0,
	}
}

// staticFileServer returns a handler for static assets, preferring disk then embed.
func staticFileServer(staticDir string) http.Handler {
	if staticDir != "" {
		if info, err := os.Stat(staticDir); err == nil && info.IsDir() {
			return http.FileServer(http.Dir(staticDir))
		}
	}

	embedded, err := web.StaticFS()
	if err != nil {
		log.Printf("static assets unavailable: %v", err)
		return http.NotFoundHandler()
	}
	return http.FileServer(http.FS(embedded))
}

// handleFavicon avoids noisy 404s for the default browser request.
func handleFavicon(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}
