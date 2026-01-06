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

let controlClient = null;
let webrtcClient = null;
let calibrator = null;

setStatus("offline");
if (video) {
  video.addEventListener("loadedmetadata", updatePreviewVisibility);
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
});

modePresetupBtn.addEventListener("click", () => {
  controlClient?.setMode("presetup");
});

modeRunBtn.addEventListener("click", () => {
  controlClient?.setMode("run");
});

monitorSelect.addEventListener("change", () => {
  const idx = Number.parseInt(monitorSelect.value, 10);
  if (!Number.isNaN(idx)) {
    controlClient?.setMonitor(idx);
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

  webrtcClient = new WebRTCClient(video, setStatus);
  try {
    await webrtcClient.connect(buildWsUrl("/ws/signal"));
  } catch (err) {
    hintText.textContent = "WebRTC failed to connect. Check server logs.";
  }

  if (mjpegImg) {
    mjpegImg.src = `/mjpeg/desktop?ts=${Date.now()}`;
    mjpegImg.addEventListener("error", () => {
      mjpegImg.style.display = "none";
      video.style.display = "block";
    }, { once: true });
  }
}

function applyState(state) {
  updateModeButtons(state.mode || "presetup");
  inputToggle.checked = Boolean(state.inputEnabled);
  hintText.textContent = state.mode === "run" ? "Run mode active." : "Presetup mode active.";
}

function updateModeButtons(mode) {
  const isRun = mode === "run";
  modeRunBtn.classList.toggle("active", isRun);
  modePresetupBtn.classList.toggle("active", !isRun);
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
  const x = clamp((event.clientX - bounds.left) / bounds.width, 0, 1);
  const y = clamp((event.clientY - bounds.top) / bounds.height, 0, 1);
  return { x, y };
}

function setStatus(state) {
  statusText.textContent = state;
  statusDot.style.background = state === "streaming" ? "#2a6f6d" : "#b2472f";
  updatePreviewVisibility();
}

function updatePreviewVisibility() {
  if (!mjpegImg) return;
  const hasVideo = video.videoWidth > 0 && statusText.textContent === "streaming";
  mjpegImg.style.display = hasVideo ? "none" : "block";
  video.style.display = hasVideo ? "block" : "none";
}

function buildWsUrl(path) {
  const protocol = location.protocol === "https:" ? "wss:" : "ws:";
  return `${protocol}//${location.host}${path}`;
}

function clamp(value, min, max) {
  return Math.min(max, Math.max(min, value));
}
