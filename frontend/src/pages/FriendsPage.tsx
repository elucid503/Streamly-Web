import { api } from "@/api/client";

import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Modal } from "@/components/ui/Modal";
import { Switch } from "@/components/ui/Switch";

import { cn } from "@/lib/utils";
import { store } from "@/lib/store";

import type { FriendRequestItem, FriendSummary, ProfileMedia, PublicProfile, SearchHit, UserProfile } from "@/lib/types";

import { Component } from "react";
import { createPortal } from "react-dom";
import { AnimatePresence, motion } from "framer-motion";
import { Check, ChevronDown, Clock, Expand, Film, MoreHorizontal, Pencil, Plus, Search, Tv, UserCheck, UserMinus, UserPlus, Users, X, } from "lucide-react";

// Banner definitions

export const BANNERS: Record<string, string> = {

  aurora: "linear-gradient(135deg, #667eea 0%, #764ba2 100%)",
  sunset: "linear-gradient(135deg, #f6d365 0%, #fda085 55%, #f093fb 100%)",
  ocean: "linear-gradient(135deg, #43b89c 0%, #0099ff 50%, #2b5876 100%)",
  forest: "linear-gradient(135deg, #134e5e 0%, #71b280 100%)",
  midnight: "linear-gradient(135deg, #0f0c29 0%, #302b63 50%, #24243e 100%)",
  rose: "linear-gradient(135deg, #f857a6 0%, #ff5858 100%)",
  ember: "linear-gradient(135deg, #f7971e 0%, #ffd200 100%)",
  slate: "linear-gradient(135deg, #243b55 0%, #141e30 100%)",
  nebula: "linear-gradient(135deg, #ee0979 0%, #ff6a00 100%)",
  cosmos: "linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)",

};

export const ACCENT_COLORS = [ "#6366f1", "#8b5cf6", "#ec4899", "#ef4444", "#f97316", "#eab308", "#22c55e", "#14b8a6", "#3b82f6", "#94a3b8" ];

// Helpers

function initials(name: string): string {

  return name.split(/\s+/).slice(0, 2).map((w) => w[0]?.toUpperCase() ?? "").join("");

}

function bannerStyle(banner: string): React.CSSProperties {

  return { background: BANNERS[banner] ?? BANNERS.aurora };

}

function historyEpisodeSubtitle(item: PublicProfile["recentHistory"][number]): string | null {

  if (item.season == null || item.episode == null) return null;

  const base = `S${item.season} · E${item.episode}`;

  if (item.episodeTitle) return `${base} · ${item.episodeTitle}`;

  return base;

}

function timeAgo(iso: string): string {

  const ms = Date.now() - new Date(iso).getTime();
  const mins = Math.floor(ms / 60000);

  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;

  const hrs = Math.floor(mins / 60);

  if (hrs < 24) return `${hrs}h ago`;

  const days = Math.floor(hrs / 24);

  if (days < 7) return `${days}d ago`;

  return new Date(iso).toLocaleDateString();

}

// Avatar

interface AvatarProps {

  name: string;
  accentColor: string;

  size?: "sm" | "md" | "lg" | "xl";

}

function Avatar({ name, accentColor, size = "md" }: AvatarProps) {

  const sizeClass = {

    sm: "h-8 w-8 text-xs",
    md: "h-10 w-10 text-sm",
    lg: "h-14 w-14 text-base",
    xl: "h-20 w-20 text-xl",

  }[size];

  return (

    <div className={cn("flex flex-shrink-0 items-center justify-center rounded-full font-semibold text-white ring-2 ring-black/20", sizeClass)} style={{ backgroundColor: accentColor }}>

      {initials(name) || "?"}

    </div>

  );

}

// UserCard

interface UserCardProps {

  summary: FriendSummary;

  onAction: (summary: FriendSummary) => void;
  onViewProfile: (userId: string) => void;

  actionLoading: boolean;

}

function UserCard({ summary, onAction, onViewProfile, actionLoading }: UserCardProps) {

  const { friendStatus } = summary;

  const actionLabel = friendStatus === "friends" ? "Friends" : friendStatus === "pending_sent" ? "Requested" : friendStatus === "pending_received" ? "Accept" : "Add Friend";
  const ActionIcon = friendStatus === "friends" ? UserCheck : friendStatus === "pending_sent" ? Check : friendStatus === "pending_received" ? UserCheck : UserPlus;
  const actionVariant: "default" | "outline" = friendStatus === "none" || friendStatus === "pending_received" ? "default" : "outline";

  return (

    <div className="group flex items-center gap-3 rounded-xl border border-border bg-surface-raised p-3 transition-colors hover:border-border">

      <button type="button" className="flex flex-1 items-center gap-3 min-w-0" onClick={() => onViewProfile(summary.userId)}>

        <Avatar name={summary.displayName} accentColor={summary.accentColor} size="md" />

        <div className="min-w-0 text-left">

          <p className="truncate text-sm font-medium text-foreground">{summary.displayName}</p>

          <p className="truncate text-xs text-foreground-muted">{summary.email}</p>

        </div>

      </button>

      <Button className="flex-shrink-0 gap-1.5"

        variant={actionVariant}
        size="sm"

        disabled={actionLoading || friendStatus === "pending_sent"}
        onClick={(e) => { e.stopPropagation(); onAction(summary); }}

      >

        <ActionIcon size={13} />

        <span className="hidden sm:inline">{actionLabel}</span>

      </Button>

    </div>

  );

}

// RequestCard

interface RequestCardProps {

  request: FriendRequestItem;

  onAccept: (id: string) => void;
  onDecline: (id: string) => void;
  onViewProfile: (userId: string) => void;

  loading: boolean;

}

function RequestCard({ request, onAccept, onDecline, onViewProfile, loading }: RequestCardProps) {

  return (

    <div className="flex items-center gap-3 rounded-xl border border-border bg-surface-raised p-3">

      <button type="button" className="flex flex-1 items-center gap-3 min-w-0" onClick={() => onViewProfile(request.userId)}>

        <Avatar name={request.displayName} accentColor={request.accentColor} size="md" />

        <div className="min-w-0 text-left">

          <p className="truncate text-sm font-medium text-foreground">{request.displayName}</p>
          <p className="truncate text-xs text-foreground-muted">{request.email}</p>
          <p className="mt-0.5 text-xs text-foreground-faint">{timeAgo(request.createdAt)}</p>

        </div>

      </button>

      {request.direction === "incoming" ? (

        <div className="flex flex-shrink-0 gap-2">

          <Button size="sm" onClick={() => onAccept(request.id)} disabled={loading}>

            <Check size={13} />

            <span className="hidden sm:inline">Accept</span>

          </Button>

          <Button variant="outline" size="sm" onClick={() => onDecline(request.id)} disabled={loading}>

            <X size={13} />

          </Button>

        </div>

      ) : (

        <Button variant="outline" size="sm" onClick={() => onDecline(request.id)} disabled={loading}>

          <X size={13} />

          <span className="hidden sm:inline">Cancel</span>

        </Button>

      )}

    </div>

  );

}

// HistoryGroup

