import { cn } from "@/Utils/ClassNames";

import { Component } from "react";

interface SwitchProps {

  checked: boolean;
  label: string;
  description?: string;

  onChange: (checked: boolean) => void;

  className?: string;

}

export class Switch extends Component<SwitchProps> {

  render() {

    const { checked, onChange, label, description } = this.props;

    return (

      <label className={`flex cursor-pointer items-center justify-between gap-4 py-2 ${this.props.className || ""}`}>

        <span className="min-w-0">

          <span className="block text-sm text-foreground-muted">{label}</span>

          {description ? (

            <span className="mt-0.5 block text-xs text-foreground-muted/70">{description}</span>

          ) : null}

        </span>

        <div className="relative flex-shrink-0">

          <input type="checkbox"

            role="switch"
            className="sr-only"

            checked={checked}
            onChange={() => onChange(!checked)}

          />

          <div className={cn(

              "relative h-5 w-9 rounded-full transition-colors",
              checked ? "bg-foreground" : "bg-border"

            )}

          >

            <div className={cn(

                "absolute top-0.5 left-0.5 size-4 rounded-full bg-surface shadow-sm transition-transform",
                checked && "translate-x-4"

              )}

            />

          </div>

        </div>

      </label>

    );

  }

}
