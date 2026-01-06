// Package ffmpeg builds ffmpeg command presets for streaming.
package ffmpeg

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/frudas24/deskslice/internal/calib"
	"github.com/frudas24/deskslice/internal/mjpeg"
	"github.com/frudas24/deskslice/internal/monitor"
)

const previewRestartBackoff = 2 * time.Second

// Preview captures raw frames via ffmpeg and publishes MJPEG previews.
type Preview struct {
	mu      sync.Mutex
	cmd     *exec.Cmd
	stdout  io.ReadCloser
	stream  *mjpeg.Stream
	quality int
	w       int
	h       int
	closed  bool
	path    string
	args    []string
}

// NewPreview returns a preview pipeline bound to the given MJPEG stream.
func NewPreview(stream *mjpeg.Stream, quality int) *Preview {
	if quality <= 0 || quality > 100 {
		quality = 60
	}
	return &Preview{
		stream:  stream,
		quality: quality,
	}
}

// StartPresetup starts a full-screen MJPEG preview for the selected monitor.
func (p *Preview) StartPresetup(m monitor.Monitor, opts Options) error {
	return p.start(m, calib.Rect{}, opts, false)
}

// StartRun starts a cropped MJPEG preview of the plugin area.
func (p *Preview) StartRun(m monitor.Monitor, plugin calib.Rect, opts Options) error {
	return p.start(m, plugin, opts, true)
}

// Stop terminates the preview process.
func (p *Preview) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	return p.stopLocked()
}

// start configures and launches the preview pipeline for the given mode.
func (p *Preview) start(m monitor.Monitor, plugin calib.Rect, opts Options, cropped bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = false
	if err := p.stopLocked(); err != nil {
		return err
	}
	if opts.FFmpegPath == "" {
		return errors.New("FFmpegPath is required")
	}
	if opts.FPS <= 0 {
		opts.FPS = 30
	}
	useD3D11 := opts.CaptureDriver == "" || strings.EqualFold(opts.CaptureDriver, "d3d11grab")
	args := buildInputArgs(m, opts, useD3D11)

	outW, outH := m.W, m.H
	if cropped {
		plugin = normalizeCropRect(plugin, m)
		outW, outH = plugin.W, plugin.H
		args = append(args, "-vf", fmt.Sprintf("crop=%d:%d:%d:%d", plugin.W, plugin.H, plugin.X, plugin.Y))
	}
	args = append(args, "-an", "-pix_fmt", "rgb24", "-f", "rawvideo", "-")

	p.path = opts.FFmpegPath
	p.args = args
	p.w = outW
	p.h = outH

	log.Printf("ffmpeg: preview %s %s", p.path, strings.Join(args, " "))
	if err := p.startProcessLocked(); err != nil {
		return err
	}
	go p.loop()
	return nil
}

// startProcessLocked launches ffmpeg while holding the preview lock.
func (p *Preview) startProcessLocked() error {
	cmd := exec.Command(p.path, append([]string{"-hide_banner", "-loglevel", "error"}, p.args...)...)
	configureCmd(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	p.cmd = cmd
	p.stdout = stdout
	return nil
}

// stopLocked stops any running ffmpeg process while holding the preview lock.
func (p *Preview) stopLocked() error {
	if p.stdout != nil {
		_ = p.stdout.Close()
		p.stdout = nil
	}
	if p.cmd != nil && p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
		_, _ = p.cmd.Process.Wait()
	}
	p.cmd = nil
	return nil
}

// loop reads raw frames and publishes them to the MJPEG stream.
func (p *Preview) loop() {
	raw := make([]byte, p.w*p.h*3)
	for {
		p.mu.Lock()
		stdout := p.stdout
		closed := p.closed
		p.mu.Unlock()
		if closed || stdout == nil {
			return
		}
		if _, err := io.ReadFull(stdout, raw); err != nil {
			if !p.handleReadError(err) {
				return
			}
			continue
		}
		if p.stream != nil {
			jpg := mjpeg.EncodeRGBToJPEG(raw, p.w, p.h, p.quality)
			p.stream.Publish(jpg)
		}
	}
}

// handleReadError restarts ffmpeg after a read failure.
func (p *Preview) handleReadError(err error) bool {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return false
	}
	p.mu.Unlock()

	log.Printf("ffmpeg: preview read error: %v (restart in %s)", err, previewRestartBackoff)
	time.Sleep(previewRestartBackoff)

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return false
	}
	if err := p.stopLocked(); err != nil {
		log.Printf("ffmpeg: preview stop error: %v", err)
		return false
	}
	if err := p.startProcessLocked(); err != nil {
		log.Printf("ffmpeg: preview restart error: %v", err)
		return false
	}
	return true
}
