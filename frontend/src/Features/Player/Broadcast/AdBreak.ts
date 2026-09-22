// Fragments arrive ~10–15s ahead of the live playhead; confirm on the buffer, show on the playhead.
export const SILENCE_TO_START_BREAK_SEC = 15;
export const SCRIPTED_CAPTIONS_TO_END_BREAK_SEC = 30;

const MIN_SCRIPTED_COMMANDS = 3;
const MIN_LIVE_CAPTION_LETTERS = 8;
const TIMELINE_KEEP_SEC = 180;

const GA94 = new Uint8Array([0x47, 0x41, 0x39, 0x34]);

interface Span {

  start: number;
  end: number;
  commercial: boolean;

}

export class AdBreakDetector {

  private silenceSec = 0;
  private scriptedSec = 0;

  private inCommercial = false;
  private dismissed = false;

  private seenLiveShow = false;
  private inScriptedShow = false;

  private lastSn: number | string | null = null;

  private lastNote = "none";

  private timeline: Span[] = [];
  private lastMediaEnd = 0;

  reset() {

    this.silenceSec = 0;
    this.scriptedSec = 0;

    this.inCommercial = false;
    this.dismissed = false;

    this.seenLiveShow = false;
    this.inScriptedShow = false;

    this.lastSn = null;

    this.lastNote = "none";

    this.timeline = [];
    this.lastMediaEnd = 0;

  }

  dismiss() {

    this.dismissed = true;

  }

  visibleAt(playheadSec: number) {

    const idx = this.spanIndexAt(playheadSec);

    if (idx < 0) {

      return false;

    }

    if (!this.timeline[idx].commercial) {

      this.dismissed = false;

      return false;

    }

    return !this.dismissed;

  }

  debugLine(playheadSec?: number) {

    const overlay = playheadSec != null && this.visibleAt(playheadSec);
    const ahead = this.inCommercial ? "commercial" : "show";

    return `${overlay ? "overlay on" : "overlay off"}; ${ahead}; ${this.lastNote}`;

  }

  addFragment(payload: ArrayBuffer | Uint8Array, durationSec: number, sn?: number | string, startSec?: number) {

    if (sn === "initSegment") {

      return;

    }

    if (sn != null && sn === this.lastSn) {

      return;

    }

    const data = payload instanceof Uint8Array ? payload : new Uint8Array(payload);
    const captions = extractCaptions(data);

    const duration = Number.isFinite(durationSec) && durationSec > 0 ? durationSec : 2;
    const start = Number.isFinite(startSec) ? startSec as number : this.lastMediaEnd;
    const end = start + duration;

    if (captions.packets === 0) {

      this.lastNote = "no captions";

      return;

    }

    if (sn != null) {

      this.lastSn = sn;

    }

    const textJunk = isRepeatingJunk(captions.text);
    const liveShow = captions.live > 0 && captions.letters >= MIN_LIVE_CAPTION_LETTERS && !textJunk;
    const someLiveCaptions = captions.live > 0 && captions.letters > 0 && !textJunk;
    const scriptedCaptions = captions.scripted >= MIN_SCRIPTED_COMMANDS;
    const watermark = isRepeatingJunk(captions.extraText);

    if (liveShow) {

      this.seenLiveShow = true;
      this.inScriptedShow = false;

      this.markShow(start, end, "live show");

      return;

    }

    if (!this.inCommercial && someLiveCaptions) {

      this.markShow(start, end, "live captions");

      return;

    }

    if (scriptedCaptions && this.inCommercial) {

      this.scriptedSec += duration;

      if (!this.seenLiveShow && this.scriptedSec >= SCRIPTED_CAPTIONS_TO_END_BREAK_SEC) {

        this.inScriptedShow = true;

        this.markShow(start, end, "scripted show");

        return;

      }

      this.lastNote = "scripted captions";

      this.addSpan(start, end, true);

      return;

    }

    if (!this.seenLiveShow && scriptedCaptions && this.inScriptedShow) {

      this.markShow(start, end, "scripted show");

      return;

    }

    if (!this.seenLiveShow && watermark) {

      this.scriptedSec = 0;
      this.lastNote = "watermark";

      this.addSpan(start, end, this.inCommercial);

      return;

    }

    this.scriptedSec = 0;
    this.silenceSec += duration;

    this.lastNote = captions.live > 0 ? "empty live captions" : "silence";

    if (this.silenceSec >= SILENCE_TO_START_BREAK_SEC) {

      this.inCommercial = true;

      this.markCommercialFrom(end - this.silenceSec);

    }

    this.addSpan(start, end, this.inCommercial);

  }

  private markShow(start: number, end: number, note: string) {

    this.silenceSec = 0;
    this.scriptedSec = 0;
    this.inCommercial = false;
    this.lastNote = note;

    this.addSpan(start, end, false);

  }

  private addSpan(start: number, end: number, commercial: boolean) {

    if (!(end > start)) {

      return;

    }

    const span: Span = { start, end, commercial };

    let i = this.timeline.length;

    while (i > 0 && this.timeline[i - 1].start > start) {

      i--;

    }

    this.timeline.splice(i, 0, span);

    this.lastMediaEnd = Math.max(this.lastMediaEnd, end);

    this.prune();

  }

  private markCommercialFrom(from: number) {

    for (const span of this.timeline) {

      if (span.start >= from - 0.05) {

        span.commercial = true;

      }

    }

  }

  private prune() {

    const cutoff = this.lastMediaEnd - TIMELINE_KEEP_SEC;

    while (this.timeline.length > 2 && this.timeline[0].end < cutoff) {

      this.timeline.shift();

    }

  }

  private spanIndexAt(t: number) {

    if (this.timeline.length === 0 || !Number.isFinite(t)) {

      return -1;

    }

    for (let i = this.timeline.length - 1; i >= 0; i--) {

      if (t >= this.timeline[i].start) {

        return i;

      }

    }

    return -1;

  }

}

function extractCaptions(data: Uint8Array) {

  let packets = 0;

  let live = 0;
  let scripted = 0;

  let letters = 0;
  let tsPackets = 0;

  let text = "";
  let extraText = "";

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
    packets++;

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
      const field = marker & 0x03;

      if (!valid || field > 1) {

        continue;

      }

      if (b1 >= 0x10 && b1 <= 0x1f) {

        if (field === 0 && (b1 === 0x14 || b1 === 0x15)) {

          // 0x25–0x27 live scrolling captions; 0x20/0x2f/0x29 pre-written caption blocks.
          if (b2 === 0x25 || b2 === 0x26 || b2 === 0x27) {

            live++;

          } else if (b2 === 0x20 || b2 === 0x2f || b2 === 0x29) {

            scripted++;

          }

        }

        continue;

      }

      if (b1 === 0 && b2 === 0) {

        continue;

      }

      const chunk = printableCaptionByte(b1) + printableCaptionByte(b2);

      if (field === 0) {

        letters += letterCount(b1) + letterCount(b2);
        text += chunk;

      } else {

        extraText += chunk;

      }

    }

  }

  return { packets, live, scripted, letters, tsPackets, text, extraText };

}

function isRepeatingJunk(text: string) {

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

function printableCaptionByte(b: number) {

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
