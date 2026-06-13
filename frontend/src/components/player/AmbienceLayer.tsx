import { Component, createRef, type RefObject } from "react";

interface AmbienceLayerProps {
  videoRef: RefObject<HTMLVideoElement | null>;
  enabled: boolean;
}

const SAMPLE_W = 12;
const SAMPLE_H = 8;
const SAMPLE_INTERVAL_MS = 2400;
const ACTIVE_OPACITY = 0.82;
const VIDEO_RETRY_MS = 400;
const VIDEO_RETRY_MAX = 60;
const COLOR_LERP = 0.028;

interface Rgb {
  r: number;
  g: number;
  b: number;
}

const DEFAULT_PRIMARY: Rgb = { r: 36, g: 34, b: 44 };
const DEFAULT_SECONDARY: Rgb = { r: 28, g: 36, b: 42 };

const toRgba = (color: Rgb, alpha: number) =>
  `rgba(${Math.round(color.r)}, ${Math.round(color.g)}, ${Math.round(color.b)}, ${alpha})`;

const lerp = (from: number, to: number, t: number) => from + (to - from) * t;

const lerpRgb = (from: Rgb, to: Rgb, t: number): Rgb => ({
  r: lerp(from.r, to.r, t),
  g: lerp(from.g, to.g, t),
  b: lerp(from.b, to.b, t),
});

const gradientFor = (primary: Rgb, secondary: Rgb) => {
  const blend = lerpRgb(primary, secondary, 0.5);
  return (
    `radial-gradient(ellipse 120% 96% at 50% 50%, ${toRgba(blend, 0.58)} 0%, transparent 70%),` +
    `radial-gradient(ellipse 104% 90% at 25% 50%, ${toRgba(primary, 0.66)} 0%, transparent 64%),` +
    `radial-gradient(ellipse 104% 90% at 75% 50%, ${toRgba(secondary, 0.66)} 0%, transparent 64%)`
  );
};

const DEFAULT_GRADIENT = gradientFor(DEFAULT_PRIMARY, DEFAULT_SECONDARY);

const luminance = (color: Rgb) =>
  0.299 * color.r + 0.587 * color.g + 0.114 * color.b;

const saturation = (r: number, g: number, b: number) => {
  const max = Math.max(r, g, b);
  const min = Math.min(r, g, b);
  return max === 0 ? 0 : (max - min) / max;
};

const clampChannel = (value: number) => Math.max(0, Math.min(value, 255));

const colorDistance = (a: Rgb, b: Rgb) =>
  Math.abs(a.r - b.r) + Math.abs(a.g - b.g) + Math.abs(a.b - b.b);

export class AmbienceLayer extends Component<AmbienceLayerProps> {
  private layerRef = createRef<HTMLDivElement>();
  private sampleCanvas = document.createElement("canvas");
  private sampleCtx = this.sampleCanvas.getContext("2d", { willReadFrequently: true });
  private sampleTimer: ReturnType<typeof setInterval> | null = null;
  private videoRetryTimer: ReturnType<typeof setInterval> | null = null;
  private animFrameId: number | null = null;
  private videoRetryCount = 0;
  private videoListenersAttached = false;
  private lastSample = 0;
  private sampling = false;
  private displayPrimary = DEFAULT_PRIMARY;
  private displaySecondary = DEFAULT_SECONDARY;
  private targetPrimary = DEFAULT_PRIMARY;
  private targetSecondary = DEFAULT_SECONDARY;
  private lastRenderedGradient = "";

  componentDidMount() {
    this.sampleCanvas.width = SAMPLE_W;
    this.sampleCanvas.height = SAMPLE_H;
    if (this.props.enabled) {
      this.activate();
    }
  }

  componentDidUpdate(prev: AmbienceLayerProps) {
    if (prev.enabled !== this.props.enabled) {
      if (this.props.enabled) {
        this.activate();
      } else {
        this.deactivate();
      }
    }
  }

  componentWillUnmount() {
    this.deactivate();
  }

  activate = () => {
    this.renderGradient(this.displayPrimary, this.displaySecondary);
    this.bindVideoEvents();
    this.startSampling();
    this.startAnimation();
    this.startVideoRetry();
  };

  deactivate = () => {
    this.stopVideoRetry();
    this.unbindVideoEvents();
    this.stopSampling();
    this.stopAnimation();
  };

