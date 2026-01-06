export class Calibrator {
  constructor(video, overlay, sendRect, setHint, fallback) {
    this.video = video;
    this.fallback = fallback;
    this.overlay = overlay;
    this.sendRect = sendRect;
    this.setHint = setHint;
    this.ctx = overlay.getContext("2d");
    this.active = false;
    this.debug = false;
    this.mode = "presetup";
    this.editEnabled = false;
    this.step = null;
    this.start = null;
    this.rect = null;
    this.lastRect = null;
    this.pluginRect = null;
    this.selectedStep = null;
    this.drag = null;
    this.onSelectionChange = null;
    this.rects = {
      plugin: null,
      chat: null,
      scroll: null,
    };
    this.expectedSize = null;
    this.bind();
  }

  setDebugEnabled(enabled) {
    this.debug = Boolean(enabled);
    this.render();
  }

  setEditEnabled(enabled) {
    this.editEnabled = Boolean(enabled);
    if (!this.canEdit()) {
      this.selectedStep = null;
      this.drag = null;
    }
    this.render();
  }

  setSelectionListener(listener) {
    this.onSelectionChange = typeof listener === "function" ? listener : null;
  }

  setMode(mode) {
    this.mode = mode === "run" ? "run" : "presetup";
    if (!this.canEdit()) {
      this.selectedStep = null;
      this.drag = null;
    }
    this.render();
  }

  selectStep(step) {
    const key = step === "chat" || step === "scroll" || step === "plugin" ? step : "plugin";
    const changed = this.selectedStep !== key;
    this.selectedStep = key;
    if (changed) {
      this.onSelectionChange?.(key);
    }
    this.render();
  }

  nudgeSelected(dx, dy) {
    if (!this.canEdit() || !this.selectedStep) {
      this.setHint("Enable Edit rectangles first");
      return;
    }
    const rect = this.rects[this.selectedStep];
    if (!rect) {
      this.setHint(`Missing ${this.selectedStep} rectangle`);
      return;
    }
    const next = { x: rect.x + dx, y: rect.y + dy, w: rect.w, h: rect.h };
    const clamped = this.clampRect(this.selectedStep, next);
    this.setRect(this.selectedStep, clamped);
    this.commitRect(this.selectedStep);
    this.render();
  }

  setCalibData(calibData) {
    if (!calibData || !calibData.PluginAbs) {
      this.pluginRect = null;
      this.rects = { plugin: null, chat: null, scroll: null };
      this.render();
      return;
    }

    const plugin = {
      x: calibData.PluginAbs.X || 0,
      y: calibData.PluginAbs.Y || 0,
      w: calibData.PluginAbs.W || 0,
      h: calibData.PluginAbs.H || 0,
    };
    if (plugin.w <= 0 || plugin.h <= 0) {
      this.pluginRect = null;
      this.rects = { plugin: null, chat: null, scroll: null };
      this.render();
      return;
    }

    this.pluginRect = plugin;
    this.rects.plugin = plugin;

    const chatRel = calibData.ChatRel;
    if (chatRel && (chatRel.W || 0) > 0 && (chatRel.H || 0) > 0) {
      this.rects.chat = { x: plugin.x + (chatRel.X || 0), y: plugin.y + (chatRel.Y || 0), w: chatRel.W || 0, h: chatRel.H || 0 };
    } else {
      this.rects.chat = null;
    }

    const scrollRel = calibData.ScrollRel;
    if (scrollRel && (scrollRel.W || 0) > 0 && (scrollRel.H || 0) > 0) {
      this.rects.scroll = { x: plugin.x + (scrollRel.X || 0), y: plugin.y + (scrollRel.Y || 0), w: scrollRel.W || 0, h: scrollRel.H || 0 };
    } else {
      this.rects.scroll = null;
    }

    this.render();
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
    this.drag = null;
    this.setHint(`Draw ${step} rectangle`);
    this.render();
  }

  onDown(event) {
    if (!this.active && this.canEdit()) {
      const hit = this.hitTest(event);
      if (!hit) {
        return;
      }
      this.consume(event);
      this.overlay.setPointerCapture(event.pointerId);
      const changed = this.selectedStep !== hit.step;
      this.selectedStep = hit.step;
      if (changed) {
        this.onSelectionChange?.(hit.step);
      }
      const rect = this.rects[hit.step];
      if (!rect) {
        this.drag = null;
        this.render();
        return;
      }
      const p = this.pointFromEvent(event);
      this.drag = {
        pointerId: event.pointerId,
        step: hit.step,
        handle: hit.handle,
        startPoint: p,
        startRect: { ...rect },
      };
      this.render();
      return;
    }

    if (!this.active) return;
    this.consume(event);
    this.overlay.setPointerCapture(event.pointerId);
    this.start = this.pointFromEvent(event);
    this.rect = null;
  }

  onMove(event) {
    if (!this.active && this.drag && event.pointerId === this.drag.pointerId) {
      this.consume(event);
      const current = this.pointFromEvent(event);
      const next = this.applyDrag(this.drag, current);
      this.setRect(this.drag.step, next);
      this.render();
      return;
    }

    if (!this.active || !this.start) return;
    this.consume(event);
    const current = this.pointFromEvent(event);
    this.rect = rectFromPoints(this.start, current);
    this.render();
  }

  onUp(event) {
    if (!this.active && this.drag && event.pointerId === this.drag.pointerId) {
      this.consume(event);
      if (this.overlay.hasPointerCapture(event.pointerId)) {
        this.overlay.releasePointerCapture(event.pointerId);
      }
      const step = this.drag.step;
      this.drag = null;
      this.commitRect(step);
      this.setHint("Calibration updated");
      this.render();
      return;
    }

    if (!this.active || !this.start) return;
    this.consume(event);
    if (this.overlay.hasPointerCapture(event.pointerId)) {
      this.overlay.releasePointerCapture(event.pointerId);
    }
    const end = this.pointFromEvent(event);
    const rect = rectFromPoints(this.start, end);
    this.lastRect = rect;
    this.applyRectForStep(this.step, rect);
    this.active = false;
    this.start = null;
    this.rect = null;
    this.setHint("Saved calibration");
    this.render();
  }

  onCancel(event) {
    if (!this.active && this.drag && event.pointerId === this.drag.pointerId) {
      this.consume(event);
      if (this.overlay.hasPointerCapture(event.pointerId)) {
        this.overlay.releasePointerCapture(event.pointerId);
      }
      this.drag = null;
      this.render();
      return;
    }

    if (!this.active) return;
    this.consume(event);
    if (this.overlay.hasPointerCapture(event.pointerId)) {
      this.overlay.releasePointerCapture(event.pointerId);
    }
    this.start = null;
    this.rect = null;
  }

  save() {
    if (this.canEdit() && this.selectedStep && this.rects[this.selectedStep]) {
      this.commitRect(this.selectedStep);
      this.setHint("Calibration sent");
      this.render();
      return;
    }
    if (!this.step || !this.lastRect) {
      this.setHint("Draw a rectangle first");
      return;
    }
    this.applyRectForStep(this.step, this.lastRect);
    this.setHint("Calibration sent");
    this.render();
  }

  applyRectForStep(step, rect) {
    if (!rect || !step) return;
    this.setRect(step, rect);
    const payload = this.payloadForStep(step, rect);
    if (payload) {
      this.sendRect(step, payload);
    }
  }

  commitRect(step) {
    if (!step) return;
    const rect = this.rects[step];
    if (!rect) return;
    const payload = this.payloadForStep(step, rect);
    if (payload) {
      this.sendRect(step, payload);
    }
  }

  payloadForStep(step, rect) {
    if (!rect || !step) return null;
    if (step === "plugin") {
      return { ...rect };
    }
    if (this.rects.plugin) {
      const plugin = this.rects.plugin;
      return { x: rect.x - plugin.x, y: rect.y - plugin.y, w: rect.w, h: rect.h };
    }
    return { ...rect };
  }

  setRect(step, rect) {
    if (!step || !rect) return;
    const key = step === "chat" || step === "scroll" || step === "plugin" ? step : null;
    if (!key) return;
    const next = this.clampRect(key, rect);
    if (key === "plugin") {
      const prev = this.rects.plugin;
      this.rects.plugin = next;
      this.pluginRect = next;
      if (prev) {
        const dx = next.x - prev.x;
        const dy = next.y - prev.y;
        if (dx !== 0 || dy !== 0) {
          if (this.rects.chat) {
            this.rects.chat = this.clampRect("chat", { ...this.rects.chat, x: this.rects.chat.x + dx, y: this.rects.chat.y + dy });
          }
          if (this.rects.scroll) {
            this.rects.scroll = this.clampRect("scroll", { ...this.rects.scroll, x: this.rects.scroll.x + dx, y: this.rects.scroll.y + dy });
          }
        }
      }
      if (this.rects.chat) {
        this.rects.chat = this.clampRect("chat", this.rects.chat);
      }
      if (this.rects.scroll) {
        this.rects.scroll = this.clampRect("scroll", this.rects.scroll);
      }
      return;
    }
    this.rects[key] = next;
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
    if (this.active || this.debug || this.canEdit()) {
      this.drawStored();
    }
    if (!this.active && this.canEdit()) {
      this.drawSelection();
    }
    if (!this.rect || !this.step) return;
    this.drawRect(this.rect, stepColor(this.step), true, this.step);
  }

  drawStored() {
    const showLabels = this.canEdit();
    if (this.rects.plugin) {
      this.drawRect(this.rects.plugin, stepColor("plugin"), false, "plugin", showLabels ? { label: "plugin" } : null);
    }
    if (this.rects.chat) {
      this.drawRect(this.rects.chat, stepColor("chat"), false, "chat", showLabels ? { label: "chat" } : null);
    }
    if (this.rects.scroll) {
      this.drawRect(this.rects.scroll, stepColor("scroll"), false, "scroll", showLabels ? { label: "scroll" } : null);
    }
  }

  drawRect(rect, color, dashed, step, opts) {
    const canvasRect = this.canvasRectFor(rect, step);
    if (!canvasRect) return;
    const { sx, sy, sw, sh, scale } = canvasRect;
    this.ctx.save();
    this.ctx.strokeStyle = color;
    this.ctx.lineWidth = 2 * scale;
    if (dashed) {
      this.ctx.setLineDash([6 * scale, 4 * scale]);
    } else {
      this.ctx.setLineDash([]);
    }
    this.ctx.strokeRect(sx, sy, sw, sh);
    if (opts?.label) {
      const label = String(opts.label);
      this.ctx.setLineDash([]);
      this.ctx.fillStyle = "rgba(10, 10, 10, 0.55)";
      this.ctx.strokeStyle = "rgba(255, 255, 255, 0.55)";
      this.ctx.lineWidth = 1 * scale;
      this.ctx.font = `${Math.round(11 * scale)}px ui-sans-serif, system-ui, -apple-system, Segoe UI, sans-serif`;
      const padX = 6 * scale;
      const padY = 4 * scale;
      const tw = this.ctx.measureText(label).width;
      const bx = sx + 6 * scale;
      const by = sy + 6 * scale;
      const bw = tw + padX * 2;
      const bh = 16 * scale;
      if (typeof this.ctx.roundRect === "function") {
        this.ctx.beginPath();
        this.ctx.roundRect(bx, by, bw, bh, 6 * scale);
        this.ctx.fill();
        this.ctx.stroke();
      } else {
        this.ctx.fillRect(bx, by, bw, bh);
        this.ctx.strokeRect(bx, by, bw, bh);
      }
      this.ctx.fillStyle = "rgba(255,255,255,0.92)";
      this.ctx.fillText(label, bx + padX, by + (bh - padY));
    }
    this.ctx.restore();
  }

  drawSelection() {
    const step = this.selectedStep;
    if (!step) return;
    const rect = this.rects[step];
    if (!rect) return;
    const canvasRect = this.canvasRectFor(rect, step);
    if (!canvasRect) return;
    const { sx, sy, sw, sh, scale } = canvasRect;
    this.ctx.save();
    this.ctx.setLineDash([]);
    this.ctx.strokeStyle = "rgba(255,255,255,0.92)";
    this.ctx.lineWidth = 3 * scale;
    this.ctx.strokeRect(sx, sy, sw, sh);
    this.ctx.fillStyle = "rgba(255,255,255,0.75)";
    const hs = 10 * scale;
    const handles = [
      [sx, sy],
      [sx + sw / 2, sy],
      [sx + sw, sy],
      [sx + sw, sy + sh / 2],
      [sx + sw, sy + sh],
      [sx + sw / 2, sy + sh],
      [sx, sy + sh],
      [sx, sy + sh / 2],
    ];
    for (const [hx, hy] of handles) {
      this.ctx.fillRect(hx - hs / 2, hy - hs / 2, hs, hs);
    }
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

  canvasRectFor(rect, step) {
    const scale = window.devicePixelRatio || 1;
    const bounds = this.overlay.getBoundingClientRect();
    const size = this.mediaSize(bounds);
    const width = size.width;
    const height = size.height;
    const displayRect = this.adjustRect(rect, step, width, height);
    if (!displayRect || width <= 0 || height <= 0) return null;
    const rectBounds = this.contentRect(bounds);
    const sx = (displayRect.x / width) * rectBounds.width * scale + rectBounds.x * scale;
    const sy = (displayRect.y / height) * rectBounds.height * scale + rectBounds.y * scale;
    const sw = (displayRect.w / width) * rectBounds.width * scale;
    const sh = (displayRect.h / height) * rectBounds.height * scale;
    return { sx, sy, sw, sh, scale };
  }

  contentRect(bounds) {
    const size = this.mediaSize(bounds);
    const mediaW = size.width;
    const mediaH = size.height;
    if (mediaW <= 0 || mediaH <= 0) {
      return { x: 0, y: 0, width: bounds.width, height: bounds.height };
    }
    const scale = Math.min(bounds.width / mediaW, bounds.height / mediaH);
    const width = mediaW * scale;
    const height = mediaH * scale;
    const base = { x: (bounds.width - width) / 2, y: (bounds.height - height) / 2, width, height };
    if (!document.body.classList.contains("is-fullscreen")) {
      return base;
    }
    const styles = getComputedStyle(this.overlay.parentElement);
    const sx = Number.parseFloat(styles.getPropertyValue("--fs-scale-x")) || 1;
    const sy = Number.parseFloat(styles.getPropertyValue("--fs-scale-y")) || 1;
    const scaledW = base.width * sx;
    const scaledH = base.height * sy;
    return {
      x: base.x + (base.width - scaledW) / 2,
      y: base.y + (base.height - scaledH) / 2,
      width: scaledW,
      height: scaledH,
    };
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

  canEdit() {
    return this.editEnabled && this.mode === "presetup";
  }

  hitTest(event) {
    const p = this.pointFromEvent(event);
    const bounds = this.overlay.getBoundingClientRect();
    const rectBounds = this.contentRect(bounds);
    const size = this.mediaSize(bounds);
    const mw = size.width;
    const mh = size.height;
    if (mw <= 0 || mh <= 0 || rectBounds.width <= 0 || rectBounds.height <= 0) {
      return null;
    }

    const hitPx = 16;
    const tolX = (hitPx / rectBounds.width) * mw;
    const tolY = (hitPx / rectBounds.height) * mh;

    const candidates = ["chat", "scroll", "plugin"];
    for (const step of candidates) {
      const rect = this.rects[step];
      if (!rect) continue;
      const left = rect.x;
      const top = rect.y;
      const right = rect.x + rect.w;
      const bottom = rect.y + rect.h;
      if (p.x < left - tolX || p.x > right + tolX || p.y < top - tolY || p.y > bottom + tolY) {
        continue;
      }

      const nearL = Math.abs(p.x - left) <= tolX;
      const nearR = Math.abs(p.x - right) <= tolX;
      const nearT = Math.abs(p.y - top) <= tolY;
      const nearB = Math.abs(p.y - bottom) <= tolY;
      const inside = p.x >= left && p.x <= right && p.y >= top && p.y <= bottom;

      let handle = null;
      if (nearL && nearT) handle = "nw";
      else if (nearR && nearT) handle = "ne";
      else if (nearR && nearB) handle = "se";
      else if (nearL && nearB) handle = "sw";
      else if (nearT && inside) handle = "n";
      else if (nearR && inside) handle = "e";
      else if (nearB && inside) handle = "s";
      else if (nearL && inside) handle = "w";
      else if (inside) handle = "move";

      if (!handle) continue;
      return { step, handle };
    }
    return null;
  }

  applyDrag(drag, current) {
    const rect = drag.startRect;
    const dx = current.x - drag.startPoint.x;
    const dy = current.y - drag.startPoint.y;
    const next = { ...rect };
    const handle = drag.handle || "move";
    const right = rect.x + rect.w;
    const bottom = rect.y + rect.h;
    const minSize = 12;

    switch (handle) {
      case "move":
        next.x += dx;
        next.y += dy;
        break;
      case "e":
        next.w = rect.w + dx;
        break;
      case "s":
        next.h = rect.h + dy;
        break;
      case "w":
        next.x = rect.x + dx;
        next.w = rect.w - dx;
        break;
      case "n":
        next.y = rect.y + dy;
        next.h = rect.h - dy;
        break;
      case "se":
        next.w = rect.w + dx;
        next.h = rect.h + dy;
        break;
      case "ne":
        next.y = rect.y + dy;
        next.h = rect.h - dy;
        next.w = rect.w + dx;
        break;
      case "sw":
        next.x = rect.x + dx;
        next.w = rect.w - dx;
        next.h = rect.h + dy;
        break;
      case "nw":
        next.x = rect.x + dx;
        next.w = rect.w - dx;
        next.y = rect.y + dy;
        next.h = rect.h - dy;
        break;
      default:
        break;
    }

    if (next.w < 0) {
      next.x += next.w;
      next.w = Math.abs(next.w);
    }
    if (next.h < 0) {
      next.y += next.h;
      next.h = Math.abs(next.h);
    }

    if (next.w < minSize) {
      if (handle.includes("w")) {
        next.x = right - minSize;
      }
      next.w = minSize;
    }
    if (next.h < minSize) {
      if (handle.includes("n")) {
        next.y = bottom - minSize;
      }
      next.h = minSize;
    }

    return this.clampRect(drag.step, next);
  }

  clampRect(step, rect) {
    const bounds = this.overlay.getBoundingClientRect();
    const size = this.mediaSize(bounds);
    const mw = size.width;
    const mh = size.height;
    if (mw <= 0 || mh <= 0) {
      return rect;
    }

    const minSize = 12;
    let minX = 0;
    let minY = 0;
    let maxX = mw;
    let maxY = mh;

    if ((step === "chat" || step === "scroll") && this.rects.plugin) {
      const plugin = this.rects.plugin;
      minX = plugin.x;
      minY = plugin.y;
      maxX = plugin.x + plugin.w;
      maxY = plugin.y + plugin.h;
    }

    const out = {
      x: Math.round(rect.x),
      y: Math.round(rect.y),
      w: Math.round(rect.w),
      h: Math.round(rect.h),
    };

    out.w = clamp(out.w, minSize, maxX - minX);
    out.h = clamp(out.h, minSize, maxY - minY);
    out.x = clamp(out.x, minX, maxX - out.w);
    out.y = clamp(out.y, minY, maxY - out.h);

    return out;
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
