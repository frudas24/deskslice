// Package main starts the DeskSlice server.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
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
func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

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

	appInstance, err := app.New(cfg, sess, runner, publisher, injector, signaling.ViewerReject)
	if err != nil {
		return err
	}
	if err := appInstance.Start(); err != nil {
		return err
	}

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
