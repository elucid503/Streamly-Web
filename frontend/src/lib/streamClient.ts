import type { StreamInfo, StreamQuality } from "@/lib/types";

export function isWebPlayableUrl(url: string): boolean {

  const path = url.split("?")[0]?.toLowerCase() ?? "";

  return (
    !path.endsWith(".mkv") &&
    !path.endsWith(".avi") &&
    !path.endsWith(".wmv") &&
    !path.endsWith(".flv")
  );

}

export function streamPlaybackUrl(stream: { url?: string; proxyUrl?: string }): string {

  return stream.proxyUrl?.trim() || stream.url?.trim() || "";

}

export function isProxiedStream(url: string): boolean {

  return url.includes("/api/proxy/");

}

export function qualityPlaybackUrl(quality: StreamQuality): string {

  const url = quality.proxyUrl?.trim() || quality.url?.trim() || "";

  if (!url || !isWebPlayableUrl(url)) return "";

  return url;

}

export function pickQualityByHeight(
  qualities: StreamQuality[],
  height: number
): StreamQuality | null {

  if (height <= 0) return null;

  return qualities.find((quality) => quality.height === height && quality.url) ?? null;

}

export function qualityHasProxy(quality: StreamQuality): boolean {

  return !!quality.proxyUrl?.trim();

}

export function streamFromQuality(
  qualities: StreamQuality[],
  quality: StreamQuality,
  selectedHeight?: number
): StreamInfo {

  return {

    qualities,
    url: quality.url,
    proxyUrl: quality.proxyUrl,
    isHls: quality.isHls,

    selectedHeight: selectedHeight ?? quality.height,

  };

}