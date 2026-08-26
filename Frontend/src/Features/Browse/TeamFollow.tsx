import { Button } from "@/UI/Button";
import { CachedImage } from "@/UI/CachedImage";
import { Input } from "@/UI/Input";
import { Modal } from "@/UI/Modal";

import type { SportsMatch, SportsTeamAlert } from "@/Types";
import { cn } from "@/Utils/ClassNames";

import { BellPlus, Check, X } from "lucide-react";
import { Component } from "react";

export interface TeamOption {

  team: string;
  logo?: string;
  league?: string;

}

export function teamsForMatch(match: SportsMatch): string[] {

  const teams: string[] = [];

  if (match.homeTeam) teams.push(match.homeTeam);

  if (match.awayTeam) teams.push(match.awayTeam);

  if (teams.length > 0) return teams;

  const parsed = match.title.match(/^(.*?)\s+(?:vs\.?|at)\s+(.*)$/i);

  if (parsed) return [parsed[1].trim(), parsed[2].trim()].filter(Boolean);

  return match.title.trim() ? [match.title.trim()] : [];

}

export function matchHasTeam(match: SportsMatch, team: string): boolean {

  const needle = team.trim().toLowerCase();

  if (!needle) return false;

  const names = [

    ...teamsForMatch(match),
    match.homeShortName ?? "",
    match.awayShortName ?? "",

  ];

  return names.some((name) => name.trim().toLowerCase() === needle);

}

export function teamOptionsFromMatches(matches: SportsMatch[]): TeamOption[] {

  const seen = new Map<string, TeamOption>();

  for (const match of matches) {

    const add = (name?: string, logo?: string) => {

      const team = name?.trim();

      if (!team) return;

      const key = team.toLowerCase();

      if (seen.has(key)) return;

      seen.set(key, { team, logo, league: match.league });

    };

    add(match.homeTeam, match.homeLogo);
    add(match.awayTeam, match.awayLogo);

    if (!match.homeTeam && !match.awayTeam) {

      for (const team of teamsForMatch(match)) add(team);

    }

  }

  return [...seen.values()].sort((a, b) => a.team.localeCompare(b.team));

}

function TeamMark({ team, logo, size = "md" }: { team: string; logo?: string; size?: "sm" | "md" }) {

  const initial = team.trim().charAt(0).toUpperCase() || "•";
  const dim = size === "sm" ? "size-7" : "size-9";

  return (

    <div className={cn("relative shrink-0 overflow-hidden rounded-full bg-surface-overlay ring-1 ring-border-subtle", dim)}>

      {logo ? (

        <CachedImage src={logo} alt="" rounded="rounded-full" className="size-full" imgClassName="size-full object-contain p-0.5" fallback={

          <span className="flex size-full items-center justify-center text-[11px] font-semibold text-foreground-muted">{initial}</span>

        } />

      ) : (

        <span className="flex size-full items-center justify-center text-[11px] font-semibold text-foreground-muted">{initial}</span>

      )}

    </div>

  );

}

interface TeamChipRowProps {

  teams: SportsTeamAlert[];
  busyTeam: string | null;
  onRemove: (team: string) => void;

}

export function TeamChipRow({ teams, busyTeam, onRemove }: TeamChipRowProps) {

  if (teams.length === 0) return null;

  return (

    <div className="flex flex-wrap gap-2">

      {teams.map((item) => (

        <div
          key={item.team}
          className="flex items-center gap-2 rounded-full border border-border-subtle bg-surface-raised py-1 pl-1 pr-1.5 shadow-sm"
        >

          <TeamMark team={item.team} logo={item.logo} size="sm" />

          <span className="max-w-[10rem] truncate text-xs font-medium text-foreground">{item.team}</span>

          <button
            type="button"
            aria-label={`Stop following ${item.team}`}
            disabled={busyTeam === item.team}
            onClick={() => onRemove(item.team)}
            className="flex size-6 items-center justify-center rounded-full text-foreground-muted transition-colors hover:bg-surface-overlay hover:text-foreground disabled:opacity-50"
          >

            <X className="size-3" />

          </button>

        </div>

      ))}

    </div>

  );

}

interface TeamFollowModalProps {

  open: boolean;
  options: TeamOption[];
  followed: string[];
  busyTeam: string | null;
  onClose: () => void;
  onToggle: (team: string, following: boolean) => void;

}

interface TeamFollowModalState {

  query: string;

}

export class TeamFollowModal extends Component<TeamFollowModalProps, TeamFollowModalState> {

  state: TeamFollowModalState = { query: "" };

  componentDidUpdate(prev: TeamFollowModalProps) {

    if (this.props.open && !prev.open) this.setState({ query: "" });

  }

  render() {

    const { open, options, followed, busyTeam, onClose, onToggle } = this.props;
    const query = this.state.query.trim().toLowerCase();
    const followedKeys = new Set(followed.map((team) => team.toLowerCase()));

    const visible = query
      ? options.filter((option) => option.team.toLowerCase().includes(query) || (option.league ?? "").toLowerCase().includes(query))
      : options;

    return (

      <Modal open={open} onClose={onClose} title="Follow teams" className="max-w-lg">

        <p className="mb-4 text-sm text-foreground-muted">

          Get an alert for every game these teams play.

        </p>

        <Input
          value={this.state.query}
          onChange={(event) => this.setState({ query: event.target.value })}
          placeholder="Search teams"
          aria-label="Search teams"
          className="mb-4"
        />

        <div className="flex max-h-[22rem] flex-col gap-1 overflow-y-auto pr-0.5">

          {visible.length === 0 && (

            <p className="px-1 py-8 text-center text-sm text-foreground-faint">

              {options.length === 0 ? "No teams on the board right now." : "No matching teams."}

            </p>

          )}

          {visible.map((option) => {

            const following = followedKeys.has(option.team.toLowerCase());
            const busy = busyTeam === option.team;

            return (

              <div
                key={option.team}
                className="flex items-center gap-3 rounded-lg px-2 py-2 hover:bg-surface-overlay/70"
              >

                <TeamMark team={option.team} logo={option.logo} />

                <div className="min-w-0 flex-1">

                  <p className="truncate text-sm font-medium text-foreground">{option.team}</p>

                  {option.league && (

                    <p className="truncate text-[11px] text-foreground-faint">{option.league}</p>

                  )}

                </div>

                <Button
                  type="button"
                  size="sm"
                  variant={following ? "secondary" : "outline"}
                  disabled={busy}
                  onClick={() => onToggle(option.team, following)}
                  className={cn(following && "text-foreground")}
                >

                  {following ? <Check className="size-3.5" /> : <BellPlus className="size-3.5" />}
                  {following ? "Following" : "Follow"}

                </Button>

              </div>

            );

          })}

        </div>

      </Modal>

    );

  }

}
