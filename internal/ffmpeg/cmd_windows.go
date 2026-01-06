//go:build windows

// Package ffmpeg builds ffmpeg command presets for streaming.
package ffmpeg

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// configureCmd applies Windows-specific process settings.
func configureCmd(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NO_WINDOW,
	}
}
