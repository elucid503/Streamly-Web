import { api } from "@/api/client";

import { LiveView } from "@/components/catalog/LiveView";
import { MoviesView } from "@/components/catalog/MoviesView";
import { ShowsView } from "@/components/catalog/ShowsView";
import { ContentRow } from "@/components/catalog/ContentRow";
import { TitleCard } from "@/components/catalog/TitleCard";
import { Header } from "@/components/layout/Header";
import { HomeBackdrop } from "@/components/layout/HomeBackdrop";
import { HomeBottomBar } from "@/components/layout/HomeBottomBar";
import { ViewCarousel } from "@/components/layout/ViewCarousel";
import type { ContextActionId } from "@/components/layout/ViewContextBar";
import { AdminPanel } from "@/pages/AdminPanel";
import { SettingsPanel } from "@/pages/SettingsPanel";

import type { NavigateFn } from "@/lib/navigation";
import { lastWatched, lastWatchedPath, resumePath, showResumeItem } from "@/lib/history";
import { store } from "@/lib/store";

import type { FavoriteItem, LiveChannel, MainView, SearchHit, WatchHistoryItem } from "@/lib/types";

import { Component } from "react";

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

  contextLoading: ContextActionId | null;

}

export class HomePage extends Component<HomePageProps, HomePageState> {

  private searchDebounce: ReturnType<typeof setTimeout> | null = null;

  private unsubscribe = () => {};

  state: HomePageState = {

    view: "shows",

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

    contextLoading: null,

  };

  componentDidMount() {

    this.unsubscribe = store.subscribe(() => this.forceUpdate());

    this.loadHomeData();

  }

  componentWillUnmount() {

    this.unsubscribe();

    if (this.searchDebounce) clearTimeout(this.searchDebounce);

  }

  loadHomeData = async () => {

    try {

      const [history, favorites] = await Promise.all([

        api.getHistory(),
        api.getFavorites(),

      ]);

      this.setState({ history: history ?? [], favorites: favorites ?? [] });

    } catch {

      /* ignore */

    }

  };

  favoriteKey = (item: FavoriteItem | SearchHit | LiveChannel) => {

    if ("daddyId" in item) return `live:${item.daddyId}`;

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

      await api.deleteFavorite(existing.kind, deleteKey).catch(() => this.loadHomeData());

      return;

    }

