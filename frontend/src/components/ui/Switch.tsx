import { Component } from "react";
import { cn } from "@/lib/utils";

interface SwitchProps {
  checked: boolean;
  onChange: (checked: boolean) => void;
  label: string;
}

export class Switch extends Component<SwitchProps> {
  render() {
    const { checked, onChange, label } = this.props;
    return (
      <label className="flex cursor-pointer items-center justify-between gap-4 py-2">
        <span className="text-sm text-foreground-muted">{label}</span>
        <button
          type="button"
          role="switch"
          aria-checked={checked}
          onClick={() => onChange(!checked)}
          className={cn(
            "relative h-5 w-9 rounded-full transition-colors",
            checked ? "bg-foreground" : "bg-border",
          )}
        >
          <span
            className={cn(
              "absolute top-0.5 left-0.5 h-4 w-4 rounded-full bg-surface transition-transform",
              checked && "translate-x-4",
            )}
          />
        </button>
      </label>
    );
  }
}