export class WebRTCClient {
  constructor(video, setStatus) {
    this.video = video;
    this.setStatus = setStatus;
    this.ws = null;
    this.pc = null;
    this.url = null;
    this.restartTimer = null;
  }

  connect(url) {
    this.url = url;
    return new Promise((resolve, reject) => {
      this.ws = new WebSocket(url);
      this.ws.onopen = async () => {
        try {
          await this.startPeer();
          resolve();
        } catch (err) {
          reject(err);
        }
      };
      this.ws.onmessage = (event) => {
        this.handleMessage(event.data);
      };
      this.ws.onerror = (err) => {
        reject(err);
      };
      this.ws.onclose = () => {
        this.setStatus("offline");
      };
    });
  }

  async handleMessage(raw) {
    let msg;
    try {
      msg = JSON.parse(raw);
    } catch (_) {
      return;
    }
    if (msg.t === "answer") {
      if (this.pc) {
        await this.pc.setRemoteDescription({ type: "answer", sdp: msg.sdp });
      }
    }
    if (msg.t === "ice" && msg.candidate && this.pc) {
      await this.pc.addIceCandidate(msg.candidate);
    }
    if (msg.t === "restart") {
      this.scheduleRestart();
    }
  }

  async startPeer() {
    this.setStatus("connecting");
    this.pc = new RTCPeerConnection();
    this.pc.addTransceiver("video", { direction: "recvonly" });
    this.pc.ontrack = (event) => {
      if (event.streams && event.streams[0]) {
        this.video.srcObject = event.streams[0];
        this.setStatus("streaming");
      }
    };
    this.pc.onicecandidate = (event) => {
      if (event.candidate && this.ws) {
        this.ws.send(JSON.stringify({ t: "ice", candidate: event.candidate }));
      }
    };
    const offer = await this.pc.createOffer();
    await this.pc.setLocalDescription(offer);
    await waitForIceGathering(this.pc);
    if (this.ws) {
      this.ws.send(JSON.stringify({ t: "offer", sdp: this.pc.localDescription.sdp }));
    }
  }

  scheduleRestart() {
    if (this.restartTimer) return;
    this.restartTimer = setTimeout(async () => {
      this.restartTimer = null;
      await this.restart();
    }, 300);
  }

  async restart() {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      return;
    }
    if (this.pc) {
      this.pc.ontrack = null;
      this.pc.onicecandidate = null;
      this.pc.close();
      this.pc = null;
    }
    await this.startPeer();
  }

  close() {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    if (this.pc) {
      this.pc.close();
      this.pc = null;
    }
  }
}

function waitForIceGathering(peer) {
  if (peer.iceGatheringState === "complete") {
    return Promise.resolve();
  }
  return new Promise((resolve) => {
    const onStateChange = () => {
      if (peer.iceGatheringState === "complete") {
        peer.removeEventListener("icegatheringstatechange", onStateChange);
        resolve();
      }
    };
    peer.addEventListener("icegatheringstatechange", onStateChange);
  });
}
