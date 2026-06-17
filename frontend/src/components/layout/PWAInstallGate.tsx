import { Component } from "react";

type Platform = "ios" | "android" | "other";

interface State {

  show: boolean;
  platform: Platform;

}

const isMobile = () => /Mobi|Android|iPhone|iPad|iPod/i.test(navigator.userAgent);

const isStandalone = () =>
  window.matchMedia("(display-mode: standalone)").matches ||
  (navigator as Navigator & { standalone?: boolean }).standalone === true;

const detectPlatform = (): Platform => {

  if (/iPhone|iPad|iPod/i.test(navigator.userAgent)) return "ios";
  if (/Android/i.test(navigator.userAgent)) return "android";
  return "other";

};

const IOSSteps = () => (

  <ol className="space-y-4 text-sm text-foreground-muted">

    <li className="flex items-start gap-3">

      <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-surface-overlay text-xs font-medium text-foreground">
        1
      </span>

      <span className="pt-0.5">
        Tap the <strong className="text-foreground">Share</strong> button at the bottom (or top) of your browser.
      </span>

    </li>

    <li className="flex items-start gap-3">

      <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-surface-overlay text-xs font-medium text-foreground">
        2
      </span>

      <span className="pt-0.5">
        Scroll down and tap the <strong className="text-foreground">Add to Home Screen</strong> option.
      </span>

    </li>

    <li className="flex items-start gap-3">

      <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-surface-overlay text-xs font-medium text-foreground">
        3
      </span>

      <span className="pt-0.5">
        Tap <strong className="text-foreground">Add</strong> to confirm.
      </span>

    </li>

  </ol>

);

const AndroidSteps = () => (

  <ol className="space-y-4 text-sm text-foreground-muted">

    <li className="flex items-start gap-3">

      <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-surface-overlay text-xs font-medium text-foreground">
        1
      </span>

      <span className="pt-0.5">
        Find and tap the <strong className="text-foreground">menu</strong> in your browser.
      </span>

    </li>

    <li className="flex items-start gap-3">

      <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-surface-overlay text-xs font-medium text-foreground">
        2
      </span>

      <span className="pt-0.5">
        Tap <strong className="text-foreground">Add to Home screen</strong> or <strong className="text-foreground">Install app</strong>.
      </span>

    </li>

    <li className="flex items-start gap-3">

      <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-surface-overlay text-xs font-medium text-foreground">
        3
      </span>

      <span className="pt-0.5">
        Tap <strong className="text-foreground">Install</strong> to confirm.
      </span>

    </li>

  </ol>

);

export class PWAInstallGate extends Component<object, State> {

  state: State = {

    show: false,
    platform: "other",

  };

  componentDidMount() {

    if (isMobile() && !isStandalone()) {

      this.setState({ show: true, platform: detectPlatform() });

    }

  }

  render() {

    const { show, platform } = this.state;

    if (!show) return null;

    return (

      <div className="fixed inset-0 z-[100] flex flex-col items-center justify-center bg-surface/90 backdrop-blur-md px-6">

        <div className="w-full max-w-sm rounded-2xl border border-border-subtle bg-surface p-6">

          <div className="mb-6 flex items-center gap-4">

            <img src="/icon-192.png" alt="Streamly" className="h-14 w-14 rounded-2xl shadow-sm" />

            <div>

              <p className="text-base font-semibold text-foreground">Install Streamly</p>

              <p className="mt-0.5 text-sm text-foreground-muted">
                Add to your home screen to continue.
              </p>

            </div>

          </div>

          {platform === "ios" && <IOSSteps />}
          {platform === "android" && <AndroidSteps />}

          {platform === "other" && (

            <p className="text-sm text-foreground-muted">
              Open this page in your mobile browser and use the browser menu to
              add Streamly to your home screen.
            </p>

          )}

        </div>

      </div>

    );

  }

}
