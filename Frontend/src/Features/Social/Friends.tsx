import { ContentRow } from "@/Cards/ContentRow";
import { LiveLogo } from "@/Cards/LiveLogo";
import { ModuleComponent } from "@/Core/Store";
import { Button } from "@/UI/Button";
import { Input } from "@/UI/Input";
import { Modal } from "@/UI/Modal";
import { Switch } from "@/UI/Switch";

import Net from "@/Net";
import Stores from "@/Stores";
import type { FriendRequestItem, FriendSummary, LiveChannel, PublicProfile, UserProfile, WatchHistoryItem } from "@/Types";
import { cn } from "@/Utils/ClassNames";

import { Component, type ReactNode } from "react";
import { Check, ChevronDown, Clock3, Pencil, Radio, Search, UserMinus, UserPlus, X } from "lucide-react";

const ACCENT_COLORS = [ "#6366f1", "#8b5cf6", "#ec4899", "#ef4444", "#f97316", "#22c55e", "#14b8a6", "#3b82f6" ];

function initials(name: string): string {

  return name.split(/\s+/).slice(0, 2).map((word) => word[0]?.toUpperCase() ?? "").join("");

}

function timeAgo(iso: string): string {

  const then = new Date(iso).getTime();

  if (Number.isNaN(then)) return "Recently";

  const minutes = Math.max(0, Math.floor((Date.now() - then) / 60_000));

  if (minutes < 1) return "Just now";
  if (minutes < 60) return `${minutes}m ago`;

  const hours = Math.floor(minutes / 60);

  if (hours < 24) return `${hours}h ago`;

  const days = Math.floor(hours / 24);

  if (days < 7) return `${days}d ago`;

  return new Date(iso).toLocaleDateString(undefined, { month: "short", day: "numeric" });

}

function isCurrentlyLive(item: WatchHistoryItem): boolean {

  return item.kind === "live" && Math.abs(Date.now() - new Date(item.updatedAt).getTime()) <= 2 * 60_000;

}

function activityVerb(item: WatchHistoryItem): string {

  if (isCurrentlyLive(item)) return "is watching";
  if (item.kind === "live") return "watched";
  if (item.completed) return "finished";
  if (item.positionMs > 0) return "continued";

  return "started";

}

function episodeLabel(item: WatchHistoryItem): string | null {

  if (item.season == null || item.episode == null) return null;

  const number = `S${item.season} · E${item.episode}`;

  return item.episodeTitle ? `${number} · ${item.episodeTitle}` : number;

}

function liveChannelFromHistory(item: WatchHistoryItem): LiveChannel {

  const logo = item.poster?.trim() ?? "";

  return {

    id: item.channelId ?? item.id,
    name: item.title,
    slug: "",
    code: "",
    logo,
    country: "",
    category: "",
    enriched: logo.length > 0,

  };

}

interface AvatarProps {

  name: string;
  accentColor: string;
  size?: "sm" | "md" | "lg";

}

function Avatar({ name, accentColor, size = "md" }: AvatarProps) {

  const sizeClass = {

    sm: "size-8 text-xs",
    md: "size-10 text-sm",
    lg: "size-16 text-lg",

  }[size];

  return (

    <div
      className={cn("flex flex-shrink-0 items-center justify-center rounded-full font-semibold text-white ring-2 ring-white/10", sizeClass)}
      style={{ backgroundColor: accentColor }}
    >

      {initials(name) || "?"}

    </div>

  );

}

interface SectionHeadingProps {

  title: string;
  subtitle?: string;
  action?: ReactNode;

}

function SectionHeading({ title, subtitle, action }: SectionHeadingProps) {

  return (

    <div className="mb-4 flex flex-col items-stretch justify-between gap-3 sm:flex-row sm:items-end sm:gap-4">

      <div className="min-w-0">

        <h2 className="text-base font-semibold tracking-tight text-foreground">{title}</h2>

        {subtitle ? <p className="mt-0.5 text-sm text-foreground-muted">{subtitle}</p> : null}

      </div>

      {action}

    </div>

  );

}

interface ViewMoreButtonProps {

  expanded: boolean;
  onToggle: () => void;

}

function ViewMoreButton({ expanded, onToggle }: ViewMoreButtonProps) {

  return (

    <button
      type="button"
      aria-expanded={expanded}
      onClick={onToggle}
      className="flex flex-shrink-0 items-center gap-1 text-xs font-medium text-foreground-muted transition-colors hover:text-foreground"
    >

      {expanded ? "Show less" : "View more"}
      <ChevronDown className={cn("size-3.5 transition-transform", expanded && "rotate-180")} />

    </button>

  );

}

