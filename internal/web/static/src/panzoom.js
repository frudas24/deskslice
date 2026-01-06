export function bindPanZoom({ target, getEnabled, apply, onTap }) {
  const pointers = new Map();
  let panning = false;
  let lastPan = null;
  let base = null;
  let lastTapAt = 0;

  function isEnabled() {
    return Boolean(getEnabled?.());
  }

  function reset() {
    pointers.clear();
    panning = false;
    lastPan = null;
    base = null;
  }

  function onPointerDown(event) {
    if (!isEnabled()) return;
    if (event.target?.closest?.(".edge-tab, .edge-exit, .edge-icon-btn, .fs-tune, .control-panel, .typing-panel")) {
      return;
    }
    pointers.set(event.pointerId, { x: event.clientX, y: event.clientY });
    target.setPointerCapture(event.pointerId);

    if (pointers.size === 2) {
      const pts = [...pointers.values()];
      base = {
        dist: distance(pts[0], pts[1]),
        center: center(pts[0], pts[1]),
      };
      panning = false;
      lastPan = null;
    }
  }

  function onPointerMove(event) {
    if (!isEnabled()) return;
    const p = pointers.get(event.pointerId);
    if (!p) return;
    pointers.set(event.pointerId, { x: event.clientX, y: event.clientY });

    if (pointers.size >= 2) {
      const pts = [...pointers.values()].slice(0, 2);
      if (!base) {
        base = { dist: distance(pts[0], pts[1]), center: center(pts[0], pts[1]) };
      }
      const dist = distance(pts[0], pts[1]);
      const c = center(pts[0], pts[1]);
      const ratio = base.dist > 0 ? dist / base.dist : 1;
      const dx = c.x - base.center.x;
      const dy = c.y - base.center.y;
      apply?.({ type: "pinch", ratio, dx, dy, bounds: target.getBoundingClientRect() });
      event.preventDefault();
      return;
    }

    const current = pointers.get(event.pointerId);
    if (!current) return;
    if (!lastPan) {
      lastPan = { x: current.x, y: current.y };
      return;
    }
    const dx = current.x - lastPan.x;
    const dy = current.y - lastPan.y;
    if (!panning && Math.hypot(dx, dy) < 3) {
      return;
    }
    panning = true;
    lastPan = { x: current.x, y: current.y };
    apply?.({ type: "pan", dx, dy, bounds: target.getBoundingClientRect() });
    event.preventDefault();
  }

  function onPointerUp(event) {
    if (!isEnabled()) return;
    if (pointers.has(event.pointerId)) {
      pointers.delete(event.pointerId);
    }
    if (target.hasPointerCapture(event.pointerId)) {
      target.releasePointerCapture(event.pointerId);
    }
    if (pointers.size < 2) {
      base = null;
    }

    if (!panning && pointers.size === 0) {
      const now = Date.now();
      if (now-lastTapAt < 320) {
        apply?.({ type: "reset" });
        lastTapAt = 0;
      } else {
        lastTapAt = now;
        onTap?.();
      }
    }

    if (pointers.size === 0) {
      panning = false;
      lastPan = null;
    }
  }

  target.addEventListener("pointerdown", onPointerDown);
  target.addEventListener("pointermove", onPointerMove, { passive: false });
  target.addEventListener("pointerup", onPointerUp);
  target.addEventListener("pointercancel", onPointerUp);

  return {
    reset,
    destroy() {
      target.removeEventListener("pointerdown", onPointerDown);
      target.removeEventListener("pointermove", onPointerMove);
      target.removeEventListener("pointerup", onPointerUp);
      target.removeEventListener("pointercancel", onPointerUp);
      reset();
    },
  };
}

function distance(a, b) {
  return Math.hypot(a.x - b.x, a.y - b.y);
}

function center(a, b) {
  return { x: (a.x + b.x) / 2, y: (a.y + b.y) / 2 };
}

