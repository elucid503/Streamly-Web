import type { AlignedWordTiming } from "@/lib/ctcAlign";
import type { VttCue } from "@/lib/vtt";

export interface SubtitleWord {
  text: string;
  start: number;
  lineBreakBefore: boolean;
  isAnnotation?: boolean;
}

export interface AlignedSubtitleCue extends VttCue {
  words: SubtitleWord[];
  isAnnotation: boolean;
}

const wordWeight = (word: string) => {
  const letters = word.replace(/[^\p{L}\p{N}]/gu, "").length;
  const punctuationPause = /[,.!?;:]$/.test(word) ? 0.2 : 0;
  return Math.max(0.6, Math.sqrt(letters)) + punctuationPause;
};

// Cues entirely wrapped in () or [] are non-speech annotations (music, applause, etc.)
// They should show fully highlighted immediately — no word-level sync needed.
const isAnnotationText = (text: string) => /^\s*[\(\[][\s\S]*[\)\]]\s*$/.test(text.trim());

const ANNOTATION_UNIT_WEIGHT = 0.8;

type ParsedToken = {
  text: string;
  lineBreakBefore: boolean;
  isAnnotation: boolean;
};

type PaceUnit = {
  indices: number[];
  weight: number;
};

const splitAnnotationSegments = (line: string) => {
  const segments: { text: string; isAnnotation: boolean }[] = [];
  const pattern = /(\([^)]*\)|\[[^\]]*\])/g;
  let lastIndex = 0;
  let match: RegExpExecArray | null;

  while ((match = pattern.exec(line)) !== null) {
    if (match.index > lastIndex) {
      segments.push({ text: line.slice(lastIndex, match.index), isAnnotation: false });
    }
    segments.push({ text: match[1], isAnnotation: true });
    lastIndex = match.index + match[0].length;
  }

  if (lastIndex < line.length) {
    segments.push({ text: line.slice(lastIndex), isAnnotation: false });
  }

  return segments;
};

const parseCueTokens = (text: string): ParsedToken[] => {
  const tokens: ParsedToken[] = [];

  text.split("\n").forEach((line, lineIndex) => {
    const segments = splitAnnotationSegments(line);
    segments.forEach((segment, segmentIndex) => {
      segment.text.match(/\S+/g)?.forEach((word, wordIndex) => {
        tokens.push({
          text: word,
          lineBreakBefore: lineIndex > 0 && segmentIndex === 0 && wordIndex === 0,
          isAnnotation: segment.isAnnotation,
        });
      });
    });
  });

  return tokens;
};

const buildPaceUnits = (words: SubtitleWord[]): PaceUnit[] => {
  const units: PaceUnit[] = [];
  let index = 0;

  while (index < words.length) {
    if (words[index].isAnnotation) {
      const indices = [index];
      index += 1;
      while (index < words.length && words[index].isAnnotation) {
        indices.push(index);
        index += 1;
      }
      units.push({ indices, weight: ANNOTATION_UNIT_WEIGHT });
    } else {
      units.push({ indices: [index], weight: wordWeight(words[index].text) });
      index += 1;
    }
  }

  return units;
};

const assignUnitStarts = (
  words: SubtitleWord[],
  units: PaceUnit[],
  fromUnit: number,
  spanStart: number,
  spanEnd: number,
) => {
  const selected = units.slice(fromUnit);
  if (selected.length === 0) return;

  const totalWeight = selected.reduce((sum, unit) => sum + unit.weight, 0);
  const span = Math.max(spanEnd - spanStart, 0.05 * selected.length);
  let elapsedWeight = 0;

  selected.forEach((unit) => {
    const start = spanStart + (span * elapsedWeight) / totalWeight;
    elapsedWeight += unit.weight;
    for (const wordIndex of unit.indices) {
      words[wordIndex].start = start;
    }
  });
};

