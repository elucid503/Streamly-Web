import { cn } from "@/lib/utils";
import type { StreamQuality, SubtitleTrack } from "@/lib/types";

import { Component, createRef, type ReactNode } from "react";
import { Check, Gauge, Settings, Settings2, Subtitles, X } from "lucide-react";

type OptionsPanel = "quality" | "subtitles";

interface PlayerOptionsMenuProps {

  open: boolean;

  qualities: StreamQuality[];
  selectedHeight: number;

  subtitleTracks: SubtitleTrack[];
  activeSubtitleId: string | null;

  qualityEnabled: boolean;
  preferredHeight?: number;

  onToggle: () => void;
  onClose: () => void;

  onQualityChange: (height: number) => void;
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

  panelOrder = (): OptionsPanel[] => {

    const order: OptionsPanel[] = [];

    if (this.props.qualityEnabled) order.push("quality");

    order.push("subtitles");

    return order;

  };

  defaultPanel = (): OptionsPanel => this.panelOrder()[0] ?? "subtitles";

  componentDidUpdate(prev: PlayerOptionsMenuProps) {

    if (this.props.open && !prev.open) {

      this.setState({ panel: this.defaultPanel() });

    }

    if (this.props.open !== prev.open) {

      if (this.props.open) {

        document.addEventListener("mousedown", this.handleOutsideClick);

      } else {

        document.removeEventListener("mousedown", this.handleOutsideClick);

      }

    }

  }

  componentWillUnmount() {

    document.removeEventListener("mousedown", this.handleOutsideClick);

  }

  handleOutsideClick = (event: MouseEvent) => {

    const root = this.rootRef.current;

    if (!root || root.contains(event.target as Node)) return;

    this.props.onClose();

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

  renderOption = (active: boolean, label: string, detail: string | undefined, onClick: () => void, key: string) => (

    <button key={key} onClick={(e) => {

        e.stopPropagation();

        onClick();

      }} className={cn(

        "flex w-full items-center gap-3 rounded-lg px-3.5 py-3 text-left transition-colors",
        active ? "bg-white/10 text-foreground ring-1 ring-white/10" : "text-foreground-muted hover:bg-white/6 hover:text-foreground"

      )} >

      <span className={cn(

          "flex h-4 w-4 shrink-0 items-center justify-center rounded-full border transition-colors",
          active ? "border-accent bg-accent text-black" : "border-white/20 bg-transparent"

        )}

      >

        {active && <Check size={10} strokeWidth={3} />}

      </span>

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

    const { open, qualities, selectedHeight, subtitleTracks, activeSubtitleId, qualityEnabled, preferredHeight, onToggle, onClose, onQualityChange, onSubtitleChange, onOpenSettings, } = this.props;
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

        {open && (

          <div className="absolute right-0 bottom-full z-40 mb-3 w-80 animate-fade-in overflow-hidden rounded-xl border border-border-subtle bg-surface/80 shadow-2xl backdrop-blur-md"

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

                        const detail = isPreferred ? `${quality.label} · Preferred` : quality.label;

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

                      <button onClick={(e) => {

                          e.stopPropagation();

                          onClose();
                          onOpenSettings();

                        }} className="mt-2 flex w-full items-center gap-2 rounded-lg border border-white/8 px-3.5 py-2.5 text-left text-xs text-foreground-muted transition-colors hover:bg-white/6 hover:text-foreground" >

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

              </div>

            </div>

          </div>

        )}

      </div>

    );

  }

}
