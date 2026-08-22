import { api } from "@/api/client";

import { SportsRow, MatchTitle, condensedMatchTitle, matchedChannelToLiveChannel, prettyCategory, splitStart } from "@/components/catalog/SportsCard";
import { Button } from "@/components/ui/Button";
import { HScrollRow } from "@/components/ui/HScrollRow";

import { hintCopy, subscribeToMatch, unsubscribeFromMatch, type AlertHint } from "@/lib/sportsAlerts";
import { sportsBackgroundImage } from "@/lib/sportsBackgrounds";
import type { LiveChannel, SportsAlert, SportsMatch } from "@/lib/types";
import { cn } from "@/lib/utils";

import { AnimatePresence, motion } from "framer-motion";
import { Bell, Check, ChevronDown, X } from "lucide-react";
import { Component, createRef } from "react";

const STARTING_SOON_WINDOW_SECS = 3 * 60 * 60;
const ROW_PREVIEW_COUNT = 5;
const FEATURED_TEAM_KEY = "streamly:sportsFeaturedTeam";

function matchesOrEmpty(matches: SportsMatch[] | null | undefined): SportsMatch[] {

  return matches ?? [];

}

interface SportsPageProps {

  onSelectChannel: (channel: LiveChannel) => void;

  searchQuery: string;

}

interface SportsPageState {

  matches: SportsMatch[];
  loading: boolean;

  category: string;
  categories: string[];

  showAllLive: boolean;
  showAllSoon: boolean;
  showAllUpcoming: boolean;

  /** Preferred team name from localStorage; applied only when still in Live Now. */
  featuredTeam: string;

  alerts: SportsAlert[];
  alertingId: string | null;
  pushHint: AlertHint | null;

}

function readFeaturedTeam(): string {

  try {

    return localStorage.getItem(FEATURED_TEAM_KEY) ?? "";

  } catch {

    return "";

  }

}

function writeFeaturedTeam(team: string) {

  try {

    if (!team) localStorage.removeItem(FEATURED_TEAM_KEY);
    else localStorage.setItem(FEATURED_TEAM_KEY, team);

  } catch {

    // ignore quota / private mode

  }

}

/** Team names for a match — prefers structured fields, falls back to "A vs B" title. */
function teamsForMatch(match: SportsMatch): string[] {

  const teams: string[] = [];

  if (match.homeTeam) teams.push(match.homeTeam);

  if (match.awayTeam) teams.push(match.awayTeam);

  if (teams.length > 0) return teams;

  const parsed = match.title.match(/^(.*?)\s+vs\.?\s+(.*)$/i);

  if (parsed) return [parsed[1].trim(), parsed[2].trim()].filter(Boolean);

  return match.title.trim() ? [match.title.trim()] : [];

}

function matchHasTeam(match: SportsMatch, team: string): boolean {

  const needle = team.trim().toLowerCase();

  if (!needle) return false;

  return teamsForMatch(match).some((t) => t.toLowerCase() === needle);

}

/** Unique live teams in Live Now order (home then away per match). */
function liveTeamOptions(live: SportsMatch[]): string[] {

  const seen = new Set<string>();
  const options: string[] = [];

  for (const match of live) {

    for (const team of teamsForMatch(match)) {

      const key = team.toLowerCase();

      if (seen.has(key)) continue;

      seen.add(key);
      options.push(team);

    }

  }

  return options;

}

interface FeaturedTeamSelectProps {

  value: string;
  options: string[];
  onChange: (team: string) => void;

}

interface FeaturedTeamSelectState {

  open: boolean;

}

/** Subtle light-themed select, styled to sit flush with the banner Watch button. */
class FeaturedTeamSelect extends Component<FeaturedTeamSelectProps, FeaturedTeamSelectState> {

  private rootRef = createRef<HTMLDivElement>();

  state: FeaturedTeamSelectState = {

    open: false,

  };

  componentDidMount() {

    document.addEventListener("mousedown", this.handleDocumentMouseDown);

  }

  componentWillUnmount() {

    document.removeEventListener("mousedown", this.handleDocumentMouseDown);

  }

  handleDocumentMouseDown = (event: MouseEvent) => {

    const root = this.rootRef.current;

    if (!root || root.contains(event.target as Node)) return;

    this.setState({ open: false });

  };

