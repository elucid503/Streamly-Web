import type { Category, Episode, HomeFeed, ResolveResult, SearchHit, Season, TitleDetails } from "@/Types";

import { request } from "./Request";

export const catalogAPI = {

  search(q: string) {

    return request<SearchHit[]>(`/api/search?q=${encodeURIComponent(q)}`);

  },

  homeFeed(kind: "movie" | "show") {

    return request<HomeFeed>(kind === "movie" ? "/api/feed/movies" : "/api/feed/shows");

  },

  resolveTitle(item: { tmdbId?: number; kind: "movie" | "show"; title?: string; year?: number }) {

    const params = new URLSearchParams({

      kind: item.kind,
      tmdbId: String(item.tmdbId ?? 0),

    });

    if (item.title) params.set("title", item.title);

    if (item.year) params.set("year", String(item.year));

    return request<ResolveResult>(`/api/resolve?${params}`);

  },

  movieTrending(limit = 12) {

    return request<SearchHit[]>(`/api/movies/trending?limit=${limit}`);

  },

  showTrending(limit = 12) {

    return request<SearchHit[]>(`/api/shows/trending?limit=${limit}`);

  },

  movieCategories() {

    return request<Category[]>("/api/movies/categories");

  },

  showCategories() {

    return request<Category[]>("/api/shows/categories");

  },

  movieCategoryTitles(id: string, page = 1) {

    return request<SearchHit[]>(`/api/movies/categories/${id}?page=${page}&limit=24`);

  },

  showCategoryTitles(id: string, page = 1) {

    return request<SearchHit[]>(`/api/shows/categories/${id}?page=${page}&limit=24`);

  },

  movieDetails(id: number) {

    return request<TitleDetails>(`/api/movies/${id}`);

  },

  showDetails(id: number) {

    return request<TitleDetails>(`/api/shows/${id}`);

  },

  showSeasons(id: number) {

    return request<Season[]>(`/api/shows/${id}/seasons`);

  },

  seasonEpisodes(showId: number, season: number) {

    return request<Episode[]>(`/api/shows/${showId}/seasons/${season}/episodes`);

  },

  episodeDetails(showId: number, season: number, episode: number) {

    return request<Episode>(`/api/shows/${showId}/seasons/${season}/episodes/${episode}`);

  },

};
