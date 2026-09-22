import { isIOS } from "@/Utils/Platform";

export type WebKitPlaybackTargetAvailability = "available" | "not-available";

export interface WebKitPlaybackTargetAvailabilityEvent extends Event {

  availability: WebKitPlaybackTargetAvailability;

}

export interface AirPlayVideoElement extends HTMLVideoElement {

  disableRemotePlayback: boolean;
  webkitShowPlaybackTargetPicker?: () => void;
  webkitCurrentPlaybackTargetIsWireless?: boolean;

}

export function forceIosMediaProxy(): boolean {

  return isIOS();

}

export function iosProxyQuery(): string {

  return forceIosMediaProxy() ? "proxy=1" : "";

}

export function withIosProxyQuery(path: string): string {

  const extra = iosProxyQuery();

  if (!extra) return path;

  return path.includes("?") ? `${path}&${extra}` : `${path}?${extra}`;

}

export function canPlayNativeHls(video?: HTMLVideoElement | null): boolean {

  if (typeof document === "undefined") return false;

  const el = video ?? document.createElement("video");

  return el.canPlayType("application/vnd.apple.mpegurl") !== "";

}

// iOS 17.1+ reports MediaSource support, so hls.js would otherwise steal the
// element from AVPlayer and AirPlay would have no URL to send to the TV.
export function shouldUseNativeHls(video?: HTMLVideoElement | null): boolean {

  return isIOS() && canPlayNativeHls(video);

}

export function supportsAirPlayPicker(video?: HTMLVideoElement | null): boolean {

  if (typeof document === "undefined") return false;

  const el = (video ?? document.createElement("video")) as AirPlayVideoElement;

  return typeof el.webkitShowPlaybackTargetPicker === "function";

}

export function prepareVideoForAirPlay(video: HTMLVideoElement | null): void {

  if (!video) return;

  const el = video as AirPlayVideoElement;

  // Deliberately does not touch crossorigin: attachSource/videoCrossOrigin own
  // that, and clearing it here only to have React re-apply it on the next
  // render reloads the media element and kills whatever is playing. iOS (the
  // AirPlay path) already renders the element without it.
  el.setAttribute("x-webkit-airplay", "allow");
  el.disableRemotePlayback = false;

}

export function isAirPlayActive(video: HTMLVideoElement | null): boolean {

  return !!(video as AirPlayVideoElement | null)?.webkitCurrentPlaybackTargetIsWireless;

}

export function showAirPlayPicker(video: HTMLVideoElement | null): void {

  const el = video as AirPlayVideoElement | null;

  if (!el || typeof el.webkitShowPlaybackTargetPicker !== "function") return;

  prepareVideoForAirPlay(el);
  el.webkitShowPlaybackTargetPicker();

}
