// ScriptProcessorNode runs on the main thread, so video.currentTime at the
// time of onaudioprocess accurately reflects when the audio block was rendered
// (within one block ≈ 85ms at 48kHz). AudioWorklet runs off-thread; by the
// time its message arrives, currentTime has drifted, causing timestamp skew.
const TARGET_SAMPLE_RATE = 16000;
const MAX_BUFFER_SECONDS = 60;
const BLOCK_SIZE = 4096;

interface TapChunk {
  start: number;
  end: number;
  rate: number;
  samples: Float32Array;
}

const resampleTo16k = (samples: Float32Array, rate: number): Float32Array => {
  if (rate === TARGET_SAMPLE_RATE) return samples;
  const outLength = Math.max(1, Math.round((samples.length * TARGET_SAMPLE_RATE) / rate));
  const out = new Float32Array(outLength);
  const ratio = (samples.length - 1) / Math.max(outLength - 1, 1);
  for (let i = 0; i < outLength; i++) {
    const pos = i * ratio;
    const i0 = Math.floor(pos);
    const i1 = Math.min(i0 + 1, samples.length - 1);
    out[i] = samples[i0] + (samples[i1] - samples[i0]) * (pos - i0);
  }
  return out;
};

export class AudioTap {
  private context: AudioContext | null = null;
  private processor: ScriptProcessorNode | null = null;
  private chunks: TapChunk[] = [];
  private startPromise: Promise<void> | null = null;

  constructor(private video: HTMLVideoElement) {}

  start(): Promise<void> {
    this.startPromise ??= this.init();
    return this.startPromise;
  }

  private async init() {
    const Ctor = window.AudioContext ?? window.webkitAudioContext;
    if (!Ctor) throw new Error("AudioContext not supported");

    this.context = new Ctor();
    const source = this.context.createMediaElementSource(this.video);
    this.processor = this.context.createScriptProcessor(BLOCK_SIZE, 2, 1);

    this.processor.onaudioprocess = (event) => {
      if (this.video.paused || this.video.seeking) return;
      const input = event.inputBuffer;
      const length = input.length;
      const rate = input.sampleRate;
      const samples = new Float32Array(length);
      for (let ch = 0; ch < input.numberOfChannels; ch++) {
        const data = input.getChannelData(ch);
        for (let i = 0; i < length; i++) samples[i] += data[i] / input.numberOfChannels;
      }

      const end = this.video.currentTime;
      const start = Math.max(0, end - length / rate);

      const last = this.chunks[this.chunks.length - 1];
      if (last && (end < last.end - 0.5 || start > last.end + 1.0)) {
        this.chunks = [];
      }
      this.chunks.push({ start, end, rate, samples });

      const minTime = end - MAX_BUFFER_SECONDS;
      while (this.chunks.length > 0 && this.chunks[0].end < minTime) this.chunks.shift();
    };

    source.connect(this.context.destination);
    source.connect(this.processor);
    this.processor.connect(this.context.destination);

    if (this.context.state === "suspended") {
      await this.context.resume().catch(() => undefined);
    }
  }

  stop() {
    // Clear chunks but keep the context alive — closing it would permanently
    // mute the video element since MediaElementSourceNode is a one-way door.
    this.chunks = [];
  }

  bufferedRange() {
    if (this.chunks.length === 0) return null;
    return { start: this.chunks[0].start, end: this.chunks[this.chunks.length - 1].end };
  }

  window(start: number, end: number) {
    if (this.chunks.length === 0 || !this.context) return null;
    const rate = this.context.sampleRate;
    const buffered = this.bufferedRange()!;
    const from = Math.max(start, buffered.start);
    const to = Math.min(end, buffered.end);
    if (to - from < 0.2) return null;

    const native = new Float32Array(Math.ceil((to - from) * rate));
    let peak = 0;

    for (const chunk of this.chunks) {
      const contribStart = Math.max(from, chunk.start);
      const contribEnd = Math.min(to, chunk.end);
      if (contribEnd <= contribStart) continue;

      const srcStart = Math.floor((contribStart - chunk.start) * chunk.rate);
      const srcEnd = Math.min(chunk.samples.length, Math.ceil((contribEnd - chunk.start) * chunk.rate));
      const dstStart = Math.floor((contribStart - from) * rate);

      for (let i = 0; i < srcEnd - srcStart; i++) {
        const dst = dstStart + i;
        if (dst >= native.length) break;
        const value = chunk.samples[srcStart + i];
        native[dst] = value;
        const mag = Math.abs(value);
        if (mag > peak) peak = mag;
      }
    }

    return { audio: resampleTo16k(native, rate), start: from, end: to, peak };
  }
}

declare global {
  interface Window {
    webkitAudioContext?: typeof AudioContext;
  }
}
