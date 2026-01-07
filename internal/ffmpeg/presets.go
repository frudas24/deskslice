// Package ffmpeg builds ffmpeg command presets for streaming.
package ffmpeg

import (
	"fmt"

	"github.com/frudas24/deskslice/internal/calib"
	"github.com/frudas24/deskslice/internal/monitor"
)

// Options describes ffmpeg runtime parameters.
type Options struct {
	FFmpegPath    string
	FPS           int
	BitrateKbps   int
	CaptureDriver string
}

// BuildPresetupArgs returns ffmpeg args for fullscreen capture.
func BuildPresetupArgs(m monitor.Monitor, opts Options, port int, useD3D11 bool) []string {
	input := buildInputArgs(m, opts, useD3D11)
	output := buildOutputArgs(opts, port, "")
	return append(input, output...)
}

// BuildRunArgs returns ffmpeg args for cropped capture.
func BuildRunArgs(m monitor.Monitor, plugin calib.Rect, opts Options, port int, useD3D11 bool) []string {
	plugin = normalizeCropRect(plugin, m)
	crop := fmt.Sprintf("crop=%d:%d:%d:%d", plugin.W, plugin.H, plugin.X, plugin.Y)
	input := buildInputArgs(m, opts, useD3D11)
	output := buildOutputArgs(opts, port, crop)
	return append(input, output...)
}

// buildInputArgs builds the capture-side arguments.
func buildInputArgs(m monitor.Monitor, opts Options, useD3D11 bool) []string {
	grabber := "gdigrab"
	if driver := opts.CaptureDriver; driver != "" && driver != "gdigrab" {
		grabber = driver
	} else if useD3D11 {
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
	// Keep keyframes frequent to help decoders recover quickly after restarts/crop changes.
	keyint := opts.FPS
	if keyint <= 0 {
		keyint = 30
	}
	if keyint < 15 {
		keyint = 15
	}
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
		"-profile:v", "baseline",
		"-g", fmt.Sprintf("%d", keyint),
		"-keyint_min", fmt.Sprintf("%d", keyint),
		"-bf", "0",
		"-x264-params", "scenecut=0:repeat-headers=1",
		"-pix_fmt", "yuv420p",
		"-b:v", fmt.Sprintf("%dk", opts.BitrateKbps),
		"-payload_type", "96",
		"-f", "rtp",
		fmt.Sprintf("rtp://127.0.0.1:%d?pkt_size=1200", port),
	)
	return args
}

// normalizeCropRect clamps and aligns a crop rectangle to even dimensions.
func normalizeCropRect(r calib.Rect, m monitor.Monitor) calib.Rect {
	r = calib.Normalize(r)
	if r.W < 2 {
		r.W = 2
	}
	if r.H < 2 {
		r.H = 2
	}
	if r.X < 0 {
		r.X = 0
	}
	if r.Y < 0 {
		r.Y = 0
	}
	if r.X+r.W > m.W {
		r.X = maxInt(0, m.W-r.W)
	}
	if r.Y+r.H > m.H {
		r.Y = maxInt(0, m.H-r.H)
	}

	if r.X%2 != 0 {
		r.X--
	}
	if r.Y%2 != 0 {
		r.Y--
	}
	if r.W%2 != 0 {
		r.W--
	}
	if r.H%2 != 0 {
		r.H--
	}
	if r.W < 2 {
		r.W = 2
	}
	if r.H < 2 {
		r.H = 2
	}
	if r.X < 0 {
		r.X = 0
	}
	if r.Y < 0 {
		r.Y = 0
	}
	if r.X+r.W > m.W {
		r.X = maxInt(0, m.W-r.W)
	}
	if r.Y+r.H > m.H {
		r.Y = maxInt(0, m.H-r.H)
	}
	if r.X%2 != 0 {
		r.X--
	}
	if r.Y%2 != 0 {
		r.Y--
	}
	if r.W%2 != 0 {
		r.W--
	}
	if r.H%2 != 0 {
		r.H--
	}
	if r.W < 2 {
		r.W = 2
	}
	if r.H < 2 {
		r.H = 2
	}
	return r
}

// maxInt returns the larger integer.
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
