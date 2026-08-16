import { api } from "@/api/client";

import { ContentRow } from "@/components/catalog/ContentRow";
import { LiveLogo } from "@/components/catalog/LiveLogo";
import { TVGuide } from "@/components/catalog/TVGuide";
import { Button } from "@/components/ui/Button";
import { HScrollRow } from "@/components/ui/HScrollRow";
import { PlayOverlay } from "@/components/ui/PlayOverlay";
import { SelectMenu } from "@/components/ui/SelectMenu";

import { cn } from "@/lib/utils";
import type { FavoriteItem, LiveChannel } from "@/lib/types";

import { Component } from "react";
import { Star } from "lucide-react";

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

  category: string | null;
  region: string | null;

  loading: boolean;

}

function facets(values: (string | undefined)[], limit = 7): string[] {

  const counts = new Map<string, number>();

  for (const value of values) {

    if (value) counts.set(value, (counts.get(value) ?? 0) + 1);

  }

  return [...counts.entries()]
    .sort((a, b) => b[1] - a[1])
    .slice(0, limit)
    .map(([value]) => value);

}

function regionLabel(channel: LiveChannel): string {

  return channel.countryName || channel.country || channel.code.toUpperCase() || "";

}

export class LiveView extends Component<LiveViewProps, LiveViewState> {

  state: LiveViewState = {

    popular: [],
    all: [],
    searchResults: [],

    category: null,
    region: null,

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

  matchesFilters = (channel: LiveChannel) => {

    const { category, region } = this.state;

    if (category && channel.category !== category) return false;

    if (region) {

      const label = regionLabel(channel);

      if (label !== region && channel.country !== region && channel.countryName !== region) return false;

    }

    return true;

  };

  renderChannelCard = (channel: LiveChannel) => {

    const { onSelect, onFavoriteToggle } = this.props;
    const favorite = this.isFavorite(channel.id);
    const region = regionLabel(channel);
    const meta = [region, channel.category].filter(Boolean).join(" · ");

    return (

      <div key={channel.id} className="group relative">

        <button
          type="button"
          onClick={() => onSelect(channel)}
          className={cn(
            "flex h-full w-full flex-col gap-2.5 rounded-lg border border-border bg-surface-raised p-3.5 text-left outline-none transition-[filter,border-color] hover:brightness-110 focus-visible:ring-[3px] focus-visible:ring-accent/40",
          )}
        >

          <div className="relative overflow-hidden rounded-md">

            <LiveLogo
              className="flex h-20 w-full items-center justify-center bg-surface-overlay sm:h-24"
              channel={channel}
              imgClassName="max-h-14 max-w-[85%] object-contain sm:max-h-16"
              rounded="rounded-md"
            />

            <PlayOverlay className="rounded-md" />

          </div>

          <div className="min-w-0 pr-7">

            <p className="truncate text-base font-medium leading-snug">{channel.name}</p>

            {meta && (

              <p className="mt-1 truncate text-sm text-foreground-muted">{meta}</p>

            )}

          </div>

        </button>

        <button
          className={cn(
            "absolute top-2 right-2 z-20 flex size-8 items-center justify-center rounded-md border border-border-subtle bg-surface/80 text-foreground-muted shadow-sm backdrop-blur-md transition-colors hover:bg-surface-overlay hover:text-foreground",
            favorite && "text-accent",
          )}
          type="button"
          title={favorite ? "Remove from favorites" : "Add to favorites"}
          onClick={() => onFavoriteToggle(channel)}
        >

          <Star className="size-3.5" fill={favorite ? "currentColor" : "none"} />

        </button>

      </div>

    );

  };

  renderFilters = (channels: LiveChannel[]) => {

    const { category, region } = this.state;

    const categories = facets(channels.map((ch) => ch.category));
    const regions = facets(

      channels.map((ch) => regionLabel(ch) || undefined),
      40,

    );

    if (categories.length === 0 && regions.length === 0) return null;

    const regionOptions = [

      { value: "all", label: "Everywhere" },
      ...regions.map((value) => ({ value, label: value })),

    ];

    return (

      <div className="mb-4 px-4 sm:px-8">

        <HScrollRow className="items-center gap-1.5">

          {regions.length > 0 && (

            <SelectMenu
              value={region ?? "all"}
              options={regionOptions}
              onChange={(value) => this.setState({ region: value === "all" ? null : value })}
              label="Region"
              className="shrink-0 [&_button]:min-w-28"
            />

          )}

          {categories.length > 0 && (

            <>

              <Button
                variant={category === null ? "default" : "secondary"}
                size="sm"
                className="shrink-0"
                onClick={() => this.setState({ category: null })}
              >

                All

              </Button>

              {categories.map((value) => (

                <Button
                  key={value}
                  variant={category === value ? "default" : "secondary"}
                  size="sm"
                  className="shrink-0"
                  onClick={() => this.setState({ category: category === value ? null : value })}
                >

                  {value}

                </Button>

              ))}

            </>

          )}

        </HScrollRow>

      </div>

    );

  };

  renderGrid = (channels: LiveChannel[]) => {

    return (

      <div className="grid grid-cols-[repeat(auto-fill,minmax(10.5rem,1fr))] gap-3.5 px-4 sm:grid-cols-[repeat(auto-fill,minmax(11.5rem,1fr))] sm:px-8">

        {channels.map((channel) => this.renderChannelCard(channel))}

      </div>

    );

  };

  render() {

    const { searchQuery, favorites } = this.props;
    const { popular, all, searchResults, loading } = this.state;

    const favoriteChannels = favorites.filter((item) => item.kind === "live");
    const showing = searchQuery.trim() ? searchResults.filter(this.matchesFilters) : null;
    const filteredPopular = popular.filter(this.matchesFilters);
    const filteredAll = all.filter(this.matchesFilters);
    const filterSource = all.length > 0 ? all : popular;

    return (

      <div className="animate-fade-in py-8">

        {showing ? (

          <>

            {(showing.length > 0 || loading) && (

              <section className="scroll-mt-36">

                <h2 className="mb-4 px-4 text-base font-semibold tracking-tight text-foreground sm:px-8">

                  Channels

                </h2>

                {this.renderFilters(filterSource)}

                {this.renderGrid(showing)}

              </section>

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

              <section className="mb-8 scroll-mt-36">

                <h2 className="mb-4 px-4 text-base font-semibold tracking-tight text-foreground sm:px-8">

                  Favorites

                </h2>

                {this.renderGrid(favoriteChannels.map((item) => this.favoriteAsChannel(item)))}

              </section>

            )}

            <div className="mb-8">

              <TVGuide onSelect={this.props.onSelect} />

            </div>

            <section className="mb-8 scroll-mt-36">

              <h2 className="mb-4 px-4 text-base font-semibold tracking-tight text-foreground sm:px-8">

                Popular Channels

              </h2>

              {this.renderFilters(filterSource)}

              {loading && filteredPopular.length === 0 ? (

                <ContentRow title="" loading />

              ) : filteredPopular.length > 0 ? (

                this.renderGrid(filteredPopular)

              ) : null}

            </section>

            <section id="live-all" className="scroll-mt-36">

              <h2 className="mb-4 px-4 text-base font-semibold tracking-tight text-foreground sm:px-8">

                All Channels

              </h2>

              {this.renderGrid(filteredAll)}

            </section>

          </>

        )}

      </div>

    );

  }

}
