import { CachedImage } from "@/UI/CachedImage";

import { cn } from "@/Utils/ClassNames";

import type { FeedItem } from "@/Types";

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
  direction: 1 | -1;

}

const ROTATE_MS = 8000;

const slideTransition = { duration: 0.45, ease: [0.32, 0.72, 0, 1] as const };

const slideVariants = {

  enter: (dir: number) => ({ x: dir > 0 ? "100%" : "-100%" }),
  center: { x: 0 },
  exit: (dir: number) => ({ x: dir > 0 ? "-100%" : "100%" }),

};

export class FeaturedHero extends Component<FeaturedHeroProps, FeaturedHeroState> {

  state: FeaturedHeroState = { index: 0, direction: 1 };

  private timer: ReturnType<typeof setInterval> | null = null;

  componentDidMount() {

    this.startTimer();

  }

  componentDidUpdate(prevProps: FeaturedHeroProps) {

    if (prevProps.items !== this.props.items) {

      this.setState({ index: 0, direction: 1 });
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
        direction: 1,

      }));

    }, ROTATE_MS);

  };

  go = (dir: -1 | 1) => {

    const { items } = this.props;

    if (items.length < 2) return;

    this.setState((state) => ({

      index: (state.index + dir + items.length) % items.length,
      direction: dir,

    }));

    this.startTimer();

  };

  goTo = (nextIndex: number) => {

    this.setState((state) => {

      if (nextIndex === state.index) return null;

      return {

        index: nextIndex,
        direction: nextIndex > state.index ? 1 : -1,

      };

    });

    this.startTimer();

  };

  render() {

    const { items, favoriteIds, onPlay, onFavoriteToggle } = this.props;

    if (items.length === 0) return null;

    const index = this.state.index % items.length;
    const { direction } = this.state;
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

        <div className="relative overflow-hidden rounded-xl border border-border-subtle">

          <AnimatePresence initial={false} custom={direction} mode="popLayout">

            <motion.div

              key={`${item.kind}-${item.tmdbId || item.id}`}
              custom={direction}
              variants={slideVariants}
              initial="enter"
              animate="center"
              exit="exit"
              transition={slideTransition}
              className="relative flex min-h-[280px] flex-col justify-end gap-4 p-6 sm:min-h-[320px] sm:p-8"

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

              <div className="relative flex max-w-xl flex-col gap-3">

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

                <div className="mt-1 flex flex-wrap items-center gap-1.5">

                  <button

                    type="button"
                    onClick={() => onPlay(item)}
                    className="flex h-9 items-center gap-2 rounded-md bg-white px-5 text-sm font-semibold text-black transition-opacity hover:opacity-90"

                  >

                    <Play className="size-3.5 fill-current" />
                    Play

                  </button>

                  <button

                    type="button"
                    onClick={() => onFavoriteToggle(item)}
                    className={cn(

                      "flex size-9 items-center justify-center rounded-md border border-white/20 bg-white/10 text-white backdrop-blur-md transition-colors hover:bg-white/15",
                      favorite && "text-accent"

                    )}
                    aria-label={favorite ? "Remove from favorites" : "Add to favorites"}

                  >

                    <Star className="size-3.5" fill={favorite ? "currentColor" : "none"} />

                  </button>

                </div>

              </div>

            </motion.div>

          </AnimatePresence>

          {items.length > 1 && (

            <div className="absolute right-4 top-4 z-10 flex items-center gap-1 sm:right-6 sm:top-6">

              <button

                type="button"
                onClick={() => this.go(-1)}
                className="flex size-8 items-center justify-center rounded-md border border-white/15 bg-black/30 text-white/80 backdrop-blur-md transition-colors hover:bg-black/50 hover:text-white"
                aria-label="Previous featured"

              >

                <ChevronLeft className="size-4" />

              </button>

              <button

                type="button"
                onClick={() => this.go(1)}
                className="flex size-8 items-center justify-center rounded-md border border-white/15 bg-black/30 text-white/80 backdrop-blur-md transition-colors hover:bg-black/50 hover:text-white"
                aria-label="Next featured"

              >

                <ChevronRight className="size-4" />

              </button>

            </div>

          )}

          {items.length > 1 && (

            <div className="absolute bottom-4 right-4 z-10 flex gap-1.5 sm:bottom-6 sm:right-6">

              {items.map((entry, i) => (

                <button

                  key={entry.tmdbId || entry.id}
                  type="button"
                  aria-label={`Show featured ${i + 1}`}
                  onClick={() => this.goTo(i)}
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

    );

  }

}
