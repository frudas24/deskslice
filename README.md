# DeskSlice

Remote control for the real Codex panel inside VS Code, designed for LAN use
from a phone. The goal is low-latency streaming + input injection with manual
calibration (no OCR, no UI re-implementation).

## Status

MVP scaffolding complete for Windows host. The full spec lives in `TODO/TODO_0001.md`.

## Highlights

- WebRTC video stream (H264) of the Codex panel.
- Touch input mapping (tap, drag-scroll, typing).
- Presetup mode to select monitor and trace plugin/chat/scroll rectangles.
- Run mode with a cropped stream to the Codex panel only.
- Simple password gate via `.env`.

## Requirements

- Windows 11 (host).
- Go 1.25+.
- `ffmpeg` available in PATH or configured via `FFMPEG_PATH` (absolute path recommended on Windows).

## Quick Start

1. Copy sample env: `cp data/.env.sample data/.env` and edit as needed.
2. Set `UI_PASSWORD` in `data/.env`.
3. (Windows) Set `FFMPEG_PATH` to a full path like `C:\tools\ffmpeg\bin\ffmpeg.exe` if not in PATH.
4. Build:
   - Windows: `make build OS=windows ARCH=amd64` or `GOOS=windows GOARCH=amd64 go build -o dist/win64/codex_remote.exe ./cmd/codex_remote`
   - Linux: `make build` (outputs to `dist/linux_x86-64/`)
5. Run:
   - Windows: `dist/win64/codex_remote.exe`
   - Linux: `dist/linux_x86-64/codex_remote`
6. Open `http://<host>:8787` on your phone and log in.

## Notes

- The server prefers `d3d11grab` and falls back to `gdigrab` if unavailable.
- The web client is plain HTML/CSS/JS under `internal/web/static/` (no Node build).
- Seeing **two `ffmpeg` processes is expected** when MJPEG preview is enabled:
  - `ffmpeg: start ... -f rtp rtp://127.0.0.1:<port>` is the main H264→RTP pipeline used by WebRTC.
  - `ffmpeg: preview ... -f rawvideo -` is a lightweight preview pipeline that feeds `/mjpeg/desktop` (used as a fallback when WebRTC is black or not connected).
  - Disable the preview with `MJPEG_ENABLED=false` in `data/.env`.

## License

GPL-3.0-only. See `LICENSE`.
