import type { SportsAlert } from "@/Types";

import { request } from "./Request";

export const sportsAPI = {

  listAlerts() {

    return request<SportsAlert[]>("/api/sports/alerts");

  },

  subscribe(matchId: string) {

    return request<SportsAlert>(`/api/sports/alerts/${encodeURIComponent(matchId)}`, {

      method: "PUT",

    });

  },

  unsubscribe(matchId: string) {

    return request<void>(`/api/sports/alerts/${encodeURIComponent(matchId)}`, {

      method: "DELETE",

    });

  },

};
