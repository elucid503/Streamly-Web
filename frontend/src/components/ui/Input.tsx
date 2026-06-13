import { cn } from "@/lib/utils";

import { Component, type InputHTMLAttributes } from "react";

interface InputProps extends InputHTMLAttributes<HTMLInputElement> {}

export class Input extends Component<InputProps> {

  render() {

    const { className, ...props } = this.props;

    return (

      <input className={cn(

          "flex h-10 w-full rounded-md border border-border bg-surface-raised px-3 py-2 text-sm text-foreground placeholder:text-foreground-faint focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent",
          className

        )}

        {...props}

      />

    );

  }

}
