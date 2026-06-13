import { cn } from "@/lib/utils";

import { Component, type ButtonHTMLAttributes, type ReactNode } from "react";

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {

  variant?: "default" | "ghost" | "outline";
  size?: "sm" | "md" | "lg";

  children: ReactNode;

}

export class Button extends Component<ButtonProps> {

  render() {

    const { className, variant = "default", size = "md", children, ...props } = this.props;

    return (

      <button className={cn(

          "inline-flex items-center justify-center gap-2 rounded-md font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent disabled:pointer-events-none disabled:opacity-40",

          variant === "default" && "bg-foreground text-surface hover:bg-accent",
          variant === "ghost" && "text-foreground-muted hover:bg-surface-overlay hover:text-foreground",
          variant === "outline" && "border border-border text-foreground hover:bg-surface-overlay",

          size === "sm" && "h-8 px-3 text-xs",
          size === "md" && "h-9 px-4 text-sm",
          size === "lg" && "h-11 px-6 text-sm",

          className

        )}

        {...props}

      >

        {children}

      </button>

    );

  }

}
