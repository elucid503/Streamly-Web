import { request } from "@/Core/Request";

export const versionAPI = {

  get() {

    return request<{ version: string }>("/api/version");

  },

};
