import { login, getState, getMonitors } from "./api.js";
import { ControlClient, bindPointerEvents } from "./control.js";
import { WebRTCClient } from "./webrtc.js";
import { Calibrator } from "./calib.js";

const app = document.querySelector(".app");
const statusDot = document.getElementById("status-dot");
const statusText = document.getElementById("status-text");
const hintText = document.getElementById("hint-text");
const loginForm = document.getElementById("login-form");
const passwordInput = document.getElementById("password");
const loginHint = document.getElementById("login-hint");
const controls = document.getElementById("controls");
const modePresetupBtn = document.getElementById("mode-presetup");
const modeRunBtn = document.getElementById("mode-run");
const videoWebRTCBtn = document.getElementById("video-webrtc");
const videoMJPEGBtn = document.getElementById("video-mjpeg");
const monitorSelect = document.getElementById("monitor");
const restartBtn = document.getElementById("restart-presetup");
const inputToggle = document.getElementById("input-enabled");
const setPluginBtn = document.getElementById("set-plugin");
const setChatBtn = document.getElementById("set-chat");
const setScrollBtn = document.getElementById("set-scroll");
const saveCalibBtn = document.getElementById("save-calib");
const calibHint = document.getElementById("calib-hint");
const typeBox = document.getElementById("typebox");
const sendTextBtn = document.getElementById("send-text");
const sendEnterBtn = document.getElementById("send-enter");
const video = document.getElementById("video");
const mjpegImg = document.getElementById("mjpeg");
const overlay = document.getElementById("overlay");
const videoWrap = document.querySelector(".video-wrap");

let controlClient = null;
let webrtcClient = null;
let calibrator = null;
let aspectPollTimer = null;
let lastWrapAspect = "";
let videoMode = "mjpeg";

setStatus("offline");
if (video) {
  video.addEventListener("loadedmetadata", () => {
    updatePreviewVisibility();
    updateWrapAspectRatio();
  });
}

loginForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  loginHint.textContent = "";
  try {
    await login(passwordInput.value.trim());
    app.dataset.auth = "true";
    await bootstrap();
  } catch (err) {
    loginHint.textContent = "Login failed. Check password.";
  }
});

restartBtn.addEventListener("click", () => {
  controlClient?.restartPresetup();
  startAspectRatioPoll();
});

modePresetupBtn.addEventListener("click", () => {
  controlClient?.setMode("presetup");
  startAspectRatioPoll();
});

modeRunBtn.addEventListener("click", () => {
  controlClient?.setMode("run");
  startAspectRatioPoll();
});

videoWebRTCBtn.addEventListener("click", () => {
  setVideoMode("webrtc");
});

videoMJPEGBtn.addEventListener("click", () => {
  setVideoMode("mjpeg");
});

monitorSelect.addEventListener("change", () => {
  const idx = Number.parseInt(monitorSelect.value, 10);
  if (!Number.isNaN(idx)) {
    controlClient?.setMonitor(idx);
    startAspectRatioPoll();
  }
});

inputToggle.addEventListener("change", () => {
  controlClient?.setInputEnabled(inputToggle.checked);
});

setPluginBtn.addEventListener("click", () => calibrator?.startStep("plugin"));
setChatBtn.addEventListener("click", () => calibrator?.startStep("chat"));
setScrollBtn.addEventListener("click", () => calibrator?.startStep("scroll"));
saveCalibBtn.addEventListener("click", () => calibrator?.save());

sendTextBtn.addEventListener("click", () => {
  const text = typeBox.value.trim();
  if (!text) return;
  controlClient?.sendType(text);
  typeBox.value = "";
});

sendEnterBtn.addEventListener("click", () => controlClient?.sendEnter());

async function bootstrap() {
  controls.style.display = "flex";
  const [state, monitors] = await Promise.all([getState(), getMonitors()]);
  populateMonitors(monitors, state.monitor);
  applyState(state);

  controlClient = new ControlClient(buildWsUrl("/ws/control"));
  await controlClient.connect();

  calibrator = new Calibrator(video, overlay, (step, rect) => {
    controlClient?.sendCalib(step, rect);
  }, (text) => {
    calibHint.textContent = text;
  }, mjpegImg);

  bindPointerEvents(overlay, (event) => normalizedPoint(event), (type, id, x, y) => {
    controlClient?.sendPointer(type, id, x, y);
  });

  if (videoMode === "mjpeg") {
    startMJPEG();
    setStatus("mjpeg");
  } else {
    await startWebRTCOrFallback();
  }
}

function applyState(state) {
  updateModeButtons(state.mode || "presetup");
  inputToggle.checked = Boolean(state.inputEnabled);
  videoMode = state.videoMode || "mjpeg";
  updateVideoButtons(videoMode);
  hintText.textContent = state.mode === "run" ? "Run mode active." : "Presetup mode active.";
}

function updateModeButtons(mode) {
  const isRun = mode === "run";
  modeRunBtn.classList.toggle("active", isRun);
  modePresetupBtn.classList.toggle("active", !isRun);
}

function updateVideoButtons(mode) {
  const useMJPEG = mode === "mjpeg";
  videoMJPEGBtn.classList.toggle("active", useMJPEG);
  videoWebRTCBtn.classList.toggle("active", !useMJPEG);
}

