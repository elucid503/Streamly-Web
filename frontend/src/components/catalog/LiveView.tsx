import { api } from "@/api/client";

import { ContentRow } from "@/components/catalog/ContentRow";
import { CachedImage } from "@/components/ui/CachedImage";

import { cn } from "@/lib/utils";
import type { FavoriteItem, LiveChannel } from "@/lib/types";

import { Component } from "react";
import { motion } from "framer-motion";
import { Radio, Star } from "lucide-react";

interface LiveViewProps {

  onSelect: (channel: LiveChannel) => void;
  onFavoriteToggle: (channel: LiveChannel | FavoriteItem) => void;

  searchQuery: string;
  favorites: FavoriteItem[];

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

      const [popular, all] = await Promise.all([api.livePopular(16), api.liveChannels()]);

      this.setState({ popular, all, loading: false });

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

  isFavorite = (daddyId: string) => {

    return this.props.favorites.some((item) => item.kind === "live" && item.channelId === daddyId);

  };

  favoriteAsChannel = (item: FavoriteItem): LiveChannel => ({

    id: item.channelId ?? item.id,
    daddyId: item.channelId ?? item.id,
    name: item.title,
    slug: "",
    logo: item.poster,
    country: "",
    category: item.category ?? "",

  });

  renderChannel = (channel: LiveChannel) => {

    const { onSelect, onFavoriteToggle } = this.props;

    const favorite = this.isFavorite(channel.daddyId);

    return (

      <motion.div className="group relative flex w-[140px] flex-shrink-0 flex-col items-center gap-2 sm:w-[160px]"

        key={channel.daddyId}
        whileHover={{ y: -2 }}

      >

        <button className="flex w-full flex-col items-center gap-2" type="button" onClick={() => onSelect(channel)}>

          <CachedImage className="h-20 w-20 border border-border-subtle bg-surface-raised transition-colors group-hover:border-border sm:h-24 sm:w-24"

            src={channel.logo}
            alt={channel.name}

            imgClassName="object-contain p-3"

            rounded="rounded-full"
            fallback={<Radio size={24} className="text-foreground-faint" />}

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

    const favorite = this.isFavorite(channel.daddyId);

    return (

      <div key={channel.daddyId} className="relative">

        <button onClick={() => this.props.onSelect(channel)}
          className={cn(
            "flex h-full w-full items-center gap-3 rounded-md border border-border-subtle bg-surface-raised p-3 pr-10 text-left transition-colors hover:border-border hover:bg-surface-overlay"
          )}
        >
          <CachedImage className="h-10 w-10 flex-shrink-0 bg-surface-overlay"

            src={channel.logo}
            alt={channel.name}

            imgClassName="object-contain p-1.5"
            rounded="rounded-full"

            fallback={<Radio size={14} className="text-foreground-faint" />}

          />

          <div className="min-w-0">

            <p className="truncate text-xs font-medium">{channel.name}</p>

            <p className="truncate text-[10px] text-foreground-faint">
              {channel.country}
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

    const { searchQuery, favorites } = this.props;

    const { popular, all, searchResults, loading } = this.state;

    const favoriteChannels = favorites.filter((item) => item.kind === "live");
    const showing = searchQuery.trim() ? searchResults : null;

    return (

      <div className="animate-fade-in py-6">

        {showing ? (

          <ContentRow title={`Results for "${searchQuery}"`} loading={loading}>

            {showing.map((ch) => this.renderChannel(ch))}

          </ContentRow>

        ) : (

            <>

            {favoriteChannels.length > 0 && (

              <ContentRow title="Favorites" sectionId="live-favorites">

                {favoriteChannels.map((item) => this.renderChannel(this.favoriteAsChannel(item)))}

              </ContentRow>

            )}

            <ContentRow title="Popular Channels" sectionId="live-popular" loading={loading}>

              {popular.map((ch) => this.renderChannel(ch))}

            </ContentRow>

            <section id="live-all" className="scroll-mt-36 px-4 sm:px-8">

              <h2 className="mb-4 text-sm font-medium tracking-wide text-foreground-muted uppercase">

                All Channels

              </h2>

              <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6">

                {all.map((channel) => this.renderGridChannel(channel))}

              </div>

            </section>

            </>

        )}

      </div>

    );

  }

}
