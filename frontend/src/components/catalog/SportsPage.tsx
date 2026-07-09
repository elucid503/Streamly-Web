import { api } from "@/api/client";

import { SportsRow, MatchTitle, matchedChannelToLiveChannel, prettyCategory } from "@/components/catalog/SportsCard";

import { sportsBackgroundImage } from "@/lib/sportsBackgrounds";
import type { LiveChannel, SportsMatch } from "@/lib/types";

import { Component } from "react";

const STARTING_SOON_WINDOW_SECS = 3 * 60 * 60;
const ROW_PREVIEW_COUNT = 5;
const UPCOMING_CAP = 5;

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

}

export class SportsPage extends Component<SportsPageProps, SportsPageState> {

  private refreshTimer: ReturnType<typeof setInterval> | null = null;

  state: SportsPageState = {

    matches: [],
    loading: true,

    showAllLive: false,
    showAllSoon: false,

  };

  async componentDidMount() {

    await this.loadMatches(true);

    // Scores and live flags move quickly; poll while the page is mounted.
    this.refreshTimer = setInterval(() => {

      void this.loadMatches(false);

    }, 60_000);

  }

  componentWillUnmount() {

    if (this.refreshTimer) clearInterval(this.refreshTimer);

  }

  loadMatches = async (initial: boolean) => {

    try {

      const matches = await api.liveSports();

      // Only matches with a resolvable channel are watchable.
      const watchable = (matches ?? []).filter((m) => m.channel);

      this.setState({ matches: watchable, loading: false });

      if (initial) {

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

      }

    } catch {

      if (initial) this.setState({ loading: false });

    }

  };

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
    const canToggle = matches.length > previewCount || (previewCount === 0 && matches.length > 0);

    return (

      <section className="mb-8 px-4 sm:px-8">

        <div className="mb-3 flex items-center justify-between">

          <h2 className="text-sm font-medium tracking-wide text-foreground-muted uppercase">{title}</h2>

          {canToggle && (

            <button type="button" onClick={onToggle} className="text-xs font-medium text-foreground-muted hover:text-foreground">

              {expanded ? "Show less" : previewCount === 0 ? `View ${matches.length}` : "View More"}

            </button>

          )}

        </div>

        {visible.length > 0 && (

          <div className="flex flex-col gap-2.5">

            {visible.map((match) => (

              <SportsRow key={match.id} match={match} onSelect={this.props.onSelectChannel} />

            ))}

          </div>

        )}

      </section>

    );

  };

  bucketFor = (match: SportsMatch, nowSecs: number): "live" | "soon" | "upcoming" | "past" => {

    // Prefer authoritative scoreboard lifecycle when ESPN enrichment attached.
    if (match.status === "in" || match.live) return "live";

    if (match.status === "post") return "past";

    const hasScore = match.homeScore !== undefined && match.awayScore !== undefined;
    const delta = match.startsAt - nowSecs;

    // Scored fixtures that are not live are finished (or mislabeled starts) —
    // never park them under Upcoming / Starting Soon.
    if (hasScore) {

      if (delta > 0 && match.status === "pre") return delta <= STARTING_SOON_WINDOW_SECS ? "soon" : "upcoming";

      return "past";

    }

    if (delta <= 0) return "past";

    if (delta <= STARTING_SOON_WINDOW_SECS) return "soon";

    return "upcoming";

  };

  render() {

    const { loading, showAllLive, showAllSoon } = this.state;

    const matches = this.filtered();

    const nowSecs = Date.now() / 1000;

    const live: SportsMatch[] = [];
    const startingSoon: SportsMatch[] = [];
    const upcoming: SportsMatch[] = [];
    const past: SportsMatch[] = [];

    for (const match of matches) {

      switch (this.bucketFor(match, nowSecs)) {

        case "live":
          live.push(match);
          break;
        case "soon":
          startingSoon.push(match);
          break;
        case "upcoming":
          upcoming.push(match);
          break;
        default:
          past.push(match);
          break;

      }

    }

    // Featured: prefer a live game that actually has a score, then any live, then soon.
    const featured =
      live.find((m) => m.homeScore !== undefined && m.awayScore !== undefined)
      ?? live[0]
      ?? startingSoon[0]
      ?? upcoming[0]
      ?? past[0];

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

                    {(featured.live || featured.status === "in") && (

                      <span className="flex flex-shrink-0 items-center gap-1 rounded-full bg-red-500/90 px-2 py-0.5 text-[11px] font-semibold text-white">

                        <span className="h-1.5 w-1.5 rounded-full bg-white" /> LIVE

                      </span>

                    )}

                    <span className="text-xs font-medium uppercase tracking-wide text-white/70">

                      {prettyCategory(featured.category)}

                    </span>

                  </div>

                  <MatchTitle title={featured.title} className="text-2xl font-bold text-white sm:text-3xl" />

                  {(featured.homeScore !== undefined && featured.awayScore !== undefined) && (

                    <div className="flex items-baseline gap-3">

                      <span className="text-3xl font-bold tabular-nums text-white sm:text-4xl">

                        {featured.homeScore}
                        <span className="mx-2 font-normal text-white/40">–</span>
                        {featured.awayScore}

                      </span>

                      {featured.statusDetail && (

                        <span className="text-sm text-white/60">

                          {featured.statusDetail}

                        </span>

                      )}

                    </div>

                  )}

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

        {this.renderSection("Upcoming", upcoming.slice(0, UPCOMING_CAP), true, () => {}, UPCOMING_CAP)}

        {this.renderSection("Past", past, true, () => {}, Number.MAX_SAFE_INTEGER)}

      </div>

    );

  }

}
