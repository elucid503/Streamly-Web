import { cn } from "@/lib/utils";

import { Component, createRef, type CSSProperties, type ReactNode } from "react";
import { ChevronLeft, ChevronRight } from "lucide-react";

interface HScrollRowProps {

  className?: string;
  children: ReactNode;

}

interface HScrollRowState {

  canLeft: boolean;
  canRight: boolean;

}

const buttonClass = "flex size-7 shrink-0 items-center justify-center rounded-md border border-border-subtle bg-surface-raised text-foreground-muted transition-colors hover:border-border hover:bg-surface-overlay hover:text-foreground";

export class HScrollRow extends Component<HScrollRowProps, HScrollRowState> {

  state: HScrollRowState = {

    canLeft: false,
    canRight: false,

  };

  private scroller = createRef<HTMLDivElement>();
  private observer: ResizeObserver | null = null;

  componentDidMount() {

    const el = this.scroller.current;

    if (!el) return;

    el.addEventListener("scroll", this.sync, { passive: true });

    this.observer = new ResizeObserver(this.sync);
    this.observer.observe(el);

    this.sync();

  }

  componentDidUpdate() {

    this.sync();

  }

  componentWillUnmount() {

    this.scroller.current?.removeEventListener("scroll", this.sync);
    this.observer?.disconnect();

  }

  sync = () => {

    const el = this.scroller.current;

    if (!el) return;

    const canLeft = el.scrollLeft > 4;
    const canRight = el.scrollWidth - el.clientWidth - el.scrollLeft > 4;

    if (canLeft === this.state.canLeft && canRight === this.state.canRight) return;

    this.setState({ canLeft, canRight });

  };

  scroll = (direction: -1 | 1) => {

    const el = this.scroller.current;

    if (!el) return;

    el.scrollBy({ left: direction * Math.max(el.clientWidth * 0.7, 120), behavior: "smooth" });

  };

  maskStyle(): CSSProperties | undefined {

    const { canLeft, canRight } = this.state;

    if (!canLeft && !canRight) return undefined;

    const maskImage = canLeft && canRight
      ? "linear-gradient(to right, transparent, #000 1.25rem, #000 calc(100% - 1.25rem), transparent)"
      : canLeft
        ? "linear-gradient(to right, transparent, #000 1.25rem, #000)"
        : "linear-gradient(to right, #000, #000 calc(100% - 1.25rem), transparent)";

    return { maskImage, WebkitMaskImage: maskImage };

  }

  render() {

    const { className, children } = this.props;
    const { canLeft, canRight } = this.state;

    return (

      <div className="flex min-w-0 items-center gap-1">

        {canLeft && (

          <button type="button" aria-label="Scroll left" onClick={(e) => {

              e.stopPropagation();

              this.scroll(-1);

            }} className={buttonClass}

          >

            <ChevronLeft size={16} />

          </button>

        )}

        <div ref={this.scroller} className={cn("flex min-w-0 flex-1 overflow-x-auto scrollbar-hide", className)} style={this.maskStyle()}>

          {children}

        </div>

        {canRight && (

          <button type="button" aria-label="Scroll right" onClick={(e) => {

              e.stopPropagation();

              this.scroll(1);

            }} className={buttonClass}

          >

            <ChevronRight size={16} />

          </button>

        )}

      </div>

    );

  }

}
