import { Component } from "react";
import { motion } from "framer-motion";
import { PosterImage } from "@/components/catalog/PosterImage";
import { cn, progressPercent } from "@/lib/utils";

interface TitleCardProps {
  id: number;
  kind: "movie" | "show";
  title: string;
  poster?: string;
  year?: number | string;
  onClick: () => void;
  compact?: boolean;
  progressMs?: number;
  durationMs?: number;
  progressLabel?: string;
}

export class TitleCard extends Component<TitleCardProps> {
  render() {
    const { title, poster, year, onClick, compact, progressMs, durationMs, progressLabel } =
      this.props;
    const progress = progressPercent(progressMs, durationMs);

    return (
      <motion.button
        type="button"
        onClick={onClick}
        className={cn(
          "group flex-shrink-0 text-left",
          compact ? "w-[120px] sm:w-[140px]" : "w-[140px] sm:w-[160px]",
        )}
        whileHover={{ y: -4 }}
        transition={{ type: "spring", stiffness: 400, damping: 25 }}
      >
        <div className="relative overflow-hidden rounded-md border border-border-subtle bg-surface-raised transition-colors group-hover:border-border">
          <PosterImage
            src={poster}
            alt={title}
            className={cn("w-full", compact ? "aspect-[2/3] h-auto" : "aspect-[2/3]")}
          />
          {progress > 2 && (
            <div className="absolute inset-x-2 bottom-2 h-1 overflow-hidden rounded-full bg-black/45 backdrop-blur-sm">
              <motion.div
                className="h-full rounded-full bg-foreground"
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
      </motion.button>
    );
  }
}
