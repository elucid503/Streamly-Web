import { api } from "@/api/client";

import { continueWatching } from "@/lib/history";
import type { FavoriteItem, MainView, WatchHistoryItem } from "@/lib/types";

import { Component } from "react";
import { motion } from "framer-motion";

interface HomeBackdropProps {

  view: MainView;

  history: WatchHistoryItem[];
  favorites: FavoriteItem[];

}

interface HomeBackdropState {

  panels: Record<MainView, string[]>;

}

const VIEW_ORDER: MainView[] = ["shows", "movies", "live"];

const GRID_TILE_COUNT = 48;
const CATEGORY_SAMPLE = 6;

class PosterCollector {

  private seen = new Set<string>();

  posters: string[] = [];

  constructor(private limit: number) {}

  add = (url?: string) => {

    const trimmed = url?.trim();

    if (!trimmed || this.seen.has(trimmed) || this.posters.length >= this.limit) return;

    this.seen.add(trimmed);
    this.posters.push(trimmed);

  };

  full = () => this.posters.length >= this.limit;

}

export class HomeBackdrop extends Component<HomeBackdropProps, HomeBackdropState> {

  private loadGen = 0;

  state: HomeBackdropState = {

    panels: {

      shows: [],
      movies: [],
      live: [],

    },

  };

  componentDidMount() {

    void this.loadPanels();

  }

  componentDidUpdate(prevProps: HomeBackdropProps) {

    if (prevProps.history !== this.props.history || prevProps.favorites !== this.props.favorites) {

      void this.loadPanels();

    }

  }

  loadCatalogPosters = async (collector: PosterCollector, kind: "show" | "movie") => {

    const trending = kind === "show" ? await api.showTrending(24).catch(() => []) : await api.movieTrending(24).catch(() => []);

    for (const hit of trending ?? []) {

      collector.add(hit.poster);

      if (collector.full()) return;

    }

    const categories = kind === "show" ? await api.showCategories().catch(() => []) : await api.movieCategories().catch(() => []);

    for (const category of (categories ?? []).slice(0, CATEGORY_SAMPLE)) {

      if (collector.full()) return;

      const titles = kind === "show" ? await api.showCategoryTitles(category.id).catch(() => []) : await api.movieCategoryTitles(category.id).catch(() => []);

      for (const hit of titles ?? []) {

        collector.add(hit.poster);

        if (collector.full()) return;

      }

    }

  };

  loadShowPosters = async (history: WatchHistoryItem[], favorites: FavoriteItem[]): Promise<string[]> => {

    const collector = new PosterCollector(GRID_TILE_COUNT);

    continueWatching(history, "show").forEach((item) => collector.add(item.poster));

    favorites.filter((item) => item.kind === "show").forEach((item) => collector.add(item.poster));

    if (!collector.full()) await this.loadCatalogPosters(collector, "show");

    return collector.posters;

  };

  loadMoviePosters = async (history: WatchHistoryItem[], favorites: FavoriteItem[]): Promise<string[]> => {

    const collector = new PosterCollector(GRID_TILE_COUNT);

    continueWatching(history, "movie").forEach((item) => collector.add(item.poster));

    favorites.filter((item) => item.kind === "movie").forEach((item) => collector.add(item.poster));

    if (!collector.full()) await this.loadCatalogPosters(collector, "movie");

    return collector.posters;

  };

  loadPanels = async () => {

    const gen = ++this.loadGen;
    const { history, favorites } = this.props;

    const [shows, movies] = await Promise.all([

      this.loadShowPosters(history, favorites),
      this.loadMoviePosters(history, favorites),

    ]);

    if (gen !== this.loadGen) return;

    this.setState({ panels: { shows, movies, live: [] } });

  };

  renderGrid = (panel: MainView, posters: string[]) => {

    const tileAspect = "aspect-[2/3]";

    return (

      <div className="grid w-full grid-cols-6 content-start gap-[3px] bg-black p-[3px] sm:grid-cols-8 md:grid-cols-10 lg:grid-cols-12">

        {posters.map((poster, index) => (

          <div key={`${panel}-${poster}-${index}`} className={`w-full overflow-hidden bg-black ${tileAspect}`}>

            <img src={poster} alt="" className="block h-full w-full object-cover opacity-[0.1] saturate-[0.8]" loading="lazy" />

          </div>

        ))}

      </div>

    );

  };

  render() {

    const { view } = this.props;
    const { panels } = this.state;

    if (view === "live") return null;

    const hasPanels = VIEW_ORDER.some((panel) => panels[panel].length > 0);

    if (!hasPanels) return null;

    const index = VIEW_ORDER.indexOf(view);

    return (

      <div className="pointer-events-none absolute inset-x-0 top-16 z-0 h-[min(62vh,560px)] overflow-hidden" aria-hidden>

        <motion.div className="flex h-full w-full"

          animate={{ x: `-${index * 100}%` }}
          transition={{ type: "spring", stiffness: 320, damping: 34, mass: 0.8 }}

        >

          {VIEW_ORDER.map((panel) => (

            <div key={panel} className="h-full w-full flex-shrink-0 overflow-hidden">

              {this.renderGrid(panel, panels[panel])}

            </div>

          ))}

        </motion.div>

        <div className="absolute inset-0 bg-[linear-gradient(to_bottom,rgba(10,10,10,0.05)_0%,rgba(10,10,10,0.25)_18%,rgba(10,10,10,0.55)_42%,rgba(10,10,10,0.82)_68%,#0a0a0a_88%,#0a0a0a_100%)]" />

      </div>

    );

  }

}
