import { api } from "@/api/client";

import { ContentRow } from "@/components/catalog/ContentRow";
import { LiveLogo } from "@/components/catalog/LiveLogo";
import { TVGuide } from "@/components/catalog/TVGuide";

import { cn } from "@/lib/utils";
import type { FavoriteItem, LiveChannel } from "@/lib/types";

import { Component } from "react";
import { motion } from "framer-motion";
import { Star } from "lucide-react";

interface LiveViewProps {

  onSelect: (channel: LiveChannel) => void;
  onFavoriteToggle: (channel: LiveChannel | FavoriteItem) => void;

  searchQuery: string;
  favorites: FavoriteItem[];

  category?: string;
  onCategoriesChange?: (options: { value: string; label: string }[]) => void;

}

interface LiveViewState {

  popular: LiveChannel[];
  all: LiveChannel[];
  searchResults: LiveChannel[];

  loading: boolean;

}

export class LiveView extends Component<LiveViewProps, LiveViewState> {

  state: LiveViewState = {

    popular: [],
    all: [],
    searchResults: [],

    loading: true,

  };

  async componentDidMount() {

    await this.load();

  }

  async componentDidUpdate(prev: LiveViewProps) {

    if (prev.searchQuery !== this.props.searchQuery) {

      await this.search(this.props.searchQuery);

    }

  }

  load = async () => {

    try {

      const [popular, all] = await Promise.all([
        api.livePopular(16),
        api.liveChannels(),
      ]);

      this.setState({ popular, all, loading: false });

      const counts = new Map<string, number>();

      for (const ch of all) {

        if (!ch.category) continue;

        counts.set(ch.category, (counts.get(ch.category) ?? 0) + 1);

      }

      const categories = Array.from(counts.entries())

        .filter(([, count]) => count >= 4)
        .sort((a, b) => b[1] - a[1])
        .slice(0, 10)
        .map(([name]) => name)
        .sort();

      this.props.onCategoriesChange?.([

        { value: "all", label: "All" },
        ...categories.map((c) => ({ value: c, label: c })),

      ]);

    } catch {

      this.setState({ loading: false });

    }

  };

  search = async (query: string) => {

    if (!query.trim()) {

      this.setState({ searchResults: [] });

      return;

    }

    try {

      const results = await api.liveSearch(query);

      this.setState({ searchResults: results });

    } catch {

      this.setState({ searchResults: [] });

    }

  };

  isFavorite = (id: string) => {

    return this.props.favorites.some((item) => item.kind === "live" && item.channelId === id);

  };

  favoriteAsChannel = (item: FavoriteItem): LiveChannel => ({

    id: item.channelId ?? item.id,
    name: item.title,
    slug: "",
    code: "",
    logo: item.poster,
    country: "",
    category: item.category ?? "",

  });

  renderChannel = (channel: LiveChannel) => {

    const { onSelect, onFavoriteToggle } = this.props;

    const favorite = this.isFavorite(channel.id);

    return (

      <motion.div className="group relative flex w-[140px] flex-shrink-0 flex-col items-center gap-2 sm:w-[160px]"

        key={channel.id}
        whileHover={{ y: -2 }}

      >

        <button className="flex w-full flex-col items-center gap-2" type="button" onClick={() => onSelect(channel)}>

          <LiveLogo className="h-20 w-20 border border-border-subtle bg-surface-raised transition-colors group-hover:border-border sm:h-24 sm:w-24"

            channel={channel}

            imgClassName="object-contain p-3"

            rounded="rounded-full"

          />

          <p className="line-clamp-2 text-center text-xs font-medium text-foreground group-hover:text-accent">

            {channel.name}

          </p>

          {channel.category && (

            <p className="text-[10px] text-foreground-faint">{channel.category}</p>

          )}

        </button>

        <button className={cn(

            "absolute right-6 top-0 flex h-7 w-7 items-center justify-center rounded-full border border-border-subtle bg-surface/80 text-foreground shadow-sm backdrop-blur-md transition-colors hover:bg-surface-overlay",
            favorite && "text-accent"

          )}

          type="button"
          title={favorite ? "Remove from favorites" : "Add to favorites"}
          onClick={() => onFavoriteToggle(channel)}

        >

          <Star size={14} fill={favorite ? "currentColor" : "none"} />

        </button>

      </motion.div>

    );

  };

  renderGridChannel = (channel: LiveChannel) => {

    const favorite = this.isFavorite(channel.id);

    return (

      <div key={channel.id} className="relative">

        <button onClick={() => this.props.onSelect(channel)}
          className={cn(
            "flex h-full w-full items-center gap-3 rounded-md border border-border-subtle bg-surface-raised p-3 pr-10 text-left transition-colors hover:border-border hover:bg-surface-overlay"
          )}
        >
          <LiveLogo className="h-10 w-10 flex-shrink-0 bg-surface-overlay"

            channel={channel}

            imgClassName="object-contain p-1.5"
            rounded="rounded-full"

          />

          <div className="min-w-0">

            <p className="truncate text-xs font-medium">{channel.name}</p>

            <p className="truncate text-[10px] text-foreground-faint">
              {channel.country || channel.code.toUpperCase()}
            </p>

          </div>

        </button>

        <button className={cn(

            "absolute right-2 top-1/2 flex h-7 w-7 -translate-y-1/2 items-center justify-center rounded-full text-foreground-muted transition-colors hover:bg-surface-overlay hover:text-foreground",
            favorite && "text-accent"

          )}

          type="button"
          title={favorite ? "Remove from favorites" : "Add to favorites"}
          onClick={() => this.props.onFavoriteToggle(channel)}

        >

          <Star size={14} fill={favorite ? "currentColor" : "none"} />

        </button>

      </div>

    );

  };

  render() {

    const { searchQuery, favorites, category } = this.props;

    const { popular, all, searchResults, loading } = this.state;

    const byCategory = (channel: LiveChannel) => !category || category === "all" || channel.category === category;

    const favoriteChannels = favorites.filter((item) => item.kind === "live");
    const showing = searchQuery.trim() ? searchResults.filter(byCategory) : null;
    const filteredPopular = popular.filter(byCategory);
    const filteredAll = all.filter(byCategory);

    return (

      <div className="animate-fade-in py-6">

        {showing ? (

          <>

            {(showing.length > 0 || loading) && (

              <ContentRow title="Channels" loading={loading}>

                {showing.map((ch) => this.renderChannel(ch))}

              </ContentRow>

            )}

            {!loading && showing.length === 0 && (

              <div className="px-4 py-16 text-center text-sm text-foreground-muted sm:px-8">

                No results found

              </div>

            )}

          </>

        ) : (

            <>

            {favoriteChannels.length > 0 && (

              <ContentRow title="Favorites" sectionId="live-favorites">

                {favoriteChannels.map((item) => this.renderChannel(this.favoriteAsChannel(item)))}

              </ContentRow>

            )}

            <TVGuide onSelect={this.props.onSelect} />

            <ContentRow title="Popular Channels" sectionId="live-popular" loading={loading}>

              {filteredPopular.map((ch) => this.renderChannel(ch))}

            </ContentRow>

            <section id="live-all" className="scroll-mt-36 px-4 sm:px-8">

              <h2 className="mb-4 text-sm font-medium tracking-wide text-foreground-muted uppercase">

                All Channels

              </h2>

              <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6">

                {filteredAll.map((channel) => this.renderGridChannel(channel))}

              </div>

            </section>

            </>

        )}

      </div>

    );

  }

}
