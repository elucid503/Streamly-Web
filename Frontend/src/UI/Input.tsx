import { cn } from "@/Utils/ClassNames";

import { Component, type InputHTMLAttributes } from "react";

interface InputProps extends InputHTMLAttributes<HTMLInputElement> {}

export class Input extends Component<InputProps> {

  render() {

    const { className, ...props } = this.props;

    return (

      <input className={cn(

          "field-focus flex h-9 w-full min-w-0 rounded-md border border-border bg-surface-overlay/40 px-3 py-1 text-base text-foreground shadow-sm placeholder:text-foreground-faint transition-colors focus:border-border focus:bg-surface-overlay/80 lg:text-sm",
          className

        )}

        {...props}

      />

    );

  }

}
