package control

import (
	"encoding/json"
	"testing"
)

// TestProtocol_Down verifies decoding a down message.
func TestProtocol_Down(t *testing.T) {
	var msg Message
	if err := json.Unmarshal([]byte(`{"t":"down","id":1,"x":0.5,"y":0.2}`), &msg); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if msg.T != "down" || msg.ID != 1 || msg.X != 0.5 || msg.Y != 0.2 {
		t.Fatalf("unexpected message: %+v", msg)
	}
}

// TestProtocol_Move verifies decoding a move message.
func TestProtocol_Move(t *testing.T) {
	var msg Message
	if err := json.Unmarshal([]byte(`{"t":"move","id":2,"x":0.1,"y":0.25}`), &msg); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if msg.T != "move" || msg.ID != 2 || msg.X != 0.1 || msg.Y != 0.25 {
		t.Fatalf("unexpected message: %+v", msg)
	}
}

// TestProtocol_Up verifies decoding an up message.
func TestProtocol_Up(t *testing.T) {
	var msg Message
	if err := json.Unmarshal([]byte(`{"t":"up","id":3,"x":0.9,"y":0.8}`), &msg); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if msg.T != "up" || msg.ID != 3 || msg.X != 0.9 || msg.Y != 0.8 {
		t.Fatalf("unexpected message: %+v", msg)
	}
}

// TestProtocol_Type verifies decoding a type message.
func TestProtocol_Type(t *testing.T) {
	var msg Message
	if err := json.Unmarshal([]byte(`{"t":"type","text":"hola"}`), &msg); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if msg.T != "type" || msg.Text != "hola" {
		t.Fatalf("unexpected message: %+v", msg)
	}
}

// TestProtocol_Enter verifies decoding an enter message.
func TestProtocol_Enter(t *testing.T) {
	var msg Message
	if err := json.Unmarshal([]byte(`{"t":"enter"}`), &msg); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if msg.T != "enter" {
		t.Fatalf("unexpected message: %+v", msg)
	}
}

// TestProtocol_Wheel verifies decoding a wheel message.
func TestProtocol_Wheel(t *testing.T) {
	var msg Message
	if err := json.Unmarshal([]byte(`{"t":"wheel","x":0.5,"y":0.25,"wheelX":-120,"wheelY":240}`), &msg); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if msg.T != "wheel" || msg.X != 0.5 || msg.Y != 0.25 || msg.WheelX != -120 || msg.WheelY != 240 {
		t.Fatalf("unexpected message: %+v", msg)
	}
}

// TestProtocol_RelMove verifies decoding a relative mouse move message.
func TestProtocol_RelMove(t *testing.T) {
	var msg Message
	if err := json.Unmarshal([]byte(`{"t":"relMove","dx":12,"dy":-7}`), &msg); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if msg.T != "relMove" || msg.DX != 12 || msg.DY != -7 {
		t.Fatalf("unexpected message: %+v", msg)
	}
}

// TestProtocol_Click verifies decoding a click message.
func TestProtocol_Click(t *testing.T) {
	var msg Message
	if err := json.Unmarshal([]byte(`{"t":"click"}`), &msg); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if msg.T != "click" {
		t.Fatalf("unexpected message: %+v", msg)
	}
}
