export function createAlignmentWorker(): Worker {

  return new Worker(new URL("../../Workers/alignment.worker.ts", import.meta.url), {

    type: "module",

  });

}
