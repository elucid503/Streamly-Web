import type { WatchHistoryItem } from "@/lib/types";
import { formatDuration, progressPercent } from "@/lib/utils";

export function continueWatching(
  history: WatchHistoryItem[],
  kind: "movie" | "show"
): WatchHistoryItem[] {

  const seen = new Set<number>();

  return history
    .filter((item) => item.kind === kind && !item.completed)
    .sort((a, b) => Date.parse(b.updatedAt) - Date.parse(a.updatedAt))
    .filter((item) => {

      if (seen.has(item.mediaId)) return false;

      seen.add(item.mediaId);

      return true;

    });

}

export function resumePath(item: WatchHistoryItem): string | null {

  if (item.kind === "movie") {

    return `/watch/movie/${item.mediaId}`;

  }

  if (item.kind === "show" && item.season && item.episode) {

    return `/watch/show/${item.mediaId}/${item.season}/${item.episode}`;

  }

  return null;

}

export function latestTitleProgress(
  history: WatchHistoryItem[],
  kind: "movie" | "show",
  mediaId: number
): WatchHistoryItem | undefined {

  return history
    .filter((item) => item.kind === kind && item.mediaId === mediaId && !item.completed)
    .sort((a, b) => Date.parse(b.updatedAt) - Date.parse(a.updatedAt))[0];

}

export function progressLabel(item?: WatchHistoryItem): string | undefined {

  if (!item || item.completed) return undefined;

  if (item.kind === "show" && item.season && item.episode) {

    return `S${String(item.season).padStart(2, "0")}E${String(item.episode).padStart(2, "0")}`;

  }

  if (item.positionMs > 0) return formatDuration(item.positionMs);

  return undefined;

}

export function showResumeItem(
  history: WatchHistoryItem[],
  mediaId: number
): WatchHistoryItem | undefined {

  const showItems = history
    .filter((item) => item.kind === "show" && item.mediaId === mediaId)
    .sort((a, b) => Date.parse(b.updatedAt) - Date.parse(a.updatedAt));

  return showItems.find((item) => !item.completed) ?? showItems[0];

}

export function showEpisodeHistory(
  history: WatchHistoryItem[],
  mediaId: number,
  season: number,
  episode: number
): WatchHistoryItem | undefined {

  return history.find(
    (item) =>
      item.kind === "show" &&
      item.mediaId === mediaId &&
      item.season === season &&
      item.episode === episode
  );

}

export function episodeProgressPercent(item?: WatchHistoryItem): number {

  if (!item) return 0;

  if (item.completed) return 100;

  return progressPercent(item.positionMs, item.durationMs);

}