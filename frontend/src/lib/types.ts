export interface User {
  id: string;
  email: string;
  isAdmin: boolean;
}

export interface UserSettings {
  preferredHeight: number;
  autoPlayNext: boolean;
  skipIntro: boolean;
  ambienceEnabled: boolean;
  subtitlesEnabled: boolean;
}

export interface SearchHit {
  id: number;
  kind: "movie" | "show";
  title: string;
  year: number;
  poster: string;
  description: string;
  rating: string;
}

export interface TitleDetails {
  id: number;
  kind: "movie" | "show";
  title: string;
  year: string;
  poster: string;
  banner?: string;
  description: string;
  rating: string;
}

export interface Category {
  id: string;
  name: string;
  kind: string;
}

export interface Season {
  number: number;
  label: string;
}

export interface Episode {
  season: number;
  episode: number;
  title: string;
  description?: string;
  poster?: string;
}

export interface StreamQuality {
  label: string;
  height: number;
  isHls: boolean;
  url: string;
  proxyUrl?: string;
}

export interface StreamInfo {
  qualities: StreamQuality[];
  url: string;
  proxyUrl?: string;
  isHls: boolean;
  selectedHeight?: number;
}

export interface SubtitleTrack {
  id: string;
  label: string;
  language: string;
  format: string;
  proxyUrl: string;
  source?: "file" | "hls" | "febbox" | "subdl";
}

export interface IntroInfo {
  introStartMs?: number;
  introEndMs?: number;
  creditsStartMs?: number;
}

export interface NextEpisode {
  season: number;
  episode: number;
  title: string;
}

export interface LiveChannel {
  id: string;
  daddyId: string;
  name: string;
  slug: string;
  logo: string;
  country: string;
  category: string;
}

export interface WatchHistoryItem {
  id: string;
  kind: string;
  mediaId: number;
  title: string;
  poster: string;
  season?: number;
  episode?: number;
  channelId?: string;
  positionMs: number;
  durationMs: number;
  completed: boolean;
  updatedAt: string;
}

export interface AccessCode {
  id: string;
  code: string;
  maxUses: number;
  uses: number;
  expiresAt?: string;
  createdAt: string;
}

export type MainView = "shows" | "movies" | "live";