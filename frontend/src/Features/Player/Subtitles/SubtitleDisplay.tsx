import { Component, type RefObject } from "react";

import type { SubtitleTrack } from "@/Features/Player/Types";
import { cn } from "@/Utils/ClassNames";
import { loadSubtitleCues, type VttCue } from "@/Features/Player/Subtitles/Vtt";

interface SubtitleDisplayProps {

  videoRef: RefObject<HTMLVideoElement | null>;
  track: SubtitleTrack | null;
  compact?: boolean;

}

interface SubtitleDisplayState {

  cue: VttCue | null;

}

export class SubtitleDisplay extends Component<SubtitleDisplayProps, SubtitleDisplayState> {

  private timer: ReturnType<typeof setInterval> | null = null;
  private loadGen = 0;
  private cues: VttCue[] = [];

  state: SubtitleDisplayState = {

    cue: null,

  };

  componentDidMount() {

    void this.loadTrack(this.props.track);

  }

  componentDidUpdate(prev: SubtitleDisplayProps) {

    if (prev.track?.id !== this.props.track?.id || prev.track?.proxyUrl !== this.props.track?.proxyUrl) {

      void this.loadTrack(this.props.track);

    }

  }

  componentWillUnmount() {

    this.loadGen += 1;
    this.stopTimer();

  }

  private stopTimer = () => {

    if (this.timer !== null) clearInterval(this.timer);

    this.timer = null;

  };

  private loadTrack = async (track: SubtitleTrack | null) => {

    const gen = ++this.loadGen;

    this.stopTimer();
    this.cues = [];
    this.setState({ cue: null });

    if (!track) return;

    try {

      const cues = await loadSubtitleCues(track.proxyUrl, track.format);

      if (gen !== this.loadGen) return;

      this.cues = cues.sort((a, b) => a.start - b.start);
      this.sync();

      if (this.cues.length > 0) {

        this.timer = setInterval(this.sync, 100);

      }

    } catch {

      // A failed track must not leave captions from the previous selection visible.

    }

  };

  private sync = () => {

    const time = this.props.videoRef.current?.currentTime ?? 0;

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

    const cue = found >= 0 && time < this.cues[found].end ? this.cues[found] : null;

    if (cue !== this.state.cue) this.setState({ cue });

  };

  render() {

    const { cue } = this.state;
    const { compact, track } = this.props;

    if (!track || !cue) return null;

    return (

      <div className={cn("pointer-events-none absolute inset-x-0 z-[35] flex justify-center", compact ? "bottom-20 px-4 landscape:bottom-10" : "bottom-24 px-6 sm:bottom-28")}>

        <p className={cn(

          "max-w-4xl whitespace-pre-line rounded-md border border-border-subtle/50 bg-black/75 text-center font-medium leading-snug text-white shadow-lg shadow-black/35",
          compact ? "px-2 py-1 text-[14px]" : "px-4 py-2.5 text-[20px]"

        )}>

          {cue.text}

        </p>

      </div>

    );

  }

}
