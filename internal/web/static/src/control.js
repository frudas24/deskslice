export class ControlClient {
  constructor(url) {
    this.url = url;
    this.ws = null;
    this.ready = false;
  }

  connect() {
    return new Promise((resolve, reject) => {
      this.ws = new WebSocket(this.url);
      this.ws.onopen = () => {
        this.ready = true;
        resolve();
      };
      this.ws.onerror = (err) => {
        reject(err);
      };
      this.ws.onclose = () => {
        this.ready = false;
      };
    });
  }

  send(message) {
    if (!this.ws || !this.ready) {
      return;
    }
    this.ws.send(JSON.stringify(message));
  }

  sendPointer(type, id, x, y) {
    this.send({ t: type, id, x, y });
  }

  setMode(mode) {
    this.send({ t: "setMode", mode });
  }

  setVideoMode(video) {
    this.send({ t: "setVideo", video });
  }

  setMonitor(idx) {
    this.send({ t: "setMonitor", idx });
  }

  restartPresetup() {
    this.send({ t: "restartPresetup" });
  }

  setInputEnabled(enabled) {
    this.send({ t: "inputEnabled", enabled });
  }

  sendType(text) {
    this.send({ t: "type", text });
  }

  sendEnter() {
    this.send({ t: "enter" });
  }

  clearChat() {
    this.send({ t: "clearChat" });
  }

  sendCalib(step, rect) {
    this.send({ t: "calibRect", step, rect });
  }

  sendWheel(x, y, wheelX, wheelY) {
    this.send({ t: "wheel", x, y, wheelX, wheelY });
  }
}

export function bindPointerEvents(target, getPoint, send) {
  let activeId = null;

  const onDown = (event) => {
    const point = getPoint(event);
    if (!point) return;
    activeId = event.pointerId;
    target.setPointerCapture(activeId);
    send("down", activeId, point.x, point.y);
  };

  const onMove = (event) => {
    if (activeId === null || event.pointerId !== activeId) {
      return;
    }
    const point = getPoint(event);
    if (!point) return;
    send("move", activeId, point.x, point.y);
  };

  const onUp = (event) => {
    if (activeId === null || event.pointerId !== activeId) {
      return;
    }
    const point = getPoint(event);
    if (point) {
      send("up", activeId, point.x, point.y);
    }
    target.releasePointerCapture(activeId);
    activeId = null;
  };

  target.addEventListener("pointerdown", onDown);
  target.addEventListener("pointermove", onMove);
  target.addEventListener("pointerup", onUp);
  target.addEventListener("pointercancel", onUp);

  return () => {
    target.removeEventListener("pointerdown", onDown);
    target.removeEventListener("pointermove", onMove);
    target.removeEventListener("pointerup", onUp);
    target.removeEventListener("pointercancel", onUp);
  };
}
