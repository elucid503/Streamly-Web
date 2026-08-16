import { Component, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { X } from "lucide-react";

interface PlayerMobileSheetProps {

  open: boolean;

  title: string;
  icon?: ReactNode;

  onClose: () => void;

  children: ReactNode;

  headerExtra?: ReactNode;

  /** `center` for short lists (quality/source). `fill` for episode grids. */
  layout?: "center" | "fill";

}

export class PlayerMobileSheet extends Component<PlayerMobileSheetProps> {

  render() {

    const { open, title, icon, onClose, children, headerExtra, layout = "center" } = this.props;

    if (!open || typeof document === "undefined") return null;

    const fill = layout === "fill";

    return createPortal(

      <div className={fill
          ? "fixed inset-0 z-[90] flex animate-fade-in flex-col bg-surface"
          : "fixed inset-0 z-[90] overflow-y-auto bg-surface animate-fade-in"} style={{ height: "var(--player-vvh, 100dvh)" }} onClick={(e) => {

          e.stopPropagation();

          onClose();

        }}

      >

        {fill ? (

          <div className="flex min-h-0 w-full flex-1 flex-col px-5 pt-[max(0.75rem,env(safe-area-inset-top,0px))] pb-[max(0.75rem,env(safe-area-inset-bottom,0px))] landscape:px-8" onClick={(e) => e.stopPropagation()} >

            <div className="flex shrink-0 items-center justify-between gap-3 pb-3">

              <div className="flex min-w-0 items-center gap-2.5">

                {icon}

                <p className="truncate text-lg font-semibold text-foreground">

                  {title}

                </p>

              </div>

              <button type="button" onClick={(e) => {

                  e.stopPropagation();

                  onClose();

                }} className="flex size-11 shrink-0 items-center justify-center rounded-lg text-foreground-muted transition-colors hover:bg-white/10 hover:text-foreground" aria-label={`Close ${title}`} >

                <X size={22} />

              </button>

            </div>

            {headerExtra && (

              <div className="shrink-0 pb-5">

                {headerExtra}

              </div>

            )}

            <div className="min-h-0 flex-1 overflow-y-auto">

              {children}

            </div>

          </div>

        ) : (

          <div className="flex min-h-full items-center justify-center px-5 py-[max(0.75rem,env(safe-area-inset-top,0px))]" onClick={(e) => e.stopPropagation()} >

            <div className="flex w-full max-w-md flex-col">

              <div className="flex items-center justify-between gap-3 pb-4">

                <div className="flex min-w-0 items-center gap-2.5">

                  {icon}

                  <p className="truncate text-lg font-semibold text-foreground">

                    {title}

                  </p>

                </div>

                <button type="button" onClick={(e) => {

                    e.stopPropagation();

                    onClose();

                  }} className="flex size-11 shrink-0 items-center justify-center rounded-lg text-foreground-muted transition-colors hover:bg-white/10 hover:text-foreground" aria-label={`Close ${title}`} >

                  <X size={22} />

                </button>

              </div>

              {headerExtra && (

                <div className="pb-4">

                  {headerExtra}

                </div>

              )}

              <div>

                {children}

              </div>

            </div>

          </div>

        )}

      </div>,

      document.body,

    );

  }

}
