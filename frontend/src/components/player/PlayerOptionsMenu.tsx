import { cn } from "@/lib/utils";
import type { LiveChannel, StreamQuality, SubtitleTrack } from "@/lib/types";

import { Component, createRef, type ReactNode } from "react";
import { Check, Gauge, LayoutGrid, Loader2, Search, Settings, Settings2, Subtitles, X } from "lucide-react";

type OptionsPanel = "quality" | "subtitles" | "multiview";

export const MULTIVIEW_MAX_STREAMS = 6;

interface PlayerOptionsMenuProps {

  open: boolean;

  qualities: StreamQuality[];
  selectedHeight: number;

  subtitleTracks: SubtitleTrack[];
  activeSubtitleId: string | null;

  qualityEnabled: boolean;
  preferredHeight?: number;
  hdrHeights?: Set<number>;

  multiviewEnabled?: boolean;
  multiviewChannels?: LiveChannel[];
  multiviewSelectedIds?: string[];
  multiviewPendingIds?: string[];
  multiviewPrimaryId?: string;
  multiviewLoading?: boolean;

  onToggle: () => void;
  onClose: () => void;
  onOutsideClose?: (event: PointerEvent) => void;

  onQualityChange: (height: number) => void;
  onSubtitleChange: (trackId: string | null) => void;
  onOpenSettings?: () => void;

  onMultiviewSearch?: (query: string) => void;
  onMultiviewToggle?: (channel: LiveChannel) => void;

}

interface PlayerOptionsMenuState {

  panel: OptionsPanel;
  multiviewQuery: string;

}

const qualityLabel = (height: number) => {

  if (height >= 2160) return "4K";
  if (height >= 1080) return "1080p";
  if (height >= 720) return "720p";

  return `${height}p`;

};

const qualityDetailLabel = (quality: StreamQuality) => {

  const label = quality.label?.trim();

  if (!label || label.toLowerCase() === qualityLabel(quality.height).toLowerCase()) return "";

  return label;

};

const subtitleTrackDetail = (track: SubtitleTrack) => {

  switch (track.source) {

    case "subdl":

      return "Matched online";

    case "febbox":

      return "Bundled with file";

    case "hls":

      return "Embedded in stream";

    default:

      return "External track";

  }

};

export class PlayerOptionsMenu extends Component<PlayerOptionsMenuProps, PlayerOptionsMenuState> {

  state: PlayerOptionsMenuState = { panel: "quality", multiviewQuery: "" };

  private rootRef = createRef<HTMLDivElement>();
  private searchTimer: ReturnType<typeof setTimeout> | null = null;

  panelOrder = (): OptionsPanel[] => {

    const order: OptionsPanel[] = [];

    if (this.props.qualityEnabled) order.push("quality");

    order.push("subtitles");

    if (this.props.multiviewEnabled) order.push("multiview");

    return order;

  };

  defaultPanel = (): OptionsPanel => this.panelOrder()[0] ?? "subtitles";

  componentDidUpdate(prev: PlayerOptionsMenuProps) {

    if (this.props.open && !prev.open) {

      this.setState({ panel: this.defaultPanel(), multiviewQuery: "" });

      if (this.props.multiviewEnabled) {

        this.props.onMultiviewSearch?.("");

      }

    }

    if (this.props.open !== prev.open) {

      if (this.props.open) {

        document.addEventListener("pointerdown", this.handleOutsidePointerDown, true);

      } else {

        document.removeEventListener("pointerdown", this.handleOutsidePointerDown, true);

      }

    }

  }

  componentWillUnmount() {

    document.removeEventListener("pointerdown", this.handleOutsidePointerDown, true);

    if (this.searchTimer) clearTimeout(this.searchTimer);

  }

  handleOutsidePointerDown = (event: PointerEvent) => {

    const root = this.rootRef.current;

    if (!root || root.contains(event.target as Node)) return;

    event.preventDefault();
    event.stopPropagation();
    event.stopImmediatePropagation();

    this.props.onOutsideClose?.(event);
    this.props.onClose();

  };

  handleMultiviewQuery = (value: string) => {

    this.setState({ multiviewQuery: value });

    if (this.searchTimer) clearTimeout(this.searchTimer);

    this.searchTimer = setTimeout(() => {

      this.props.onMultiviewSearch?.(value);

    }, 200);

  };

  renderTab = (panel: OptionsPanel, label: string, icon?: ReactNode) => {

    const active = this.state.panel === panel;

    return (

      <button onClick={() => this.setState({ panel })} className={cn(

          "relative flex flex-1 items-center justify-center gap-1.5 rounded-md px-3 py-2 text-xs font-medium transition-colors sm:text-sm",
          active ? "text-surface" : "text-foreground-muted hover:text-foreground"

        )} >

        {active && <span className="absolute inset-0 rounded-md bg-foreground shadow-sm" />}

        <span className="relative z-10 inline-flex items-center gap-1.5">

          {icon}
          {label}

        </span>

      </button>

    );

  };

