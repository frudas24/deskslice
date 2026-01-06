import { login, getState, getMonitors } from "./api.js";
import { ControlClient } from "./control.js";
import { WebRTCClient } from "./webrtc.js";
import { Calibrator } from "./calib.js";
import { bindFullscreen } from "./fullscreen.js";
import { bindScrollPad } from "./scrollpad.js";
import { bindPanZoom } from "./panzoom.js";

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
const debugOverlaysToggle = document.getElementById("debug-overlays");
const calibHint = document.getElementById("calib-hint");
const fxClarity = document.getElementById("fx-clarity");
const fxClarityValue = document.getElementById("fx-clarity-value");
const fxDenoise = document.getElementById("fx-denoise");
const fxDenoiseValue = document.getElementById("fx-denoise-value");
const typeBox = document.getElementById("typebox");
const sendTextBtn = document.getElementById("send-text");
const sendEnterBtn = document.getElementById("send-enter");
const clearChatBtn = document.getElementById("clear-chat");
const video = document.getElementById("video");
const mjpegImg = document.getElementById("mjpeg");
const overlay = document.getElementById("overlay");
const scrollpad = document.getElementById("scrollpad");
const videoWrap = document.querySelector(".video-wrap");
const fullscreenToggle = document.getElementById("toggle-fullscreen");
const fullscreenToggleInline = document.getElementById("toggle-fullscreen-inline");
const fullscreenExit = document.getElementById("exit-fullscreen");
const leftPanelToggle = document.getElementById("toggle-left-panel");
const rightPanelToggle = document.getElementById("toggle-right-panel");
const closeLeftPanel = document.getElementById("close-left-panel");
const closeRightPanel = document.getElementById("close-right-panel");
const fullscreenBackdrop = document.getElementById("fs-backdrop");
const scaleXMinusBtn = document.getElementById("scale-x-minus");
const scaleXPlusBtn = document.getElementById("scale-x-plus");
const scaleYMinusBtn = document.getElementById("scale-y-minus");
const scaleYPlusBtn = document.getElementById("scale-y-plus");
const scaleResetBtn = document.getElementById("scale-reset");
const pointerToggleBtn = document.getElementById("toggle-pointer");
const mouseModeBtn = document.getElementById("toggle-mouse-mode");
const scrollToggleBtn = document.getElementById("toggle-scroll");

let controlClient = null;
let webrtcClient = null;
let calibrator = null;
let aspectPollTimer = null;
let lastWrapAspect = "";
let videoMode = "mjpeg";
let fullscreen = null;
let expectedMedia = null;
let cachedMonitors = null;
let currentMode = "presetup";
let currentMonitorIndex = 1;
let currentCalibData = null;
let scrollOverlay = { tickMs: 50, maxDelta: 240 };
let pointerEnabled = true;
let mouseMode = "mouse";
let scrollModeEnabled = false;
let debugOverlays = false;
let fsScaleX = 1.0;
let fsScaleY = 1.0;
const fsScaleMin = 0.25;
const fsScaleMax = 4.0;
let pzScale = 1.0;
let pzX = 0;
let pzY = 0;
let panZoom = null;
let postFX = { clarity: 0, denoise: 0 };
let bootstrapped = false;
let bootstrapping = false;

document.addEventListener("fullscreenchange", () => {
  updateWrapAspectRatio();
  calibrator?.resize();
  if (document.fullscreenElement) {
    applyUIPrefs();
    syncPointerToggle();
    syncScrollToggle();
    applySavedScaleOrReset();
  } else {
    scrollModeEnabled = false;
    syncScrollToggle();
    pointerEnabled = true;
    syncPointerToggle();
    resetPanZoom();
  }
});

