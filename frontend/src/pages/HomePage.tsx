import { api } from "@/api/client";

import { LiveView } from "@/components/catalog/LiveView";
import { MoviesView } from "@/components/catalog/MoviesView";
import { ShowsView } from "@/components/catalog/ShowsView";
import { ContentRow } from "@/components/catalog/ContentRow";
import { TitleCard } from "@/components/catalog/TitleCard";
import { Header } from "@/components/layout/Header";
import { ViewSwitcher } from "@/components/layout/ViewSwitcher";
import { ViewCarousel } from "@/components/layout/ViewCarousel";
import { AdminPanel } from "@/pages/AdminPanel";
import { SettingsPanel } from "@/pages/SettingsPanel";
import { SelectMenu } from "@/components/ui/SelectMenu";

import type { NavigateFn } from "@/lib/navigation";
import { resumePath } from "@/lib/history";
import { store } from "@/lib/store";

import type { FavoriteItem, LiveChannel, MainView, SearchHit, WatchHistoryItem } from "@/lib/types";

import { Component } from "react";
import { SlidersHorizontal } from "lucide-react";

const searchKindOptions = [
  { value: "all", label: "All titles" },
  { value: "show", label: "Shows" },
  { value: "movie", label: "Movies" },
];

const searchYearOptions = [
  { value: "all", label: "Any year" },
  { value: "2020s", label: "2020s" },
  { value: "2010s", label: "2010s" },
  { value: "2000s", label: "2000s" },
  { value: "older", label: "Before 2000" },
];

const searchRatingOptions = [
  { value: "all", label: "Any rating" },
  { value: "7", label: "7.0+" },
  { value: "8", label: "8.0+" },
];

const searchProgressOptions = [
  { value: "all", label: "Any progress" },
  { value: "unwatched", label: "Unwatched" },
  { value: "in_progress", label: "In progress" },
  { value: "completed", label: "Completed" },
];

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

  renderSearchFilters() {

    const { searchKind, searchYear, searchRating, searchProgress } = this.state;

    return (

      <div className="flex min-w-0 flex-wrap items-center justify-center gap-2">

        <div className="flex h-9 items-center gap-2 rounded-full border border-border-subtle bg-surface-raised px-3 text-xs font-medium text-foreground-muted shadow-sm">

          <SlidersHorizontal size={14} />
          <span>Filters</span>

        </div>

        <SelectMenu
          label="Title type"
          value={searchKind}
          options={searchKindOptions}
          onChange={(value) => this.setState({ searchKind: value as HomePageState["searchKind"] })}
        />

        <SelectMenu
          label="Release year"
          value={searchYear}
          options={searchYearOptions}
          onChange={(value) => this.setState({ searchYear: value as HomePageState["searchYear"] })}
        />

        <SelectMenu
          label="Rating"
          value={searchRating}
          options={searchRatingOptions}
          onChange={(value) => this.setState({ searchRating: value as HomePageState["searchRating"] })}
        />

        <SelectMenu
          label="Watch progress"
          value={searchProgress}
          options={searchProgressOptions}
          onChange={(value) => this.setState({ searchProgress: value as HomePageState["searchProgress"] })}
        />

      </div>

    );

  }

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

    const { view, searchQuery, history, favorites, settingsOpen, adminOpen } = this.state;

    const showSearch = searchQuery.trim().length > 0 && view !== "live";

    return (

      <div className="min-h-screen overflow-x-hidden">

        <Header

          searchQuery={searchQuery}

          onSearch={this.handleSearch}
          onOpenSettings={() => this.setState({ settingsOpen: true })}
          onOpenAdmin={() => this.setState({ adminOpen: true })}
          onLogout={this.handleLogout}

        />

        <div className="sticky top-16 z-30 border-b border-border-subtle bg-surface/80 py-3 backdrop-blur-md">

          <div className="mx-auto flex max-w-[1600px] flex-col items-center justify-center gap-3 px-4 sm:px-8 lg:flex-row lg:gap-4">

            <ViewSwitcher active={view} onChange={(v) => this.setState({ view: v })} />

            {showSearch && this.renderSearchFilters()}

          </div>

        </div>

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

      </div>

    );

  }

}
