import type { ServiceInterruption } from "@/Types";

import { request } from "./Request";

export const serviceAlertAPI = {

  get() {

    return request<ServiceInterruption>("/api/service-interruption");

  },

};
