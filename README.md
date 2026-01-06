# DeskSlice

Remote control for the real Codex panel inside VS Code, designed for LAN use
from a phone. The goal is low-latency streaming + input injection with manual
calibration (no OCR, no UI re-implementation).

## Status

MVP scaffolding complete for Windows host. The full spec lives in `TODO/TODO_0001.md`.

## Highlights

- WebRTC video stream (H264) of the Codex panel.
- MJPEG preview mode (default, more reliable) with optional WebRTC switching.
- Touch input mapping (tap, drag-scroll, typing).
- Presetup mode to select monitor and trace plugin/chat/scroll rectangles.
- Run mode with a cropped stream to the Codex panel only.
- Simple password gate via `.env`.
- Fullscreen mobile UX with side drawers, scaling controls, and input/scroll toggles.

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

## UI Tips

- Fullscreen: tap the video to show/hide the overlay controls; use `Menu` and `Chat` drawers.
- Mouse lock: in fullscreen, the mouse icon toggles whether touches send input; when locked, you can pinch-zoom and pan the video locally (no host input).
- Scroll mode: in fullscreen, the scroll icon enables a joystick-style scroll overlay (horizontal + vertical).
- Post FX: adjust `Clarity` and `Denoise` sliders (client-side CSS filters). Set both to `0` to disable.
- Debug overlays: enable `Debug overlays` to see the calibrated rectangles over the stream.
- Scaling: `H+/H-/V+/V-` and `Reset` adjust the fullscreen fit and are remembered per-host in your browser.
- Performance presets: `Battery/Balanced/Crisp` apply MJPEG interval/quality at runtime (see `/api/config`).

## Notes

- The server prefers `d3d11grab` and falls back to `gdigrab` if unavailable.
- The web client is plain HTML/CSS/JS under `internal/web/static/` (no Node build).
- The server runs **one** `ffmpeg` pipeline at a time:
  - `WebRTC` runs `ffmpeg: start ... -f rtp rtp://127.0.0.1:<port>` (H264→RTP).
  - `MJPEG` runs `ffmpeg: preview ... -f rawvideo -` and serves `/mjpeg/desktop`.
  - Default is `MJPEG` (more reliable); switch in the UI (Session → `WebRTC`) if you want lower latency.
- For MJPEG mode, the preview capture FPS is derived from `MJPEG_INTERVAL_MS` (smaller interval = higher FPS and more CPU).
- Runtime tuning: `POST /api/config` (auth required) accepts `{ "mjpegIntervalMs": <int>, "mjpegQuality": <int> }` and applies it immediately when in MJPEG mode.
- Reset: `POST /api/config` with `{ "reset": true }` restores MJPEG values loaded from `.env` at server startup.

## License

GPL-3.0-only. See `LICENSE`.
