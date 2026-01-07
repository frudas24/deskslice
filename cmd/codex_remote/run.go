// Package main starts the DeskSlice server.
package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/frudas24/deskslice/internal/app"
	"github.com/frudas24/deskslice/internal/config"
	"github.com/frudas24/deskslice/internal/ffmpeg"
	"github.com/frudas24/deskslice/internal/session"
	"github.com/frudas24/deskslice/internal/signaling"
	"github.com/frudas24/deskslice/internal/webrtc"
	"github.com/frudas24/deskslice/internal/wininput"
)

// run wires the application and blocks until shutdown.
func run(debug bool) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	webrtc.SetDebugLogging(debug)
	if debug {
		log.Printf("debug: enabled")
	}
	logStartup(cfg)

	sess := session.New(cfg.UIPassword)
	runner := ffmpeg.NewRunner()

	publisher, err := webrtc.NewPublisher()
	if err != nil {
		return err
	}

	injector, err := wininput.NewInjector()
	if err != nil {
		return err
	}

	appInstance, err := app.New(cfg, sess, runner, publisher, injector, signaling.ViewerReplace)
	if err != nil {
		return err
	}
	if err := appInstance.Start(); err != nil {
		return err
	}
	defer func() {
		if err := appInstance.Stop(); err != nil {
			log.Printf("shutdown: %v", err)
		}
	}()

	mux := http.NewServeMux()
	appInstance.RegisterRoutes(mux, "")
	server := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: mux,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil {
			errCh <- err
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}

// logFatal prints and exits for startup failures.
func logFatal(err error) {
	log.Printf("fatal: %v", err)
	os.Exit(1)
}

// logStartup prints startup checks and connection info.
func logStartup(cfg config.Config) {
	log.Printf("DeskSlice starting")
	logEnvStatus(cfg)
	logFFmpegStatus(cfg.FFmpegPath)
	log.Printf("capture driver: %s", cfg.CaptureDriver)
	logListenStatus(cfg.ListenAddr)
}

// logEnvStatus reports whether a .env file was found and required values are set.
func logEnvStatus(cfg config.Config) {
	envPath := filepath.Join(cfg.DataDir, ".env")
	if fileExists(envPath) {
		log.Printf("env check: ok (%s)", envPath)
	} else {
		log.Printf("env check: missing (%s)", envPath)
	}
	if cfg.PasswordMode {
		if strings.TrimSpace(os.Getenv("UI_PASSWORD")) == "" {
			log.Printf("env UI_PASSWORD: missing")
		} else {
			log.Printf("env UI_PASSWORD: set")
		}
	} else {
		log.Printf("env PASSWORD_MODE: disabled (dev mode)")
	}
}

// logFFmpegStatus reports whether the ffmpeg binary is discoverable.
func logFFmpegStatus(path string) {
	resolved := path
	ok := false
	note := ""

	if filepath.IsAbs(path) {
		info, err := os.Stat(path)
		switch {
		case err == nil && !info.IsDir():
			ok = true
		case err != nil:
			note = err.Error()
		default:
			note = "path is a directory"
		}
	} else {
		found, err := exec.LookPath(path)
		switch {
		case err == nil:
			ok = true
			resolved = found
		case errors.Is(err, exec.ErrDot):
			note = "found relative to current dir; use absolute path"
		default:
			note = err.Error()
		}
	}

	if ok {
		log.Printf("ffmpeg check: ok (%s)", resolved)
		return
	}
	if note != "" {
		log.Printf("ffmpeg check: missing (%s)", note)
		return
	}
	log.Printf("ffmpeg check: missing")
}

// logListenStatus reports the listen address and a local URL helper.
func logListenStatus(addr string) {
	log.Printf("listen addr: %s", addr)
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "localhost"
	}
	log.Printf("local url: http://%s", net.JoinHostPort(host, port))
}

// fileExists reports whether a path exists and is a file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
