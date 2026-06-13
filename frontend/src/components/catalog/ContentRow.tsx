import { Component, type ReactNode } from "react";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { cn } from "@/lib/utils";

interface ContentRowProps {
  title: string;
  children: ReactNode;
  loading?: boolean;
}

export class ContentRow extends Component<ContentRowProps> {
  private scroller: HTMLDivElement | null = null;

  scroll = (dir: -1 | 1) => {
    if (!this.scroller) return;
    this.scroller.scrollBy({ left: dir * 400, behavior: "smooth" });
  };

  render() {
    const { title, children, loading } = this.props;

    return (
      <section className="mb-8">
        <div className="mb-3 flex items-center justify-between px-4 sm:px-8">
          <h2 className="text-sm font-medium tracking-wide text-foreground-muted uppercase">
            {title}
          </h2>
          <div className="hidden gap-1 sm:flex">
            <button
              onClick={() => this.scroll(-1)}
              className="rounded-md p-1.5 text-foreground-faint transition-colors hover:bg-surface-overlay hover:text-foreground"
            >
              <ChevronLeft size={16} />
            </button>
            <button
              onClick={() => this.scroll(1)}
              className="rounded-md p-1.5 text-foreground-faint transition-colors hover:bg-surface-overlay hover:text-foreground"
            >
              <ChevronRight size={16} />
            </button>
          </div>
        </div>

        {loading ? (
          <div className="flex gap-3 overflow-hidden px-4 sm:px-8">
            {Array.from({ length: 8 }).map((_, i) => (
              <div key={i} className="skeleton h-[210px] w-[140px] flex-shrink-0" />
            ))}
          </div>
        ) : (
          <div
            ref={(el) => {
              this.scroller = el;
            }}
            className={cn(
              "flex gap-3 overflow-x-auto px-4 pb-1 scrollbar-hide sm:gap-4 sm:px-8",
            )}
          >
            {children}
          </div>
        )}
      </section>
    );
  }
}