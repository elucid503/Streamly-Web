export interface LogoBackdrop {

  primaryColor: string;
  backgroundColor: string;

}

const cache = new Map<string, Promise<LogoBackdrop>>();

const fallbackBackdrop: LogoBackdrop = {

  primaryColor: "#050505",
  backgroundColor: "#ffffff",

};

export function getLogoBackdrop(src: string): Promise<LogoBackdrop> {

  const key = src.trim();

  if (!key) {

    return Promise.resolve(fallbackBackdrop);

  }

  const cached = cache.get(key);

  if (cached) {

    return cached;

  }

  const promise = sampleLogoBackdrop(key).catch(() => fallbackBackdrop);

  cache.set(key, promise);

  return promise;

}

async function sampleLogoBackdrop(src: string): Promise<LogoBackdrop> {

  const image = await loadImage(src);

  const canvas = document.createElement("canvas");
  const size = 48;

  canvas.width = size;
  canvas.height = size;

  const ctx = canvas.getContext("2d", { willReadFrequently: true });

  if (!ctx) {

    return fallbackBackdrop;

  }

  ctx.clearRect(0, 0, size, size);
  ctx.drawImage(image, 0, 0, size, size);

  const pixels = ctx.getImageData(0, 0, size, size).data;

  let r = 0;
  let g = 0;
  let b = 0;
  let weight = 0;

  for (let i = 0; i < pixels.length; i += 4) {

    const alpha = pixels[i + 3] / 255;

    if (alpha < 0.16) {

      continue;

    }

    r += pixels[i] * alpha;
    g += pixels[i + 1] * alpha;
    b += pixels[i + 2] * alpha;
    weight += alpha;

  }

  if (weight <= 0) {

    return fallbackBackdrop;

  }

  r = Math.round(r / weight);
  g = Math.round(g / weight);
  b = Math.round(b / weight);

  const luminance = relativeLuminance(r, g, b);
  const primaryColor = rgbToHex(r, g, b);

  return {

    primaryColor,
    backgroundColor: tintForLogo(r, g, b, luminance),

  };

}

function loadImage(src: string): Promise<HTMLImageElement> {

  return new Promise((resolve, reject) => {

    const image = new Image();

    image.crossOrigin = "anonymous";
    image.decoding = "async";

    image.onload = () => resolve(image);
    image.onerror = () => reject(new Error("logo image failed"));

    image.src = src;

  });

}

function relativeLuminance(r: number, g: number, b: number): number {

  const [lr, lg, lb] = [r, g, b].map((channel) => {

    const value = channel / 255;

    return value <= 0.03928 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4;

  });

  return 0.2126 * lr + 0.7152 * lg + 0.0722 * lb;

}

function rgbToHex(r: number, g: number, b: number): string {

  return `#${hex(r)}${hex(g)}${hex(b)}`;

}

function tintForLogo(r: number, g: number, b: number, luminance: number): string {

  if (luminance < 0.42) {

    return rgbToHex(
      mixChannel(r, 255, 0.88),
      mixChannel(g, 255, 0.88),
      mixChannel(b, 255, 0.88),
    );

  }

  return rgbToHex(
    mixChannel(r, 0, 0.82),
    mixChannel(g, 0, 0.82),
    mixChannel(b, 0, 0.82),
  );

}

function mixChannel(value: number, target: number, amount: number): number {

  return Math.round(value + (target - value) * amount);

}

function hex(value: number): string {

  return Math.max(0, Math.min(255, value)).toString(16).padStart(2, "0");

}
