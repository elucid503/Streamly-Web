import { ContentRow } from "@/UI/ContentRow";
import { TitleCard } from "@/Features/Catalog/TitleCard";
import { LiveView } from "@/Features/Live/LiveView";
import { ShowsView } from "@/Features/Catalog/ShowsView";
import { SportsPage } from "@/Features/Sports/Sports";
import { AdminPanel } from "@/Features/Admin/Admin";
import { SettingsPanel } from "@/Features/Settings/Settings";
import { SearchChrome } from "@/Layout/SearchChrome";
import { ViewCarousel } from "@/Layout/ViewCarousel";
import type { ContextActionId } from "@/Layout/ViewContextBar";
import { ViewSwitcher } from "@/Layout/ViewSwitcher";
import { Button } from "@/UI/Button";
import { Modal } from "@/UI/Modal";

import { ModuleComponent } from "@/Core/Store";
import { authAPI } from "@/Features/Auth/Api";
import { catalogAPI } from "@/Features/Catalog/Api";
import { favoritesAPI } from "@/Features/Library/FavoritesApi";
import { historyAPI } from "@/Features/Library/HistoryApi";
import { liveAPI } from "@/Features/Live/Api";
import { serviceAlertAPI } from "@/Features/Admin/ServiceAlertApi";
import { auth as authStore } from "@/Features/Auth/Store";
import { settings as settingsStore } from "@/Features/Settings/Store";
import type { FavoriteItem, WatchHistoryItem } from "@/Features/Library/Types";
import type { LiveChannel } from "@/Features/Live/Types";
import type { MainView } from "@/Layout/Types";
import type { SearchHit } from "@/Features/Catalog/Types";
import type { ServiceInterruption } from "@/Features/Admin/Types";
import { lastWatched, lastWatchedPath, resumePath, showResumeItem } from "@/Features/Library/History";
import type { NavigateFn } from "@/Utils/Navigation";

function viewFromQuery(): MainView | null {

  if (typeof window === "undefined") return null;

  const raw = new URLSearchParams(window.location.search).get("view");

  if (raw === "vod" || raw === "live" || raw === "sports") return raw;

  return null;

}

function initialView(): MainView {

  const fromQuery = viewFromQuery();

  if (fromQuery) {

    localStorage.setItem("streamly:lastView", fromQuery);

    return fromQuery;

  }

  const raw = localStorage.getItem("streamly:lastView");

  if (raw === "shows" || raw === "movies" || raw === "vod" || !raw) return "vod";

  if (raw === "live" || raw === "sports") return raw;

  return "vod";

}

interface HomePageProps {

  navigate: NavigateFn;

}

interface HomePageState {

  view: MainView;

  searchQuery: string;

  searchResults: SearchHit[];
  searchKind: "all" | "movie" | "show";
  searchYear: "all" | "2020s" | "2010s" | "2000s" | "older";
  searchRating: "all" | "7" | "8";
  searchProgress: "all" | "unwatched" | "in_progress" | "completed";

  searching: boolean;

  history: WatchHistoryItem[];
  favorites: FavoriteItem[];

  settingsOpen: boolean;
  adminOpen: boolean;

  interruption: ServiceInterruption | null;
  interruptionOpen: boolean;

  contextLoading: ContextActionId | null;

}

export class HomePage extends ModuleComponent<HomePageProps, HomePageState> {

  private searchDebounce: ReturnType<typeof setTimeout> | null = null;

  state: HomePageState = {

    view: initialView(),

    searchQuery: "",

    searchResults: [],
    searchKind: "all",
    searchYear: "all",
    searchRating: "all",
    searchProgress: "all",

    searching: false,

    history: [],
    favorites: [],

    settingsOpen: false,
    adminOpen: false,

    interruption: null,
    interruptionOpen: false,

    contextLoading: null,

  };

  componentDidMount() {

    this.watch(authStore);
    this.watch(settingsStore);

    this.loadHomeData();
    this.checkServiceInterruption();

    document.addEventListener("visibilitychange", this.handleVisibility);

  }

  componentWillUnmount() {

    if (this.searchDebounce) clearTimeout(this.searchDebounce);

    document.removeEventListener("visibilitychange", this.handleVisibility);

  }

  handleVisibility = () => {

    if (document.visibilityState === "visible") {

      void this.loadHomeData();

    }

  };

  checkServiceInterruption = async () => {

    try {

      const data = await serviceAlertAPI.get();

      if (!data.enabled) return;

      const seenAt = localStorage.getItem("streamly:service-interruption-seen");

      if (seenAt === data.updatedAt) return;

      this.setState({ interruption: data, interruptionOpen: true });

    } catch {

      /* ignore */

    }

  };

  dismissInterruption = () => {

    const { interruption } = this.state;

    if (interruption?.updatedAt) {

      localStorage.setItem("streamly:service-interruption-seen", interruption.updatedAt);

    }

    this.setState({ interruptionOpen: false });

  };

