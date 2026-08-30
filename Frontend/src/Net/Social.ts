import type { FriendRequestItem, FriendSummary, PublicProfile, UserProfile } from "@/Types";

import { request } from "./Request";

export const socialAPI = {

  getMyProfile() {

    return request<UserProfile>("/api/social/profile");

  },

  updateProfile(data: {

    displayName?: string;
    bio?: string;

    accentColor?: string;

    historyVisible?: boolean;
    discoverVisible?: boolean;

  }) {

    return request<UserProfile>("/api/social/profile", {

      method: "PUT",
      body: JSON.stringify(data),

    });

  },

  getPublicProfile(userId: string) {

    return request<PublicProfile>(`/api/social/profile/${encodeURIComponent(userId)}`);

  },

  searchUsers(q: string) {

    return request<FriendSummary[]>(`/api/social/users?q=${encodeURIComponent(q)}`);

  },

  listFriends() {

    return request<FriendSummary[]>("/api/social/friends");

  },

  listFriendRequests() {

    return request<FriendRequestItem[]>("/api/social/friends/requests");

  },

  sendFriendRequest(toId: string) {

    return request<void>("/api/social/friends/requests", {

      method: "POST",
      body: JSON.stringify({ toId }),

    });

  },

  acceptFriendRequest(id: string) {

    return request<void>(`/api/social/friends/requests/${encodeURIComponent(id)}/accept`, {

      method: "PUT",

    });

  },

  deleteFriendRequest(id: string) {

    return request<void>(`/api/social/friends/requests/${encodeURIComponent(id)}`, {

      method: "DELETE",

    });

  },

  removeFriend(userId: string) {

    return request<void>(`/api/social/friends/${encodeURIComponent(userId)}`, {

      method: "DELETE",

    });

  },

};
