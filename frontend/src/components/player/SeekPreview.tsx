import type HLS from "hls.js";
import { Component, createRef } from "react";

import { isProxiedStream } from "@/lib/streamClient";
import { formatDuration } from "@/lib/utils";

interface SeekPreviewProps {

  src: string;
  isHls: boolean;

}

export class SeekPreview extends Component<SeekPreviewProps> {

  private rootRef = createRef<HTMLDivElement>();
  private tickRef = createRef<HTMLDivElement>();
  private timeRef = createRef<HTMLSpanElement>();
  private videoRef = createRef<HTMLVideoElement>();

  private hls: HLS | null = null;
  private attachedSrc = "";
  private attachToken = 0;

  private pendingSec: number | null = null;
  private seeking = false;
  private lastSeekAt = 0;
  private seekTimer: ReturnType<typeof setTimeout> | null = null;
  private ready = false;

  private static readonly SEEK_THROTTLE_MS = 140;
  private static readonly SEEK_EPSILON_SEC = 0.35;

  componentDidUpdate(prev: SeekPreviewProps) {

    if (prev.src !== this.props.src || prev.isHls !== this.props.isHls) {

      this.teardownSource();

    }

  }

  componentWillUnmount() {

    this.clearSeekTimer();
    this.teardownSource();

  }

  /** Position + seek to ratio [0, 1] for the given duration. */
  update = (ratio: number, durationMs: number) => {

    if (!Number.isFinite(ratio) || !Number.isFinite(durationMs) || durationMs <= 0) {

      this.hide();
      return;

    }

    const clamped = Math.max(0, Math.min(1, ratio));
    const ms = clamped * durationMs;
    const pct = clamped * 100;

    const root = this.rootRef.current;
    const tick = this.tickRef.current;
    const time = this.timeRef.current;

    if (root) {

      // Clamp so the 160px bubble stays inside the scrubber track near the edges.
      root.style.left = `clamp(80px, ${pct}%, calc(100% - 80px))`;
      root.style.opacity = "1";
      root.setAttribute("data-visible", "true");

    }

    if (tick) {

      tick.style.left = `${pct}%`;
      tick.style.opacity = "1";

    }

    if (time) {

      time.textContent = formatDuration(ms);

    }

    void this.ensureSource();
    this.queueSeek(ms / 1000);

  };

  hide = () => {

    this.clearSeekTimer();
    this.pendingSec = null;

    const root = this.rootRef.current;
    const tick = this.tickRef.current;

    if (root) {

      root.style.opacity = "0";
      root.setAttribute("data-visible", "false");

    }

    if (tick) {

      tick.style.opacity = "0";

    }

  };

  private clearSeekTimer = () => {

    if (!this.seekTimer) return;

    clearTimeout(this.seekTimer);
    this.seekTimer = null;

  };

  private queueSeek = (sec: number) => {

    this.pendingSec = sec;

    if (this.seeking) return;

    const elapsed = performance.now() - this.lastSeekAt;

    if (elapsed < SeekPreview.SEEK_THROTTLE_MS) {

      this.clearSeekTimer();

      this.seekTimer = setTimeout(() => {

        this.seekTimer = null;
        this.flushSeek();

      }, SeekPreview.SEEK_THROTTLE_MS - elapsed);

      return;

    }

    this.flushSeek();

  };

  private flushSeek = () => {

    const video = this.videoRef.current;
    const sec = this.pendingSec;

    if (!video || sec == null || !Number.isFinite(sec)) return;

    if (!this.ready || !Number.isFinite(video.duration) || video.duration <= 0) return;

    const target = Math.max(0, Math.min(sec, Math.max(0, video.duration - 0.05)));

    if (Math.abs(video.currentTime - target) < SeekPreview.SEEK_EPSILON_SEC) {

      this.pendingSec = null;
      video.style.opacity = "1";
      return;

    }

    this.pendingSec = null;
    this.seeking = true;
    this.lastSeekAt = performance.now();

    try {

      video.currentTime = target;

    } catch {

      this.seeking = false;

    }

  };

  private onPreviewReady = () => {

    this.ready = true;

    const video = this.videoRef.current;

    if (video && !video.paused) {

      video.pause();

    }

    if (this.pendingSec != null) {

      this.flushSeek();

    }

  };