interface ActivityCardProps {

  friend: FriendSummary;
  item: WatchHistoryItem;
  onViewProfile: (userId: string) => void;

}

function ActivityCard({ friend, item, onViewProfile }: ActivityCardProps) {

  const live = item.kind === "live";
  const current = isCurrentlyLive(item);
  const episode = episodeLabel(item);
  const progress = item.durationMs > 0 ? Math.min(100, Math.max(0, item.positionMs / item.durationMs * 100)) : 0;

  return (

    <button
      type="button"
      onClick={() => onViewProfile(friend.userId)}
      className="group flex min-h-32 w-full gap-4 rounded-xl border border-border-subtle bg-surface-raised p-4 text-left transition-[border-color,filter] hover:border-border hover:brightness-110 focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-accent/40"
    >

      <div className="relative flex-shrink-0">

        {live && item.poster ? (

          <LiveLogo
            className="flex size-20 items-center justify-center border border-border-subtle bg-surface-overlay"
            channel={liveChannelFromHistory(item)}
            imgClassName="max-h-12 max-w-[80%] object-contain"
            rounded="rounded-lg"
          />

        ) : item.poster ? (

          <img src={item.poster} alt="" className="h-24 w-16 rounded-lg object-cover" />

        ) : (

          <div className="flex h-24 w-16 items-center justify-center rounded-lg bg-surface-overlay text-foreground-faint">

            <Clock3 className="size-5" />

          </div>

        )}

        <div className="absolute -bottom-1.5 -right-2 rounded-full border-2 border-surface-raised">

          <Avatar name={friend.displayName} accentColor={friend.accentColor} size="sm" />

        </div>

      </div>

      <div className="flex min-w-0 flex-1 flex-col">

        <div className="flex items-start justify-between gap-3">

          <p className="min-w-0 text-sm text-foreground-muted">

            <span className="font-semibold text-foreground">{friend.displayName}</span> {activityVerb(item)}

          </p>

          <span className="flex-shrink-0 text-xs text-foreground-faint">{timeAgo(item.updatedAt)}</span>

        </div>

        <p className="mt-3 truncate text-base font-semibold text-foreground group-hover:text-accent">{item.title}</p>

        {episode ? <p className="mt-0.5 truncate text-xs text-foreground-muted">{episode}</p> : null}

        <div className="mt-auto pt-3">

          {current ? (

            <span className="inline-flex items-center gap-1.5 text-xs font-medium text-red-400">

              <span className="size-1.5 animate-pulse rounded-full bg-red-500" />
              Live now

            </span>

          ) : item.completed ? (

            <span className="text-xs text-foreground-faint">Watched</span>

          ) : progress > 0 ? (

            <div className="h-1 overflow-hidden rounded-full bg-surface-overlay">

              <div className="h-full rounded-full bg-foreground" style={{ width: `${progress}%` }} />

            </div>

          ) : (

            <span className="text-xs text-foreground-faint">Recently watched</span>

          )}

        </div>

      </div>

    </button>

  );

}

interface FriendCardProps {

  friend: FriendSummary;
  busy: boolean;
  onViewProfile: (userId: string) => void;
  onRemove: (userId: string) => void;

}

function FriendCard({ friend, busy, onViewProfile, onRemove }: FriendCardProps) {

  const recent = friend.recentActivity[0];

  return (

    <div className="relative w-64 flex-shrink-0 rounded-xl border border-border-subtle bg-surface-raised p-4 transition-[border-color,filter] hover:border-border hover:brightness-110">

      <button type="button" className="flex w-full items-center gap-3 text-left" onClick={() => onViewProfile(friend.userId)}>

        <Avatar name={friend.displayName} accentColor={friend.accentColor} />

        <div className="min-w-0 flex-1 pr-7">

          <p className="truncate text-sm font-semibold text-foreground">{friend.displayName}</p>
          <p className="truncate text-xs text-foreground-faint">{friend.email}</p>

        </div>

      </button>

      <button
        type="button"
        aria-label={`Remove ${friend.displayName}`}
        title="Remove friend"
        disabled={busy}
        onClick={() => onRemove(friend.userId)}
        className="absolute right-3 top-3 flex size-8 items-center justify-center rounded-md text-foreground-faint transition-colors hover:bg-surface-overlay hover:text-red-400 disabled:opacity-40"
      >

        <UserMinus className="size-3.5" />

      </button>

      <button type="button" className="mt-4 block w-full text-left" onClick={() => onViewProfile(friend.userId)}>

        {recent ? (

          <>

            <p className="truncate text-xs text-foreground-muted">

              {activityVerb(recent)} <span className="font-medium text-foreground">{recent.title}</span>

            </p>
            <p className="mt-1 text-xs text-foreground-faint">{timeAgo(recent.updatedAt)}</p>

          </>

        ) : (

          <p className="text-xs text-foreground-faint">No shared activity yet</p>

        )}

      </button>

    </div>

  );

}

