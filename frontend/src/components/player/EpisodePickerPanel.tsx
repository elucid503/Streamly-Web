import { Button } from "@/components/ui/Button";
import { CachedImage } from "@/components/ui/CachedImage";
import { HScrollRow } from "@/components/ui/HScrollRow";
import { Modal } from "@/components/ui/Modal";
import { PlayerMobileSheet } from "@/components/player/PlayerMobileSheet";

import { cn } from "@/lib/utils";
import type { Episode, Season } from "@/lib/types";

import { Component, createRef } from "react";
import { ChevronLeft, ChevronRight, Clapperboard, Film, Play, X } from "lucide-react";

interface EpisodePickerPanelProps {

  open: boolean;
  compact?: boolean;

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

              <Film size={this.props.compact ? 16 : 28} strokeWidth={1.5} />

            </span>

          }

        />

      ) : (

        <span className="flex size-full items-center justify-center text-foreground-faint">

          <Film size={this.props.compact ? 16 : 28} strokeWidth={1.5} />

        </span>

      )}

    </div>

  );

  renderEpisodeCard = (ep: Episode) => {

    const { currentSeason, currentEpisode, onEpisodeSelect, compact } = this.props;

    const key = `${ep.season}-${ep.episode}`;

    const active = currentSeason === ep.season && currentEpisode === ep.episode;

    const description = ep.description?.trim() ?? "";

    return (

      <div className={cn("flex flex-shrink-0 snap-start", compact ? "w-[min(32vw,6.75rem)]" : "w-64")}

        key={key}
        data-episode-card
        ref={(node) => this.setCardRef(key, node)}

      >
        <button type="button" onClick={(e) => {

            e.stopPropagation();

            onEpisodeSelect(ep.season, ep.episode);

          }} className={cn(

              "group flex h-full w-full flex-col overflow-hidden border text-left transition-colors",
              compact ? "rounded-md" : "rounded-lg",
              active ? "ring-2 ring-[#969696] bg-white/10" : "border-transparent bg-white/5 ring-1 ring-white/10 hover:ring-white/20 hover:bg-white/8"

          )}

        >

          <div className="relative aspect-[2/1] w-full shrink-0 overflow-hidden bg-white/5">

            {this.renderEpisodeThumbnail(ep, "size-full")}

            <span className={cn(

                "absolute font-medium tracking-wide text-foreground border border-border-subtle bg-surface/80",
                compact ? "top-1 left-1 rounded px-1 py-px text-[8px]" : "top-2 left-2 rounded-md px-2 py-0.5 text-[10px]"

              )}

            >

              E{ep.episode}

            </span>

          </div>

          <div className={cn("flex flex-1 flex-col", compact ? "p-1.5" : "p-3")}>

            <p className={cn("line-clamp-2 shrink-0 font-medium leading-snug text-foreground", compact ? "text-[10px]" : "text-sm")}>

              {ep.title}

            </p>

            <div className={cn("mt-0.5 flex flex-1 flex-col justify-between", compact ? "min-h-[1.5rem] gap-0.5" : "min-h-[3.25rem] gap-1")}>

              <p className={cn(
                  compact ? "line-clamp-1 text-[9px] leading-snug" : "line-clamp-2 text-xs leading-relaxed",
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

                }} className={cn("font-medium text-foreground/80 underline-offset-2 hover:text-foreground hover:underline", compact ? "text-[8px]" : "text-[11px]")} >

                Show more

              </span>

            </div>

          </div>

        </button>

      </div>

    );

  };

  renderMobileRow = (ep: Episode) => {

    const { currentSeason, currentEpisode, onEpisodeSelect } = this.props;

    const key = `${ep.season}-${ep.episode}`;

    const active = currentSeason === ep.season && currentEpisode === ep.episode;

    const description = ep.description?.trim() ?? "";

    return (

      <div key={key} className={cn("flex items-center gap-3 rounded-xl border p-2.5", active ? "border-white/20 bg-white/10" : "border-transparent bg-white/5")}>

        <button type="button" onClick={(e) => {

            e.stopPropagation();

            onEpisodeSelect(ep.season, ep.episode);

          }} className="relative w-[7.75rem] shrink-0 overflow-hidden rounded-lg text-left"

        >

          {this.renderEpisodeThumbnail(ep, "aspect-video w-full")}

          <span className="absolute top-1.5 left-1.5 rounded-md border border-border-subtle bg-surface/80 px-1.5 py-0.5 text-[10px] font-medium text-foreground">

            E{ep.episode}

          </span>

        </button>

        <div className="min-w-0 flex-1 py-0.5">

          <button type="button" onClick={(e) => {

              e.stopPropagation();

              onEpisodeSelect(ep.season, ep.episode);

            }} className="w-full text-left"

          >

            <p className="truncate text-sm font-medium leading-snug text-foreground landscape:text-base">

              {ep.title}

            </p>

          </button>

          <div className="relative mt-1">

            <p className={cn("truncate pr-10 text-xs leading-relaxed landscape:pr-12 landscape:text-sm", description ? "text-foreground-muted" : "text-foreground-faint")}>

              {description || "No description available"}

            </p>

            <button type="button" onClick={(e) => {

                e.stopPropagation();

                this.openDetail(ep);

              }} className="absolute right-0 bottom-0 text-xs font-medium text-foreground/80 underline-offset-2 hover:text-foreground hover:underline landscape:text-sm"

            >

              More

            </button>

          </div>

        </div>

      </div>

    );

  };

  renderMobile = () => {

    const { open, seasons, episodes, menuSeason, episodesLoading, onClose, onSeasonChange } = this.props;

    const seasonChips = seasons.length > 0 ? (

      <HScrollRow className="gap-2">

        {seasons.map((season) => (

          <button key={season.number} type="button" onClick={(e) => {

              e.stopPropagation();

              if (season.number === menuSeason) return;

              onSeasonChange(season.number);

            }} className={cn(

              "flex-shrink-0 rounded-lg border px-3 py-2 text-sm transition-colors",
              menuSeason === season.number ? "border-foreground bg-foreground text-surface" : "border-white/10 text-foreground-muted hover:border-white/20 hover:text-foreground"

            )} >

            {season.label}

          </button>

        ))}

      </HScrollRow>

    ) : null;

    return (

      <>
        <PlayerMobileSheet open={open} title="Episodes" layout="fill" icon={<Clapperboard size={16} className="text-foreground-muted" />} onClose={onClose} headerExtra={seasonChips} >

          <div className="grid grid-cols-1 gap-3 landscape:grid-cols-2">

            {episodesLoading &&

              Array.from({ length: 6 }).map((_, index) => (

                <div key={`ep-mobile-skel-${index}`} className="flex gap-3 rounded-xl bg-white/5 p-2.5">

                  <div className="aspect-video w-[7.75rem] shrink-0 animate-pulse rounded-lg bg-white/8" />

                  <div className="min-w-0 flex-1 space-y-2 py-1">

                    <div className="h-4 w-3/4 animate-pulse rounded bg-white/8" />

                    <div className="h-3 w-full animate-pulse rounded bg-white/6" />

                    <div className="h-3 w-5/6 animate-pulse rounded bg-white/6" />

                  </div>

                </div>

              ))}

            {!episodesLoading && episodes.map((ep) => this.renderMobileRow(ep))}

            {!episodesLoading && episodes.length === 0 && (

              <div className="flex items-center justify-center py-16">

                <p className="text-sm text-foreground-muted">

                  No episodes found for this season

                </p>

              </div>

            )}

          </div>

        </PlayerMobileSheet>

        {this.renderDetailModal()}

      </>

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

    const { open, compact, seasons, episodes, menuSeason, episodesLoading, onClose, onSeasonChange } = this.props;

    const { canScrollLeft, canScrollRight } = this.state;

    if (compact) return this.renderMobile();

    if (!open) return null;

    return (

      <>
        <div className="w-full animate-fade-in" onClick={(e) => e.stopPropagation()}>

          <div className={cn("-mx-4 border-t border-border-subtle bg-surface/85 shadow-[0_-16px_48px_rgba(0,0,0,0.45)] backdrop-blur-xl", !compact && "sm:-mx-6")}>

            <div className={cn("flex items-center justify-between", compact ? "gap-2 px-2 py-1" : "gap-4 px-6 py-3")}>

              <div className={cn("flex items-center", compact ? "gap-1.5" : "gap-2")}>

                <Clapperboard size={compact ? 12 : 15} className="shrink-0 text-foreground-muted" />

                <p className={cn("font-medium text-foreground", compact ? "text-xs" : "text-sm")}>

                  Episodes

                </p>

              </div>

              <button type="button" onClick={(e) => {

                  e.stopPropagation();

                  onClose();

                }} className={cn("flex shrink-0 items-center justify-center rounded-md text-foreground-muted transition-colors hover:bg-white/10 hover:text-foreground", compact ? "size-6" : "size-8")} aria-label="Close episode picker" >

                <X size={compact ? 13 : 16} />

              </button>

            </div>

            {seasons.length > 0 && (

              <HScrollRow className={cn(compact ? "mb-1 gap-1 px-2 pb-0.5" : "mb-3 gap-2 px-6 pb-1")}>

                {seasons.map((season) => (

                  <button key={season.number} type="button" onClick={(e) => {

                      e.stopPropagation();

                      if (season.number === menuSeason) return;

                      onSeasonChange(season.number);

                    }} className={cn(

                      "flex-shrink-0 transition-colors",
                      compact ? "rounded border px-1.5 py-0.5 text-[9px]" : "rounded-md border px-3 py-1.5 text-xs",
                      menuSeason === season.number ? "border-foreground bg-foreground text-surface" : "border-white/10 text-foreground-muted hover:border-white/20 hover:text-foreground"

                    )} >

                    {season.label}

                  </button>

                ))}

              </HScrollRow>

            )}

            <div className="relative">

              {canScrollLeft && (

                <>

                  <div className="pointer-events-none absolute top-0 bottom-5 left-0 z-[1] w-14 bg-gradient-to-r from-black/90 to-transparent sm:bottom-6 sm:w-16" />

                  <button type="button" onClick={(e) => {

                      e.stopPropagation();

                      this.scrollCarousel(-1);

                    }} className="absolute top-1/2 left-2 z-10 hidden size-9 -translate-y-1/2 items-center justify-center rounded-md border border-border-subtle bg-surface/80 text-foreground shadow-lg shadow-black/30 backdrop-blur-xl transition-colors hover:bg-surface-overlay sm:left-3 sm:flex" aria-label="Scroll episodes left" >

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

                    }} className="absolute top-1/2 right-2 z-10 hidden size-9 -translate-y-1/2 items-center justify-center rounded-md border border-border-subtle bg-surface/80 text-foreground shadow-lg shadow-black/30 backdrop-blur-xl transition-colors hover:bg-surface-overlay sm:right-3 sm:flex" aria-label="Scroll episodes right" >

                    <ChevronRight size={22} strokeWidth={2.5} />

                  </button>

                </>

              )}

              <div className={cn(

                  "flex items-stretch overflow-x-auto scroll-smooth snap-x snap-mandatory scrollbar-hide",
                  compact ? "gap-1.5 pb-2 pl-2 pr-2 pt-0.5 scroll-pl-2" : "gap-5 pb-6 pl-6 pr-6 pt-1 scroll-pl-6"

                )}

                ref={this.carouselRef}
                onScroll={this.updateScrollButtons}

              >
                {episodesLoading &&

                  Array.from({ length: 4 }).map((_, index) => (

                    <div key={`ep-picker-skeleton-${index}`} data-episode-card className={cn("flex flex-shrink-0 snap-start", compact ? "w-[min(32vw,6.75rem)]" : "w-64")} >

                      <div className={cn("flex h-full w-full flex-col overflow-hidden border border-white/10 bg-white/5", compact ? "rounded-md" : "rounded-lg")}>

                        <div className="aspect-[2/1] shrink-0 animate-pulse bg-white/8" />

                        <div className={cn("flex flex-1 flex-col", compact ? "p-1.5" : "p-3")}>

                          <div className={cn("w-3/4 shrink-0 animate-pulse rounded bg-white/8", compact ? "h-3" : "h-4")} />

                          <div className={cn("mt-0.5 flex flex-1 flex-col justify-between", compact ? "min-h-[1.5rem] gap-0.5" : "min-h-[3.25rem] gap-1")}>

                            <div className={compact ? "space-y-1" : "space-y-1.5"}>

                              <div className={cn("w-full animate-pulse rounded bg-white/6", compact ? "h-2" : "h-3")} />

                              {!compact && <div className="h-3 w-5/6 animate-pulse rounded bg-white/6" />}

                            </div>

                            <div className={cn("animate-pulse rounded bg-white/6", compact ? "h-2 w-10" : "h-3 w-14")} />

                          </div>

                        </div>

                      </div>

                    </div>

                  ))}

                {!episodesLoading && episodes.map((ep) => this.renderEpisodeCard(ep))}

                {!episodesLoading && episodes.length === 0 && (

                  <div className={cn("flex w-full items-center justify-center", compact ? "py-5" : "py-10")}>

                    <p className={cn("text-foreground-muted", compact ? "text-xs" : "text-sm")}>

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
