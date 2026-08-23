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
