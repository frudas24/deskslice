export function bindFullscreen({
  toggleButtons,
  toggleButton,
  exitButton,
  leftToggle,
  rightToggle,
  closeLeft,
  closeRight,
  backdrop,
  videoSurface,
}) {
  const toggles = Array.isArray(toggleButtons) ? toggleButtons.filter(Boolean) : [];
  if (toggleButton) {
    toggles.push(toggleButton);
  }
  if (toggles.length === 0 || !exitButton || !leftToggle || !rightToggle || !backdrop) {
    return {
      setEnabled: () => {},
    };
  }

  let enabled = true;

  toggles.forEach((btn) => {
    btn.addEventListener("click", async () => {
      if (!enabled) return;
      if (document.fullscreenElement) {
        await safeExitFullscreen();
        return;
      }
      await safeEnterFullscreen();
    });
  });

  exitButton.addEventListener("click", async () => {
    if (!enabled) return;
    await safeExitFullscreen();
  });

  leftToggle.addEventListener("click", () => {
    if (!enabled) return;
    document.body.classList.toggle("drawer-left-open");
    document.body.classList.remove("drawer-right-open");
    syncBackdrop();
    showControls();
  });

  rightToggle.addEventListener("click", () => {
    if (!enabled) return;
    document.body.classList.toggle("drawer-right-open");
    document.body.classList.remove("drawer-left-open");
    syncBackdrop();
    showControls();
  });

  closeLeft?.addEventListener("click", () => {
    document.body.classList.remove("drawer-left-open");
    syncBackdrop();
  });

  closeRight?.addEventListener("click", () => {
    document.body.classList.remove("drawer-right-open");
    syncBackdrop();
  });

  backdrop.addEventListener("click", () => {
    document.body.classList.remove("drawer-left-open");
    document.body.classList.remove("drawer-right-open");
    syncBackdrop();
  });

  document.addEventListener("fullscreenchange", () => {
    const isFullscreen = Boolean(document.fullscreenElement);
    document.body.classList.toggle("is-fullscreen", isFullscreen);
    document.body.classList.remove("drawer-left-open");
    document.body.classList.remove("drawer-right-open");
    syncBackdrop();
    showControls();
  });

  videoSurface?.addEventListener("pointerdown", (event) => {
    if (!document.body.classList.contains("is-fullscreen")) {
      return;
    }
    if (event.target?.closest?.(".edge-tab, .edge-exit, .control-panel, .typing-panel")) {
      return;
    }
    showControls();
  });

  function syncBackdrop() {
    backdrop.style.display = "none";
  }

  let hideTimer = null;
  function showControls() {
    if (!document.body.classList.contains("is-fullscreen")) {
      document.body.classList.remove("fs-controls-visible");
      return;
    }
    document.body.classList.add("fs-controls-visible");
    if (hideTimer) {
      clearTimeout(hideTimer);
    }
    hideTimer = setTimeout(() => {
      hideTimer = null;
      if (!document.body.classList.contains("drawer-left-open") &&
        !document.body.classList.contains("drawer-right-open")) {
        document.body.classList.remove("fs-controls-visible");
      }
    }, 2200);
  }

  async function safeEnterFullscreen() {
    try {
      await document.documentElement.requestFullscreen();
    } catch (_) {
      document.body.classList.add("is-fullscreen");
      showControls();
    }
  }

  async function safeExitFullscreen() {
    try {
      if (document.fullscreenElement) {
        await document.exitFullscreen();
      } else {
        document.body.classList.remove("is-fullscreen");
      }
    } catch (_) {
      document.body.classList.remove("is-fullscreen");
    }
    document.body.classList.remove("fs-controls-visible");
  }

  return {
    setEnabled(next) {
      enabled = Boolean(next);
      toggles.forEach((btn) => {
        btn.disabled = !enabled;
      });
      exitButton.disabled = !enabled;
      leftToggle.disabled = !enabled;
      rightToggle.disabled = !enabled;
      if (!enabled) {
        document.body.classList.remove("drawer-left-open");
        document.body.classList.remove("drawer-right-open");
        document.body.classList.remove("fs-controls-visible");
        syncBackdrop();
      }
    },
    showControls,
  };
}
