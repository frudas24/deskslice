// Package webrtc provides the WebRTC publisher pipeline.
package webrtc

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
)

type rtpListener struct {
	mu      sync.Mutex
	conn    *net.UDPConn
	ctx     context.Context
	cancel  context.CancelFunc
	running bool

	packetCount    int
	firstLogged    bool
	writeErrLogged bool

	rewrite rtpRewriter
}

// newRTPListener binds a UDP port for RTP ingestion.
func newRTPListener(port int) (*rtpListener, error) {
	addr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}
	// Reduce risk of local packet drops under load (desktop capture + H264 can be bursty).
	_ = conn.SetReadBuffer(4 << 20)
	return &rtpListener{conn: conn}, nil
}

// port returns the local UDP port for the listener.
func (l *rtpListener) port() int {
	if l == nil || l.conn == nil {
		return 0
	}
	addr, ok := l.conn.LocalAddr().(*net.UDPAddr)
	if !ok || addr == nil {
		return 0
	}
	return addr.Port
}

// start begins forwarding RTP packets into the provided track.
func (l *rtpListener) start(track *webrtc.TrackLocalStaticRTP, params func() rtpWriteParams) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.conn == nil {
		return fmt.Errorf("rtp listener not initialized")
	}
	if l.running {
		return nil
	}
	l.ctx, l.cancel = context.WithCancel(context.Background())
	l.running = true
	l.packetCount = 0
	l.firstLogged = false
	l.writeErrLogged = false
	go l.loop(track, params)
	return nil
}

// stop cancels the forward loop.
func (l *rtpListener) stop() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.cancel != nil {
		l.cancel()
	}
	l.running = false
}

// close stops forwarding and closes the UDP socket.
func (l *rtpListener) close() {
	l.stop()
	if l.conn != nil {
		_ = l.conn.Close()
		l.conn = nil
	}
}

// loop reads RTP packets and forwards them to the track.
func (l *rtpListener) loop(track *webrtc.TrackLocalStaticRTP, params func() rtpWriteParams) {
	buf := make([]byte, 1600)
	lastLog := time.Now()
	for {
		select {
		case <-l.ctx.Done():
			return
		default:
		}

		n, _, err := l.conn.ReadFromUDP(buf)
		if err != nil {
			return
		}
		var pkt rtp.Packet
		if err := pkt.Unmarshal(buf[:n]); err != nil {
			continue
		}
		l.packetCount++
		if !l.firstLogged {
			log.Printf("rtp: first packet ssrc=%d pt=%d seq=%d ts=%d", pkt.SSRC, pkt.PayloadType, pkt.SequenceNumber, pkt.Timestamp)
			l.firstLogged = true
		}
		if time.Since(lastLog) > 5*time.Second {
			log.Printf("rtp: packets=%d", l.packetCount)
			lastLog = time.Now()
		}

		writeParams := rtpWriteParams{}
		if params != nil {
			writeParams = params()
		}
		l.rewrite.Apply(&pkt, writeParams)

		if err := track.WriteRTP(&pkt); err != nil && !l.writeErrLogged {
			log.Printf("rtp: write failed: %v", err)
			l.writeErrLogged = true
		}
	}
}

// rtpWriteParams describes the RTP header fields expected by the active WebRTC sender.
type rtpWriteParams struct {
	payloadType uint8
}

// rtpRewriter rewrites incoming RTP packets to be stable across source restarts.
type rtpRewriter struct {
	initialized bool
	outSeq      uint16
	outTS       uint32
	lastInTS    uint32
	lastDelta   uint32
}

// Apply rewrites sequence/timestamp and overrides payload type/ssrc when available.
func (r *rtpRewriter) Apply(pkt *rtp.Packet, params rtpWriteParams) {
	if pkt == nil {
		return
	}

	if params.payloadType != 0 {
		pkt.PayloadType = params.payloadType
	}

	// Always resequence; this keeps the receiver happy when ffmpeg restarts/reset seq numbers.
	if !r.initialized {
		r.initialized = true
		r.outSeq = pkt.SequenceNumber
		r.lastInTS = pkt.Timestamp
		r.outTS = 0
		r.lastDelta = 0
	} else {
		r.outSeq++
	}
	pkt.SequenceNumber = r.outSeq

	// Retimestamp by translating input timestamps into a continuous timeline.
	// Keep all packets for a single input timestamp on the same output timestamp (frame boundary).
	if pkt.Timestamp == r.lastInTS {
		pkt.Timestamp = r.outTS
		return
	}

	delta := pkt.Timestamp - r.lastInTS
	// If the input jumps backwards/forwards a lot (typical after ffmpeg restart), don't forward a huge delta.
	// Instead reuse the last "reasonable" delta or a conservative default (~33ms @ 90kHz).
	if delta > 90000 {
		if r.lastDelta > 0 && r.lastDelta <= 90000 {
			delta = r.lastDelta
		} else {
			delta = 3000
		}
	} else {
		r.lastDelta = delta
	}

	r.lastInTS = pkt.Timestamp
	r.outTS += delta
	pkt.Timestamp = r.outTS
}
