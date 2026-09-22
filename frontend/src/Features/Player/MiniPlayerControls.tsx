import type { MouseEvent, RefObject } from "react";
import { Maximize, Pause, Play, X } from "lucide-react";

import { cn } from "@/Utils/ClassNames";

interface MiniPlayerControlsProps {

  live?: boolean;
  playing: boolean;
  progressRef: RefObject<HTMLDivElement | null>;
  onDismiss?: () => void;
  onReturn?: () => void;
  onTogglePlay: () => void;
  onSeek: (event: MouseEvent<HTMLDivElement>) => void;

}

const miniControlClass = "flex h-8 flex-1 items-center justify-center text-foreground-muted transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-accent/40";

export function MiniPlayerControls({ live, playing, progressRef, onDismiss, onReturn, onTogglePlay, onSeek }: MiniPlayerControlsProps) {

  return (

    <div className={cn("relative z-10 flex shrink-0 py-2 flex-col bg-surface-raised", live && "border-t border-border-subtle")}>

      {!live && (

        <div
          className="absolute inset-x-0 top-0 z-20 h-3 -translate-y-1/2 cursor-pointer"
          onClick={(event) => {

            event.stopPropagation();

            onSeek(event);

          }}
          aria-label="Seek"
        >

          <div className="absolute inset-x-0 top-1/2 h-[3px] -translate-y-1/2 overflow-hidden bg-white/20">

            <div className="h-full bg-foreground" ref={progressRef} style={{ width: "0%" }} />

          </div>

        </div>

      )}

      <div className="flex h-8 items-center">

        <button
          type="button"
          onClick={(event) => { event.stopPropagation(); onDismiss?.(); }}
          className={miniControlClass}
          aria-label="Close miniplayer"
        >

          <X size={20} />

        </button>

        <span className="h-[90%] w-px shrink-0 bg-white/10" aria-hidden />

        <button
          type="button"
          onClick={(event) => { event.stopPropagation(); onTogglePlay(); }}
          className={miniControlClass}
          aria-label={playing ? "Pause" : "Play"}
        >

          {playing ? <Pause size={16} /> : <Play size={16} className="translate-x-px" />}

        </button>

        <span className="h-[90%] w-px shrink-0 bg-white/10" aria-hidden />

        <button
          type="button"
          onClick={(event) => { event.stopPropagation(); onReturn?.(); }}
          className={miniControlClass}
          aria-label="Return to full player"
        >

          <Maximize size={16} />

        </button>

        </div>

    </div>

  );

}
