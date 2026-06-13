import { Component } from "react";
import { motion } from "framer-motion";
import { Radio } from "lucide-react";
import { api } from "@/api/client";
import { ContentRow } from "@/components/catalog/ContentRow";
import { CachedImage } from "@/components/ui/CachedImage";
import { cn } from "@/lib/utils";
import type { LiveChannel } from "@/lib/types";

interface LiveViewProps {
  onSelect: (channel: LiveChannel) => void;
  searchQuery: string;
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

  renderChannel = (channel: LiveChannel) => {
    const { onSelect } = this.props;
    return (
      <motion.button
        key={channel.daddyId}
        type="button"
        onClick={() => onSelect(channel)}
        whileHover={{ y: -2 }}
        className="group flex w-[140px] flex-shrink-0 flex-col items-center gap-2 sm:w-[160px]"
      >
        <CachedImage
          src={channel.logo}
          alt={channel.name}
          className="h-20 w-20 border border-border-subtle bg-surface-raised transition-colors group-hover:border-border sm:h-24 sm:w-24"
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
      </motion.button>
    );
  };

  render() {
    const { searchQuery } = this.props;
    const { popular, all, searchResults, loading } = this.state;
    const showing = searchQuery.trim() ? searchResults : null;

    return (
      <div className="animate-fade-in py-6">
        {showing ? (
          <ContentRow title={`Results for "${searchQuery}"`} loading={loading}>
            {showing.map((ch) => this.renderChannel(ch))}
          </ContentRow>
        ) : (
          <>
            <ContentRow title="Popular Channels" loading={loading}>
              {popular.map((ch) => this.renderChannel(ch))}
            </ContentRow>

            <section className="px-4 sm:px-8">
              <h2 className="mb-4 text-sm font-medium tracking-wide text-foreground-muted uppercase">
                All Channels
              </h2>
              <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6">
                {all.map((channel) => (
                  <button
                    key={channel.daddyId}
                    onClick={() => this.props.onSelect(channel)}
                    className={cn(
                      "flex items-center gap-3 rounded-md border border-border-subtle bg-surface-raised p-3 text-left transition-colors hover:border-border hover:bg-surface-overlay",
                    )}
                  >
                    <CachedImage
                      src={channel.logo}
                      alt={channel.name}
                      className="h-10 w-10 flex-shrink-0 bg-surface-overlay"
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
                ))}
              </div>
            </section>
          </>
        )}
      </div>
    );
  }
}