import { api } from "@/api/client";

import { ContentRow } from "@/components/catalog/ContentRow";
import { FeaturedHero } from "@/components/catalog/FeaturedHero";
import { TitleCard } from "@/components/catalog/TitleCard";

import { continueWatching, latestTitleProgress, progressLabel, resumePath } from "@/lib/history";
import type { FavoriteItem, FeedItem, HomeFeed, WatchHistoryItem } from "@/lib/types";

import { Component } from "react";

interface FeedViewProps {

  kind: "movie" | "show";

  onSelect: (id: number, kind: "movie" | "show") => void;
  onResumeWatching: (path: string) => void;
  onFavoriteToggle: (item: FavoriteItem | FeedItem) => void;
  onRemoveFromHistory: (historyId: string) => void;

  history: WatchHistoryItem[];
  favorites: FavoriteItem[];

}

interface FeedViewState {

  feed: HomeFeed | null;
  showPosters: Record<number, string>;
  loading: boolean;
  resolvingKey: string | null;

}

export class FeedView extends Component<FeedViewProps, FeedViewState> {

  state: FeedViewState = {

    feed: null,
    showPosters: {},
    loading: true,
    resolvingKey: null,

  };

  componentDidMount() {

    void this.load();

    if (this.props.kind === "show") {

      void this.loadShowPosters(this.props.history);

    }

  }

  componentDidUpdate(prevProps: FeedViewProps) {

    if (prevProps.history !== this.props.history && this.props.kind === "show") {

      void this.loadShowPosters(this.props.history);

    }

  }

  loadShowPosters = async (history: WatchHistoryItem[]) => {

    const resumeItems = continueWatching(history, "show");

    if (resumeItems.length === 0) {

      this.setState({ showPosters: {} });

      return;

    }

    const entries = await Promise.all(

      resumeItems.map(async (item) => {

        try {

          const details = await api.showDetails(item.mediaId);

          return [item.mediaId, details.poster] as const;

        } catch {

          return [item.mediaId, ""] as const;

        }

      })

    );

    this.setState({

      showPosters: Object.fromEntries(entries.filter(([, poster]) => poster)),

    });

  };

  load = async () => {

    try {

      const feed = await api.homeFeed(this.props.kind);

      this.setState({ feed: feed ?? { sections: [], refreshedAt: "" }, loading: false });

    } catch {

      this.setState({ loading: false, feed: { sections: [], refreshedAt: "" } });

    }

  };

  itemKey = (item: FeedItem) => {

    if (item.tmdbId) return `tmdb:${item.tmdbId}`;

    return `id:${item.id}`;

  };

  favoriteIds = () => {

    const { favorites, kind } = this.props;

    return new Set(favorites.filter((item) => item.kind === kind).map((item) => item.mediaId));

  };

  ensureShowboxId = async (item: FeedItem): Promise<number> => {

    if (item.id > 0) return item.id;

    if (!item.tmdbId) {

      throw new Error("missing title id");

    }

    const resolved = await api.resolveTitle({

      tmdbId: item.tmdbId,
      kind: item.kind,
      title: item.title,
      year: item.year,

    });

    // Warm local feed state so later clicks skip resolve.
    this.setState((state) => {

      if (!state.feed) return null;

      const patch = (entries: FeedItem[]) => entries.map((entry) => {

        if (entry.tmdbId === item.tmdbId) {

          return { ...entry, id: resolved.id };

        }

        return entry;

      });

      return {

        feed: {

          ...state.feed,
          featured: state.feed.featured ? patch(state.feed.featured) : state.feed.featured,
          sections: state.feed.sections.map((section) => ({

            ...section,
            items: patch(section.items),

          })),

        },

      };

    });

    return resolved.id;

  };

  openItem = async (item: FeedItem) => {

    const key = this.itemKey(item);

    this.setState({ resolvingKey: key });

    try {

      const id = await this.ensureShowboxId(item);

      this.props.onSelect(id, this.props.kind);

    } catch {

      // Keep browsing — resolve failures are expected for obscure titles.
    } finally {

      this.setState({ resolvingKey: null });

    }

  };

  toggleFavorite = async (item: FeedItem) => {

    try {

      const id = await this.ensureShowboxId(item);

      this.props.onFavoriteToggle({ ...item, id });

    } catch {

      /* ignore */

    }

  };

