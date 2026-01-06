// Package ffmpeg builds ffmpeg command presets for streaming.
package ffmpeg

import (
	"fmt"

	"github.com/frudas24/deskslice/internal/calib"
	"github.com/frudas24/deskslice/internal/monitor"
)

// Options describes ffmpeg runtime parameters.
type Options struct {
	FFmpegPath  string
	FPS         int
	BitrateKbps int
}

// BuildPresetupArgs returns ffmpeg args for fullscreen capture.
func BuildPresetupArgs(m monitor.Monitor, opts Options, port int, useD3D11 bool) []string {
	input := buildInputArgs(m, opts, useD3D11)
	output := buildOutputArgs(opts, port, "")
	return append(input, output...)
}

// BuildRunArgs returns ffmpeg args for cropped capture.
func BuildRunArgs(m monitor.Monitor, plugin calib.Rect, opts Options, port int, useD3D11 bool) []string {
	plugin = calib.Normalize(plugin)
	crop := fmt.Sprintf("crop=%d:%d:%d:%d", plugin.W, plugin.H, plugin.X, plugin.Y)
	input := buildInputArgs(m, opts, useD3D11)
	output := buildOutputArgs(opts, port, crop)
	return append(input, output...)
}

// buildInputArgs builds the capture-side arguments.
func buildInputArgs(m monitor.Monitor, opts Options, useD3D11 bool) []string {
	grabber := "gdigrab"
	if useD3D11 {
		grabber = "d3d11grab"
	}
	return []string{
		"-f", grabber,
		"-framerate", fmt.Sprintf("%d", opts.FPS),
		"-offset_x", fmt.Sprintf("%d", m.X),
		"-offset_y", fmt.Sprintf("%d", m.Y),
		"-video_size", fmt.Sprintf("%dx%d", m.W, m.H),
		"-i", "desktop",
	}
}

// buildOutputArgs builds the encode/output arguments.
func buildOutputArgs(opts Options, port int, cropFilter string) []string {
	args := []string{
		"-an",
	}
	if cropFilter != "" {
		args = append(args, "-vf", cropFilter)
	}
	args = append(args,
		"-vcodec", "libx264",
		"-preset", "ultrafast",
		"-tune", "zerolatency",
		"-pix_fmt", "yuv420p",
		"-b:v", fmt.Sprintf("%dk", opts.BitrateKbps),
		"-f", "rtp",
		fmt.Sprintf("rtp://127.0.0.1:%d?pkt_size=1200", port),
	)
	return args
}
