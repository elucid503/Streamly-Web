import { cn, formatDuration } from "@/lib/utils";

import { Component } from "react";

type PauseLayout = "movie" | "episode" | "live";

interface PauseOverlayProps {

  visible: boolean;

  poster?: string;

  title: string;
  subtitle?: string;

  episodeTitle?: string;
  description?: string;

  layout?: PauseLayout;

  pausedAt: number;
  totalDuration?: number;

  onResume: () => void;

}

export class PauseOverlay extends Component<PauseOverlayProps> {

  render() {

    const {
      visible,
      poster,
      title,
      subtitle,
      episodeTitle,
      description,
      layout = "movie",
      onResume,
      pausedAt: progress,
      totalDuration,
    } = this.props;

    const isLive = layout === "live";
    const showProgress = !isLive;

    return (

      <>
        {visible && (

          <button type="button" onClick={(e) => {

              e.stopPropagation();

              onResume();

            }} className="absolute inset-0 z-[40] flex animate-fade-in cursor-pointer items-center justify-center overflow-hidden bg-black/40 px-4 backdrop-blur-2xl sm:px-8" aria-label="Resume playback" >

            <div className="pointer-events-none absolute inset-0 bg-gradient-to-t from-black/70 via-black/35 to-black/20" />

            <div className="pointer-events-none relative z-10 flex w-full justify-center">

              <div className={cn(

                  "flex animate-fade-in flex-col items-center gap-4",
                  isLive ? "w-fit max-w-5xl md:flex-row md:items-center md:gap-7 md:text-left" : "w-full max-w-5xl -mt-3 md:flex-row md:items-center md:gap-7 md:text-left"

                )}

              >

                {poster && isLive && (

                  <div className="flex-shrink-0">

                    <img
                      src={poster}
                      alt=""
                      className="h-28 w-28 object-contain sm:h-36 sm:w-36"
                    />

                  </div>

                )}

                {poster && !isLive && (

                  <div className={cn(

                      "flex-shrink-0 overflow-hidden rounded-lg shadow-2xl ring-1 ring-white/10",
                      layout === "episode" ? "aspect-video w-full max-w-xl md:w-[min(42vw,28rem)]" : "aspect-[2/3] w-28 sm:w-36 md:w-40"

                    )}

                  >

                    <img src={poster} alt="" className="size-full object-cover object-center" />

                  </div>

                )}

                <div className="min-w-0 space-y-1.5">

                  <h2 className="text-2xl font-semibold tracking-tight sm:text-3xl">

                    {title}

                  </h2>

                  {subtitle && (

                    <p className="text-sm font-medium tracking-wide text-foreground-muted uppercase">

                      {subtitle}

                    </p>

                  )}

                  {episodeTitle && (

                    <p className="text-base text-foreground/90 sm:text-lg">

                      {episodeTitle}

                    </p>

                  )}

                  {description && (

                    <p className={cn(

                        "text-sm leading-relaxed text-foreground-muted sm:text-[15px] sm:leading-7",
                        !isLive ? "max-w-3xl sm:line-clamp-none" : "line-clamp-4"

                      )}

                    >

                      {description}

                    </p>

                  )}

                </div>

              </div>

            </div>

            {showProgress && (

              <p className="pointer-events-none absolute inset-x-0 top-24 z-10 text-center text-xs tracking-wide text-foreground/40 sm:bottom-28 sm:text-sm">

                Paused at {formatDuration(progress * 1000)} /{" "}
                {totalDuration ? formatDuration(totalDuration * 1000) : "Unknown"}

              </p>

            )}

            <p className="pointer-events-none absolute inset-x-0 bottom-24 z-10 text-center text-xs tracking-wide text-foreground/40 sm:bottom-28 sm:text-sm">

              Click Anywhere to Resume

            </p>

          </button>

        )}
      </>

    );

  }

}