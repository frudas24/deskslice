// Package webrtc provides the WebRTC publisher pipeline.
package webrtc

import (
	"fmt"
	"sync"

	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v3"
)

// Publisher manages the WebRTC peer connection and video track.
type Publisher struct {
	mu    sync.Mutex
	api   *webrtc.API
	peer  *webrtc.PeerConnection
	track *webrtc.TrackLocalStaticRTP

	rtpListener *rtpListener
}

// NewPublisher initializes a WebRTC publisher with default codecs/interceptors.
func NewPublisher() (*Publisher, error) {
	media := &webrtc.MediaEngine{}
	if err := media.RegisterDefaultCodecs(); err != nil {
		return nil, fmt.Errorf("register codecs: %w", err)
	}

	interceptors := &interceptor.Registry{}
	if err := webrtc.RegisterDefaultInterceptors(media, interceptors); err != nil {
		return nil, fmt.Errorf("register interceptors: %w", err)
	}

	api := webrtc.NewAPI(
		webrtc.WithMediaEngine(media),
		webrtc.WithInterceptorRegistry(interceptors),
	)

	return &Publisher{api: api}, nil
}

// Track returns the H264 RTP track, creating it if needed.
func (p *Publisher) Track() (*webrtc.TrackLocalStaticRTP, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.ensureTrack()
}

// NewPeer creates a new peer connection and attaches the video track.
func (p *Publisher) NewPeer() (*webrtc.PeerConnection, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.peer != nil {
		_ = p.peer.Close()
		p.peer = nil
	}

	peer, err := p.api.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		return nil, err
	}

	track, err := p.ensureTrack()
	if err != nil {
		_ = peer.Close()
		return nil, err
	}

	sender, err := peer.AddTrack(track)
	if err != nil {
		_ = peer.Close()
		return nil, err
	}

	go func() {
		buf := make([]byte, 1500)
		for {
			if _, _, rtcpErr := sender.Read(buf); rtcpErr != nil {
				return
			}
		}
	}()

	p.peer = peer
	return peer, nil
}

// ClosePeer closes the current peer connection.
func (p *Publisher) ClosePeer() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.peer != nil {
		_ = p.peer.Close()
		p.peer = nil
	}
}

// AttachRTP binds a local UDP port for RTP ingest.
func (p *Publisher) AttachRTP(port int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.rtpListener != nil {
		p.rtpListener.close()
		p.rtpListener = nil
	}

	listener, err := newRTPListener(port)
	if err != nil {
		return err
	}
	p.rtpListener = listener
	return nil
}

// StartForwarding begins forwarding RTP packets into the WebRTC track.
func (p *Publisher) StartForwarding() error {
	p.mu.Lock()
	listener := p.rtpListener
	track := p.track
	p.mu.Unlock()

	if listener == nil || track == nil {
		return fmt.Errorf("rtp listener or track not ready")
	}
	return listener.start(track)
}

// StopForwarding stops RTP forwarding without closing the listener.
func (p *Publisher) StopForwarding() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.rtpListener != nil {
		p.rtpListener.stop()
	}
}

// ensureTrack initializes the track if it does not already exist.
func (p *Publisher) ensureTrack() (*webrtc.TrackLocalStaticRTP, error) {
	if p.track != nil {
		return p.track, nil
	}
	track, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264},
		"video",
		"deskslice",
	)
	if err != nil {
		return nil, err
	}
	p.track = track
	return track, nil
}
