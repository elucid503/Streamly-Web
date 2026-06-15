import { activeWordIndex, alignCue, mergeModelTimings, type AlignedSubtitleCue, } from "@/lib/subtitleAlignment";
import { alignWords, isModelReady, warmupAligner } from "@/lib/alignmentClient";

import { AudioTap } from "@/lib/audioTap";

import { loadSubtitleCues } from "@/lib/vtt";
import { cn } from "@/lib/utils";
import type { SubtitleTrack } from "@/lib/types";

import { Component, type RefObject } from "react";

interface SubtitleDisplayProps {

  videoRef: RefObject<HTMLVideoElement | null>;

  track: SubtitleTrack | null;

}

interface SubtitleDisplayState {

  cue: AlignedSubtitleCue | null;

  activeWord: number;

  exitingCue: AlignedSubtitleCue | null;

  exitingActiveWord: number;

  exitMode: "slide" | "fade";

  exitKey: string;

  enterMode: "slide" | "fade";

}

interface RefinedCue {

  cue: AlignedSubtitleCue;

  final: boolean;

  applied: boolean;

}

const ALIGN_INTERVAL_MS = 250;
const MAX_ALIGN_SECONDS = 15;
const SUBTITLE_EXIT_MS = 180;
const IMMEDIATE_NEXT_GAP_SECONDS = 0.65;

const SILENCE_PEAK = 1e-4;
const MAX_MODEL_ERRORS = 3;

export class SubtitleDisplay extends Component<SubtitleDisplayProps, SubtitleDisplayState> {

  private rafId: number | null = null;
  private exitTimer: number | null = null;

  private loadGen = 0;

  private cues: AlignedSubtitleCue[] = [];
  private lastCueIndex = -1;

  private tap: AudioTap | null = null;

  private refined = new Map<number, RefinedCue>();

  private skipReasons = new Map<number, string>();
  private loggedFallback = new Set<number>();
  private attempted = new Set<number>(); // tracks first attempt per cue to bypass interval

  private aligning = false;

  private lastAttempt = 0;
  private modelErrors = 0;

  private modelDisabled = false;

  state: SubtitleDisplayState = {

    cue: null,
    activeWord: -1,

    exitingCue: null,
    exitingActiveWord: -1,

    exitMode: "fade",
    exitKey: "",
    enterMode: "fade",

  };

  componentDidMount() {

    void this.loadTrack(this.props.track);

    this.rafId = requestAnimationFrame(this.sync);

  }

  componentDidUpdate(prev: SubtitleDisplayProps) {

    if (prev.track?.id !== this.props.track?.id || prev.track?.proxyUrl !== this.props.track?.proxyUrl) {

      void this.loadTrack(this.props.track);

    }

  }

  componentWillUnmount() {

    this.loadGen += 1;

    if (this.rafId !== null) cancelAnimationFrame(this.rafId);
    if (this.exitTimer !== null) window.clearTimeout(this.exitTimer);

    this.tap?.stop();

  }

  private loadTrack = async (track: SubtitleTrack | null) => {

    const gen = ++this.loadGen;

    this.cues = [];

    this.refined.clear();
    this.skipReasons.clear();
    this.loggedFallback.clear();
    this.attempted.clear();

    this.lastCueIndex = -1;

    if (!track) return;

    if (!this.modelDisabled) warmupAligner();

    try {

      const cues = await loadSubtitleCues(track.proxyUrl, track.format);

      if (gen !== this.loadGen) return;

      this.cues = cues.sort((a, b) => a.start - b.start).map(alignCue);

    } catch {

      // No cues to show; the sync loop clears any stale state.

    }

  };

  private findCueIndex(time: number) {

    let lo = 0;

    let hi = this.cues.length - 1;

    let found = -1;

    while (lo <= hi) {

      const mid = (lo + hi) >> 1;

      if (this.cues[mid].start <= time) {

        found = mid;

        lo = mid + 1;

      } else {

        hi = mid - 1;

      }

    }

    return found >= 0 && time < this.cues[found].end ? found : -1;

  }

