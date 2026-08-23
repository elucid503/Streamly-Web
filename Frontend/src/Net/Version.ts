import { request } from "./Request";

export const versionAPI = {

  get() {

    return request<{ version: string }>("/api/version");

  },

};
