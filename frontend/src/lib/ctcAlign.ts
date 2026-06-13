export interface AlignedWordTiming {
  index: number;
  start: number;
  end: number;
}

export interface CtcAlignInput {
  logits: Float32Array;
  frameCount: number;
  vocabSize: number;
  blankId: number;
  tokenIds: number[];
  tokenWordIndex: number[];
  windowStart: number;
  windowEnd: number;
}

const NEG_INF = -1e30;

let logProbBuffer: Float32Array | null = null;
let trellisBuffer: Float32Array | null = null;
let fromAdvanceBuffer: Uint8Array | null = null;

const ensureFloatBuffer = (buffer: Float32Array | null, length: number) =>
  buffer && buffer.length >= length ? buffer : new Float32Array(length);

const ensureUintBuffer = (buffer: Uint8Array | null, length: number) =>
  buffer && buffer.length >= length ? buffer : new Uint8Array(length);

const logSoftmaxRows = (logits: Float32Array, frameCount: number, vocabSize: number) => {
  const out = ensureFloatBuffer(logProbBuffer, logits.length);
  logProbBuffer = out;
  for (let frame = 0; frame < frameCount; frame += 1) {
    const offset = frame * vocabSize;
    let max = NEG_INF;
    for (let i = 0; i < vocabSize; i += 1) max = Math.max(max, logits[offset + i]);
    let sum = 0;
    for (let i = 0; i < vocabSize; i += 1) sum += Math.exp(logits[offset + i] - max);
    const logSum = max + Math.log(sum);
    for (let i = 0; i < vocabSize; i += 1) out[offset + i] = logits[offset + i] - logSum;
  }
  return out;
};

// Forced alignment with a free endpoint: the best path may consume only a
// prefix of the transcript, so partially captured audio yields exact timings
// for the words spoken so far instead of squeezing unspoken words into it.
export function alignCtc(input: CtcAlignInput): AlignedWordTiming[] {
  const { frameCount, vocabSize, blankId, tokenIds, tokenWordIndex } = input;
  const tokenCount = tokenIds.length;
  if (frameCount <= 0 || tokenCount === 0 || tokenWordIndex.length !== tokenCount) return [];

  const logProbs = logSoftmaxRows(input.logits, frameCount, vocabSize);
  const emit = (frame: number, id: number) =>
    id >= 0 && id < vocabSize ? logProbs[frame * vocabSize + id] : NEG_INF;

  const width = tokenCount + 1;
  const trellisSize = (frameCount + 1) * width;
  const trellis = ensureFloatBuffer(trellisBuffer, trellisSize);
  trellisBuffer = trellis;
  trellis.fill(NEG_INF, 0, trellisSize);
  // Backpointer per cell; ties prefer advance so word starts snap to the
  // earliest optimal frame instead of lagging the audio.
  const fromAdvance = ensureUintBuffer(fromAdvanceBuffer, frameCount * width);
  fromAdvanceBuffer = fromAdvance;
  fromAdvance.fill(0);
  trellis[0] = 0;

  for (let frame = 0; frame < frameCount; frame += 1) {
    const blank = emit(frame, blankId);
    trellis[(frame + 1) * width] = trellis[frame * width] + blank;
    for (let token = 0; token < tokenCount; token += 1) {
      const tokenEmit = emit(frame, tokenIds[token]);
      const stay = trellis[frame * width + token + 1] + Math.max(blank, tokenEmit);
      const advance = trellis[frame * width + token] + tokenEmit;
      if (advance >= stay) {
        trellis[(frame + 1) * width + token + 1] = advance;
        fromAdvance[frame * width + token + 1] = 1;
      } else {
        trellis[(frame + 1) * width + token + 1] = stay;
      }
    }
  }

  let consumed = 0;
  let best = NEG_INF;
  for (let token = 0; token <= tokenCount; token += 1) {
    const score = trellis[frameCount * width + token];
    if (score > best) {
      best = score;
      consumed = token;
    }
  }
  if (consumed === 0 || best <= NEG_INF / 2) return [];

  const advanceFrame = new Int32Array(consumed).fill(-1);
  let frame = frameCount;
  let token = consumed;
  while (frame > 0 && token > 0) {
    if (fromAdvance[(frame - 1) * width + token]) {
      advanceFrame[token - 1] = frame - 1;
      token -= 1;
    }
    frame -= 1;
  }

  const frameToSeconds = (f: number) =>
    input.windowStart + ((input.windowEnd - input.windowStart) * f) / Math.max(frameCount, 1);

  const words: AlignedWordTiming[] = [];
  let currentWord = -1;
  let startF = 0;
  let endF = 0;
  let valid = false;
  const flush = () => {
    if (currentWord >= 0 && valid) {
      words.push({
        index: currentWord,
        start: frameToSeconds(startF),
        end: frameToSeconds(endF + 1),
      });
    }
  };

  for (let token = 0; token < consumed; token += 1) {
    const wordIndex = tokenWordIndex[token];
    if (wordIndex < 0) continue;
    const f = advanceFrame[token];
    if (wordIndex !== currentWord) {
      flush();
      currentWord = wordIndex;
      startF = f;
      endF = f;
      valid = f >= 0;
    } else if (f >= 0) {
      endF = f;
    } else {
      valid = false;
    }
  }
  // The path may end mid-word; only emit the last word if all its tokens were consumed.
  if (consumed >= tokenCount || tokenWordIndex[consumed] !== currentWord) flush();

  return words;
}
