import { CachedImage } from "@/components/ui/CachedImage";

import { cn } from "@/lib/utils";

import type { FeedItem } from "@/lib/types";

import { Component } from "react";
import { AnimatePresence, motion } from "framer-motion";
import { Play, Star, ChevronLeft, ChevronRight } from "lucide-react";

interface FeaturedHeroProps {

  items: FeedItem[];
  favoriteIds: Set<number>;

  onPlay: (item: FeedItem) => void;
  onFavoriteToggle: (item: FeedItem) => void;

}

interface FeaturedHeroState {

  index: number;

}

const ROTATE_MS = 8000;

export class FeaturedHero extends Component<FeaturedHeroProps, FeaturedHeroState> {

  state: FeaturedHeroState = { index: 0 };

  private timer: ReturnType<typeof setInterval> | null = null;

  componentDidMount() {

    this.startTimer();

  }

  componentDidUpdate(prevProps: FeaturedHeroProps) {

    if (prevProps.items !== this.props.items) {

      this.setState({ index: 0 });
      this.startTimer();

    }

  }

  componentWillUnmount() {

    this.clearTimer();

  }

  clearTimer = () => {

    if (this.timer) {

      clearInterval(this.timer);
      this.timer = null;

    }

  };

  startTimer = () => {

    this.clearTimer();

    if (this.props.items.length < 2) return;

    this.timer = setInterval(() => {

      this.setState((state) => ({

        index: (state.index + 1) % this.props.items.length,

      }));

    }, ROTATE_MS);

  };

  go = (dir: -1 | 1) => {

    const { items } = this.props;

    if (items.length < 2) return;

    this.setState((state) => ({

      index: (state.index + dir + items.length) % items.length,

    }));

    this.startTimer();

  };

  render() {

    const { items, favoriteIds, onPlay, onFavoriteToggle } = this.props;

    if (items.length === 0) return null;

    const index = this.state.index % items.length;
    const item = items[index];
    const favorite = item.id > 0 && favoriteIds.has(item.id);

    const meta = [

      item.year || null,
      item.rating ? `★ ${item.rating}` : null,
      item.genres?.slice(0, 2).join(", ") || null,

    ].filter(Boolean).join(" · ");

    const art = item.backdrop || item.poster;

    return (

      <div className="mb-8 px-4 sm:px-8">

        <div className="relative overflow-hidden rounded-xl">

          <AnimatePresence mode="wait">

            <motion.div

              key={item.tmdbId || item.id}
              className="absolute inset-0"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              transition={{ duration: 0.45 }}

            >

              {art ? (

                <CachedImage

                  src={art}
                  alt=""
                  rounded="rounded-none"
                  className="absolute inset-0 h-full w-full"
                  imgClassName="object-cover"

                />

              ) : (

                <div className="absolute inset-0 bg-surface-raised" />

              )}

              <div className="absolute inset-0 bg-black/55" />

              <div className="absolute inset-0 bg-gradient-to-t from-surface via-surface/70 to-transparent" />

            </motion.div>

          </AnimatePresence>

          <div className="relative flex min-h-[280px] flex-col justify-end gap-4 p-6 sm:min-h-[320px] sm:p-8">

            <div className="flex max-w-xl flex-col gap-3">

              {item.matchReason && (

                <p className="text-xs font-medium text-foreground-muted">

                  {item.matchReason}

                </p>

              )}

              <h2 className="text-2xl font-bold text-white sm:text-3xl">

                {item.title}

              </h2>

              {meta && (

                <p className="text-sm text-white/70">

                  {meta}

                </p>

              )}

              {item.description && (

                <p className="line-clamp-2 text-sm text-white/60">

                  {item.description}

                </p>

              )}

              <div className="mt-1 flex flex-wrap items-center gap-2">

                <button

                  type="button"
                  onClick={() => onPlay(item)}
                  className="flex h-9 items-center gap-2 rounded-full bg-white px-5 text-sm font-semibold text-black transition-opacity hover:opacity-90"

                >

                  <Play size={14} fill="currentColor" />
                  Play

                </button>

                <button

                  type="button"
                  onClick={() => onFavoriteToggle(item)}
                  className={cn(

                    "flex h-9 w-9 items-center justify-center rounded-full border border-white/20 bg-white/10 text-white backdrop-blur-md transition-colors hover:bg-white/15",
                    favorite && "text-accent"

                  )}
                  aria-label={favorite ? "Remove from favorites" : "Add to favorites"}

                >

                  <Star size={14} fill={favorite ? "currentColor" : "none"} />

                </button>

              </div>

            </div>

            {items.length > 1 && (

              <div className="absolute right-4 top-4 flex items-center gap-1 sm:right-6 sm:top-6">

                <button

                  type="button"
                  onClick={() => this.go(-1)}
                  className="flex h-8 w-8 items-center justify-center rounded-full border border-white/15 bg-black/30 text-white/80 backdrop-blur-md transition-colors hover:bg-black/50 hover:text-white"
                  aria-label="Previous featured"

                >

                  <ChevronLeft size={16} />

                </button>

                <button

                  type="button"
                  onClick={() => this.go(1)}
                  className="flex h-8 w-8 items-center justify-center rounded-full border border-white/15 bg-black/30 text-white/80 backdrop-blur-md transition-colors hover:bg-black/50 hover:text-white"
                  aria-label="Next featured"

                >

                  <ChevronRight size={16} />

                </button>

              </div>

            )}

            {items.length > 1 && (

              <div className="absolute bottom-4 right-4 flex gap-1.5 sm:bottom-6 sm:right-6">

                {items.map((entry, i) => (

                  <button

                    key={entry.tmdbId || entry.id}
                    type="button"
                    aria-label={`Show featured ${i + 1}`}
                    onClick={() => {

                      this.setState({ index: i });
                      this.startTimer();

                    }}
                    className={cn(

                      "h-1.5 rounded-full transition-all",
                      i === index ? "w-5 bg-white" : "w-1.5 bg-white/35 hover:bg-white/55"

                    )}

                  />

                ))}

              </div>

            )}

          </div>

        </div>

      </div>

    );

  }

}
