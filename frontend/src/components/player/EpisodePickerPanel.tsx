import { Button } from "@/components/ui/Button";
import { CachedImage } from "@/components/ui/CachedImage";
import { Modal } from "@/components/ui/Modal";

import { cn } from "@/lib/utils";
import type { Episode, Season } from "@/lib/types";

import { Component, createRef } from "react";
import { ChevronLeft, ChevronRight, Clapperboard, Film, Play, X } from "lucide-react";

interface EpisodePickerPanelProps {

  open: boolean;

  seasons: Season[];
  episodes: Episode[];

  currentSeason?: number;
  currentEpisode?: number;
  menuSeason: number;

  episodesLoading?: boolean;

  onClose: () => void;
  onSeasonChange: (season: number) => void;
  onEpisodeSelect: (season: number, episode: number) => void;

}

interface EpisodePickerPanelState {

  detailEpisode: Episode | null;

  canScrollLeft: boolean;
  canScrollRight: boolean;

}

export class EpisodePickerPanel extends Component<EpisodePickerPanelProps, EpisodePickerPanelState> {

  state: EpisodePickerPanelState = {

    detailEpisode: null,

    canScrollLeft: false,
    canScrollRight: false,

  };

  private carouselRef = createRef<HTMLDivElement>();

  private cardRefs = new Map<string, HTMLDivElement>();

  componentDidUpdate(prev: EpisodePickerPanelProps) {

    if (prev.menuSeason !== this.props.menuSeason) {

      this.setState({ detailEpisode: null });

    }

    if (this.props.open && !prev.open) {

      this.setState({ detailEpisode: null });

      requestAnimationFrame(() => {

        this.scrollToCurrent();

        this.updateScrollButtons();

      });

    }

    if (!this.props.open && prev.open) {

      this.setState({ detailEpisode: null });

    }

    if ( this.props.open && (!prev.open || prev.menuSeason !== this.props.menuSeason || prev.episodesLoading !== this.props.episodesLoading || prev.episodes !== this.props.episodes)) {

      if (!this.props.episodesLoading) {

        requestAnimationFrame(() => {

          this.scrollToCurrent();
          this.updateScrollButtons();

        });

      }

    }

  }

  updateScrollButtons = () => {

    const carousel = this.carouselRef.current;

    if (!carousel) return;

    const { scrollLeft, scrollWidth, clientWidth } = carousel;

    const maxScroll = scrollWidth - clientWidth;

    this.setState({

      canScrollLeft: scrollLeft > 4,
      canScrollRight: maxScroll > 4 && scrollLeft < maxScroll - 4,

    });

  };

  scrollCarousel = (direction: -1 | 1) => {

    const carousel = this.carouselRef.current;

    if (!carousel) return;

    const card = carousel.querySelector<HTMLElement>("[data-episode-card]");

    const stride = card ? card.offsetWidth + 16 : carousel.clientWidth * 0.8;

    carousel.scrollBy({ left: direction * stride, behavior: "smooth" });

  };

  scrollToCurrent = () => {

    const { currentSeason, currentEpisode } = this.props;

    if (!currentSeason || !currentEpisode) return;

    const key = `${currentSeason}-${currentEpisode}`;

    const card = this.cardRefs.get(key);
    const carousel = this.carouselRef.current;

    if (!card || !carousel) return;

    const offset = card.offsetLeft - carousel.clientWidth / 2 + card.offsetWidth / 2;

    carousel.scrollTo({ left: Math.max(0, offset), behavior: "smooth" });

    requestAnimationFrame(() => this.updateScrollButtons());

  };

  setCardRef = (key: string, node: HTMLDivElement | null) => {

    if (node) this.cardRefs.set(key, node);

    else this.cardRefs.delete(key);

  };

  openDetail = (ep: Episode) => {

    this.setState({ detailEpisode: ep });

  };

  closeDetail = () => {

    this.setState({ detailEpisode: null });

  };

  playFromDetail = () => {

    const { detailEpisode } = this.state;

    if (!detailEpisode) return;

    this.props.onEpisodeSelect(detailEpisode.season, detailEpisode.episode);

    this.setState({ detailEpisode: null });

  };

  renderEpisodeThumbnail = (ep: Episode, className?: string) => (

    <div className={cn("relative overflow-hidden bg-white/5", className)}>

      {ep.poster ? (

        <CachedImage className="size-full"

          src={ep.poster}
          alt={ep.title}

          imgClassName="object-cover object-center"
          rounded="rounded-none"

          fallback={

            <span className="flex size-full items-center justify-center text-foreground-faint">

              <Film size={28} strokeWidth={1.5} />

            </span>

          }

        />

      ) : (

        <span className="flex size-full items-center justify-center text-foreground-faint">

          <Film size={28} strokeWidth={1.5} />

        </span>

      )}

    </div>

  );

