import type { IntroInfo, NextEpisode, StreamQuality, SubtitleTrack } from "@/Types";

import { request } from "./Request";

export const streamAPI = {

  movie(id: number) {

    return request<{ qualities: StreamQuality[] }>(`/api/movies/${id}/stream`);

  },

  episode(showId: number, season: number, episode: number) {

    return request<{ qualities: StreamQuality[] }>(
      `/api/shows/${showId}/seasons/${season}/episodes/${episode}/stream`
    );

  },

  movieSubtitles(id: number) {

    return request<SubtitleTrack[]>(`/api/movies/${id}/subtitles`);

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

};
