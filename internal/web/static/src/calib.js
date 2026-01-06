export class Calibrator {
  constructor(video, overlay, sendRect, setHint, fallback) {
    this.video = video;
    this.fallback = fallback;
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
    this.rects = {
      plugin: null,
      chat: null,
      scroll: null,
    };
    this.expectedSize = null;
    this.bind();
  }

  setExpectedSize(size) {
    if (!size || !size.width || !size.height) {
      this.expectedSize = null;
      return;
    }
    this.expectedSize = { width: size.width, height: size.height };
    this.resize();
  }

  bind() {
    this.overlay.style.pointerEvents = "auto";
    this.overlay.addEventListener("pointerdown", (event) => this.onDown(event));
    this.overlay.addEventListener("pointermove", (event) => this.onMove(event));
    this.overlay.addEventListener("pointerup", (event) => this.onUp(event));
    this.overlay.addEventListener("pointercancel", (event) => this.onCancel(event));
    window.addEventListener("resize", () => this.resize());
    this.video.addEventListener("loadedmetadata", () => this.resize());
    if (this.fallback) {
      this.fallback.addEventListener("load", () => this.resize());
    }
    this.resize();
  }

  startStep(step) {
    this.step = step;
    this.active = true;
    this.setHint(`Draw ${step} rectangle`);
    this.render();
  }

  onDown(event) {
    if (!this.active) return;
    this.consume(event);
    this.overlay.setPointerCapture(event.pointerId);
    this.start = this.pointFromEvent(event);
    this.rect = null;
  }

  onMove(event) {
    if (!this.active || !this.start) return;
    this.consume(event);
    const current = this.pointFromEvent(event);
    this.rect = rectFromPoints(this.start, current);
    this.render();
  }

  onUp(event) {
    if (!this.active || !this.start) return;
    this.consume(event);
    if (this.overlay.hasPointerCapture(event.pointerId)) {
      this.overlay.releasePointerCapture(event.pointerId);
    }
    const end = this.pointFromEvent(event);
    const rect = rectFromPoints(this.start, end);
    this.lastRect = rect;
    this.applyRect(rect);
    this.active = false;
    this.start = null;
    this.rect = null;
    this.setHint("Saved calibration");
    this.render();
  }

  onCancel(event) {
    if (!this.active) return;
    this.consume(event);
    if (this.overlay.hasPointerCapture(event.pointerId)) {
      this.overlay.releasePointerCapture(event.pointerId);
    }
    this.start = null;
    this.rect = null;
  }

  save() {
    if (!this.step || !this.lastRect) {
      this.setHint("Draw a rectangle first");
      return;
    }
    this.applyRect(this.lastRect);
    this.setHint("Calibration sent");
    this.render();
  }

  applyRect(rect) {
    if (!rect) return;
    let payload = { ...rect };
    if (this.step === "plugin") {
      this.pluginRect = rect;
      this.rects.plugin = rect;
    } else if (this.pluginRect) {
      if (this.step === "chat") {
        this.rects.chat = rect;
      } else if (this.step === "scroll") {
        this.rects.scroll = rect;
      }
      payload = {
        x: rect.x - this.pluginRect.x,
        y: rect.y - this.pluginRect.y,
        w: rect.w,
        h: rect.h,
      };
    } else if (this.step) {
      this.rects[this.step] = rect;
    }
    this.sendRect(this.step, payload);
  }

  pointFromEvent(event) {
    const bounds = this.overlay.getBoundingClientRect();
    const rect = this.contentRect(bounds);
    const cx = clamp(event.clientX - bounds.left - rect.x, 0, rect.width);
    const cy = clamp(event.clientY - bounds.top - rect.y, 0, rect.height);
    const nx = rect.width > 0 ? clamp(cx / rect.width, 0, 1) : 0;
    const ny = rect.height > 0 ? clamp(cy / rect.height, 0, 1) : 0;
    const size = this.mediaSize(bounds);
    const width = size.width;
    const height = size.height;
    return { x: Math.round(nx * width), y: Math.round(ny * height) };
  }

  render() {
    this.clear();
    this.drawStored();
    if (!this.rect || !this.step) return;
    this.drawRect(this.rect, stepColor(this.step), true, this.step);
  }

  drawStored() {
    if (this.rects.plugin) {
      this.drawRect(this.rects.plugin, stepColor("plugin"), false, "plugin");
    }
    if (this.rects.chat) {
      this.drawRect(this.rects.chat, stepColor("chat"), false, "chat");
    }
    if (this.rects.scroll) {
      this.drawRect(this.rects.scroll, stepColor("scroll"), false, "scroll");
    }
  }

  drawRect(rect, color, dashed, step) {
    const scale = window.devicePixelRatio || 1;
    const bounds = this.overlay.getBoundingClientRect();
    const size = this.mediaSize(bounds);
    const width = size.width;
    const height = size.height;
    const displayRect = this.adjustRect(rect, step, width, height);
    if (!displayRect) return;
    const rectBounds = this.contentRect(bounds);
    const sx = (displayRect.x / width) * rectBounds.width * scale + rectBounds.x * scale;
    const sy = (displayRect.y / height) * rectBounds.height * scale + rectBounds.y * scale;
    const sw = (displayRect.w / width) * rectBounds.width * scale;
    const sh = (displayRect.h / height) * rectBounds.height * scale;
    this.ctx.save();
    this.ctx.strokeStyle = color;
    this.ctx.lineWidth = 2 * scale;
    if (dashed) {
      this.ctx.setLineDash([6 * scale, 4 * scale]);
    } else {
      this.ctx.setLineDash([]);
    }
    this.ctx.strokeRect(sx, sy, sw, sh);
    this.ctx.restore();
  }

  clear() {
    this.ctx.clearRect(0, 0, this.overlay.width, this.overlay.height);
  }

  resize() {
    const scale = window.devicePixelRatio || 1;
    this.overlay.width = this.overlay.clientWidth * scale;
    this.overlay.height = this.overlay.clientHeight * scale;
    this.render();
  }

  consume(event) {
    event.preventDefault();
    event.stopImmediatePropagation();
  }

  mediaSize(bounds) {
    const fallbackWidth = this.fallback?.naturalWidth || 0;
    const fallbackHeight = this.fallback?.naturalHeight || 0;
    const expectedWidth = this.expectedSize?.width || 0;
    const expectedHeight = this.expectedSize?.height || 0;
    return {
      width: this.video.videoWidth || fallbackWidth || expectedWidth || bounds.width,
      height: this.video.videoHeight || fallbackHeight || expectedHeight || bounds.height,
    };
  }

  contentRect(bounds) {
    const size = this.mediaSize(bounds);
    const mediaW = size.width;
    const mediaH = size.height;
    if (mediaW <= 0 || mediaH <= 0) {
      return { x: 0, y: 0, width: bounds.width, height: bounds.height };
    }
    const fit = document.body.classList.contains("is-fullscreen") ? "cover" : "contain";
    const scale = fit === "cover"
      ? Math.max(bounds.width / mediaW, bounds.height / mediaH)
      : Math.min(bounds.width / mediaW, bounds.height / mediaH);
    const width = mediaW * scale;
    const height = mediaH * scale;
    return { x: (bounds.width - width) / 2, y: (bounds.height - height) / 2, width, height };
  }

  adjustRect(rect, step, width, height) {
    if (!rect) return null;
    const plugin = this.rects.plugin;
    if (!plugin) return rect;
    const isCropped = Math.abs(width - plugin.w) <= 2 && Math.abs(height - plugin.h) <= 2;
    if (!isCropped) return rect;
    if (step === "plugin") {
      return { x: 0, y: 0, w: width, h: height };
    }
    return {
      x: rect.x - plugin.x,
      y: rect.y - plugin.y,
      w: rect.w,
      h: rect.h,
    };
  }
}

function stepColor(step) {
  switch (step) {
    case "plugin":
      return "rgba(214, 90, 48, 0.95)";
    case "chat":
      return "rgba(45, 130, 152, 0.95)";
    case "scroll":
      return "rgba(214, 172, 60, 0.95)";
    default:
      return "rgba(194, 91, 46, 0.9)";
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
