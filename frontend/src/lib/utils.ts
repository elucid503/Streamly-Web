import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {

  return twMerge(clsx(inputs));

}

export function formatDuration(ms: number): string {

  if (!Number.isFinite(ms) || ms <= 0) return "0:00";

  const total = Math.floor(ms / 1000);

  const h = Math.floor(total / 3600), m = Math.floor((total % 3600) / 60), s = total % 60;

  if (h > 0) {

    return `${h}:${String(m).padStart(2, "0")}:${String(s).padStart(2, "0")}`;

  }

  return `${m}:${String(s).padStart(2, "0")}`;

}

export function progressPercent(positionMs?: number, durationMs?: number): number {

  if (!positionMs || !durationMs || durationMs <= 0) return 0;

  return Math.max(0, Math.min(100, (positionMs / durationMs) * 100));

}
