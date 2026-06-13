import { api } from "@/api/client";

import { consumeReturnPath, currentPath, history, navigate, parseRoute, saveReturnPath, } from "@/lib/navigation";

import { store } from "@/lib/store";

import { Component, lazy, Suspense, type ReactNode } from "react";
import type { Location } from "history";

// Lazy loads pages for better performance.

const AuthPage = lazy(() => import("@/pages/AuthPage").then((m) => ({ default: m.AuthPage })));
const DetailPage = lazy(() => import("@/pages/DetailPage").then((m) => ({ default: m.DetailPage })) );
const HomePage = lazy(() => import("@/pages/HomePage").then((m) => ({ default: m.HomePage })));
const WatchPage = lazy(() => import("@/pages/WatchPage").then((m) => ({ default: m.WatchPage })));

interface AppState {

  location: Location;
  booting: boolean; // Indicates whether the app is still booting (used to show a loading spinner).

}

export class App extends Component<object, AppState> {

  private unlisten = () => {};

  state: AppState = {

    location: history.location,

    booting: true,

  };

  async componentDidMount() {

    this.unlisten = history.listen(() => {

      this.setState({ location: history.location });

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

        <div className="flex min-h-screen items-center justify-center">

          <div className="h-8 w-8 animate-spin rounded-full border-2 border-foreground/20 border-t-foreground" />

        </div>

      }

    >

      {children}

    </Suspense>

  );

  render() {

    const { location, booting } = this.state;

    if (booting) {

      return (

        <div className="flex min-h-screen items-center justify-center">

          <div className="h-8 w-8 animate-spin rounded-full border-2 border-foreground/20 border-t-foreground" />

        </div>

      );

    }

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

        return this.renderShell(<WatchPage navigate={navigate} watchPath={route.watchPath!} />);

      case "home":

        return this.renderShell(<HomePage navigate={navigate} />);

      default:

        return (

          <div className="flex min-h-screen flex-col items-center justify-center gap-4">

            <p className="text-sm text-foreground-muted">Page not found</p>

            <button onClick={() => navigate("/")} className="text-sm underline-offset-2 hover:underline" >

              Go home

            </button>

          </div>

        );

     }

  }

}
