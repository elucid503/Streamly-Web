import type { StreamInfo, StreamQuality } from "@/Types";

import { forceIosMediaProxy } from "@/Utils/Player/AirPlay";

export function isWebPlayableUrl(url: string): boolean {

  const path = url.split("?")[0]?.toLowerCase() ?? "";

  return (!path.endsWith(".mkv") && !path.endsWith(".avi") && !path.endsWith(".wmv") && !path.endsWith(".flv")); // not generally supported by browsers

}

function preferPlaybackUrl(direct: string, proxy: string, isHls: boolean): string {

  if (forceIosMediaProxy() && proxy) return proxy;

  if (direct && !isHls && isWebPlayableUrl(direct)) return direct;

  return proxy || direct;

}

export function streamPlaybackUrl(stream: { url?: string; proxyUrl?: string; isHls?: boolean }): string {

  return preferPlaybackUrl(stream.url?.trim() || "", stream.proxyUrl?.trim() || "", !!stream.isHls);

}

export function isProxiedStream(url: string): boolean {

  return url.includes("/api/proxy/");

}

export function qualityPlaybackUrl(quality: StreamQuality): string {

  const url = preferPlaybackUrl(quality.url?.trim() || "", quality.proxyUrl?.trim() || "", quality.isHls);

  if (!url || !isWebPlayableUrl(url)) return "";

  return url;

}

export function pickQualityByHeight(qualities: StreamQuality[], height: number): StreamQuality | null {

  if (height <= 0) return null;

  return qualities.find((quality) => quality.height === height && quality.url) ?? null;

}

export function streamFromQuality(qualities: StreamQuality[], quality: StreamQuality, selectedHeight?: number): StreamInfo {

  return {

    qualities,
    selectedHeight: selectedHeight ?? quality.height,

    url: quality.url,
    proxyUrl: quality.proxyUrl,

    isHls: quality.isHls,

  };

}