  renderOption = (
    active: boolean,
    label: string,
    detail: string | undefined,
    onClick: () => void,
    key: string,
    disabled = false,
    loading = false,
  ) => (

    <button key={key} disabled={disabled} onClick={(e) => {

        e.stopPropagation();

        if (disabled) return;

        onClick();

      }} className={cn(

        "flex w-full items-center gap-3 rounded-lg px-3.5 py-3 text-left transition-colors",
        disabled && "cursor-not-allowed opacity-40",
        !disabled && (active || loading) && "bg-white/10 text-foreground ring-1 ring-white/10",
        !disabled && !active && !loading && "text-foreground-muted hover:bg-white/6 hover:text-foreground"

      )} >

      {loading ? (

        <Loader2 size={16} className="h-4 w-4 shrink-0 animate-spin text-foreground-muted" strokeWidth={2.5} />

      ) : (

        <span className={cn(

            "flex h-4 w-4 shrink-0 items-center justify-center rounded-full border transition-colors",
            active ? "border-accent bg-accent text-black" : "border-white/20 bg-transparent"

          )}

        >

          {active && <Check size={10} strokeWidth={3} />}

        </span>

      )}

      <div className="min-w-0 flex-1">

        <p className="truncate text-sm font-medium leading-tight">

          {label}

        </p>

        {detail && (

          <p className="mt-0.5 truncate text-[11px] leading-tight text-foreground-faint">

            {detail}

          </p>

        )}

      </div>

    </button>

  );

  renderMultiviewPanel = () => {

    const {
      multiviewChannels = [],
      multiviewSelectedIds = [],
      multiviewPendingIds = [],
      multiviewPrimaryId,
      multiviewLoading,
      onMultiviewToggle,
    } = this.props;

    const { multiviewQuery } = this.state;

    const selectedSet = new Set(multiviewSelectedIds);
    const pendingSet = new Set(multiviewPendingIds);
    const selectedCount = multiviewSelectedIds.length;
    const atCap = selectedCount >= MULTIVIEW_MAX_STREAMS;

    return (

      <div className="max-h-72 w-full flex-shrink-0 overflow-hidden pr-1">

        <div className="mb-2 flex items-center gap-2 rounded-lg border border-white/10 bg-white/5 px-2.5 py-2">

          <Search size={13} className="shrink-0 text-foreground-faint" />

          <input

            type="search"
            value={multiviewQuery}
            placeholder="Search channels…"
            onChange={(e) => this.handleMultiviewQuery(e.target.value)}
            onClick={(e) => e.stopPropagation()}
            className="min-w-0 flex-1 bg-transparent text-sm text-foreground outline-none placeholder:text-foreground-faint"

          />

          {multiviewQuery && (

            <button

              type="button"
              onClick={(e) => {

                e.stopPropagation();
                this.handleMultiviewQuery("");

              }}
              className="rounded p-0.5 text-foreground-faint hover:text-foreground"

            >

              <X size={12} />

            </button>

          )}

        </div>

        <p className="mb-2 px-1 text-[11px] text-foreground-faint">

          {selectedCount}/{MULTIVIEW_MAX_STREAMS} streams selected

        </p>

        <div className="max-h-48 space-y-1 overflow-y-auto">

          {multiviewLoading && multiviewChannels.length === 0 && (

            <div className="rounded-lg px-4 py-6 text-center text-sm text-foreground-muted">

              Loading channels…

            </div>

          )}

          {!multiviewLoading && multiviewChannels.length === 0 && (

            <div className="rounded-lg px-4 py-6 text-center">

              <LayoutGrid size={20} className="mx-auto mb-2 text-foreground-faint" />

              <p className="text-sm text-foreground-muted">

                No channels found

              </p>

            </div>

          )}

          {multiviewChannels.map((channel) => {

            const isPrimary = channel.id === multiviewPrimaryId;
            const pending = pendingSet.has(channel.id);
            const selected = selectedSet.has(channel.id) || isPrimary;

            const detail = isPrimary
              ? "Current channel"
              : selected
                ? "In multiview — click to remove"
                : atCap
                  ? "Limit reached"
                  : channel.category || undefined;

            return this.renderOption(

              selected && !pending,
              channel.name,
              detail,
              () => {

                if (isPrimary) return;

                onMultiviewToggle?.(channel);

              },
              `mv-${channel.id}`,
              // Allow deselecting / cancelling even at the cap; only block new adds.
              isPrimary || (!selected && !pending && atCap),
              pending

            );

          })}

        </div>

      </div>

    );

  };