interface UserCardProps {

  summary: FriendSummary;
  busy: boolean;
  onAction: (summary: FriendSummary) => void;
  onViewProfile: (userId: string) => void;

}

function UserCard({ summary, busy, onAction, onViewProfile }: UserCardProps) {

  const label = summary.friendStatus === "friends"
    ? "Friends"
    : summary.friendStatus === "pending_sent"
      ? "Requested"
      : summary.friendStatus === "pending_received"
        ? "Accept"
        : "Add";

  return (

    <div className="flex items-center gap-3 rounded-xl border border-border-subtle bg-surface-raised p-3.5">

      <button type="button" className="flex min-w-0 flex-1 items-center gap-3 text-left" onClick={() => onViewProfile(summary.userId)}>

        <Avatar name={summary.displayName} accentColor={summary.accentColor} />

        <div className="min-w-0">

          <p className="truncate text-sm font-semibold text-foreground">{summary.displayName}</p>
          <p className="truncate text-xs text-foreground-faint">{summary.email}</p>

        </div>

      </button>

      <Button
        size="sm"
        variant={summary.friendStatus === "none" || summary.friendStatus === "pending_received" ? "default" : "outline"}
        disabled={busy || summary.friendStatus === "friends" || summary.friendStatus === "pending_sent"}
        onClick={() => onAction(summary)}
      >

        {summary.friendStatus === "pending_received" ? <Check className="size-3.5" /> : <UserPlus className="size-3.5" />}
        {label}

      </Button>

    </div>

  );

}

interface RequestCardProps {

  request: FriendRequestItem;
  busy: boolean;
  onAccept: (id: string) => void;
  onDecline: (id: string) => void;
  onViewProfile: (userId: string) => void;

}

function RequestCard({ request, busy, onAccept, onDecline, onViewProfile }: RequestCardProps) {

  return (

    <div className="flex items-center gap-3 rounded-xl border border-border-subtle bg-surface-raised p-3.5">

      <button type="button" className="flex min-w-0 flex-1 items-center gap-3 text-left" onClick={() => onViewProfile(request.userId)}>

        <Avatar name={request.displayName} accentColor={request.accentColor} />

        <div className="min-w-0">

          <p className="truncate text-sm font-semibold text-foreground">{request.displayName}</p>
          <p className="truncate text-xs text-foreground-faint">{timeAgo(request.createdAt)}</p>

        </div>

      </button>

      <div className="flex flex-shrink-0 gap-1.5">

        <Button size="sm" disabled={busy} onClick={() => onAccept(request.id)}>

          <Check className="size-3.5" />
          Accept

        </Button>

        <Button size="icon-sm" variant="outline" aria-label={`Decline request from ${request.displayName}`} disabled={busy} onClick={() => onDecline(request.id)}>

          <X className="size-3.5" />

        </Button>

      </div>

    </div>

  );

}

interface EditForm {

  displayName: string;
  bio: string;
  accentColor: string;
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

}

class ProfileEditor extends Component<ProfileEditorProps, ProfileEditorState> {

  constructor(props: ProfileEditorProps) {

    super(props);

    this.state = { form: { ...props.initial } };

  }

  componentDidUpdate(prevProps: ProfileEditorProps) {

    if (!prevProps.open && this.props.open) {

      this.setState({ form: { ...this.props.initial } });

    }

  }

  setForm = (patch: Partial<EditForm>) => {

    this.setState({ form: { ...this.state.form, ...patch } });

  };

