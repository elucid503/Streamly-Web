import { cn } from "@/lib/utils";
import type { LiveSourceProvider, StreamQuality, SubtitleTrack } from "@/lib/types";

import { Component, createRef } from "react";
import { Check, Loader2, Settings, Settings2, Subtitles, X } from "lucide-react";

type OptionsPanel = "quality" | "subtitles" | "source";

interface PlayerOptionsMenuProps {

  open: boolean;

  qualities: StreamQuality[];
  selectedHeight: number;

  subtitleTracks: SubtitleTrack[];
  activeSubtitleId: string | null;

  qualityEnabled: boolean;
  preferredHeight?: number;
  hdrHeights?: Set<number>;

  /** Live TV: anonymized stream sources (Source 1…); only shown when provided. */
  sourceProviders?: LiveSourceProvider[];
  selectedSourceKey?: string;
  sourceSwitching?: boolean;
  onSourceChange?: (key: string) => void;

  onToggle: () => void;
  onClose: () => void;
  onOutsideClose?: (event: PointerEvent) => void;

  onQualityChange?: (height: number) => void;
  onSubtitleChange: (trackId: string | null) => void;
  onOpenSettings?: () => void;

}

interface PlayerOptionsMenuState {

  panel: OptionsPanel;

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

  state: PlayerOptionsMenuState = { panel: "quality" };

  private rootRef = createRef<HTMLDivElement>();

  sourceEnabled = () => (this.props.sourceProviders?.length ?? 0) > 0 && !!this.props.onSourceChange;

  panelOrder = (): OptionsPanel[] => {

    const order: OptionsPanel[] = [];

    if (this.sourceEnabled()) order.push("source");

    if (this.props.qualityEnabled) order.push("quality");

    // Live TV: source switcher only. VOD: quality + subtitles.
    if (!this.sourceEnabled() || this.props.subtitleTracks.length > 0) {

      order.push("subtitles");

    }

    return order.length > 0 ? order : ["subtitles"];

  };

  defaultPanel = (): OptionsPanel => this.panelOrder()[0] ?? "subtitles";

  componentDidUpdate(prev: PlayerOptionsMenuProps) {

    if (this.props.open && !prev.open) {

      this.setState({ panel: this.defaultPanel() });

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

  renderTab = (panel: OptionsPanel, label: string) => {

    const active = this.state.panel === panel;

    return (

      <button onClick={() => this.setState({ panel })} className={cn(

          "relative flex flex-1 items-center justify-center rounded-md px-3 py-2 text-sm font-medium transition-colors",
          active ? "text-surface" : "text-foreground-muted hover:text-foreground"

        )} >

        {active && <span className="absolute inset-0 rounded-md bg-foreground shadow-sm" />}

        <span className="relative z-10">

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

        "flex w-full items-center gap-2.5 rounded-md px-2.5 py-2.5 text-left text-sm transition-colors",
        disabled && "cursor-not-allowed opacity-40",
        !disabled && (active || loading) && "bg-white/10 text-foreground",
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

  render() {

    const {
      open, qualities, selectedHeight, subtitleTracks, activeSubtitleId, qualityEnabled,
      preferredHeight, hdrHeights, sourceProviders = [], selectedSourceKey = "auto",
      sourceSwitching = false, onSourceChange, onToggle, onClose, onQualityChange,
      onSubtitleChange, onOpenSettings,
    } = this.props;

    const { panel } = this.state;

    const sortedQualities = [...qualities].sort((a, b) => b.height - a.height);

    const panelOrder = this.panelOrder();

    const panelIndex = Math.max(0, panelOrder.indexOf(panel));

    const sourceOn = this.sourceEnabled();

    const showTabs = panelOrder.length > 1;

    return (

      <div ref={this.rootRef} className="relative">

        <button onClick={(e) => {

            e.stopPropagation();

            onToggle();

          }} className={cn(

            "flex size-9 shrink-0 items-center justify-center rounded-md text-foreground transition-colors hover:bg-white/15",
            open && "bg-white/15"

          )} aria-label="Playback options" >

          <Settings2 size={20} />

        </button>

        <div className={cn(

            "absolute right-0 bottom-full z-40 mb-3 w-80 origin-bottom-right overflow-hidden rounded-xl border border-border-subtle bg-surface/80 shadow-2xl shadow-black/40 backdrop-blur-xl transition-[opacity,transform] duration-200 ease-out",
            open ? "pointer-events-auto translate-y-0 scale-100 opacity-100" : "pointer-events-none translate-y-2 scale-95 opacity-0"

          )}

            onClick={(e) => e.stopPropagation()}

          >

            <div className="flex items-center justify-between px-4 py-3">

              <div className="flex items-center gap-2">

                <Settings2 size={14} className="text-foreground-muted" />

                <p className="text-sm font-medium text-foreground">

                  {sourceOn && !qualityEnabled ? "Stream source" : "Playback"}

                </p>

              </div>

              <button onClick={(e) => {

                  e.stopPropagation();

                  onClose();

                }} className="flex size-8 shrink-0 items-center justify-center rounded-md text-foreground-muted transition-colors hover:bg-white/10 hover:text-foreground" aria-label="Close options" >

                <X size={14} />

              </button>

            </div>

            {showTabs && (

              <div className="mx-4 mb-3 rounded-xl border border-border-subtle bg-white/5 p-1">

                <div className="flex gap-1">

                  {sourceOn && this.renderTab("source", "Source")}

                  {qualityEnabled && this.renderTab("quality", "Quality")}

                  {panelOrder.includes("subtitles") && this.renderTab("subtitles", "Subtitles")}

                </div>

              </div>

            )}

            <div className="max-h-72 overflow-hidden px-3 pb-3">

              <div className="flex w-full"

                style={{

                  transform: `translateX(-${panelIndex * 100}%)`,
                  transition: "transform 320ms cubic-bezier(0.22, 1, 0.36, 1)",

                }}

              >

                {sourceOn && (

                  <div className="max-h-72 w-full flex-shrink-0 overflow-y-auto pr-1">

                    <div className="space-y-1.5">

                      {sourceProviders.map((provider) => {

                        const active = selectedSourceKey === provider.key;
                        const loading = sourceSwitching && active;

                        return this.renderOption(

                          active,
                          provider.label,
                          provider.description,
                          () => {

                            if (provider.key === selectedSourceKey || sourceSwitching) return;

                            onSourceChange?.(provider.key);

                          },
                          `source-${provider.key}`,
                          sourceSwitching && !active,
                          loading,

                        );

                      })}

                    </div>

                  </div>

                )}

                {qualityEnabled && (

                  <div className="max-h-72 w-full flex-shrink-0 overflow-y-auto pr-1">

                    <div className="space-y-1.5">

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

                          () => onQualityChange?.(quality.height),
                          `quality-${quality.height}`

                        );

                      })}

                    </div>

                    {onOpenSettings && (

                      <button className="mt-2 mb-2 flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-left text-sm text-foreground-muted transition-colors hover:bg-white/6 hover:text-foreground" onClick={(e) => {

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

                  <div className="space-y-1.5">

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

                      <div className="rounded-md px-3 py-5 text-center">

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

              </div>

            </div>

        </div>

      </div>

    );

  }

}
