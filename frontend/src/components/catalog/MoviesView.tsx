import { FeedView } from "@/components/catalog/FeedView";

import type { FavoriteItem, FeedItem, WatchHistoryItem } from "@/lib/types";

import { Component } from "react";

interface MoviesViewProps {

  onSelect: (id: number, kind: "movie" | "show") => void;
  onResumeWatching: (path: string) => void;
  onFavoriteToggle: (item: FavoriteItem | FeedItem) => void;
  onRemoveFromHistory: (historyId: string) => void;

  history: WatchHistoryItem[];
  favorites: FavoriteItem[];

}

export class MoviesView extends Component<MoviesViewProps> {

  render() {

    return (

      <FeedView kind="movie" {...this.props} />

    );

  }

}