  render() {

    const { open, saving, onClose, onSave } = this.props;
    const { form } = this.state;

    return (

      <Modal open={open} onClose={onClose} title="Edit profile" className="max-w-lg">

        <div className="space-y-5">

          <div className="space-y-1.5">

            <label htmlFor="profile-display-name" className="text-sm font-medium text-foreground-muted">Display name</label>
            <Input id="profile-display-name" value={form.displayName} onChange={(event) => this.setForm({ displayName: event.target.value })} maxLength={32} autoFocus />

          </div>

          <div className="space-y-1.5">

            <div className="flex items-center justify-between gap-3">

              <label htmlFor="profile-bio" className="text-sm font-medium text-foreground-muted">Bio</label>
              <span className="text-xs text-foreground-faint">{form.bio.length}/160</span>

            </div>

            <textarea
              id="profile-bio"
              className="field-focus min-h-20 w-full resize-none rounded-md border border-border bg-surface-overlay/40 px-3 py-2 text-sm text-foreground placeholder:text-foreground-faint focus:bg-surface-overlay/80"
              value={form.bio}
              onChange={(event) => this.setForm({ bio: event.target.value })}
              placeholder="A short introduction"
              maxLength={160}
            />

          </div>

          <div className="space-y-2">

            <p className="text-sm font-medium text-foreground-muted">Profile color</p>

            <div className="flex flex-wrap gap-2.5">

              {ACCENT_COLORS.map((color) => (

                <button
                  key={color}
                  type="button"
                  aria-label={`Use ${color} as profile color`}
                  aria-pressed={form.accentColor === color}
                  onClick={() => this.setForm({ accentColor: color })}
                  className="size-8 rounded-full transition-transform hover:scale-105 focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-accent/40"
                  style={{ backgroundColor: color, boxShadow: form.accentColor === color ? `0 0 0 2px #111111, 0 0 0 4px ${color}` : undefined }}
                />

              ))}

            </div>

          </div>

          <div className="rounded-lg border border-border-subtle bg-surface-overlay/30 px-3">

            <Switch checked={form.historyVisible} label="Share watch activity" description="Friends can see what you recently watched." onChange={(historyVisible) => this.setForm({ historyVisible })} />

            <div className="border-t border-border-subtle" />

            <Switch checked={form.discoverVisible} label="Appear in discovery" description="Other people can find and add you." onChange={(discoverVisible) => this.setForm({ discoverVisible })} />

          </div>

          <div className="flex justify-end gap-2">

            <Button variant="outline" onClick={onClose} disabled={saving}>Cancel</Button>
            <Button onClick={() => onSave(form)} disabled={saving || form.displayName.trim().length === 0}>

              {saving ? "Saving…" : "Save profile"}

            </Button>

          </div>

        </div>

      </Modal>

    );

  }

}

function CompactActivityRow({ item }: { item: WatchHistoryItem }) {

  const episode = episodeLabel(item);

  return (

    <div className="flex items-center gap-3 rounded-lg border border-border-subtle bg-surface-overlay/30 p-3">

      {item.kind === "live" && item.poster ? (

        <LiveLogo className="flex size-12 flex-shrink-0 items-center justify-center bg-surface-overlay" channel={liveChannelFromHistory(item)} imgClassName="max-h-8 max-w-[80%] object-contain" rounded="rounded-md" />

      ) : item.poster ? (

        <img src={item.poster} alt="" className="h-14 w-10 flex-shrink-0 rounded-md object-cover" />

      ) : (

        <div className="h-14 w-10 flex-shrink-0 rounded-md bg-surface-overlay" />

      )}

      <div className="min-w-0 flex-1">

        <p className="truncate text-sm font-medium text-foreground">{item.title}</p>
        {episode ? <p className="truncate text-xs text-foreground-muted">{episode}</p> : null}

      </div>

      <span className="flex-shrink-0 text-xs text-foreground-faint">{timeAgo(item.updatedAt)}</span>

    </div>

  );

}

interface FriendProfileModalProps {

  open: boolean;
  userId: string | null;
  busyUserId: string | null;
  onClose: () => void;
  onAction: (profile: PublicProfile) => void;
  onRemove: (userId: string) => void;

}

interface FriendProfileModalState {

  profile: PublicProfile | null;
  loading: boolean;

}

class FriendProfileModal extends Component<FriendProfileModalProps, FriendProfileModalState> {

  state: FriendProfileModalState = { profile: null, loading: false };

  componentDidMount() {

    if (this.props.open && this.props.userId) void this.load(this.props.userId);

  }