  loadHomeData = async () => {

    try {

      const [history, favorites] = await Promise.all([

        historyAPI.get(),
        favoritesAPI.get(),

      ]);

      this.setState({ history: history ?? [], favorites: favorites ?? [] });

    } catch {

      /* ignore */

    }

  };

  favoriteKey = (item: FavoriteItem | SearchHit | LiveChannel) => {

    if ("code" in item) return `live:${item.id}`;

    if (item.kind === "live") return `live:${item.channelId ?? item.id}`;

    const mediaId = "mediaId" in item ? item.mediaId : item.id;

    return `${item.kind}:${mediaId}`;

  };

  isFavorite = (item: FavoriteItem | SearchHit | LiveChannel) => {

    const key = this.favoriteKey(item);

    return this.state.favorites.some((favorite) => this.favoriteKey(favorite) === key);

  };

  handleFavoriteToggle = async (item: FavoriteItem | SearchHit | LiveChannel) => {

    const key = this.favoriteKey(item);
    const existing = this.state.favorites.find((favorite) => this.favoriteKey(favorite) === key);

    if (existing) {

      this.setState({ favorites: this.state.favorites.filter((favorite) => favorite.id !== existing.id) });

      const deleteKey = existing.kind === "live" ? existing.channelId ?? existing.id : existing.mediaId;

      await favoritesAPI.delete(existing.kind, deleteKey).catch(() => this.loadHomeData());

      return;

    }

    if ("code" in item) {

      const optimistic: FavoriteItem = {

        id: `pending-live-${item.id}`,
        kind: "live",
        mediaId: 0,
        channelId: item.id,
        title: item.name,
        poster: item.logo,
        createdAt: new Date().toISOString(),

      };

      this.setState({ favorites: [optimistic, ...this.state.favorites] });

      const saved = await favoritesAPI.upsert(optimistic).catch(() => null);

      if (saved) this.setState({ favorites: this.state.favorites.map((favorite) => favorite.id === optimistic.id ? saved : favorite) });
      else void this.loadHomeData();

      return;

    }

    if (item.kind === "live") return;

    const mediaId = "mediaId" in item ? item.mediaId : item.id;

    const optimistic: FavoriteItem = {

      id: `pending-${item.kind}-${mediaId}`,
      kind: item.kind,
      mediaId,
      title: item.title,
      poster: item.poster,
      year: item.year,
      rating: item.rating,
      createdAt: new Date().toISOString(),

    };

    this.setState({ favorites: [optimistic, ...this.state.favorites] });

    const saved = await favoritesAPI.upsert(optimistic).catch(() => null);

    if (saved) this.setState({ favorites: this.state.favorites.map((favorite) => favorite.id === optimistic.id ? saved : favorite) });
    else void this.loadHomeData();

  };

  searchProgressFor = (hit: SearchHit) => {

    return this.state.history.find((item) => item.kind === hit.kind && item.mediaId === hit.id);

  };

  filteredSearchResults = () => {

    const { searchResults, searchKind, searchYear, searchRating, searchProgress } = this.state;

    return searchResults.filter((hit) => {

      if (searchKind !== "all" && hit.kind !== searchKind) return false;

      if (searchYear !== "all") {

        if (searchYear === "2020s" && hit.year < 2020) return false;
        if (searchYear === "2010s" && (hit.year < 2010 || hit.year > 2019)) return false;
        if (searchYear === "2000s" && (hit.year < 2000 || hit.year > 2009)) return false;
        if (searchYear === "older" && hit.year >= 2000) return false;

      }

      if (searchRating !== "all") {

        const rating = Number.parseFloat(hit.rating);

        if (!Number.isFinite(rating) || rating < Number(searchRating)) return false;

      }

      if (searchProgress !== "all") {

        const progress = this.searchProgressFor(hit);

        if (searchProgress === "unwatched" && progress) return false;
        if (searchProgress === "in_progress" && (!progress || progress.completed)) return false;
        if (searchProgress === "completed" && !progress?.completed) return false;

      }

      return true;

    });

  };

  handleSearch = (query: string) => {

    this.setState({ searchQuery: query });

    if (this.searchDebounce) clearTimeout(this.searchDebounce);

    if (!query.trim()) {

      this.setState({ searchResults: [], searching: false });

      return;

    }

    this.setState({ searching: true });

    this.searchDebounce = setTimeout(async () => {

      try {

        const results = await catalogAPI.search(query);

        this.setState({ searchResults: results ?? [], searching: false });

      } catch {

        this.setState({ searchResults: [], searching: false });

      }

    }, 350);

  };

  handleSelect = (id: number, kind: "movie" | "show") => {

    this.props.navigate(kind === "movie" ? `/watch/movie/${id}` : `/show/${id}`);

  };

