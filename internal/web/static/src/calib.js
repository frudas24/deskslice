export class Calibrator {
  constructor(video, overlay, sendRect, setHint) {
    this.video = video;
    this.overlay = overlay;
    this.sendRect = sendRect;
    this.setHint = setHint;
    this.ctx = overlay.getContext("2d");
    this.active = false;
    this.step = null;
    this.start = null;
    this.rect = null;
    this.lastRect = null;
    this.pluginRect = null;
    this.bind();
  }

  bind() {
    this.overlay.style.pointerEvents = "auto";
    this.overlay.addEventListener("pointerdown", (event) => this.onDown(event));
    this.overlay.addEventListener("pointermove", (event) => this.onMove(event));
    this.overlay.addEventListener("pointerup", (event) => this.onUp(event));
    window.addEventListener("resize", () => this.resize());
    this.video.addEventListener("loadedmetadata", () => this.resize());
    this.resize();
  }

  startStep(step) {
    this.step = step;
    this.active = true;
    this.setHint(`Draw ${step} rectangle`);
    this.clear();
  }

  onDown(event) {
    if (!this.active) return;
    this.start = this.pointFromEvent(event);
    this.rect = null;
  }

  onMove(event) {
    if (!this.active || !this.start) return;
    const current = this.pointFromEvent(event);
    this.rect = rectFromPoints(this.start, current);
    this.draw(this.rect);
  }

  onUp(event) {
    if (!this.active || !this.start) return;
    const end = this.pointFromEvent(event);
    const rect = rectFromPoints(this.start, end);
    this.lastRect = rect;
    this.applyRect(rect);
    this.active = false;
    this.start = null;
    this.rect = null;
    this.setHint("Saved calibration");
  }

  save() {
    if (!this.step || !this.lastRect) {
      this.setHint("Draw a rectangle first");
      return;
    }
    this.applyRect(this.lastRect);
    this.setHint("Calibration sent");
  }

  applyRect(rect) {
    if (!rect) return;
    let payload = { ...rect };
    if (this.step === "plugin") {
      this.pluginRect = rect;
    } else if (this.pluginRect) {
      payload = {
        x: rect.x - this.pluginRect.x,
        y: rect.y - this.pluginRect.y,
        w: rect.w,
        h: rect.h,
      };
    }
    this.sendRect(this.step, payload);
  }

  pointFromEvent(event) {
    const bounds = this.video.getBoundingClientRect();
    const x = clamp((event.clientX - bounds.left) / bounds.width, 0, 1);
    const y = clamp((event.clientY - bounds.top) / bounds.height, 0, 1);
    const width = this.video.videoWidth || bounds.width;
    const height = this.video.videoHeight || bounds.height;
    return { x: Math.round(x * width), y: Math.round(y * height) };
  }

  draw(rect) {
    this.clear();
    if (!rect) return;
    const scale = window.devicePixelRatio || 1;
    const bounds = this.video.getBoundingClientRect();
    const width = this.video.videoWidth || bounds.width;
    const height = this.video.videoHeight || bounds.height;
    const sx = (rect.x / width) * bounds.width * scale;
    const sy = (rect.y / height) * bounds.height * scale;
    const sw = (rect.w / width) * bounds.width * scale;
    const sh = (rect.h / height) * bounds.height * scale;
    this.ctx.strokeStyle = "rgba(194, 91, 46, 0.9)";
    this.ctx.lineWidth = 2 * scale;
    this.ctx.setLineDash([6 * scale, 4 * scale]);
    this.ctx.strokeRect(sx, sy, sw, sh);
  }

  clear() {
    this.ctx.clearRect(0, 0, this.overlay.width, this.overlay.height);
  }

  resize() {
    const scale = window.devicePixelRatio || 1;
    this.overlay.width = this.overlay.clientWidth * scale;
    this.overlay.height = this.overlay.clientHeight * scale;
    this.clear();
  }
}

function rectFromPoints(a, b) {
  const x = Math.min(a.x, b.x);
  const y = Math.min(a.y, b.y);
  const w = Math.abs(a.x - b.x);
  const h = Math.abs(a.y - b.y);
  return { x, y, w, h };
}

function clamp(value, min, max) {
  return Math.min(max, Math.max(min, value));
}
