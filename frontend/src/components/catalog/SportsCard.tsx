import { cn } from "@/lib/utils";
import type { LiveChannel, MatchedChannel, SportsMatch } from "@/lib/types";

import { ChevronRight } from "lucide-react";

interface SportsRowProps {

  match: SportsMatch;
  onSelect: (channel: LiveChannel) => void;

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
    "american-football": "American Football",
    "motor-sports": "Formula 1",

  };

  if (overrides[category]) return overrides[category];

  return category.replace(/-/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());

}

// MatchTitle renders "Team A vs Team B" with the teams in full contrast and
// the "vs" separator dimmed, so the two names read clearly at a glance.
export function MatchTitle({ title, className }: { title: string; className?: string }) {

  const match = title.match(/^(.*?)\s+vs\.?\s+(.*)$/i);

  if (!match) {

    return <span className={className}>{title}</span>;

  }

  return (

    <span className={className}>

      {match[1]}

      <span className="mx-1.5 font-normal text-foreground-faint">vs</span>

      {match[2]}

    </span>

  );

}

export function SportsRow({ match, onSelect }: SportsRowProps) {

  const channel = match.channel ? matchedChannelToLiveChannel(match.channel, match.category) : null;
  const { day, time } = splitStart(match.startsAt);

  return (

    <button

      type="button"
      disabled={!channel}
      onClick={() => channel && onSelect(channel)}

      className={cn(

        "flex w-full items-center gap-4 rounded-lg border border-border-subtle bg-surface-raised px-4 py-3 text-left transition-colors",
        channel ? "hover:border-border hover:bg-surface-overlay" : "cursor-default opacity-70"

      )}

    >

      <div className="hidden md:flex w-30 flex-shrink-0 flex-col items-center justify-center whitespace-nowrap border-r border-border-subtle pr-4 text-center">

        {match.live ? (

          <span className="text-sm font-bold text-red-500">LIVE</span>

        ) : (

          <>

            {day && <span className="text-xs text-foreground-faint">{day}</span>}

            <span className="text-lg font-bold text-foreground-muted">{time}</span>

          </>

        )}

      </div>

      <div className="min-w-0 flex-1">

        <div className="mb-1 flex items-center gap-2">

          {match.live && (

            <span className="flex flex-shrink-0 items-center gap-1 rounded-full bg-red-500/15 px-2 py-0.5 text-[11px] font-semibold text-red-500">

              <span className="h-1.5 w-1.5 rounded-full bg-red-500" /> LIVE

            </span>

          )}

          <span className="truncate text-xs font-medium text-foreground-faint">

            {prettyCategory(match.category)}

          </span>

        </div>

        <MatchTitle title={match.title} className="truncate text-base font-semibold text-foreground" />

      </div>

      <div className="flex flex-shrink-0 items-center gap-2">

        {channel ? (

          <span className="hidden md:block text-sm font-medium text-foreground"><span className="text-foreground-faint mr-1">Watch on</span> {channel.name}</span>

        ) : (

          <span className="hidden md:block text-xs text-foreground-faint">Unavailable</span>

        )}

        {channel && <ChevronRight size={18} className="text-foreground-faint" />}

      </div>

    </button>

  );

}
