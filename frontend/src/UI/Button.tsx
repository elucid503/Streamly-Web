import { cn } from "@/Utils/ClassNames";

import { Component, type ButtonHTMLAttributes, type ReactNode } from "react";

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {

  variant?: "default" | "ghost" | "outline" | "secondary";
  size?: "sm" | "md" | "lg" | "icon" | "icon-sm";

  children: ReactNode;

}

export class Button extends Component<ButtonProps> {

  render() {

    const { className, variant = "default", size = "md", children, ...props } = this.props;

    return (

      <button className={cn(

          "inline-flex shrink-0 items-center justify-center gap-2 whitespace-nowrap rounded-md font-medium transition-all focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-accent/40 disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0",

          variant === "default" && "bg-foreground text-surface shadow-sm hover:bg-accent",
          variant === "ghost" && "text-foreground-muted hover:bg-surface-overlay hover:text-foreground",
          variant === "outline" && "border border-border bg-transparent text-foreground shadow-sm hover:bg-surface-overlay",
          variant === "secondary" && "bg-surface-overlay text-foreground shadow-sm hover:bg-border",

          size === "sm" && "h-8 gap-1.5 px-3 text-xs",
          size === "md" && "h-9 px-4 text-sm has-[>svg]:px-3",
          size === "lg" && "h-10 px-6 text-sm has-[>svg]:px-4",
          size === "icon" && "size-9 p-0",
          size === "icon-sm" && "size-8 p-0",

          className

        )}

        {...props}

      >

        {children}

      </button>

    );

  }

}
