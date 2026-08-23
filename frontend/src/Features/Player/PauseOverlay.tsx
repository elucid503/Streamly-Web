import { cn } from "@/Utils/ClassNames";
import { formatDuration } from "@/Utils/Time";

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

  simplified?: boolean;

}

interface PauseOverlayState {

  posterFailed: boolean;

}

export class PauseOverlay extends Component<PauseOverlayProps, PauseOverlayState> {

  state: PauseOverlayState = {

    posterFailed: false,

  };

  componentDidUpdate(prevProps: PauseOverlayProps) {

    if (prevProps.poster !== this.props.poster) {

      this.setState({ posterFailed: false });

    }

  }

  handlePosterError = () => {

    this.setState({ posterFailed: true });

  };

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
      simplified,
    } = this.props;

    const { posterFailed } = this.state;

    const isLive = layout === "live";
    const showProgress = !isLive;
    const showPoster = !!poster && !posterFailed;

    return (

      <>
        {visible && (

          <button type="button" onClick={(e) => {

              e.stopPropagation();

              onResume();

            }} className="absolute inset-0 z-[40] flex animate-fade-in cursor-pointer items-center justify-center overflow-hidden bg-surface/50 px-4 backdrop-blur-2xl sm:px-8" aria-label="Resume playback" >

            <div className="pointer-events-none absolute inset-0 bg-gradient-to-t from-black/70 via-black/35 to-black/20" />

            <div className="pointer-events-none relative z-10 flex w-full justify-center">

              <div className={cn(

                  "flex animate-fade-in flex-col items-center",
                  simplified
                    ? "w-fit max-w-2xl gap-4 px-2 text-center landscape:flex-row landscape:items-center landscape:gap-6 landscape:text-left"
                    : isLive ? "w-fit max-w-5xl gap-4 md:flex-row md:items-center md:gap-7 md:text-left" : "w-full max-w-5xl -mt-3 gap-4 md:flex-row md:items-center md:gap-7 md:text-left"

                )}

              >

                {showPoster && isLive && (

                  <div className="flex-shrink-0">

                    <img
                      src={poster}
                      alt=""
                      className={simplified ? "h-24 w-24 object-contain landscape:h-28 landscape:w-28" : "h-28 w-28 object-contain sm:h-36 sm:w-36"}
                      onError={this.handlePosterError}
                    />

                  </div>

                )}

                {showPoster && !isLive && (

                  <div className={cn(

                      "flex-shrink-0 overflow-hidden rounded-lg shadow-2xl ring-1 ring-white/10",
                      layout === "episode"
                        ? (simplified ? "aspect-video w-56 landscape:w-72" : "aspect-video w-full max-w-xl md:w-[min(42vw,28rem)]")
                        : (simplified ? "aspect-[2/3] w-28 landscape:w-32" : "aspect-[2/3] w-28 sm:w-36 md:w-40")

                    )}

                  >

                    <img
                      src={poster}
                      alt=""
                      className="size-full object-cover object-center"
                      onError={this.handlePosterError}
                    />

                  </div>

                )}

                <div className={cn("min-w-0", simplified ? "space-y-1.5" : "space-y-1.5")}>

                  <h2 className={simplified ? "text-2xl font-semibold tracking-tight landscape:text-3xl" : "text-2xl font-semibold tracking-tight sm:text-3xl"}>

                    {title}

                  </h2>

                  {isLive && subtitle && (

                    <p className={cn(

                        "font-medium tracking-wide text-foreground-muted uppercase",
                        simplified ? "text-xs" : "text-sm"

                      )}

                    >

                      {subtitle}

                    </p>

                  )}

                  {episodeTitle && (

                    <p className={simplified ? "text-base text-foreground/90 landscape:text-lg" : "text-base text-foreground/90 sm:text-lg"}>

                      {episodeTitle}

                    </p>

                  )}

                  {description && (

                    <p className={cn(

                        "text-foreground-muted",
                        simplified ? "line-clamp-3 max-w-xl text-sm leading-relaxed landscape:text-[15px]" : cn("text-sm leading-relaxed sm:text-[15px] sm:leading-7", !isLive ? "max-w-3xl sm:line-clamp-none" : "line-clamp-4")

                      )}

                    >

                      {description}

                    </p>

                  )}

                </div>

              </div>

            </div>

            {showProgress && !simplified && (

              <p className="pointer-events-none absolute inset-x-0 top-24 z-10 text-center text-xs tracking-wide text-foreground/40 sm:bottom-28 sm:text-sm">

                Paused at {formatDuration(progress * 1000)} /{" "}
                {totalDuration ? formatDuration(totalDuration * 1000) : "Unknown"}

              </p>

            )}

            {!simplified && (

              <p className="pointer-events-none absolute inset-x-0 bottom-24 z-10 text-center text-xs tracking-wide text-foreground/40 sm:bottom-28 sm:text-sm">

                Click Anywhere to Resume

              </p>

            )}

          </button>

        )}
      </>

    );

  }

}
