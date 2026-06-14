import { cn } from "@/lib/utils";

import { Component, createRef } from "react";
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

}

interface SelectMenuState {

  open: boolean;

}

export class SelectMenu extends Component<SelectMenuProps, SelectMenuState> {

  private rootRef = createRef<HTMLDivElement>();

  state: SelectMenuState = {

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

  render() {

    const { value, options, onChange, label, className } = this.props;
    const { open } = this.state;

    const selected = options.find((option) => option.value === value) ?? options[0];

    return (

      <div ref={this.rootRef} className={cn("relative", className)}>

        <button
          type="button"
          className={cn(
            "flex h-9 min-w-[132px] items-center justify-between gap-2 rounded-full border border-border-subtle bg-surface-raised px-3 text-left text-xs font-medium text-foreground shadow-sm transition-colors hover:border-border hover:bg-surface-overlay",
            open && "border-border bg-surface-overlay"
          )}
          aria-haspopup="listbox"
          aria-expanded={open}
          title={label}
          onClick={() => this.setState({ open: !open })}
        >

          <span className="truncate">{selected?.label}</span>
          <ChevronDown size={14} className={cn("shrink-0 text-foreground-muted transition-transform", open && "rotate-180")} />

        </button>

        {open && (

          <div className="absolute left-0 top-[calc(100%+6px)] z-50 min-w-full overflow-hidden rounded-md border border-border bg-surface-raised py-1 shadow-xl">

            <div role="listbox" aria-label={label} className="max-h-64 overflow-y-auto py-1">

              {options.map((option) => {

                const isSelected = option.value === value;

                return (

                  <button
                    key={option.value}
                    type="button"
                    role="option"
                    aria-selected={isSelected}
                    className={cn(
                      "flex w-full items-center justify-between gap-3 px-3 py-2 text-left text-xs text-foreground-muted transition-colors hover:bg-surface-overlay hover:text-foreground",
                      isSelected && "text-foreground"
                    )}
                    onClick={() => {

                      onChange(option.value);
                      this.setState({ open: false });

                    }}
                  >

                    <span className="whitespace-nowrap">{option.label}</span>
                    {isSelected && <Check size={13} className="text-accent" />}

                  </button>

                );

              })}

            </div>

          </div>

        )}

      </div>

    );

  }

}
