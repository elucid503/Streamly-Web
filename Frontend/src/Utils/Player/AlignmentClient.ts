import type { AlignedWordTiming } from "@/Utils/Player/CtcAlign";
import { isIOS, isTV, supportsOnnxWasm } from "@/Utils/Platform";

interface PendingRequest {

  resolve: (words: AlignedWordTiming[]) => void;
  reject: (error: Error) => void;

}

const hasBrowserApis = () => typeof window !== "undefined" && typeof navigator !== "undefined";

const hasAudioContext = () => hasBrowserApis() && (typeof AudioContext !== "undefined" || typeof window.webkitAudioContext !== "undefined");

let unsupportedReason: string | null | undefined;

const markUnsupported = (reason: string) => {

  unsupportedReason = reason;

};

const detectUnsupportedReason = () => {

  if (!hasBrowserApis()) return "browser lacks alignment runtime";

  if (isTV()) return "tv wasm runtime";

  if (isIOS()) return "iOS audio routing";

  if (!supportsOnnxWasm()) return "browser lacks alignment runtime";

  if (!hasAudioContext()) return "AudioContext not supported";

  return null;

};

export const alignmentUnsupportedReason = () => {

  if (unsupportedReason === undefined) unsupportedReason = detectUnsupportedReason();

  return unsupportedReason;

};

export const isAlignmentSupported = () => alignmentUnsupportedReason() === null;

let worker: Worker | null = null;
let workerPromise: Promise<Worker | null> | null = null;

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

const attachWorker = (instance: Worker) => {

  instance.onmessage = (event: MessageEvent) => {

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

  worker = instance;

  return instance;

};

const ensureWorker = () => {

  if (!isAlignmentSupported()) return Promise.resolve(null);

  if (worker) return Promise.resolve(worker);

  if (!workerPromise) {

    workerPromise = import("./createAlignmentWorker")
      .then((mod) => attachWorker(mod.createAlignmentWorker()))
      .catch(() => {

        workerPromise = null;
        markUnsupported("module workers not supported");

        return null;

      });

  }

  return workerPromise;

};

export function warmupAligner() {

  void ensureWorker().then((instance) => instance?.postMessage({ type: "warmup" }));

}

export function alignWords(input: { audio: Float32Array; words: string[]; start: number; end: number; }) {

  return ensureWorker().then((workerInstance) => {

    if (!workerInstance) {

      return Promise.reject(new Error(alignmentUnsupportedReason() ?? "alignment not supported"));

    }

    const id = ++requestId;

    return new Promise<AlignedWordTiming[]>((resolve, reject) => {

      pending.set(id, { resolve, reject });

      workerInstance.postMessage({ id, type: "align", ...input }, [input.audio.buffer]);

    });

  });

}
