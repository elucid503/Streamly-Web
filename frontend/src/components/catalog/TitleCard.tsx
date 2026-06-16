import { PosterImage } from "@/components/catalog/PosterImage";

import { cn, progressPercent } from "@/lib/utils";

import { Component } from "react";
import { motion } from "framer-motion";
import { Star } from "lucide-react";

interface TitleCardProps {

  id: number;
  kind: "movie" | "show";

  title: string;
  poster?: string;
  year?: number | string;

  onClick: () => void;
  onFavoriteToggle?: () => void;

  compact?: boolean;
  favorite?: boolean;

  progressMs?: number;
  durationMs?: number;
  progressLabel?: string;

}

export class TitleCard extends Component<TitleCardProps> {

  render() {

    const { title, poster, year, onClick, onFavoriteToggle, compact, favorite, progressMs, durationMs, progressLabel } =
      this.props;

    const progress = progressPercent(progressMs, durationMs);

    return (

      <motion.div className={cn(

          "group relative flex-shrink-0 text-left",
          compact ? "w-[120px] sm:w-[140px]" : "w-[140px] sm:w-[160px]"

        )}
        whileHover={{ y: -4 }}
        transition={{ type: "spring", stiffness: 400, damping: 25 }}

      >

        <button className="block w-full text-left" type="button" onClick={onClick}>

          <div className="relative overflow-hidden rounded-md border border-border-subtle bg-surface-raised transition-colors group-hover:border-border">

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

        {onFavoriteToggle && (

          <button className={cn(

              "absolute right-2 top-2 z-10 flex h-8 w-8 items-center justify-center rounded-full border border-border-subtle bg-surface/80 text-foreground shadow-sm backdrop-blur-md transition-colors hover:bg-surface-overlay",
              favorite && "text-accent"

            )}

            type="button"
            title={favorite ? "Remove from favorites" : "Add to favorites"}
            onClick={(event) => {

              event.stopPropagation();
              onFavoriteToggle();

            }}

          >

            <Star size={15} fill={favorite ? "currentColor" : "none"} />

          </button>

        )}

      </motion.div>

    );

  }

}
