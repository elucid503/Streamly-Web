import { Bell, BellPlus, X } from "lucide-react";
import { Component, Fragment } from "react";

import { FeaturedTeamSelect } from "@/Features/Sports/FeaturedTeamSelect";
import { SportsRow, MatchTitle, condensedMatchTitle, DateSeparator, matchedChannelToLiveChannel, prettyCategory, splitStart, startDayKey, startDayLabel } from "@/Features/Sports/SportsCard";
import { Button } from "@/UI/Button";
import { HScrollRow } from "@/UI/HScrollRow";
import { TeamChipRow, TeamFollowModal, matchHasTeam, teamOptionsFromMatches, teamsForMatch } from "@/Features/Sports/TeamFollow";

import { sportsAPI } from "@/Features/Sports/Api";
import { hintCopy, subscribeToMatch, subscribeToTeam, unsubscribeFromMatch, unsubscribeFromTeam, type AlertHint } from "@/Features/Sports/Alerts";
import { sportsBackgroundImage } from "@/Features/Sports/Backgrounds";
import type { LiveChannel } from "@/Features/Live/Types";
import type { SportsAlert, SportsMatch, SportsTeamAlert } from "@/Features/Sports/Types";
import { cn } from "@/Utils/ClassNames";

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
  teamAlerts: SportsTeamAlert[];
  alertingId: string | null;
  alertingTeam: string | null;
  pushHint: AlertHint | null;
  teamPickerOpen: boolean;

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
    teamAlerts: [],
    alertingId: null,
    alertingTeam: null,
    pushHint: null,
    teamPickerOpen: false,

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

      const matches = matchesOrEmpty(await sportsAPI.matches());

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

      const alerts = await sportsAPI.listAlerts();

      this.setState({ alerts: alerts?.matches ?? [], teamAlerts: alerts?.teams ?? [] });

    } catch {

      /* alerts are optional when push is unconfigured */

    }

  };

  isCoveredByTeam = (match: SportsMatch) => {

    return this.state.teamAlerts.some((alert) => matchHasTeam(match, alert.team));

  };

  isMatchAlerted = (matchId: string) => {

    return this.state.alerts.some((alert) => alert.matchId === matchId);

  };

  isSubscribed = (match: SportsMatch) => {

    return this.isMatchAlerted(match.id) || this.isCoveredByTeam(match);

  };

  handleToggleAlert = async (match: SportsMatch) => {

    if (this.state.alertingId) return;

    const subscribed = this.isSubscribed(match);
    const coveredByTeam = this.isCoveredByTeam(match);

    if (subscribed && coveredByTeam && !this.isMatchAlerted(match.id)) return;

    this.setState({ alertingId: match.id, pushHint: null });

    try {

      if (this.isMatchAlerted(match.id)) {

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

  handleToggleTeam = async (team: string, following: boolean) => {

    if (this.state.alertingTeam) return;

    this.setState({ alertingTeam: team, pushHint: null });

    try {

      if (following) {

        await unsubscribeFromTeam(team);
        await this.loadAlerts();

        return;

      }

      const result = await subscribeToTeam(team);

      if (!result.ok) {

        this.setState({ pushHint: result.hint });

        return;

      }

      await this.loadAlerts();

    } catch {

      this.setState({ pushHint: "unavailable" });

    } finally {

      this.setState({ alertingTeam: null });

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

    const { alerts, teamAlerts, pushHint, teamPickerOpen, alertingTeam, matches } = this.state;

    const byId = new Map([...startingSoon, ...upcoming].map((match) => [match.id, match]));
    const teamOptions = teamOptionsFromMatches(matches);

    return (

      <section className="mb-6 px-4 sm:px-8">

        <div className="mb-3 flex items-center justify-between gap-3">

          <div className="min-w-0">

            <h2 className="text-sm font-semibold tracking-tight text-foreground">Your Alerts</h2>

            <p className="text-[11px] text-foreground-faint">Follow a team, or tap the bell on a game.</p>

          </div>

          <Button type="button" variant="secondary" size="sm" className="shrink-0" onClick={() => this.setState({ teamPickerOpen: true })}>

            <BellPlus className="size-3.5" />
            Follow teams

          </Button>

        </div>

        {pushHint && (

          <p className="mb-3 text-xs text-foreground-muted">

            {hintCopy(pushHint)}

          </p>

        )}

        {teamAlerts.length > 0 && (

          <div className="mb-3">

            <TeamChipRow teams={teamAlerts} busyTeam={alertingTeam} onRemove={(team) => void this.handleToggleTeam(team, true)} />

          </div>

        )}

        {alerts.length > 0 && (

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

        )}

        <TeamFollowModal
          open={teamPickerOpen}
          options={teamOptions}
          followed={teamAlerts.map((alert) => alert.team)}
          busyTeam={alertingTeam}
          onClose={() => this.setState({ teamPickerOpen: false })}
          onToggle={(team, following) => void this.handleToggleTeam(team, following)}
        />

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

            {visible.map((match, index) => {

              const prev = index > 0 ? visible[index - 1] : null;
              const showSeparator = Boolean(prev) && startDayKey(match.startsAt) !== startDayKey(prev!.startsAt);

              return (

                <Fragment key={match.id}>

                  {showSeparator && <DateSeparator label={startDayLabel(match.startsAt)} />}

                  <SportsRow
                    match={match}
                    onSelect={this.props.onSelectChannel}
                    alertable={alertable}
                    subscribed={this.isSubscribed(match)}
                    teamFollowed={this.isCoveredByTeam(match)}
                    alerting={this.state.alertingId === match.id}
                    onToggleAlert={this.handleToggleAlert}
                  />

                </Fragment>

              );

            })}

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

                    {featured.delayed && (

                      <span className="flex flex-shrink-0 items-center rounded-md bg-amber-400/90 px-2 py-0.5 text-[11px] font-semibold text-black">

                        Rain Delay

                      </span>

                    )}

                    {!featured.delayed && (featured.live || featured.status === "in") && (

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

                      <Bell className={cn("size-3.5", this.isSubscribed(featured) && "fill-current")} />

                      {this.isSubscribed(featured) ? "Subscribed" : "Notify"}

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
