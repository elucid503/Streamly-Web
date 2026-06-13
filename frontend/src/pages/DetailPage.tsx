import { Component } from "react";
import { motion } from "framer-motion";
import { ArrowLeft, Play } from "lucide-react";
import { api } from "@/api/client";
import type { NavigateFn } from "@/lib/navigation";
import { CachedImage } from "@/components/ui/CachedImage";
import { PosterImage } from "@/components/catalog/PosterImage";
import { Button } from "@/components/ui/Button";
import {
  episodeProgressPercent,
  resumePath,
  showEpisodeHistory,
  showResumeItem,
} from "@/lib/history";
import { cn } from "@/lib/utils";
import type { Episode, Season, TitleDetails, WatchHistoryItem } from "@/lib/types";

interface DetailPageProps {
  navigate: NavigateFn;
  kind: "movie" | "show";
  id: string;
}

interface DetailPageState {
  details: TitleDetails | null;
  seasons: Season[];
  episodes: Episode[];
  episodeCache: Record<number, Episode[]>;
  selectedSeason: number;
  loadingSeason: number | null;
  seasonsError: string;
  history: WatchHistoryItem[];
  loading: boolean;
  error: string;
}

export class DetailPage extends Component<DetailPageProps, DetailPageState> {
  private loadGen = 0;

  state: DetailPageState = {
    details: null,
    seasons: [],
    episodes: [],
    episodeCache: {},
    selectedSeason: 1,
    loadingSeason: null,
    seasonsError: "",
    history: [],
    loading: true,
    error: "",
  };

  async componentDidMount() {
    await this.load();
  }

  async componentDidUpdate(prev: DetailPageProps) {
    if (prev.id !== this.props.id || prev.kind !== this.props.kind) {
      await this.load();
    }
  }

  load = async () => {
    const { kind, id: idStr } = this.props;
    const id = Number(idStr);
    const gen = ++this.loadGen;

    if (!id || (kind !== "movie" && kind !== "show")) {
      this.setState({ error: "invalid title", loading: false });
      return;
    }

    this.setState({ loading: true, error: "", seasonsError: "" });

    try {
      if (kind === "movie") {
        const [details, history] = await Promise.all([
          api.movieDetails(id),
          api.getHistory().catch(() => []),
        ]);
        if (gen !== this.loadGen) return;
        this.setState({ details, history, loading: false });
      } else {
        const [details, history] = await Promise.all([
          api.showDetails(id),
          api.getHistory(500, id).catch(() => []),
        ]);
        if (gen !== this.loadGen) return;

        let seasons: Season[] = [];
        let seasonsError = "";
        try {
          seasons = await api.showSeasons(id);
        } catch (err) {
          seasonsError =
            err instanceof Error ? err.message : "seasons temporarily unavailable";
        }
        if (gen !== this.loadGen) return;

        const selectedSeason = seasons[0]?.number ?? 1;
        let episodes: Episode[] = [];
        if (seasons.length > 0) {
          try {
            episodes = await api.seasonEpisodes(id, selectedSeason);
          } catch {
            /* episodes may load on season select */
          }
        }
        if (gen !== this.loadGen) return;

        const episodeCache = episodes.length > 0 ? { [selectedSeason]: episodes } : {};
        this.setState({
          details,
          seasons,
          episodes,
          episodeCache,
          selectedSeason,
          loadingSeason: null,
          seasonsError,
          history,
          loading: false,
        });
        this.prefetchSeasons(id, seasons, selectedSeason, episodeCache);
      }
    } catch (err) {
      if (gen !== this.loadGen) return;
      this.setState({
        error: err instanceof Error ? err.message : "failed to load",
        loading: false,
      });
    }
  };

  prefetchSeasons = (
    showId: number,
    seasons: Season[],
    activeSeason: number,
    cached: Record<number, Episode[]>,
  ) => {
    seasons
      .filter((season) => season.number !== activeSeason && !cached[season.number])
      .slice(0, 3)
      .forEach((season) => {
        void api.seasonEpisodes(showId, season.number).then((episodes) => {
          this.setState((prev) => {
            if (prev.episodeCache[season.number]) return null;
            return {
              episodeCache: { ...prev.episodeCache, [season.number]: episodes },
            };
          });
        });
      });
  };