  render() {

    const { open, qualities, selectedHeight, subtitleTracks, activeSubtitleId, qualityEnabled, preferredHeight, hdrHeights, multiviewEnabled, onToggle, onClose, onQualityChange, onSubtitleChange, onOpenSettings, } = this.props;
    const { panel } = this.state;

    const sortedQualities = [...qualities].sort((a, b) => b.height - a.height);

    const panelOrder = this.panelOrder();

    const panelIndex = Math.max(0, panelOrder.indexOf(panel));

    return (

      <div ref={this.rootRef} className="relative">

        <button onClick={(e) => {

            e.stopPropagation();

            onToggle();

          }} className={cn(

            "rounded-md p-1.5 text-foreground transition-colors hover:bg-white/10",
            open && "bg-white/10"

          )} aria-label="Playback options" >

          <Settings2 size={18} />

        </button>

        <div className={cn(

            "absolute right-0 bottom-full z-40 mb-3 w-80 origin-bottom-right overflow-hidden rounded-xl border border-white/10 bg-surface/75 shadow-2xl shadow-black/40 backdrop-blur-xl backdrop-saturate-150 transition-[opacity,transform,filter] duration-200 ease-out",
            open ? "pointer-events-auto translate-y-0 scale-100 opacity-100 blur-0" : "pointer-events-none translate-y-2 scale-95 opacity-0 blur-sm"

          )}

            onClick={(e) => e.stopPropagation()}

          >

            <div className="flex items-center justify-between px-4 py-3.5">

              <div className="flex items-center gap-2">

                <Settings2 size={14} className="text-foreground-muted" />

                <p className="text-sm font-medium text-foreground">

                  Playback

                </p>

              </div>

              <button onClick={(e) => {

                  e.stopPropagation();

                  onClose();

                }} className="rounded-md p-1.5 text-foreground-muted transition-colors hover:bg-white/8 hover:text-foreground" aria-label="Close options" >

                <X size={14} />

              </button>

            </div>

            <div className="mx-4 mb-3 rounded-lg border border-white/10 bg-white/5 p-1">

              <div className="flex gap-1">

                {qualityEnabled && this.renderTab("quality", "Quality", <Gauge size={13} className="opacity-80" />)}

                {this.renderTab("subtitles", "Subtitles", <Subtitles size={13} className="opacity-80" />)}

                {multiviewEnabled && this.renderTab("multiview", "Multiview", <LayoutGrid size={13} className="opacity-80" />)}

              </div>

            </div>

            <div className="max-h-72 overflow-hidden px-3 pb-3">

              <div className="flex w-full"

                style={{

                  transform: `translateX(-${panelIndex * 100}%)`,
                  transition: "transform 320ms cubic-bezier(0.22, 1, 0.36, 1)",

                }}

              >
                {qualityEnabled && (

                  <div className="max-h-72 w-full flex-shrink-0 overflow-y-auto pr-1">

                    <div className="space-y-1">

                      {sortedQualities.map((quality) => {

                        const isPreferred = preferredHeight === quality.height;
                        const isHdr = hdrHeights?.has(quality.height) ?? false;

                        const parts: string[] = [];
                        const label = qualityDetailLabel(quality);

                        if (label) parts.push(label);
                        if (isHdr) parts.push("HDR");
                        if (isPreferred) parts.push("Preferred");

                        const detail = parts.join(" · ") || undefined;

                        return this.renderOption(

                          selectedHeight === quality.height,

                          qualityLabel(quality.height),
                          detail,

                          () => onQualityChange(quality.height),
                          `quality-${quality.height}`

                        );

                      })}

                    </div>

                    {onOpenSettings && (

                      <button className="mt-2 mb-4 flex w-full items-center gap-2 rounded-lg border border-white/8 px-3.5 py-2.5 text-left text-xs text-foreground-muted transition-colors hover:bg-white/6 hover:text-foreground" onClick={(e) => {

                          e.stopPropagation();

                          onClose();
                          onOpenSettings();

                        }}>

                        <Settings size={13} className="shrink-0 opacity-80" />
                        <span>Change Quality Settings</span>

                      </button>

                    )}

                  </div>

                )}

                <div className="max-h-72 w-full flex-shrink-0 overflow-y-auto pr-1">

                  <div className="space-y-1">

                    {this.renderOption(

                      activeSubtitleId === null,

                      "Off",
                      "No subtitles",

                      () => onSubtitleChange(null),

                      "subtitle-off"

                    )}

                    {subtitleTracks.map((track) =>

                      this.renderOption(

                        activeSubtitleId === track.id,

                        track.label,

                        subtitleTrackDetail(track),
                        () => onSubtitleChange(track.id),

                        track.id

                      )

                    )}

                    {subtitleTracks.length === 0 && (

                      <div className="rounded-lg px-4 py-6 text-center">

                        <Subtitles size={20} className="mx-auto mb-2 text-foreground-faint" />

                        <p className="text-sm text-foreground-muted">

                          No subtitles available

                        </p>

                        <p className="mt-1 text-xs text-foreground-faint">

                          Subtitles may not be available for this stream, or there may have been an error loading them.

                        </p>

                      </div>

                    )}

                  </div>

                </div>

                {multiviewEnabled && this.renderMultiviewPanel()}

              </div>

            </div>

        </div>

      </div>

    );

  }

}
