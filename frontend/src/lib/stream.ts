import { isWebPlayableUrl, qualityPlaybackUrl, streamPlaybackUrl } from "@/lib/streamClient";
import type { StreamInfo, StreamQuality } from "@/lib/types";

function qualityPreferenceScore(quality: StreamQuality): number {

  const url = qualityPlaybackUrl(quality) || quality.url;
  if (!url || !isWebPlayableUrl(url)) return -100; // unplayable urls are the least preferred

  let score = 0;
  const path = url.split("?")[0].toLowerCase();

  if (path.endsWith(".mp4") || path.endsWith(".m4v")) score += 30; // prefer direct mp4 links

  if (!quality.isHls) score += 10; // prefer non-hls streams

  return score;

}

const QUALITY_TIERS = [2160, 1440, 1080, 720, 480, 360];

export function uniqueQualityHeights(qualities: StreamQuality[]): number[] {

  return [...new Set(qualities.map((q) => q.height).filter((h) => h > 0))].sort((a, b) => b - a);

}

export function dedupeQualitiesByHeight(qualities: StreamQuality[]): StreamQuality[] {

  const byHeight = new Map<number, StreamQuality>();

  for (const quality of qualities) {

    if (quality.height <= 0) continue;

    const existing = byHeight.get(quality.height);

    if (!existing) {

      byHeight.set(quality.height, quality);
      continue;

    }

    const existingScore = qualityPreferenceScore(existing);
    const nextScore = qualityPreferenceScore(quality);

    if (nextScore > existingScore) {

      byHeight.set(quality.height, quality);

    }

  }

  return [...byHeight.values()].sort((a, b) => b.height - a.height);

}

export function closestAvailableHeight(qualities: StreamQuality[], preferredHeight: number): number | null {

  const heights = uniqueQualityHeights(qualities);

  if (heights.length === 0) return null;

  if (heights.includes(preferredHeight)) return preferredHeight;

  const atOrBelow = heights.filter((height) => height <= preferredHeight);

  if (atOrBelow.length > 0) return atOrBelow[0];

  return heights[heights.length - 1];

}

export function nextLowerQualityHeight( qualities: StreamQuality[], currentHeight: number ): number | null {

  if (currentHeight <= 0) return null;

  const lower = uniqueQualityHeights(qualities).filter((h) => h < currentHeight);

  return lower[0] ?? null;

}

export function initialQualityAttempts(preferredHeight?: number): number[] {

  const preferred = preferredHeight && preferredHeight > 0 ? preferredHeight : 1080;
  const attempts = new Set<number>([preferred]);

  for (const tier of QUALITY_TIERS) {

    if (tier < preferred) attempts.add(tier);

  }

  return [...attempts].sort((a, b) => b - a);

}

export async function fetchStreamWithFallback(heights: number[], fetchStream: (height: number) => Promise<StreamInfo>): Promise<{ stream: StreamInfo; height: number }> {

  let lastError: unknown;

  for (const height of heights) {

    try {

      const stream = await fetchStream(height);

      if (streamPlaybackUrl(stream)) return { stream, height };

    } catch (err) {

      lastError = err;

    }

  }

  if (lastError instanceof Error) throw lastError;

  throw new Error("no stream available");

}
