const PALETTE = [

  "#ef4444", "#f97316", "#f59e0b", "#84cc16",
  "#10b981", "#14b8a6", "#06b6d4", "#3b82f6",
  "#6366f1", "#8b5cf6", "#d946ef", "#ec4899",

];

export function channelColor(name: string): string {

  let hash = 0;

  for (let i = 0; i < name.length; i++) {

    hash = (hash * 31 + name.charCodeAt(i)) | 0;

  }

  return PALETTE[Math.abs(hash) % PALETTE.length]!;

}

export function channelInitial(name: string): string {

  const trimmed = name.trim();

  return trimmed ? trimmed[0]!.toUpperCase() : "?";

}