  private sync = () => {

    const time = this.props.videoRef.current?.currentTime ?? 0;

    const index = this.findCueIndex(time);

    let enterMode: SubtitleDisplayState["enterMode"] | null = null;

    if (index !== this.lastCueIndex) {

      const prev = this.lastCueIndex;

      this.lastCueIndex = index;

      if (prev >= 0) this.onCueExit(prev, index, time);
      if (index >= 0) enterMode = this.isImmediateCueTransition(prev, index, time) ? "slide" : "fade";

    }

    if (index >= 0) this.maybeRefine(index);

    const cue = index >= 0 ? (this.refined.get(index)?.cue ?? this.cues[index]) : null;

    const activeWord = cue ? activeWordIndex(cue, time) : -1;

    if (cue !== this.state.cue || activeWord !== this.state.activeWord) {

      if (enterMode) {
        this.setState({ cue, activeWord, enterMode });
      } else {
        this.setState({ cue, activeWord });
      }

    }

    this.rafId = requestAnimationFrame(this.sync);

  };

  private maybeRefine(index: number, force = false) {

    if (this.modelDisabled || this.aligning) return;

    if (this.refined.get(index)?.final) return;

    const cue = this.cues[index];
    if (!cue || cue.words.length === 0 || cue.isAnnotation) return;

    if (cue.end - cue.start > MAX_ALIGN_SECONDS) return;

    // First attempt per cue bypasses the interval throttle so we start

    const isFirst = !this.attempted.has(index);
    if (isFirst) this.attempted.add(index);

    if (!force && !isFirst && performance.now() - this.lastAttempt < ALIGN_INTERVAL_MS) return;

    const video = this.props.videoRef.current;

    if (!video || video.paused) return;

    void this.attemptRefine(index, cue, video);

  }

  private onCueExit(index: number, nextIndex: number, time: number) {

    this.maybeRefine(index, true);

    this.showExitingCue(index, nextIndex, time);

    if (!this.refined.get(index)?.applied && !this.loggedFallback.has(index)) {

      // Suppress warnings while the model is still loading

      if (!isModelReady()) return;

      this.loggedFallback.add(index);

      console.warn("[subtitles] estimated word timings used for cue", {

        index,
        text: this.cues[index]?.text,

        reason: this.skipReasons.get(index) ?? (this.modelDisabled ? "model alignment disabled" : "model result not ready in time"),

      });

    }

  }

  private showExitingCue(index: number, nextIndex: number, time: number) {

    const cue = this.refined.get(index)?.cue ?? this.cues[index];
    if (!cue || this.state.cue !== cue) return;

    const nextCue = this.nextCueForExit(index, nextIndex);
    const hasImmediateNext = this.isImmediateCueTransition(index, nextIndex, time, cue, nextCue);

    if (this.exitTimer !== null) window.clearTimeout(this.exitTimer);

    this.setState({

      exitingCue: cue,
      exitingActiveWord: this.state.activeWord,

      exitMode: hasImmediateNext ? "slide" : "fade",
      exitKey: `${cue.start}-${cue.end}-${hasImmediateNext ? "slide" : "fade"}`,

    });

    this.exitTimer = window.setTimeout(() => {

      this.setState({ exitingCue: null });
      this.exitTimer = null;

    }, SUBTITLE_EXIT_MS);

  }

  private nextCueForExit(index: number, nextIndex: number) {

    if (nextIndex >= 0) return this.cues[nextIndex];

    return this.cues[index + 1] ?? null;

  }

  private isImmediateCueTransition(previousIndex: number, nextIndex: number, time: number, previousCue = this.cues[previousIndex], nextCue = nextIndex >= 0 ? this.cues[nextIndex] : null) {

    if (!previousCue || !nextCue) return false;

    const gap = Math.max(0, nextCue.start - previousCue.end);

    return gap <= IMMEDIATE_NEXT_GAP_SECONDS && time >= previousCue.end - 0.08;

  }

  private markUnalignable(index: number, cue: AlignedSubtitleCue, reason: string) {

    const existing = this.refined.get(index);

    this.refined.set(index, {

      cue: existing?.cue ?? cue,
      final: true,

      applied: existing?.applied ?? false,

    });

    this.skipReasons.set(index, reason);

  }

