// Package ffmpeg builds ffmpeg command presets for streaming.
package ffmpeg

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
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

// StartRun starts cropped capture and returns the RTP port and stop function.
func (r *Runner) StartRun(m monitor.Monitor, plugin calib.Rect, opts Options) (int, func() error, error) {
	return r.start(ModeRun, m, plugin, opts)
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

	args, err := buildArgs(mode, m, plugin, opts, port, true)
	if err != nil {
		return 0, nil, err
	}

	cmd, waitCh, err := startWithFallback(opts.FFmpegPath, args, func() ([]string, error) {
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
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
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
		_ = cmd.Process.Kill()
		<-waitCh
		fallbackArgs, err := fallback()
		if err != nil {
			return nil, nil, err
		}
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
