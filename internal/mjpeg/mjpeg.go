// Package mjpeg provides a minimal MJPEG stream for browser previews.
package mjpeg

import (
	"bytes"
	"image"
	"image/jpeg"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const boundary = "frame"

// Stream broadcasts JPEG frames to connected HTTP clients.
type Stream struct {
	mu          sync.RWMutex
	subs        map[chan []byte]struct{}
	last        []byte
	minInterval time.Duration
	lastPush    time.Time
}

// NewStream creates a new stream with a minimum publish interval.
func NewStream(minInterval time.Duration) *Stream {
	return &Stream{
		subs:        make(map[chan []byte]struct{}),
		minInterval: minInterval,
	}
}

// SetMinInterval sets the minimum interval between published frames.
func (s *Stream) SetMinInterval(d time.Duration) {
	s.mu.Lock()
	s.minInterval = d
	s.mu.Unlock()
}

// Publish sends a JPEG frame to all subscribers with throttling.
func (s *Stream) Publish(jpg []byte) {
	now := time.Now()
	s.mu.Lock()
	if s.minInterval > 0 && now.Sub(s.lastPush) < s.minInterval {
		s.last = append([]byte(nil), jpg...)
		s.mu.Unlock()
		return
	}
	frame := append([]byte(nil), jpg...)
	s.last = frame
	s.lastPush = now
	for ch := range s.subs {
		select {
		case <-ch:
		default:
		}
		select {
		case ch <- frame:
		default:
		}
	}
	s.mu.Unlock()
}

// Handler serves the MJPEG multipart stream to the HTTP client.
func (s *Stream) Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary="+boundary)
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Pragma", "no-cache")

	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	ch := s.subscribe()
	defer s.unsubscribe(ch)

	keep := time.NewTicker(1 * time.Second)
	defer keep.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case jpg := <-ch:
			if err := writePart(w, jpg); err != nil {
				return
			}
			fl.Flush()
		case <-keep.C:
			s.mu.RLock()
			j := append([]byte(nil), s.last...)
			s.mu.RUnlock()
			if len(j) > 0 {
				if err := writePart(w, j); err != nil {
					return
				}
				fl.Flush()
			}
		}
	}
}

// EncodeRGBToJPEG encodes RGB24 bytes into a JPEG buffer.
func EncodeRGBToJPEG(rgb []byte, w, h int, quality int) []byte {
	if quality <= 0 || quality > 100 {
		quality = 60
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	si := 0
	di := 0
	stride := img.Stride
	for y := 0; y < h; y++ {
		di = y * stride
		for x := 0; x < w; x++ {
			if si+2 >= len(rgb) {
				break
			}
			img.Pix[di+0] = rgb[si+0]
			img.Pix[di+1] = rgb[si+1]
			img.Pix[di+2] = rgb[si+2]
			img.Pix[di+3] = 255
			si += 3
			di += 4
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
	return buf.Bytes()
}

// subscribe registers a new client for MJPEG frames.
func (s *Stream) subscribe() chan []byte {
	ch := make(chan []byte, 1)
	s.mu.Lock()
	s.subs[ch] = struct{}{}
	if len(s.last) > 0 {
		ch <- append([]byte(nil), s.last...)
	}
	s.mu.Unlock()
	return ch
}

// unsubscribe removes a client subscription.
func (s *Stream) unsubscribe(ch chan []byte) {
	s.mu.Lock()
	delete(s.subs, ch)
	close(ch)
	s.mu.Unlock()
}

// writePart writes a single JPEG frame to the multipart response.
func writePart(w http.ResponseWriter, jpg []byte) error {
	_, _ = w.Write([]byte("\r\n--" + boundary + "\r\n"))
	_, _ = w.Write([]byte("Content-Type: image/jpeg\r\n"))
	_, _ = w.Write([]byte("Content-Length: " + strconv.Itoa(len(jpg)) + "\r\n\r\n"))
	_, err := w.Write(jpg)
	return err
}
