import { PosterImage } from "@/components/catalog/PosterImage";
import { PlayOverlay } from "@/components/ui/PlayOverlay";

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
  rating?: string;
  genres?: string[];

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

    const { title, poster, year, rating, genres, onClick, onFavoriteToggle, onResume, onRemoveFromHistory, compact, favorite, progressMs, durationMs, progressLabel } = this.props;
    const { menuPos } = this.state;

    const progress = progressPercent(progressMs, durationMs);
    const genreLabel = genres?.slice(0, 1)[0];

    return (

      <>

        <motion.div className={cn(

            "group relative flex-shrink-0 text-left",
            compact ? "w-[8.5rem] sm:w-32" : "w-[8.5rem] sm:w-36"

          )}

        >

          <button className="block w-full text-left" type="button" onClick={onClick}>

            <div className="relative overflow-hidden rounded-lg border border-border-subtle bg-surface-raised transition-[filter,border-color] duration-300 group-hover:border-border group-hover:brightness-110">

              <PosterImage

                src={poster}
                alt={title}
                className={cn("w-full", compact ? "aspect-[2/3] h-auto" : "aspect-[2/3]")}

              />

              <PlayOverlay />

              {rating && (

                <span className="absolute right-1.5 top-1.5 z-20 rounded-md border border-border-subtle bg-surface/80 px-1.5 py-0.5 text-[10px] font-medium text-foreground backdrop-blur-md">

                  ★ {rating}

                </span>

              )}

              {progress > 2 && (

                <div className="absolute inset-x-2 bottom-2 z-20 h-1 overflow-hidden rounded-full bg-black/45 backdrop-blur-sm">

                  <motion.div className="h-full rounded-full bg-foreground"

                    initial={{ width: 0 }}
                    animate={{ width: `${progress}%` }}
                    transition={{ duration: 0.35, ease: "easeOut" }}

                  />

                </div>

              )}

            </div>

            <p className="mt-2 line-clamp-2 text-sm font-medium text-foreground transition-colors group-hover:text-accent">

              {title}

            </p>

            {progressLabel ? (

              <p className="text-xs text-foreground-muted">{progressLabel}</p>

            ) : (

              <p className="text-xs text-foreground-muted">

                {[year, genreLabel].filter(Boolean).join(" · ")}

              </p>

            )}

          </button>

          {(onFavoriteToggle || onResume || onRemoveFromHistory) && (

            <div className="absolute right-1.5 top-1.5 z-10 flex gap-1">

              <button

                className={cn(

                  "flex size-8 items-center justify-center rounded-md border border-border-subtle bg-surface/80 text-foreground shadow-sm backdrop-blur-md transition-colors hover:bg-surface-overlay",
                  menuPos && "bg-surface-overlay"

                )}

                type="button"
                aria-label="More options"
                onClick={this.openMenu}

              >

                <MoreHorizontal className="size-4" />

              </button>

            </div>

          )}

        </motion.div>

        {menuPos && createPortal(

          <>

            <div className="fixed inset-0 z-[99]" onClick={this.closeMenu} />

            <motion.div

              className="fixed z-[100] min-w-[172px] overflow-hidden rounded-lg border border-border-subtle bg-surface/95 p-1 shadow-2xl shadow-black/40 ring-1 ring-white/[0.04] backdrop-blur-xl backdrop-saturate-150"

              style={{ top: menuPos.top, left: menuPos.left }}

              initial={{ opacity: 0, scale: 0.96, y: -6 }}
              animate={{ opacity: 1, scale: 1, y: 0 }}
              transition={{ type: "spring", stiffness: 500, damping: 32 }}

            >

              {onResume && (

                <button className="flex h-9 w-full items-center gap-2 rounded-md px-3 text-left text-xs font-medium text-foreground-muted transition-colors hover:bg-surface-overlay/80 hover:text-foreground" type="button"

                  onClick={(e) => {

                    e.stopPropagation();

                    this.closeMenu();
                    onResume();

                  }}

                >

                  <Play className="size-3.5" />
                  <span>Resume Watching</span>

                </button>

              )}

              {onRemoveFromHistory && (

                <button type="button" className="flex h-9 w-full items-center gap-2 rounded-md px-3 text-left text-xs font-medium text-foreground-muted transition-colors hover:bg-surface-overlay/80 hover:text-foreground"

                  onClick={(e) => {

                    e.stopPropagation();

                    this.closeMenu();
                    onRemoveFromHistory();

                  }}

                >

                  <Trash2 className="size-3.5" />
                  <span>Remove from History</span>

                </button>

              )}

              {onFavoriteToggle && (

                <button type="button" className="flex h-9 w-full items-center gap-2 rounded-md px-3 text-left text-xs font-medium text-foreground-muted transition-colors hover:bg-surface-overlay/80 hover:text-foreground"

                  onClick={(e) => {

                    e.stopPropagation();

                    this.closeMenu();
                    onFavoriteToggle();

                  }}

                >

                  <Star className={cn("size-3.5", favorite && "text-accent")} fill={favorite ? "currentColor" : "none"} />
                  <span>{favorite ? "Remove from Favorites" : "Add to Favorites"}</span>

                </button>

              )}

              <button type="button" className="flex h-9 w-full items-center gap-2 rounded-md px-3 text-left text-xs font-medium text-foreground-muted transition-colors hover:bg-surface-overlay/80 hover:text-foreground" onClick={this.copyLink}>

                <Link className="size-3.5" />
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
