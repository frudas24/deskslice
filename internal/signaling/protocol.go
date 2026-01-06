// Package signaling defines signaling protocol messages for WebRTC.
package signaling

import "github.com/pion/webrtc/v3"

// Message is a websocket signaling payload.
type Message struct {
	T         string                   `json:"t"`
	SDP       string                   `json:"sdp,omitempty"`
	Candidate *webrtc.ICECandidateInit `json:"candidate,omitempty"`
}
