import type { AlignedWordTiming } from "@/lib/ctcAlign";

interface PendingRequest {
  resolve: (words: AlignedWordTiming[]) => void;
  reject: (error: Error) => void;
}

let worker: Worker | null = null;
let requestId = 0;
const pending = new Map<number, PendingRequest>();

// Tracks whether the model has finished loading at least once.
let modelReady = false;
const readyCallbacks: Array<() => void> = [];

export const isModelReady = () => modelReady;

const notifyReady = () => {
  modelReady = true;
  readyCallbacks.splice(0).forEach((cb) => cb());
};

const getWorker = () => {
  if (worker) return worker;
  worker = new Worker(new URL("../workers/alignment.worker.ts", import.meta.url), {
    type: "module",
  });
  worker.onmessage = (event: MessageEvent) => {
    const { id, type, words, error } = event.data ?? {};
    if (type === "ready") {
      notifyReady();
      return;
    }
    const request = pending.get(id);
    if (!request) return;
    pending.delete(id);
    if (type === "result") request.resolve(words ?? []);
    else request.reject(new Error(error || "alignment failed"));
  };
  return worker;
};

export function warmupAligner() {
  getWorker().postMessage({ type: "warmup" });
}

export function alignWords(input: {
  audio: Float32Array;
  words: string[];
  start: number;
  end: number;
}) {
  const id = ++requestId;
  return new Promise<AlignedWordTiming[]>((resolve, reject) => {
    pending.set(id, { resolve, reject });
    getWorker().postMessage({ id, type: "align", ...input }, [input.audio.buffer]);
  });
}