  renderEpisodeCard = (ep: Episode) => {

    const { currentSeason, currentEpisode, onEpisodeSelect } = this.props;

    const key = `${ep.season}-${ep.episode}`;

    const active = currentSeason === ep.season && currentEpisode === ep.episode;

    const description = ep.description?.trim() ?? "";

    return (

      <div className="flex w-[min(78vw,16.5rem)] flex-shrink-0 snap-start sm:w-64"

        key={key}
        data-episode-card
        ref={(node) => this.setCardRef(key, node)}

      >
        <button type="button" onClick={(e) => {

            e.stopPropagation();

            onEpisodeSelect(ep.season, ep.episode);

          }} className={cn(

              "group flex h-full w-full flex-col overflow-hidden rounded-xl border text-left transition-colors",
              active ? "border-foreground bg-white/10 ring-1 ring-foreground/30" : "border-white/10 bg-white/5 hover:border-white/20 hover:bg-white/8"

          )}

        >

          <div className="relative aspect-[2/1] w-full shrink-0 overflow-hidden bg-white/5">

            {this.renderEpisodeThumbnail(ep, "size-full")}

            <span className="absolute top-2 left-2 rounded-md bg-black/70 px-2 py-0.5 text-[10px] font-medium tracking-wide text-foreground backdrop-blur-sm">

              E{ep.episode}

            </span>

          </div>

          <div className="flex flex-1 flex-col p-3">

            <p className="line-clamp-2 shrink-0 text-sm font-medium leading-snug text-foreground">

              {ep.title}

            </p>

            <div className="mt-0.5 flex min-h-[3.25rem] flex-1 flex-col justify-between gap-1">

              <p className={cn(
                  "line-clamp-2 text-xs leading-relaxed",
                  description ? "text-foreground-muted" : "text-foreground-faint"
                )}
              >

                {description || "No description available"}

              </p>

              <span role="button" tabIndex={0} onClick={(e) => {

                  e.stopPropagation();

                  this.openDetail(ep);

                }} onKeyDown={(e) => {

                  if (e.key === "Enter" || e.key === " ") {

                    e.preventDefault();
                    e.stopPropagation();

                    this.openDetail(ep);

                  }

                }} className="text-[11px] font-medium text-foreground/80 underline-offset-2 hover:text-foreground hover:underline" >

                Show more

              </span>

            </div>

          </div>

        </button>

      </div>

    );

  };

  renderDetailModal = () => {

    const { detailEpisode } = this.state;

    const { currentSeason, currentEpisode } = this.props;

    if (!detailEpisode) return null;

    const description = detailEpisode.description?.trim() ?? "";

    const isCurrent = currentSeason === detailEpisode.season && currentEpisode === detailEpisode.episode;

    return (

      <Modal open onClose={this.closeDetail} title={detailEpisode.title} className="max-w-lg">

        <div className="space-y-4">

          {this.renderEpisodeThumbnail(

            detailEpisode,
            "aspect-video w-full overflow-hidden rounded-lg"

          )}

          <p className="text-xs font-medium tracking-wide text-foreground-muted uppercase">

            Season {detailEpisode.season}, Episode {detailEpisode.episode}

          </p>

          {description ? (

            <p className="text-sm leading-relaxed text-foreground-muted">

              {description}

            </p>

          ) : (

            <p className="text-sm text-foreground-faint">

              No description available

            </p>

          )}

          <Button className="w-full" onClick={this.playFromDetail} disabled={isCurrent}>

            <Play size={14} />

            {isCurrent ? "Now playing" : "Play episode"}

          </Button>

        </div>

      </Modal>

    );

  };

