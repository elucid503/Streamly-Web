import type { SportsAlert, SportsAlertsList, SportsTeamAlert } from "@/Types";

import { request } from "./Request";

export const sportsAPI = {

  listAlerts() {

    return request<SportsAlertsList>("/api/sports/alerts");

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

  subscribeTeam(team: string) {

    return request<SportsTeamAlert>("/api/sports/alerts/teams", {

      method: "PUT",
      body: JSON.stringify({ team }),

    });

  },

  unsubscribeTeam(team: string) {

    return request<void>("/api/sports/alerts/teams", {

      method: "DELETE",
      body: JSON.stringify({ team }),

    });

  },

};
