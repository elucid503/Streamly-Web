import { api } from "@/api/client";

import { consumeReturnPath, currentPath, history, navigate, parseRoute, saveReturnPath, } from "@/lib/navigation";

import { store } from "@/lib/store";
import { isMobile, shouldReduceMotion } from "@/lib/platform";

import { PWAInstallDesktop } from "@/components/layout/PWAInstallDesktop";
import { Button } from "@/components/ui/Button";

import { Component, lazy, Suspense, type ReactNode } from "react";
import { motion, MotionConfig } from "framer-motion";
import type { Location } from "history";
import { PWAInstallGate } from "./components/layout/PWAInstallGate";

// Lazy loads pages for better performance.

const AuthPage = lazy(() => import("@/pages/AuthPage").then((m) => ({ default: m.AuthPage })));
const DetailPage = lazy(() => import("@/pages/DetailPage").then((m) => ({ default: m.DetailPage })) );
const HomePage = lazy(() => import("@/pages/HomePage").then((m) => ({ default: m.HomePage })));
const WatchPage = lazy(() => import("@/pages/WatchPage").then((m) => ({ default: m.WatchPage })));

interface AppState {

  location: Location;
  booting: boolean; // Indicates whether the app is still booting (used to show a loading spinner).
  activeWatchPath: string | null;
  playerReady: boolean;

}

function persistMiniPlayer(): boolean {

  if (typeof window === "undefined") return true;

  if (isMobile()) return false;

  return window.matchMedia("(min-width: 768px) and (pointer: fine)").matches;

}

export class App extends Component<object, AppState> {

  private unlisten = () => {};

  state: AppState = {

    location: history.location,

    booting: true,
    activeWatchPath: parseRoute(history.location).watchPath ?? null,
    playerReady: false,

  };

  async componentDidMount() {

    this.syncPlayerViewport();

    window.visualViewport?.addEventListener("resize", this.syncPlayerViewport);
    window.addEventListener("resize", this.syncPlayerViewport);

    this.unlisten = history.listen(({ location }) => {

      const route = parseRoute(location);

      this.setState((previous) => ({

        location,
        activeWatchPath: route.name === "watch"
          ? route.watchPath ?? null
          : persistMiniPlayer() ? previous.activeWatchPath : null,
        playerReady: route.name === "watch" && route.watchPath !== previous.activeWatchPath
          ? false
          : persistMiniPlayer() || route.name === "watch" ? previous.playerReady : false,

      }));

    });

    await this.bootstrap();

  }

  componentWillUnmount() {

    this.unlisten();

    window.visualViewport?.removeEventListener("resize", this.syncPlayerViewport);
    window.removeEventListener("resize", this.syncPlayerViewport);

  }

  syncPlayerViewport = () => {

    const height = Math.max(
      window.innerHeight,
      window.visualViewport?.height ?? 0,
      document.documentElement.clientHeight,
    );

    document.documentElement.style.setProperty("--player-vvh", `${Math.round(height)}px`);

  };

  bootstrap = async () => {

    const route = parseRoute(history.location);

    if (route.name === "auth") {

      this.setState({ booting: false });

      return;

    }

    try {

      const [user, settings] = await Promise.all([api.me(), api.getSettings()]);

      store.setUser(user);
      store.setSettings(settings);

    } catch {

      store.setUser(null);
      store.setSettings(null);

      saveReturnPath(currentPath(history.location));

      navigate("/auth");

    } finally {

      this.setState({ booting: false });

    }

  };

  onAuthSuccess = async () => {

    const [user, settings] = await Promise.all([api.me(), api.getSettings()]);

    store.setUser(user);

    store.setSettings(settings);

    navigate(consumeReturnPath("/"));

  };

  renderShell = (children: ReactNode) => (

    <Suspense

      fallback={

        <div className="flex min-h-screen flex-col items-center justify-center gap-3 bg-surface">

          <div className="h-8 w-8 animate-spin rounded-full border-2 border-foreground/20 border-t-foreground" />

          <p className="text-sm text-foreground-muted">Loading…</p>

        </div>

      }

    >

      {children}

    </Suspense>

  );

  renderPage(): ReactNode {

    const { location } = this.state;

    const route = parseRoute(location);

    if (!store.isAuthenticated && route.name !== "auth") {

      saveReturnPath(currentPath(location));

      return this.renderShell(<AuthPage onSuccess={this.onAuthSuccess} />);

    }

    switch (route.name) {

      case "auth":

        return this.renderShell(

          store.isAuthenticated ? (

            <HomePage navigate={navigate} />

          ) : (

            <AuthPage onSuccess={this.onAuthSuccess} />

          )

        );

      case "detail":

        return this.renderShell(

          <DetailPage navigate={navigate} kind={route.kind!} id={route.id!} />

        );

      case "watch":

        return null;

      case "home":

        return this.renderShell(<HomePage navigate={navigate} />);

      default:

        return (

          <div className="flex min-h-screen flex-col items-center justify-center gap-4 bg-surface">

            <p className="text-sm text-foreground-muted">Page not found</p>

            <Button variant="outline" size="sm" onClick={() => navigate("/")}>

              Go home

            </Button>

          </div>

        );

    }

  }

  renderPlayer() {

    const { activeWatchPath, location, playerReady } = this.state;

    if (!activeWatchPath || !store.isAuthenticated) return null;

    const route = parseRoute(location);
    const minimized = route.name !== "watch";

    if (minimized && !persistMiniPlayer()) return null;

    return (

      <motion.div
        layout={minimized}
        transition={{ type: "spring", stiffness: 360, damping: 34 }}
        className={minimized
          ? `${playerReady ? "pointer-events-auto visible" : "pointer-events-none invisible"} fixed right-5 bottom-5 z-[80] w-[min(22rem,calc(100vw-2.5rem))] overflow-hidden rounded-xl border border-border-subtle bg-surface shadow-2xl`
          : "player-screen z-[80] bg-black"}
      >

        {this.renderShell(

          <WatchPage
            navigate={navigate}
            watchPath={activeWatchPath}
            minimized={minimized}
            onMinimize={(path) => {

              if (!persistMiniPlayer()) {

                this.setState({ activeWatchPath: null, playerReady: false });

              }

              navigate(path);

            }}
            onReturn={() => navigate(`/watch/${activeWatchPath}`)}
            onDismiss={() => this.setState({ activeWatchPath: null, playerReady: false })}
            onReadyChange={(ready) => this.setState({ playerReady: ready })}
          />

        )}

      </motion.div>

    );

  }

  render() {

    const { booting } = this.state;

    if (booting) {

      return (

        <div className="flex min-h-screen flex-col items-center justify-center gap-3 bg-surface">

          <div className="h-8 w-8 animate-spin rounded-full border-2 border-foreground/20 border-t-foreground" />

          <p className="text-sm text-foreground-muted">Loading…</p>

        </div>

      );

    }

    return (

      <MotionConfig reducedMotion={shouldReduceMotion() ? "always" : "user"}>

        {this.renderPage()}
        {this.renderPlayer()}

        <PWAInstallGate />

        <PWAInstallDesktop />

      </MotionConfig>

    );

  }

}