  componentDidUpdate(prevProps: FriendProfileModalProps) {

    if (this.props.open && this.props.userId && (!prevProps.open || prevProps.userId !== this.props.userId)) {

      void this.load(this.props.userId);

    }

  }

  load = async (userId: string) => {

    this.setState({ profile: null, loading: true });

    try {

      const profile = await Net.Social.getPublicProfile(userId);

      if (this.props.userId === userId) this.setState({ profile });

    } catch {

      if (this.props.userId === userId) this.setState({ profile: null });

    } finally {

      if (this.props.userId === userId) this.setState({ loading: false });

    }

  };

  renderAction(profile: PublicProfile) {

    const { busyUserId, onAction, onRemove } = this.props;
    const busy = busyUserId === profile.userId;

    if (profile.friendStatus === "friends") {

      return (

        <Button variant="outline" size="sm" disabled={busy} onClick={() => onRemove(profile.userId)}>

          <UserMinus className="size-3.5" />
          Remove friend

        </Button>

      );

    }

    return (

      <Button size="sm" disabled={busy || profile.friendStatus === "pending_sent"} onClick={() => onAction(profile)}>

        {profile.friendStatus === "pending_received" ? <Check className="size-3.5" /> : <UserPlus className="size-3.5" />}
        {profile.friendStatus === "pending_sent" ? "Requested" : profile.friendStatus === "pending_received" ? "Accept" : "Add friend"}

      </Button>

    );

  }

  render() {

    const { open, onClose } = this.props;
    const { profile, loading } = this.state;

    return (

      <Modal open={open} onClose={onClose} title={profile?.displayName ?? "Profile"} className="max-w-xl">

        {loading ? (

          <div className="space-y-3 py-2">

            <div className="skeleton h-16 w-full rounded-lg" />
            <div className="skeleton h-16 w-full rounded-lg" />
            <div className="skeleton h-16 w-full rounded-lg" />

          </div>

        ) : profile ? (

          <div>

            <div className="flex flex-col gap-4 sm:flex-row sm:items-start">

              <div className="flex min-w-0 flex-1 items-center gap-4">

                <Avatar name={profile.displayName} accentColor={profile.accentColor} size="lg" />

                <div className="min-w-0 flex-1">

                  <p className="truncate text-base text-foreground-muted">{profile.email}</p>
                  <p className="text-sm leading-relaxed text-foreground-faint">{profile.bio || "No bio yet."}</p>

                </div>

              </div>

              {this.renderAction(profile)}

            </div>

            {profile.friendStatus === "friends" ? (

              <div className="mt-6">

                {profile.recentHistory.length > 0 ? (

                  <div className="max-h-80 space-y-2 overflow-y-auto">

                    {profile.recentHistory.map((item) => <CompactActivityRow key={item.id} item={item} />)}

                  </div>

                ) : (

                  <p className="rounded-lg border border-border-subtle bg-surface-overlay/20 px-4 py-8 text-center text-sm text-foreground-faint">No shared activity yet.</p>

                )}

              </div>

            ) : null}

          </div>

        ) : (

          <p className="py-10 text-center text-sm text-foreground-muted">This profile could not be loaded.</p>

        )}

      </Modal>

    );

  }

}

interface FriendsPageProps {}

interface ActivityEntry {

  friend: FriendSummary;
  item: WatchHistoryItem;

}

interface FriendsPageState {

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
  activityExpanded: boolean;
  viewportWidth: number;
  actionLoadingId: string | null;
  viewingUserId: string | null;

}

export class FriendsPage extends ModuleComponent<FriendsPageProps, FriendsPageState> {

  private discoverDebounce: ReturnType<typeof setTimeout> | null = null;
  private lastSseVersion = 0;

  state: FriendsPageState = {

    profile: null,
    profileLoading: true,
    friends: [],
    friendsLoading: true,
    requests: [],
    requestsLoading: true,
    discoverQuery: "",
    discoverResults: [],
    discoverLoading: true,
    editOpen: false,
    editSaving: false,
    activityExpanded: false,
    viewportWidth: typeof window === "undefined" ? 1280 : window.innerWidth,
    actionLoadingId: null,
    viewingUserId: null,

  };

  componentDidMount() {

    this.lastSseVersion = Stores.Social.sseEventVersion;
    this.watch(Stores.Social);

    window.addEventListener("resize", this.handleResize);

    void this.loadAll();

  }

