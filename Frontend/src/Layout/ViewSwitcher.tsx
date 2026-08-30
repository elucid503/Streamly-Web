import { motion } from "framer-motion";
import { Clapperboard, Radio, Trophy, Users } from "lucide-react";

import { ModuleComponent } from "@/Core/Store";
import Stores from "@/Stores";
import type { MainView } from "@/Types";
import { cn } from "@/Utils/ClassNames";

interface ViewSwitcherProps {

  active: MainView;

  onChange: (view: MainView) => void;

}

const views: { id: Exclude<MainView, "friends">; label: string; icon: typeof Clapperboard }[] = [

  { id: "vod", label: "Movies & Shows", icon: Clapperboard },
  { id: "live", label: "Live TV", icon: Radio },
  { id: "sports", label: "Sports", icon: Trophy },

];

export class ViewSwitcher extends ModuleComponent<ViewSwitcherProps> {

  componentDidMount() {

    this.watch(Stores.Social);

  }

  render() {

    const { active, onChange } = this.props;
    const friendRequestCount = Stores.Social.incomingRequestCount;
    const friendsActive = active === "friends";

    return (

      <div className="pointer-events-none fixed inset-x-0 bottom-0 z-40 flex justify-center px-3 pb-[max(0.75rem,env(safe-area-inset-bottom))]">

        <div className="pointer-events-auto flex items-center gap-2">

          <nav className="flex items-center gap-0.5 rounded-full border border-border bg-surface-raised/90 p-1 shadow-xl backdrop-blur-md sm:gap-1">

          {views.map((view) => {

            const isActive = active === view.id;
            const Icon = view.icon;

            return (

              <button
                key={view.id}
                type="button"
                onClick={() => onChange(view.id)}
                className={cn(
                  "relative flex items-center justify-center gap-1.5 rounded-full px-5 py-2 text-xs font-medium sm:gap-2 sm:px-3.5 sm:text-sm",
                  isActive ? "text-surface" : "text-foreground-muted hover:text-foreground"
                )}
                title={view.label}
                aria-label={view.label}
              >

                {isActive && (

                  <motion.span
                    layoutId="bottom-nav-pill"
                    className="absolute inset-0 rounded-full bg-foreground shadow-sm"
                    transition={{ type: "spring", stiffness: 400, damping: 32 }}
                  />

                )}

                <span className="relative z-10 flex items-center gap-1.5 sm:gap-2">

                  <Icon className="size-5 sm:size-4" />

                  <span className="hidden whitespace-nowrap sm:inline">{view.label}</span>

                </span>

              </button>

            );

          })}

          </nav>

          <nav className="rounded-full border border-border bg-surface-raised/90 p-1 shadow-xl backdrop-blur-md">

            <button
              type="button"
              onClick={() => onChange("friends")}
              className={cn(
                "relative flex items-center justify-center gap-1.5 rounded-full px-5 py-2 text-xs font-medium sm:gap-2 sm:px-3.5 sm:text-sm",
                friendsActive ? "text-surface" : "text-foreground-muted hover:text-foreground"
              )}
              title="Friends"
              aria-label="Friends"
            >

              {friendsActive && (

                <motion.span
                  layoutId="bottom-nav-pill"
                  className="absolute inset-0 rounded-full bg-foreground shadow-sm"
                  transition={{ type: "spring", stiffness: 400, damping: 32 }}
                />

              )}

              <span className="relative z-10 flex items-center gap-1.5 sm:gap-2">

                <Users className="size-5 sm:size-4" />
                <span className="hidden whitespace-nowrap sm:inline">Friends</span>

                {friendRequestCount > 0 && (

                  <span className={cn(
                    "flex h-4 min-w-[1rem] items-center justify-center rounded-full px-1 text-[10px] font-semibold",
                    friendsActive ? "bg-surface text-foreground" : "bg-foreground text-surface"
                  )}>

                    {friendRequestCount}

                  </span>

                )}

              </span>

            </button>

          </nav>

        </div>

      </div>

    );

  }

}
