// WebRTC-specific view helpers for calibration and pointer mapping.

export function isWebRTCFullscreenActive(videoMode) {
  return videoMode === "webrtc" && document.body.classList.contains("is-fullscreen");
}

// webRTCContentRect returns the video element box relative to the overlay box.
// We use the actual DOM rect instead of object-fit math because WebRTC fullscreen avoids transforms.
export function webRTCContentRect(overlayEl, videoEl) {
  if (!overlayEl || !videoEl) return null;
  const ob = overlayEl.getBoundingClientRect();
  const vb = videoEl.getBoundingClientRect();

  const fit = getComputedStyle(videoEl).objectFit;
  if (fit === "contain") {
    const mediaW = videoEl.videoWidth || 0;
    const mediaH = videoEl.videoHeight || 0;
    if (mediaW > 0 && mediaH > 0 && vb.width > 0 && vb.height > 0) {
      const scale = Math.min(vb.width / mediaW, vb.height / mediaH);
      const width = mediaW * scale;
      const height = mediaH * scale;
      const x = (vb.width - width) / 2;
      const y = (vb.height - height) / 2;
      return {
        x: vb.left - ob.left + x,
        y: vb.top - ob.top + y,
        width,
        height,
      };
    }
  }

  return {
    x: vb.left - ob.left,
    y: vb.top - ob.top,
    width: vb.width,
    height: vb.height,
  };
}
