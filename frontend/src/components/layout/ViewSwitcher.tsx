import { cn } from "@/lib/utils";
import { store } from "@/lib/store";
import type { MainView } from "@/lib/types";

import { Component } from "react";
import { motion } from "framer-motion";

interface ViewSwitcherProps {

  active: MainView;

  onChange: (view: MainView) => void;

}

interface ViewSwitcherState {

  count: number;

}

const views: { id: MainView; label: string }[] = [

  { id: "shows", label: "TV Shows" },
  { id: "movies", label: "Movies" },
  { id: "live", label: "Live TV" },
  { id: "friends", label: "Friends" },

];

export class ViewSwitcher extends Component<ViewSwitcherProps, ViewSwitcherState> {

  private unsub = () => {};

  state: ViewSwitcherState = { count: store.incomingRequestCount };

  componentDidMount() {

    this.unsub = store.subscribe(() => this.setState({ count: store.incomingRequestCount }));

  }

  componentWillUnmount() {

    this.unsub();

  }

  render() {

    const { active, onChange } = this.props;
    const { count } = this.state;

    return (

      <div className="inline-flex rounded-full border border-border bg-surface-raised p-1">

        {views.map((view) => {

          const isActive = active === view.id;
          const badge = view.id === "friends" && count > 0 ? count : null;

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

              <span className="relative z-10 inline-flex items-center gap-1.5 whitespace-nowrap">

                {view.label}

                {badge !== null && (

                  <span className={cn(

                    "flex h-4 min-w-[1rem] items-center justify-center rounded-full px-1 text-[10px] font-semibold",
                    isActive ? "bg-surface text-foreground" : "bg-foreground text-surface"

                  )}>

                    {badge}

                  </span>

                )}

              </span>

            </button>

          );

        })}

      </div>

    );

  }

}
