export const AD_BREAK_THRESHOLD_SEC = 5;
export const AD_BREAK_POPON_EXIT_SEC = 30;

const MIN_POPON_COMMANDS = 3;
const MIN_TALKING_LETTERS = 8;

const GA94 = new Uint8Array([0x47, 0x41, 0x39, 0x34]); // GA94 is the "CC data" packet identifier in MPEG-TS streams.

export class AdBreakDetector {

  private quietSec = 0;
  private popOnSec = 0;

  private inBreak = false;

  private dismissed = false;
  private seenTalking = false;
  private popOnLocked = false;

  private lastSn: number | string | null = null;

  private lastFragAt = 0;
  private lastFrag = "none";

  reset() {

    this.quietSec = 0;
    this.popOnSec = 0;

    this.inBreak = false;

    this.dismissed = false;
    this.seenTalking = false;
    this.popOnLocked = false;

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

    return `${this.debugLabel()} quiet=${this.quietSec.toFixed(1)}s hold=${this.popOnSec.toFixed(1)}s overlay=${this.overlayActive()} primed=${this.seenTalking} last=${this.lastFrag} age=${age}s`;

  }

  // Returns whether the commercial overlay should show after this fragment.
  push(payload: ArrayBuffer | Uint8Array, durationSec: number, sn?: number | string) {

    if (sn === "initSegment") {

      return this.overlayActive(); // no heuristic for init segments, so we just keep the overlay state as-is

    }

    if (sn != null && sn === this.lastSn) {

      return this.overlayActive(); // duplicate or invalid fragment, so we ignore

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

    // Roll-up keepalives (RU2/RU3 with no letters) ride through ads. Real program has roll-up plus CC1 text.
    const talking = sample.rollup > 0 && sample.letters >= MIN_TALKING_LETTERS && !isRepeatedCaptionJunk(sample.cc1Text);
    const popOnProgram = sample.popon + sample.painton >= MIN_POPON_COMMANDS;

    const watermark = isRepeatedCaptionJunk(sample.cc2Text);
    const stats = `ru=${sample.rollup} pop=${sample.popon} let=${sample.letters} ga94=${sample.ga94} dur=${duration.toFixed(2)}`;

    if (talking) {

      this.seenTalking = true;
      this.popOnLocked = false;

      this.quietSec = 0;
      this.popOnSec = 0;

      this.inBreak = false;
      this.dismissed = false;

      this.lastFrag = `talking ${stats}`;

      return false;

    }

    if (sample.rollup > 0) {

      this.lastFrag = `ru-hold ${stats}`;

    }

    // Scripted channels caption with pop-on; so do many ads. One pop-on fragment must not drop an in-progress break, so we require a long stretch.
    if (popOnProgram && this.inBreak) {

      this.popOnSec += duration;
      this.lastFrag = `pop-on-hold ${stats}`;

      if (this.popOnSec >= AD_BREAK_POPON_EXIT_SEC) {

        this.quietSec = 0;
        this.popOnSec = 0;
        this.popOnLocked = true;

        this.inBreak = false;

        this.lastFrag = `pop-on-exit ${stats}`;

        return false;

      }

      return this.overlayActive();

    }

    // After pop-on-exit, scripted channels stay in program. News ads still enter on the first unprimed pop-on stretch.
    if (!this.seenTalking && popOnProgram && this.popOnLocked) {

      this.quietSec = 0;
      this.popOnSec = 0;
      this.lastFrag = `pop-on ${stats}`;

      return false;

    }

    // Cable watermarks (HDHDHD, "FX Movie") sit on CC2 and are not silence.
    if (!this.seenTalking && watermark) {

      this.popOnSec = 0;
      this.lastFrag = `watermark ${stats}`;

      return this.overlayActive();

    }

    this.popOnSec = 0;
    this.quietSec += duration;

    if (!sample.rollup) {

      this.lastFrag = `quiet ${stats}`;

    }

    // Mid-join ads have no talking→quiet edge, so skip the 5s hold until primed.
    if (!this.seenTalking || this.quietSec >= AD_BREAK_THRESHOLD_SEC) {

      this.inBreak = true;

    }

    return this.overlayActive();

  }

}

// Extracts CC608 caption data from a MPEG-TS stream.
function extractCC608(data: Uint8Array) {

  let ga94 = 0;

  let rollup = 0;
  let popon = 0;

  let painton = 0;

  let letters = 0;
  let tsPackets = 0;

  let cc1Text = "";
  let cc2Text = "";

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

        if (ccType === 0 && (b1 === 0x14 || b1 === 0x15)) {

          if (b2 === 0x25 || b2 === 0x26 || b2 === 0x27) {

            rollup++;

          } else if (b2 === 0x20 || b2 === 0x2f) {

            popon++;

          } else if (b2 === 0x29) {

            painton++;

          }

        }

        continue;

      }

      if (b1 === 0 && b2 === 0) {

        continue;

      }

      const chunk = printable608(b1) + printable608(b2);

      if (ccType === 0) {

        letters += letterCount(b1) + letterCount(b2);
        cc1Text += chunk;

      } else {

        cc2Text += chunk;

      }

    }

  }

  return { ga94, rollup, popon, painton, letters, tsPackets, cc1Text, cc2Text };

}

function isRepeatedCaptionJunk(text: string) {

  const s = text.replace(/\s+/g, " ").trim();

  if (s.length < 8) {

    return false;

  }

  const grams = new Map<string, number>();
  let bestBigram = 0;

  for (let i = 0; i < s.length - 1; i++) {

    const gram = s.slice(i, i + 2);
    const n = (grams.get(gram) ?? 0) + 1;
    grams.set(gram, n);

    if (n > bestBigram) {

      bestBigram = n;

    }

  }

  if (bestBigram * 2 >= s.length * 0.7) {

    return true;

  }

  for (let len = 16; len >= 4; len--) {

    if (s.length < len * 3) {

      continue;

    }

    const seen = new Set<string>();

    for (let i = 0; i + len * 3 <= s.length; i++) {

      const phrase = s.slice(i, i + len);

      if (phrase.trim().length < 4 || seen.has(phrase)) {

        continue;

      }

      seen.add(phrase);

      let hits = 0;
      let covered = 0;

      for (let j = 0; j + len <= s.length; ) {

        if (s.slice(j, j + len) === phrase) {

          hits++;
          covered += len;
          j += len;
          continue;

        }

        j++;

      }

      if (hits >= 3 && covered >= s.length * 0.45) {

        return true;

      }

    }

  }

  return false;

}

function printable608(b: number) {

  if (b < 0x20 || b === 0x7f) {

    return "";

  }

  if (b === 0x2a) {

    return "'";

  }

  return String.fromCharCode(b);

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
