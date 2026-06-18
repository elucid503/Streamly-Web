// iOS Safari/PWA renders framer-motion transform tweens (page slides, etc.) janky, so  we detect the platform once and disable motion globally there.
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
