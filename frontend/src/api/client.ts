import type { AccessCode, Category, Episode, FavoriteItem, IntroInfo, LiveChannel, NextEpisode, SearchHit, Season, StreamInfo, SubtitleTrack, TitleDetails, User, UserSettings, WatchHistoryItem, } from "@/lib/types";

export class ApiError extends Error {

  status: number;

  constructor(status: number, message: string) {

    super(message);

    this.status = status;

  }

}

async function request<T>(path: string, init?: RequestInit): Promise<T> {

  const res = await fetch(path, {

    credentials: "include",

    headers: {

      "Content-Type": "application/json",
      ...(init?.headers ?? {}),

    },

    ...init,

  });

  if (res.status === 204) {

    return undefined as T;

  }

  const data = await res.json().catch(() => ({}));

  if (!res.ok) {

    throw new ApiError(res.status, data.error ?? "request failed");

  }

  return data as T;

}

export const api = {

  register(email: string, password: string, accessCode: string) {

    return request<User>("/api/auth/register", {

      method: "POST",
      body: JSON.stringify({ email, password, accessCode }),

    });

  },

  login(email: string, password: string) {

    return request<User>("/api/auth/login", {

      method: "POST",
      body: JSON.stringify({ email, password }),

    });

  },

  logout() {

    return request<void>("/api/auth/logout", { method: "POST" });

  },

  me() {

    return request<User>("/api/auth/me");

  },

  getSettings() {

    return request<UserSettings>("/api/settings");

  },

  updateSettings(settings: Partial<UserSettings>) {

    return request<UserSettings>("/api/settings", {

      method: "PUT",
      body: JSON.stringify(settings),

    });

  },

  getHistory(limit = 50, mediaId?: number) {

    const params = new URLSearchParams({ limit: String(limit) });

    if (mediaId != null) params.set("mediaId", String(mediaId));

    return request<WatchHistoryItem[]>(`/api/history?${params}`);

  },

  upsertHistory(item: Partial<WatchHistoryItem> & { kind: string; mediaId: number; title: string }) {

    return request<WatchHistoryItem>("/api/history", {

      method: "POST",
      body: JSON.stringify(item),

    });

  },

  deleteHistory(id: string) {

    return request<void>(`/api/history/${id}`, { method: "DELETE" });

  },

  getFavorites() {

    return request<FavoriteItem[]>("/api/favorites");

  },

  upsertFavorite(item: Partial<FavoriteItem> & { kind: string; mediaId: number; title: string }) {

    return request<FavoriteItem>("/api/favorites", {

      method: "POST",
      body: JSON.stringify(item),

    });

  },

  deleteFavorite(kind: FavoriteItem["kind"], key: number | string) {

    return request<void>(`/api/favorites/${kind}/${encodeURIComponent(String(key))}`, { method: "DELETE" });

  },

  search(q: string) {

    return request<SearchHit[]>(`/api/search?q=${encodeURIComponent(q)}`);

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

  movieStream(id: number, height?: number) {

    const params = new URLSearchParams();

    if (height) params.set("height", String(height));

    const q = params.toString() ? `?${params}` : "";

    return request<StreamInfo>(`/api/movies/${id}/stream${q}`);

  },

  movieSubtitles(id: number) {

    return request<SubtitleTrack[]>(`/api/movies/${id}/subtitles`);

  },

  episodeStream(showId: number, season: number, episode: number, height?: number) {

    const params = new URLSearchParams();

    if (height) params.set("height", String(height));

    const q = params.toString() ? `?${params}` : "";

    return request<StreamInfo>(
      `/api/shows/${showId}/seasons/${season}/episodes/${episode}/stream${q}`
    );

  },

  episodeSubtitles(showId: number, season: number, episode: number) {

    return request<SubtitleTrack[]>(
      `/api/shows/${showId}/seasons/${season}/episodes/${episode}/subtitles`
    );

  },

  movieIntro(id: number, durationMs?: number) {

    const q = durationMs ? `?durationMs=${durationMs}` : "";

    return request<IntroInfo>(`/api/movies/${id}/intro${q}`);

  },

  episodeIntro(showId: number, season: number, episode: number, durationMs?: number) {

    const q = durationMs ? `?durationMs=${durationMs}` : "";

    return request<IntroInfo>(`/api/shows/${showId}/seasons/${season}/episodes/${episode}/intro${q}`);

  },

  nextEpisode(showId: number, season: number, episode: number) {

    return request<NextEpisode | null>(`/api/shows/${showId}/seasons/${season}/episodes/${episode}/next`);

  },

  liveChannels() {

    return request<LiveChannel[]>("/api/live/channels");

  },

  livePopular(limit = 24) {

    return request<LiveChannel[]>(`/api/live/channels/popular?limit=${limit}`);

  },

  liveSearch(q: string) {

    return request<LiveChannel[]>(`/api/live/channels/search?q=${encodeURIComponent(q)}&limit=48`);

  },

  liveStream(daddyId: string) {

    return request<{ url?: string; proxyUrl: string; isHls: boolean; channel: LiveChannel }>(`/api/live/channels/${daddyId}/stream`);

  },

  createAccessCode(maxUses: number, expiresIn?: string) {

    return request<AccessCode>("/api/admin/access-codes", {

      method: "POST",
      body: JSON.stringify({ maxUses, expiresIn }),

    });

  },

  listAccessCodes() {

    return request<AccessCode[]>("/api/admin/access-codes");

  },

  deleteAccessCode(code: string) {

    return request<void>(`/api/admin/access-codes/${code}`, { method: "DELETE" });

  },

};
