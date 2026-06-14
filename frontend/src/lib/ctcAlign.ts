export interface AlignedWordTiming {

  index: number;
  start: number;
  end: number;

  // Mean log-probability of character frames for this word — used to filter
  // uncertain alignments before merging into subtitle timings.
  confidence: number;

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

// Scan backward from the CTC advance frame to find where the character's
// log-prob first crossed this threshold — gives a tighter acoustic onset.
const ONSET_THRESHOLD = Math.log(0.15);
const MAX_ONSET_SCAN = 8; // frames (~160ms at 50 fps)

let logProbBuffer: Float32Array | null = null;
let trellisBuffer: Float32Array | null = null;
let fromAdvanceBuffer: Uint8Array | null = null;

const ensureFloatBuffer = (buffer: Float32Array | null, length: number) => buffer && buffer.length >= length ? buffer : new Float32Array(length);

const ensureUintBuffer = (buffer: Uint8Array | null, length: number) => buffer && buffer.length >= length ? buffer : new Uint8Array(length);

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

// Forced alignment with a free endpoint: the best path may consume only a prefix of the transcript
export function alignCtc(input: CtcAlignInput): AlignedWordTiming[] {

  const { frameCount, vocabSize, blankId, tokenIds, tokenWordIndex } = input;
  const tokenCount = tokenIds.length;

  if (frameCount <= 0 || tokenCount === 0 || tokenWordIndex.length !== tokenCount) return [];

  const logProbs = logSoftmaxRows(input.logits, frameCount, vocabSize);
  const emit = (frame: number, id: number) => id >= 0 && id < vocabSize ? logProbs[frame * vocabSize + id] : NEG_INF;

  const width = tokenCount + 1;
  const trellisSize = (frameCount + 1) * width;
  const trellis = ensureFloatBuffer(trellisBuffer, trellisSize);

  trellisBuffer = trellis;
  trellis.fill(NEG_INF, 0, trellisSize);

  const fromAdvance = ensureUintBuffer(fromAdvanceBuffer, frameCount * width); // Backpointer per cell

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

  const frameToSeconds = (f: number) => input.windowStart + ((input.windowEnd - input.windowStart) * f) / Math.max(frameCount, 1);

  const words: AlignedWordTiming[] = [];
  let currentWord = -1;

  let startF = 0;
  let endF = 0;
  let startTokenId = -1;

  let wordLogprobSum = 0;
  let wordLogprobCount = 0;

  let valid = false;

  const flush = () => {

    if (currentWord >= 0 && valid) {

      // Scan backward from the advance frame to find the acoustic onset —
      // the earliest frame where the character's probability was already
      // above ONSET_THRESHOLD, catching cases where the model held blank
      // while the phoneme had already started.
      let onsetF = startF;

      if (startTokenId >= 0 && startF > 0) {

        const scanLimit = Math.max(0, startF - MAX_ONSET_SCAN);

        for (let f = startF - 1; f >= scanLimit; f -= 1) {

          if (logProbs[f * vocabSize + startTokenId] < ONSET_THRESHOLD) {

            onsetF = f + 1;
            break;

          }

          onsetF = f;

        }

      }

      words.push({

        index: currentWord,
        start: frameToSeconds(onsetF),
        end: frameToSeconds(endF + 1),
        confidence: wordLogprobCount > 0 ? wordLogprobSum / wordLogprobCount : NEG_INF,

      });

    }

  };

  for (let token = 0; token < consumed; token += 1) {

    const wordIndex = tokenWordIndex[token];

    if (wordIndex < 0) continue;

    const f = advanceFrame[token];
    const tokenId = tokenIds[token];

    if (wordIndex !== currentWord) {

      flush();

      currentWord = wordIndex;

      startF = f;
      endF = f;
      startTokenId = tokenId;

      valid = f >= 0;

      wordLogprobSum = valid ? logProbs[f * vocabSize + tokenId] : 0;
      wordLogprobCount = valid ? 1 : 0;

    } else if (f >= 0) {

      endF = f;
      wordLogprobSum += logProbs[f * vocabSize + tokenId];
      wordLogprobCount += 1;

    } else {

      valid = false;

    }

  }

  // The path may end mid-word; only emit the last word if all its tokens were consumed.
  if (consumed >= tokenCount || tokenWordIndex[consumed] !== currentWord) flush();

  return words;

}
