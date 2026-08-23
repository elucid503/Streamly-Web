export type MainView = "vod" | "live" | "sports" | "friends";

export interface SearchHit {

  id: number;
  kind: "movie" | "show";

  title: string;
  year: number;
  poster: string;

  description: string;
  rating: string;

  backdrop?: string;
  genres?: string[];
  runtime?: number;
  matchReason?: string;

}

export interface FeedItem {

  id: number;
  tmdbId?: number;
  kind: "movie" | "show";

  title: string;
  year: number;
  poster: string;

  description: string;
  rating: string;

  backdrop?: string;
  genres?: string[];
  runtime?: number;
  matchReason?: string;

}

export interface ResolveResult {

  id: number;
  tmdbId: number;
  kind: "movie" | "show";
  title: string;
  year: number;
  poster: string;

}

export interface FeedSection {

  id: string;
  title: string;
  subtitle?: string;
  kind: string;
  items: FeedItem[];

}

export interface HomeFeed {

  featured?: FeedItem[];
  sections: FeedSection[];
  refreshedAt: string;

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