let lastFullscreenClass = document.body.classList.contains("is-fullscreen");
new MutationObserver(() => {
  const current = document.body.classList.contains("is-fullscreen");
  if (current && !lastFullscreenClass) {
    applyUIPrefs();
    syncPointerToggle();
    syncScrollToggle();
    applySavedScaleOrReset();
  }
  if (!current && lastFullscreenClass) {
    scrollModeEnabled = false;
    syncScrollToggle();
    pointerEnabled = true;
    syncPointerToggle();
    resetPanZoom();
  }
  lastFullscreenClass = current;
}).observe(document.body, { attributes: true, attributeFilter: ["class"] });

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
  currentMode = "presetup";
  updateExpectedMedia();
  startAspectRatioPoll();
});

modeRunBtn.addEventListener("click", () => {
  controlClient?.setMode("run");
  currentMode = "run";
  updateExpectedMedia();
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
    currentMonitorIndex = idx;
    updateExpectedMedia();
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
clearChatBtn.addEventListener("click", () => controlClient?.clearChat());

scaleXMinusBtn?.addEventListener("click", () => adjustScale("x", -0.05));
scaleXPlusBtn?.addEventListener("click", () => adjustScale("x", 0.05));
scaleYMinusBtn?.addEventListener("click", () => adjustScale("y", -0.05));
scaleYPlusBtn?.addEventListener("click", () => adjustScale("y", 0.05));
scaleResetBtn?.addEventListener("click", () => resetScale());
pointerToggleBtn?.addEventListener("click", () => {
  pointerEnabled = !pointerEnabled;
  syncPointerToggle();
  saveUIPrefs();
  if (pointerEnabled) {
    resetPanZoom();
  }
});

mouseModeBtn?.addEventListener("click", () => {
  if (!pointerEnabled) return;
  mouseMode = mouseMode === "mouse" ? "touch" : "mouse";
  syncMouseModeToggle();
  saveUIPrefs();
});

scrollToggleBtn?.addEventListener("click", () => {
  if (!pointerEnabled) {
    return;
  }
  scrollModeEnabled = !scrollModeEnabled;
  syncScrollToggle();
  saveUIPrefs();
});

debugOverlaysToggle?.addEventListener("change", () => {
  debugOverlays = Boolean(debugOverlaysToggle.checked);
  saveDebugPrefs();
  calibrator?.setDebugEnabled?.(debugOverlays);
});

fxClarity?.addEventListener("input", () => {
  postFX.clarity = clampInt(Number.parseInt(fxClarity.value, 10) || 0, 0, 30);
  syncFXUI();
  applyPostFX();
  savePostFXPrefs();
});

fxDenoise?.addEventListener("input", () => {
  postFX.denoise = clampInt(Number.parseInt(fxDenoise.value, 10) || 0, 0, 10);
  syncFXUI();
  applyPostFX();
  savePostFXPrefs();
});

async function bootstrap() {
  if (bootstrapping || bootstrapped) {
    return;
  }
  bootstrapping = true;
  try {
    controls.style.display = "flex";
    const [state, monitors] = await Promise.all([getState(), getMonitors()]);
    cachedMonitors = monitors;
    populateMonitors(monitors, state.monitor);
    applyState(state);
    loadScalePrefs();
    loadDebugPrefs();
    loadPostFXPrefs();
    applyUIPrefs();
    syncPointerToggle();
    syncScrollToggle();
    syncFXUI();

    controlClient = new ControlClient(buildWsUrl("/ws/control"));
    await controlClient.connect();

    calibrator = new Calibrator(video, overlay, (step, rect) => {
      controlClient?.sendCalib(step, rect);
      if (step === "plugin") {
        currentCalibData = currentCalibData || {};
        currentCalibData.MonitorIndex = currentMonitorIndex;
        currentCalibData.PluginAbs = rect;
        updateExpectedMedia();
      }
    }, (text) => {
      calibHint.textContent = text;
    }, mjpegImg);
    calibrator.setDebugEnabled?.(debugOverlays);
    calibrator.setCalibData?.(currentCalibData);
    calibrator.setExpectedSize?.(expectedMedia);

    bindScrollPad({
      overlay,
      canvas: scrollpad,
      getPoint: (event) => normalizedPoint(event),
      getMetrics: () => overlayMetrics(),
      getContext: () => ({ mode: currentMode, inputEnabled: inputToggle.checked, pointerEnabled, mouseMode, scrollModeEnabled, scroll: scrollOverlay }),
      sendPointer: (type, id, x, y) => controlClient?.sendPointer(type, id, x, y),
      sendWheel: (x, y, wheelX, wheelY) => controlClient?.sendWheel(x, y, wheelX, wheelY),
      sendRelMove: (dx, dy) => controlClient?.sendRelMove(dx, dy),
      sendClick: () => controlClient?.sendClick(),
    });

    fullscreen = bindFullscreen({
      toggleButtons: [fullscreenToggle, fullscreenToggleInline],
      exitButton: fullscreenExit,
      leftToggle: leftPanelToggle,
      rightToggle: rightPanelToggle,
      closeLeft: closeLeftPanel,
      closeRight: closeRightPanel,
      backdrop: fullscreenBackdrop,
      videoSurface: videoWrap,
    });
    fullscreen.setEnabled(app.dataset.auth === "true");
    fullscreen.showControls?.();
    applySavedScaleOrReset();
    applyPanZoomVars();
    applyPostFX();

    panZoom = bindPanZoom({
      target: videoWrap,
      getEnabled: () => document.body.classList.contains("is-fullscreen") && !pointerEnabled,
      apply: (evt) => applyPanZoom(evt),
      onTap: () => fullscreen?.showControls?.(),
    });

    if (videoMode === "mjpeg") {
      startMJPEG();
      setStatus("mjpeg");
    } else {
      await startWebRTCOrFallback();
    }
    bootstrapped = true;
  } finally {
    bootstrapping = false;
  }
}

function applyState(state) {
  updateModeButtons(state.mode || "presetup");
  currentMode = state.mode || "presetup";
  currentMonitorIndex = state.monitor || 1;
  currentCalibData = state.calibData || null;
  inputToggle.checked = Boolean(state.inputEnabled);
  videoMode = state.videoMode || "mjpeg";
  scrollOverlay = { ...scrollOverlay, ...(state.scroll || {}) };
  updateVideoButtons(videoMode);
  expectedMedia = computeExpectedMedia(currentMode, currentMonitorIndex, currentCalibData, cachedMonitors);
  calibrator?.setCalibData?.(currentCalibData);
  calibrator?.setExpectedSize?.(expectedMedia);
  hintText.textContent = state.mode === "run" ? "Run mode active." : "Presetup mode active.";
}

async function tryAutoBootstrap() {
  if (app.dataset.auth === "true") {
    await bootstrap();
    return;
  }
  try {
    await getState();
  } catch (err) {
    if (err?.status === 401) {
      return;
    }
    return;
  }
  app.dataset.auth = "true";
  await bootstrap();
}

async function resumeConnections() {
  if (app.dataset.auth !== "true") {
    return;
  }
  if (videoMode === "mjpeg") {
    refreshMJPEG();
  }
  try {
    const [state, monitors] = await Promise.all([getState(), cachedMonitors ? Promise.resolve(cachedMonitors) : getMonitors()]);
    cachedMonitors = monitors;
    populateMonitors(monitors, state.monitor);
    applyState(state);

    if (!controlClient || !controlClient.ready) {
      controlClient = new ControlClient(buildWsUrl("/ws/control"));
      await controlClient.connect();
    }

    if (videoMode === "mjpeg") {
      refreshMJPEG();
      setStatus("mjpeg");
      return;
    }
    await startWebRTCOrFallback();
  } catch (err) {
    if (err?.status === 401) {
      app.dataset.auth = "false";
      bootstrapped = false;
      stopMJPEG();
    }
  }
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
    applyPostFX();
    updateWrapAspectRatio();
    return;
  }
  mjpegImg.style.display = "none";
  video.style.display = "block";
  applyPostFX();
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
  applyPostFX();
  startAspectRatioPoll();
}

function refreshMJPEG() {
  if (!mjpegImg || videoMode !== "mjpeg") return;
  mjpegImg.style.display = "block";
  mjpegImg.src = `/mjpeg/desktop?ts=${Date.now()}`;
  applyPostFX();
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
  const base = containRect(bounds, mediaW, mediaH);
  if (!document.body.classList.contains("is-fullscreen")) {
    return base;
  }
  const scaledW = base.width * fsScaleX;
  const scaledH = base.height * fsScaleY;
  return {
    x: base.x + (base.width - scaledW) / 2,
    y: base.y + (base.height - scaledH) / 2,
    width: scaledW,
    height: scaledH,
  };
}

function containRect(bounds, mediaW, mediaH) {
  const scale = Math.min(bounds.width / mediaW, bounds.height / mediaH);
  const width = mediaW * scale;
  const height = mediaH * scale;
  return { x: (bounds.width - width) / 2, y: (bounds.height - height) / 2, width, height };
}

function adjustScale(axis, delta) {
  const step = Number(delta) || 0;
  if (!videoWrap || !document.body.classList.contains("is-fullscreen")) return;
  if (axis === "x") {
    fsScaleX = clamp(Math.round((fsScaleX + step) * 20) / 20, fsScaleMin, fsScaleMax);
  } else {
    fsScaleY = clamp(Math.round((fsScaleY + step) * 20) / 20, fsScaleMin, fsScaleMax);
  }
  applyScale();
  saveScalePrefs();
}

function resetScale() {
  if (!videoWrap) return;
  const bounds = videoWrap.getBoundingClientRect();
  const size = mediaSize(bounds);
  if (!size.width || !size.height) return;
  const base = containRect(bounds, size.width, size.height);
  fsScaleX = base.width > 0 ? clamp(bounds.width / base.width, fsScaleMin, fsScaleMax) : 1.0;
  fsScaleY = base.height > 0 ? clamp(bounds.height / base.height, fsScaleMin, fsScaleMax) : 1.0;
  fsScaleX = Math.round(fsScaleX * 20) / 20;
  fsScaleY = Math.round(fsScaleY * 20) / 20;
  applyScale();
  saveScalePrefs();
}

function applyScale() {
  videoWrap.style.setProperty("--fs-scale-x", String(fsScaleX));
  videoWrap.style.setProperty("--fs-scale-y", String(fsScaleY));
  calibrator?.resize();
}

function applySavedScaleOrReset() {
  if (!videoWrap) return;
  if (!loadScalePrefs()) {
    resetScale();
    return;
  }
  applyScale();
}

function scaleStorageKey() {
  return `deskslice:fsScale:${location.host}`;
}

function loadScalePrefs() {
  try {
    const raw = window.localStorage.getItem(scaleStorageKey());
    if (!raw) return false;
    const parsed = JSON.parse(raw);
    const sx = Number(parsed?.x);
    const sy = Number(parsed?.y);
    if (!Number.isFinite(sx) || !Number.isFinite(sy)) return false;
    fsScaleX = clamp(sx, fsScaleMin, fsScaleMax);
    fsScaleY = clamp(sy, fsScaleMin, fsScaleMax);
    return true;
  } catch (_) {
    return false;
  }
}

function saveScalePrefs() {
  try {
    window.localStorage.setItem(scaleStorageKey(), JSON.stringify({ x: fsScaleX, y: fsScaleY }));
  } catch (_) {
    // ignore
  }
}

function syncPointerToggle() {
  if (!pointerToggleBtn) return;
  pointerToggleBtn.classList.toggle("is-disabled", !pointerEnabled);
  document.body.classList.toggle("pointer-locked", document.body.classList.contains("is-fullscreen") && !pointerEnabled);
  if (!pointerEnabled) {
    scrollModeEnabled = false;
    syncScrollToggle();
  }
  syncMouseModeToggle();
}

function syncMouseModeToggle() {
  if (!mouseModeBtn) return;
  mouseModeBtn.classList.toggle("is-mouse", pointerEnabled && mouseMode === "mouse");
  mouseModeBtn.classList.toggle("is-touch", !pointerEnabled || mouseMode !== "mouse");
}

function uiPrefsKey() {
  return `deskslice:uiPrefs:${location.host}`;
}

function loadUIPrefs() {
  try {
    const raw = window.localStorage.getItem(uiPrefsKey());
    if (!raw) return null;
    const parsed = JSON.parse(raw);
    const mm = parsed?.mouseMode === "touch" ? "touch" : "mouse";
    const pe = parsed?.pointerEnabled !== undefined ? Boolean(parsed.pointerEnabled) : true;
    const sm = Boolean(parsed?.scrollModeEnabled);
    return { mouseMode: mm, pointerEnabled: pe, scrollModeEnabled: sm };
  } catch (_) {
    return null;
  }
}

function applyUIPrefs() {
  const prefs = loadUIPrefs();
  if (!prefs) return;
  mouseMode = prefs.mouseMode;
  if (document.body.classList.contains("is-fullscreen")) {
    pointerEnabled = prefs.pointerEnabled;
    scrollModeEnabled = prefs.scrollModeEnabled && prefs.pointerEnabled;
  }
}

function saveUIPrefs() {
  try {
    const prefs = { mouseMode, pointerEnabled, scrollModeEnabled: scrollModeEnabled && pointerEnabled };
    window.localStorage.setItem(uiPrefsKey(), JSON.stringify(prefs));
  } catch (_) {
    // ignore
  }
}

function resetPanZoom() {
  pzScale = 1.0;
  pzX = 0;
  pzY = 0;
  applyPanZoomVars();
  panZoom?.reset?.();
}

function applyPanZoomVars() {
  if (!videoWrap) return;
  videoWrap.style.setProperty("--pz-scale", String(pzScale));
  videoWrap.style.setProperty("--pz-x", `${Math.round(pzX)}px`);
  videoWrap.style.setProperty("--pz-y", `${Math.round(pzY)}px`);
}

function applyPanZoom(evt) {
  if (!evt || !videoWrap) return;
  if (evt.type === "reset") {
    resetPanZoom();
    return;
  }

  const bounds = evt.bounds || videoWrap.getBoundingClientRect();
  if (!bounds || bounds.width <= 0 || bounds.height <= 0) return;

  if (evt.type === "pinch") {
    const ratio = Number(evt.ratio) || 1;
    const nextScale = clamp(pzScale*ratio, 1.0, 4.0);
    const scaleChange = pzScale > 0 ? nextScale / pzScale : 1;
    pzScale = nextScale;
    pzX += Number(evt.dx || 0);
    pzY += Number(evt.dy || 0);
    if (scaleChange !== 1) {
      pzX *= scaleChange;
      pzY *= scaleChange;
    }
  } else if (evt.type === "pan") {
    if (pzScale <= 1.01) return;
    pzX += Number(evt.dx || 0);
    pzY += Number(evt.dy || 0);
  }

  const maxX = (pzScale - 1) * bounds.width * 0.5;
  const maxY = (pzScale - 1) * bounds.height * 0.5;
  pzX = clamp(pzX, -maxX, maxX);
  pzY = clamp(pzY, -maxY, maxY);
  applyPanZoomVars();
}

function syncScrollToggle() {
  if (!scrollToggleBtn) return;
  scrollToggleBtn.classList.toggle("is-disabled", !scrollModeEnabled);
  if (!pointerEnabled) {
    scrollToggleBtn.classList.add("is-disabled");
  }
}

function overlayMetrics() {
  const bounds = overlay.getBoundingClientRect();
  const rect = contentRect(bounds);
  const size = mediaSize(bounds);
  return {
    rectWidth: rect.width,
    rectHeight: rect.height,
    mediaWidth: size.width,
    mediaHeight: size.height,
  };
}

function debugStorageKey() {
  return `deskslice:debugOverlays:${location.host}`;
}

function loadDebugPrefs() {
  try {
    const raw = window.localStorage.getItem(debugStorageKey());
    debugOverlays = raw === "1";
  } catch (_) {
    debugOverlays = false;
  }
  if (debugOverlaysToggle) {
    debugOverlaysToggle.checked = debugOverlays;
  }
}

function saveDebugPrefs() {
  try {
    window.localStorage.setItem(debugStorageKey(), debugOverlays ? "1" : "0");
  } catch (_) {
    // ignore
  }
}

function postFXStorageKey() {
  return `deskslice:postFX:${location.host}`;
}

function loadPostFXPrefs() {
  try {
    const raw = window.localStorage.getItem(postFXStorageKey());
    if (!raw) return;
    const parsed = JSON.parse(raw);
    const clarity = clampInt(Number(parsed?.clarity) || 0, 0, 30);
    const denoise = clampInt(Number(parsed?.denoise) || 0, 0, 10);
    postFX = { clarity, denoise };
  } catch (_) {
    postFX = { clarity: 0, denoise: 0 };
  }
}

function savePostFXPrefs() {
  try {
    window.localStorage.setItem(postFXStorageKey(), JSON.stringify(postFX));
  } catch (_) {
    // ignore
  }
}

function syncFXUI() {
  if (fxClarity) fxClarity.value = String(postFX.clarity);
  if (fxClarityValue) fxClarityValue.textContent = String(postFX.clarity);
  if (fxDenoise) fxDenoise.value = String(postFX.denoise);
  if (fxDenoiseValue) fxDenoiseValue.textContent = String(postFX.denoise);
}

function applyPostFX() {
  const clarity = clampInt(Number(postFX.clarity) || 0, 0, 30);
  const denoise = clampInt(Number(postFX.denoise) || 0, 0, 10);

  let filter = "none";
  if (clarity > 0 || denoise > 0) {
    const blurPx = denoise * 0.12;
    const contrast = 1 + clarity * 0.015;
    const brightness = 1 + clarity * 0.003;
    const saturate = 1 + clarity * 0.006;
    const parts = [];
    if (blurPx > 0) parts.push(`blur(${blurPx.toFixed(2)}px)`);
    parts.push(`contrast(${contrast.toFixed(3)})`);
    parts.push(`brightness(${brightness.toFixed(3)})`);
    parts.push(`saturate(${saturate.toFixed(3)})`);
    filter = parts.join(" ");
  }

  if (video) video.style.filter = filter;
  if (mjpegImg) mjpegImg.style.filter = filter;
}

function mediaSize(bounds) {
  const mjpegW = mjpegImg?.naturalWidth || 0;
  const mjpegH = mjpegImg?.naturalHeight || 0;
  const expectedW = expectedMedia?.width || 0;
  const expectedH = expectedMedia?.height || 0;
  return {
    width: video.videoWidth || mjpegW || expectedW || bounds.width,
    height: video.videoHeight || mjpegH || expectedH || bounds.height,
  };
}

function computeExpectedMedia(mode, monitorIndex, calib, monitors) {
  if (!monitors || !Array.isArray(monitors)) {
    return null;
  }
  const monitor = monitors.find((m) => (m.Index ?? m.index) === monitorIndex);
  if (!monitor) return null;
  if (mode === "run" && calib?.PluginAbs?.W && calib?.PluginAbs?.H) {
    return { width: calib.PluginAbs.W, height: calib.PluginAbs.H };
  }
  const w = monitor.W ?? monitor.w;
  const h = monitor.H ?? monitor.h;
  if (!w || !h) return null;
  return { width: w, height: h };
}

function updateExpectedMedia() {
  expectedMedia = computeExpectedMedia(currentMode, currentMonitorIndex, currentCalibData, cachedMonitors);
  calibrator?.setExpectedSize?.(expectedMedia);
  updateWrapAspectRatio();
}

function buildWsUrl(path) {
  const protocol = location.protocol === "https:" ? "wss:" : "ws:";
  return `${protocol}//${location.host}${path}`;
}

function clamp(value, min, max) {
  return Math.min(max, Math.max(min, value));
}

function clampInt(value, min, max) {
  return Math.min(max, Math.max(min, Math.trunc(value)));
}

document.addEventListener("visibilitychange", () => {
  if (document.visibilityState === "visible") {
    void resumeConnections();
  }
});

window.addEventListener("pageshow", () => {
  void resumeConnections();
});

void tryAutoBootstrap();