function populateMonitors(monitors, activeIndex) {
  monitorSelect.innerHTML = "";
  monitors.forEach((m) => {
    const opt = document.createElement("option");
    opt.value = String(m.Index ?? m.index);
    const w = m.W ?? m.w;
    const h = m.H ?? m.h;
    const size = w && h ? ` ${w}x${h}` : "";
    const primary = m.Primary ? " (primary)" : "";
    opt.textContent = `Monitor ${m.Index ?? m.index}${size}${primary}`;
    monitorSelect.appendChild(opt);
  });
  if (activeIndex) {
    monitorSelect.value = String(activeIndex);
  }
}

function normalizedPoint(event) {
  const bounds = overlay.getBoundingClientRect();
  const rect = contentRect(bounds);
  const cx = clamp(event.clientX - bounds.left - rect.x, 0, rect.width);
  const cy = clamp(event.clientY - bounds.top - rect.y, 0, rect.height);
  const x = rect.width > 0 ? clamp(cx / rect.width, 0, 1) : 0;
  const y = rect.height > 0 ? clamp(cy / rect.height, 0, 1) : 0;
  return { x, y };
}

function setStatus(state) {
  statusText.textContent = state;
  statusDot.style.background = state === "streaming" || state === "mjpeg" ? "#2a6f6d" : "#b2472f";
  updatePreviewVisibility();
}

function updatePreviewVisibility() {
  if (!mjpegImg) return;
  if (videoMode === "mjpeg") {
    mjpegImg.style.display = "block";
    video.style.display = "none";
    updateWrapAspectRatio();
    return;
  }
  mjpegImg.style.display = "none";
  video.style.display = "block";
  updateWrapAspectRatio();
}

function updateWrapAspectRatio() {
  if (!videoWrap) return;
  const bounds = videoWrap.getBoundingClientRect();
  const size = mediaSize(bounds);
  if (!size.width || !size.height) return;
  const aspect = `${size.width} / ${size.height}`;
  if (aspect === lastWrapAspect) return;
  videoWrap.style.aspectRatio = aspect;
  lastWrapAspect = aspect;
  calibrator?.resize();
}

function startAspectRatioPoll() {
  if (aspectPollTimer) return;
  let tries = 0;
  aspectPollTimer = window.setInterval(() => {
    tries += 1;
    updateWrapAspectRatio();
    const videoReady = video.videoWidth > 0 && video.videoHeight > 0;
    const mjpegReady = (mjpegImg?.naturalWidth || 0) > 0 && (mjpegImg?.naturalHeight || 0) > 0;
    if (videoReady || mjpegReady || tries >= 40) {
      window.clearInterval(aspectPollTimer);
      aspectPollTimer = null;
    }
  }, 250);
}

async function setVideoMode(next) {
  if (!next || next === videoMode) {
    updatePreviewVisibility();
    return;
  }
  videoMode = next;
  updateVideoButtons(videoMode);
  controlClient?.setVideoMode(videoMode);

  if (videoMode === "mjpeg") {
    webrtcClient?.close();
    webrtcClient = null;
    startMJPEG();
    setStatus("mjpeg");
    return;
  }

  stopMJPEG();
  await startWebRTCOrFallback();
}

function startMJPEG() {
  if (!mjpegImg) return;
  mjpegImg.style.display = "block";
  mjpegImg.src = `/mjpeg/desktop?ts=${Date.now()}`;
  mjpegImg.addEventListener("error", () => {
    mjpegImg.style.display = "none";
  }, { once: true });
  startAspectRatioPoll();
}

function stopMJPEG() {
  if (!mjpegImg) return;
  mjpegImg.src = "";
  mjpegImg.style.display = "none";
}

async function startWebRTCOrFallback() {
  if (!video) return;
  webrtcClient?.close();
  webrtcClient = new WebRTCClient(video, setStatus);
  try {
    await webrtcClient.connect(buildWsUrl("/ws/signal"));
    hintText.textContent = "WebRTC connecting...";
    window.setTimeout(() => {
      if (videoMode !== "webrtc") return;
      if (statusText.textContent === "streaming") return;
      setVideoMode("mjpeg");
    }, 5000);
  } catch (err) {
    hintText.textContent = "WebRTC failed. Switching to MJPEG.";
    setVideoMode("mjpeg");
  }
}

function contentRect(bounds) {
  const size = mediaSize(bounds);
  const mediaW = size.width;
  const mediaH = size.height;
  if (mediaW <= 0 || mediaH <= 0) {
    return { x: 0, y: 0, width: bounds.width, height: bounds.height };
  }
  const containerAR = bounds.width / bounds.height;
  const mediaAR = mediaW / mediaH;
  if (mediaAR > containerAR) {
    const width = bounds.width;
    const height = width / mediaAR;
    return { x: 0, y: (bounds.height - height) / 2, width, height };
  }
  const height = bounds.height;
  const width = height * mediaAR;
  return { x: (bounds.width - width) / 2, y: 0, width, height };
}

function mediaSize(bounds) {
  const mjpegW = mjpegImg?.naturalWidth || 0;
  const mjpegH = mjpegImg?.naturalHeight || 0;
  return {
    width: video.videoWidth || mjpegW || bounds.width,
    height: video.videoHeight || mjpegH || bounds.height,
  };
}

function buildWsUrl(path) {
  const protocol = location.protocol === "https:" ? "wss:" : "ws:";
  return `${protocol}//${location.host}${path}`;
}

function clamp(value, min, max) {
  return Math.min(max, Math.max(min, value));
}
