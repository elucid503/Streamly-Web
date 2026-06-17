import { Component } from "react";

interface BeforeInstallPromptEvent extends Event {

  prompt(): Promise<void>;
  userChoice: Promise<{ outcome: "accepted" | "dismissed" }>;

}

interface State {

  show: boolean;
  deferred: BeforeInstallPromptEvent | null;

}

const DISMISSED_KEY = "streamly:pwa-cta-dismissed";

export class PWAInstallDesktop extends Component<object, State> {

  state: State = {

    show: false,
    deferred: null,

  };

  componentDidMount() {

    window.addEventListener("beforeinstallprompt", this.handlePrompt);
    window.addEventListener("appinstalled", this.handleInstalled);

  }

  componentWillUnmount() {

    window.removeEventListener("beforeinstallprompt", this.handlePrompt);
    window.removeEventListener("appinstalled", this.handleInstalled);

  }

  handlePrompt = (e: Event) => {

    e.preventDefault();

    if (localStorage.getItem(DISMISSED_KEY)) return;

    this.setState({ show: true, deferred: e as BeforeInstallPromptEvent });

  };

  handleInstalled = () => {

    this.setState({ show: false, deferred: null });

  };

  handleInstall = async () => {

    const { deferred } = this.state;

    if (!deferred) return;

    await deferred.prompt();

    const { outcome } = await deferred.userChoice;

    if (outcome === "accepted") {

      this.setState({ show: false, deferred: null });

    }

  };

  handleDismiss = () => {

    localStorage.setItem(DISMISSED_KEY, "1");

    this.setState({ show: false });

  };

  render() {

    const { show } = this.state;

    if (!show) return null;

    return (

      <div className="fixed right-4 bottom-4 z-50 w-64 rounded-xl border border-border-subtle bg-surface-raised p-4 shadow-xl">

        <p className="text-sm font-medium text-foreground">Install Streamly</p>

        <p className="mt-1 text-xs text-foreground-muted">
          Add to your desktop for a faster, fullscreen experience.
        </p>

        <div className="mt-4 flex gap-2">

          <button
            type="button"
            onClick={this.handleInstall}
            className="flex-1 rounded-lg bg-foreground py-1.5 text-xs font-medium text-surface transition-opacity hover:opacity-80"
          >
            Install
          </button>

          <button
            type="button"
            onClick={this.handleDismiss}
            className="flex-1 rounded-lg border border-border py-1.5 text-xs font-medium text-foreground-muted transition-colors hover:text-foreground"
          >
            Not now
          </button>

        </div>

      </div>

    );

  }

}