interface HistoryGroupProps {

  group: PublicProfile["recentHistory"];
  accentColor: string;

}

interface HistoryGroupState {

  expanded: boolean;

}

class HistoryGroup extends Component<HistoryGroupProps, HistoryGroupState> {

  state: HistoryGroupState = { expanded: false };

  toggle = () => this.setState({ expanded: !this.state.expanded });

  render() {

    const { group, accentColor } = this.props;
    const { expanded } = this.state;

    if (!group?.length) return null;

    const [main, ...extras] = group;

    return (

      <div>

        <HistoryRow item={main} accentColor={accentColor} showProgress progressExtra={extras.length > 0 ? (

            <button type="button" onClick={this.toggle} className="flex flex-shrink-0 items-center gap-0.5 text-[12px] text-foreground-faint transition-colors hover:text-foreground">

            <span>{expanded ? "see less" : `+${extras.length} more ${extras.length == 1 ? "episode" : "episodes"}`}</span>

              <ChevronDown size={10} className={cn("transition-transform duration-200", expanded && "rotate-180")} />

            </button>

          ) : undefined}
        />

        {extras.length > 0 && (

          <>

            <motion.div initial={false} animate={{ height: expanded ? "auto" : 0 }} transition={{ duration: 0.22, ease: [0.16, 1, 0.3, 1] }} style={{ overflow: "hidden" }} >

              <div className="mt-2 space-y-3">

                {extras.map((item) => (

                  <HistoryRow key={item.id} item={item} accentColor={accentColor} showProgress />

                ))}

              </div>

            </motion.div>

          </>

        )}

      </div>

    );

  }

}

// Friend profile popup

interface FriendProfilePopupProps {

  open: boolean;
  onClose: () => void;

  profile: PublicProfile | null;

  loading: boolean;

}

function FavoriteMediaCard({ item }: { item: ProfileMedia }) {

  return (

    <div className="overflow-hidden rounded-xl border border-border bg-surface-raised">

      <div className="aspect-[2/3] overflow-hidden bg-surface-overlay">

        {item.poster ? (

          <img src={item.poster} alt="" className="h-full w-full object-cover" />

        ) : (

          <div className="flex h-full items-center justify-center text-foreground-faint">

            {item.kind === "show" ? <Tv size={28} /> : <Film size={28} />}

          </div>

        )}

      </div>

      <div className="p-3">

        <p className="truncate text-sm font-medium text-foreground">{item.title}</p>

        {item.year ? <p className="text-xs text-foreground-faint">{item.year}</p> : null}

      </div>

    </div>

  );

}

function FriendProfilePopup({ open, onClose, profile, loading }: FriendProfilePopupProps) {

  const hasFavorites = profile != null && (profile.favoriteShows.length > 0 || profile.favoriteMovies.length > 0);

  return createPortal(

    <AnimatePresence>

      {open && (

        <motion.div className="fixed inset-0 z-50 flex items-center justify-center p-4" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }} >

          <div className="absolute inset-0 bg-surface/60 backdrop-blur-md" onClick={onClose} />

          <motion.div className="relative z-10 flex max-h-[min(90vh,720px)] w-full max-w-2xl flex-col overflow-hidden rounded-2xl border border-border-subtle bg-surface/90 shadow-2xl backdrop-blur-md"

            initial={{ opacity: 0, scale: 0.96, y: 8 }}
            exit={{ opacity: 0, scale: 0.96, y: 8 }}

            animate={{ opacity: 1, scale: 1, y: 0 }}
            transition={{ type: "spring", stiffness: 400, damping: 30 }}

          >

            <button type="button" onClick={onClose} className="absolute right-4 top-4 z-20 rounded-full bg-surface/80 p-1.5 text-foreground-muted backdrop-blur-sm transition-colors hover:bg-surface-overlay hover:text-foreground">

              <X size={16} />

            </button>

            <div className="overflow-y-auto">

              {loading && (

                <div className="flex items-center justify-center py-20">

                  <div className="h-8 w-8 animate-spin rounded-full border-2 border-border border-t-foreground-muted" />

                </div>

              )}

              {!loading && profile && (

                <div className="space-y-5 pb-5">

                  <ProfileBanner banner={profile.banner} accentColor={profile.accentColor} displayName={profile.displayName} bio={profile.bio} embedded />

                  <div className="space-y-5 px-6">

                    {hasFavorites ? (

                      <>

                        {profile.favoriteShows.length > 0 && (

                          <div>

                            <p className="mb-3 text-xs font-medium uppercase tracking-wide text-foreground-muted">

                              Favourite Shows

                            </p>

                            <div className="grid grid-cols-4 gap-4">

                              {profile.favoriteShows.map((m) => (

                                <FavoriteMediaCard key={m.mediaId} item={m} />

                              ))}

                            </div>

                          </div>

                        )}

                        {profile.favoriteMovies.length > 0 && (

                          <div>

                            <p className="mb-3 text-xs font-medium uppercase tracking-wide text-foreground-muted">

                              Favourite Movies

                            </p>

                            <div className="grid grid-cols-4 gap-4">

                              {profile.favoriteMovies.map((m) => (

                                <FavoriteMediaCard key={m.mediaId} item={m} />

                              ))}

                            </div>

                          </div>

                        )}

                      </>

                    ) : (

                      <p className="py-2 text-center text-sm text-foreground-muted">No favourites shared yet.</p>

                    )}

                  </div>

                </div>

              )}

              {!loading && !profile && (

                <p className="py-12 text-center text-sm text-foreground-muted">Could not load profile.</p>

              )}

            </div>

          </motion.div>

        </motion.div>

      )}

    </AnimatePresence>,

    document.body

  );

}

// ExpandableFriendCard

interface ExpandableFriendCardProps {

  summary: FriendSummary;
  onRemove: (userId: string) => void;
  removeLoading: boolean;

}

interface ExpandableFriendCardState {

  expanded: boolean;
  profileOpen: boolean;
  profile: PublicProfile | null;
  profileLoading: boolean;
  menuPos: { top: number; left: number } | null;

}

interface HistoryRowProps {

  item: PublicProfile["recentHistory"][number];
  accentColor: string;
  showProgress?: boolean;
  progressExtra?: React.ReactNode;

}

