export const AD_BREAK_THRESHOLD_SEC = 5;

const GA94 = new Uint8Array([0x47, 0x41, 0x39, 0x34]);

export class AdBreakDetector {

  private quietSec = 0;
  private inBreak = false;
  private dismissed = false;
  private seenTalking = false;
  private lastSn: number | string | null = null;
  private lastFragAt = 0;
  private lastFrag = "none";

  reset() {

    this.quietSec = 0;
    this.inBreak = false;
    this.dismissed = false;
    this.seenTalking = false;
    this.lastSn = null;
    this.lastFragAt = 0;
    this.lastFrag = "none";

  }

  dismiss() {

    this.dismissed = true;

  }

  overlayActive() {

    return this.inBreak && !this.dismissed;

  }

  debugLabel() {

    return this.inBreak ? "ad" : "program";

  }

  debugLine() {

    const age = this.lastFragAt ? ((Date.now() - this.lastFragAt) / 1000).toFixed(1) : "-";

    return `${this.debugLabel()} quiet=${this.quietSec.toFixed(1)}s overlay=${this.overlayActive()} primed=${this.seenTalking} last=${this.lastFrag} age=${age}s`;

  }

  // Returns whether the commercial overlay should show after this fragment.
  push(payload: ArrayBuffer | Uint8Array, durationSec: number, sn?: number | string) {

    if (sn === "initSegment") {

      return this.overlayActive();

    }

    if (sn != null && sn === this.lastSn) {

      return this.overlayActive();

    }

    const data = payload instanceof Uint8Array ? payload : new Uint8Array(payload);
    const sample = extractCC608(data);

    this.lastFragAt = Date.now();

    const duration = Number.isFinite(durationSec) && durationSec > 0 ? durationSec : 2;

    if (sample.ga94 === 0) {

      this.lastFrag = `no-608 ts=${sample.tsPackets} bytes=${data.byteLength} dur=${duration.toFixed(2)}`;

      return this.overlayActive();

    }

    if (sn != null) {

      this.lastSn = sn;

    }

    // Roll-up is the only reliable "announcers talking" bit. Letter counts
    // fire on 608 stuffing (H@c) and kept SNY ads looking like program.
    const talking = sample.rollup > 0;

    this.lastFrag = `${talking ? "talking" : "quiet"} ru=${sample.rollup} ga94=${sample.ga94} dur=${duration.toFixed(2)}`;

    if (talking) {

      this.seenTalking = true;
      this.quietSec = 0;
      this.inBreak = false;
      this.dismissed = false;

      return false;

    }

    this.quietSec += duration;

    // Mid-join ads have no talking→quiet edge, so skip the 5s hold until primed.
    if (!this.seenTalking || this.quietSec >= AD_BREAK_THRESHOLD_SEC) {

      this.inBreak = true;

    }

    return this.overlayActive();

  }

}

function extractCC608(data: Uint8Array) {

  let ga94 = 0;
  let rollup = 0;
  let letters = 0;
  let tsPackets = 0;
  let i = 0;

  for (let p = 0; p + 188 <= data.length && p < 188 * 8; p += 188) {

    if (data[p] === 0x47) {

      tsPackets++;

    }

  }

  while (i < data.length) {

    const hit = indexOfBytes(data, GA94, i);

    if (hit < 0) {

      break;

    }

    i = hit + 4;
    ga94++;

    if (i >= data.length || data[i] !== 0x03) {

      continue;

    }

    i++;

    if (i >= data.length) {

      break;

    }

    const flags = data[i++];
    const count = flags & 0x1f;

    if ((flags & 0x40) === 0) {

      continue;

    }

    i++;

    for (let n = 0; n < count && i + 2 < data.length; n++) {

      const marker = data[i];
      const b1 = data[i + 1] & 0x7f;
      const b2 = data[i + 2] & 0x7f;

      i += 3;

      const valid = (marker & 0x04) !== 0;
      const ccType = marker & 0x03;

      if (!valid || ccType > 1) {

        continue;

      }

      if (b1 >= 0x10 && b1 <= 0x1f) {

        if ((b1 === 0x14 || b1 === 0x15 || b1 === 0x1c || b1 === 0x1d) &&
          (b2 === 0x25 || b2 === 0x26 || b2 === 0x27)) {

          rollup++;

        }

        continue;

      }

      letters += letterCount(b1) + letterCount(b2);

    }

  }

  return { ga94, rollup, letters, tsPackets };

}

function letterCount(b: number) {

  if (b >= 0x41 && b <= 0x5a) return 1;
  if (b >= 0x61 && b <= 0x7a) return 1;
  if (b >= 0x30 && b <= 0x39) return 1;

  return 0;

}

function indexOfBytes(hay: Uint8Array, needle: Uint8Array, from: number) {

  const last = hay.length - needle.length;

  for (let i = from; i <= last; i++) {

    let ok = true;

    for (let j = 0; j < needle.length; j++) {

      if (hay[i + j] !== needle[j]) {

        ok = false;
        break;

      }

    }

    if (ok) {

      return i;

    }

  }

  return -1;

}
