export function bindFullscreen({
  toggleButton,
  exitButton,
  leftToggle,
  rightToggle,
  backdrop,
}) {
  if (!toggleButton || !exitButton || !leftToggle || !rightToggle || !backdrop) {
    return {
      setEnabled: () => {},
    };
  }

  let enabled = true;

  toggleButton.addEventListener("click", async () => {
    if (!enabled) return;
    if (document.fullscreenElement) {
      await safeExitFullscreen();
      return;
    }
    await safeEnterFullscreen();
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
  });

  rightToggle.addEventListener("click", () => {
    if (!enabled) return;
    document.body.classList.toggle("drawer-right-open");
    document.body.classList.remove("drawer-left-open");
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
  });

  function syncBackdrop() {
    const open = document.body.classList.contains("drawer-left-open") ||
      document.body.classList.contains("drawer-right-open");
    backdrop.style.display = open ? "block" : "none";
  }

  async function safeEnterFullscreen() {
    try {
      await document.documentElement.requestFullscreen();
    } catch (_) {
      document.body.classList.add("is-fullscreen");
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
  }

  return {
    setEnabled(next) {
      enabled = Boolean(next);
      toggleButton.disabled = !enabled;
      exitButton.disabled = !enabled;
      leftToggle.disabled = !enabled;
      rightToggle.disabled = !enabled;
      if (!enabled) {
        document.body.classList.remove("drawer-left-open");
        document.body.classList.remove("drawer-right-open");
        syncBackdrop();
      }
    },
  };
}

