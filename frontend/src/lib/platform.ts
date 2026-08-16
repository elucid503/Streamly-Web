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

  return isMobile() || prefersReducedMotion();

}
