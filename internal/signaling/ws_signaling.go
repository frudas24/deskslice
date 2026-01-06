// Package signaling defines signaling protocol messages for WebRTC.
package signaling

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	pub "github.com/frudas24/deskslice/internal/webrtc"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

// ViewerPolicy controls how additional viewers are handled.
type ViewerPolicy int

const (
	// ViewerReject rejects new connections when one is active.
	ViewerReject ViewerPolicy = iota
	// ViewerReplace closes the active connection when a new one arrives.
	ViewerReplace
)

// Server handles WebRTC signaling over WebSocket.
type Server struct {
	mu        sync.Mutex
	writeMu   sync.Mutex
	upgrader  websocket.Upgrader
	publisher *pub.Publisher
	policy    ViewerPolicy
	authFn    func() bool
	conn      *websocket.Conn
	peer      *webrtc.PeerConnection
}

// NewServer creates a signaling server with the chosen viewer policy and auth function.
func NewServer(publisher *pub.Publisher, policy ViewerPolicy, authFn func() bool) *Server {
	return &Server{
		publisher: publisher,
		policy:    policy,
		authFn:    authFn,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin:     func(*http.Request) bool { return true },
		},
	}
}

// ServeHTTP upgrades the request and starts the signaling loop.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.authFn != nil && !s.authFn() {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	if err := s.acceptConn(conn); err != nil {
		s.rejectConn(conn, err.Error())
		return
	}
	defer s.cleanupConn(conn)

	peer, err := s.publisher.NewPeer()
	if err != nil {
		return
	}
	if err := s.attachPeer(conn, peer); err != nil {
		_ = peer.Close()
		return
	}

	peer.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		candidate := c.ToJSON()
		_ = s.sendTo(conn, Message{T: "ice", Candidate: &candidate})
	})

	for {
		var msg Message
		if err := conn.ReadJSON(&msg); err != nil {
			return
		}
		if err := s.handleMessage(conn, peer, msg); err != nil {
			return
		}
	}
}

// NotifyRestart sends a restart message to the active viewer.
func (s *Server) NotifyRestart() {
	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()
	if conn == nil {
		return
	}
	_ = s.sendTo(conn, Message{T: "restart"})
}

// acceptConn registers a new websocket connection or returns an error.
func (s *Server) acceptConn(conn *websocket.Conn) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn != nil {
		switch s.policy {
		case ViewerReplace:
			_ = s.conn.Close()
			s.conn = nil
			s.peer = nil
		default:
			return fmt.Errorf("viewer already connected")
		}
	}
	s.conn = conn
	return nil
}

// rejectConn sends a policy violation close and closes the socket.
func (s *Server) rejectConn(conn *websocket.Conn, reason string) {
	message := websocket.FormatCloseMessage(websocket.ClosePolicyViolation, reason)
	_ = conn.WriteControl(websocket.CloseMessage, message, time.Now().Add(1*time.Second))
	_ = conn.Close()
}

// attachPeer stores the peer connection when the websocket is still active.
func (s *Server) attachPeer(conn *websocket.Conn, peer *webrtc.PeerConnection) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn != conn {
		return fmt.Errorf("connection no longer active")
	}
	s.peer = peer
	return nil
}

// cleanupConn clears state if the connection is still the active one.
func (s *Server) cleanupConn(conn *websocket.Conn) {
	s.mu.Lock()
	if s.conn == conn {
		s.conn = nil
		if s.peer != nil {
			_ = s.peer.Close()
			s.peer = nil
		}
	}
	s.mu.Unlock()
	_ = conn.Close()
}

// handleMessage dispatches signaling messages.
func (s *Server) handleMessage(conn *websocket.Conn, peer *webrtc.PeerConnection, msg Message) error {
	switch msg.T {
	case "offer":
		return s.handleOffer(conn, peer, msg.SDP)
	case "ice":
		return s.handleICE(peer, msg.Candidate)
	case "restart":
		return nil
	default:
		return nil
	}
}

// handleOffer processes an SDP offer and replies with an answer.
func (s *Server) handleOffer(conn *websocket.Conn, peer *webrtc.PeerConnection, sdp string) error {
	if sdp == "" {
		return fmt.Errorf("empty offer")
	}
	if err := peer.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  sdp,
	}); err != nil {
		return err
	}
	answer, err := peer.CreateAnswer(nil)
	if err != nil {
		return err
	}
	gatherComplete := webrtc.GatheringCompletePromise(peer)
	if err := peer.SetLocalDescription(answer); err != nil {
		return err
	}
	<-gatherComplete
	local := peer.LocalDescription()
	if local == nil {
		return fmt.Errorf("missing local description")
	}
	return s.sendTo(conn, Message{T: "answer", SDP: local.SDP})
}

// handleICE adds a remote ICE candidate.
func (s *Server) handleICE(peer *webrtc.PeerConnection, candidate *webrtc.ICECandidateInit) error {
	if candidate == nil {
		return nil
	}
	return peer.AddICECandidate(*candidate)
}

// sendTo writes a message to the active connection.
func (s *Server) sendTo(conn *websocket.Conn, msg Message) error {
	s.mu.Lock()
	active := s.conn
	s.mu.Unlock()
	if active != conn {
		return fmt.Errorf("connection not active")
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return conn.WriteJSON(msg)
}
