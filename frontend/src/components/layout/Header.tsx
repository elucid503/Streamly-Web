import { navigate } from "@/lib/navigation";
import { store } from "@/lib/store";

import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";

import { Component } from "react";
import { LogOut, Search, Settings, Shield } from "lucide-react";

interface HeaderProps {

  onSearch: (query: string) => void;
  onOpenSettings: () => void;
  onOpenAdmin: () => void;
  onLogout: () => void;

  searchQuery: string;

}

export class Header extends Component<HeaderProps> {

  render() {

    const { onSearch, onOpenSettings, onOpenAdmin, onLogout, searchQuery } = this.props;

    const user = store.user;

    return (

      <header className="sticky top-0 z-40 border-b border-border-subtle bg-surface/80 backdrop-blur-md">

        <div className="relative mx-auto grid h-16 max-w-[1600px] grid-cols-[auto_1fr_auto] items-center gap-4 px-4 sm:px-8">

          <button type="button" onClick={() => navigate("/")} className="flex shrink-0 items-baseline gap-1.5" >

            <span className="text-sm font-semibold tracking-tight">streamly</span>

            <span className="rounded border border-border px-1 py-px text-[9px] font-medium tracking-wider text-foreground-faint uppercase">Web</span>

          </button>

          <div className="pointer-events-none absolute inset-x-0 flex justify-center px-20 sm:px-32">

            <div className="pointer-events-auto relative w-full max-w-md">

              <Search size={16} className="absolute top-1/2 left-4 -translate-y-1/2 text-foreground-faint" />

              <Input className="h-10 w-full rounded-full pl-11 text-sm"

                value={searchQuery}
                onChange={(e) => onSearch(e.target.value)}
                placeholder="Search titles..."

              />

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
