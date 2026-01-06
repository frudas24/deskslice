// Package app wires HTTP, signaling, and pipeline state together.
package app

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/frudas24/deskslice/internal/calib"
	"github.com/frudas24/deskslice/internal/config"
	"github.com/frudas24/deskslice/internal/control"
	"github.com/frudas24/deskslice/internal/ffmpeg"
	"github.com/frudas24/deskslice/internal/mjpeg"
	"github.com/frudas24/deskslice/internal/monitor"
	"github.com/frudas24/deskslice/internal/session"
	"github.com/frudas24/deskslice/internal/signaling"
	"github.com/frudas24/deskslice/internal/webrtc"
	"github.com/frudas24/deskslice/internal/wininput"
)

// App coordinates the HTTP API, websocket servers, and media pipeline.
type App struct {
	mu            sync.Mutex
	cfg           config.Config
	session       *session.Session
	runner        *ffmpeg.Runner
	preview       *ffmpeg.Preview
	previewStream *mjpeg.Stream
	publisher     *webrtc.Publisher
	signaling     *signaling.Server
	control       *control.Server
	monitors      []monitor.Monitor
}

// New creates a new application with its dependencies wired.
func New(cfg config.Config, sess *session.Session, runner *ffmpeg.Runner, publisher *webrtc.Publisher, injector wininput.Injector, policy signaling.ViewerPolicy) (*App, error) {
	if sess == nil {
		return nil, errors.New("session is required")
	}
	if runner == nil {
		return nil, errors.New("ffmpeg runner is required")
	}
	if publisher == nil {
		return nil, errors.New("webrtc publisher is required")
	}
	if injector == nil {
		return nil, errors.New("injector is required")
	}

	app := &App{
		cfg:       cfg,
		session:   sess,
		runner:    runner,
		publisher: publisher,
	}
	if cfg.MJPEGEnabled {
		interval := time.Duration(cfg.MJPEGIntervalMs) * time.Millisecond
		app.previewStream = mjpeg.NewStream(interval)
		app.preview = ffmpeg.NewPreview(app.previewStream, cfg.MJPEGQuality)
	}

	app.signaling = signaling.NewServer(publisher, policy, sess.IsAuthenticated)
	app.control = control.NewServer(sess, injector, app.ListMonitors, func(reason string) {
		_ = app.RestartPipeline(reason)
	}, func(c calib.Calib) error {
		return calib.Save(cfg.CalibPath, c)
	})

	return app, nil
}

// Start initializes runtime state and starts the presetup pipeline.
func (a *App) Start() error {
	monitors, err := monitor.ListMonitors()
	if err != nil {
		return err
	}
	a.monitors = monitors
	logMonitors(monitors)

	c, err := calib.Load(a.cfg.CalibPath)
	if err != nil {
		return err
	}
	a.session.SetCalib(c)

	monitorIndex := a.cfg.MonitorIndex
	if c.MonitorIndex > 0 {
		monitorIndex = c.MonitorIndex
	}
	a.session.SetMonitor(monitorIndex)
	a.session.SetMode(session.ModePresetup)

	return a.RestartPipeline("startup")
}

// Stop shuts down the media pipeline and closes active peers.
func (a *App) Stop() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.publisher.StopForwarding()
	a.publisher.ClosePeer()
	if a.preview != nil {
		_ = a.preview.Stop()
	}
	return a.runner.Stop()
}

// logMonitors prints the available monitors for debugging.
func logMonitors(list []monitor.Monitor) {
	for _, m := range list {
		log.Printf("monitor %d: %dx%d @ %d,%d primary=%t", m.Index, m.W, m.H, m.X, m.Y, m.Primary)
	}
}

// RestartPipeline restarts ffmpeg and RTP forwarding.
func (a *App) RestartPipeline(reason string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.publisher.StopForwarding()
	if err := a.runner.Stop(); err != nil {
		return err
	}

	mode := a.session.Mode()
	monitors := a.monitors
	if len(monitors) == 0 {
		return fmt.Errorf("no monitors loaded")
	}
	m, ok := monitor.GetMonitorByIndex(monitors, a.session.Monitor())
	if !ok {
		return fmt.Errorf("monitor %d not found", a.session.Monitor())
	}

	opts := ffmpeg.Options{
		FFmpegPath:    a.cfg.FFmpegPath,
		FPS:           a.cfg.FPS,
		BitrateKbps:   a.cfg.BitrateKbps,
		CaptureDriver: a.cfg.CaptureDriver,
	}

	var (
		port int
		err  error
	)
	if mode == session.ModeRun {
		c := a.session.GetCalib()
		port, _, err = a.runner.StartRun(m, c.PluginAbs, opts)
	} else {
		port, _, err = a.runner.StartPresetup(m, opts)
	}
	if err != nil {
		return err
	}

	if err := a.publisher.AttachRTP(port); err != nil {
		return err
	}
	if err := a.publisher.StartForwarding(); err != nil {
		return err
	}
	a.restartPreview(mode, m, opts)
	a.signaling.NotifyRestart()
	_ = reason
	return nil
}

// PreviewStream returns the MJPEG preview stream, if enabled.
func (a *App) PreviewStream() *mjpeg.Stream {
	return a.previewStream
}

// restartPreview starts or restarts the MJPEG preview pipeline.
func (a *App) restartPreview(mode string, m monitor.Monitor, opts ffmpeg.Options) {
	if a.preview == nil {
		return
	}
	var err error
	if mode == session.ModeRun {
		c := a.session.GetCalib()
		err = a.preview.StartRun(m, c.PluginAbs, opts)
	} else {
		err = a.preview.StartPresetup(m, opts)
	}
	if err != nil {
		log.Printf("preview: start failed: %v", err)
	}
}

// ListMonitors returns the cached monitor list.
func (a *App) ListMonitors() ([]monitor.Monitor, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]monitor.Monitor, len(a.monitors))
	copy(out, a.monitors)
	return out, nil
}

// Signaling returns the signaling websocket handler.
func (a *App) Signaling() *signaling.Server {
	return a.signaling
}

// Control returns the control websocket handler.
func (a *App) Control() *control.Server {
	return a.control
}
