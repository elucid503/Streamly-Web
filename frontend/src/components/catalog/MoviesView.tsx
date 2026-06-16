import { api } from "@/api/client";

import { ContentRow } from "@/components/catalog/ContentRow";
import { TitleCard } from "@/components/catalog/TitleCard";

import { continueWatching, latestTitleProgress, progressLabel } from "@/lib/history";
import type { Category, FavoriteItem, SearchHit, WatchHistoryItem } from "@/lib/types";

import { Component } from "react";

interface MoviesViewProps {

  onSelect: (id: number, kind: "movie" | "show") => void;
  onResume: (item: WatchHistoryItem) => void;
  onFavoriteToggle: (item: FavoriteItem | SearchHit) => void;

  history: WatchHistoryItem[];
  favorites: FavoriteItem[];

}

interface MoviesViewState {

  categories: Category[];
  rows: Record<string, SearchHit[]>;
  trending: SearchHit[];

  loading: boolean;

}

export class MoviesView extends Component<MoviesViewProps, MoviesViewState> {

  state: MoviesViewState = {

    categories: [],
    rows: {},
    trending: [],

    loading: true,

  };

  async componentDidMount() {

    await this.load();

  }

  load = async () => {

    try {

      const [categories, trending] = await Promise.all([

        api.movieCategories(),
        api.movieTrending(12), // initially 12 trending movies

      ]);

      const safeCategories = categories ?? [];

      const safeTrending = trending ?? [];

      this.setState({ categories: safeCategories, trending: safeTrending, loading: false });

      const rowEntries = await Promise.all(

        safeCategories.map(async (cat) => {

          try {

            const titles = await api.movieCategoryTitles(cat.id);
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

    const { onSelect, onResume, onFavoriteToggle, history, favorites } = this.props;

    const { categories, rows, trending, loading } = this.state;

    const resumeItems = continueWatching(history, "movie");
    const favoriteMovies = favorites.filter((item) => item.kind === "movie");
    const favoriteIds = new Set(favoriteMovies.map((item) => item.mediaId));

    return (

      <div className="animate-fade-in py-6">

        {resumeItems.length > 0 && (

          <ContentRow title="Continue Last" sectionId="movies-continue">

            {resumeItems.map((item) => (

              <TitleCard

                key={item.id}
                id={item.mediaId}
                kind="movie"

                title={item.title}
                poster={item.poster}

                progressMs={item.positionMs}
                durationMs={item.durationMs}
                progressLabel={progressLabel(item)}

                favorite={favoriteIds.has(item.mediaId)}
                onFavoriteToggle={() => onFavoriteToggle({
                  id: item.id,
                  kind: "movie",
                  mediaId: item.mediaId,
                  title: item.title,
                  poster: item.poster,
                  createdAt: item.updatedAt,
                })}

                onClick={() => onResume(item)}

              />

            ))}

          </ContentRow>

        )}

        {favoriteMovies.length > 0 && (

          <ContentRow title="Favorites" sectionId="movies-favorites">

            {favoriteMovies.map((item) => {

              const progress = latestTitleProgress(history, "movie", item.mediaId);

              return (

                <TitleCard

                  key={item.id}
                  id={item.mediaId}
                  kind="movie"

                  title={item.title}
                  poster={item.poster}
                  year={item.year}

                  favorite
                  onFavoriteToggle={() => onFavoriteToggle(item)}

                  progressMs={progress?.positionMs}
                  durationMs={progress?.durationMs}
                  progressLabel={progressLabel(progress)}

                  onClick={() => onSelect(item.mediaId, "movie")}

                />

              );

            })}

          </ContentRow>

        )}

        {trending.length > 0 && (

          <ContentRow title="Trending Now" sectionId="movies-trending">

            {trending.map((hit) => {

              const progress = latestTitleProgress(history, "movie", hit.id);

              return (

                <TitleCard

                  key={hit.id}
                  id={hit.id}
                  kind="movie"

                  title={hit.title}
                  poster={hit.poster}
                  year={hit.year}

                  favorite={favoriteIds.has(hit.id)}
                  onFavoriteToggle={() => onFavoriteToggle(hit)}

                  progressMs={progress?.positionMs}
                  durationMs={progress?.durationMs}
                  progressLabel={progressLabel(progress)}

                  onClick={() => onSelect(hit.id, "movie")}

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

                const progress = latestTitleProgress(history, "movie", hit.id);

                return (

                  <TitleCard

                    key={hit.id}
                    id={hit.id}
                    kind="movie"

                    title={hit.title}
                    poster={hit.poster}
                    year={hit.year}

                    favorite={favoriteIds.has(hit.id)}
                    onFavoriteToggle={() => onFavoriteToggle(hit)}

                    progressMs={progress?.positionMs}
                    durationMs={progress?.durationMs}
                    progressLabel={progressLabel(progress)}

                    onClick={() => onSelect(hit.id, "movie")}

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
