import { PosterImage } from "@/components/catalog/PosterImage";

import { cn, progressPercent } from "@/lib/utils";

import { Component } from "react";
import { createPortal } from "react-dom";
import { motion } from "framer-motion";
import { MoreHorizontal, Star, Play, Link, Trash2 } from "lucide-react";

interface TitleCardProps {

  id: number;
  kind: "movie" | "show";

  title: string;
  poster?: string;
  year?: number | string;

  onClick: () => void;
  onFavoriteToggle?: () => void;
  onResume?: () => void;
  onRemoveFromHistory?: () => void;

  compact?: boolean;
  favorite?: boolean;

  progressMs?: number;
  durationMs?: number;
  progressLabel?: string;

}

interface TitleCardState {

  menuPos: { top: number; left: number } | null;

}

export class TitleCard extends Component<TitleCardProps, TitleCardState> {

  state: TitleCardState = { menuPos: null };

  copyLink = async (e: React.MouseEvent) => {

    e.stopPropagation();

    const { id, kind } = this.props;
    const url = `${window.location.origin}/${kind}/${id}`;

    await navigator.clipboard.writeText(url).catch(() => {});

    this.setState({ menuPos: null });

  };

  openMenu = (e: React.MouseEvent<HTMLButtonElement>) => {

    e.stopPropagation();

    if (this.state.menuPos) {

      this.setState({ menuPos: null });

      return;

    }

    const rect = e.currentTarget.getBoundingClientRect();
    const menuWidth = 172;

    this.setState({

      menuPos: {
        top: rect.bottom + 6,
        left: Math.min(rect.left, window.innerWidth - menuWidth - 8),
      },

    });

  };

  closeMenu = () => this.setState({ menuPos: null });

  render() {

    const { title, poster, year, onClick, onFavoriteToggle, onResume, onRemoveFromHistory, compact, favorite, progressMs, durationMs, progressLabel } = this.props;
    const { menuPos } = this.state;

    const progress = progressPercent(progressMs, durationMs);

    return (

      <>

        <motion.div className={cn(

            "group relative flex-shrink-0 text-left",
            compact ? "w-[120px] sm:w-[140px]" : "w-[140px] sm:w-[160px]"

          )}

        >

          <button className="block w-full text-left" type="button" onClick={onClick}>

            <div className="relative overflow-hidden rounded-md border border-border-subtle bg-surface-raised transition-[filter,border-color] duration-300 group-hover:border-border group-hover:brightness-[1.15]">

              <PosterImage

                src={poster}
                alt={title}
                className={cn("w-full", compact ? "aspect-[2/3] h-auto" : "aspect-[2/3]")}

              />

              {progress > 2 && (

                <div className="absolute inset-x-2 bottom-2 h-1 overflow-hidden rounded-full bg-black/45 backdrop-blur-sm">

                  <motion.div className="h-full rounded-full bg-foreground"

                    initial={{ width: 0 }}
                    animate={{ width: `${progress}%` }}
                    transition={{ duration: 0.35, ease: "easeOut" }}

                  />

                </div>

              )}

            </div>

            <p className="mt-2 line-clamp-2 text-xs font-medium text-foreground transition-colors group-hover:text-accent">

              {title}

            </p>

            {progressLabel ? (

              <p className="text-xs text-foreground-muted">{progressLabel}</p>

            ) : year ? (

              <p className="text-xs text-foreground-muted">{year}</p>

            ) : null}

          </button>

          {(onFavoriteToggle || onResume) && (

            <div className="absolute right-2 top-2 z-10 flex gap-1">

              {onResume && (

                <button

                  className="flex h-8 w-8 items-center justify-center rounded-full border border-border-subtle bg-surface/80 text-foreground shadow-sm backdrop-blur-md transition-colors hover:bg-surface-overlay"

                  type="button"
                  aria-label="Quick resume"

                  onClick={(e) => {

                    e.stopPropagation();

                    onResume();

                  }}

                >

                  <Play size={13} />

                </button>

              )}

              {onFavoriteToggle && (

                <button

                  className={cn(

                    "flex h-8 w-8 items-center justify-center rounded-full border border-border-subtle bg-surface/80 text-foreground shadow-sm backdrop-blur-md transition-colors hover:bg-surface-overlay",
                    menuPos && "bg-surface-overlay"

                  )}

                  type="button"
                  aria-label="More options"
                  onClick={this.openMenu}

                >

                  <MoreHorizontal size={15} />

                </button>

              )}

            </div>

          )}

        </motion.div>

        {menuPos && createPortal(

          <>

            <div className="fixed inset-0 z-[99]" onClick={this.closeMenu} />

            <motion.div

              className="fixed z-[100] min-w-[172px] overflow-hidden rounded-[1.25rem] border border-white/10 bg-surface/70 p-1 shadow-2xl shadow-black/40 ring-1 ring-white/[0.04] backdrop-blur-xl backdrop-saturate-150"

              style={{ top: menuPos.top, left: menuPos.left }}

              initial={{ opacity: 0, scale: 0.96, y: -6 }}
              animate={{ opacity: 1, scale: 1, y: 0 }}
              transition={{ type: "spring", stiffness: 500, damping: 32 }}

            >

              {onResume && (

                <button className="flex h-9 w-full items-center gap-2 rounded-xl px-3 text-left text-xs font-medium text-foreground-muted transition-colors hover:bg-surface-overlay/80 hover:text-foreground" type="button"

                  onClick={(e) => {

                    e.stopPropagation();

                    this.closeMenu();
                    onResume();

                  }}

                >

                  <Play size={13} />
                  <span>Resume Watching</span>

                </button>

              )}

              {onRemoveFromHistory && (

                <button type="button" className="flex h-9 w-full items-center gap-2 rounded-xl px-3 text-left text-xs font-medium text-foreground-muted transition-colors hover:bg-surface-overlay/80 hover:text-foreground"

                  onClick={(e) => {

                    e.stopPropagation();

                    this.closeMenu();
                    onRemoveFromHistory();

                  }}

                >

                  <Trash2 size={13} />
                  <span>Remove from History</span>

                </button>

              )}

              {onFavoriteToggle && (

                <button type="button" className="flex h-9 w-full items-center gap-2 rounded-xl px-3 text-left text-xs font-medium text-foreground-muted transition-colors hover:bg-surface-overlay/80 hover:text-foreground"

                  onClick={(e) => {

                    e.stopPropagation();

                    this.closeMenu();
                    onFavoriteToggle();

                  }}

                >

                  <Star size={13} fill={favorite ? "currentColor" : "none"} className={cn(favorite && "text-accent")} />
                  <span>{favorite ? "Remove from Favorites" : "Add to Favorites"}</span>

                </button>

              )}

              <button type="button" className="flex h-9 w-full items-center gap-2 rounded-xl px-3 text-left text-xs font-medium text-foreground-muted transition-colors hover:bg-surface-overlay/80 hover:text-foreground" onClick={this.copyLink}>

                <Link size={13} />
                <span>Copy Link</span>

              </button>

            </motion.div>

          </>,

          document.body

        )}

      </>

    );

  }

}
