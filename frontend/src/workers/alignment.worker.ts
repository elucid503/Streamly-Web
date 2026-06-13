import { AutoModelForCTC, AutoProcessor, AutoTokenizer, env } from "@huggingface/transformers";
import { alignCtc, type AlignedWordTiming } from "@/lib/ctcAlign";

interface AlignRequest {
  id: number;
  type: "align";
  audio: Float32Array;
  words: string[];
  start: number;
  end: number;
}

// Lightweight base model (~95 MB q8) — fast enough for real-time subtitle sync.
const MODEL_ID = "Xenova/wav2vec2-base-960h";

env.allowLocalModels = false;

interface LoadedModel {
  processor: any;
  vocab: Map<string, number>;
  uppercase: boolean;
  blankId: number;
  separatorId: number | undefined;
  model: any;
}

let loading: Promise<LoadedModel> | null = null;

const hasWebGpu = () => typeof (self.navigator as any)?.gpu !== "undefined";

const load = () =>
  (loading ??= (async () => {
    const [processor, tokenizer] = await Promise.all([
      AutoProcessor.from_pretrained(MODEL_ID),
      AutoTokenizer.from_pretrained(MODEL_ID),
    ]);

    let model: any;
    if (hasWebGpu()) {
      try {
        model = await AutoModelForCTC.from_pretrained(MODEL_ID, { device: "webgpu", dtype: "q8" });
      } catch {
        model = await AutoModelForCTC.from_pretrained(MODEL_ID, { dtype: "q8" });
      }
    } else {
      model = await AutoModelForCTC.from_pretrained(MODEL_ID, { dtype: "q8" });
    }

    const vocab = tokenizer.get_vocab() as Map<string, number>;
    const uppercase = vocab.has("A") && !vocab.has("a");
    const blankId = tokenizer.pad_token_id ?? vocab.get("<pad>") ?? 0;

    self.postMessage({ type: "ready" });
    return { processor, vocab, uppercase, blankId, separatorId: vocab.get("|"), model };
  })());

const normalizeWord = (word: string, uppercase: boolean) =>
  word
    .normalize("NFKD")
    .replace(/[\u0300-\u036f]/g, "")
    [uppercase ? "toUpperCase" : "toLowerCase"]();

// Tokens carry the index of the display word they came from, so alignment
// results map straight back to rendered words without any text matching.
const tokenize = (
  words: string[],
  vocab: Map<string, number>,
  uppercase: boolean,
  separatorId: number | undefined,
) => {
  const tokenIds: number[] = [];
  const tokenWordIndex: number[] = [];

  for (let wordIndex = 0; wordIndex < words.length; wordIndex += 1) {
    const normalized = normalizeWord(words[wordIndex], uppercase);
    let charCount = 0;
    for (const char of normalized) {
      if (vocab.has(char)) charCount += 1;
    }
    if (charCount === 0) continue;

    if (tokenIds.length > 0 && separatorId !== undefined) {
      tokenIds.push(separatorId);
      tokenWordIndex.push(-1);
    }
    for (const char of normalized) {
      const id = vocab.get(char);
      if (id !== undefined) {
        tokenIds.push(id);
        tokenWordIndex.push(wordIndex);
      }
    }
  }

  return { tokenIds, tokenWordIndex };
};

self.onmessage = async (event: MessageEvent<AlignRequest | { type: "warmup" }>) => {
  const message = event.data;
  if (message.type === "warmup") {
    void load().catch(() => undefined);
    return;
  }
  if (message.type !== "align") return;

  try {
    const { processor, vocab, uppercase, blankId, separatorId, model } = await load();
    const { tokenIds, tokenWordIndex } = tokenize(message.words, vocab, uppercase, separatorId);

    let words: AlignedWordTiming[] = [];
    if (tokenIds.length > 0) {
      const inputs = await processor(message.audio);
      const { logits } = await model(inputs);
      const dims = logits.dims as number[];
      words = alignCtc({
        logits: logits.data as Float32Array,
        frameCount: dims[dims.length - 2],
        vocabSize: dims[dims.length - 1],
        blankId,
        tokenIds,
        tokenWordIndex,
        windowStart: message.start,
        windowEnd: message.end,
      });
      logits.dispose?.();
    }

    self.postMessage({ id: message.id, type: "result", words });
  } catch (error) {
    self.postMessage({
      id: message.id,
      type: "error",
      error: error instanceof Error ? error.message : "alignment failed",
    });
  }
};
