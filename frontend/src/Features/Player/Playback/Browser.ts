type NativeFullscreenVideo = HTMLVideoElement & {

  webkitSupportsFullscreen?: boolean;
  webkitEnterFullscreen?: () => void;

};

export const nativeFullscreenVideo = (video: HTMLVideoElement | null): NativeFullscreenVideo | null => {

  const el = video as NativeFullscreenVideo | null;

  return el && typeof el.webkitEnterFullscreen === "function" ? el : null;

};

export const readPortrait = (): boolean => {

  if (typeof window === "undefined") {

    return false;

  }

  return window.matchMedia("(orientation: portrait)").matches;

};

export const readStoredVolume = (): { volume: number; muted: boolean } => {

  try {

    const v = parseFloat(localStorage.getItem("player:volume") ?? "");

    return {

      volume: Number.isFinite(v) && v >= 0 && v <= 1 ? v : 1,
      muted: localStorage.getItem("player:muted") === "true",

    };

  } catch {

    return { volume: 1, muted: false };

  }

};
