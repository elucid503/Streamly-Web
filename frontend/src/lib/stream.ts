import type { StreamInfo, StreamQuality } from "@/lib/types";

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
    if (existing.isHls && !quality.isHls) {
      byHeight.set(quality.height, quality);
    }
  }
  return [...byHeight.values()].sort((a, b) => b.height - a.height);
}

export function nextLowerQualityHeight(
  qualities: StreamQuality[],
  currentHeight: number,
): number | null {
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

export async function fetchStreamWithFallback(
  heights: number[],
  fetchStream: (height: number) => Promise<StreamInfo>,
): Promise<{ stream: StreamInfo; height: number }> {
  let lastError: unknown;
  for (const height of heights) {
    try {
      const stream = await fetchStream(height);
      if (stream?.proxyUrl) return { stream, height };
    } catch (err) {
      lastError = err;
    }
  }
  if (lastError instanceof Error) throw lastError;
  throw new Error("no stream available");
}