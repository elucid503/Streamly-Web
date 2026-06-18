import type { AlignedWordTiming } from "@/lib/ctcAlign";

interface PendingRequest {

  resolve: (words: AlignedWordTiming[]) => void;
  reject: (error: Error) => void;

}

const hasBrowserApis = () => typeof window !== "undefined" && typeof navigator !== "undefined";

const isIos = () => {

  if (!hasBrowserApis()) return false;

  const ua = navigator.userAgent;

  if (/iPhone|iPad|iPod/i.test(ua)) return true;

  // iPadOS 13+ reports as MacIntel in desktop mode.
  return navigator.platform === "MacIntel" && navigator.maxTouchPoints > 1;

};

const hasAudioContext = () => hasBrowserApis() && (typeof AudioContext !== "undefined" || typeof window.webkitAudioContext !== "undefined");

const hasModelRuntime = () => hasBrowserApis() && typeof Worker !== "undefined" && typeof WebAssembly !== "undefined" && typeof fetch !== "undefined";

let unsupportedReason: string | null | undefined;

const markUnsupported = (reason: string) => {

  unsupportedReason = reason;

};

const detectUnsupportedReason = () => {

  if (!hasBrowserApis()) return "browser lacks alignment runtime";

  if (isIos()) return "iOS audio routing";

  if (!hasModelRuntime()) return "browser lacks alignment runtime";

  if (!hasAudioContext()) return "AudioContext not supported";

  return null;

};

export const alignmentUnsupportedReason = () => {

  if (unsupportedReason === undefined) unsupportedReason = detectUnsupportedReason();

  return unsupportedReason;

};

export const isAlignmentSupported = () => alignmentUnsupportedReason() === null;

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

  if (!isAlignmentSupported()) return null;

  if (worker) return worker;

  try {

    worker = new Worker(new URL("../workers/alignment.worker.ts", import.meta.url), {

      type: "module",

    });

  } catch {

    markUnsupported("module workers not supported");

    return null;

  }

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

  getWorker()?.postMessage({ type: "warmup" });

}

export function alignWords(input: { audio: Float32Array; words: string[]; start: number; end: number; }) {

  const workerInstance = getWorker();

  if (!workerInstance) {

    return Promise.reject(new Error(alignmentUnsupportedReason() ?? "alignment not supported"));

  }

  const id = ++requestId;

  return new Promise<AlignedWordTiming[]>((resolve, reject) => {

    pending.set(id, { resolve, reject });

    workerInstance.postMessage({ id, type: "align", ...input }, [input.audio.buffer]);

  });

}
