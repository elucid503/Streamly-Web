import { PlayerMobileSheet } from "@/components/player/PlayerMobileSheet";

import { cn } from "@/lib/utils";
import type { LiveChannel } from "@/lib/types";

import { Component, createRef } from "react";
import { Check, LayoutGrid, Loader2, Search, X } from "lucide-react";

export const MULTIVIEW_MAX_STREAMS = 6;

interface MultiviewMenuProps {

  open: boolean;
  compact?: boolean;

  channels: LiveChannel[];
  selectedIds: string[];
  pendingIds: string[];
  primaryId?: string;
  loading?: boolean;

  onToggle: () => void;
  onClose: () => void;
  onOutsideClose?: (event: PointerEvent) => void;

  onSearch?: (query: string) => void;
  onToggleChannel?: (channel: LiveChannel) => void;

}

interface MultiviewMenuState {

  query: string;

}

export class MultiviewMenu extends Component<MultiviewMenuProps, MultiviewMenuState> {

  state: MultiviewMenuState = { query: "" };

  private rootRef = createRef<HTMLDivElement>();
  private searchTimer: ReturnType<typeof setTimeout> | null = null;

  componentDidUpdate(prev: MultiviewMenuProps) {

    if (this.props.open && !prev.open) {

      this.setState({ query: "" });
      this.props.onSearch?.("");

    }

    if (this.props.compact) return;

    if (this.props.open !== prev.open) {

      if (this.props.open) {

        document.addEventListener("pointerdown", this.handleOutsidePointerDown, true);

      } else {

        document.removeEventListener("pointerdown", this.handleOutsidePointerDown, true);

      }

    }

  }

  componentWillUnmount() {

    document.removeEventListener("pointerdown", this.handleOutsidePointerDown, true);

    if (this.searchTimer) clearTimeout(this.searchTimer);

  }

  handleOutsidePointerDown = (event: PointerEvent) => {

    const root = this.rootRef.current;

    if (!root || root.contains(event.target as Node)) return;

    event.preventDefault();
    event.stopPropagation();
    event.stopImmediatePropagation();

    this.props.onOutsideClose?.(event);
    this.props.onClose();

  };

  handleQuery = (value: string) => {

    this.setState({ query: value });

    if (this.searchTimer) clearTimeout(this.searchTimer);

    this.searchTimer = setTimeout(() => {

      this.props.onSearch?.(value);

    }, 200);

  };

  renderOption = (
    active: boolean,
    label: string,
    detail: string | undefined,
    onClick: () => void,
    key: string,
    disabled = false,
    loading = false,
  ) => (

    <button key={key} disabled={disabled} onClick={(e) => {

        e.stopPropagation();

        if (disabled) return;

        onClick();

      }} className={cn(

        "flex w-full items-center rounded-lg text-left transition-colors",
        this.props.compact ? "min-h-14 gap-3 px-4 py-3.5 text-sm" : "gap-2 px-2 py-1.5 text-sm",
        disabled && "cursor-not-allowed opacity-40",
        !disabled && (active || loading) && "bg-white/10 text-foreground",
        !disabled && !active && !loading && "text-foreground-muted hover:bg-white/6 hover:text-foreground"

      )} >

      {loading ? (

        <Loader2 size={this.props.compact ? 18 : 16} className="shrink-0 animate-spin text-foreground-muted" strokeWidth={2.5} />

      ) : (

        <span className={cn(

            "flex shrink-0 items-center justify-center rounded-full border transition-colors",
            this.props.compact ? "h-5 w-5" : "h-4 w-4",
            active ? "border-accent bg-accent text-black" : "border-white/20 bg-transparent"

          )}

        >

          {active && <Check size={this.props.compact ? 12 : 10} strokeWidth={3} />}

        </span>

      )}

      <div className="min-w-0 flex-1">

        <p className="truncate text-sm font-medium leading-tight">

          {label}

        </p>

        {detail && (

          <p className={cn("truncate leading-tight text-foreground-faint", this.props.compact ? "mt-0.5 text-xs" : "mt-0.5 text-[11px]")}>

            {detail}

          </p>

        )}

      </div>

    </button>

  );

