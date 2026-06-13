import type { IntroInfo } from "@/lib/types";

export function hasIntroWindow(intro?: IntroInfo | null): intro is IntroInfo & {
  introStartMs: number;
  introEndMs: number;
} {
  return (
    intro != null &&
    typeof intro.introStartMs === "number" &&
    typeof intro.introEndMs === "number" &&
    intro.introEndMs > intro.introStartMs
  );
}

export function isInIntroWindow(intro: IntroInfo | null | undefined, currentMs: number): boolean {
  if (!hasIntroWindow(intro)) return false;
  return currentMs >= intro.introStartMs && currentMs < intro.introEndMs;
}