  renderCard = (hit: FeedItem) => {

    const { kind, history, onResumeWatching, onRemoveFromHistory } = this.props;
    const favoriteIds = this.favoriteIds();
    const showboxId = hit.id > 0 ? hit.id : 0;

    const progress = showboxId ? latestTitleProgress(history, kind, showboxId) : undefined;
    const resumable = progress ? resumePath(progress) : null;

    return (

      <TitleCard

        key={this.itemKey(hit)}
        id={showboxId || hit.tmdbId || 0}
        kind={kind}

        title={hit.title}
        poster={hit.poster}
        year={hit.year}
        rating={hit.rating}
        genres={hit.genres}

        favorite={showboxId > 0 && favoriteIds.has(showboxId)}
        onFavoriteToggle={() => void this.toggleFavorite(hit)}

        progressMs={progress?.positionMs}
        durationMs={progress?.durationMs}
        progressLabel={progressLabel(progress)}

        onResume={resumable ? () => onResumeWatching(resumable) : undefined}
        onRemoveFromHistory={progress ? () => onRemoveFromHistory(progress.id) : undefined}

        onClick={() => void this.openItem(hit)}

      />

    );

  };

  render() {

    const { kind, history, favorites, onResumeWatching, onFavoriteToggle, onRemoveFromHistory } = this.props;
    const { feed, showPosters, loading } = this.state;

    const resumeItems = continueWatching(history, kind);
    const kindFavorites = favorites.filter((item) => item.kind === kind);
    const favoriteIds = this.favoriteIds();
    const featured = feed?.featured ?? [];
    const featuredKeys = new Set(featured.map((item) => this.itemKey(item)));

    return (

      <div className="animate-fade-in py-6">

        {loading && !feed && (

          <>

            <div className="mb-8 px-4 sm:px-8">

              <div className="skeleton h-[280px] w-full rounded-xl sm:h-[320px]" />

            </div>

            <ContentRow title="" loading />

            <ContentRow title="" loading />

          </>

        )}

        {featured.length > 0 && (

          <FeaturedHero

            items={featured}
            favoriteIds={favoriteIds}
            onPlay={(item) => void this.openItem(item)}
            onFavoriteToggle={(item) => void this.toggleFavorite(item)}

          />

        )}

        {resumeItems.length > 0 && (

          <ContentRow title="Continue Watching" sectionId={`${kind}s-continue`}>

            {resumeItems.map((item) => (

              <TitleCard

                key={item.id}
                id={item.mediaId}
                kind={kind}

                title={item.title}
                poster={kind === "show" ? (showPosters[item.mediaId] || item.poster) : item.poster}

                progressMs={item.positionMs}
                durationMs={item.durationMs}
                progressLabel={progressLabel(item)}

                favorite={favoriteIds.has(item.mediaId)}
                onFavoriteToggle={() => onFavoriteToggle({

                  id: item.id,
                  kind,
                  mediaId: item.mediaId,
                  title: item.title,
                  poster: item.poster,
                  createdAt: item.updatedAt,

                })}

                onResume={() => onResumeWatching(resumePath(item)!)}
                onRemoveFromHistory={() => onRemoveFromHistory(item.id)}

                onClick={() => onResumeWatching(resumePath(item)!)}

              />

            ))}

          </ContentRow>

        )}

        {kindFavorites.length > 0 && (

          <ContentRow title="Favorites" sectionId={`${kind}s-favorites`}>

            {kindFavorites.map((item) => {

              const progress = latestTitleProgress(history, kind, item.mediaId);
              const resumable = progress ? resumePath(progress) : null;

              return (

                <TitleCard

                  key={item.id}
                  id={item.mediaId}
                  kind={kind}

                  title={item.title}
                  poster={item.poster}
                  year={item.year}
                  rating={item.rating}

                  favorite
                  onFavoriteToggle={() => onFavoriteToggle(item)}

                  progressMs={progress?.positionMs}
                  durationMs={progress?.durationMs}
                  progressLabel={progressLabel(progress)}

                  onResume={resumable ? () => onResumeWatching(resumable) : undefined}
                  onRemoveFromHistory={progress ? () => onRemoveFromHistory(progress.id) : undefined}

                  onClick={() => this.props.onSelect(item.mediaId, kind)}

                />

              );

            })}

          </ContentRow>

        )}

        {(feed?.sections ?? []).map((section) => {

          if (!loading && section.items.length === 0) return null;

          const items = section.kind === "personalized"
            ? section.items.filter((item) => !featuredKeys.has(this.itemKey(item)))
            : section.items;

          if (!loading && items.length === 0) return null;

          return (

            <ContentRow

              key={section.id}
              title={section.title}
              subtitle={section.subtitle}
              sectionId={`${kind}s-${section.id}`}
              loading={loading && items.length === 0}

            >

              {items.map((hit) => this.renderCard(hit))}

            </ContentRow>

          );

        })}

      </div>

    );

  }

}
