import { api } from "@/api/client";

import { ContentRow } from "@/components/catalog/ContentRow";
import { TitleCard } from "@/components/catalog/TitleCard";

import { continueWatching, latestTitleProgress, progressLabel } from "@/lib/history";
import type { Category, FavoriteItem, SearchHit, WatchHistoryItem } from "@/lib/types";

import { Component } from "react";

interface ShowsViewProps {

  onSelect: (id: number, kind: "movie" | "show") => void;
  onFavoriteToggle: (item: FavoriteItem | SearchHit) => void;

  history: WatchHistoryItem[];
  favorites: FavoriteItem[];

}

interface ShowsViewState {

  categories: Category[];
  rows: Record<string, SearchHit[]>;
  trending: SearchHit[];
  showPosters: Record<number, string>;

  loading: boolean;

}

export class ShowsView extends Component<ShowsViewProps, ShowsViewState> {

  state: ShowsViewState = {

    categories: [],
    rows: {},
    trending: [],
    showPosters: {},

    loading: true,

  };

  async componentDidMount() {

    await this.load();

    void this.loadShowPosters(this.props.history);

  }

  componentDidUpdate(prevProps: ShowsViewProps) {

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

      const [categories, trending] = await Promise.all([

        api.showCategories(),
        api.showTrending(12),

      ]);

      const safeCategories = categories ?? [];

      const safeTrending = trending ?? [];

      this.setState({ categories: safeCategories, trending: safeTrending, loading: false });

      const rowEntries = await Promise.all(
        safeCategories.map(async (cat) => {

          try {

            const titles = await api.showCategoryTitles(cat.id);

            return [cat.id, titles] as const;

          } catch {

            return [cat.id, []] as const;

          }

        })
      );

      this.setState({ rows: Object.fromEntries(rowEntries) });

    } catch {

      this.setState({ loading: false });

    }

  };

  render() {

    const { onSelect, onFavoriteToggle, history, favorites } = this.props;

    const { categories, rows, trending, showPosters, loading } = this.state;

    const resumeItems = continueWatching(history, "show");
    const favoriteShows = favorites.filter((item) => item.kind === "show");
    const favoriteIds = new Set(favoriteShows.map((item) => item.mediaId));

    return (

      <div className="animate-fade-in py-6">

        {resumeItems.length > 0 && (

          <ContentRow title="Continue Watching">

            {resumeItems.map((item) => (

              <TitleCard

                key={item.id}
                id={item.mediaId}
                kind="show"

                title={item.title}
                poster={showPosters[item.mediaId] ?? item.poster}

                progressMs={item.positionMs}
                durationMs={item.durationMs}
                progressLabel={progressLabel(item)}

                onClick={() => onSelect(item.mediaId, "show")}

              />

            ))}

          </ContentRow>

        )}

        {favoriteShows.length > 0 && (

          <ContentRow title="Favorites">

            {favoriteShows.map((item) => {

              const progress = latestTitleProgress(history, "show", item.mediaId);

              return (

                <TitleCard

                  key={item.id}
                  id={item.mediaId}
                  kind="show"

                  title={item.title}
                  poster={item.poster}
                  year={item.year}

                  favorite
                  onFavoriteToggle={() => onFavoriteToggle(item)}

                  progressMs={progress?.positionMs}
                  durationMs={progress?.durationMs}
                  progressLabel={progressLabel(progress)}

                  onClick={() => onSelect(item.mediaId, "show")}

                />

              );

            })}

          </ContentRow>

        )}

        {trending.length > 0 && (

          <ContentRow title="Trending Now">

            {trending.map((hit) => {

              const progress = latestTitleProgress(history, "show", hit.id);

              return (

                <TitleCard

                  key={hit.id}
                  id={hit.id}
                  kind="show"

                  title={hit.title}
                  poster={hit.poster}
                  year={hit.year}

                  favorite={favoriteIds.has(hit.id)}
                  onFavoriteToggle={() => onFavoriteToggle(hit)}

                  progressMs={progress?.positionMs}
                  durationMs={progress?.durationMs}
                  progressLabel={progressLabel(progress)}

                  onClick={() => onSelect(hit.id, "show")}

                />

              );

            })}

          </ContentRow>

        )}

        {categories.map((cat) => {

          const titles = rows[cat.id];

          if (!loading && titles && titles.length === 0) return null;

          return (

            <ContentRow key={cat.id} title={cat.name} loading={loading && !titles}>

              {(titles ?? []).map((hit) => {

                const progress = latestTitleProgress(history, "show", hit.id);

                return (

                  <TitleCard

                    key={hit.id}
                    id={hit.id}
                    kind="show"

                    title={hit.title}
                    poster={hit.poster}
                    year={hit.year}

                    favorite={favoriteIds.has(hit.id)}
                    onFavoriteToggle={() => onFavoriteToggle(hit)}

                    progressMs={progress?.positionMs}
                    durationMs={progress?.durationMs}
                    progressLabel={progressLabel(progress)}

                    onClick={() => onSelect(hit.id, "show")}

                  />

                );

              })}

            </ContentRow>

          );

        })}

      </div>

    );

  }

}
