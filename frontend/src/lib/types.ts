export interface User {

  id: string;
  email: string;

  isAdmin: boolean;

}

export interface UserSettings {

  preferredHeight: number;
  autoPlayNext: boolean;
  skipIntro: boolean;
  disablePauseOverlay: boolean;
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
  enriched?: boolean;

}

export interface SportsChannel {

  daddyId: string;
  name: string;
  logo: string;
  enriched: boolean;

}

export interface SportsEvent {

  title: string;
  league: string;
  time: string;
  startsAt: number;
  live: boolean;
  channels: SportsChannel[];

}

export interface WatchHistoryItem {

  id: string;
  kind: string;
  mediaId: number;
  title: string;
  poster: string;

  season?: number;
  episode?: number;
  episodeTitle?: string;
  channelId?: string;

  positionMs: number;
  durationMs: number;
  completed: boolean;
  updatedAt: string;

}

export interface FavoriteItem {

  id: string;
  kind: "movie" | "show" | "live";
  mediaId: number;
  channelId?: string;

  title: string;
  poster: string;
  year?: number;
  rating?: string;
  category?: string;

  createdAt: string;

}

export interface AccessCode {

  id: string;
  code: string;
  maxUses: number;
  uses: number;

  expiresAt?: string;
  createdAt: string;

}

export interface ProgramEntry {

  title: string;
  startsAt: number; // Unix seconds
  runtime: number; // minutes
  image?: string;

}

export interface ChannelGuideEntry {

  channel: LiveChannel;
  current?: ProgramEntry;
  next?: ProgramEntry;

}

export interface ServiceInterruption {

  id?: string;
  enabled: boolean;
  title: string;
  message: string;
  updatedAt?: string;

}

export type MainView = "shows" | "movies" | "live" | "friends";

export interface ProfileMedia {

  mediaId: number;
  title: string;
  poster: string;
  year?: number;
  kind: "movie" | "show";

}

export interface UserProfile {

  id: string;
  userId: string;

  displayName: string;
  bio: string;
  accentColor: string;
  banner: string;

  favoriteMovies: ProfileMedia[];
  favoriteShows: ProfileMedia[];
  historyVisible: boolean;
  discoverVisible?: boolean;

  updatedAt: string;

}

export interface FriendSummary {

  userId: string;
  email: string;
  displayName: string;
  accentColor: string;
  banner: string;
  friendStatus: "none" | "pending_sent" | "pending_received" | "friends";

}

export interface PublicProfile {

  userId: string;
  email: string;
  displayName: string;
  bio: string;
  accentColor: string;
  banner: string;

  favoriteMovies: ProfileMedia[];
  favoriteShows: ProfileMedia[];
  recentHistory: WatchHistoryItem[];

  friendStatus: "none" | "pending_sent" | "pending_received" | "friends";

}

export interface FriendRequestItem {

  id: string;
  userId: string;
  email: string;
  displayName: string;
  accentColor: string;
  banner: string;
  createdAt: string;
  direction: "incoming" | "outgoing";

}
