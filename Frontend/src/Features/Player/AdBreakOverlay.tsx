import { cn } from "@/Utils/ClassNames";

import { Component } from "react";

interface AdBreakOverlayProps {

  visible: boolean;

  poster?: string;
  title: string;
  subtitle?: string;

  simplified?: boolean;

  onDismiss: () => void;

}

interface AdBreakOverlayState {

  posterFailed: boolean;

}

export class AdBreakOverlay extends Component<AdBreakOverlayProps, AdBreakOverlayState> {

  state: AdBreakOverlayState = {

    posterFailed: false,

  };

  componentDidUpdate(prevProps: AdBreakOverlayProps) {

    if (prevProps.poster !== this.props.poster) {

      this.setState({ posterFailed: false });

    }

  }

  handlePosterError = () => {

    this.setState({ posterFailed: true });

  };

  render() {

    const { visible, poster, title, subtitle, simplified, onDismiss } = this.props;
    const { posterFailed } = this.state;
    const showPoster = !!poster && !posterFailed;

    return (

      <>

        {visible && (

          <button type="button" onClick={(e) => {

              e.stopPropagation();

              onDismiss();

            }} className="absolute inset-0 z-[41] flex animate-fade-in cursor-pointer items-center justify-center overflow-hidden bg-surface/50 px-4 backdrop-blur-2xl sm:px-8" aria-label="Continue watching" >

            {!simplified && (

              <p className="pointer-events-none absolute inset-x-0 top-24 z-10 text-center text-xs tracking-wide text-foreground/40 sm:bottom-28 sm:text-sm">

                Breaks are Auto-Detected

              </p>

            )}

            <div className="pointer-events-none absolute inset-0 bg-gradient-to-t from-black/70 via-black/35 to-black/20" />

            <div className="pointer-events-none relative z-10 flex w-full justify-center">

              <div className={cn(

                  "flex animate-fade-in flex-col items-center",
                  simplified
                    ? "w-fit max-w-2xl gap-4 px-2 text-center landscape:flex-row landscape:items-center landscape:gap-6 landscape:text-left"
                    : "w-fit max-w-5xl gap-4 md:flex-row md:items-center md:gap-7 md:text-left"

                )}

              >

                {showPoster && (

                  <div className="flex-shrink-0">

                    <img
                      src={poster}
                      alt=""
                      className={simplified ? "h-24 w-24 object-contain landscape:h-28 landscape:w-28" : "h-28 w-28 object-contain sm:h-36 sm:w-36"}
                      onError={this.handlePosterError}
                    />

                  </div>

                )}

                <div className={simplified ? "space-y-1.5" : "space-y-1.5"}>

                  <h2 className={simplified ? "text-2xl font-semibold tracking-tight landscape:text-3xl" : "text-2xl font-semibold tracking-tight sm:text-3xl"}>

                    Commercial Break

                  </h2>

                  <p className={cn(

                      "font-medium tracking-wide text-foreground-muted uppercase",
                      simplified ? "text-xs" : "text-sm"

                    )}

                  >

                    {title} <span className="opacity-50">/</span> {subtitle}

                  </p>

                </div>

              </div>

            </div>

            {!simplified && (

              <p className="pointer-events-none absolute inset-x-0 bottom-24 z-10 text-center text-xs tracking-wide text-foreground/40 sm:bottom-28 sm:text-sm">

                Click Anywhere to Hide

              </p>

            )}

          </button>

        )}
      </>

    );

  }

}
