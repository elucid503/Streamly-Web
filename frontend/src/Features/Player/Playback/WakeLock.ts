type Sentinel = Awaited<ReturnType<WakeLock["request"]>>;

// Keeps the screen on while playing. The browser drops the lock whenever the
// document is hidden, so callers must re-enable on visibilitychange.
export class ScreenWakeLock {

  private sentinel: Sentinel | null = null;
  private wanted = false;

  enable = (): void => {

    this.wanted = true;

    if (this.sentinel && !this.sentinel.released) return;

    const api: WakeLock | undefined = navigator.wakeLock;

    if (!api || document.visibilityState !== "visible") return;

    void api.request("screen").then((sentinel) => {

      if (!this.wanted) {

        void sentinel.release().catch(() => {});
        return;

      }

      this.sentinel = sentinel;

      sentinel.addEventListener("release", () => {

        if (this.sentinel === sentinel) this.sentinel = null;

      });

    }).catch(() => {});

  };

  disable = (): void => {

    this.wanted = false;

    const sentinel = this.sentinel;

    this.sentinel = null;

    void sentinel?.release().catch(() => {});

  };

}
