// Package ffmpeg builds ffmpeg command presets for streaming.
package ffmpeg

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/frudas24/deskslice/internal/calib"
	"github.com/frudas24/deskslice/internal/monitor"
)

const (
	// ModePresetup captures a full monitor.
	ModePresetup = "presetup"
	// ModeRun captures a cropped plugin rectangle.
	ModeRun = "run"
)

// Runner manages the ffmpeg process lifecycle.
type Runner struct {
	mu     sync.Mutex
	cmd    *exec.Cmd
	waitCh chan error
}

// NewRunner returns a new Runner instance.
func NewRunner() *Runner {
	return &Runner{}
}

// StartPresetup starts fullscreen capture and returns the RTP port and stop function.
func (r *Runner) StartPresetup(m monitor.Monitor, opts Options) (int, func() error, error) {
	return r.start(ModePresetup, m, calib.Rect{}, opts)
}

// StartPresetupOnPort starts fullscreen capture to a fixed RTP port (used to keep a stable RTP ingest port).
func (r *Runner) StartPresetupOnPort(m monitor.Monitor, opts Options, port int) (func() error, error) {
	_, stop, err := r.startOnPort(ModePresetup, m, calib.Rect{}, opts, port)
	return stop, err
}

// StartRun starts cropped capture and returns the RTP port and stop function.
func (r *Runner) StartRun(m monitor.Monitor, plugin calib.Rect, opts Options) (int, func() error, error) {
	return r.start(ModeRun, m, plugin, opts)
}

// StartRunOnPort starts cropped capture to a fixed RTP port (used to keep a stable RTP ingest port).
func (r *Runner) StartRunOnPort(m monitor.Monitor, plugin calib.Rect, opts Options, port int) (func() error, error) {
	_, stop, err := r.startOnPort(ModeRun, m, plugin, opts, port)
	return stop, err
}

// Stop terminates any running ffmpeg process.
func (r *Runner) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.stopLocked()
}

// Restart stops the current process and starts a new one.
func (r *Runner) Restart(mode string, m monitor.Monitor, plugin calib.Rect, opts Options) (int, func() error, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.stopLocked(); err != nil {
		return 0, nil, err
	}
	return r.startLocked(mode, m, plugin, opts)
}

// start is the public wrapper that handles locking.
func (r *Runner) start(mode string, m monitor.Monitor, plugin calib.Rect, opts Options) (int, func() error, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.startLocked(mode, m, plugin, opts)
}

// startOnPort starts ffmpeg using a fixed RTP port.
func (r *Runner) startOnPort(mode string, m monitor.Monitor, plugin calib.Rect, opts Options, port int) (int, func() error, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.startLockedOnPort(mode, m, plugin, opts, port)
}

// startLocked starts ffmpeg while holding the runner lock.
func (r *Runner) startLocked(mode string, m monitor.Monitor, plugin calib.Rect, opts Options) (int, func() error, error) {
	if opts.FFmpegPath == "" {
		return 0, nil, errors.New("FFmpegPath is required")
	}
	if opts.FPS <= 0 {
		opts.FPS = 30
	}
	if opts.BitrateKbps <= 0 {
		opts.BitrateKbps = 6000
	}

	port, err := allocatePort()
	if err != nil {
		return 0, nil, err
	}

	return r.startLockedOnPort(mode, m, plugin, opts, port)
}

// startLockedOnPort starts ffmpeg while holding the runner lock, targeting the provided RTP port.
func (r *Runner) startLockedOnPort(mode string, m monitor.Monitor, plugin calib.Rect, opts Options, port int) (int, func() error, error) {
	if port <= 0 {
		return 0, nil, fmt.Errorf("invalid rtp port %d", port)
	}

	useD3D11 := opts.CaptureDriver == "" || strings.EqualFold(opts.CaptureDriver, "d3d11grab")
	args, err := buildArgs(mode, m, plugin, opts, port, useD3D11)
	if err != nil {
		return 0, nil, err
	}
	log.Printf("ffmpeg: start %s %s", opts.FFmpegPath, strings.Join(args, " "))

	cmd, waitCh, err := startWithRetry(opts.FFmpegPath, args, func() ([]string, error) {
		return buildArgs(mode, m, plugin, opts, port, false)
	})
	if err != nil {
		return 0, nil, err
	}

	r.cmd = cmd
	r.waitCh = waitCh
	stop := func() error {
		r.mu.Lock()
		defer r.mu.Unlock()
		return r.stopLocked()
	}
	return port, stop, nil
}