  private onPreviewSeeked = () => {

    this.seeking = false;

    const video = this.videoRef.current;

    if (video) {

      if (!video.paused) video.pause();

      video.style.opacity = "1";

    }

    if (this.pendingSec != null) {

      this.flushSeek();

    }

  };

  private ensureSource = async () => {

    const { src, isHls } = this.props;
    const video = this.videoRef.current;

    if (!video || !src.trim()) return;

    if (this.attachedSrc === src) return;

    this.teardownSource();

    const token = ++this.attachToken;
    this.attachedSrc = src;

    const proxied = isProxiedStream(src);

    if (proxied) {

      video.crossOrigin = "use-credentials";

    } else {

      video.removeAttribute("crossorigin");

    }

    this.ready = false;
    video.muted = true;
    video.playsInline = true;
    video.preload = "auto";
    video.style.opacity = "0";

    video.removeEventListener("seeked", this.onPreviewSeeked);
    video.removeEventListener("loadedmetadata", this.onPreviewReady);
    video.addEventListener("seeked", this.onPreviewSeeked);
    video.addEventListener("loadedmetadata", this.onPreviewReady);

    if (isHls) {

      const { default: HlsConstructor } = await import("hls.js");

      if (token !== this.attachToken || this.videoRef.current !== video) return;

      if (!HlsConstructor.isSupported()) {

        if (video.canPlayType("application/vnd.apple.mpegurl")) {

          video.src = src;

        }

        return;

      }

      this.hls = new HlsConstructor({

        enableWorker: true,
        maxBufferLength: 4,
        maxMaxBufferLength: 8,
        backBufferLength: 0,
        startLevel: 0,
        capLevelToPlayerSize: true,
        xhrSetup: proxied ? (xhr) => { xhr.withCredentials = true; } : undefined,

      });

      this.hls.loadSource(src);
      this.hls.attachMedia(video);

      this.hls.on(HlsConstructor.Events.MANIFEST_PARSED, () => {

        if (!this.hls || this.hls.levels.length === 0) return;

        // Prefer the lowest rung so previews stay cheap while scrubbing.
        this.hls.currentLevel = 0;
        this.hls.loadLevel = 0;

      });

      this.hls.on(HlsConstructor.Events.MEDIA_ATTACHED, () => {

        // Metadata may already be present after a prior attach in some browsers.
        if (video.readyState >= 1) this.onPreviewReady();

      });

      return;

    }

    video.src = src;

  };

  private teardownSource = () => {

    this.attachToken += 1;
    this.attachedSrc = "";
    this.seeking = false;
    this.ready = false;
    this.pendingSec = null;
    this.clearSeekTimer();

    if (this.hls) {

      this.hls.destroy();
      this.hls = null;

    }

    const video = this.videoRef.current;

    if (video) {

      video.removeEventListener("seeked", this.onPreviewSeeked);
      video.removeEventListener("loadedmetadata", this.onPreviewReady);
      video.style.opacity = "0";
      video.removeAttribute("src");
      video.load();

    }

  };

  render() {

    return (

      <>

        <div
          ref={this.tickRef}
          className="pointer-events-none absolute top-1/2 z-10 h-3 w-px -translate-x-1/2 -translate-y-1/2 bg-white/90 opacity-0 transition-opacity duration-75"
          aria-hidden
        />

        <div
          ref={this.rootRef}
          data-visible="false"
          className="pointer-events-none absolute bottom-[calc(100%+15px)] z-20 w-[160px] -translate-x-1/2 opacity-0 transition-opacity duration-100"
          aria-hidden
        >

          <div className="overflow-hidden rounded-md border border-white/15 bg-black shadow-xl shadow-black/50">

            <video
              ref={this.videoRef}
              className="aspect-video h-[90px] w-full bg-black object-contain opacity-0"
              muted
              playsInline
              preload="none"
              tabIndex={-1}
            />

          </div>

          <span
            ref={this.timeRef}
            className="mt-2 block text-center text-[11px] font-medium tabular-nums text-white drop-shadow-[0_1px_2px_rgba(0,0,0,0.85)]"
          >

            0:00

          </span>

        </div>

      </>

    );

  }

}
