import type { WatchHistoryItem } from "./Library";

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
