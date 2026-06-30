import { ContentRow } from "@/components/catalog/ContentRow";
import { LiveLogo } from "@/components/catalog/LiveLogo";

import type { LiveChannel, SportsChannel, SportsEvent } from "@/lib/types";

import { Radio } from "lucide-react";

const SPORT_KEYWORDS: Record<string, string[]> = {

  NFL: ["nfl", "american football"],
  NBA: ["nba", "basketball"],
  MLB: ["mlb", "baseball"],
  NHL: ["nhl", "ice hockey"],
  Soccer: ["premier league", "la liga", "bundesliga", "serie a", "ligue 1", "champions league", "europa league", "fa cup", "copa", "carabao", "mls", "world cup", "euro 20", "football", "soccer"],
  UFC: ["ufc", "mma", "bellator", "boxing", "fighting"],
  Cricket: ["cricket", "ipl", "bbl", "t20", "odi"],
  F1: ["formula 1", "formula one", "f1", "grand prix"],

};

export function eventMatchesLeague(eventLeague: string, filterLeague: string): boolean {

  if (!eventLeague) return false;

  const lower = eventLeague.toLowerCase();
  const keywords = SPORT_KEYWORDS[filterLeague];

  if (!keywords) return lower.includes(filterLeague.toLowerCase());

  return keywords.some((k) => lower.includes(k));

}

interface SportsSectionProps {

  onSelect: (channel: LiveChannel) => void;
  events: SportsEvent[];
  loading?: boolean;
  leagueFilter?: string;
  title?: string;

}

export function SportsSection({ onSelect, events, loading, leagueFilter, title = "Sports" }: SportsSectionProps) {

  const filtered = (!leagueFilter || leagueFilter === "all")
    ? events
    : events.filter((e) => eventMatchesLeague(e.league, leagueFilter));

  if (!loading && filtered.length === 0) return null;

  return (

    <ContentRow title={title} sectionId="live-sports" loading={loading}>

      {filtered.map((event) => (

        <SportsCard
          key={`${event.league}-${event.title}-${event.time}`}
          event={event}
          onSelect={onSelect}
        />

      ))}

    </ContentRow>

  );

}

export function channelToLive(ch: SportsChannel): LiveChannel {

  return {

    id: ch.daddyId,
    daddyId: ch.daddyId,

    name: ch.name,
    slug: "",

    logo: ch.logo,
    country: "",
    category: "Sports",

    enriched: ch.enriched,

  };

}

function formatStartTime(startsAt: number): string {

  if (!startsAt) return "";

  return new Date(startsAt * 1000).toLocaleTimeString([], {

    hour: "numeric",
    minute: "2-digit",

  });

}

interface SportsCardProps {

  event: SportsEvent;
  onSelect: (channel: LiveChannel) => void;

}

export function SportsCard({ event, onSelect }: SportsCardProps) {

  const { title, league, live: isLive, startsAt, channels } = event;

  const timeLabel = formatStartTime(startsAt);
  const visibleChannels = channels.slice(0, 4);
  const firstChannel = channels[0];

  return (

    <div className="flex w-[240px] flex-shrink-0 flex-col overflow-hidden rounded-md border border-border-subtle bg-surface-raised transition-colors hover:border-border">

      <button
        type="button"
        disabled={!firstChannel}
        onClick={() => firstChannel && onSelect(channelToLive(firstChannel))}
        className="flex flex-col text-left"
      >

        <div className="flex items-center justify-between border-b border-border-subtle px-3 py-2">

          <p className="truncate text-[10px] font-semibold uppercase tracking-wider text-foreground-muted">

            {league || "Sports"}

          </p>

          {isLive && (

            <span className="ml-2 shrink-0 rounded bg-red-500/15 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-red-400">

              Live

            </span>

          )}

        </div>

        <div className="flex flex-col gap-1.5 px-3 py-2.5">

          <p className="line-clamp-2 text-xs font-medium leading-snug text-foreground">{title}</p>

          {timeLabel && (

            <p className="text-[10px] text-foreground-faint">{timeLabel}</p>

          )}

        </div>

      </button>

      <div className="flex items-center gap-1.5 border-t border-border-subtle px-3 py-2">

        {visibleChannels.map((ch) => {

          const liveChannel = channelToLive(ch);

          return (

            <button
              key={ch.daddyId}
              type="button"
              title={ch.name}
              onClick={() => onSelect(liveChannel)}
              className="h-8 w-8 shrink-0 rounded-full border border-border-subtle bg-surface-overlay transition-colors hover:border-border"
            >

              <LiveLogo
                className="h-8 w-8"
                channel={liveChannel}
                imgClassName="object-contain p-1.5"
                rounded="rounded-full"
                fallback={<Radio size={12} className="text-foreground-faint" />}
                lazy={false}
              />

            </button>

          );

        })}

        {channels.length > 4 && (

          <p className="shrink-0 text-[10px] text-foreground-faint">+{channels.length - 4}</p>

        )}

      </div>

    </div>

  );

}
