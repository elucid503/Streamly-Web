import { Component, createRef } from "react";
import { AnimatePresence, motion } from "framer-motion";
import { Check, ChevronDown } from "lucide-react";

import { cn } from "@/Utils/ClassNames";

interface FeaturedTeamSelectProps {

  value: string;
  options: string[];
  onChange: (team: string) => void;

}

interface FeaturedTeamSelectState {

  open: boolean;

}

/** Subtle light-themed select, styled to sit flush with the banner Watch button. */
export class FeaturedTeamSelect extends Component<FeaturedTeamSelectProps, FeaturedTeamSelectState> {

  private rootRef = createRef<HTMLDivElement>();

  state: FeaturedTeamSelectState = {

    open: false,

  };

  componentDidMount() {

    document.addEventListener("mousedown", this.handleDocumentMouseDown);

  }

  componentWillUnmount() {

    document.removeEventListener("mousedown", this.handleDocumentMouseDown);

  }

  handleDocumentMouseDown = (event: MouseEvent) => {

    const root = this.rootRef.current;

    if (!root || root.contains(event.target as Node)) return;

    this.setState({ open: false });

  };

  toggleOpen = () => {

    this.setState((s) => ({ open: !s.open }));

  };

  render() {

    const { value, options, onChange } = this.props;
    const { open } = this.state;

    // Only show a team label when that team is currently selectable (in Live Now).
    const activeValue = value && options.some((t) => t.toLowerCase() === value.toLowerCase()) ? value : "";
    const triggerLabel = activeValue || "Auto";

    return (

      <div ref={this.rootRef} className="relative">

        <button type="button" className={cn("hidden field-focus md:flex h-9 min-w-[132px] max-w-[11rem] items-center justify-between gap-2 rounded-md border border-border-subtle bg-surface-raised px-3 text-left text-xs font-medium text-foreground hover:border-border hover:bg-surface-overlay", open && "border-border bg-surface-overlay" )}
          aria-haspopup="listbox"
          aria-expanded={open}
          aria-label="Featured team"
          title="Feature a team in the banner"
          onClick={this.toggleOpen}

        >

          <span className="truncate">{triggerLabel}</span>
          <ChevronDown className={cn("size-3.5 shrink-0 text-foreground-muted transition-transform", open && "rotate-180")} />

        </button>

        <AnimatePresence>

          {open && (

            <motion.div
              className="absolute right-0 top-[calc(100%+8px)] z-50 min-w-[16rem] overflow-hidden rounded-lg border border-border-subtle bg-surface/95 p-1.5 shadow-2xl ring-1 ring-white/[0.04] backdrop-blur-lg"
              initial={{ opacity: 0, scale: 0.96, y: -6 }}
              animate={{ opacity: 1, scale: 1, y: 0 }}
              exit={{ opacity: 0, scale: 0.96, y: -6 }}
              transition={{ type: "spring", stiffness: 500, damping: 32 }}
              style={{ transformOrigin: "top right" }}
            >

              {/* Same surface language as SelectMenu, with roomier rows for long team lists. */}
              <div role="listbox" aria-label="Featured team" className="flex max-h-72 flex-col gap-0.5 overflow-y-auto">

                <button
                  type="button"
                  role="option"
                  aria-selected={!activeValue}
                  className={cn(
                    "flex min-h-10 w-full items-center justify-between gap-2.5 rounded-md px-3 py-2 text-left text-sm font-medium transition-colors",
                    !activeValue
                      ? "bg-surface-raised text-foreground shadow-sm"
                      : "text-foreground-muted hover:bg-surface-overlay/80 hover:text-foreground"
                  )}
                  onClick={() => {

                    onChange("");
                    this.setState({ open: false });

                  }}
                >

                  <span className="truncate">Auto</span>

                  {!activeValue && (

                    <span className="flex size-5 shrink-0 items-center justify-center rounded-full bg-foreground/10">

                      <Check className="size-2.5 text-foreground" strokeWidth={2.5} />

                    </span>

                  )}

                </button>

                {options.map((team) => {

                  const isSelected = activeValue.toLowerCase() === team.toLowerCase();

                  return (

                    <button
                      key={team}
                      type="button"
                      role="option"
                      aria-selected={isSelected}
                      className={cn(
                        "flex min-h-10 w-full items-center justify-between gap-2.5 rounded-md px-3 py-2 text-left text-sm font-medium transition-colors",
                        isSelected
                          ? "bg-surface-raised text-foreground shadow-sm"
                          : "text-foreground-muted hover:bg-surface-overlay/80 hover:text-foreground"
                      )}
                      onClick={() => {

                        onChange(team);
                        this.setState({ open: false });

                      }}
                    >

                      <span className="truncate leading-snug">{team}</span>

                      {isSelected && (

                        <span className="flex size-5 shrink-0 items-center justify-center rounded-full bg-foreground/10">

                          <Check className="size-2.5 text-foreground" strokeWidth={2.5} />

                        </span>

                      )}

                    </button>

                  );

                })}

              </div>

            </motion.div>

          )}

        </AnimatePresence>

      </div>

    );

  }

}
