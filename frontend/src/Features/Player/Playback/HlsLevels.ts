export type HlsLevelLike = {

  attrs?: Record<string, string | undefined>;
  height?: number;
  videoCodec?: string;
  codecSet?: string;

};

const videoCodecFromLevel = (level: HlsLevelLike): string => {

  const explicit = level.videoCodec?.trim();
  if (explicit) return explicit;

  const codecs = (level.attrs?.["CODECS"] ?? level.codecSet ?? "").split(",");
  const video = codecs.find((codec) => /^(avc1|avc3|hvc1|hev1|dvh1|dvhe|av01|vp09)\./i.test(codec.trim()));

  return video?.trim() ?? "";

};

export const isHdrLevel = (level: HlsLevelLike): boolean => {

  const videoRange = level.attrs?.["VIDEO-RANGE"];
  const codec = videoCodecFromLevel(level);

  return (
    videoRange === "PQ" ||
    videoRange === "HLG" ||
    /hvc1\.2\./i.test(codec) ||
    /hev1\.2\./i.test(codec) ||
    /dvh1\.|dvhe\./i.test(codec)
  );

};

const isHlsLevelSupported = (level: HlsLevelLike): boolean => {

  const codec = videoCodecFromLevel(level);

  if (!codec) return true;

  const mime = `video/mp4; codecs="${codec}"`;

  if (window.MediaSource?.isTypeSupported(mime)) return true;

  const video = document.createElement("video");

  return video.canPlayType(mime) !== "";

};

export const bestSupportedHlsLevel = (levels: HlsLevelLike[], selectedHeight: number): { index: number; isExact: boolean } | null => {

  const supported = levels
    .map((level, index) => ({ level, index }))
    .filter(({ level }) => isHlsLevelSupported(level));

  if (supported.length === 0) return null;

  const capped = selectedHeight > 0
    ? supported.filter(({ level }) => (level.height ?? 0) > 0 && (level.height ?? 0) <= selectedHeight)
    : supported;

  const target = (capped.length > 0 ? capped : supported)
    .reduce((best, item) => ((item.level.height ?? 0) > (best.level.height ?? 0) ? item : best));

  return {

    index: target.index,
    isExact: selectedHeight <= 0 || (target.level.height ?? 0) === selectedHeight,

  };

};
