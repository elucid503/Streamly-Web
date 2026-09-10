// iOS Safari/PWA renders framer-motion transform tweens (page slides, etc.) janky, so we detect the platform once and disable motion globally there.
export function isIOS(): boolean {

  if (typeof navigator === "undefined") {

    return false;

  }

  const ua = navigator.userAgent;

  if (/iPhone|iPad|iPod/i.test(ua)) {

    return true;

  }

  // iPadOS 13+ reports as MacIntel in desktop mode.
  return navigator.platform === "MacIntel" && navigator.maxTouchPoints > 1;

}

export function isStandalone(): boolean {

  if (typeof window === "undefined") {

    return false;

  }

  return window.matchMedia("(display-mode: standalone)").matches ||
    (navigator as Navigator & { standalone?: boolean }).standalone === true;

}

export function iosNeedsInstallForPush(): boolean {

  return isIOS() && !isStandalone();

}

export function pushSupported(): boolean {

  return typeof window !== "undefined" &&
    "serviceWorker" in navigator &&
    "PushManager" in window &&
    "Notification" in window;

}

export function isTV(): boolean {

  if (typeof navigator === "undefined") {

    return false;

  }

  return /Web0S|WebOS|SmartTV|SMART-TV|Tizen|NetCast|VIDAA|Viera|Bravia|Roku|AppleTV|PlayStation|Xbox/i.test(navigator.userAgent);

}

function uaMajor(ua: string, pattern: RegExp): number {

  const match = ua.match(pattern);

  return match ? Number(match[1]) : 0;

}

// onnxruntime-web ships WASM SIMD (~23MB). TVs and old engines hang or black-screen if it loads.
export function supportsOnnxWasm(): boolean {

  if (typeof navigator === "undefined") {

    return false;

  }

  if (isTV() || isIOS()) {

    return false;

  }

  if (typeof Worker === "undefined" || typeof WebAssembly === "undefined") {

    return false;

  }

  const ua = navigator.userAgent;

  const edge = uaMajor(ua, /Edg\/(\d+)/);

  if (edge) return edge >= 91;

  const firefox = uaMajor(ua, /Firefox\/(\d+)/);

  if (firefox) return firefox >= 90;

  const chrome = uaMajor(ua, /Chrome\/(\d+)/);

  if (chrome) return chrome >= 91;

  const safari = /Safari\//.test(ua) && !/Chrome\/|Chromium\/|Android/i.test(ua);

  if (safari) return uaMajor(ua, /Version\/(\d+)/) >= 16;

  return false;

}

export function isMobile(): boolean {

  if (typeof navigator === "undefined") {

    return false;

  }

  if (isIOS()) {

    return true;

  }

  if (/Android|Mobi|IEMobile|Opera Mini/i.test(navigator.userAgent)) {

    return true;

  }

  if (typeof window === "undefined") {

    return false;

  }

  return window.matchMedia("(pointer: coarse)").matches && window.innerWidth < 768;

}

export function prefersReducedMotion(): boolean {

  if (typeof window === "undefined") {

    return false;

  }

  return window.matchMedia("(prefers-reduced-motion: reduce)").matches;

}

// Phones drop frames on spring and layout animations regardless of OS, so mobile takes the reduced-motion path whether or not the system setting is on.
export function shouldReduceMotion(): boolean {

  return isMobile() || isTV() || prefersReducedMotion();

}
