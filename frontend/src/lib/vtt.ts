export interface VttCue {
  start: number;
  end: number;
  text: string;
}

const normalizeTimestamp = (value: string) => value.trim().replace(",", ".");

const timeToSeconds = (value: string) => {
  const normalized = normalizeTimestamp(value);
  const parts = normalized.split(":");
  if (parts.length === 3) {
    const [h, m, rest] = parts;
    const [s, ms = "0"] = rest.split(".");
    const seconds = Number(h) * 3600 + Number(m) * 60 + Number(s) + Number(ms) / 1000;
    return Number.isFinite(seconds) ? seconds : NaN;
  }
  if (parts.length === 2) {
    const [m, rest] = parts;
    const [s, ms = "0"] = rest.split(".");
    const seconds = Number(m) * 60 + Number(s) + Number(ms) / 1000;
    return Number.isFinite(seconds) ? seconds : NaN;
  }
  return NaN;
};

const cleanCueText = (lines: string[]) =>
  lines
    .join("\n")
    .replace(/<[^>]+>/g, "")
    .replace(/\{[^}]+\}/g, "")
    .trim();

export const parseVtt = (raw: string): VttCue[] => {
  const lines = raw.replace(/\r/g, "").split("\n");
  const cues: VttCue[] = [];
  let i = 0;

  while (i < lines.length) {
    const line = lines[i].trim();
    i += 1;
    if (!line || line.startsWith("WEBVTT") || line.startsWith("NOTE") || line.startsWith("STYLE")) {
      continue;
    }
    if (!line.includes("-->")) continue;

    const [startRaw, endRaw] = line.split("-->").map((part) => normalizeTimestamp(part.trim().split(" ")[0]));
    const textLines: string[] = [];
    while (i < lines.length && lines[i].trim() !== "") {
      textLines.push(lines[i].trim());
      i += 1;
    }

    const text = cleanCueText(textLines);
    const start = timeToSeconds(startRaw);
    const end = timeToSeconds(endRaw);
    if (!text || !Number.isFinite(start) || !Number.isFinite(end)) continue;

    cues.push({ start, end, text });
  }

  return cues;
};

export const parseSrt = (raw: string): VttCue[] => {
  const lines = raw.replace(/\r/g, "").split("\n");
  const cues: VttCue[] = [];
  let i = 0;

  while (i < lines.length) {
    while (i < lines.length && !lines[i].trim()) i += 1;
    if (i >= lines.length) break;

    if (/^\d+$/.test(lines[i].trim())) i += 1;
    if (i >= lines.length) break;

    const timingLine = lines[i]?.trim() ?? "";
    if (!timingLine.includes("-->")) {
      i += 1;
      continue;
    }
    i += 1;

    const [startRaw, endRaw] = timingLine
      .split("-->")
      .map((part) => normalizeTimestamp(part.trim().split(" ")[0]));

    const textLines: string[] = [];
    while (i < lines.length && lines[i].trim()) {
      textLines.push(lines[i].trim());
      i += 1;
    }

    const text = cleanCueText(textLines);
    const start = timeToSeconds(startRaw);
    const end = timeToSeconds(endRaw);
    if (!text || !Number.isFinite(start) || !Number.isFinite(end)) continue;

    cues.push({ start, end, text });
  }

  return cues;
};

export const srtToVtt = (raw: string) => {
  const cues = parseSrt(raw);
  if (cues.length === 0) return raw;

  const lines = ["WEBVTT", ""];
  for (const cue of cues) {
    const start = formatVttTime(cue.start);
    const end = formatVttTime(cue.end);
    lines.push(`${start} --> ${end}`, cue.text, "");
  }
  return lines.join("\n");
};

const formatVttTime = (seconds: number) => {
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = Math.floor(seconds % 60);
  const ms = Math.round((seconds % 1) * 1000);
  return `${String(h).padStart(2, "0")}:${String(m).padStart(2, "0")}:${String(s).padStart(2, "0")}.${String(ms).padStart(3, "0")}`;
};

const looksLikeSrt = (raw: string) => /-->\s*\d/.test(raw) && !raw.trim().startsWith("WEBVTT");

export const loadSubtitleCues = async (url: string, format: string): Promise<VttCue[]> => {
  const res = await fetch(url, { credentials: "include" });
  if (!res.ok) throw new Error("subtitle fetch failed");
  const raw = await res.text();
  const trimmed = raw.trim();
  if (!trimmed) return [];

  const fmt = format.toLowerCase();
  let cues: VttCue[] = [];

  if (fmt === "srt" || looksLikeSrt(trimmed)) {
    cues = parseSrt(trimmed);
  } else {
    cues = parseVtt(trimmed);
  }

  if (cues.length === 0 && looksLikeSrt(trimmed)) {
    cues = parseSrt(trimmed);
  }
  if (cues.length === 0) {
    cues = parseVtt(srtToVtt(trimmed));
  }
  if (cues.length === 0 && trimmed.startsWith("WEBVTT")) {
    cues = parseVtt(trimmed);
  }

  return cues.filter((cue) => Number.isFinite(cue.start) && Number.isFinite(cue.end) && cue.end > cue.start);
};
