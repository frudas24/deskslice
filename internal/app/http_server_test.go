package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/frudas24/deskslice/internal/config"
	"github.com/frudas24/deskslice/internal/ffmpeg"
	"github.com/frudas24/deskslice/internal/mjpeg"
	"github.com/frudas24/deskslice/internal/session"
)

// TestHandleConfig_Unauthorized verifies /api/config requires authentication.
func TestHandleConfig_Unauthorized(t *testing.T) {
	sess := session.New("pw")
	app := newTestAppForConfig(sess, 120, 60)

	req := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()
	app.handleConfig(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

// TestHandleConfig_UpdatesRuntimeSettings verifies updating MJPEG interval/quality updates runtime config.
func TestHandleConfig_UpdatesRuntimeSettings(t *testing.T) {
	sess := session.New("pw")
	if !sess.Authenticate("pw") {
		t.Fatalf("expected authenticate success")
	}
	sess.SetVideoMode(session.VideoWebRTC)
	app := newTestAppForConfig(sess, 120, 60)

	body := `{"mjpegIntervalMs":80,"mjpegQuality":90}`
	req := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	app.handleConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp configResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !resp.Applied || resp.MJPEGIntervalMs != 80 || resp.MJPEGQuality != 90 {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if app.cfg.MJPEGIntervalMs != 80 || app.cfg.MJPEGQuality != 90 {
		t.Fatalf("unexpected app cfg: interval=%d quality=%d", app.cfg.MJPEGIntervalMs, app.cfg.MJPEGQuality)
	}
}

// TestHandleConfig_ResetRestoresDefaults verifies reset restores the .env defaults captured at startup.
func TestHandleConfig_ResetRestoresDefaults(t *testing.T) {
	sess := session.New("pw")
	if !sess.Authenticate("pw") {
		t.Fatalf("expected authenticate success")
	}
	sess.SetVideoMode(session.VideoWebRTC)
	app := newTestAppForConfig(sess, 120, 60)

	reqUpdate := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewBufferString(`{"mjpegIntervalMs":80,"mjpegQuality":90}`))
	recUpdate := httptest.NewRecorder()
	app.handleConfig(recUpdate, reqUpdate)
	if recUpdate.Code != http.StatusOK {
		t.Fatalf("expected update 200, got %d: %s", recUpdate.Code, recUpdate.Body.String())
	}

	reqReset := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewBufferString(`{"reset":true}`))
	recReset := httptest.NewRecorder()
	app.handleConfig(recReset, reqReset)

	if recReset.Code != http.StatusOK {
		t.Fatalf("expected reset 200, got %d: %s", recReset.Code, recReset.Body.String())
	}
	var resp configResponse
	if err := json.Unmarshal(recReset.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !resp.Applied || resp.MJPEGIntervalMs != 120 || resp.MJPEGQuality != 60 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

// TestHandleConfig_ValidatesInput verifies the endpoint rejects invalid values.
func TestHandleConfig_ValidatesInput(t *testing.T) {
	sess := session.New("pw")
	if !sess.Authenticate("pw") {
		t.Fatalf("expected authenticate success")
	}
	sess.SetVideoMode(session.VideoWebRTC)
	app := newTestAppForConfig(sess, 120, 60)

	reqBad := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewBufferString(`{"mjpegIntervalMs":1,"mjpegQuality":500}`))
	recBad := httptest.NewRecorder()
	app.handleConfig(recBad, reqBad)

	if recBad.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recBad.Code, recBad.Body.String())
	}
}

// newTestAppForConfig returns an App suitable for handleConfig tests without starting ffmpeg.
func newTestAppForConfig(sess *session.Session, intervalMs int, quality int) *App {
	stream := mjpeg.NewStream(time.Duration(intervalMs) * time.Millisecond)
	cfg := config.Config{
		MJPEGIntervalMs: intervalMs,
		MJPEGQuality:    quality,
	}
	return &App{
		cfg: cfg,
		defaultMJPEG: mjpegDefaults{
			intervalMs: intervalMs,
			quality:    quality,
		},
		session:       sess,
		previewStream: stream,
		preview:       ffmpeg.NewPreview(stream, quality),
	}
}
