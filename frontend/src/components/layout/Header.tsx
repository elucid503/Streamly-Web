import { navigate } from "@/lib/navigation";
import { store } from "@/lib/store";
import type { MainView } from "@/lib/types";

import { ViewSwitcher } from "@/components/layout/ViewSwitcher";
import { Button } from "@/components/ui/Button";

import { Component } from "react";
import { LogOut, Settings, Shield } from "lucide-react";

interface HeaderProps {

  view: MainView;

  onViewChange: (view: MainView) => void;
  onOpenSettings: () => void;
  onOpenAdmin: () => void;
  onLogout: () => void;

}

export class Header extends Component<HeaderProps> {

  render() {

    const { view, onViewChange, onOpenSettings, onOpenAdmin, onLogout } = this.props;

    const user = store.user;

    return (

      <header className="fixed inset-x-0 top-0 z-50 border-b border-border-subtle bg-surface/80 backdrop-blur-md">

        <div className="relative mx-auto grid h-16 max-w-[1600px] grid-cols-[auto_1fr_auto] items-center gap-4 px-4 sm:px-8">

          <button type="button" onClick={() => navigate("/")} className="shrink-0" >

            <span className="text-sm font-semibold tracking-tight">

              Streamly <span className="font-light text-foreground-muted">Web</span>

            </span>

          </button>

          <div className="pointer-events-none absolute inset-x-0 flex justify-center px-20 sm:px-32">

            <div className="pointer-events-auto">

              <ViewSwitcher active={view} onChange={onViewChange} />

            </div>

          </div>

          <div className="flex shrink-0 items-center justify-end gap-1">

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

      </header>

    );

  }

}