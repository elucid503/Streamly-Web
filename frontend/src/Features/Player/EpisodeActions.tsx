import { SkipForward } from "lucide-react";

import type { NextEpisode } from "@/Features/Player/Types";
import { cn } from "@/Utils/ClassNames";

interface EpisodeActionsProps {

  showSkipIntro: boolean;
  showUpNext: boolean;
  showUpNextMini: boolean;
  upNextCountdown: number;
  portraitMobile: boolean;
  nextEpisode?: NextEpisode | null;
  currentSeason?: number;
  onSkipIntro: () => void;
  onNextEpisode?: () => void;
  onCancel: () => void;

}

export function EpisodeActions({ showSkipIntro, showUpNext, showUpNextMini, upNextCountdown, portraitMobile, nextEpisode, currentSeason, onSkipIntro, onNextEpisode, onCancel }: EpisodeActionsProps) {

  return (

    <>

      {showSkipIntro && (

        <button onClick={(e) => {

            e.stopPropagation();

            onSkipIntro();

          }} className={cn(

            "pointer-events-auto absolute right-4 z-40 flex animate-fade-in items-center gap-2 rounded-md border border-border-subtle bg-surface/80 px-3 py-2 text-xs font-medium shadow-lg shadow-black/30 backdrop-blur-xl transition-colors hover:bg-surface-overlay sm:right-6 sm:px-4 sm:py-2.5 sm:text-sm",
            portraitMobile ? "player-skip-portrait" : "bottom-20"

          )} >

          <SkipForward size={14} />

          Skip Intro

        </button>

      )}

      {showUpNextMini && !showUpNext && nextEpisode && (

        <button onClick={(e) => {

            e.stopPropagation();

            onNextEpisode?.();

          }} className={cn(

            "pointer-events-auto absolute right-4 z-40 flex animate-fade-in items-center gap-2 rounded-md border border-border-subtle bg-surface/80 px-3 py-2 text-xs font-medium shadow-lg shadow-black/30 backdrop-blur-xl transition-colors hover:bg-surface-overlay sm:right-6 sm:px-4 sm:py-2.5 sm:text-sm",
            portraitMobile ? "player-upnext-portrait" : "bottom-28"

          )} >

          <SkipForward size={14} />

          {nextEpisode.season !== currentSeason ? "Next Season" : "Next Episode"}

        </button>

      )}

      {showUpNext && nextEpisode && (

        <div className={cn(

          "pointer-events-auto absolute right-4 z-40 w-[min(17rem,calc(100vw-2rem))] animate-fade-in rounded-lg border border-border-subtle bg-surface/80 p-3.5 shadow-lg shadow-black/30 backdrop-blur-xl sm:right-6 sm:w-72 sm:p-4",
          portraitMobile ? "player-upnext-portrait" : "bottom-28"

        )}>

          <p className="text-[11px] tracking-wide text-foreground-faint uppercase">

            Up Next

          </p>

          <p className="mt-1 text-sm font-medium">

            {nextEpisode.title}

          </p>

          <p className="text-xs text-foreground-muted">

            S{String(nextEpisode.season).padStart(2, "0")}E
            {String(nextEpisode.episode).padStart(2, "0")}

          </p>

          <div className="mt-3 flex gap-2">

            <button onClick={() => onCancel()} className="flex-1 rounded-md border border-border px-3 py-1.5 text-xs transition-colors hover:bg-surface-overlay" >

              Cancel

            </button>

            <button onClick={() => onNextEpisode?.()} className="flex-1 rounded-md bg-foreground px-3 py-1.5 text-xs text-surface transition-colors hover:bg-accent" >

              {upNextCountdown > 0 ? `Play (${upNextCountdown})` : "Play"}

            </button>

          </div>

        </div>

      )}

    </>

  );

}
