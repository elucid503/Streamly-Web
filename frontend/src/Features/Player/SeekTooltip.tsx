import { Component, createRef } from "react";

import { formatDuration } from "@/Utils/Time";

export class SeekTooltip extends Component {

  private rootRef = createRef<HTMLSpanElement>();

  update = (ratio: number, durationMs: number) => {

    const root = this.rootRef.current;

    if (!root || !Number.isFinite(ratio) || !Number.isFinite(durationMs) || durationMs <= 0) return;

    const clamped = Math.max(0, Math.min(1, ratio));

    root.textContent = formatDuration(clamped * durationMs);
    root.style.left = `clamp(32px, ${clamped * 100}%, calc(100% - 32px))`;
    root.style.opacity = "1";

  };

  hide = () => {

    if (this.rootRef.current) this.rootRef.current.style.opacity = "0";

  };

  render() {

    return (

      <span
        ref={this.rootRef}
        className="pointer-events-none absolute bottom-full z-20 mb-2 -translate-x-1/2 rounded-md bg-black/80 px-2 py-1 text-xs text-white tabular-nums opacity-0"
        aria-hidden
      />

    );

  }

}
