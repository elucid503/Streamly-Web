import type { StreamInfo, StreamQuality } from "@/lib/types";

export function isWebPlayableUrl(url: string): boolean {

  const path = url.split("?")[0]?.toLowerCase() ?? "";

  return (!path.endsWith(".mkv") && !path.endsWith(".avi") && !path.endsWith(".wmv") && !path.endsWith(".flv")); // not generally supported by browsers

}

export function streamPlaybackUrl(stream: { url?: string; proxyUrl?: string; isHls?: boolean }): string {

  const direct = stream.url?.trim() || "";

  if (direct && !stream.isHls && isWebPlayableUrl(direct)) {

    return direct;

  }

  return stream.proxyUrl?.trim() || direct;

}

export function isProxiedStream(url: string): boolean {

  return url.includes("/api/proxy/");

}

export function qualityPlaybackUrl(quality: StreamQuality): string {

  const direct = quality.url?.trim() || "";

  const url = direct && !quality.isHls && isWebPlayableUrl(direct) ? direct : quality.proxyUrl?.trim() || direct;

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