  render() {

    const { open, compact, channels, selectedIds, pendingIds, primaryId, loading, onToggle, onClose, onToggleChannel } = this.props;
    const { query } = this.state;

    const selectedSet = new Set(selectedIds);
    const pendingSet = new Set(pendingIds);
    const atCap = 1 + selectedIds.length >= MULTIVIEW_MAX_STREAMS;

    const channelList = (

      <>
        {loading && channels.length === 0 && (

          <div className="flex items-center justify-center gap-2 py-10 text-sm text-foreground-muted">

            <Loader2 size={16} className="animate-spin" />
            Loading channels…

          </div>

        )}

        {!loading && channels.length === 0 && (

          <div className="px-4 py-10 text-center text-sm text-foreground-muted">

            No channels found

          </div>

        )}

        <div className={this.props.compact ? "grid grid-cols-1 gap-2" : "space-y-1"}>

          {channels.map((channel) => {

            const isPrimary = channel.id === primaryId;
            const selected = selectedSet.has(channel.id);
            const pending = pendingSet.has(channel.id);
            const active = isPrimary || selected;

            return this.renderOption(

              active,

              channel.name,
              channel.category || channel.country || undefined,

              () => {

                if (isPrimary) return;

                onToggleChannel?.(channel);

              },
              `mv-${channel.id}`,
              isPrimary || (!selected && !pending && atCap),
              pending

            );

          })}

        </div>
      </>

    );

    const search = (

      <div className="relative">

        <Search size={16} className="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-foreground-faint" />

        <input value={query} onChange={(e) => this.handleQuery(e.target.value)} placeholder="Search channels…" className="w-full rounded-lg border border-border-subtle bg-white/5 py-2.5 pr-3 pl-9 text-sm text-foreground placeholder:text-foreground-faint focus:border-white/20 focus:outline-none" />

      </div>

    );

    const trigger = (

      <button onClick={(e) => {

          e.stopPropagation();

          onToggle();

        }} className={cn(

          "flex size-9 shrink-0 items-center justify-center rounded-md text-foreground transition-colors hover:bg-white/15",
          open && "bg-white/15"

        )} aria-label="Multiview" >

        <LayoutGrid size={20} />

      </button>

    );

    if (compact) {

      return (

        <div className="relative">

          {trigger}

          <PlayerMobileSheet open={open} title="Multiview" icon={<LayoutGrid size={16} className="text-foreground-muted" />} onClose={onClose} headerExtra={search} >

            {channelList}

          </PlayerMobileSheet>

        </div>

      );

    }

    return (

      <div ref={this.rootRef} className="relative">

        {trigger}

        <div className={cn(

            "absolute right-0 bottom-full z-40 mb-3 w-72 origin-bottom-right overflow-hidden rounded-lg border border-border-subtle bg-surface/80 shadow-2xl shadow-black/40 backdrop-blur-xl transition-[opacity,transform] duration-200 ease-out",
            open ? "pointer-events-auto translate-y-0 scale-100 opacity-100" : "pointer-events-none translate-y-2 scale-95 opacity-0"

          )}

            onClick={(e) => e.stopPropagation()}

          >

            <div className="flex items-center justify-between px-3 py-2.5">

              <div className="flex items-center gap-2">

                <LayoutGrid size={14} className="text-foreground-muted" />

                <p className="text-sm font-medium text-foreground">

                  Multiview

                </p>

              </div>

              <button onClick={(e) => {

                  e.stopPropagation();

                  onClose();

                }} className="flex size-8 shrink-0 items-center justify-center rounded-md text-foreground-muted transition-colors hover:bg-white/10 hover:text-foreground" aria-label="Close multiview" >

                <X size={14} />

              </button>

            </div>

            <div className="px-3 pb-2">

              {search}

            </div>

            <div className="max-h-72 overflow-y-auto px-2 pb-2">

              {channelList}

            </div>

        </div>

      </div>

    );

  }

}