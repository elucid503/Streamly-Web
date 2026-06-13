import { Component } from "react";
import { api } from "@/api/client";
import type { NavigateFn } from "@/lib/navigation";
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
import { resumePath } from "@/lib/history";
import type { LiveChannel, MainView, SearchHit, WatchHistoryItem } from "@/lib/types";
import { store } from "@/lib/store";

interface HomePageProps {
  navigate: NavigateFn;
}

interface HomePageState {
  view: MainView;
  searchQuery: string;
  searchResults: SearchHit[];
  searching: boolean;
  history: WatchHistoryItem[];
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
    searching: false,
    history: [],
    settingsOpen: false,
    adminOpen: false,
  };

  componentDidMount() {
    this.unsubscribe = store.subscribe(() => this.forceUpdate());
    this.loadHistory();
  }

  componentWillUnmount() {
    this.unsubscribe();
    if (this.searchDebounce) clearTimeout(this.searchDebounce);
  }

  loadHistory = async () => {
    try {
      const history = await api.getHistory();
      this.setState({ history: history ?? [] });
    } catch {
      /* ignore */
    }
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

  renderSearchResults() {
    const { searchResults, searching } = this.state;
    const shows = searchResults.filter((h) => h.kind === "show");
    const movies = searchResults.filter((h) => h.kind === "movie");

    if (!searching && searchResults.length === 0) {
      return (
        <div className="px-4 py-16 text-center text-sm text-foreground-muted sm:px-8">
          No results found
        </div>
      );
    }

    return (
      <div className="py-6">
        {shows.length > 0 && (
          <ContentRow title="TV Shows">
            {shows.map((hit) => (
              <TitleCard
                key={`show-${hit.id}`}
                id={hit.id}
                kind="show"
                title={hit.title}
                poster={hit.poster}
                year={hit.year}
                onClick={() => this.handleSelect(hit.id, hit.kind)}
              />
            ))}
          </ContentRow>
        )}
        {movies.length > 0 && (
          <ContentRow title="Movies">
            {movies.map((hit) => (
              <TitleCard
                key={`movie-${hit.id}`}
                id={hit.id}
                kind="movie"
                title={hit.title}
                poster={hit.poster}
                year={hit.year}
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
    const { view, searchQuery, history, settingsOpen, adminOpen } = this.state;
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
          <ViewSwitcher active={view} onChange={(v) => this.setState({ view: v })} />
        </div>

        {showSearch ? (
          this.renderSearchResults()
        ) : (
          <ViewCarousel
            active={view}
            panels={{
              shows: (
                <ShowsView onSelect={this.handleSelect} history={history} />
              ),
              movies: (
                <MoviesView
                  onSelect={this.handleSelect}
                  onResume={this.handleResume}
                  history={history}
                />
              ),
              live: <LiveView onSelect={this.handleLiveSelect} searchQuery={searchQuery} />,
            }}
          />
        )}

        <SettingsPanel
          open={settingsOpen}
          onClose={() => this.setState({ settingsOpen: false })}
        />
        <AdminPanel open={adminOpen} onClose={() => this.setState({ adminOpen: false })} />
      </div>
    );
  }
}