  render() {

    const { open, seasons, episodes, menuSeason, episodesLoading, onClose, onSeasonChange } = this.props;

    const { canScrollLeft, canScrollRight } = this.state;

    if (!open) return null;

    return (

      <>
        <div className="w-full animate-fade-in" onClick={(e) => e.stopPropagation()}>

          <div className="-mx-4 border-t border-white/10 bg-black/75 shadow-[0_-16px_48px_rgba(0,0,0,0.45)] backdrop-blur-xl sm:-mx-6">

            <div className="flex items-center justify-between gap-4 px-4 py-3 sm:px-6">

              <div className="flex items-center gap-2">

                <Clapperboard size={15} className="shrink-0 text-foreground-muted" />

                <p className="text-sm font-medium text-foreground">

                  Episodes

                </p>

              </div>

              <button type="button" onClick={(e) => {

                  e.stopPropagation();

                  onClose();

                }} className="rounded-md p-1 text-foreground-muted transition-colors hover:bg-white/8 hover:text-foreground" aria-label="Close episode picker" >

                <X size={16} />

              </button>

            </div>

            {seasons.length > 0 && (

              <div className="mb-3 flex gap-2 overflow-x-auto px-4 pb-1 scrollbar-hide sm:px-6">

                {seasons.map((season) => (

                  <button key={season.number} type="button" onClick={(e) => {

                      e.stopPropagation();

                      if (season.number === menuSeason) return;

                      onSeasonChange(season.number);

                    }} className={cn(

                      "flex-shrink-0 rounded-md border px-3 py-1.5 text-xs transition-colors",
                      menuSeason === season.number ? "border-foreground bg-foreground text-surface" : "border-white/10 text-foreground-muted hover:border-white/20 hover:text-foreground"

                    )} >

                    {season.label}

                  </button>

                ))}

              </div>

            )}

            <div className="relative">

              {canScrollLeft && (

                <>

                  <div className="pointer-events-none absolute top-0 bottom-5 left-0 z-[1] w-14 bg-gradient-to-r from-black/90 to-transparent sm:bottom-6 sm:w-16" />

                  <button type="button" onClick={(e) => {

                      e.stopPropagation();

                      this.scrollCarousel(-1);

                    }} className="absolute top-1/2 left-2 z-10 flex h-11 w-11 -translate-y-1/2 items-center justify-center rounded-full border border-white/25 bg-white/15 text-foreground shadow-[0_4px_24px_rgba(0,0,0,0.45)] backdrop-blur-md transition-colors hover:border-white/40 hover:bg-white/25 sm:left-3" aria-label="Scroll episodes left" >

                    <ChevronLeft size={22} strokeWidth={2.5} />

                  </button>

                </>

              )}

              {canScrollRight && (

                <>
                  <div className="pointer-events-none absolute top-0 right-0 bottom-5 z-[1] w-14 bg-gradient-to-l from-black/90 to-transparent sm:bottom-6 sm:w-16" />

                  <button type="button" onClick={(e) => {

                      e.stopPropagation();

                      this.scrollCarousel(1);

                    }} className="absolute top-1/2 right-2 z-10 flex h-11 w-11 -translate-y-1/2 items-center justify-center rounded-full border border-white/25 bg-white/15 text-foreground shadow-[0_4px_24px_rgba(0,0,0,0.45)] backdrop-blur-md transition-colors hover:border-white/40 hover:bg-white/25 sm:right-3" aria-label="Scroll episodes right" >

                    <ChevronRight size={22} strokeWidth={2.5} />

                  </button>

                </>

              )}

              <div className="flex items-stretch gap-4 overflow-x-auto scroll-smooth pb-5 pl-4 pr-4 pt-1 snap-x snap-mandatory scrollbar-hide scroll-pl-4 sm:gap-5 sm:pb-6 sm:pl-6 sm:pr-6 sm:scroll-pl-6"

                ref={this.carouselRef}
                onScroll={this.updateScrollButtons}

              >
                {episodesLoading &&

                  Array.from({ length: 4 }).map((_, index) => (

                    <div key={`ep-picker-skeleton-${index}`} data-episode-card className="flex w-[min(78vw,16.5rem)] flex-shrink-0 snap-start sm:w-64" >

                      <div className="flex h-full w-full flex-col overflow-hidden rounded-xl border border-white/10 bg-white/5">

                        <div className="aspect-[2/1] shrink-0 animate-pulse bg-white/8" />

                        <div className="flex flex-1 flex-col p-3">

                          <div className="h-4 w-3/4 shrink-0 animate-pulse rounded bg-white/8" />

                          <div className="mt-0.5 flex min-h-[3.25rem] flex-1 flex-col justify-between gap-1">

                            <div className="space-y-1.5">

                              <div className="h-3 w-full animate-pulse rounded bg-white/6" />

                              <div className="h-3 w-5/6 animate-pulse rounded bg-white/6" />

                            </div>

                            <div className="h-3 w-14 animate-pulse rounded bg-white/6" />

                          </div>

                        </div>

                      </div>

                    </div>

                  ))}

                {!episodesLoading && episodes.map((ep) => this.renderEpisodeCard(ep))}

                {!episodesLoading && episodes.length === 0 && (

                  <div className="flex w-full items-center justify-center py-10">

                    <p className="text-sm text-foreground-muted">

                      No episodes found for this season

                    </p>

                  </div>

                )}

              </div>

            </div>

          </div>

        </div>

        {this.renderDetailModal()}

      </>

    );

  }

}
