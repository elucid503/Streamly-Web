import { cn } from "@/lib/utils";
import type { LiveChannel, MatchedChannel, SportsMatch } from "@/lib/types";

import { Bell, ChevronRight } from "lucide-react";

interface SportsRowProps {

  match: SportsMatch;
  onSelect: (channel: LiveChannel) => void;

  alertable?: boolean;
  subscribed?: boolean;
  alerting?: boolean;
  onToggleAlert?: (match: SportsMatch) => void;

}

export function matchedChannelToLiveChannel(channel: MatchedChannel, category = ""): LiveChannel {

  return { id: channel.id, name: channel.name, slug: "", code: "", logo: channel.logo, country: "", category };

}

export function splitStart(unixSecs: number): { day: string | null; time: string } {

  const date = new Date(unixSecs * 1000);
  const now = new Date();

  const isToday = date.toDateString() === now.toDateString();
  const time = date.toLocaleTimeString([], { hour: "numeric", minute: "2-digit" });

  if (isToday) return { day: null, time };

  return { day: date.toLocaleDateString([], { weekday: "short", month: "short", day: "numeric" }), time };

}

export function prettyCategory(category: string): string {

  const overrides: Record<string, string> = {

    afl: "AFL",
    mma: "MMA",
    ufc: "UFC",
    football: "Football",
    soccer: "Football",
    "american-football": "American Football",
    "motor-sports": "Formula 1",

  };

  if (overrides[category]) return overrides[category];

  return category.replace(/-/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());

}

// MatchTitle renders "Team A vs/at Team B" with the teams in full contrast and
// the separator dimmed, so the two names read clearly at a glance.
export function MatchTitle({ title, className, separatorClassName }: { title: string; className?: string; separatorClassName?: string }) {

  const match = title.match(/^(.*?)\s+(vs\.?|at)\s+(.*)$/i);

  if (!match) {

    return <span className={className}>{title}</span>;

  }

  const separator = /^at$/i.test(match[2]) ? "at" : "vs";

  return (

    <span className={className}>

      {match[1]}

      <span className={cn("mx-1.5 font-normal", separatorClassName ?? "text-foreground-faint")}>{separator}</span>

      {match[3]}

    </span>

  );

}

// Prefer ESPN nicknames ("Red Sox at Pirates") when they actually shorten the title.
export function condensedMatchTitle(match: SportsMatch): string {

  const away = match.awayShortName?.trim();
  const home = match.homeShortName?.trim();

  if (!away || !home) return match.title;

  const connector = /\s+at\s+/i.test(match.title) ? "at" : "vs";
  const condensed = `${away} ${connector} ${home}`;

  if (condensed.length >= match.title.trim().length) return match.title;

  return condensed;

}

function hasScore(match: SportsMatch): boolean {

  return match.homeScore !== undefined && match.awayScore !== undefined;

}

function Scoreline({ match, className }: { match: SportsMatch; className?: string }) {

  if (!hasScore(match)) return null;

  // Prefer away–home when both sides exist (US scoreboard convention).
  const left = match.awayScore ?? match.homeScore;
  const right = match.homeScore ?? match.awayScore;

  return (

    <span className={cn("inline-flex shrink-0 items-center justify-center tabular-nums font-semibold", className)}>

      {match.startsAt > Date.now() / 1000 ? (left || "") : left}
      <span className="mx-1 font-normal text-foreground-faint">–</span>
      {match.startsAt > Date.now() / 1000 ? (right || "") : right}

    </span>

  );

}

export function SportsRow({ match, onSelect, alertable, subscribed, alerting, onToggleAlert }: SportsRowProps) {

  const channel = match.channel ? matchedChannelToLiveChannel(match.channel, match.category) : null;

  const { day, time } = splitStart(match.startsAt);
  const scored = hasScore(match);
  const isLive = match.live || match.status === "in";
  const isFinished = match.status === "post";
  const showKickoff = !isLive && !isFinished && match.startsAt > Date.now() / 1000;

  return (

    <div

      className={cn(

        "flex w-full items-stretch rounded-xl border border-border-subtle bg-surface-raised text-left transition-colors",
        channel ? "hover:border-border hover:bg-surface-overlay" : "opacity-70"

      )}

    >

    <button

      type="button"
      disabled={!channel}
      onClick={() => channel && onSelect(channel)}

      className={cn(

        "flex min-w-0 flex-1 items-center gap-4 px-4 py-3.5 text-left",
        channel ? "cursor-pointer" : "cursor-default"

      )}

    >

      <div className="hidden md:flex w-[70px] flex-shrink-0 flex-col items-center justify-center overflow-hidden border-r border-border-subtle pr-4 text-center">

        {isLive ? (

          <>

            <span className="text-sm font-bold text-red-500">LIVE</span>

            {scored && (

              <Scoreline match={match} className="mt-0.5 text-base text-foreground" />

            )}

          </>

        ) : showKickoff || !(scored || isFinished) ? (

          <>

            {day && <span className="text-xs text-foreground-faint">{day}</span>}

            <span className="text-sm font-semibold text-foreground">{time}</span>

          </>

        ) : (

          <Scoreline match={match} className="text-base text-foreground-muted" />

        )}

      </div>

      <div className="min-w-0 flex-1">

        <div className="mb-1 flex items-center gap-2">

          {(match.live || match.status === "in") && (

            <span className="flex flex-shrink-0 items-center gap-1 rounded-md bg-red-500/15 px-2 py-0.5 text-[11px] font-semibold text-red-500">

              <span className="size-1.5 rounded-full bg-red-500" /> LIVE

            </span>

          )}

          <span className="truncate text-xs font-medium text-foreground-faint">

            {prettyCategory(match.category)}

          </span>

          {showKickoff ? (

            <span className="md:hidden shrink-0 text-xs font-medium tabular-nums text-foreground">

              {day ? `${day} · ${time}` : time}

            </span>

          ) : scored ? (

            <Scoreline match={match} className="md:hidden text-xs text-foreground" />

          ) : null}

        </div>

        <MatchTitle title={match.title} className="block min-w-0 truncate text-base font-semibold text-foreground" />

      </div>

      <div className="flex flex-shrink-0 items-center gap-2">

        {channel ? (

          <span className="hidden md:block text-sm font-medium text-foreground"><span className="text-foreground-faint mr-1">Watch on</span> {channel.name}</span>

        ) : (

          <span className="hidden md:block text-xs text-foreground-faint">Unavailable</span>

        )}

        {channel && <ChevronRight className="size-4 text-foreground-faint" />}

      </div>

    </button>

      {alertable && onToggleAlert && (

        <div className="flex items-center border-l border-border-subtle pr-2 pl-1">

          <button

            type="button"
            disabled={alerting}
            aria-pressed={Boolean(subscribed)}
            aria-label={subscribed ? "Cancel kickoff alert" : "Notify when this starts"}
            title={subscribed ? "Cancel kickoff alert" : "Notify when this starts"}
            onClick={() => onToggleAlert(match)}
            className={cn(

              "flex size-9 items-center justify-center rounded-md transition-colors",
              subscribed ? "text-foreground" : "text-foreground-muted hover:bg-surface-overlay hover:text-foreground",
              alerting && "pointer-events-none opacity-60"

            )}

          >

            <Bell className={cn("size-4", subscribed && "fill-current")} />

          </button>

        </div>

      )}

    </div>

  );

}
