// Package config loads environment configuration for DeskSlice.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	defaultListenAddr      = "0.0.0.0:8787"
	defaultDataDir         = "./data"
	defaultFFmpegPath      = "ffmpeg"
	defaultCapture         = "gdigrab"
	defaultFPS             = 30
	defaultBitrateKbps     = 6000
	defaultMonitorIdx      = 1
	defaultMJPEGEnabled    = true
	defaultMJPEGIntervalMs = 120
	defaultMJPEGQuality    = 60
	defaultScrollHoldMs    = 2500
	defaultScrollTickMs    = 50
	defaultScrollMaxDelta  = 240
)

// Config holds runtime configuration values.
type Config struct {
	ListenAddr      string
	UIPassword      string
	DataDir         string
	CalibPath       string
	FFmpegPath      string
	CaptureDriver   string
	FPS             int
	BitrateKbps     int
	MonitorIndex    int
	MJPEGEnabled    bool
	MJPEGIntervalMs int
	MJPEGQuality    int
	ScrollHoldMs    int
	ScrollTickMs    int
	ScrollMaxDelta  int
}

// Load reads configuration from ./data/.env and environment variables.
func Load() (Config, error) {
	cfg := Config{
		ListenAddr:      defaultListenAddr,
		DataDir:         defaultDataDir,
		CalibPath:       filepath.Join(defaultDataDir, "calib.json"),
		FFmpegPath:      defaultFFmpegPath,
		CaptureDriver:   defaultCapture,
		FPS:             defaultFPS,
		BitrateKbps:     defaultBitrateKbps,
		MonitorIndex:    defaultMonitorIdx,
		MJPEGEnabled:    defaultMJPEGEnabled,
		MJPEGIntervalMs: defaultMJPEGIntervalMs,
		MJPEGQuality:    defaultMJPEGQuality,
		ScrollHoldMs:    defaultScrollHoldMs,
		ScrollTickMs:    defaultScrollTickMs,
		ScrollMaxDelta:  defaultScrollMaxDelta,
	}

	if err := loadEnvFile(filepath.Join(cfg.DataDir, ".env")); err != nil {
		return Config{}, err
	}

	cfg.ListenAddr = envString("LISTEN_ADDR", cfg.ListenAddr)
	cfg.DataDir = envString("DATA_DIR", cfg.DataDir)
	cfg.CalibPath = envString("CALIB_PATH", filepath.Join(cfg.DataDir, "calib.json"))
	cfg.FFmpegPath = envString("FFMPEG_PATH", cfg.FFmpegPath)
	cfg.CaptureDriver = normalizeCaptureDriver(envString("CAPTURE_DRIVER", cfg.CaptureDriver))
	cfg.UIPassword = strings.TrimSpace(os.Getenv("UI_PASSWORD"))

	fps, err := envInt("FPS", cfg.FPS)
	if err != nil {
		return Config{}, err
	}
	cfg.FPS = fps

	bitrate, err := envInt("BITRATE_KBPS", cfg.BitrateKbps)
	if err != nil {
		return Config{}, err
	}
	cfg.BitrateKbps = bitrate

	monitorIdx, err := envInt("MONITOR_INDEX", cfg.MonitorIndex)
	if err != nil {
		return Config{}, err
	}
	cfg.MonitorIndex = monitorIdx

	cfg.MJPEGEnabled = envBool("MJPEG_ENABLED", cfg.MJPEGEnabled)

	mjpegInterval, err := envInt("MJPEG_INTERVAL_MS", cfg.MJPEGIntervalMs)
	if err != nil {
		return Config{}, err
	}
	cfg.MJPEGIntervalMs = mjpegInterval

	mjpegQuality, err := envInt("MJPEG_QUALITY", cfg.MJPEGQuality)
	if err != nil {
		return Config{}, err
	}
	if mjpegQuality <= 0 || mjpegQuality > 100 {
		return Config{}, fmt.Errorf("MJPEG_QUALITY must be 1-100")
	}
	cfg.MJPEGQuality = mjpegQuality

	scrollHold, err := envInt("SCROLL_OVERLAY_HOLD_MS", cfg.ScrollHoldMs)
	if err != nil {
		return Config{}, err
	}
	if scrollHold < 0 {
		return Config{}, fmt.Errorf("SCROLL_OVERLAY_HOLD_MS must be >= 0")
	}
	cfg.ScrollHoldMs = scrollHold

	scrollTick, err := envInt("SCROLL_OVERLAY_TICK_MS", cfg.ScrollTickMs)
	if err != nil {
		return Config{}, err
	}
	if scrollTick <= 0 {
		return Config{}, fmt.Errorf("SCROLL_OVERLAY_TICK_MS must be > 0")
	}
	cfg.ScrollTickMs = scrollTick

	scrollMaxDelta, err := envInt("SCROLL_OVERLAY_MAX_DELTA", cfg.ScrollMaxDelta)
	if err != nil {
		return Config{}, err
	}
	if scrollMaxDelta <= 0 {
		return Config{}, fmt.Errorf("SCROLL_OVERLAY_MAX_DELTA must be > 0")
	}
	cfg.ScrollMaxDelta = scrollMaxDelta

	if cfg.UIPassword == "" {
		return Config{}, errors.New("UI_PASSWORD is required")
	}

	return cfg, nil
}

// normalizeCaptureDriver ensures a supported capture driver value.
func normalizeCaptureDriver(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "d3d11grab":
		return "d3d11grab"
	default:
		return "gdigrab"
	}
}

// envString returns an env override when present, otherwise a default.
func envString(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

// envInt returns an int env override when present, otherwise a default.
func envInt(key string, def int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	return value, nil
}

// envBool returns a bool env override when present, otherwise a default.
func envBool(key string, def bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return def
	}
}

// loadEnvFile loads KEY=VALUE pairs from a .env file.
func loadEnvFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := parseEnvLine(line)
		if !ok {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			if err := os.Setenv(key, value); err != nil {
				return err
			}
		}
	}

	return nil
}

// parseEnvLine parses a single .env line into key/value.
func parseEnvLine(line string) (string, string, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false
	}
	if strings.HasPrefix(line, "export ") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
	}
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])
	if key == "" {
		return "", "", false
	}
	value = strings.Trim(value, `"'`)
	return key, value, true
}
