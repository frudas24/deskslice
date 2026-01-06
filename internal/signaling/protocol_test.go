package signaling

import (
	"encoding/json"
	"testing"
)

// TestProtocol_Offer verifies decoding an offer message.
func TestProtocol_Offer(t *testing.T) {
	var msg Message
	if err := json.Unmarshal([]byte(`{"t":"offer","sdp":"v=0"}`), &msg); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if msg.T != "offer" || msg.SDP != "v=0" {
		t.Fatalf("unexpected message: %+v", msg)
	}
}

// TestProtocol_ICE verifies decoding an ICE candidate message.
func TestProtocol_ICE(t *testing.T) {
	var msg Message
	payload := `{"t":"ice","candidate":{"candidate":"candidate:1 1 UDP 2122252543 192.0.2.3 54400 typ host"}}`
	if err := json.Unmarshal([]byte(payload), &msg); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if msg.T != "ice" || msg.Candidate == nil || msg.Candidate.Candidate == "" {
		t.Fatalf("unexpected message: %+v", msg)
	}
}
