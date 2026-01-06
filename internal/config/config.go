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
	defaultListenAddr  = "0.0.0.0:8787"
	defaultDataDir     = "./data"
	defaultFFmpegPath  = "ffmpeg"
	defaultFPS         = 30
	defaultBitrateKbps = 6000
	defaultMonitorIdx  = 1
)

// Config holds runtime configuration values.
type Config struct {
	ListenAddr   string
	UIPassword   string
	DataDir      string
	CalibPath    string
	FFmpegPath   string
	FPS          int
	BitrateKbps  int
	MonitorIndex int
}

// Load reads configuration from ./data/.env and environment variables.
func Load() (Config, error) {
	cfg := Config{
		ListenAddr:   defaultListenAddr,
		DataDir:      defaultDataDir,
		CalibPath:    filepath.Join(defaultDataDir, "calib.json"),
		FFmpegPath:   defaultFFmpegPath,
		FPS:          defaultFPS,
		BitrateKbps:  defaultBitrateKbps,
		MonitorIndex: defaultMonitorIdx,
	}

	if err := loadEnvFile(filepath.Join(cfg.DataDir, ".env")); err != nil {
		return Config{}, err
	}

	cfg.ListenAddr = envString("LISTEN_ADDR", cfg.ListenAddr)
	cfg.DataDir = envString("DATA_DIR", cfg.DataDir)
	cfg.CalibPath = envString("CALIB_PATH", filepath.Join(cfg.DataDir, "calib.json"))
	cfg.FFmpegPath = envString("FFMPEG_PATH", cfg.FFmpegPath)
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

	if cfg.UIPassword == "" {
		return Config{}, errors.New("UI_PASSWORD is required")
	}

	return cfg, nil
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
