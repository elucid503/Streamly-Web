import { cn } from "@/lib/utils";

import { Component, createRef } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { Check, ChevronDown } from "lucide-react";

export interface SelectMenuOption {

  value: string;
  label: string;

}

interface SelectMenuProps {

  value: string;
  options: SelectMenuOption[];
  onChange: (value: string) => void;

  label?: string;
  className?: string;
  placement?: "top" | "bottom" | "auto";

}

interface SelectMenuState {

  open: boolean;
  openUpward: boolean;

}

export class SelectMenu extends Component<SelectMenuProps, SelectMenuState> {

  private rootRef = createRef<HTMLDivElement>();

  state: SelectMenuState = {

    open: false,
    openUpward: false,

  };

  componentDidMount() {

    document.addEventListener("mousedown", this.handleDocumentMouseDown);

  }

  componentWillUnmount() {

    document.removeEventListener("mousedown", this.handleDocumentMouseDown);

  }

  resolveOpenUpward = (): boolean => {

    const { placement = "auto", options } = this.props;

    if (placement === "top") return true;

    if (placement === "bottom") return false;

    const root = this.rootRef.current;

    if (!root) return false;

    const rect = root.getBoundingClientRect();
    const estimatedHeight = Math.min(options.length * 36 + 8, 272);

    return window.innerHeight - rect.bottom < estimatedHeight + 12;

  };

  toggleOpen = () => {

    const { open } = this.state;

    if (open) {

      this.setState({ open: false });

      return;

    }

    this.setState({ open: true, openUpward: this.resolveOpenUpward() });

  };

  handleDocumentMouseDown = (event: MouseEvent) => {

    const root = this.rootRef.current;

    if (!root || root.contains(event.target as Node)) return;

    this.setState({ open: false });

  };

  render() {

    const { value, options, onChange, label, className } = this.props;
    const { open, openUpward } = this.state;

    const selected = options.find((option) => option.value === value) ?? options[0];

    return (

      <div ref={this.rootRef} className={cn("relative", className)}>

        <button
          type="button"
          className={cn(
            "field-focus flex h-9 min-w-[132px] items-center justify-between gap-2 rounded-full border border-border-subtle bg-surface-raised px-3 text-left text-xs font-medium text-foreground hover:border-border hover:bg-surface-overlay",
            open && "border-border bg-surface-overlay"
          )}
          aria-haspopup="listbox"
          aria-expanded={open}
          title={label}
          onClick={this.toggleOpen}
        >

          <span className="truncate">{selected?.label}</span>
          <ChevronDown size={14} className={cn("shrink-0 text-foreground-muted transition-transform", open && "rotate-180")} />

        </button>

        <AnimatePresence>

          {open && (

            <motion.div className={cn(

                "absolute left-0 z-50 min-w-full overflow-hidden rounded-[1.25rem] border border-border-subtle bg-surface/95 p-1 shadow-2xl ring-1 ring-white/[0.04] backdrop-blur-lg",
                openUpward ? "bottom-[calc(100%+8px)]" : "top-[calc(100%+8px)]"

              )}

              initial={{ opacity: 0, scale: 0.96, y: openUpward ? 6 : -6 }}
              animate={{ opacity: 1, scale: 1, y: 0 }}

              exit={{ opacity: 0, scale: 0.96, y: openUpward ? 6 : -6 }}

              transition={{ type: "spring", stiffness: 500, damping: 32 }}
              style={{ transformOrigin: openUpward ? "bottom center" : "top center" }}

            >

              <div role="listbox" aria-label={label} className="flex max-h-64 flex-col gap-0.5 overflow-y-auto">

                {options.map((option) => {

                  const isSelected = option.value === value;

                  return (

                    <button
                      key={option.value}
                      type="button"
                      role="option"
                      aria-selected={isSelected}
                      className={cn(
                        "flex h-9 w-full items-center justify-between gap-2 rounded-xl px-3 text-left text-xs font-medium transition-colors",
                        isSelected
                          ? "bg-surface-raised text-foreground shadow-sm"
                          : "text-foreground-muted hover:bg-surface-overlay/80 hover:text-foreground"
                      )}
                      onClick={() => {

                        onChange(option.value);
                        this.setState({ open: false });

                      }}
                    >

                      <span className="truncate">{option.label}</span>

                      {isSelected && (

                        <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-foreground/10">

                          <Check size={11} className="text-foreground" strokeWidth={2.5} />

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
