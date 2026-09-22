import { Component } from "react";

import { FeedView } from "@/Features/Catalog/FeedView";

import type { FavoriteItem, WatchHistoryItem } from "@/Features/Library/Types";
import type { FeedItem } from "@/Features/Catalog/Types";

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