  selectSeason = async (season: number) => {
    const id = Number(this.props.id);
    if (season === this.state.selectedSeason && this.state.loadingSeason === null) return;

    const cached = this.state.episodeCache[season];
    if (cached) {
      this.setState({ selectedSeason: season, episodes: cached, loadingSeason: null });
      return;
    }

    this.setState({ loadingSeason: season });
    try {
      const episodes = await api.seasonEpisodes(id, season);
      this.setState((prev) => ({
        selectedSeason: season,
        episodes,
        loadingSeason: null,
        episodeCache: { ...prev.episodeCache, [season]: episodes },
      }));
    } catch {
      this.setState({ loadingSeason: null });
    }
  };

  play = (season?: number, episode?: number) => {
    const { kind, id, navigate } = this.props;

    if (kind === "movie") {
      navigate(`/watch/movie/${id}`);
    } else if (season && episode) {
      navigate(`/watch/show/${id}/${season}/${episode}`);
    } else {
      const first = this.state.episodes[0];
      if (first) {
        navigate(`/watch/show/${id}/${first.season}/${first.episode}`);
      }
    }
  };

  resume = () => {
    const resumeItem = showResumeItem(this.state.history, Number(this.props.id));
    const path = resumeItem ? resumePath(resumeItem) : null;
    if (path) {
      this.props.navigate(path);
      return;
    }
    this.play();
  };

  episodeProgress = (episode: Episode) =>
    showEpisodeHistory(
      this.state.history,
      Number(this.props.id),
      episode.season,
      episode.episode,
    );

  heroImage = (details: TitleDetails) => details.banner || details.poster;

