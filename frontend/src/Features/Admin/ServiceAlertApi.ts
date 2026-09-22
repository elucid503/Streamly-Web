import type { ServiceInterruption } from "@/Features/Admin/Types";
import { request } from "@/Core/Request";

export const serviceAlertAPI = {

  get() {

    return request<ServiceInterruption>("/api/service-interruption");

  },

};
