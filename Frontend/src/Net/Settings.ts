import type { UserSettings } from "@/Types";

import { request } from "./Request";

export const settingsAPI = {

  get() {

    return request<UserSettings>("/api/settings");

  },

  update(settings: Partial<UserSettings>) {

    return request<UserSettings>("/api/settings", {

      method: "PUT",
      body: JSON.stringify(settings),

    });

  },

};
