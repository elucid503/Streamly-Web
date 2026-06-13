import { cn } from "@/lib/utils";

import { Component } from "react";

interface SwitchProps {

  checked: boolean;
  label: string;

  onChange: (checked: boolean) => void;


}

export class Switch extends Component<SwitchProps> {

  render() {

    const { checked, onChange, label } = this.props;

    return (

      <label className="flex cursor-pointer items-center justify-between gap-4 py-2">

        <span className="text-sm text-foreground-muted">{label}</span>

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

                "absolute top-0.5 left-0.5 h-4 w-4 rounded-full bg-surface transition-transform",
                checked && "translate-x-4"

              )}

            />

          </div>

        </div>

      </label>

    );

  }

}
