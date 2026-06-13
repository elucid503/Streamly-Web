import { cn } from "@/lib/utils";

import { Component, type ReactNode } from "react";
import { Volume2, VolumeX } from "lucide-react";

interface VolumeControlProps {

  volume: number;
  muted: boolean;

  onVolumeChange: (volume: number) => void;
  onToggleMute: () => void;

}

interface VolumeControlState {

  hovering: boolean;

}

export class VolumeControl extends Component<VolumeControlProps, VolumeControlState> {

  state: VolumeControlState = { hovering: false };

  private hideTimer: ReturnType<typeof setTimeout> | null = null;

  componentWillUnmount() {

    if (this.hideTimer) clearTimeout(this.hideTimer);

  }

  showSlider = () => {

    if (this.hideTimer) clearTimeout(this.hideTimer);

    this.setState({ hovering: true });

  };

  scheduleHide = () => {

    if (this.hideTimer) clearTimeout(this.hideTimer);

    this.hideTimer = setTimeout(() => this.setState({ hovering: false }), 500);

  };

  render() {

    const { volume, muted, onVolumeChange, onToggleMute } = this.props;

    const { hovering } = this.state;

    const pct = Math.round((muted ? 0 : volume) * 100);

    return (

      <div className="group relative flex items-center"

        onMouseEnter={this.showSlider}
        onMouseLeave={this.scheduleHide}

      >
        <button onClick={(e) => {

            e.stopPropagation();

            onToggleMute();

          }} className="rounded-md p-1.5 text-foreground transition-colors hover:bg-white/10" >

          {muted || volume === 0 ? <VolumeX size={18} /> : <Volume2 size={18} />}

        </button>

        <div className={ hovering ? "w-[104px] overflow-hidden opacity-100 transition-all duration-200 ease-out" : "w-0 overflow-hidden opacity-0 transition-all duration-200 ease-out" } >

          <div className="relative mx-2 flex h-7 w-24 items-center">

            <div className="pointer-events-none absolute inset-x-0 top-1/2 h-0.5 -translate-y-1/2 rounded-full bg-white/20 transition-all duration-150 group-hover:h-1">

              <div className="h-full rounded-full bg-foreground" style={{ width: `${pct}%` }} />

            </div>

            <input className="player-range relative z-10 h-7 w-full cursor-pointer appearance-none bg-transparent"

              type="range"

              min={0}
              max={100}

              value={pct}
              onChange={(e) => {

                e.stopPropagation();

                onVolumeChange(Number(e.target.value) / 100);

              }}

              aria-label="Volume"

            />

          </div>

        </div>

      </div>

    );

  }

}

export class ControlButton extends Component<{

  onClick: () => void;

  children: ReactNode;
  className?: string;

  "aria-label"?: string;

}> {

  render() {

    const { onClick, children, className, "aria-label": ariaLabel } = this.props;

    return (

      <button onClick={(e) => {

          e.stopPropagation();

          onClick();

        }} aria-label={ariaLabel} className={cn(

          "rounded-md p-1.5 text-foreground transition-colors hover:bg-white/10",
          className

        )} >

        {children}

      </button>

    );

  }

}