  toggleOpen = () => {

    this.setState((s) => ({ open: !s.open }));

  };

  render() {

    const { value, options, onChange } = this.props;
    const { open } = this.state;

    // Only show a team label when that team is currently selectable (in Live Now).
    const activeValue = value && options.some((t) => t.toLowerCase() === value.toLowerCase()) ? value : "";
    const triggerLabel = activeValue || "Auto";

    return (

      <div ref={this.rootRef} className="relative">

        <button type="button" className={cn("hidden field-focus md:flex h-9 min-w-[132px] max-w-[11rem] items-center justify-between gap-2 rounded-md border border-border-subtle bg-surface-raised px-3 text-left text-xs font-medium text-foreground hover:border-border hover:bg-surface-overlay", open && "border-border bg-surface-overlay" )}
          aria-haspopup="listbox"
          aria-expanded={open}
          aria-label="Featured team"
          title="Feature a team in the banner"
          onClick={this.toggleOpen}

        >

          <span className="truncate">{triggerLabel}</span>
          <ChevronDown className={cn("size-3.5 shrink-0 text-foreground-muted transition-transform", open && "rotate-180")} />

        </button>

        <AnimatePresence>

          {open && (

            <motion.div
              className="absolute right-0 top-[calc(100%+8px)] z-50 min-w-[16rem] overflow-hidden rounded-lg border border-border-subtle bg-surface/95 p-1.5 shadow-2xl ring-1 ring-white/[0.04] backdrop-blur-lg"
              initial={{ opacity: 0, scale: 0.96, y: -6 }}
              animate={{ opacity: 1, scale: 1, y: 0 }}
              exit={{ opacity: 0, scale: 0.96, y: -6 }}
              transition={{ type: "spring", stiffness: 500, damping: 32 }}
              style={{ transformOrigin: "top right" }}
            >

              {/* Same surface language as SelectMenu, with roomier rows for long team lists. */}
              <div role="listbox" aria-label="Featured team" className="flex max-h-72 flex-col gap-0.5 overflow-y-auto">

                <button
                  type="button"
                  role="option"
                  aria-selected={!activeValue}
                  className={cn(
                    "flex min-h-10 w-full items-center justify-between gap-2.5 rounded-md px-3 py-2 text-left text-sm font-medium transition-colors",
                    !activeValue
                      ? "bg-surface-raised text-foreground shadow-sm"
                      : "text-foreground-muted hover:bg-surface-overlay/80 hover:text-foreground"
                  )}
                  onClick={() => {

                    onChange("");
                    this.setState({ open: false });

                  }}
                >

                  <span className="truncate">Auto</span>

                  {!activeValue && (

                    <span className="flex size-5 shrink-0 items-center justify-center rounded-full bg-foreground/10">

                      <Check className="size-2.5 text-foreground" strokeWidth={2.5} />

                    </span>

                  )}

                </button>

                {options.map((team) => {

                  const isSelected = activeValue.toLowerCase() === team.toLowerCase();

                  return (

                    <button
                      key={team}
                      type="button"
                      role="option"
                      aria-selected={isSelected}
                      className={cn(
                        "flex min-h-10 w-full items-center justify-between gap-2.5 rounded-md px-3 py-2 text-left text-sm font-medium transition-colors",
                        isSelected
                          ? "bg-surface-raised text-foreground shadow-sm"
                          : "text-foreground-muted hover:bg-surface-overlay/80 hover:text-foreground"
                      )}
                      onClick={() => {

                        onChange(team);
                        this.setState({ open: false });

                      }}
                    >

                      <span className="truncate leading-snug">{team}</span>

                      {isSelected && (

                        <span className="flex size-5 shrink-0 items-center justify-center rounded-full bg-foreground/10">

                          <Check className="size-2.5 text-foreground" strokeWidth={2.5} />

                        </span>

                      )}

                    </button>

                  );

                })}

              </div>

            </motion.div>

          )}

        </AnimatePresence>

      </div>

    );

  }

}

export class SportsPage extends Component<SportsPageProps, SportsPageState> {

  private refreshTimer: ReturnType<typeof setInterval> | null = null;