    if ("daddyId" in item) {

      const optimistic: FavoriteItem = {

        id: `pending-live-${item.daddyId}`,
        kind: "live",
        mediaId: 0,
        channelId: item.daddyId,
        title: item.name,
        poster: item.logo,
        category: item.category,
        createdAt: new Date().toISOString(),

      };

      this.setState({ favorites: [optimistic, ...this.state.favorites] });

      const saved = await api.upsertFavorite(optimistic).catch(() => null);

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

    const saved = await api.upsertFavorite(optimistic).catch(() => null);

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

        const results = await api.search(query);

        this.setState({ searchResults: results ?? [], searching: false });

      } catch {

        this.setState({ searchResults: [], searching: false });

      }

    }, 350);

  };

  handleSelect = (id: number, kind: "movie" | "show") => {

    this.props.navigate(`/${kind}/${id}`);

  };

  handleResume = (item: WatchHistoryItem) => {

    const path = resumePath(item);

    if (path) {

      this.props.navigate(path);

      return;

    }

    this.handleSelect(item.mediaId, item.kind as "movie" | "show");

  };

  handleLiveSelect = (channel: LiveChannel) => {

    this.props.navigate(`/live/${channel.daddyId}`);

  };

  handleLogout = async () => {

    await api.logout();

    store.setUser(null);

    store.setSettings(null);

    this.props.navigate("/auth");

  };

  pickRandom = <T,>(items: T[]): T | null => {

    if (items.length === 0) return null;

    return items[Math.floor(Math.random() * items.length)] ?? null;

  };

  handleContinueWatching = () => {

    const { view } = this.state;

    const kind = view === "shows" ? "show" : view === "movies" ? "movie" : "live";
    const item = lastWatched(this.state.history, kind);

    if (!item) return;

    const path = lastWatchedPath(item);

    if (path) this.props.navigate(path);

  };

  handleDiceRoll = async () => {

    const { view } = this.state;

    this.setState({ contextLoading: "dice" });

    try {

      if (view === "movies") {

        const trending = await api.movieTrending(24);
        const pick = this.pickRandom(trending ?? []);

        if (pick) this.props.navigate(`/watch/movie/${pick.id}`);

        return;

      }

      if (view === "shows") {

        const trending = await api.showTrending(24);
        const pick = this.pickRandom(trending ?? []);

        if (!pick) return;

        const seasons = await api.showSeasons(pick.id).catch(() => []);
        const season = seasons[0]?.number ?? 1;
        const episodes = await api.seasonEpisodes(pick.id, season).catch(() => []);
        const episode = this.pickRandom(episodes) ?? { season, episode: 1 };

        this.props.navigate(`/watch/show/${pick.id}/${episode.season}/${episode.episode}`);

        return;

      }

      const channels = await api.livePopular(24);
      const pick = this.pickRandom(channels ?? []);

      if (pick) this.props.navigate(`/live/${pick.daddyId}`);

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

      const kind = view === "shows" ? "show" : "movie";
      const pool = favorites.filter((item) => item.kind === kind);
      const pick = this.pickRandom(pool);

      if (!pick) return;

      if (kind === "movie") {

        this.props.navigate(`/watch/movie/${pick.mediaId}`);

        return;

      }

      const resumeItem = showResumeItem(history, pick.mediaId);
      const path = resumeItem ? resumePath(resumeItem) : null;

      if (path) {

        this.props.navigate(path);

        return;

      }

      const seasons = await api.showSeasons(pick.mediaId).catch(() => []);
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

            {shows.map((hit) => (

              <TitleCard key={`show-${hit.id}`}

                id={hit.id}
                kind="show"

                title={hit.title}
                poster={hit.poster}
                year={hit.year}

                favorite={this.isFavorite(hit)}
                onFavoriteToggle={() => this.handleFavoriteToggle(hit)}

                onClick={() => this.handleSelect(hit.id, hit.kind)}

              />

            ))}

          </ContentRow>

        )}

        {movies.length > 0 && (

          <ContentRow title="Movies">

            {movies.map((hit) => (

              <TitleCard key={`movie-${hit.id}`}

                id={hit.id}
                kind="movie"

                title={hit.title}
                poster={hit.poster}
                year={hit.year}

                favorite={this.isFavorite(hit)}
                onFavoriteToggle={() => this.handleFavoriteToggle(hit)}

                onClick={() => this.handleSelect(hit.id, hit.kind)}

              />

            ))}

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

    const { view, searchQuery, searchKind, searchYear, searchRating, searchProgress, history, favorites, settingsOpen, adminOpen, contextLoading } = this.state;

    const showSearch = searchQuery.trim().length > 0 && view !== "live";

    return (

      <div className="relative min-h-screen">

        <HomeBackdrop view={view} history={history} favorites={favorites} />

        <Header

          view={view}
          onViewChange={(v) => this.setState({ view: v, contextLoading: null })}
          onOpenSettings={() => this.setState({ settingsOpen: true })}
          onOpenAdmin={() => this.setState({ adminOpen: true })}
          onLogout={this.handleLogout}

        />

        <div className="relative z-10 overflow-x-hidden pt-16 pb-24 lg:pb-32">

        {showSearch ? (

          this.renderSearchResults()

        ) : (

          <ViewCarousel

            active={view}

            panels={{

              shows: (

                <ShowsView

                  onSelect={this.handleSelect}
                  onFavoriteToggle={this.handleFavoriteToggle}

                  history={history}
                  favorites={favorites}

                />

              ),

              movies: (

                <MoviesView

                  onSelect={this.handleSelect}
                  onResume={this.handleResume}
                  onFavoriteToggle={this.handleFavoriteToggle}

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

            }}

          />

        )}

        <SettingsPanel open={settingsOpen} onClose={() => this.setState({ settingsOpen: false })} />

        <AdminPanel open={adminOpen} onClose={() => this.setState({ adminOpen: false })} />

        <HomeBottomBar

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

        />

        </div>

      </div>

    );

  }

}