  handleResumeWatching = (path: string) => {

    this.props.navigate(path);

  };

  handleRemoveFromHistory = async (historyId: string) => {

    this.setState({ history: this.state.history.filter((item) => item.id !== historyId) });

    await historyAPI.delete(historyId).catch(() => this.loadHomeData());

  };

  handleLiveSelect = (channel: LiveChannel) => {

    this.props.navigate(`/live/${channel.id}`);

  };

  handleLogout = async () => {

    await authAPI.logout();

    authStore.setUser(null);

    settingsStore.setSettings(null);

    this.props.navigate("/auth");

  };

  pickRandom = <T,>(items: T[]): T | null => {

    if (items.length === 0) return null;

    return items[Math.floor(Math.random() * items.length)] ?? null;

  };

  handleContinueWatching = () => {

    const { view } = this.state;

    const kind = view === "vod" ? "vod" : "live";
    const item = lastWatched(this.state.history, kind);

    if (!item) return;

    const path = lastWatchedPath(item);

    if (path) this.props.navigate(path);

  };

  handleDiceRoll = async () => {

    const { view } = this.state;

    this.setState({ contextLoading: "dice" });

    try {

      if (view === "vod") {

        const [movies, shows] = await Promise.all([

          catalogAPI.movieTrending(24).catch(() => []),
          catalogAPI.showTrending(24).catch(() => []),

        ]);

        const pool = [

          ...(movies ?? []).map((item) => ({ kind: "movie" as const, item })),
          ...(shows ?? []).map((item) => ({ kind: "show" as const, item })),

        ];

        const pick = this.pickRandom(pool);

        if (!pick) return;

        if (pick.kind === "movie") {

          this.props.navigate(`/watch/movie/${pick.item.id}`);

          return;

        }

        const seasons = await catalogAPI.showSeasons(pick.item.id).catch(() => []);
        const season = seasons[0]?.number ?? 1;
        const episodes = await catalogAPI.seasonEpisodes(pick.item.id, season).catch(() => []);
        const episode = this.pickRandom(episodes) ?? { season, episode: 1 };

        this.props.navigate(`/watch/show/${pick.item.id}/${episode.season}/${episode.episode}`);

        return;

      }

      const channels = await liveAPI.popular(24);
      const pick = this.pickRandom(channels ?? []);

      if (pick) this.props.navigate(`/live/${pick.id}`);

    } catch {

      /* ignore */

    } finally {

      this.setState({ contextLoading: null });

    }

  };

  handleShuffleFavorites = async () => {

    const { view, favorites, history } = this.state;

    this.setState({ contextLoading: "shuffle-favorites" });

    try {

      if (view === "live") {

        const pool = favorites.filter((item) => item.kind === "live");
        const pick = this.pickRandom(pool);

        if (pick?.channelId) this.props.navigate(`/live/${pick.channelId}`);

        return;

      }

      const pool = favorites.filter((item) => item.kind === "movie" || item.kind === "show");
      const pick = this.pickRandom(pool);

      if (!pick) return;

      if (pick.kind === "movie") {

        this.props.navigate(`/watch/movie/${pick.mediaId}`);

        return;

      }

      const resumeItem = showResumeItem(history, pick.mediaId);
      const path = resumeItem ? resumePath(resumeItem) : null;

      if (path) {

        this.props.navigate(path);

        return;

      }

      const seasons = await catalogAPI.showSeasons(pick.mediaId).catch(() => []);
      const season = seasons[0]?.number ?? 1;

      this.props.navigate(`/watch/show/${pick.mediaId}/${season}/1`);

    } catch {

      /* ignore */

    } finally {

      this.setState({ contextLoading: null });

    }

  };

  handleContextAction = (actionId: ContextActionId) => {

    if (this.state.contextLoading) return;

    if (actionId === "continue") {

      this.handleContinueWatching();

      return;

    }

    if (actionId === "dice") {

      void this.handleDiceRoll();

      return;

    }

    void this.handleShuffleFavorites();

  };