  componentDidUpdate() {

    if (Stores.Social.sseEventVersion === this.lastSseVersion) return;

    this.lastSseVersion = Stores.Social.sseEventVersion;

    void this.loadRequests();
    void this.loadFriends();

  }

  componentWillUnmount() {

    if (this.discoverDebounce) clearTimeout(this.discoverDebounce);

    window.removeEventListener("resize", this.handleResize);

  }

  handleResize = () => {

    this.setState({ viewportWidth: window.innerWidth });

  };

  loadAll = async () => {

    await Promise.all([ this.loadProfile(), this.loadFriends(), this.loadRequests(), this.loadDiscover("") ]);

  };

  loadProfile = async () => {

    this.setState({ profileLoading: true });

    try {

      const profile = await Net.Social.getMyProfile();

      this.setState({ profile });

    } catch {

      this.setState({ profile: null });

    } finally {

      this.setState({ profileLoading: false });

    }

  };

  loadFriends = async () => {

    this.setState({ friendsLoading: true });

    try {

      const friends = await Net.Social.listFriends() ?? [];

      friends.sort((left, right) => {

        const leftUpdated = left.recentActivity[0]?.updatedAt ?? "";
        const rightUpdated = right.recentActivity[0]?.updatedAt ?? "";

        if (leftUpdated !== rightUpdated) return rightUpdated.localeCompare(leftUpdated);

        return left.displayName.localeCompare(right.displayName);

      });

      this.setState({ friends });

    } catch {

      this.setState({ friends: [] });

    } finally {

      this.setState({ friendsLoading: false });

    }

  };

  loadRequests = async () => {

    this.setState({ requestsLoading: true });

    try {

      const requests = await Net.Social.listFriendRequests();

      this.setState({ requests: requests ?? [] });

    } catch {

      this.setState({ requests: [] });

    } finally {

      this.setState({ requestsLoading: false });

    }

  };

  loadDiscover = async (query: string) => {

    this.setState({ discoverLoading: true });

    try {

      const results = await Net.Social.searchUsers(query);

      this.setState({ discoverResults: results ?? [] });

    } catch {

      this.setState({ discoverResults: [] });

    } finally {

      this.setState({ discoverLoading: false });

    }

  };

  handleDiscoverSearch = (query: string) => {

    this.setState({ discoverQuery: query });

    if (this.discoverDebounce) clearTimeout(this.discoverDebounce);

    this.discoverDebounce = setTimeout(() => void this.loadDiscover(query), 300);

  };

  handleUserAction = async (summary: { userId: string; friendStatus: FriendSummary["friendStatus"] }) => {

    this.setState({ actionLoadingId: summary.userId });

    try {

      if (summary.friendStatus === "none") {

        await Net.Social.sendFriendRequest(summary.userId);

      } else if (summary.friendStatus === "pending_received") {

        const request = this.state.requests.find((item) => item.userId === summary.userId && item.direction === "incoming");

        if (request) await Net.Social.acceptFriendRequest(request.id);

      }

      await Promise.all([ this.loadFriends(), this.loadRequests(), this.loadDiscover(this.state.discoverQuery) ]);

      if (this.state.viewingUserId === summary.userId) this.setState({ viewingUserId: null });

    } catch {

      // The surrounding state remains usable if the request fails.
    } finally {

      this.setState({ actionLoadingId: null });

    }

  };

  handleAcceptRequest = async (id: string) => {

    this.setState({ actionLoadingId: id });

    try {

      await Net.Social.acceptFriendRequest(id);
      await Promise.all([ this.loadFriends(), this.loadRequests(), this.loadDiscover(this.state.discoverQuery) ]);

    } catch {

      // The request stays visible so it can be retried.
    } finally {

      this.setState({ actionLoadingId: null });

    }

  };

  handleDeclineRequest = async (id: string) => {

    this.setState({ actionLoadingId: id });

    try {

      await Net.Social.deleteFriendRequest(id);
      await Promise.all([ this.loadRequests(), this.loadDiscover(this.state.discoverQuery) ]);

    } catch {

      // The request stays visible so it can be retried.
    } finally {

      this.setState({ actionLoadingId: null });

    }

  };

  handleRemoveFriend = async (userId: string) => {

    this.setState({ actionLoadingId: userId });

    try {

      await Net.Social.removeFriend(userId);
      await Promise.all([ this.loadFriends(), this.loadDiscover(this.state.discoverQuery) ]);

      if (this.state.viewingUserId === userId) this.setState({ viewingUserId: null });

    } catch {

      // The friend remains visible so removal can be retried.
    } finally {

      this.setState({ actionLoadingId: null });

    }

  };

