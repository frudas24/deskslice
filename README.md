# DeskSlice

Remote control for the real Codex panel inside VS Code, designed for LAN use
from a phone. The goal is low-latency streaming + input injection with manual
calibration (no OCR, no UI re-implementation).

## Status

Early scaffolding. The full spec lives in `TODO/TODO_0001.md`.

## Highlights

- WebRTC video stream (H264) of the Codex panel.
- Touch input mapping (tap, drag-scroll, typing).
- Presetup mode to select monitor and trace plugin/chat/scroll rectangles.
- Run mode with a cropped stream to the Codex panel only.
- Simple password gate via `.env`.

## Requirements

- Windows 11 (host).
- Go 1.25+.
- `ffmpeg` available in PATH.

## Quick Start (planned)

1. Set `UI_PASSWORD` in `./data/.env`.
2. `make build`
3. Run `./dist/<os>_<arch>/codex_remote`
4. Open `http://<host>:8787` on your phone.

## License

GPL-3.0-only. See `LICENSE`.
