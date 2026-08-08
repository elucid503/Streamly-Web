import { api } from "@/api/client";

import { ContentRow } from "@/components/catalog/ContentRow";
import { FeaturedHero } from "@/components/catalog/FeaturedHero";
import { TitleCard } from "@/components/catalog/TitleCard";

import { continueWatching, latestTitleProgress, progressLabel, resumePath } from "@/lib/history";
import type { FavoriteItem, FeedItem, FeedSection, HomeFeed, WatchHistoryItem } from "@/lib/types";

import { Component } from "react";

interface FeedViewProps {

  onSelect: (id: number, kind: "movie" | "show") => void;
  onResumeWatching: (path: string) => void;
  onFavoriteToggle: (item: FavoriteItem | FeedItem) => void;
  onRemoveFromHistory: (historyId: string) => void;

  history: WatchHistoryItem[];
  favorites: FavoriteItem[];

}

interface FeedViewState {

  movieFeed: HomeFeed | null;
  showFeed: HomeFeed | null;
  showPosters: Record<number, string>;
  loading: boolean;
  resolvingKey: string | null;

}

function interleaveFeatured(movies: FeedItem[], shows: FeedItem[]): FeedItem[] {

  const out: FeedItem[] = [];
  const max = Math.max(movies.length, shows.length);

  for (let i = 0; i < max; i++) {

    if (movies[i]) out.push(movies[i]!);
    if (shows[i]) out.push(shows[i]!);

  }

  return out;

}

export class FeedView extends Component<FeedViewProps, FeedViewState> {

  state: FeedViewState = {

    movieFeed: null,
    showFeed: null,
    showPosters: {},
    loading: true,
    resolvingKey: null,

  };

  componentDidMount() {

    void this.load();
    void this.loadShowPosters(this.props.history);

  }

  componentDidUpdate(prevProps: FeedViewProps) {

    if (prevProps.history !== this.props.history) {

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

      const [movieFeed, showFeed] = await Promise.all([

        api.homeFeed("movie"),
        api.homeFeed("show"),

      ]);

      this.setState({

        movieFeed: movieFeed ?? { sections: [], refreshedAt: "" },
        showFeed: showFeed ?? { sections: [], refreshedAt: "" },
        loading: false,

      });

    } catch {

      this.setState({

        loading: false,
        movieFeed: { sections: [], refreshedAt: "" },
        showFeed: { sections: [], refreshedAt: "" },

      });

    }

  };

  itemKey = (item: FeedItem) => {

    if (item.tmdbId) return `${item.kind}:tmdb:${item.tmdbId}`;

    return `${item.kind}:id:${item.id}`;

  };

  favoriteIds = () => {

    const { favorites } = this.props;

    return new Set(

      favorites
        .filter((item) => item.kind === "movie" || item.kind === "show")
        .map((item) => `${item.kind}:${item.mediaId}`)

    );

  };

  isFavorite = (kind: "movie" | "show", mediaId: number) => {

    return this.favoriteIds().has(`${kind}:${mediaId}`);

  };

  patchFeeds = (item: FeedItem, id: number) => {

    this.setState((state) => {

      const patch = (entries: FeedItem[]) => entries.map((entry) => {

        if (entry.kind === item.kind && entry.tmdbId === item.tmdbId) {

          return { ...entry, id };

        }

        return entry;

      });

      const patchFeed = (feed: HomeFeed | null): HomeFeed | null => {

        if (!feed) return feed;

        return {

          ...feed,
          featured: feed.featured ? patch(feed.featured) : feed.featured,
          sections: feed.sections.map((section) => ({

            ...section,
            items: patch(section.items),

          })),

        };

      };

      return {

        movieFeed: patchFeed(state.movieFeed),
        showFeed: patchFeed(state.showFeed),

      };

    });

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

    this.patchFeeds(item, resolved.id);

    return resolved.id;

  };

  openItem = async (item: FeedItem) => {

    const key = this.itemKey(item);

    this.setState({ resolvingKey: key });

    try {

      const id = await this.ensureShowboxId(item);

      this.props.onSelect(id, item.kind);

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

    const { history, onResumeWatching, onRemoveFromHistory } = this.props;
    const showboxId = hit.id > 0 ? hit.id : 0;

    const progress = showboxId ? latestTitleProgress(history, hit.kind, showboxId) : undefined;
    const resumable = progress ? resumePath(progress) : null;

    return (

      <TitleCard

        key={this.itemKey(hit)}
        id={showboxId || hit.tmdbId || 0}
        kind={hit.kind}

        title={hit.title}
        poster={hit.poster}
        year={hit.year}
        rating={hit.rating}
        genres={hit.genres}

        favorite={showboxId > 0 && this.isFavorite(hit.kind, showboxId)}
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

  renderSections = (feed: HomeFeed | null, prefix: string, featuredKeys: Set<string>, loading: boolean) => {

    return (feed?.sections ?? []).map((section: FeedSection) => {

      if (!loading && section.items.length === 0) return null;

      const items = section.kind === "personalized"
        ? section.items.filter((item) => !featuredKeys.has(this.itemKey(item)))
        : section.items;

      if (!loading && items.length === 0) return null;

      return (

        <ContentRow

          key={`${prefix}-${section.id}`}
          title={section.title}
          subtitle={section.subtitle}
          sectionId={`${prefix}-${section.id}`}
          loading={loading && items.length === 0}

        >

          {items.map((hit) => this.renderCard(hit))}

        </ContentRow>

      );

    });

  };

  render() {

    const { history, favorites, onResumeWatching, onFavoriteToggle, onRemoveFromHistory } = this.props;
    const { movieFeed, showFeed, showPosters, loading } = this.state;

    const resumeItems = continueWatching(history, "vod");
    const kindFavorites = favorites.filter((item) => item.kind === "movie" || item.kind === "show");

    const featured = interleaveFeatured(movieFeed?.featured ?? [], showFeed?.featured ?? []);
    const featuredKeys = new Set(featured.map((item) => this.itemKey(item)));
    const featuredFavoriteIds = new Set(

      featured
        .filter((item) => item.id > 0 && this.isFavorite(item.kind, item.id))
        .map((item) => item.id)

    );

    const ready = movieFeed !== null || showFeed !== null;

    return (

      <div className="animate-fade-in py-8">

        {loading && !ready && (

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
            favoriteIds={featuredFavoriteIds}
            onPlay={(item) => void this.openItem(item)}
            onFavoriteToggle={(item) => void this.toggleFavorite(item)}

          />

        )}

        {resumeItems.length > 0 && (

          <ContentRow title="Continue Watching" sectionId="vod-continue">

            {resumeItems.map((item) => {

              const kind = item.kind as "movie" | "show";

              return (

                <TitleCard

                  key={item.id}
                  id={item.mediaId}
                  kind={kind}

                  title={item.title}
                  poster={kind === "show" ? (showPosters[item.mediaId] || item.poster) : item.poster}

                  progressMs={item.positionMs}
                  durationMs={item.durationMs}
                  progressLabel={progressLabel(item)}

                  favorite={this.isFavorite(kind, item.mediaId)}
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

              );

            })}

          </ContentRow>

        )}

        {kindFavorites.length > 0 && (

          <ContentRow title="Favorites" sectionId="vod-favorites">

            {kindFavorites.map((item) => {

              const kind = item.kind as "movie" | "show";
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

        {this.renderSections(showFeed, "shows", featuredKeys, loading)}

        {this.renderSections(movieFeed, "movies", featuredKeys, loading)}

      </div>

    );

  }

}