  private attemptRefine = async (index: number, cue: AlignedSubtitleCue, video: HTMLVideoElement) => {

    this.aligning = true;
    this.lastAttempt = performance.now();

    try {

      if (video.muted || video.volume === 0) {

        this.skipReasons.set(index, "player is muted");

        return;

      }

      this.tap ??= new AudioTap(video);

      await this.tap.start();

      const buffered = this.tap.bufferedRange();

      if (!buffered || buffered.end < cue.start + 0.5) {

        this.skipReasons.set(index, "audio not captured yet");

        return;

      }

      if (buffered.start > cue.start + 0.2) {

        this.markUnalignable(index, cue, "playback joined mid-cue");

        return;

      }

      const window = this.tap.window(

        Math.max(cue.start - 0.15, 0),
        Math.min(buffered.end, cue.end + 0.15)

      );

      if (!window) {

        this.skipReasons.set(index, "audio window unavailable");

        return;

      }

      if (window.peak < SILENCE_PEAK) {

        this.markUnalignable(index, cue, "captured audio is silent");

        return;

      }

      if (!isModelReady()) return; // Doesn't block on alignWords() during model download

      const speechWords = cue.words.flatMap((word, index) =>

        word.isAnnotation ? [] : [{ index, text: word.text }]

      );

      if (speechWords.length === 0) return;

      const gen = this.loadGen;

      const currentTime = video.currentTime;

      let timings = await alignWords({

        audio: window.audio,
        words: speechWords.map((word) => word.text),

        start: window.start,
        end: window.end,

      });

      timings = timings.map((timing) => ({

        ...timing,
        index: speechWords[timing.index]?.index ?? timing.index,

      }));

      if (gen !== this.loadGen) return;

      const final = window.end >= cue.end;

      // Trim only trailing words that may be cut off near the window edge.

      if (!final) {

        while (timings.length > 0 && timings[timings.length - 1].end > window.end - 0.08) {

          timings = timings.slice(0, -1);

        }

      }

      if (timings.length === 0) {

        this.skipReasons.set(index, "no confident word timings");

        return;

      }

      // Base each merge on the previous refined result

      const baseCue = this.refined.get(index)?.cue ?? cue;

      this.refined.set(index, {

        cue: mergeModelTimings(baseCue, timings, final ? Infinity : currentTime),

        final,
        applied: true,

      });

    } catch (error) {

      this.modelErrors += 1;

      this.skipReasons.set(index, error instanceof Error ? error.message : "alignment failed");

      if (this.modelErrors >= MAX_MODEL_ERRORS && !this.modelDisabled) {

        this.modelDisabled = true;

        console.warn("[subtitles] disabling model alignment after repeated failures", {

          lastError: this.skipReasons.get(index),

        });

      }

    } finally {

      this.aligning = false;

    }

  };

  render() {

    const { cue, activeWord, exitingCue, exitingActiveWord, exitMode, exitKey, enterMode } = this.state;

    if (!this.props.track || (!cue && !exitingCue)) return null;

    const renderCue = (displayCue: AlignedSubtitleCue, displayActiveWord: number, className: string, key: string) => (

      <p className={cn("max-w-4xl rounded-md bg-black/50 px-4 py-2.5 text-center text-[18px] leading-snug font-medium shadow-2xl shadow-black/20 backdrop-blur-md sm:text-[20px]", className)}

        key={key}
        style={{ textShadow: "0 1px 2px rgba(0, 0, 0, 0.75)" }}

      >
        {displayCue.words.map((word, index) => (

          <span key={index}>

            {word.lineBreakBefore ? <br /> : index > 0 ? " " : null}

            <span className={ index <= displayActiveWord ? "text-white transition-colors duration-150" : "text-white/40 transition-colors duration-150" } >

              {word.text}

            </span>

          </span>

        ))}
      </p>

    );

    return (

      <div className="pointer-events-none absolute inset-x-0 bottom-24 z-[35] flex justify-center px-6 sm:bottom-28">

        <div className="relative flex w-full justify-center">

          {exitingCue && renderCue(

            exitingCue,
            exitingActiveWord,

            cn(

              "absolute bottom-0 left-1/2 -translate-x-1/2",
              exitMode === "slide" ? "subtitle-slide-out" : "subtitle-fade-out"

            ),

            `exit-${exitKey}`

          )}

          {cue && renderCue(

            cue,
            activeWord,

            enterMode === "slide" ? "subtitle-slide-in" : "animate-fade-in",
            `${cue.start}-${cue.text}`

          )}

        </div>

      </div>

    );

  }

}