  handleSaveProfile = async (form: EditForm) => {

    this.setState({ editSaving: true });

    try {

      const profile = await Net.Social.updateProfile({

        displayName: form.displayName.trim(),
        bio: form.bio.trim(),
        accentColor: form.accentColor,
        historyVisible: form.historyVisible,
        discoverVisible: form.discoverVisible,

      });

      this.setState({ profile, editOpen: false });

    } catch {

      // Keep the editor open so the user can retry.
    } finally {

      this.setState({ editSaving: false });

    }

  };

  activityEntries(): ActivityEntry[] {

    const entries = this.state.friends.flatMap((friend) => friend.recentActivity.map((item) => ({ friend, item })));

    entries.sort((left, right) => right.item.updatedAt.localeCompare(left.item.updatedAt));

    return entries.slice(0, 24);

  }

  renderProfile() {

    const { profile, profileLoading } = this.state;

    if (profileLoading) return <div className="skeleton h-24 w-full rounded-xl" />;
    if (!profile) return null;

    return (

      <div className="flex items-center gap-3 rounded-xl border border-border-subtle bg-surface-raised p-3 sm:gap-4 sm:p-4">

        <Avatar name={profile.displayName} accentColor={profile.accentColor} />

        <div className="min-w-0 flex-1">

          <p className="truncate text-base font-semibold text-foreground sm:text-lg">{profile.displayName}</p>
          <p className={cn("mt-0.5 line-clamp-1 max-w-2xl text-sm", profile.bio ? "text-foreground-muted" : "text-foreground-faint")}>{profile.bio || "Add a short bio so friends know it’s you."}</p>

        </div>

        <Button variant="outline" size="sm" className="px-2 sm:px-3" aria-label="Edit profile" onClick={() => this.setState({ editOpen: true })}>

          <Pencil className="size-3.5" />
          <span className="hidden sm:inline">Edit profile</span>

        </Button>

      </div>

    );

  }

  renderRequests() {

    const { requests, requestsLoading, actionLoadingId } = this.state;
    const incoming = requests.filter((request) => request.direction === "incoming");

    if (!requestsLoading && incoming.length === 0) return null;

    return (

      <section className="mb-8 px-4 sm:px-8">

        <SectionHeading title="Friend requests" subtitle="People who want to connect with you" />

        {requestsLoading ? (

          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">

            {Array.from({ length: 2 }).map((_, index) => <div key={index} className="skeleton h-[70px] rounded-xl" />)}

          </div>

        ) : (

          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">

            {incoming.map((request) => (

              <RequestCard key={request.id} request={request} busy={actionLoadingId === request.id} onAccept={this.handleAcceptRequest} onDecline={this.handleDeclineRequest} onViewProfile={(userId) => this.setState({ viewingUserId: userId })} />

            ))}

          </div>

        )}

      </section>

    );

  }

  renderActivity() {

    const { friendsLoading, friends, activityExpanded, viewportWidth } = this.state;
    const activity = this.activityEntries();
    const previewCount = viewportWidth >= 1024 ? 4 : 2;
    const visibleActivity = activityExpanded ? activity : activity.slice(0, previewCount);

    return (

      <section className="mb-8 px-4 sm:px-8">

        <SectionHeading
          title="Friend activity"
          subtitle="What your circle has been watching"
          action={activity.length > previewCount ? (

            <ViewMoreButton expanded={activityExpanded} onToggle={() => this.setState({ activityExpanded: !activityExpanded })} />

          ) : undefined}
        />

        {friendsLoading ? (

          <div className="grid gap-3 lg:grid-cols-2">

            {Array.from({ length: 4 }).map((_, index) => <div key={index} className="skeleton h-32 rounded-xl" />)}

          </div>

        ) : activity.length > 0 ? (

          <div className="grid gap-3 lg:grid-cols-2">

            {visibleActivity.map(({ friend, item }) => <ActivityCard key={`${friend.userId}:${item.id}`} friend={friend} item={item} onViewProfile={(userId) => this.setState({ viewingUserId: userId })} />)}

          </div>

        ) : (

          <EmptyState icon={<Radio className="size-7" />} title={friends.length === 0 ? "Your activity feed is ready" : "No activity shared yet"} description={friends.length === 0 ? "Add a few people and their recent watches will appear here." : "New watches from friends will appear here."} />

        )}

      </section>

    );

  }