// stopLocked stops the current ffmpeg process without acquiring the lock.
func (r *Runner) stopLocked() error {
	if r.cmd == nil || r.cmd.Process == nil {
		return nil
	}
	if err := r.cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return err
	}
	if r.waitCh != nil {
		<-r.waitCh
	}
	r.cmd = nil
	r.waitCh = nil
	return nil
}

// buildArgs selects the correct preset for the requested mode.
func buildArgs(mode string, m monitor.Monitor, plugin calib.Rect, opts Options, port int, useD3D11 bool) ([]string, error) {
	switch mode {
	case ModePresetup:
		return BuildPresetupArgs(m, opts, port, useD3D11), nil
	case ModeRun:
		return BuildRunArgs(m, plugin, opts, port, useD3D11), nil
	default:
		return nil, fmt.Errorf("unknown mode %q", mode)
	}
}

// startCmd launches ffmpeg with the provided args.
func startCmd(path string, args []string) (*exec.Cmd, error) {
	cmd := exec.Command(path, args...)
	configureCmd(cmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}

// startWithRetry launches ffmpeg and retries with backoff if it exits early.
func startWithRetry(path string, args []string, fallback func() ([]string, error)) (*exec.Cmd, chan error, error) {
	var (
		cmd    *exec.Cmd
		waitCh chan error
		err    error
	)
	backoff := 500 * time.Millisecond
	for attempt := 0; attempt < 3; attempt++ {
		cmd, waitCh, err = startWithFallback(path, args, fallback)
		if err == nil {
			return cmd, waitCh, nil
		}
		time.Sleep(backoff)
		backoff *= 2
	}
	return nil, nil, err
}

// startWithFallback launches ffmpeg and falls back if it exits early.
func startWithFallback(path string, args []string, fallback func() ([]string, error)) (*exec.Cmd, chan error, error) {
	cmd, err := startCmd(path, args)
	if err != nil {
		return nil, nil, err
	}
	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()

	exited, exitErr := waitForExit(waitCh, 700*time.Millisecond)
	if exited {
		if exitErr != nil {
			log.Printf("ffmpeg: exited early: %v", exitErr)
		} else {
			log.Printf("ffmpeg: exited early without error")
		}
		fallbackArgs, err := fallback()
		if err != nil {
			return nil, nil, err
		}
		log.Printf("ffmpeg: fallback %s %s", path, strings.Join(fallbackArgs, " "))
		cmd, err = startCmd(path, fallbackArgs)
		if err != nil {
			if exitErr != nil {
				return nil, nil, fmt.Errorf("ffmpeg exited early: %w", exitErr)
			}
			return nil, nil, err
		}
		waitCh = make(chan error, 1)
		go func() {
			waitCh <- cmd.Wait()
		}()
		exited, exitErr = waitForExit(waitCh, 700*time.Millisecond)
		if exited {
			if exitErr != nil {
				log.Printf("ffmpeg: fallback exited early: %v", exitErr)
			} else {
				log.Printf("ffmpeg: fallback exited early without error")
			}
			if exitErr != nil {
				return nil, nil, fmt.Errorf("ffmpeg exited early: %w", exitErr)
			}
			return nil, nil, fmt.Errorf("ffmpeg exited early")
		}
	}

	return cmd, waitCh, nil
}

// waitForExit waits for a process to exit or times out.
func waitForExit(waitCh <-chan error, timeout time.Duration) (bool, error) {
	select {
	case err := <-waitCh:
		return true, err
	case <-time.After(timeout):
		return false, nil
	}
}

// allocatePort reserves a local UDP port and returns it.
func allocatePort() (int, error) {
	addr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return 0, err
	}
	port := conn.LocalAddr().(*net.UDPAddr).Port
	if err := conn.Close(); err != nil {
		return 0, err
	}
	return port, nil
}