  startAnimation = () => {
    this.stopAnimation();
    const tick = () => {
      if (!this.props.enabled) return;

      const prevPrimary = this.displayPrimary;
      const prevSecondary = this.displaySecondary;

      this.displayPrimary = lerpRgb(this.displayPrimary, this.targetPrimary, COLOR_LERP);
      this.displaySecondary = lerpRgb(this.displaySecondary, this.targetSecondary, COLOR_LERP);

      const moved =
        colorDistance(prevPrimary, this.displayPrimary) +
          colorDistance(prevSecondary, this.displaySecondary) >
        0.4;

      if (moved) {
        this.renderGradient(this.displayPrimary, this.displaySecondary);
      }

      this.animFrameId = requestAnimationFrame(tick);
    };
    this.animFrameId = requestAnimationFrame(tick);
  };

  stopAnimation = () => {
    if (this.animFrameId !== null) {
      cancelAnimationFrame(this.animFrameId);
      this.animFrameId = null;
    }
  };

  startVideoRetry = () => {
    this.stopVideoRetry();
    this.videoRetryCount = 0;
    this.videoRetryTimer = setInterval(() => {
      if (!this.props.enabled) return;
      this.videoRetryCount += 1;
      if (!this.props.videoRef.current) {
        if (this.videoRetryCount >= VIDEO_RETRY_MAX) this.stopVideoRetry();
        return;
      }
      this.bindVideoEvents();
      if (!this.sampleTimer) this.startSampling();
      if (this.videoListenersAttached) this.stopVideoRetry();
    }, VIDEO_RETRY_MS);
  };

  stopVideoRetry = () => {
    if (this.videoRetryTimer) {
      clearInterval(this.videoRetryTimer);
      this.videoRetryTimer = null;
    }
  };

  bindVideoEvents = () => {
    if (this.videoListenersAttached) return;
    const video = this.props.videoRef.current;
    if (!video) return;
    video.addEventListener("loadeddata", this.onVideoReady);
    video.addEventListener("playing", this.onVideoReady);
    this.videoListenersAttached = true;
  };

  unbindVideoEvents = () => {
    if (!this.videoListenersAttached) return;
    const video = this.props.videoRef.current;
    if (video) {
      video.removeEventListener("loadeddata", this.onVideoReady);
      video.removeEventListener("playing", this.onVideoReady);
    }
    this.videoListenersAttached = false;
  };

  onVideoReady = () => {
    if (!this.props.enabled) return;
    this.lastSample = 0;
    this.maybeSample();
  };

  startSampling = () => {
    this.stopSampling();
    if (!this.props.enabled) return;
    this.sampleTimer = setInterval(this.maybeSample, SAMPLE_INTERVAL_MS);
    this.maybeSample();
  };

  stopSampling = () => {
    if (this.sampleTimer) {
      clearInterval(this.sampleTimer);
      this.sampleTimer = null;
    }
  };

  maybeSample = () => {
    if (this.sampling || !this.props.enabled) return;

    const now = performance.now();
    if (now - this.lastSample < SAMPLE_INTERVAL_MS * 0.9) return;

    const video = this.props.videoRef.current;
    if (!video || video.readyState < 2 || video.videoWidth === 0) {
      return;
    }

    this.lastSample = now;
    this.sampling = true;
    this.extractTargets(video);
  };

  extractTargets = (video: HTMLVideoElement) => {
    try {
      const sampled = this.extractColors(video);
      if (sampled) {
        this.targetPrimary = sampled.primary;
        this.targetSecondary = sampled.secondary;
      }
    } finally {
      this.sampling = false;
    }
  };

  renderGradient = (primary: Rgb, secondary: Rgb) => {
    const gradient = gradientFor(primary, secondary);
    if (gradient === this.lastRenderedGradient) return;
    this.lastRenderedGradient = gradient;
    if (this.layerRef.current) {
      this.layerRef.current.style.background = gradient;
    }
  };

  muteBrightPixel = (r: number, g: number, b: number): [number, number, number] | null => {
    const lum = luminance({ r, g, b });
    const sat = saturation(r, g, b);

    let nr = r;
    let ng = g;
    let nb = b;

    if (lum > 175 && sat < 0.18) {
      const warmth = lum > 220 ? 0.34 : 0.42;
      nr = 84 * warmth;
      ng = 80 * warmth;
      nb = 70 * warmth;
      return [clampChannel(nr), clampChannel(ng), clampChannel(nb)];
    }

    if (lum > 185) {
      const crush = 0.1;
      nr *= crush;
      ng *= crush;
      nb *= crush;
    } else if (lum > 120) {
      const t = (lum - 120) / 65;
      const crush = 0.55 - t * 0.42;
      nr *= crush;
      ng *= crush;
      nb *= crush;
    }

    const avg = (nr + ng + nb) / 3;
    const desat = lum > 110 ? 0.42 : 0.62;
    nr = nr * desat + avg * (1 - desat);
    ng = ng * desat + avg * (1 - desat);
    nb = nb * desat + avg * (1 - desat);

    const chromaBoost = 1.42;
    nr = avg + (nr - avg) * chromaBoost;
    ng = avg + (ng - avg) * chromaBoost;
    nb = avg + (nb - avg) * chromaBoost;

    return [
      clampChannel(nr),
      clampChannel(ng),
      clampChannel(nb),
    ];
  };