  renderFriends() {

    const { friends, friendsLoading, actionLoadingId } = this.state;

    if (friendsLoading) {

      return (

        <section className="mb-8 px-4 sm:px-8">

          <SectionHeading title="Your friends" />

          <div className="flex gap-3 overflow-hidden">

            {Array.from({ length: 4 }).map((_, index) => <div key={index} className="skeleton h-32 w-64 flex-shrink-0 rounded-xl" />)}

          </div>

        </section>

      );

    }

    if (friends.length === 0) return null;

    return (

      <ContentRow title="Your friends" subtitle={`${friends.length} ${friends.length === 1 ? "person" : "people"} in your circle`} sectionId="friends-list">

        {friends.map((friend) => <FriendCard key={friend.userId} friend={friend} busy={actionLoadingId === friend.userId} onViewProfile={(userId) => this.setState({ viewingUserId: userId })} onRemove={this.handleRemoveFriend} />)}

      </ContentRow>

    );

  }

  renderDiscovery() {

    const { discoverQuery, discoverResults, discoverLoading, viewportWidth, actionLoadingId } = this.state;
    const previewCount = viewportWidth >= 1280 ? 6 : viewportWidth >= 640 ? 4 : 2;
    const visibleResults = discoverResults.slice(0, previewCount);

    return (

      <section className="mb-8 px-4 sm:px-8">

        <SectionHeading
          title="Find people"
          subtitle="Search by email"
          action={(

            <div className="relative w-full sm:w-72">

              <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-foreground-faint" />
              <Input className="pl-9" aria-label="Search people by email" placeholder="Email address" value={discoverQuery} onChange={(event) => this.handleDiscoverSearch(event.target.value)} />

            </div>

          )}
        />

        {discoverLoading ? (

          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">

            {Array.from({ length: 3 }).map((_, index) => <div key={index} className="skeleton h-[70px] rounded-xl" />)}

          </div>

        ) : discoverResults.length > 0 ? (

          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">

            {visibleResults.map((summary) => <UserCard key={summary.userId} summary={summary} busy={actionLoadingId === summary.userId} onAction={this.handleUserAction} onViewProfile={(userId) => this.setState({ viewingUserId: userId })} />)}

          </div>

        ) : (

          <EmptyState icon={<UserPlus className="size-7" />} title={discoverQuery ? "No people found" : "No suggestions yet"} description={discoverQuery ? "Try a different email address." : "New people will appear here when they join."} />

        )}

      </section>

    );

  }

  render() {

    const { profile, editOpen, editSaving, actionLoadingId, viewingUserId } = this.state;

    const editInitial: EditForm = {

      displayName: profile?.displayName ?? "",
      bio: profile?.bio ?? "",
      accentColor: profile?.accentColor ?? "#6366f1",
      historyVisible: profile?.historyVisible ?? true,
      discoverVisible: profile?.discoverVisible ?? true,

    };

    return (

      <div className="animate-fade-in py-8">

        <section className="mb-8 px-4 sm:px-8">

          <div className="mb-6">

            <h1 className="text-xl font-semibold tracking-tight text-foreground sm:text-2xl">Friends</h1>

          </div>

          {this.renderProfile()}

        </section>

        {this.renderRequests()}
        {this.renderFriends()}
        {this.renderActivity()}
        {this.renderDiscovery()}

        <ProfileEditor open={editOpen} saving={editSaving} initial={editInitial} onClose={() => this.setState({ editOpen: false })} onSave={this.handleSaveProfile} />

        <FriendProfileModal open={viewingUserId !== null} userId={viewingUserId} busyUserId={actionLoadingId} onClose={() => this.setState({ viewingUserId: null })} onAction={this.handleUserAction} onRemove={this.handleRemoveFriend} />

      </div>

    );

  }

}

interface EmptyStateProps {

  icon: ReactNode;
  title: string;
  description: string;

}

function EmptyState({ icon, title, description }: EmptyStateProps) {

  return (

    <div className="flex min-h-40 flex-col items-center justify-center rounded-xl border border-dashed border-border bg-surface-raised/40 px-6 py-10 text-center text-foreground-faint">

      {icon}

      <p className="mt-3 text-sm font-semibold text-foreground">{title}</p>
      <p className="mt-1 max-w-md text-sm text-foreground-muted">{description}</p>

    </div>

  );

}