const anchorAnnotationSpans = (words: SubtitleWord[]) => {
  let index = 0;
  while (index < words.length) {
    if (!words[index].isAnnotation) {
      index += 1;
      continue;
    }

    const runStart = index;
    while (index < words.length && words[index].isAnnotation) index += 1;

    let start = words[runStart].start;
    if (runStart > 0) {
      start = Math.max(start, words[runStart - 1].start);
    }
    for (let wordIndex = runStart; wordIndex < index; wordIndex += 1) {
      words[wordIndex].start = start;
    }
  }
};

export function alignCue(cue: VttCue): AlignedSubtitleCue {
  const tokens = parseCueTokens(cue.text);
  const annotation = isAnnotationText(cue.text);

  if (tokens.length === 0) return { ...cue, words: [], isAnnotation: annotation };

  // Annotations: all words start at cue.start so they light up immediately.
  if (annotation) {
    return {
      ...cue,
      isAnnotation: true,
      words: tokens.map((token) => ({ ...token, start: cue.start })),
    };
  }

  const words: SubtitleWord[] = tokens.map((token) => ({ ...token, start: cue.start }));
  const units = buildPaceUnits(words);
  assignUnitStarts(words, units, 0, cue.start, cue.end);
  anchorAnnotationSpans(words);

  return { ...cue, words, isAnnotation: false };
}

export function activeWordIndex(cue: AlignedSubtitleCue, time: number) {
  let index = -1;
  while (index + 1 < cue.words.length && time >= cue.words[index + 1].start) index += 1;

  // Parenthetical / bracketed spans highlight as one block once their slot is reached.
  if (index >= 0 && cue.words[index]?.isAnnotation) {
    let runEnd = index;
    while (runEnd + 1 < cue.words.length && cue.words[runEnd + 1].isAnnotation) {
      runEnd += 1;
    }
    index = runEnd;
  }

  return index;
}

// Overlays model timings onto estimated words and re-paces everything the
// model has not (yet) heard into the remaining cue time. A timing bias shifts
// word highlights earlier to account for model alignment latency.
//
// currentTime: the playhead at the moment of the merge. Used to prevent two
// visual glitches:
//   - Un-highlighting: word starts only ever decrease, so a later model pass
//     can never push a start forward and un-highlight a word.
//   - Batch lighting: if the model would retroactively set a not-yet-lit word
//     into the past, we leave it at its existing start so it lights up
//     naturally as the playhead reaches it, instead of all at once.
// Pass Infinity (the default) to skip both constraints, e.g. for the final
// cue-exit pass that pre-bakes timings for replay.
const TIMING_BIAS_SECONDS = 0.05;

export function mergeModelTimings(
  cue: AlignedSubtitleCue,
  timings: AlignedWordTiming[],
  currentTime = Infinity,
): AlignedSubtitleCue {
  if (timings.length === 0) return cue;
  const words = cue.words.map((word) => ({ ...word }));

  for (const timing of timings) {
    const word = words[timing.index];
    if (word && !word.isAnnotation) {
      const biased = timing.start - TIMING_BIAS_SECONDS;
      const clamped = Math.min(Math.max(biased, cue.start), cue.end);
      if (clamped < word.start) {
        // Model wants to move start earlier — apply unless doing so would
        // retroactively light up a not-yet-lit word (causes batch lighting).
        const retroactive = clamped < currentTime && word.start >= currentTime;
        if (!retroactive) word.start = clamped;
      }
      // If clamped >= word.start the model wants to move the start later;
      // keep the existing earlier start so words never un-highlight.
    }
  }

  const last = timings[timings.length - 1];
  const units = buildPaceUnits(words);
  const lastUnitIndex = units.findIndex((unit) => unit.indices.includes(last.index));
  if (lastUnitIndex >= 0 && lastUnitIndex < units.length - 1) {
    const from = Math.min(Math.max(last.end, cue.start), cue.end);
    assignUnitStarts(words, units, lastUnitIndex + 1, from, cue.end);
  }

  anchorAnnotationSpans(words);

  // Model starts are monotonic; pull estimated neighbours into order around them.
  for (let i = words.length - 2; i >= 0; i -= 1) {
    words[i].start = Math.min(words[i].start, words[i + 1].start);
  }

  anchorAnnotationSpans(words);

  return { ...cue, words };
}
