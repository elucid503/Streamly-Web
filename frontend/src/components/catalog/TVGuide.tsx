import { api } from "@/api/client";

import { ContentRow } from "@/components/catalog/ContentRow";
import { CachedImage } from "@/components/ui/CachedImage";

import type { ChannelGuideEntry, LiveChannel, ProgramEntry } from "@/lib/types";

import { Component } from "react";
import { Radio } from "lucide-react";

interface TVGuideProps {

  onSelect: (channel: LiveChannel) => void;

}

interface TVGuideState {

  entries: ChannelGuideEntry[];
  loading: boolean;

}

export class TVGuide extends Component<TVGuideProps, TVGuideState> {

  state: TVGuideState = {

    entries: [],
    loading: true,

  };

  async componentDidMount() {

    try {

      const raw = await api.liveSchedule() ?? [];

      const sorted = [...raw].sort((a, b) => (a.current ? 0 : 1) - (b.current ? 0 : 1));

      const seen = new Set<string>();

      const entries = sorted.filter((entry) => {

        const title = entry.current?.title ?? entry.next?.title;

        if (!title || seen.has(title)) return false;

        seen.add(title);

        return true;

      });

      this.setState({ entries, loading: false });

    } catch {

      this.setState({ loading: false });

    }

  }

  render() {

    const { onSelect } = this.props;

    const { entries, loading } = this.state;

    if (!loading && entries.length === 0) return null;

    return (

      <ContentRow title="What's On Now" sectionId="live-guide" loading={loading}>

        {entries.map((entry) => (

          <GuideCard key={entry.channel.daddyId} entry={entry} onSelect={onSelect} />

        ))}

      </ContentRow>

    );

  }

}

function formatTime(unixSecs: number): string {

  return new Date(unixSecs * 1000).toLocaleTimeString([], {

    hour: "numeric",
    minute: "2-digit",

  });

}

function progressPct(program: ProgramEntry): number {

  const now = Date.now() / 1000;

  return Math.min(100, Math.max(0, ((now - program.startsAt) / (program.runtime * 60)) * 100));

}

interface GuideCardProps {

  entry: ChannelGuideEntry;
  onSelect: (channel: LiveChannel) => void;

}

function GuideCard({ entry, onSelect }: GuideCardProps) {

  const { channel, current, next } = entry;

  return (

    <button

      type="button"
      className="flex w-[220px] flex-shrink-0 flex-col overflow-hidden rounded-md border border-border-subtle bg-surface-raised text-left transition-colors hover:border-border hover:bg-surface-overlay"
      onClick={() => onSelect(channel)}

    >

      <div className="flex items-center gap-2.5 border-b border-border-subtle px-3 py-2.5">

        <CachedImage

          className="h-7 w-7 flex-shrink-0 bg-surface-overlay"
          src={channel.logo}
          alt={channel.name}
          imgClassName="object-contain p-1"
          rounded="rounded-full"
          fallback={<Radio size={12} className="text-foreground-faint" />}

        />

        <p className="truncate text-xs font-medium text-foreground">{channel.name}</p>

      </div>

      <div className="flex flex-1 flex-col gap-1.5 px-3 py-2.5">

        {current ? (

          <>

            <p className="truncate text-xs font-medium text-foreground">{current.title}</p>

            <div className="flex items-center gap-1.5">

              <div className="h-1 flex-1 overflow-hidden rounded-full bg-border-subtle">

                <div

                  className="h-full rounded-full bg-accent transition-all"
                  style={{ width: `${progressPct(current)}%` }}

                />

              </div>

              <p className="shrink-0 text-[10px] text-foreground-faint">

                {formatTime(current.startsAt + current.runtime * 60)}

              </p>

            </div>

            {next && (

              <p className="truncate text-[10px] text-foreground-faint">

                Next: {next.title} · {formatTime(next.startsAt)}

              </p>

            )}

          </>

        ) : next ? (

          <>

            <p className="text-[10px] text-foreground-faint uppercase tracking-wide">Up next</p>

            <p className="truncate text-xs font-medium text-foreground">{next.title}</p>

            <p className="text-[10px] text-foreground-faint">{formatTime(next.startsAt)}</p>

          </>

        ) : null}

      </div>

    </button>

  );

}