  quadrantAverage = (
    data: Uint8ClampedArray,
    x0: number,
    y0: number,
    x1: number,
    y1: number,
  ): Rgb => {
    let r = 0;
    let g = 0;
    let b = 0;
    let count = 0;

    for (let y = y0; y < y1; y++) {
      for (let x = x0; x < x1; x++) {
        const i = (y * SAMPLE_W + x) * 4;
        const muted = this.muteBrightPixel(data[i], data[i + 1], data[i + 2]);
        if (!muted) continue;
        const [mr, mg, mb] = muted;
        const weight = 0.2 + saturation(mr, mg, mb) * 1.6;
        r += mr * weight;
        g += mg * weight;
        b += mb * weight;
        count += weight;
      }
    }

    if (count === 0) return { r: 24, g: 24, b: 32 };
    return this.refineColor({ r: r / count, g: g / count, b: b / count });
  };

  refineColor = (color: Rgb): Rgb => {
    const lum = luminance(color);
    const sat = saturation(color.r, color.g, color.b);
    let { r, g, b } = color;

    if (lum > 96) {
      const scale = 96 / lum;
      r *= scale;
      g *= scale;
      b *= scale;
    }

    if (sat < 0.14) {
      const pull = 0.72;
      r = lerp(r, DEFAULT_PRIMARY.r, pull);
      g = lerp(g, DEFAULT_PRIMARY.g, pull);
      b = lerp(b, DEFAULT_PRIMARY.b, pull);
    }

    const avg = (r + g + b) / 3;
    const chroma = 1.34;
    return {
      r: clampChannel(avg + (r - avg) * chroma),
      g: clampChannel(avg + (g - avg) * chroma),
      b: clampChannel(avg + (b - avg) * chroma),
    };
  };

  colorScore = (color: Rgb) => {
    const { r, g, b } = color;
    const lum = luminance(color);
    const sat = saturation(r, g, b);
    if (lum > 105 && sat < 0.16) return -200;
    return sat * 160 - Math.max(0, lum - 70) * 0.85;
  };

  extractColors = (video: HTMLVideoElement): { primary: Rgb; secondary: Rgb } | null => {
    const ctx = this.sampleCtx;
    if (!ctx) return null;

    try {
      ctx.drawImage(video, 0, 0, SAMPLE_W, SAMPLE_H);
      const data = ctx.getImageData(0, 0, SAMPLE_W, SAMPLE_H).data;
      return this.colorsFromPixels(data);
    } catch {
      return null;
    }
  };

  colorsFromPixels = (data: Uint8ClampedArray): { primary: Rgb; secondary: Rgb } => {
    const midX = Math.floor(SAMPLE_W / 2);
    const midY = Math.floor(SAMPLE_H / 2);

    const quadrants: Rgb[] = [
      this.quadrantAverage(data, 0, 0, midX, midY),
      this.quadrantAverage(data, midX, 0, SAMPLE_W, midY),
      this.quadrantAverage(data, 0, midY, midX, SAMPLE_H),
      this.quadrantAverage(data, midX, midY, SAMPLE_W, SAMPLE_H),
    ];

    const ranked = quadrants
      .map((color) => ({ color, score: this.colorScore(color) }))
      .sort((a, b) => b.score - a.score);

    const viable = ranked.filter((entry) => entry.score > -50).map((entry) => entry.color);
    const primary = viable[0] ?? DEFAULT_PRIMARY;
    const secondary = viable[1] ?? viable[0] ?? DEFAULT_SECONDARY;

    return {
      primary: this.refineColor(primary),
      secondary: this.refineColor(secondary),
    };
  };

  render() {
    if (!this.props.enabled) return null;

    return (
      <div className="pointer-events-none absolute inset-0 z-0 overflow-hidden bg-black" aria-hidden>
        <div
          ref={this.layerRef}
          className="absolute inset-[-26%] scale-[1.16] blur-[92px] saturate-[1.5]"
          style={{
            background: DEFAULT_GRADIENT,
            opacity: ACTIVE_OPACITY,
          }}
        />
        <div className="absolute inset-0 bg-black/12" />
      </div>
    );
  }
}
