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
}

// newRTPListener binds a UDP port for RTP ingestion.
func newRTPListener(port int) (*rtpListener, error) {
	addr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}
	return &rtpListener{conn: conn}, nil
}

// start begins forwarding RTP packets into the provided track.
func (l *rtpListener) start(track *webrtc.TrackLocalStaticRTP) error {
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
	go l.loop(track)
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
func (l *rtpListener) loop(track *webrtc.TrackLocalStaticRTP) {
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
		if err := track.WriteRTP(&pkt); err != nil && !l.writeErrLogged {
			log.Printf("rtp: write failed: %v", err)
			l.writeErrLogged = true
		}
	}
}
