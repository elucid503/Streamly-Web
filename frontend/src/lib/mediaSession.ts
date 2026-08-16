interface MediaSessionInfo {

  title: string;
  artist?: string;
  album?: string;
  artwork?: string;

}

interface MediaSessionHandlers {

  onPlay?: () => void;
  onPause?: () => void;
  onSeekBackward?: () => void;
  onSeekForward?: () => void;
  onSeekTo?: (positionMs: number) => void;
  onNextTrack?: () => void;

}

// Safari ships audioSession outside the DOM types, so the cast lives here instead of at every call site.
interface AudioSessionNavigator extends Navigator {

  audioSession?: { type: string };

}

const ARTWORK_SIZES = ["256x256", "384x384", "512x512"];

function available(): boolean {

  return typeof navigator !== "undefined" && "mediaSession" in navigator;

}

export function setMediaSessionMetadata(info: MediaSessionInfo): void {

  if (!available()) {

    return;

  }

  const poster = info.artwork;

  const artwork = poster ? ARTWORK_SIZES.map((sizes) => ({ src: poster, sizes, type: "image/jpeg" })) : [];

  try {

    navigator.mediaSession.metadata = new MediaMetadata({

      title: info.title || "Streamly",
      artist: info.artist || "",
      album: info.album || "",

      artwork,

    });

  } catch {

    /* metadata is best effort */

  }

}

export function setMediaSessionPlaybackState(state: "playing" | "paused" | "none"): void {

  if (!available()) {

    return;

  }

  try {

    navigator.mediaSession.playbackState = state;

  } catch {

    /* ignore */

  }

}

export function setMediaSessionPosition(durationMs: number, positionMs: number, playbackRate = 1): void {

  if (!available() || typeof navigator.mediaSession.setPositionState !== "function") {

    return;

  }

  if (!Number.isFinite(durationMs) || durationMs <= 0) {

    return;

  }

  try {

    navigator.mediaSession.setPositionState({

      duration: durationMs / 1000,
      position: Math.min(Math.max(positionMs, 0), durationMs) / 1000,
      playbackRate: playbackRate > 0 ? playbackRate : 1,

    });

  } catch {

    /* Safari throws while the element is still loading */

  }

}

export function setMediaSessionHandlers(handlers: MediaSessionHandlers): void {

  if (!available()) {

    return;

  }

  const bind = (action: MediaSessionAction, handler: ((details: MediaSessionActionDetails) => void) | undefined) => {

    try {

      navigator.mediaSession.setActionHandler(action, handler ?? null);

    } catch {

      /* action unsupported on this browser */

    }

  };

  bind("play", handlers.onPlay ? () => handlers.onPlay?.() : undefined);
  bind("pause", handlers.onPause ? () => handlers.onPause?.() : undefined);

  bind("seekbackward", handlers.onSeekBackward ? () => handlers.onSeekBackward?.() : undefined);
  bind("seekforward", handlers.onSeekForward ? () => handlers.onSeekForward?.() : undefined);

  bind("nexttrack", handlers.onNextTrack ? () => handlers.onNextTrack?.() : undefined);

  bind("seekto", handlers.onSeekTo ? (details) => {

    const seconds = details.seekTime;

    if (typeof seconds === "number") {

      handlers.onSeekTo?.(seconds * 1000);

    }

  } : undefined);

}

export function clearMediaSession(): void {

  if (!available()) {

    return;

  }

  setMediaSessionHandlers({});

  try {

    navigator.mediaSession.metadata = null;
    navigator.mediaSession.playbackState = "none";

  } catch {

    /* ignore */

  }

}

// iOS only keeps audio running in a backgrounded PWA when the page claims a playback audio session.
export function enableBackgroundAudio(): void {

  if (typeof navigator === "undefined") {

    return;

  }

  const audioNavigator = navigator as AudioSessionNavigator;

  if (!audioNavigator.audioSession) {

    return;

  }

  try {

    audioNavigator.audioSession.type = "playback";

  } catch {

    /* not supported */

  }

}