function HistoryRow({ item, accentColor, showProgress, progressExtra }: HistoryRowProps) {

  const progress = item.durationMs > 0 ? Math.min(1, item.positionMs / item.durationMs) : 0;
  const episodeSubtitle = historyEpisodeSubtitle(item);

  return (

    <div className="flex items-stretch gap-3">

      {item.poster ? (

        <img src={item.poster} alt="" className="h-20 w-[52px] flex-shrink-0 rounded-lg object-cover" />

      ) : (

        <div className="h-20 w-[52px] flex-shrink-0 rounded-lg bg-surface-overlay" />

      )}

      <div className="flex min-w-0 flex-1 flex-col">

        <div className="flex min-w-0 flex-1 items-start justify-between gap-2">

          <div className="min-w-0">

            <p className="truncate text-sm font-medium text-foreground">{item.title}</p>

            {episodeSubtitle && (

              <p className="truncate text-xs text-foreground-faint">{episodeSubtitle}</p>

            )}

          </div>

          <span className="flex-shrink-0 text-xs text-foreground-faint">{timeAgo(item.updatedAt)}</span>

        </div>

        {showProgress && (

          <div className="mt-auto flex items-center gap-2 pb-2">

            {!item.completed && item.durationMs > 0 ? (

              <div className="h-0.5 flex-1 rounded-full bg-surface-overlay">

                <div
                  className="h-full rounded-full"
                  style={{ width: `${progress * 100}%`, backgroundColor: accentColor }}
                />

              </div>

            ) : item.completed ? (

              <p className="flex-1 text-xs text-foreground-faint">Watched</p>

            ) : (

              <div className="flex-1" />

            )}

            {progressExtra}

          </div>

        )}

      </div>

    </div>

  );

}

class ExpandableFriendCard extends Component<ExpandableFriendCardProps, ExpandableFriendCardState> {

  state: ExpandableFriendCardState = {

    expanded: false,

    profileOpen: false,

    profile: null,
    profileLoading: false,

    menuPos: null,

  };

  componentDidMount() {

    void this.loadProfile();

  }

  loadProfile = async () => {

    this.setState({ profileLoading: true });

    try {

      const profile = await api.getPublicProfile(this.props.summary.userId);

      this.setState({ profile, profileLoading: false });

    } catch {

      this.setState({ profileLoading: false });

    }

  };

  toggle = () => this.setState({ expanded: !this.state.expanded });

  openProfile = () => this.setState({ profileOpen: true });

  closeProfile = () => this.setState({ profileOpen: false });

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
        left: Math.min(rect.left, window.innerWidth - menuWidth - 8),

      },

    });

  };

  closeMenu = () => this.setState({ menuPos: null });

  render() {

    const { summary, onRemove, removeLoading } = this.props;
    const { expanded, profileOpen, profile, profileLoading, menuPos } = this.state;

    const firstItem = profile?.recentHistory?.[0] ?? null;

    return (

      <>

        <div className="overflow-hidden rounded-2xl border border-border bg-surface-raised">

          {/* Header */}

          <div className="flex items-center gap-3 px-4 py-3.5">

            <button type="button" onClick={this.openProfile} className="flex min-w-0 flex-1 items-center gap-3 text-left transition-opacity hover:opacity-80">

              <Avatar name={summary.displayName} accentColor={summary.accentColor} size="md" />

              <div className="min-w-0 flex-1">

                <p className="truncate text-sm font-semibold text-foreground">{summary.displayName}</p>
                <p className="truncate text-xs text-foreground-faint">{summary.email}</p>

              </div>

            </button>

            <button type="button" onClick={this.openMenu} className="flex-shrink-0 rounded-full p-1.5 text-foreground-faint transition-colors hover:bg-surface-overlay hover:text-foreground">

              <MoreHorizontal size={15} />

            </button>

            <button type="button" onClick={this.toggle} className="flex-shrink-0 rounded-full p-1.5 text-foreground-faint transition-colors hover:bg-surface-overlay hover:text-foreground">

              <ChevronDown size={15} className={cn("transition-transform duration-200", expanded && "rotate-180")} />

            </button>

          </div>

          {/* First item */}

          {profileLoading && (

            <div className="flex items-center gap-3 border-t border-border/40 px-4 py-3">

              <div className="h-20 w-[52px] flex-shrink-0 animate-pulse rounded-lg bg-surface-overlay" />

              <div className="flex-1 space-y-2">

                <div className="h-3 w-3/4 animate-pulse rounded bg-surface-overlay" />
                <div className="h-2 w-1/2 animate-pulse rounded bg-surface-overlay" />

              </div>

            </div>

          )}

          {!profileLoading && firstItem && (

            <div className="border-t border-border/40 px-4 py-3">

              <HistoryRow item={firstItem} accentColor={summary.accentColor} showProgress />

            </div>

          )}

          {/* Expanded body */}

          <motion.div initial={false} animate={{ height: expanded ? "auto" : 0 }} transition={{ duration: 0.26, ease: [0.16, 1, 0.3, 1] }} style={{ overflow: "hidden" }}>

            <div className="space-y-4 border-t border-border/40 px-4 pb-4 pt-3">

              {profileLoading && (

                <div className="flex items-center justify-center py-4">

                  <div className="h-5 w-5 animate-spin rounded-full border-2 border-border border-t-foreground-muted" />

                </div>

              )}

              {!profileLoading && profile && (profile.recentHistory?.length ?? 0) > 1 && (() => {

                const groups = new Map<number, PublicProfile["recentHistory"]>();

                for (const item of (profile.recentHistory ?? []).slice(1)) {

                  const g = groups.get(item.mediaId);
                  if (g) { g.push(item); } else { groups.set(item.mediaId, [item]); }

                }

                return (

                  <div className="space-y-3">

                    {Array.from(groups.values()).map((group) => (

                      <HistoryGroup key={group[0].id} group={group} accentColor={summary.accentColor} />

                    ))}

                  </div>

                );

              })()}

            </div>

          </motion.div>

        </div>

        <FriendProfilePopup open={profileOpen} onClose={this.closeProfile} profile={profile} loading={profileLoading}/>

        {/* Context menu */}

        {menuPos && createPortal(

          <>

            <div className="fixed inset-0 z-[99]" onClick={this.closeMenu} />

            <motion.div className="fixed z-[100] min-w-[172px] overflow-hidden rounded-[1.25rem] border border-white/10 bg-surface/70 p-1 shadow-2xl shadow-black/40 ring-1 ring-white/[0.04] backdrop-blur-xl backdrop-saturate-150"

              style={{ top: menuPos.top, left: menuPos.left }}

              initial={{ opacity: 0, scale: 0.96, y: -6 }}
              animate={{ opacity: 1, scale: 1, y: 0 }}
              transition={{ type: "spring", stiffness: 500, damping: 32 }}

            >

              <button className="flex h-9 w-full items-center gap-2 rounded-xl px-3 text-left text-xs font-medium text-foreground-muted transition-colors hover:bg-surface-overlay/80 hover:text-foreground disabled:opacity-40"

                type="button"
                disabled={removeLoading}

                onClick={(e) => {

                  e.stopPropagation();
                  this.closeMenu();

                  this.openProfile();

                }}

              >

                <Expand size={13} />

                <span>View Profile</span>

              </button>

              <button className="flex h-9 w-full items-center gap-2 rounded-xl px-3 text-left text-xs font-medium text-red-400 transition-colors hover:bg-surface-overlay/80 hover:text-red-300 disabled:opacity-40"

                type="button"
                disabled={removeLoading}

                onClick={(e) => {

                  e.stopPropagation();
                  this.closeMenu();
                  onRemove(summary.userId);

                }}

              >


                <UserMinus size={13} />

                <span>Remove Friend</span>

              </button>

            </motion.div>

          </>,

          document.body

        )}

      </>

    );

  }

}

// ProfileBanner (large profile display)

interface ProfileBannerProps {

