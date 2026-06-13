import { createBrowserHistory, type History, type Location } from "history";

export const history: History = createBrowserHistory();

const RETURN_KEY = "streamly:returnTo";

export type NavigateFn = (path: string) => void;

export const navigate: NavigateFn = (path) => {

  history.push(path);

};

export function saveReturnPath(path: string) {

  const trimmed = path.trim();

  if (!trimmed || trimmed === "/auth") return;

  sessionStorage.setItem(RETURN_KEY, trimmed);

}

export function consumeReturnPath(fallback = "/"): string {

  const saved = sessionStorage.getItem(RETURN_KEY);

  sessionStorage.removeItem(RETURN_KEY);

  return saved && saved !== "/auth" ? saved : fallback;

}

export function currentPath(location: Location = history.location): string {

  return location.pathname + location.search + location.hash;

}

export function parseRoute(location: Location): {

  name: "home" | "auth" | "detail" | "watch" | "notfound";

  kind?: "movie" | "show";
  id?: string;
  watchPath?: string;

} {

  const path = location.pathname;

  if (path === "/" || path === "") return { name: "home" };

  if (path === "/auth") return { name: "auth" };

  const detail = path.match(/^\/(movie|show)\/(\d+)$/);

  if (detail) {

    return { name: "detail", kind: detail[1] as "movie" | "show", id: detail[2] };

  }

  const watch = path.match(/^\/watch\/(.+)$/);

  if (watch) {

    return { name: "watch", watchPath: watch[1] };

  }

  const live = path.match(/^\/live\/([^/]+)$/);

  if (live) {

    return { name: "watch", watchPath: `live/${live[1]}` };

  }

  return { name: "notfound" };

}