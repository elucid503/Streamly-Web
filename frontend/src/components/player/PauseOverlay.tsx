import { cn, formatDuration } from "@/lib/utils";

import { Component } from "react";

interface PauseOverlayProps {

  visible: boolean;

  poster?: string;
  still?: boolean;

  title: string;
  subtitle?: string;

  episodeTitle?: string;
  description?: string;

  pausedAt: number; // in seconds

  totalDuration?: number; // in seconds

  onResume: () => void;

}

export class PauseOverlay extends Component<PauseOverlayProps> {

  render() {

    const { visible, poster, still, title, subtitle, episodeTitle, description, onResume, pausedAt: progress, totalDuration } = this.props;

    return (

      <>
        {visible && (

          <button type="button" onClick={(e) => {

              e.stopPropagation();

              onResume();

            }} className="absolute inset-0 z-[40] flex animate-fade-in cursor-pointer items-center justify-center overflow-hidden bg-black/35 px-4 backdrop-blur-md sm:px-8" aria-label="Resume playback" >

            <div className="pointer-events-none absolute inset-0">

              {poster && (

                <img className={cn(

                    "h-full w-full opacity-35 blur-2xl",
                    still ? "object-cover" : "scale-110 object-cover"

                  )}

                  src={poster}
                  alt=""

                />

              )}

              <div className="absolute inset-0 bg-gradient-to-t from-black/85 via-black/50 to-black/30" />

            </div>

            <div className={cn(

                "pointer-events-none relative z-10 flex w-full animate-fade-in flex-col items-center gap-4",
                still ? "max-w-5xl -mt-3 md:flex-row md:items-center md:gap-7 md:text-left" : "max-w-2xl text-center"

              )}

            >

              {poster && (

                <div className={cn(

                    "flex-shrink-0 overflow-hidden rounded-lg shadow-2xl ring-1 ring-white/10",
                    still ? "aspect-video w-full max-w-xl md:w-[min(42vw,28rem)]" : "aspect-[2/3] w-28 sm:w-36"

                  )}

                >

                  <img src={poster} alt="" className="size-full object-cover object-center" />

                </div>

              )}

              <div className={cn("min-w-0 space-y-1.5", still && "flex-1")}>

                <h2 className="text-2xl font-semibold tracking-tight sm:text-3xl">

                  {title}

                </h2>

                {subtitle && (

                  <p className={cn(

                      "text-sm font-medium text-foreground-muted",
                      !still && "tracking-wide uppercase"

                    )}

                  >

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
                      still ? "max-w-3xl sm:line-clamp-none" : "line-clamp-4 max-w-xl"

                    )}

                  >

                    {description}

                  </p>

                )}

              </div>

            </div>

            <p className="pointer-events-none absolute inset-x-0 top-24 z-10 text-center text-xs tracking-wide text-foreground/40 sm:bottom-28 sm:text-sm">

              Paused at {formatDuration(progress * 1000)} /{" "}
              {totalDuration ? formatDuration(totalDuration * 1000) : "Unknown"}

            </p>

            <p className="pointer-events-none absolute inset-x-0 bottom-24 z-10 text-center text-xs tracking-wide text-foreground/40 sm:bottom-28 sm:text-sm">

              Click Anywhere to Resume

            </p>

          </button>

        )}
      </>

    );

  }

}
