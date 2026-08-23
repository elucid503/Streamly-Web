import type { User } from "@/Types";

import { request } from "./Request";

export const authAPI = {

  register(email: string, password: string, accessCode: string) {

    return request<User>("/api/auth/register", {

      method: "POST",
      body: JSON.stringify({ email, password, accessCode }),

    });

  },

  login(email: string, password: string) {

    return request<User>("/api/auth/login", {

      method: "POST",
      body: JSON.stringify({ email, password }),

    });

  },

  logout() {

    return request<void>("/api/auth/logout", { method: "POST" });

  },

  me() {

    return request<User>("/api/auth/me");

  },

};
