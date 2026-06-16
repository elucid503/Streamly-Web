import { cn } from "@/lib/utils";
import { lastWatched } from "@/lib/history";
import type { FavoriteItem, MainView, WatchHistoryItem } from "@/lib/types";

import { Component, type ReactNode } from "react";
import { Dices, Heart, Play, Radio, Sparkles } from "lucide-react";

export type ContextActionId = "continue" | "dice" | "shuffle-favorites";

interface ViewContextBarProps {

  view: MainView;
  side?: "left" | "right";
  compact?: boolean;

  history: WatchHistoryItem[];
  favorites: FavoriteItem[];

  loadingAction: ContextActionId | null;

  onAction: (actionId: ContextActionId) => void;

}

interface ContextAction {

  id: ContextActionId;
  label: string;
  icon: ReactNode;

}

export class ViewContextBar extends Component<ViewContextBarProps> {

  actionsForView = (): ContextAction[] => {

    const { view, history, favorites } = this.props;

    if (view === "shows") {

      const actions: ContextAction[] = [];

      if (lastWatched(history, "show")) {

        actions.push({

          id: "continue",
          label: "Continue Last",
          icon: <Play size={13} className="opacity-80" />,

        });

      }

      actions.push({

        id: "dice",
        label: "Pick For Me",
        icon: <Dices size={13} className="opacity-80" />,

      });

      if (favorites.some((item) => item.kind === "show")) {

        actions.push({

          id: "shuffle-favorites",
          label: "Lucky Favorite",
          icon: <Heart size={13} className="opacity-80" />,

        });

      }

      return actions;

    }

    if (view === "movies") {

      const actions: ContextAction[] = [];

      if (lastWatched(history, "movie")) {

        actions.push({

          id: "continue",
          label: "Continue Last",
          icon: <Play size={13} className="opacity-80" />,

        });

      }

      actions.push({

        id: "dice",
        label: "Pick For Me",
        icon: <Dices size={13} className="opacity-80" />,

      });

      if (favorites.some((item) => item.kind === "movie")) {

        actions.push({

          id: "shuffle-favorites",
          label: "Lucky Favorite",
          icon: <Heart size={13} className="opacity-80" />,

        });

      }

      return actions;

    }

    const actions: ContextAction[] = [];

    if (lastWatched(history, "live")) {

      actions.push({

        id: "continue",
        label: "Last Channel",
        icon: <Radio size={13} className="opacity-80" />,

      });

    }

    actions.push({

      id: "dice",
      label: "Pick For Me",
      icon: <Dices size={13} className="opacity-80" />,

    });

    if (favorites.some((item) => item.kind === "live")) {

      actions.push({

        id: "shuffle-favorites",
        label: "Lucky Favorite",
        icon: <Sparkles size={13} className="opacity-80" />,

      });

    }

    return actions;

  };

  visibleActions = (): ContextAction[] => {

    const actions = this.actionsForView();

    if (this.props.compact) return actions;

    const splitAt = Math.ceil(actions.length / 2);

    if (this.props.side === "left") return actions.slice(0, splitAt);

    return actions.slice(splitAt);

  };

  render() {

    const { loadingAction, onAction, side, compact } = this.props;

    const actions = this.visibleActions();

    if (actions.length === 0) return null;

    return (

      <div className={cn(

        "flex min-w-0 items-center gap-1.5",
        compact ? "w-full gap-1" : "flex-wrap",
        !compact && side === "left" && "justify-center lg:justify-end",
        !compact && side === "right" && "justify-center lg:justify-start"

      )}>

        {actions.map((action) => {

          const loading = loadingAction === action.id;

          return (

            <button key={action.id} type="button"
              disabled={loadingAction !== null}

              onClick={() => onAction(action.id)}

              className={cn(

                "flex h-8 items-center gap-1 rounded-full border border-border-subtle bg-surface-raised text-[11px] font-medium shadow-sm transition-all lg:h-9 lg:gap-1.5 lg:px-3 lg:text-xs",
                compact ? "min-w-0 flex-1 justify-center px-2" : "shrink-0 px-2.5",
                "text-foreground-muted hover:border-border hover:text-foreground active:scale-95",
                loading && "pointer-events-none opacity-70",
                loadingAction !== null && !loading && "opacity-50"

              )}

            >

              <span className={cn("inline-flex shrink-0", loading && "animate-spin")}>

                {action.icon}

              </span>

              <span className={cn(compact ? "truncate" : "whitespace-nowrap")}>

                {loading ? (action.id === "dice" ? "Rolling..." : "Picking...") : action.label}

              </span>

            </button>

          );

        })}

      </div>

    );

  }

}