  state: SportsPageState = {

    matches: [],
    loading: true,

    category: "all",
    categories: [],

    showAllLive: false,
    showAllSoon: false,
    showAllUpcoming: false,

    featuredTeam: readFeaturedTeam(),

    alerts: [],
    alertingId: null,
    pushHint: null,

  };

  async componentDidMount() {

    await this.loadMatches(true);
    void this.loadAlerts();

    // Scores and live flags move quickly; poll while the page is mounted.
    this.refreshTimer = setInterval(() => {

      void this.loadMatches(false);
      void this.loadAlerts();

    }, 60_000);

  }

  componentWillUnmount() {

    if (this.refreshTimer) clearInterval(this.refreshTimer);

  }

  loadMatches = async (initial: boolean) => {

    try {

      const matches = matchesOrEmpty(await api.liveSports());

      const categories = Array.from(new Set([

        "motor-sports",
        ...matches.map((m) => m.category).filter(Boolean),

      ])).sort();

      this.setState({ matches, loading: false, categories });

    } catch {

      if (initial) this.setState({ loading: false });

    }

  };

  loadAlerts = async () => {

    try {

      const alerts = await api.listSportsAlerts();

      this.setState({ alerts: alerts ?? [] });

    } catch {

      /* alerts are optional when push is unconfigured */

    }

  };

  isSubscribed = (matchId: string) => {

    return this.state.alerts.some((alert) => alert.matchId === matchId);

  };

  handleToggleAlert = async (match: SportsMatch) => {

    if (this.state.alertingId) return;

    this.setState({ alertingId: match.id, pushHint: null });

    const subscribed = this.isSubscribed(match.id);

    try {

      if (subscribed) {

        await unsubscribeFromMatch(match.id);

        this.setState({ alerts: this.state.alerts.filter((alert) => alert.matchId !== match.id) });

        return;

      }

      const result = await subscribeToMatch(match.id);

      if (!result.ok) {

        this.setState({ pushHint: result.hint });

        return;

      }

      this.setState({ alerts: [{ matchId: match.id, title: match.title }, ...this.state.alerts.filter((alert) => alert.matchId !== match.id)] });

    } catch {

      this.setState({ pushHint: "unavailable" });

    } finally {

      this.setState({ alertingId: null });

    }

  };

  setFeaturedTeam = (team: string) => {

    writeFeaturedTeam(team);
    this.setState({ featuredTeam: team });

  };

  filtered = () => {

    const { searchQuery } = this.props;
    const { matches, category } = this.state;

    const query = searchQuery.trim().toLowerCase();

    return matches.filter((m) => {

      if (category && category !== "all" && m.category !== category) return false;

      if (query && !m.title.toLowerCase().includes(query)) return false;

      return true;

    });

  };

  renderAlerts = (startingSoon: SportsMatch[], upcoming: SportsMatch[]) => {

    const { alerts, pushHint } = this.state;

    if (alerts.length === 0 && !pushHint) return null;

    const byId = new Map([...startingSoon, ...upcoming].map((match) => [match.id, match]));

    return (

      <section className="mb-6 px-4 sm:px-8">

        {pushHint && (

          <p className="mb-3 text-xs text-foreground-muted">

            {hintCopy(pushHint)}

          </p>

        )}

        {alerts.length > 0 && (

          <>

            <h2 className="mb-3 text-sm font-semibold tracking-tight text-foreground">Your Alerts</h2>

            <div className="flex flex-col gap-2">

              {alerts.map((alert) => {

                const match = byId.get(alert.matchId);
                const when = match ? splitStart(match.startsAt) : null;

                return (

                  <div key={alert.matchId} className="flex items-center gap-3 rounded-lg border border-border-subtle bg-surface-raised px-3 py-2">

                    <Bell className="size-3.5 shrink-0 fill-current text-foreground-muted" />

                    <div className="min-w-0 flex-1">

                      <p className="truncate text-sm font-medium text-foreground">{match?.title ?? alert.title}</p>

                      {when && (

                        <p className="text-[11px] text-foreground-faint">

                          {when.day ? `${when.day} · ` : ""}{when.time}

                        </p>

                      )}

                    </div>

                    <button

                      type="button"
                      aria-label="Cancel kickoff alert"
                      disabled={this.state.alertingId === alert.matchId}
                      onClick={() => this.handleToggleAlert(match ?? { id: alert.matchId, title: alert.title, category: "", startsAt: 0, live: false })}
                      className="flex size-8 items-center justify-center rounded-md text-foreground-muted hover:bg-surface-overlay hover:text-foreground"

                    >

                      <X className="size-3.5" />

                    </button>

                  </div>

                );

              })}

            </div>

          </>

        )}

      </section>

    );

  };

