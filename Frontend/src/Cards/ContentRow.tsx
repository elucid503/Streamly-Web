import { cn } from "@/Utils/ClassNames";

import { Component, type ReactNode } from "react";
import { ChevronLeft, ChevronRight } from "lucide-react";

interface ContentRowProps {

  title: string;
  subtitle?: string;
  sectionId?: string;

  children?: ReactNode;

  loading?: boolean;

}

export class ContentRow extends Component<ContentRowProps> {

  private scroller: HTMLDivElement | null = null;

  scroll = (dir: -1 | 1) => {

    if (!this.scroller) return;

    this.scroller.scrollBy({ left: dir * 400, behavior: "smooth" });

  };

  render() {

    const { title, subtitle, sectionId, children, loading } = this.props;

    return (

      <section id={sectionId} className="mb-8 flex flex-col gap-3 scroll-mt-36">

        <div className="flex items-center justify-between gap-3 px-4 sm:px-8">

          <div className="min-w-0">

            {title ? (

              <h2 className="text-base font-semibold tracking-tight text-foreground">

                {title}

              </h2>

            ) : null}

            {subtitle && (

              <p className="mt-0.5 text-sm text-foreground-muted">

                {subtitle}

              </p>

            )}

          </div>

          <div className="hidden shrink-0 items-center gap-1 sm:flex">

            <button onClick={() => this.scroll(-1)} className="flex size-8 items-center justify-center rounded-md border border-border-subtle bg-surface-raised text-foreground-muted transition-colors hover:border-border hover:bg-surface-overlay hover:text-foreground" >

              <ChevronLeft className="size-4" />

            </button>

            <button onClick={() => this.scroll(1)} className="flex size-8 items-center justify-center rounded-md border border-border-subtle bg-surface-raised text-foreground-muted transition-colors hover:border-border hover:bg-surface-overlay hover:text-foreground" >

              <ChevronRight className="size-4" />

            </button>

          </div>

        </div>

        {loading ? (

          <div className="flex gap-3 overflow-hidden px-4 sm:px-8">

            {Array.from({ length: 8 }).map((_, i) => (

              <div key={i} className="skeleton aspect-[2/3] w-[8.5rem] flex-shrink-0 rounded-lg sm:w-36" />

            ))}

          </div>

        ) : (

          <div className={cn("flex gap-3 overflow-x-auto px-4 pb-1 scrollbar-hide sm:px-8")} ref={(el) => { this.scroller = el; }} >

            {children}

          </div>

        )}

      </section>

    );

  }

}
