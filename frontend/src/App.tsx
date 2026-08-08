import { api } from "@/api/client";

import { consumeReturnPath, currentPath, history, navigate, parseRoute, saveReturnPath, } from "@/lib/navigation";

import { store } from "@/lib/store";
import { isIOS } from "@/lib/platform";

import { PWAInstallDesktop } from "@/components/layout/PWAInstallDesktop";
import { PWAInstallGate } from "@/components/layout/PWAInstallGate";
import { Button } from "@/components/ui/Button";

import { Component, createRef, lazy, Suspense, type ReactNode } from "react";
import { motion, MotionConfig, type PanInfo } from "framer-motion";
import type { Location } from "history";

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
  miniPlayerCorner: "top-left" | "top-right" | "bottom-left" | "bottom-right";

}

export class App extends Component<object, AppState> {

  private unlisten = () => {};
  private miniPlayerBounds = createRef<HTMLDivElement>();
  private suppressMiniPlayerReturnUntil = 0;

  state: AppState = {

    location: history.location,

    booting: true,
    activeWatchPath: parseRoute(history.location).watchPath ?? null,
    playerReady: false,
    miniPlayerCorner: "bottom-right",

  };

  async componentDidMount() {

    this.unlisten = history.listen(({ location }) => {

      const route = parseRoute(location);

      this.setState((previous) => ({

        location,
        activeWatchPath: route.name === "watch" ? route.watchPath ?? null : previous.activeWatchPath,
        playerReady: route.name === "watch" && route.watchPath !== previous.activeWatchPath ? false : previous.playerReady,

      }));

    });

    await this.bootstrap();

  }

  componentWillUnmount() {

    this.unlisten();

  }

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

    const { activeWatchPath, location, miniPlayerCorner, playerReady } = this.state;

    if (!activeWatchPath || !store.isAuthenticated) return null;

    const route = parseRoute(location);
    const minimized = route.name !== "watch";
    const verticalCorner = miniPlayerCorner.startsWith("top")
      ? "top-0"
      : "bottom-[calc(env(safe-area-inset-bottom,0px)+4.5rem)] sm:bottom-0";
    const horizontalCorner = miniPlayerCorner.endsWith("left") ? "left-0" : "right-0";

    const returnToPlayer = () => {

      if (Date.now() < this.suppressMiniPlayerReturnUntil) return;

      navigate(`/watch/${activeWatchPath}`);

    };

    const snapToCorner = (_event: MouseEvent | TouchEvent | PointerEvent, info: PanInfo) => {

      this.suppressMiniPlayerReturnUntil = Date.now() + 300;

      const vertical = info.point.y < window.innerHeight / 2 ? "top" : "bottom";
      const horizontal = info.point.x < window.innerWidth / 2 ? "left" : "right";

      this.setState({ miniPlayerCorner: `${vertical}-${horizontal}` as AppState["miniPlayerCorner"] });

    };

    return (

      <div ref={this.miniPlayerBounds} className={minimized ? "pointer-events-none fixed inset-3 z-[80] sm:inset-5" : "fixed inset-0 z-[80]"}>

        <motion.div
          layout
          drag={minimized ? true : false}
          dragConstraints={this.miniPlayerBounds}
          dragElastic={0.05}
          dragMomentum={false}
          dragSnapToOrigin
          onDragStart={() => { this.suppressMiniPlayerReturnUntil = Number.POSITIVE_INFINITY; }}
          onDragEnd={snapToCorner}
          animate={{ x: 0, y: 0 }}
          transition={{ type: "spring", stiffness: 360, damping: 34 }}
          className={minimized
            ? `${playerReady ? "pointer-events-auto visible" : "pointer-events-none invisible"} absolute ${verticalCorner} ${horizontalCorner} aspect-video w-[min(22rem,calc(100vw-1.5rem))] overflow-hidden rounded-xl border border-border-subtle bg-surface shadow-2xl`
            : "absolute inset-0 bg-surface"}
        >

          {this.renderShell(

            <WatchPage
              navigate={navigate}
              watchPath={activeWatchPath}
              minimized={minimized}
              onMinimize={navigate}
              onReturn={returnToPlayer}
              onDismiss={() => this.setState({ activeWatchPath: null, playerReady: false })}
              onReadyChange={(ready) => this.setState({ playerReady: ready })}
            />

          )}

        </motion.div>

      </div>

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

      <MotionConfig reducedMotion={isIOS() ? "always" : "user"}>

        {this.renderPage()}
        {this.renderPlayer()}

        <PWAInstallGate />

        <PWAInstallDesktop />

      </MotionConfig>

    );

  }

}
