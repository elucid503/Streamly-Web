import { Component, createRef, type CSSProperties } from "react";
import { createPortal } from "react-dom";
import { AnimatePresence, motion } from "framer-motion";
import { Check, ChevronDown } from "lucide-react";

import { cn } from "@/lib/utils";

export interface SelectMenuOption {

  value: string;
  label: string;

}

interface SelectMenuProps {

  value: string;
  options: SelectMenuOption[];
  onChange: (value: string) => void;

  label?: string;
  text?: "normal" | "faint";

  className?: string;
  placement?: "top" | "bottom" | "auto";

}

interface SelectMenuState {

  open: boolean;
  openUpward: boolean;
  menuStyle: CSSProperties;

}

const MENU_GAP = 6;

export class SelectMenu extends Component<SelectMenuProps, SelectMenuState> {

  private rootRef = createRef<HTMLDivElement>();
  private menuRef = createRef<HTMLDivElement>();

  state: SelectMenuState = {

    open: false,
    openUpward: false,
    menuStyle: {},

  };

  componentDidMount() {

    document.addEventListener("mousedown", this.handleDocumentMouseDown);

  }

  componentDidUpdate(_: SelectMenuProps, prev: SelectMenuState) {

    if (this.state.open && !prev.open) {

      window.addEventListener("resize", this.close);
      window.addEventListener("scroll", this.handleScroll, true);

    } else if (!this.state.open && prev.open) {

      this.unbindPositionListeners();

    }

  }

  componentWillUnmount() {

    document.removeEventListener("mousedown", this.handleDocumentMouseDown);
    this.unbindPositionListeners();

  }

  unbindPositionListeners = () => {

    window.removeEventListener("resize", this.close);
    window.removeEventListener("scroll", this.handleScroll, true);

  };

  close = () => this.setState({ open: false });

  measure = (): Pick<SelectMenuState, "openUpward" | "menuStyle"> => {

    const { placement = "auto", options } = this.props;
    const root = this.rootRef.current;

    if (!root) return { openUpward: false, menuStyle: {} };

    const rect = root.getBoundingClientRect();
    const estimatedHeight = Math.min(options.length * 36 + 8, 272);

    const openUpward = placement === "top"
      || (placement === "auto" && window.innerHeight - rect.bottom < estimatedHeight + 12);

    const minWidth = rect.width;
    const left = Math.max(8, Math.min(rect.left, window.innerWidth - minWidth - 8));

    return {

      openUpward,
      menuStyle: openUpward
        ? { left, minWidth, bottom: window.innerHeight - rect.top + MENU_GAP }
        : { left, minWidth, top: rect.bottom + MENU_GAP },

    };

  };

  toggleOpen = () => {

    if (this.state.open) {

      this.close();

      return;

    }

    this.setState({ open: true, ...this.measure() });

  };

  handleDocumentMouseDown = (event: MouseEvent) => {

    const target = event.target as Node;

    if (this.rootRef.current?.contains(target)) return;

    if (this.menuRef.current?.contains(target)) return;

    this.close();

  };

  handleScroll = (event: Event) => {

    const menu = this.menuRef.current;

    if (menu && event.target instanceof Node && menu.contains(event.target)) return;

    this.close();

  };

  render() {

    const { value, options, onChange, label, className } = this.props;
    const { open, openUpward, menuStyle } = this.state;

    const selected = options.find((option) => option.value === value) ?? options[0];

    return (

      <div ref={this.rootRef} className={cn("relative", className)}>

        <button
          type="button"
          className={cn(
            "field-focus flex h-8 min-w-[7rem] items-center justify-between gap-2 rounded-md border border-border bg-surface-overlay px-3 text-left text-xs font-medium text-foreground shadow-sm hover:bg-border/60",
            open && "border-border bg-border/60"
          )}
          aria-haspopup="listbox"
          aria-expanded={open}
          title={label}
          onClick={this.toggleOpen}
        >

          <span className={`truncate ${this.props.text === "faint" ? "text-foreground-muted" : "text-foreground"}`}>{selected?.label}</span>
          <ChevronDown size={14} className={cn("shrink-0 text-foreground-muted transition-transform", open && "rotate-180")} />

        </button>

        {createPortal(

          <AnimatePresence>

            {open && (

              <motion.div
                ref={this.menuRef}
                className="fixed z-[100] overflow-hidden rounded-md border border-border bg-surface-raised p-1 shadow-lg"
                initial={{ opacity: 0, scale: 0.96, y: openUpward ? 6 : -6 }}
                animate={{ opacity: 1, scale: 1, y: 0 }}
                exit={{ opacity: 0, scale: 0.96, y: openUpward ? 6 : -6 }}
                transition={{ type: "spring", stiffness: 500, damping: 32 }}
                style={{ ...menuStyle, transformOrigin: openUpward ? "bottom center" : "top center" }}
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
                          "flex w-full items-center justify-between gap-2 rounded-sm px-2 py-1.5 text-left text-sm transition-colors",
                          isSelected
                            ? "bg-surface-overlay text-foreground"
                            : "text-foreground-muted hover:bg-surface-overlay/80 hover:text-foreground"
                        )}
                        onClick={() => {

                          onChange(option.value);
                          this.close();

                        }}
                      >

                        <span className="truncate">{option.label}</span>

                        {isSelected && (

                          <span className="flex size-5 shrink-0 items-center justify-center rounded-md bg-foreground/10">

                            <Check size={11} className="text-foreground" strokeWidth={2.5} />

                          </span>

                        )}

                      </button>

                    );

                  })}

                </div>

              </motion.div>

            )}

          </AnimatePresence>,
          document.body

        )}

      </div>

    );

  }

}
