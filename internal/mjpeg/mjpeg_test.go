package mjpeg

import (
	"bytes"
	"context"
	"net/http"
	"sync"
	"testing"
	"time"
)

// threadSafeRecorder is a minimal http.ResponseWriter + http.Flusher that is safe to use across goroutines.
type threadSafeRecorder struct {
	mu     sync.Mutex
	header http.Header
	buf    bytes.Buffer
	status int
}

// Header returns the response headers.
func (r *threadSafeRecorder) Header() http.Header {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.header == nil {
		r.header = make(http.Header)
	}
	return r.header
}

// Write appends bytes to the response body.
func (r *threadSafeRecorder) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.buf.Write(p)
}

// WriteHeader sets the HTTP status code.
func (r *threadSafeRecorder) WriteHeader(statusCode int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.status = statusCode
}

// Flush implements http.Flusher.
func (r *threadSafeRecorder) Flush() {}

// bodyString returns the current body as a string.
func (r *threadSafeRecorder) bodyString() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.buf.String()
}

// bodyBytes returns a copy of the current body as bytes.
func (r *threadSafeRecorder) bodyBytes() []byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]byte(nil), r.buf.Bytes()...)
}

// TestEncodeRGBToJPEG validates the encoder produces non-empty JPEG output for a tiny RGB frame.
func TestEncodeRGBToJPEG(t *testing.T) {
	t.Parallel()
	jpg := EncodeRGBToJPEG([]byte{255, 0, 0}, 1, 1, 60)
	if len(jpg) == 0 {
		t.Fatal("expected non-empty jpeg output")
	}
}

// TestStreamHandlerWritesFrame validates the handler writes a multipart MJPEG frame when a last frame is available.
func TestStreamHandlerWritesFrame(t *testing.T) {
	t.Parallel()

	s := NewStream(0)
	jpg := EncodeRGBToJPEG([]byte{0, 255, 0}, 1, 1, 60)
	s.Publish(jpg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example/mjpeg", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	rec := &threadSafeRecorder{}

	done := make(chan struct{})
	go func() {
		s.Handler(rec, req)
		close(done)
	}()

	deadline := time.NewTimer(500 * time.Millisecond)
	defer deadline.Stop()
	for {
		if bytes.Contains(rec.bodyBytes(), []byte("--"+boundary)) {
			break
		}
		select {
		case <-deadline.C:
			cancel()
			<-done
			t.Fatalf("timed out waiting for mjpeg boundary, body=%q", rec.bodyString())
		case <-time.After(5 * time.Millisecond):
		}
	}

	cancel()
	<-done

	ct := rec.Header().Get("Content-Type")
	if ct != "multipart/x-mixed-replace; boundary="+boundary {
		t.Fatalf("unexpected content-type: %q", ct)
	}

	body := rec.bodyBytes()
	if !bytes.Contains(body, []byte("Content-Type: image/jpeg")) {
		t.Fatalf("expected jpeg part header, body=%q", rec.bodyString())
	}
	if !bytes.Contains(body, []byte("Content-Length:")) {
		t.Fatalf("expected content length header, body=%q", rec.bodyString())
	}
	if !bytes.Contains(body, jpg) {
		t.Fatalf("expected jpeg payload in body")
	}
}

// TestStreamPublishThrottle ensures a throttled publish updates the last frame but does not broadcast immediately.
func TestStreamPublishThrottle(t *testing.T) {
	t.Parallel()

	s := NewStream(time.Hour)
	ch := s.subscribe()
	defer s.unsubscribe(ch)

	jpgA := EncodeRGBToJPEG([]byte{0, 0, 255}, 1, 1, 60)
	jpgB := EncodeRGBToJPEG([]byte{255, 255, 0}, 1, 1, 60)

	s.Publish(jpgA)
	select {
	case got := <-ch:
		if !bytes.Equal(got, jpgA) {
			t.Fatalf("expected first publish to broadcast jpgA")
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out waiting for first publish")
	}

	s.Publish(jpgB)
	select {
	case <-ch:
		t.Fatal("expected throttled publish to not broadcast immediately")
	case <-time.After(50 * time.Millisecond):
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if !bytes.Equal(s.last, jpgB) {
		t.Fatal("expected last frame to update even when throttled")
	}
}

// TestStreamPublishConcurrent does a basic concurrent publish/subscribe churn to help catch panics and race issues under -race.
func TestStreamPublishConcurrent(t *testing.T) {
	t.Parallel()

	s := NewStream(0)
	jpg := EncodeRGBToJPEG([]byte{10, 20, 30}, 1, 1, 60)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 500; j++ {
				s.Publish(jpg)
			}
		}()
	}

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				ch := s.subscribe()
				select {
				case <-ch:
				default:
				}
				s.unsubscribe(ch)
			}
		}()
	}

	wg.Wait()
}
