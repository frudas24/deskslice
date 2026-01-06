// Package session holds runtime state for the active viewer.
package session

import (
	"sync"

	"github.com/frudas24/deskslice/internal/calib"
)

// ModePresetup is the calibration mode.
const ModePresetup = "presetup"

// ModeRun is the cropped streaming mode.
const ModeRun = "run"

// VideoWebRTC runs the RTP pipeline for WebRTC video.
const VideoWebRTC = "webrtc"

// VideoMJPEG runs the MJPEG preview pipeline only.
const VideoMJPEG = "mjpeg"

// Snapshot represents a read-only view of the current session state.
type Snapshot struct {
	Authenticated bool
	InputEnabled  bool
	Mode          string
	MonitorIndex  int
	VideoMode     string
	Calib         calib.Calib
}

// Session holds runtime state for the active viewer.
type Session struct {
	mu            sync.RWMutex
	password      string
	authenticated bool
	inputEnabled  bool
	mode          string
	monitorIndex  int
	videoMode     string
	calib         calib.Calib
}

// New returns an initialized session with the given password.
func New(password string) *Session {
	return &Session{
		password:     password,
		inputEnabled: true,
		mode:         ModePresetup,
		videoMode:    VideoMJPEG,
	}
}

// Authenticate validates the password and marks the session as authenticated.
func (s *Session) Authenticate(pass string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if pass != "" && pass == s.password {
		s.authenticated = true
		return true
	}
	s.authenticated = false
	return false
}

// Logout clears authentication state.
func (s *Session) Logout() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.authenticated = false
}

// IsAuthenticated reports whether the session is authenticated.
func (s *Session) IsAuthenticated() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.authenticated
}

// SetInputEnabled toggles whether inputs are forwarded to the host.
func (s *Session) SetInputEnabled(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inputEnabled = enabled
}

// InputEnabled reports whether inputs are forwarded to the host.
func (s *Session) InputEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.inputEnabled
}

// SetMode sets the current session mode.
func (s *Session) SetMode(mode string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mode = mode
}

// Mode returns the current session mode.
func (s *Session) Mode() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mode
}

// SetMonitor sets the selected monitor index.
func (s *Session) SetMonitor(idx int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.monitorIndex = idx
}

// Monitor returns the selected monitor index.
func (s *Session) Monitor() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.monitorIndex
}

// SetVideoMode sets which video pipeline the server should run.
func (s *Session) SetVideoMode(mode string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch mode {
	case VideoMJPEG:
		s.videoMode = VideoMJPEG
	default:
		s.videoMode = VideoWebRTC
	}
}

// VideoMode returns the active video pipeline mode.
func (s *Session) VideoMode() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.videoMode == "" {
		return VideoMJPEG
	}
	return s.videoMode
}

// SetCalib stores calibration data.
func (s *Session) SetCalib(c calib.Calib) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calib = c
}

// GetCalib returns the current calibration data.
func (s *Session) GetCalib() calib.Calib {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.calib
}

// Snapshot returns a copy of the current session state.
func (s *Session) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return Snapshot{
		Authenticated: s.authenticated,
		InputEnabled:  s.inputEnabled,
		Mode:          s.mode,
		MonitorIndex:  s.monitorIndex,
		VideoMode:     s.videoMode,
		Calib:         s.calib,
	}
}