  renderSearchResults() {

    const { searchResults, searching } = this.state;

    const filtered = this.filteredSearchResults();

    const shows = filtered.filter((h) => h.kind === "show");

    const movies = filtered.filter((h) => h.kind === "movie");

    if (!searching && searchResults.length === 0) {

      return (

        <div className="px-4 py-16 text-center text-sm text-foreground-muted sm:px-8">

          No results found

        </div>

      );

    }

    return (

      <div className="py-6">

        {!searching && searchResults.length > 0 && filtered.length === 0 && (

          <div className="px-4 py-16 text-center text-sm text-foreground-muted sm:px-8">

            No results match these filters

          </div>

        )}

        {shows.length > 0 && (

          <ContentRow title="TV Shows">

            {shows.map((hit) => {

              const progress = this.searchProgressFor(hit);
              const resumable = progress ? resumePath(progress) : null;

              return (

                <TitleCard key={`show-${hit.id}`}

                  id={hit.id}
                  kind="show"

                  title={hit.title}
                  poster={hit.poster}
                  year={hit.year}

                  favorite={this.isFavorite(hit)}
                  onFavoriteToggle={() => this.handleFavoriteToggle(hit)}

                  onResume={resumable ? () => this.handleResumeWatching(resumable) : undefined}
                  onRemoveFromHistory={progress ? () => this.handleRemoveFromHistory(progress.id) : undefined}

                  onClick={() => this.handleSelect(hit.id, hit.kind)}

                />

              );

            })}

          </ContentRow>

        )}

        {movies.length > 0 && (

          <ContentRow title="Movies">

            {movies.map((hit) => {

              const progress = this.searchProgressFor(hit);
              const resumable = progress ? resumePath(progress) : null;

              return (

                <TitleCard key={`movie-${hit.id}`}

                  id={hit.id}
                  kind="movie"

                  title={hit.title}
                  poster={hit.poster}
                  year={hit.year}

                  favorite={this.isFavorite(hit)}
                  onFavoriteToggle={() => this.handleFavoriteToggle(hit)}

                  onResume={resumable ? () => this.handleResumeWatching(resumable) : undefined}
                  onRemoveFromHistory={progress ? () => this.handleRemoveFromHistory(progress.id) : undefined}

                  onClick={() => this.handleSelect(hit.id, hit.kind)}

                />

              );

            })}

          </ContentRow>

        )}

        {searching && searchResults.length === 0 && (

          <ContentRow title="Searching" loading>

            {null}

          </ContentRow>

        )}

      </div>

    );

  }

  render() {

    const { view, searchQuery, searchKind, searchYear, searchRating, searchProgress, history, favorites, settingsOpen, adminOpen, interruption, interruptionOpen, contextLoading } = this.state;

    const showSearch = searchQuery.trim().length > 0 && view !== "live" && view !== "sports";

    const onViewChange = (v: MainView) => {

      localStorage.setItem("streamly:lastView", v);
      this.setState({ view: v, contextLoading: null });

    };

    const chrome = (

      <SearchChrome

        searchQuery={searchQuery}
        onSearch={this.handleSearch}

        view={view}
        showSearch={showSearch}

        searchKind={searchKind}
        searchYear={searchYear}
        searchRating={searchRating}
        searchProgress={searchProgress}

        onSearchKindChange={(value) => this.setState({ searchKind: value })}
        onSearchYearChange={(value) => this.setState({ searchYear: value })}
        onSearchRatingChange={(value) => this.setState({ searchRating: value })}
        onSearchProgressChange={(value) => this.setState({ searchProgress: value })}

        history={history}
        favorites={favorites}

        contextLoading={contextLoading}
        onContextAction={this.handleContextAction}

        onOpenSettings={() => this.setState({ settingsOpen: true })}
        onOpenAdmin={() => this.setState({ adminOpen: true })}
        onLogout={this.handleLogout}

      />

    );

    const modals = (

      <>

        <SettingsPanel open={settingsOpen} onClose={() => this.setState({ settingsOpen: false })} />

        <AdminPanel open={adminOpen} onClose={() => this.setState({ adminOpen: false })} />

        {interruption && (

          <Modal open={interruptionOpen} onClose={this.dismissInterruption} title={interruption.title || "Service Interruption"}>

            <p className="mb-5 text-sm text-foreground-muted">{interruption.message}</p>

            <Button onClick={this.dismissInterruption} className="w-full">

              Dismiss

            </Button>

          </Modal>

        )}

      </>

    );

    return (

      <div className="relative min-h-screen">

        <div className="relative z-10 overflow-x-clip pb-dock">

          {chrome}

          {showSearch ? (

            this.renderSearchResults()

          ) : (

            <ViewCarousel
              active={view}

              panels={{

                vod: (

                  <ShowsView

                    onSelect={this.handleSelect}
                    onFavoriteToggle={this.handleFavoriteToggle}
                    onResumeWatching={this.handleResumeWatching}
                    onRemoveFromHistory={this.handleRemoveFromHistory}

                    history={history}
                    favorites={favorites}

                  />

                ),

                live: (

                  <LiveView

                    onSelect={this.handleLiveSelect}
                    onFavoriteToggle={this.handleFavoriteToggle}

                    searchQuery={searchQuery}
                    favorites={favorites}

                  />

                ),

                sports: (

                  <SportsPage

                    onSelectChannel={this.handleLiveSelect}

                    searchQuery={searchQuery}

                  />

                ),

              }}

            />

          )}

        </div>

        <ViewSwitcher active={view} onChange={onViewChange} />

        {modals}

      </div>

    );

  }

}
