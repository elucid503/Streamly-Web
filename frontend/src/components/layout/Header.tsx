import { cn } from "@/lib/utils";
import { navigate } from "@/lib/navigation";
import { store } from "@/lib/store";
import type { MainView } from "@/lib/types";

import { ViewSwitcher } from "@/components/layout/ViewSwitcher";
import { Button } from "@/components/ui/Button";

import { Component } from "react";
import { createPortal } from "react-dom";
import { LogOut, Menu, Settings, Shield } from "lucide-react";
import { motion } from "framer-motion";

interface HeaderProps {

  view: MainView;

  onViewChange: (view: MainView) => void;
  onOpenSettings: () => void;
  onOpenAdmin: () => void;
  onLogout: () => void;

}

interface HeaderState {

  friendRequestCount: number;
  menuPos: { top: number; left: number } | null;

}

export class Header extends Component<HeaderProps, HeaderState> {

  private unsub = () => {};

  state: HeaderState = { friendRequestCount: store.incomingRequestCount, menuPos: null };

  componentDidMount() {

    this.unsub = store.subscribe(() => this.setState({ friendRequestCount: store.incomingRequestCount }));

  }

  componentWillUnmount() {

    this.unsub();

  }

  openMenu = (e: React.MouseEvent<HTMLButtonElement>) => {

    e.stopPropagation();

    if (this.state.menuPos) {

      this.setState({ menuPos: null });
      return;

    }

    const rect = e.currentTarget.getBoundingClientRect();
    const menuWidth = 172;

    this.setState({

      menuPos: {

        top: rect.bottom + 6,
        left: Math.max(8, rect.right - menuWidth),

      },

    });

  };

  closeMenu = () => this.setState({ menuPos: null });

  render() {

    const { view, onViewChange, onOpenSettings, onOpenAdmin, onLogout } = this.props;
    const { friendRequestCount, menuPos } = this.state;

    const user = store.user;

    return (

      <header className="fixed inset-x-0 top-0 z-50 border-b border-border-subtle bg-surface/80 pt-[env(safe-area-inset-top)] backdrop-blur-md">

        <div className="relative mx-auto grid h-16 max-w-[1600px] grid-cols-[auto_1fr_auto] items-center px-4 sm:gap-4 sm:px-8">

          <div className="shrink-0">

            <button type="button" onClick={() => navigate("/")} className="hidden sm:block">

              <span className="text-sm font-semibold tracking-tight">

                Streamly <span className="font-light text-foreground-muted">Web</span>

              </span>

            </button>

          </div>

          <div className="pointer-events-none absolute inset-x-0 flex justify-start pl-4 pr-12 sm:justify-center sm:px-20 lg:px-32">

            <div className="pointer-events-auto flex items-center gap-2">

              <ViewSwitcher active={view} onChange={onViewChange} />

              <button type="button" onClick={() => onViewChange("friends")}

                className={cn(

                  "relative flex h-8 items-center justify-center gap-1.5 rounded-full border border-border px-4 text-xs font-medium transition-colors sm:h-9 sm:px-6 sm:text-sm",
                  view === "friends" ? "bg-foreground text-surface" : "bg-surface-raised text-foreground-muted hover:text-foreground"

                )}

              >

                Friends

                {friendRequestCount > 0 && (

                  <span className={cn(

                    "flex h-4 min-w-[1rem] items-center justify-center rounded-full px-1 text-[10px] font-semibold",
                    view === "friends" ? "bg-surface text-foreground" : "bg-foreground text-surface"

                  )}>

                    {friendRequestCount}

                  </span>

                )}

              </button>

            </div>

          </div>

          <div className="flex shrink-0 items-center justify-end gap-1">

            <Button variant="ghost" size="sm" onClick={this.openMenu} title="Menu" className="sm:hidden hover:bg-transparent">

              <Menu size={15} />

            </Button>

            <div className="hidden sm:flex sm:items-center sm:gap-1">

              {user?.isAdmin && (

                <Button variant="ghost" size="sm" onClick={onOpenAdmin} title="Admin">

                  <Shield size={15} />

                </Button>

              )}

              <Button variant="ghost" size="sm" onClick={onOpenSettings} title="Settings">

                <Settings size={15} />

              </Button>

              <Button variant="ghost" size="sm" onClick={onLogout} title="Sign out">

                <LogOut size={15} />

              </Button>

            </div>

          </div>

        </div>

        {menuPos && createPortal(

          <>

            <div className="fixed inset-0 z-[99] sm:hidden" onClick={this.closeMenu} />

            <motion.div

              className="fixed z-[100] min-w-[172px] overflow-hidden rounded-[1.25rem] border border-white/10 bg-surface/70 p-1 shadow-2xl shadow-black/40 ring-1 ring-white/[0.04] backdrop-blur-xl backdrop-saturate-150 sm:hidden"

              style={{ top: menuPos.top, left: menuPos.left }}

              initial={{ opacity: 0, scale: 0.96, y: -6 }}
              animate={{ opacity: 1, scale: 1, y: 0 }}
              transition={{ type: "spring", stiffness: 500, damping: 32 }}

            >

              <button

                type="button"
                className="flex h-9 w-full items-center gap-2 rounded-xl px-3 text-left text-xs font-medium text-foreground-muted transition-colors hover:bg-surface-overlay/80 hover:text-foreground"

                onClick={(e) => {

                  e.stopPropagation();
                  this.closeMenu();
                  onOpenSettings();

                }}

              >

                <Settings size={13} />
                <span>Settings</span>

              </button>

              {user?.isAdmin && (

                <button

                  type="button"
                  className="flex h-9 w-full items-center gap-2 rounded-xl px-3 text-left text-xs font-medium text-foreground-muted transition-colors hover:bg-surface-overlay/80 hover:text-foreground"

                  onClick={(e) => {

                    e.stopPropagation();
                    this.closeMenu();
                    onOpenAdmin();

                  }}

                >

                  <Shield size={13} />
                  <span>Admin</span>

                </button>

              )}

              <button

                type="button"
                className="flex h-9 w-full items-center gap-2 rounded-xl px-3 text-left text-xs font-medium text-red-400 transition-colors hover:bg-surface-overlay/80 hover:text-red-300"

                onClick={(e) => {

                  e.stopPropagation();
                  this.closeMenu();
                  onLogout();

                }}

              >

                <LogOut size={13} />
                <span>Log Out</span>

              </button>

            </motion.div>

          </>,

          document.body

        )}

      </header>

    );

  }

}