  banner: string;
  accentColor: string;

  displayName: string;

  bio: string;

  isOwn?: boolean;
  embedded?: boolean;

  onEdit?: () => void;

  extra?: React.ReactNode;

}

function ProfileBanner({ banner, accentColor, displayName, bio, isOwn, onEdit, extra, embedded }: ProfileBannerProps) {

  return (

    <div className={cn(

      "overflow-hidden border border-border bg-surface-raised",
      embedded ? "rounded-b-none border-x-0 border-t-0" : "rounded-2xl"

    )}>

      <div className="relative h-32 w-full sm:h-40" style={bannerStyle(banner)} />

      <div className="px-5 pb-5 z-10 relative">

        <div className="flex items-end justify-between gap-3 -mt-7">

          <div className="ring-4 ring-surface-raised rounded-full">

            <Avatar name={displayName} accentColor={accentColor} size="xl" />

          </div>

          <div className="pb-1 flex gap-2">

            {isOwn && onEdit && (

              <Button variant="outline" size="sm" onClick={onEdit}>

                <Pencil size={13} />

                Edit Profile

              </Button>

            )}

            {extra}

          </div>

        </div>

        <div className="mt-3">

          <h2 className="text-lg font-semibold text-foreground">{displayName}</h2>

          {bio && <p className="mt-1 text-sm text-foreground-muted leading-relaxed">{bio}</p>}

        </div>

      </div>

    </div>

  );

}

// MediaPicker

interface MediaPickerProps {

  open: boolean;
  kind: "movies" | "shows";

  selected: ProfileMedia[];

  onClose: () => void;
  onSelect: (item: ProfileMedia) => void;
  onRemove: (mediaId: number) => void;

}

interface MediaPickerState {

  query: string;
  results: SearchHit[];

  searching: boolean;

}

class MediaPicker extends Component<MediaPickerProps, MediaPickerState> {

  private debounce: ReturnType<typeof setTimeout> | null = null;

  state: MediaPickerState = {

    query: "",
    results: [],
    searching: false,

  };

  componentWillUnmount() {

    if (this.debounce) clearTimeout(this.debounce);

  }

  handleSearch = (q: string) => {

    this.setState({ query: q });

    if (this.debounce) clearTimeout(this.debounce);

    if (!q.trim()) {

      this.setState({ results: [], searching: false });
      return;

    }

    this.setState({ searching: true });

    this.debounce = setTimeout(async () => {

      try {

        const all = await api.search(q);
        const kind = this.props.kind === "movies" ? "movie" : "show";

        this.setState({ results: (all ?? []).filter((h) => h.kind === kind), searching: false });

      } catch {

        this.setState({ results: [], searching: false });

      }

    }, 350);

  };

  render() {

    const { open, kind, selected, onClose, onSelect, onRemove } = this.props;
    const { query, results, searching } = this.state;

    const kindLabel = kind === "movies" ? "Movie" : "Show";

    return (

      <Modal open={open} onClose={() => { this.setState({ query: "", results: [] }); onClose() }} title={`Pick Favorite ${kindLabel}s`}>

        <div className="space-y-4">

          {selected.length > 0 && (

            <div className="flex flex-wrap gap-2">

              {selected.map((item) => (

                <div key={item.mediaId} className="flex items-center gap-1.5 rounded-full border border-border bg-surface-raised py-1 pl-3 pr-1.5 text-xs">

                  <span className="max-w-[120px] truncate">{item.title}</span>

                  <button type="button" onClick={() => onRemove(item.mediaId)} className="rounded-full p-0.5 text-foreground-faint hover:text-foreground">

                    <X size={11} />

                  </button>

                </div>

              ))}

            </div>

          )}

          {selected.length < 3 && (

            <div className="relative">

              <Search size={14} className="absolute top-1/2 left-3 -translate-y-1/2 text-foreground-faint" />

              <Input className="w-full pl-9" placeholder={`Search ${kind}...`} value={query} onChange={(e) => this.handleSearch(e.target.value)} />

            </div>

          )}

          {searching && (

            <p className="py-4 text-center text-sm text-foreground-muted">Searching...</p>

          )}

          {!searching && results.length > 0 && selected.length < 3 && (

            <div className="max-h-64 space-y-1 overflow-y-auto">

              {results.map((hit) => {

                const already = selected.some((s) => s.mediaId === hit.id);

                return (

                  <button className="flex w-full items-center gap-3 rounded-lg p-2 text-left transition-colors hover:bg-surface-overlay disabled:opacity-40"

                    key={hit.id}
                    type="button"

                    disabled={already}

                    onClick={() => onSelect({ mediaId: hit.id, title: hit.title, poster: hit.poster, year: hit.year, kind: hit.kind })}

                  >

                    {hit.poster ? (

                      <img src={hit.poster} alt="" className="h-10 w-7 flex-shrink-0 rounded object-cover" />

                    ) : (

                      <div className="h-10 w-7 flex-shrink-0 rounded bg-surface-overlay" />

                    )}

                    <div className="min-w-0">

                      <p className="truncate text-sm text-foreground">{hit.title}</p>

                      <p className="text-xs text-foreground-faint">{hit.year}</p>

                    </div>

                    {already && <Check size={13} className="ml-auto flex-shrink-0 text-foreground-muted" />}

                  </button>

                );

              })}

            </div>

          )}

          {!searching && query && results.length === 0 && (

            <p className="py-4 text-center text-sm text-foreground-muted">No results</p>

          )}

          <p className="text-xs text-foreground-faint">Select up to 3 favorites</p>

        </div>

      </Modal>

    );

  }

}

// ProfileEditor

interface EditForm {

  displayName: string;
  bio: string;

  accentColor: string;
  banner: string;

  favoriteMovies: ProfileMedia[];
  favoriteShows: ProfileMedia[];

  historyVisible: boolean;
  discoverVisible: boolean;

}

interface ProfileEditorProps {

  open: boolean;
  saving: boolean;

  initial: EditForm;

  onClose: () => void;
  onSave: (form: EditForm) => void;

}

interface ProfileEditorState {

  form: EditForm;
  mediaPickerFor: "movies" | "shows" | null;

}

class ProfileEditor extends Component<ProfileEditorProps, ProfileEditorState> {

  constructor(props: ProfileEditorProps) {

    super(props);

    this.state = {

      form: { ...props.initial },
      mediaPickerFor: null,

    };

  }

  componentDidUpdate(prev: ProfileEditorProps) {

    if (!prev.open && this.props.open) {

      this.setState({ form: { ...this.props.initial } });

    }

  }

  setForm = (patch: Partial<EditForm>) => {

    this.setState({ form: { ...this.state.form, ...patch } });

  };

  handleMediaSelect = (item: ProfileMedia) => {

    const { mediaPickerFor, form } = this.state;

    if (!mediaPickerFor) return;

    if (mediaPickerFor === "shows") {

      if (form.favoriteShows.some((m) => m.mediaId === item.mediaId) || form.favoriteShows.length >= 3) return;

      this.setForm({ favoriteShows: [...form.favoriteShows, item] });

    } else {

      if (form.favoriteMovies.some((m) => m.mediaId === item.mediaId) || form.favoriteMovies.length >= 3) return;

      this.setForm({ favoriteMovies: [...form.favoriteMovies, item] });

    }

  };