  renderFilters = () => {

    const { category, categories } = this.state;

    if (categories.length === 0) return null;

    return (

      <div className="mb-6 px-4 sm:px-8">

        <HScrollRow className="items-center gap-1.5">

          <Button
            variant={category === "all" ? "default" : "secondary"}
            size="sm"
            className="shrink-0"
            onClick={() => this.setState({ category: "all" })}
          >

            All

          </Button>

          {categories.map((value) => (

            <Button
              key={value}
              variant={category === value ? "default" : "secondary"}
              size="sm"
              className="shrink-0"
              onClick={() => this.setState({ category: category === value ? "all" : value })}
            >

              {prettyCategory(value)}

            </Button>

          ))}

        </HScrollRow>

      </div>

    );

  };

  renderSection = (title: string, matches: SportsMatch[], expanded: boolean, onToggle: () => void, previewCount = ROW_PREVIEW_COUNT, alertable = false) => {

    if (matches.length === 0) return null;

    const visible = expanded ? matches : matches.slice(0, previewCount);
    const canToggle = matches.length > previewCount || (previewCount === 0 && matches.length > 0);

    return (

      <section className="mb-8 flex flex-col gap-3 px-4 sm:px-8">

        <div className="flex items-center justify-between gap-3">

          <h2 className="text-sm font-semibold tracking-tight text-foreground">{title}</h2>

          {canToggle && (

            <button type="button" onClick={onToggle} className="text-xs font-medium text-foreground-muted hover:text-foreground">

              {expanded ? "Show less" : previewCount === 0 ? `View ${matches.length}` : "View More"}

            </button>

          )}

        </div>

        {visible.length > 0 && (

          <div className="flex flex-col gap-3">

            {visible.map((match) => (

              <SportsRow
                key={match.id}
                match={match}
                onSelect={this.props.onSelectChannel}
                alertable={alertable}
                subscribed={this.isSubscribed(match.id)}
                alerting={this.state.alertingId === match.id}
                onToggleAlert={this.handleToggleAlert}
              />

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

  resolveFeatured = (
    live: SportsMatch[],
    startingSoon: SportsMatch[],
    upcoming: SportsMatch[],
    past: SportsMatch[],
  ): SportsMatch | undefined => {

    const { featuredTeam } = this.state;

    // Preferred team only applies while that team is still in Live Now.
    if (featuredTeam) {

      const preferred = live.find((m) => matchHasTeam(m, featuredTeam));

      if (preferred) return preferred;

    }

    return (
      live.find((m) => m.homeScore !== undefined && m.awayScore !== undefined)
      ?? live[0]
      ?? startingSoon[0]
      ?? upcoming[0]
      ?? past[0]
    );

  };

  render() {

    const { loading, showAllLive, showAllSoon, showAllUpcoming, featuredTeam } = this.state;

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

    const featured = this.resolveFeatured(live, startingSoon, upcoming, past);
    const featuredBucket = featured ? this.bucketFor(featured, nowSecs) : null;
    const featuredAlertable = featuredBucket === "soon" || featuredBucket === "upcoming";
    const teamOptions = liveTeamOptions(live);

    if (loading) {

      return (

        <div className="animate-fade-in px-4 py-10 sm:px-8">

          <div className="skeleton mb-8 h-56 w-full rounded-xl" />

          <div className="flex flex-col gap-3">

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

          No sports events available right now

        </div>

      );

    }

    return (

      <div className="animate-fade-in py-8 pt-4">

        {this.renderFilters()}

        {this.renderAlerts(startingSoon, upcoming)}


        {featured && (

          <div className="mb-8 px-4 sm:px-8">

            {/* Overflow only on media layers so the team menu can open outside the card. */}
            <div className="relative rounded-xl border border-border-subtle">

              <div className="pointer-events-none absolute inset-0 overflow-hidden rounded-xl">

                <div

                  className="absolute inset-0 bg-cover bg-center"
                  style={{ backgroundImage: `url(${sportsBackgroundImage(featured.category)})` }}

                />

                <div className="absolute inset-0 bg-black/55" />

                <div className="absolute inset-0 bg-gradient-to-t from-black via-black/85 to-black/40" />

              </div>

              <div className="relative flex min-h-[280px] flex-col justify-end gap-4 p-6 sm:flex-row sm:items-end sm:justify-between sm:p-8">

                <div className="flex min-w-0 flex-col gap-3">

                  <div className="flex items-center gap-2">

                    {(featured.live || featured.status === "in") && (

                      <span className="flex flex-shrink-0 items-center gap-1 rounded-md bg-red-500/90 px-2 py-0.5 text-[11px] font-semibold text-white">

                        <span className="size-1.5 rounded-full bg-white" /> LIVE

                      </span>

                    )}

                    <span className="text-xs font-medium uppercase tracking-wide text-white/70">

                      {prettyCategory(featured.category)}

                    </span>

                  </div>

                  <MatchTitle
                    title={condensedMatchTitle(featured)}
                    className="block min-w-0 truncate text-xl font-bold text-white sm:text-3xl"
                    separatorClassName="text-white/40"
                  />

                  {(featured.homeScore !== undefined && featured.awayScore !== undefined) && (

                    <div className="flex items-baseline gap-3">

                      <span className="text-3xl font-bold tabular-nums text-white sm:text-4xl">

                        {featured.awayScore ?? featured.homeScore}
                        <span className="mx-2 font-normal text-white/40">–</span>
                        {featured.homeScore ?? featured.awayScore}

                      </span>

                      {featured.statusDetail && (

                        <span className="text-sm text-white/60">

                          {featured.statusDetail}

                        </span>

                      )}

                    </div>

                  )}

                </div>

                <div className="flex w-fit flex-shrink-0 items-center gap-1.5">

                  {featured.channel ? (

                    <button type="button" onClick={() => this.props.onSelectChannel(matchedChannelToLiveChannel(featured.channel!, featured.category))} className="flex h-9 items-center gap-2 rounded-md bg-white px-5 text-sm font-semibold text-black text-nowrap transition-opacity hover:opacity-90" >

                      Watch on {featured.channel.name}

                    </button>

                  ) : (

                    <span className="flex h-9 items-center rounded-md bg-white/15 px-5 text-sm font-semibold text-white/70">

                      Scores only

                    </span>

                  )}

                  {featuredAlertable && (

                    <button

                      type="button"
                      disabled={this.state.alertingId === featured.id}
                      onClick={() => this.handleToggleAlert(featured)}
                      className="flex h-9 items-center gap-2 rounded-md border border-white/20 bg-white/10 px-4 text-sm font-semibold text-white text-nowrap transition-opacity hover:bg-white/15"

                    >

                      <Bell className={cn("size-3.5", this.isSubscribed(featured.id) && "fill-current")} />

                      {this.isSubscribed(featured.id) ? "Subscribed" : "Notify"}

                    </button>

                  )}

                  {teamOptions.length > 0 && (

                    <FeaturedTeamSelect
                      value={featuredTeam}
                      options={teamOptions}
                      onChange={this.setFeaturedTeam}
                    />

                  )}

                </div>

              </div>

            </div>

          </div>

        )}

        {this.renderSection("Live Now", live, showAllLive, () => this.setState({ showAllLive: !showAllLive }))}

        {this.renderSection("Starting Soon", startingSoon, showAllSoon, () => this.setState({ showAllSoon: !showAllSoon }), ROW_PREVIEW_COUNT, true)}

        {this.renderSection("Upcoming", upcoming, showAllUpcoming, () => this.setState({ showAllUpcoming: !showAllUpcoming }), ROW_PREVIEW_COUNT, true)}

        {this.renderSection("Past", past, true, () => {}, Number.MAX_SAFE_INTEGER)}

      </div>

    );

  }

}
