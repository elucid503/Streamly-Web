import type { UserSettings } from "@/Features/Settings/Types";
import { request } from "@/Core/Request";

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