  handleMediaRemove = (mediaId: number) => {

    const { mediaPickerFor, form } = this.state;

    if (!mediaPickerFor) return;

    if (mediaPickerFor === "shows") {

      this.setForm({ favoriteShows: form.favoriteShows.filter((m) => m.mediaId !== mediaId) });

    } else {

      this.setForm({ favoriteMovies: form.favoriteMovies.filter((m) => m.mediaId !== mediaId) });

    }

  };

  render() {

    const { open, saving, onClose, onSave } = this.props;
    const { form, mediaPickerFor } = this.state;

    return (

      <>

        <Modal open={open} onClose={onClose} title="Edit Profile">

          <div className="space-y-5">

            <div className="space-y-1.5">

              <label className="text-xs font-medium text-foreground-muted uppercase tracking-wide">Display Name</label>
              <Input value={form.displayName} onChange={(e) => this.setForm({ displayName: e.target.value })} placeholder="Your name" maxLength={32} />

            </div>

            <div className="space-y-1.5 relative">

              <label className="text-xs font-medium text-foreground-muted uppercase tracking-wide">Bio</label>

              <textarea className="w-full resize-none rounded-md border border-border bg-surface-raised px-3 py-2 text-sm text-foreground placeholder:text-foreground-faint focus:border-border focus:outline-none focus:ring-1 focus:ring-accent"

                value={form.bio}
                onChange={(e) => this.setForm({ bio: e.target.value })}

                placeholder="A short description about yourself…"

                maxLength={160}
                rows={2}

              />

              <p className="absolute bottom-2 right-2 text-right text-xs text-foreground-faint">{form.bio.length}/160</p>

            </div>

            <div className="space-y-2">

              <label className="text-xs font-medium text-foreground-muted uppercase tracking-wide">Accent Color</label>

              <div className="flex flex-wrap gap-2">

                {ACCENT_COLORS.map((color) => (

                  <button className="h-7 w-7 rounded-full transition-transform hover:scale-110"

                    key={color}
                    type="button"

                    title={color}

                    onClick={() => this.setForm({ accentColor: color })}

                    style={{ backgroundColor: color, outline: form.accentColor === color ? `2px solid ${color}` : "2px solid transparent", outlineOffset: "2px" }}

                  />

                ))}

              </div>

            </div>

            <div className="space-y-2">

              <label className="text-xs font-medium text-foreground-muted uppercase tracking-wide">Banner</label>

              <div className="grid grid-cols-5 gap-2">

                {Object.entries(BANNERS).map(([id, grad]) => (

                  <button className={cn( "h-10 rounded-lg transition-all hover:scale-105", form.banner === id && "ring-2 ring-white ring-offset-2 ring-offset-surface" )}

                    key={id}
                    type="button"

                    title={id}

                    onClick={() => this.setForm({ banner: id })}

                    style={{ background: grad }}

                  />

                ))}

              </div>

            </div>

            <div className="space-y-2">

              <label className="text-xs font-medium text-foreground-muted uppercase tracking-wide">Favorite Shows</label>

              <div className="flex flex-wrap gap-2">

                {form.favoriteShows.map((item) => (

                  <div key={item.mediaId} className="flex items-center gap-1.5 rounded-full border border-border bg-surface-raised py-1 pl-3 pr-1.5 text-xs">

                    <span className="max-w-[110px] truncate">{item.title}</span>

                    <button type="button" onClick={() => this.setForm({ favoriteShows: form.favoriteShows.filter((m) => m.mediaId !== item.mediaId) })} className="rounded-full p-0.5 text-foreground-faint hover:text-foreground">

                      <X size={11} />

                    </button>

                  </div>

                ))}

                {form.favoriteShows.length < 3 && (

                  <button type="button" onClick={() => this.setState({ mediaPickerFor: "shows" })} className="flex items-center gap-1 rounded-full border border-dashed border-border py-1 pl-2 pr-3 text-xs text-foreground-muted hover:text-foreground">

                    <Plus size={11} />

                    Add Show

                  </button>

                )}

              </div>

            </div>

            <div className="space-y-2">

              <label className="text-xs font-medium text-foreground-muted uppercase tracking-wide">Favorite Movies</label>

              <div className="flex flex-wrap gap-2">

                {form.favoriteMovies.map((item) => (

                  <div key={item.mediaId} className="flex items-center gap-1.5 rounded-full border border-border bg-surface-raised py-1 pl-3 pr-1.5 text-xs">

                    <span className="max-w-[110px] truncate">{item.title}</span>

                    <button type="button" onClick={() => this.setForm({ favoriteMovies: form.favoriteMovies.filter((m) => m.mediaId !== item.mediaId) })} className="rounded-full p-0.5 text-foreground-faint hover:text-foreground">

                      <X size={11} />

                    </button>

                  </div>

                ))}

                {form.favoriteMovies.length < 3 && (

                  <button type="button" onClick={() => this.setState({ mediaPickerFor: "movies" })} className="flex items-center gap-1 rounded-full border border-dashed border-border py-1 pl-2 pr-3 text-xs text-foreground-muted hover:text-foreground">

                    <Plus size={11} />

                    Add Movie

                  </button>

                )}

              </div>

            </div>

            <Switch label="Show watch history to friends" checked={form.historyVisible} onChange={(v) => this.setForm({ historyVisible: v })} className="pb-0" />

            <Switch label="Appear in Discover" checked={form.discoverVisible} onChange={(v) => this.setForm({ discoverVisible: v })} />

            <div className="flex gap-2 pt-1">

              <Button className="flex-1" onClick={() => onSave(form)} disabled={saving}>

                {saving ? "Saving…" : "Save Changes"}

              </Button>

              <Button variant="outline" onClick={onClose}>

                Cancel

              </Button>

            </div>

          </div>

        </Modal>

        <MediaPicker

          open={mediaPickerFor !== null}
          kind={mediaPickerFor ?? "movies"}

          selected={mediaPickerFor === "shows" ? form.favoriteShows : form.favoriteMovies}

          onClose={() => this.setState({ mediaPickerFor: null })}
          onSelect={this.handleMediaSelect}
          onRemove={this.handleMediaRemove}

        />

      </>

    );

  }

}

// PublicProfileView

interface PublicProfileViewProps {

  open: boolean;

  userId: string | null;

  onClose: () => void;
  onFriendStatusChange: () => void;

}

interface PublicProfileViewState {

  profile: PublicProfile | null;

  loading: boolean;
  actionLoading: boolean;

}

class PublicProfileView extends Component<PublicProfileViewProps, PublicProfileViewState> {

  state: PublicProfileViewState = {

    profile: null,

    loading: false,
    actionLoading: false,

  };

  componentDidUpdate(prev: PublicProfileViewProps) {

    if (this.props.userId && this.props.userId !== prev.userId) {

      void this.load(this.props.userId);

    }

    if (!this.props.userId && prev.userId) {

      this.setState({ profile: null });

    }

  }

