import type { WatchHistoryItem } from "./Library";

export interface UserProfile {

  id: string;
  userId: string;

  displayName: string;
  bio: string;
  accentColor: string;
  historyVisible: boolean;
  discoverVisible?: boolean;

  updatedAt: string;

}

export interface FriendSummary {

  userId: string;
  email: string;
  displayName: string;
  accentColor: string;
  recentActivity: WatchHistoryItem[];
  friendStatus: "none" | "pending_sent" | "pending_received" | "friends";

}

export interface PublicProfile {

  userId: string;
  email: string;
  displayName: string;
  bio: string;
  accentColor: string;
  recentHistory: WatchHistoryItem[];

  friendStatus: "none" | "pending_sent" | "pending_received" | "friends";

}

export interface FriendRequestItem {

  id: string;
  userId: string;
  email: string;
  displayName: string;
  accentColor: string;
  createdAt: string;
  direction: "incoming" | "outgoing";

}
