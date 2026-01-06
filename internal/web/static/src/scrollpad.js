export function bindScrollPad({ overlay, canvas, getPoint, getMetrics, getContext, sendPointer, sendWheel, sendRelMove, sendClick }) {
  let activeId = null;
  let startNorm = null;
  let lastNorm = null;
  let startRel = null;
  let lastRel = null;
  let tickTimer = null;
  let dragging = false;
  let scrollActive = false;
  let radius = 90;
  let mouseMoved = false;
  let lastClient = null;

  const stopTick = () => {
    if (tickTimer) {
      window.clearInterval(tickTimer);
      tickTimer = null;
    }
  };

  const hide = () => {
    if (!canvas) return;
    canvas.hidden = true;
    const ctx = canvas.getContext("2d");
    if (ctx) {
      ctx.clearRect(0, 0, canvas.width, canvas.height);
    }
  };

  const resizeCanvas = () => {
    if (!canvas) return;
    const scale = window.devicePixelRatio || 1;
    canvas.width = Math.max(1, Math.round(canvas.clientWidth * scale));
    canvas.height = Math.max(1, Math.round(canvas.clientHeight * scale));
  };

  const render = () => {
    if (!canvas || !scrollActive || !startRel || !lastRel) return;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;
    const scale = window.devicePixelRatio || 1;
    ctx.clearRect(0, 0, canvas.width, canvas.height);

    const cx = startRel.x * scale;
    const cy = startRel.y * scale;
    const px = lastRel.x * scale;
    const py = lastRel.y * scale;

    ctx.save();
    ctx.lineWidth = 2 * scale;
    ctx.strokeStyle = "rgba(255,255,255,0.32)";
    ctx.fillStyle = "rgba(20,20,20,0.25)";
    ctx.beginPath();
    ctx.arc(cx, cy, radius * scale, 0, Math.PI * 2);
    ctx.fill();
    ctx.stroke();

    ctx.strokeStyle = "rgba(214, 172, 60, 0.9)";
    ctx.beginPath();
    ctx.moveTo(cx, cy);
    ctx.lineTo(px, py);
    ctx.stroke();

    ctx.fillStyle = "rgba(214, 172, 60, 0.9)";
    ctx.beginPath();
    ctx.arc(px, py, 10 * scale, 0, Math.PI * 2);
    ctx.fill();
    ctx.restore();
  };

  const startScroll = (ctx) => {
    if (scrollActive || !startNorm || !startRel) {
      return;
    }
    scrollActive = true;
    dragging = false;
    if (canvas) {
      canvas.hidden = false;
      resizeCanvas();
    }
    render();

    const scroll = ctx.scroll || {};
    const tickMs = Number(scroll.tickMs) > 0 ? Number(scroll.tickMs) : 50;
    const maxDelta = Number(scroll.maxDelta) > 0 ? Number(scroll.maxDelta) : 240;

    tickTimer = window.setInterval(() => {
      if (!scrollActive || !lastRel || !startRel || !startNorm) return;
      const dx = lastRel.x - startRel.x;
      const dy = lastRel.y - startRel.y;
      const dist = Math.hypot(dx, dy);
      if (dist < 6) return;

      const nx = clamp(dx / radius, -1, 1);
      const ny = clamp(dy / radius, -1, 1);
      const strength = clamp(dist / radius, 0, 1);
      const eased = Math.pow(strength, 1.2);

      const wheelX = Math.round(nx * eased * maxDelta);
      const wheelY = Math.round(-ny * eased * maxDelta);
      if (wheelX === 0 && wheelY === 0) return;
      sendWheel?.(startNorm.x, startNorm.y, wheelX, wheelY);
    }, tickMs);
  };

  const endInteraction = (event) => {
    if (activeId === null || event.pointerId !== activeId) {
      return;
    }

    const ctx = getContext?.() || {};
    const point = getPoint(event);
    lastNorm = point || lastNorm;

    if (scrollActive) {
      stopTick();
      scrollActive = false;
      hide();
    } else if (ctx.mode === "run" && ctx.inputEnabled && ctx.mouseMode === "mouse") {
      if (!mouseMoved) {
        sendClick?.();
      }
    } else if (!dragging && startNorm) {
      sendPointer?.("down", activeId, startNorm.x, startNorm.y);
      if (point) {
        sendPointer?.("up", activeId, point.x, point.y);
      }
    } else if (dragging && point) {
      sendPointer?.("up", activeId, point.x, point.y);
    }

    if (overlay.hasPointerCapture(activeId)) {
      overlay.releasePointerCapture(activeId);
    }
    activeId = null;
    startNorm = null;
    lastNorm = null;
    startRel = null;
    lastRel = null;
    dragging = false;
    mouseMoved = false;
    lastClient = null;
  };

  const onDown = (event) => {
    const point = getPoint(event);
    if (!point) return;
    const ctx = getContext?.() || {};
    if (ctx.pointerEnabled === false) {
      return;
    }
    activeId = event.pointerId;
    startNorm = point;
    lastNorm = point;
    const bounds = overlay.getBoundingClientRect();
    startRel = { x: event.clientX - bounds.left, y: event.clientY - bounds.top };
    lastRel = startRel;
    dragging = false;
    scrollActive = false;
    mouseMoved = false;
    lastClient = { x: event.clientX, y: event.clientY };

    overlay.setPointerCapture(activeId);

    if (ctx.mode === "run" && ctx.inputEnabled && ctx.scrollModeEnabled) {
      startScroll(ctx);
      return;
    }

    if (ctx.mode === "run" && ctx.inputEnabled && ctx.mouseMode === "mouse") {
      return;
    }

    sendPointer?.("down", activeId, point.x, point.y);
    dragging = true;
  };

  const onMove = (event) => {
    if (activeId === null || event.pointerId !== activeId) {
      return;
    }
    const point = getPoint(event);
    if (!point) return;
    lastNorm = point;

    const bounds = overlay.getBoundingClientRect();
    lastRel = { x: event.clientX - bounds.left, y: event.clientY - bounds.top };

    if (scrollActive) {
      render();
      return;
    }

    const ctx = getContext?.() || {};
    if (ctx.mode === "run" && ctx.inputEnabled && ctx.mouseMode === "mouse") {
      const prev = lastClient || { x: event.clientX, y: event.clientY };
      const dxPx = event.clientX - prev.x;
      const dyPx = event.clientY - prev.y;
      lastClient = { x: event.clientX, y: event.clientY };

      const metrics = getMetrics?.();
      const rw = Number(metrics?.rectWidth) || 0;
      const rh = Number(metrics?.rectHeight) || 0;
      const mw = Number(metrics?.mediaWidth) || 0;
      const mh = Number(metrics?.mediaHeight) || 0;
      if (rw > 0 && rh > 0 && mw > 0 && mh > 0) {
        const dx = Math.round((dxPx / rw) * mw);
        const dy = Math.round((dyPx / rh) * mh);
        if (dx !== 0 || dy !== 0) {
          sendRelMove?.(dx, dy);
          mouseMoved = true;
        }
      }
      return;
    }

    const moved = startRel ? Math.hypot(lastRel.x - startRel.x, lastRel.y - startRel.y) : 0;
    if (!dragging && moved > 10) {
      if (startNorm) {
        sendPointer?.("down", activeId, startNorm.x, startNorm.y);
      }
      dragging = true;
    }
    if (dragging) {
      sendPointer?.("move", activeId, point.x, point.y);
    }
  };

  const onUp = (event) => endInteraction(event);
  const onCancel = (event) => endInteraction(event);

  overlay.addEventListener("pointerdown", onDown);
  overlay.addEventListener("pointermove", onMove);
  overlay.addEventListener("pointerup", onUp);
  overlay.addEventListener("pointercancel", onCancel);

  window.addEventListener("resize", () => {
    if (scrollActive) {
      resizeCanvas();
      render();
    }
  });

  return () => {
    overlay.removeEventListener("pointerdown", onDown);
    overlay.removeEventListener("pointermove", onMove);
    overlay.removeEventListener("pointerup", onUp);
    overlay.removeEventListener("pointercancel", onCancel);
    stopTick();
    hide();
  };
}

function clamp(value, min, max) {
  return Math.min(max, Math.max(min, value));
}