  load = async (userId: string) => {

    this.setState({ loading: true, profile: null });

    try {

      const profile = await api.getPublicProfile(userId);

      this.setState({ profile });

    } catch {

      this.setState({ profile: null });

    } finally {

      this.setState({ loading: false });

    }

  };

  handleAction = async () => {

    const { profile } = this.state;

    if (!profile) return;

    this.setState({ actionLoading: true });

    try {

      if (profile.friendStatus === "none") {

        await api.sendFriendRequest(profile.userId);

        this.setState({ profile: { ...profile, friendStatus: "pending_sent" } });

      } else if (profile.friendStatus === "friends") {

        await api.removeFriend(profile.userId);

        this.setState({ profile: { ...profile, friendStatus: "none" } });

      }

      this.props.onFriendStatusChange();

    } catch {

      /* ignore */

    } finally {

      this.setState({ actionLoading: false });

    }

  };

  render() {

    const { open, onClose } = this.props;
    const { profile, loading, actionLoading } = this.state;

    const actionLabel = profile?.friendStatus === "friends" ? "Remove Friend"  : profile?.friendStatus === "pending_sent" ? "Request Sent" : profile?.friendStatus === "pending_received" ? "Accept Request" : "Add Friend";
    const canAct = profile?.friendStatus === "none" || profile?.friendStatus === "friends";

    return (

      <Modal open={open} onClose={onClose} title="">

        {loading && (

          <div className="flex items-center justify-center py-12">

            <div className="h-8 w-8 animate-spin rounded-full border-2 border-border border-t-foreground-muted" />

          </div>

        )}

        {!loading && !profile && (

          <p className="py-8 text-center text-sm text-foreground-muted">Could not load profile.</p>

        )}

        {!loading && profile && (

          <div className="space-y-5">

            <ProfileBanner

              banner={profile.banner}
              accentColor={profile.accentColor}

              displayName={profile.displayName}
              bio={profile.bio}

              extra={

                <Button variant={profile.friendStatus === "none" ? "default" : "outline"} size="sm" disabled={actionLoading || !canAct} onClick={this.handleAction}>

                  {profile.friendStatus === "friends" ? <UserMinus size={13} /> : <UserPlus size={13} />}
                  {actionLabel}

                </Button>

              }
            />

            {(profile.favoriteShows.length > 0 || profile.favoriteMovies.length > 0) && (

              <div className="space-y-3">

                {profile.favoriteShows.length > 0 && (

                  <div>

                    <p className="mb-2 flex items-center gap-1.5 text-xs font-medium uppercase tracking-wide text-foreground-muted">

                      <Tv size={12} />

                      Favorite Shows

                    </p>

                    <div className="flex flex-wrap gap-2">

                      {profile.favoriteShows.map((m) => (

                        <FavoriteMediaChip key={m.mediaId} item={m} />

                      ))}

                    </div>

                  </div>

                )}

                {profile.favoriteMovies.length > 0 && (

                  <div>

                    <p className="mb-2 flex items-center gap-1.5 text-xs font-medium uppercase tracking-wide text-foreground-muted">

                      <Film size={12} />

                      Favorite Movies

                    </p>

                    <div className="flex flex-wrap gap-2">

                      {profile.favoriteMovies.map((m) => (

                        <FavoriteMediaChip key={m.mediaId} item={m} />

                      ))}

                    </div>

                  </div>

                )}

              </div>

            )}

            {profile.recentHistory && profile.recentHistory.length > 0 && (

              <div>

                <p className="mb-2 flex items-center gap-1.5 text-xs font-medium uppercase tracking-wide text-foreground-muted">

                  <Clock size={12} />

                  Recently Watched

                </p>

                <div className="space-y-1.5">

                  {profile.recentHistory.slice(0, 5).map((item) => (

                    <div key={item.id} className="flex items-center gap-2.5 rounded-lg border border-border bg-surface-raised px-3 py-2">

                      {item.poster && (

                        <img src={item.poster} alt="" className="h-9 w-6 flex-shrink-0 rounded object-cover" />

                      )}

                      <div className="min-w-0">

                        <p className="truncate text-sm text-foreground">{item.title}</p>

                        {item.season != null && item.episode != null && (

                          <p className="text-xs text-foreground-faint">S{item.season} E{item.episode}</p>

                        )}

                      </div>

                      <span className="ml-auto flex-shrink-0 text-xs text-foreground-faint">{timeAgo(item.updatedAt)}</span>

                    </div>

                  ))}

                </div>

              </div>

            )}

          </div>

        )}

      </Modal>

    );

  }

}

function FavoriteMediaChip({ item }: { item: ProfileMedia }) {

  return (

    <div className="flex items-center gap-1.5 rounded-full border border-border bg-surface-raised py-1 pl-1.5 pr-3 text-xs">

      {item.poster && (

        <img src={item.poster} alt="" className="h-5 w-3.5 rounded object-cover" />

      )}

      <span className="max-w-[120px] truncate text-foreground">{item.title}</span>

      {item.year && <span className="text-foreground-faint">({item.year})</span>}

    </div>

  );

}

// SideCollage

const COLLAGE_ROTATIONS = [-2.2, 1.5, -0.8, 2.4, -1.6, 0.7, -2.8, 1.1, -0.4, 2.0, -1.3, 0.9];

interface SideCollageProps {

  posters: string[];
  side: "left" | "right";

}

function SideCollage({ posters, side }: SideCollageProps) {

  if (posters.length === 0) return null;

  const isLeft = side === "left";

  return (

    <div className={cn("pointer-events-none fixed top-0 bottom-0 z-0 hidden lg:block overflow-hidden", isLeft ? "left-0" : "right-0", "w-56 xl:w-72 2xl:w-80" )}>

      {/* Poster grid */}

      <div className="grid grid-cols-2 gap-1.5 p-1.5 h-full content-start">

        {posters.slice(0, 24).map((poster, i) => (

          <div className="overflow-hidden rounded-sm"

            key={`${poster}-${i}`}

            style={{

              opacity: 0.1,

              filter: "saturate(0.55)",
              transform: `rotate(${COLLAGE_ROTATIONS[i % COLLAGE_ROTATIONS.length]}deg)`,

            }}

          >

            <img src={poster} alt="" className="w-full aspect-[2/3] object-cover block" loading="lazy" />

          </div>

        ))}

      </div>

      {/* Top fade */}

      <div className="absolute inset-x-0 top-0 h-24 bg-gradient-to-b from-[#0a0a0a] to-transparent" />

      {/* Bottom fade */}

      <div className="absolute inset-x-0 bottom-0 h-20 bg-gradient-to-t from-[#0a0a0a] to-transparent" />

      {/* Inner-edge fade */}

      <div className={cn("absolute inset-y-0 w-64 xl:w-80", isLeft ? "right-0 bg-gradient-to-r from-transparent to-[#0a0a0a]" : "left-0 bg-gradient-to-l from-transparent to-[#0a0a0a]" )} />

    </div>

  );

}

// FriendsPage

interface FriendsPageProps {}

