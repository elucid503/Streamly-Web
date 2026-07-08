import { api } from "@/api/client";

import { SportsRow, MatchTitle, matchedChannelToLiveChannel, prettyCategory } from "@/components/catalog/SportsCard";

import { sportsBackgroundImage } from "@/lib/sportsBackgrounds";
import type { LiveChannel, SportsMatch } from "@/lib/types";

import { Component } from "react";

const STARTING_SOON_WINDOW_SECS = 3 * 60 * 60;
const ROW_PREVIEW_COUNT = 5;
const PAST_PREVIEW_COUNT = 3;

interface SportsPageProps {

  onSelectChannel: (channel: LiveChannel) => void;

  searchQuery: string;
  category?: string;
  onCategoriesChange?: (options: { value: string; label: string }[]) => void;

}

interface SportsPageState {

  matches: SportsMatch[];
  loading: boolean;

  showAllLive: boolean;
  showAllSoon: boolean;
  showAllPast: boolean;

}

export class SportsPage extends Component<SportsPageProps, SportsPageState> {

  state: SportsPageState = {

    matches: [],
    loading: true,

    showAllLive: false,
    showAllSoon: false,
    showAllPast: false,

  };

  async componentDidMount() {

    try {

      const matches = await api.liveSports();

      // Only matches with a resolvable channel are watchable.
      const watchable = (matches ?? []).filter((m) => m.channel);

      this.setState({ matches: watchable, loading: false });

      // Always offer Formula 1 as a filter option, even if no motor-sports
      // match happens to be in the current lookup window.
      const categories = Array.from(new Set([

        "motor-sports",
        ...watchable.map((m) => m.category).filter(Boolean),

      ])).sort();

      this.props.onCategoriesChange?.([

        { value: "all", label: "All" },
        ...categories.map((c) => ({ value: c, label: prettyCategory(c) })),

      ]);

    } catch {

      this.setState({ loading: false });

    }

  }

  filtered = () => {

    const { category, searchQuery } = this.props;
    const { matches } = this.state;

    const query = searchQuery.trim().toLowerCase();

    return matches.filter((m) => {

      if (category && category !== "all" && m.category !== category) return false;

      if (query && !m.title.toLowerCase().includes(query)) return false;

      return true;

    });

  };

  renderSection = (title: string, matches: SportsMatch[], expanded: boolean, onToggle: () => void, previewCount = ROW_PREVIEW_COUNT) => {

    if (matches.length === 0) return null;

    const visible = expanded ? matches : matches.slice(0, previewCount);

    return (

      <section className="mb-8 px-4 sm:px-8">

        <div className="mb-3 flex items-center justify-between">

          <h2 className="text-sm font-medium tracking-wide text-foreground-muted uppercase">{title}</h2>

          {matches.length > previewCount && (

            <button type="button" onClick={onToggle} className="text-xs font-medium text-foreground-muted hover:text-foreground">

              {expanded ? "Show less" : "View More"}

            </button>

          )}

        </div>

        <div className="flex flex-col gap-2.5">

          {visible.map((match) => (

            <SportsRow key={match.id} match={match} onSelect={this.props.onSelectChannel} />

          ))}

        </div>

      </section>

    );

  };

  render() {

    const { loading, showAllLive, showAllSoon, showAllPast } = this.state;

    const matches = this.filtered();

    const nowSecs = Date.now() / 1000;

    const live = matches.filter((m) => m.live);
    const startingSoon = matches.filter((m) => !m.live && m.startsAt - nowSecs <= STARTING_SOON_WINDOW_SECS && m.startsAt - nowSecs > 0);
    const upcoming = matches.filter((m) => !m.live && m.startsAt - nowSecs > STARTING_SOON_WINDOW_SECS);
    const past = matches.filter((m) => !m.live && m.startsAt <= nowSecs);

    // Prioritize a live match for the banner, but always show something.
    const featured = live[0] ?? startingSoon[0] ?? upcoming[0] ?? past[0];

    if (loading) {

      return (

        <div className="animate-fade-in px-4 py-10 sm:px-8">

          <div className="skeleton mb-8 h-56 w-full rounded-xl" />

          <div className="flex flex-col gap-2.5">

            {Array.from({ length: 4 }).map((_, i) => (

              <div key={i} className="skeleton h-[76px] w-full rounded-lg" />

            ))}

          </div>

        </div>

      );

    }

    if (matches.length === 0) {

      return (

        <div className="px-4 py-16 text-center text-sm text-foreground-muted sm:px-8">

          No watchable sports events available right now

        </div>

      );

    }

    return (

      <div className="animate-fade-in py-6">

        {featured && (

          <div className="mb-8 px-4 sm:px-8">

            <div className="relative overflow-hidden rounded-xl">

              <div

                className="absolute inset-0 bg-cover bg-center"
                style={{ backgroundImage: `url(${sportsBackgroundImage(featured.category)})` }}

              />

              <div className="absolute inset-0 bg-black/55" />

              <div className="absolute inset-0 bg-gradient-to-t from-black via-black/85 to-black/40" />

              <div className="relative flex min-h-[280px] flex-col justify-end gap-4 p-6 sm:flex-row sm:items-end sm:justify-between sm:p-8">

                <div className="flex min-w-0 flex-col gap-3">

                  <div className="flex items-center gap-2">

                    {featured.live && (

                      <span className="flex flex-shrink-0 items-center gap-1 rounded-full bg-red-500/90 px-2 py-0.5 text-[11px] font-semibold text-white">

                        <span className="h-1.5 w-1.5 rounded-full bg-white" /> LIVE

                      </span>

                    )}

                    <span className="text-xs font-medium uppercase tracking-wide text-white/70">

                      {prettyCategory(featured.category)}

                    </span>

                  </div>

                  <MatchTitle title={featured.title} className="text-2xl font-bold text-white sm:text-3xl" />

                </div>

                <button type="button" onClick={() => this.props.onSelectChannel(matchedChannelToLiveChannel(featured.channel!, featured.category))}

                  className="flex w-fit flex-shrink-0 items-center gap-2 rounded-full bg-white px-5 py-2.5 text-sm font-semibold text-black transition-opacity hover:opacity-90"

                >

                  Watch on {featured.channel!.name}

                </button>

              </div>

            </div>

          </div>

        )}

        {this.renderSection("Live Now", live, showAllLive, () => this.setState({ showAllLive: !showAllLive }))}

        {this.renderSection("Starting Soon", startingSoon, showAllSoon, () => this.setState({ showAllSoon: !showAllSoon }))}

        {this.renderSection("Past", past, showAllPast, () => this.setState({ showAllPast: !showAllPast }), PAST_PREVIEW_COUNT)}

        {upcoming.length > 0 && (

          <section className="px-4 sm:px-8">

            <h2 className="mb-3 text-sm font-medium tracking-wide text-foreground-muted uppercase">Upcoming</h2>

            <div className="flex flex-col gap-2.5">

              {upcoming.map((match) => (

                <SportsRow key={match.id} match={match} onSelect={this.props.onSelectChannel} />

              ))}

            </div>

          </section>

        )}

      </div>

    );

  }

}
