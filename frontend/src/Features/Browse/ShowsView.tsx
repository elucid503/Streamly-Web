import { FeedView } from "@/Features/Browse/FeedView";

import type { FavoriteItem, FeedItem, WatchHistoryItem } from "@/Types";

import { Component } from "react";

interface ShowsViewProps {

  onSelect: (id: number, kind: "movie" | "show") => void;
  onFavoriteToggle: (item: FavoriteItem | FeedItem) => void;
  onResumeWatching: (path: string) => void;
  onRemoveFromHistory: (historyId: string) => void;

  history: WatchHistoryItem[];
  favorites: FavoriteItem[];

}

/** Combined Movies & Shows browse (Streamly-Redux-style VOD home). */
export class ShowsView extends Component<ShowsViewProps> {

  render() {

    return (

      <FeedView {...this.props} />

    );

  }

}