type Tab = "friends" | "discover" | "requests";

interface FriendsPageState {

  tab: Tab;

  profile: UserProfile | null;
  profileLoading: boolean;

  friends: FriendSummary[];
  friendsLoading: boolean;

  requests: FriendRequestItem[];
  requestsLoading: boolean;

  discoverQuery: string;
  discoverResults: FriendSummary[];
  discoverLoading: boolean;

  editOpen: boolean;
  editSaving: boolean;

  actionLoadingId: string | null;

  viewingUserId: string | null;

  posters: string[];

}

export class FriendsPage extends Component<FriendsPageProps, FriendsPageState> {

  private discoverDebounce: ReturnType<typeof setTimeout> | null = null;
  private unsubscribeStore = () => {};
  private lastSseVersion = 0;

  state: FriendsPageState = {

    tab: "friends",

    profile: null,
    profileLoading: true,

    friends: [],
    friendsLoading: true,

    requests: [],
    requestsLoading: true,

    discoverQuery: "",
    discoverResults: [],
    discoverLoading: false,

    editOpen: false,
    editSaving: false,

    actionLoadingId: null,

    viewingUserId: null,

    posters: [],

  };

  componentDidMount() {

    this.lastSseVersion = store.sseEventVersion;

    this.unsubscribeStore = store.subscribe(() => {

      if (store.sseEventVersion !== this.lastSseVersion) {

        this.lastSseVersion = store.sseEventVersion;

        void this.loadRequests();
        void this.loadFriends();

      }

    });

    void this.loadAll();

  }

  componentWillUnmount() {

    this.unsubscribeStore();

    if (this.discoverDebounce) clearTimeout(this.discoverDebounce);

  }

  loadAll = async () => {

    void this.loadProfile();
    void this.loadFriends();
    void this.loadRequests();
    void this.loadDiscover("");
    void this.loadPosters();

  };

  loadPosters = async () => {

    try {

      const [favorites, history, movieTrending, showTrending] = await Promise.all([

        api.getFavorites().catch(() => []),
        api.getHistory(50).catch(() => []),
        api.movieTrending(24).catch(() => []),
        api.showTrending(24).catch(() => []),

      ]);

      const seen = new Set<string>();
      const posters: string[] = [];

      const add = (url?: string) => {

        const u = url?.trim();

        if (u && !seen.has(u) && posters.length < 28) {

          seen.add(u);
          posters.push(u);

        }

      };

      // Favorites and recent history first

      for (const f of favorites ?? []) add(f.poster);
      for (const h of history ?? []) add(h.poster);
      for (const t of movieTrending ?? []) add(t.poster);
      for (const t of showTrending ?? []) add(t.poster);

      if (posters.length > 0) this.setState({ posters });

    } catch {

      /* decorative, ignore */

    }

  };

  loadProfile = async () => {

    this.setState({ profileLoading: true });

    try {

      const profile = await api.getMyProfile();

      this.setState({ profile });

    } catch {

      /* ignore */

    } finally {

      this.setState({ profileLoading: false });

    }

  };

  loadFriends = async () => {

    this.setState({ friendsLoading: true });

    try {

      const friends = await api.listFriends();

      this.setState({ friends: friends ?? [] });

    } catch {

      /* ignore */

    } finally {

      this.setState({ friendsLoading: false });

    }

  };

  loadRequests = async () => {

    this.setState({ requestsLoading: true });

    try {

      const requests = await api.listFriendRequests();

      this.setState({ requests: requests ?? [] });

    } catch {

      /* ignore */

    } finally {

      this.setState({ requestsLoading: false });

    }

  };

  loadDiscover = async (q: string) => {

    this.setState({ discoverLoading: true });

    try {

      const results = await api.searchUsers(q);

      this.setState({ discoverResults: results ?? [] });

    } catch {

      /* ignore */

    } finally {

      this.setState({ discoverLoading: false });

    }

  };

  handleDiscoverSearch = (q: string) => {

    this.setState({ discoverQuery: q });

    if (this.discoverDebounce) clearTimeout(this.discoverDebounce);

    this.discoverDebounce = setTimeout(() => void this.loadDiscover(q), 350);

  };

  handleUserAction = async (summary: FriendSummary) => {

    this.setState({ actionLoadingId: summary.userId });

    try {

      if (summary.friendStatus === "none") {

        await api.sendFriendRequest(summary.userId);

      } else if (summary.friendStatus === "pending_received") {

        const req = this.state.requests.find(
          (r) => r.userId === summary.userId && r.direction === "incoming"
        );

        if (req) await api.acceptFriendRequest(req.id);

      }

      await Promise.all([this.loadFriends(), this.loadRequests(), this.loadDiscover(this.state.discoverQuery)]);

    } catch {

      /* ignore */

    } finally {

      this.setState({ actionLoadingId: null });

    }

  };

  handleAcceptRequest = async (id: string) => {

    this.setState({ actionLoadingId: id });

    try {

      await api.acceptFriendRequest(id);

      await Promise.all([this.loadFriends(), this.loadRequests(), this.loadDiscover(this.state.discoverQuery)]);

    } catch {

      /* ignore */

    } finally {

      this.setState({ actionLoadingId: null });

    }

  };

  handleDeclineRequest = async (id: string) => {

    this.setState({ actionLoadingId: id });

    try {

      await api.deleteFriendRequest(id);

      await Promise.all([this.loadRequests(), this.loadDiscover(this.state.discoverQuery)]);

    } catch {

      /* ignore */

    } finally {

      this.setState({ actionLoadingId: null });

    }

  };

  handleRemoveFriend = async (userId: string) => {

    this.setState({ actionLoadingId: userId });

    try {

      await api.removeFriend(userId);

      await this.loadFriends();

    } catch {

      /* ignore */

    } finally {

      this.setState({ actionLoadingId: null });

    }

  };

  handleSaveProfile = async (form: EditForm) => {

    this.setState({ editSaving: true });

    try {

      const updated = await api.updateProfile({

        displayName: form.displayName,
        bio: form.bio,

        accentColor: form.accentColor,
        banner: form.banner,

        favoriteMovies: form.favoriteMovies,
        favoriteShows: form.favoriteShows,

        historyVisible: form.historyVisible,
        discoverVisible: form.discoverVisible,

      });

      this.setState({ profile: updated, editOpen: false });

      await this.loadDiscover(this.state.discoverQuery);

    } catch {

      /* ignore */

    } finally {

      this.setState({ editSaving: false });

    }

  };

  get incomingCount(): number {

    return this.state.requests.filter((r) => r.direction === "incoming").length;

  }

