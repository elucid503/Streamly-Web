import { Component, type ReactNode } from "react";
import { Dices, Heart, Play, Radio, Sparkles } from "lucide-react";

import { HScrollRow } from "@/UI/HScrollRow";

import { lastWatched } from "@/Utils/History";
import type { FavoriteItem, MainView, WatchHistoryItem } from "@/Types";
import { cn } from "@/Utils/ClassNames";

export type ContextActionId = "continue" | "dice" | "shuffle-favorites";

interface ViewContextBarProps {

  view: MainView;

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

    if (view === "vod") {

      const actions: ContextAction[] = [];

      if (lastWatched(history, "vod")) {

        actions.push({

          id: "continue",
          label: "Continue Last",
          icon: <Play size={14} className="opacity-80" />,

        });

      }

      actions.push({

        id: "dice",
        label: "Pick For Me",
        icon: <Dices size={14} className="opacity-80" />,

      });

      if (favorites.some((item) => item.kind === "movie" || item.kind === "show")) {

        actions.push({

          id: "shuffle-favorites",
          label: "Lucky Favorite",
          icon: <Heart size={14} className="opacity-80" />,

        });

      }

      return actions;

    }

    if (view === "sports" || view === "friends") return [];

    const actions: ContextAction[] = [];

    if (lastWatched(history, "live")) {

      actions.push({

        id: "continue",
        label: "Last Channel",
        icon: <Radio size={14} className="opacity-80" />,

      });

    }

    actions.push({

      id: "dice",
      label: "Pick For Me",
      icon: <Dices size={14} className="opacity-80" />,

    });

    if (favorites.some((item) => item.kind === "live")) {

      actions.push({

        id: "shuffle-favorites",
        label: "Lucky Favorite",
        icon: <Sparkles size={14} className="opacity-80" />,

      });

    }

    return actions;

  };

  render() {

    const { loadingAction, onAction } = this.props;

    const actions = this.actionsForView();

    if (actions.length === 0) return null;

    return (

      <HScrollRow className="items-center gap-1.5">

        {actions.map((action) => {

          const loading = loadingAction === action.id;

          return (

            <button key={action.id} type="button"
              disabled={loadingAction !== null}

              onClick={() => onAction(action.id)}

              className={cn(

                "flex h-8 shrink-0 items-center gap-1.5 rounded-md border border-border bg-surface-overlay px-3 text-xs font-medium shadow-sm transition-all",
                "text-foreground-muted hover:bg-border/60 hover:text-foreground active:scale-[0.98]",
                loading && "pointer-events-none opacity-70",
                loadingAction !== null && !loading && "opacity-50"

              )}

            >

              <span className={cn("inline-flex shrink-0", loading && "animate-spin")}>

                {action.icon}

              </span>

              <span className="whitespace-nowrap">

                {loading ? (action.id === "dice" ? "Rolling..." : "Picking...") : action.label}

              </span>

            </button>

          );

        })}

      </HScrollRow>

    );

  }

}
