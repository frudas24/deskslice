// Package webrtc provides the WebRTC publisher pipeline.
package webrtc

import "sync/atomic"

// debugRTP controls whether verbose RTP packet logs are emitted.
var debugRTP atomic.Bool

// SetDebugLogging enables/disables verbose WebRTC/RTP debug logs.
func SetDebugLogging(enabled bool) {
	debugRTP.Store(enabled)
}

// debugRTPEnabled reports whether RTP debug logs are enabled.
func debugRTPEnabled() bool {
	return debugRTP.Load()
}