  renderTabs() {

    const { tab } = this.state;
    const count = this.incomingCount;

    const tabs: { id: Tab; label: string; badge?: number }[] = [

      { id: "friends", label: "Friends" },
      { id: "discover", label: "Discover" },
      { id: "requests", label: "Requests", badge: count > 0 ? count : undefined },

    ];

    return (

      <div className="flex gap-1 rounded-2xl border border-border bg-surface-raised p-1">

        {tabs.map(({ id, label, badge }) => (

          <button className={cn( "relative flex flex-1 items-center justify-center gap-1.5 rounded-3xl px-3 py-1.5 text-sm font-medium transition-colors", tab === id ? "bg-foreground text-surface" : "text-foreground-muted hover:text-foreground")}

            key={id}
            type="button"

            onClick={() => this.setState({ tab: id })}

          >

            {label}

            {badge != null && (

              <span className={cn("flex h-4 min-w-[1rem] items-center justify-center rounded-full px-1 text-[10px] font-semibold", tab === id ? "bg-surface text-foreground" : "bg-foreground text-surface")}>

                {badge}

              </span>

            )}

          </button>

        ))}

      </div>

    );

  }

  renderFriendsTab() {

    const { friends, friendsLoading } = this.state;

    if (friendsLoading) {

      return <LoadingSpinner />;

    }

    if (friends.length === 0) {

      return (

        <EmptyState

          icon={<Users size={32} className="text-foreground-faint" />}

          title="No friends yet"
          description="Find people to add in the Discover tab."

          action={<Button onClick={() => this.setState({ tab: "discover" })}>Discover People</Button>}

        />

      );

    }

    return (

      <div className="space-y-2">

        {friends.map((f) => (

          <ExpandableFriendCard

            key={f.userId}
            summary={f}

            onRemove={this.handleRemoveFriend}
            removeLoading={this.state.actionLoadingId === f.userId}

          />

        ))}

      </div>

    );

  }

  renderDiscoverTab() {

    const { discoverQuery, discoverResults, discoverLoading } = this.state;

    return (

      <div className="space-y-4">

        <div className="relative">

          <Search size={18} className="absolute top-1/2 left-4 -translate-y-1/2 text-foreground-faint" />

          <Input className="w-full pl-10 h-12 rounded-3xl text-sm md:text-lg"

            placeholder="Search by email…"
            value={discoverQuery}

            onChange={(e) => this.handleDiscoverSearch(e.target.value)}

          />

        </div>

        {discoverLoading && <LoadingSpinner />}

        {!discoverLoading && discoverResults.length === 0 && (

          <EmptyState

            icon={<UserPlus size={32} className="text-foreground-faint" />}

            title={discoverQuery ? "No users found" : "No other users yet"}
            description={discoverQuery ? "Try a different email." : "Once others join, they'll appear here."}

          />

        )}

        {!discoverLoading && discoverResults.length > 0 && (

          <div className="space-y-2">

            {discoverResults.map((s) => (

              <UserCard

                key={s.userId}
                summary={s}

                onAction={this.handleUserAction}
                onViewProfile={(id) => this.setState({ viewingUserId: id })}

                actionLoading={this.state.actionLoadingId === s.userId}

              />

            ))}

          </div>

        )}

      </div>

    );

  }

  renderRequestsTab() {

    const { requests, requestsLoading } = this.state;

    if (requestsLoading) {

      return <LoadingSpinner />;

    }

    const incoming = requests.filter((r) => r.direction === "incoming");
    const outgoing = requests.filter((r) => r.direction === "outgoing");

    if (incoming.length === 0 && outgoing.length === 0) {

      return (

        <EmptyState

          icon={<UserPlus size={32} className="text-foreground-faint" />}

          title="No pending requests"
          description="When you receive or send a friend request, it'll show up here."

        />

      );

    }

    return (

      <div className="space-y-6">

        {incoming.length > 0 && (

          <div className="space-y-2">

            <p className="text-xs font-medium uppercase tracking-wide text-foreground-muted">Incoming</p>

            {incoming.map((r) => (

              <RequestCard

                key={r.id}
                request={r}

                onAccept={this.handleAcceptRequest}
                onDecline={this.handleDeclineRequest}
                onViewProfile={(id) => this.setState({ viewingUserId: id })}

                loading={this.state.actionLoadingId === r.id}

              />

            ))}

          </div>

        )}

        {outgoing.length > 0 && (

          <div className="space-y-2">

            <p className="text-xs font-medium uppercase tracking-wide text-foreground-muted">Sent</p>

            {outgoing.map((r) => (

              <RequestCard

                key={r.id}
                request={r}

                onAccept={this.handleAcceptRequest}
                onDecline={this.handleDeclineRequest}

                onViewProfile={(id) => this.setState({ viewingUserId: id })}

                loading={this.state.actionLoadingId === r.id}

              />

            ))}

          </div>

        )}

      </div>

    );

  }

  render() {

    const { profile, profileLoading, editOpen, editSaving, viewingUserId, tab, posters } = this.state;

    const editInitial: EditForm = {

      displayName: profile?.displayName ?? "",
      bio: profile?.bio ?? "",

      accentColor: profile?.accentColor ?? "#6366f1",
      banner: profile?.banner ?? "aurora",

      favoriteMovies: profile?.favoriteMovies ?? [],
      favoriteShows: profile?.favoriteShows ?? [],

      historyVisible: profile?.historyVisible ?? true,
      discoverVisible: profile?.discoverVisible ?? true,

    };

    return (

      <>

      <SideCollage posters={posters} side="left" />
      <SideCollage posters={posters} side="right" />

      <div className="mx-auto max-w-3xl space-y-6 px-4 py-6 sm:px-8">

        {profileLoading ? (

          <div className="h-52 animate-pulse rounded-2xl bg-surface-raised" />

        ) : profile ? (

          <ProfileBanner

            banner={profile.banner}
            accentColor={profile.accentColor}

            displayName={profile.displayName}
            bio={profile.bio}

            isOwn

            onEdit={() => this.setState({ editOpen: true })}

          />

        ) : null}

        {this.renderTabs()}

        <div>

          {tab === "friends"  && this.renderFriendsTab()}
          {tab === "discover" && this.renderDiscoverTab()}
          {tab === "requests" && this.renderRequestsTab()}

        </div>

        <ProfileEditor

          open={editOpen}
          initial={editInitial}
          saving={editSaving}

          onClose={() => this.setState({ editOpen: false })}
          onSave={this.handleSaveProfile}

        />

        <PublicProfileView

          open={viewingUserId !== null}

          userId={viewingUserId}

          onClose={() => this.setState({ viewingUserId: null })}
          onFriendStatusChange={() => {

            void this.loadFriends();
            void this.loadRequests();
            void this.loadDiscover(this.state.discoverQuery);

          }}

        />

      </div>

      </>

    );

  }

}

// Shared small components

function LoadingSpinner() {

  return (

    <div className="flex items-center justify-center py-12">

      <div className="h-6 w-6 animate-spin rounded-full border-2 border-border border-t-foreground-muted" />

    </div>

  );

}

interface EmptyStateProps {

  icon: React.ReactNode;

  title: string;
  description: string;

  action?: React.ReactNode;

}

function EmptyState({ icon, title, description, action }: EmptyStateProps) {

  return (

    <div className="flex flex-col items-center gap-3 py-14 text-center">

      {icon}

      <div>

        <p className="text-sm font-medium text-foreground">{title}</p>
        <p className="mt-1 mb-2 text-sm text-foreground-muted">{description}</p>

      </div>

      {action}

    </div>

  );

}
