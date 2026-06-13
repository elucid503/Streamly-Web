import { cn } from "@/lib/utils";
import type { MainView } from "@/lib/types";

import { Component } from "react";
import { motion } from "framer-motion";

interface ViewSwitcherProps {

  active: MainView;

  onChange: (view: MainView) => void;

}

const views: { id: MainView; label: string }[] = [

  { id: "shows", label: "TV Shows" },
  { id: "movies", label: "Movies" },
  { id: "live", label: "Live TV" },

];

export class ViewSwitcher extends Component<ViewSwitcherProps> {

  render() {

    const { active, onChange } = this.props;

    return (

      <div className="flex w-full justify-center">

        <div className="inline-flex rounded-full border border-border bg-surface-raised p-1">

          {views.map((view) => {

            const isActive = active === view.id;

            return (

              <button key={view.id} type="button" onClick={() => onChange(view.id)}

                className={cn(

                  "relative rounded-full px-4 py-1.5 text-xs font-medium transition-colors sm:px-6 sm:text-sm",
                  isActive ? "text-surface" : "text-foreground-muted hover:text-foreground"

                )}

              >

                {isActive && (

                  <motion.span layoutId="view-switcher-pill" className="absolute inset-0 rounded-full bg-foreground"

                    transition={{ type: "spring", stiffness: 400, damping: 32 }}

                  />

                )}

                <span className="relative z-10 whitespace-nowrap">{view.label}</span>

              </button>

            );

          })}

        </div>

      </div>

    );

  }

}
