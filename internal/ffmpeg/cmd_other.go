//go:build !windows

// Package ffmpeg builds ffmpeg command presets for streaming.
package ffmpeg

import "os/exec"

// configureCmd is a no-op outside Windows.
func configureCmd(cmd *exec.Cmd) {
	_ = cmd
}