  render() {
    const { navigate } = this.props;
    const { details, seasons, episodes, selectedSeason, loadingSeason, seasonsError, loading, error } =
      this.state;
    const kind = this.props.kind;

    if (loading) {
      return (
        <div className="flex min-h-screen items-center justify-center">
          <div className="h-8 w-8 animate-spin rounded-full border-2 border-foreground/20 border-t-foreground" />
        </div>
      );
    }

    if (error || !details) {
      return (
        <div className="flex min-h-screen flex-col items-center justify-center gap-4">
          <p className="text-sm text-foreground-muted">{error || "not found"}</p>
          <Button variant="outline" onClick={() => navigate("/")}>
            Go Back
          </Button>
        </div>
      );
    }

    const hero = this.heroImage(details);
    const hasBanner = !!details.banner;
    const showId = Number(this.props.id);
    const resumeItem = kind === "show" ? showResumeItem(this.state.history, showId) : undefined;

    return (
      <div className="min-h-screen overflow-x-hidden">
        <div className="relative h-[42vh] min-h-[280px] max-h-[480px] overflow-hidden sm:h-[48vh]">
          {hero &&
            (hasBanner ? (
              <img
                src={hero}
                alt=""
                decoding="async"
                className="absolute inset-0 size-full object-cover object-[center_20%] brightness-[0.55]"
              />
            ) : (
              <CachedImage
                src={hero}
                alt=""
                rounded=""
                className="absolute inset-0 size-full"
                imgClassName="scale-110 object-cover object-center blur-2xl brightness-[0.3]"
              />
            ))}
          <div className="absolute inset-0 bg-gradient-to-t from-surface via-surface/50 to-surface/10" />

          <button
            onClick={() => navigate("/")}
            className="absolute top-4 left-4 z-10 flex items-center gap-2 rounded-md bg-black/40 px-3 py-1.5 text-xs backdrop-blur-sm transition-colors hover:bg-black/60"
          >
            <ArrowLeft size={14} />
            Back
          </button>

          <div className="absolute inset-x-0 bottom-0 flex items-end gap-5 px-4 pb-6 sm:gap-6 sm:px-8 sm:pb-8">
            <motion.div
              initial={{ opacity: 0, y: 12 }}
              animate={{ opacity: 1, y: 0 }}
              className="hidden w-36 flex-shrink-0 overflow-hidden rounded-md border border-border shadow-lg sm:block sm:w-44"
            >
              <PosterImage src={details.poster} alt={details.title} className="aspect-[2/3] w-full" />
            </motion.div>

            <div className="min-w-0 flex-1 pb-1">
              <h1 className="text-xl font-semibold tracking-tight sm:text-3xl">{details.title}</h1>
              <div className="mt-2 flex flex-wrap items-center gap-3 text-sm text-foreground-muted">
                {details.year && <span>{details.year}</span>}
                {details.rating && <span>{details.rating}</span>}
                {kind === "show" && seasons.length > 0 && (
                  <span>
                    {seasons.length} season{seasons.length === 1 ? "" : "s"}
                  </span>
                )}
              </div>
              {details.description && (
                <p className="mt-3 line-clamp-3 max-w-2xl text-sm leading-relaxed text-foreground-muted sm:mt-4">
                  {details.description}
                </p>
              )}
              <Button
                className="mt-4 sm:mt-5"
                onClick={() => (resumeItem ? this.resume() : this.play())}
                disabled={kind === "show" && episodes.length === 0}
              >
                <Play size={14} />
                {resumeItem ? "Resume" : "Play"}
              </Button>
            </div>
          </div>
        </div>

        {kind === "show" && (
          <div className="px-4 py-6 sm:px-8 sm:py-8">
            {seasonsError && seasons.length === 0 && (
              <p className="mb-4 text-sm text-foreground-muted">{seasonsError}</p>
            )}

            {seasons.length > 0 && (
              <>
                <div className="mb-4 flex gap-2 overflow-x-auto scrollbar-hide">
                  {seasons.map((sn) => {
                    const isActive = selectedSeason === sn.number && loadingSeason === null;
                    const isLoading = loadingSeason === sn.number;
                    return (
                      <button
                        key={sn.number}
                        onClick={() => this.selectSeason(sn.number)}
                        disabled={isLoading}
                        className={cn(
                          "flex-shrink-0 rounded-md border px-3 py-1.5 text-xs transition-colors",
                          isActive
                            ? "border-foreground bg-foreground text-surface"
                            : "border-border text-foreground-muted hover:border-border hover:text-foreground",
                        )}
                      >
                        <span className={cn(isLoading && "animate-pulse")}>{sn.label}</span>
                      </button>
                    );
                  })}
                </div>

                <div
                  className={cn(
                    "divide-y divide-border rounded-md border border-border transition-opacity duration-200",
                    loadingSeason !== null && "opacity-60",
                  )}
                >
                  {episodes.map((ep) => {
                    const progress = this.episodeProgress(ep);
                    const pct = episodeProgressPercent(progress);
                    return (
                      <button
                        key={`${ep.season}-${ep.episode}`}
                        onClick={() => this.play(ep.season, ep.episode)}
                        className="group flex w-full items-center gap-4 px-3 py-3 text-left transition-colors first:rounded-t-md last:rounded-b-md hover:bg-surface-raised"
                      >
                        <span className="w-8 text-xs text-foreground-faint tabular-nums">
                          {ep.episode}
                        </span>
                        <span className="min-w-0 flex-1">
                          <span className="block truncate text-sm">{ep.title}</span>
                          {pct > 2 && (
                            <span className="mt-1.5 block h-1 max-w-56 overflow-hidden rounded-full bg-white/10">
                              <motion.span
                                className="block h-full rounded-full bg-foreground/80"
                                initial={{ width: 0 }}
                                animate={{ width: `${pct}%` }}
                                transition={{ duration: 0.3, ease: "easeOut" }}
                              />
                            </span>
                          )}
                        </span>
                        {pct > 2 && (
                          <span className="rounded-full border border-white/10 px-2 py-0.5 text-[10px] text-foreground-muted">
                            {Math.round(pct)}%
                          </span>
                        )}
                        <Play
                          size={14}
                          className="flex-shrink-0 text-foreground-faint group-hover:text-foreground"
                        />
                      </button>
                    );
                  })}
                  {loadingSeason === null && episodes.length === 0 && (
                    <p className="px-3 py-6 text-center text-sm text-foreground-muted">
                      No episodes found for this season
                    </p>
                  )}
                </div>
              </>
            )}
          </div>
        )}
      </div>
    );
  }
}
