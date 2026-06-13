export type WatchRoute =
  | {

      valid: true;
      kind: "movie";
      id: number;

    }
  | {

      valid: true;
      kind: "show";
      showId: number;
      season: number;
      episode: number;

    }
  | {

      valid: true;
      kind: "live";
      channelId: string;

    }
  | {

      valid: false;
      reason: string;

    };

function parsePositiveInt(value: string): number | null {

  const n = Number(value);

  if (!Number.isInteger(n) || n <= 0) return null;

  return n;

}

export function parseWatchPath(path: string): WatchRoute {

  const parts = path.split("/").filter(Boolean);

  if (parts.length === 0) {

    return { valid: false, reason: "missing playback path" };

  }

  if (parts[0] === "movie") {

    if (parts.length !== 2) {

      return { valid: false, reason: "movie URL must be /watch/movie/:id" };

    }

    const id = parsePositiveInt(parts[1]);

    if (id === null) return { valid: false, reason: "invalid movie id" };

    return { valid: true, kind: "movie", id };

  }

  if (parts[0] === "show") {

    if (parts.length !== 4) {

      return { valid: false, reason: "show URL must be /watch/show/:id/:season/:episode" };

    }

    const showId = parsePositiveInt(parts[1]);
    const season = parsePositiveInt(parts[2]);
    const episode = parsePositiveInt(parts[3]);

    if (showId === null || season === null || episode === null) {

      return { valid: false, reason: "invalid show, season, or episode" };

    }

    return { valid: true, kind: "show", showId, season, episode };

  }

  if (parts[0] === "live") {

    if (parts.length !== 2 || !parts[1].trim()) {

      return { valid: false, reason: "live URL must be /live/:channelId" };

    }

    return { valid: true, kind: "live", channelId: parts[1].trim() };

  }

  return { valid: false, reason: "unrecognized playback path" };

}

export function watchPathFromLocation(pathname: string): string | null {

  const watch = pathname.match(/^\/watch\/(.+)$/);

  if (watch) return watch[1];

  const live = pathname.match(/^\/live\/([^/]+)$/);

  if (live) return `live/${live[1]}`;

  return null;

}