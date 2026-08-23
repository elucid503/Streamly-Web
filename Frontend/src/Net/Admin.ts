import type { AccessCode, ServiceInterruption } from "@/Types";

import { request } from "./Request";

export const adminAPI = {

  createAccessCode(maxUses: number, expiresIn?: string) {

    return request<AccessCode>("/api/admin/access-codes", {

      method: "POST",
      body: JSON.stringify({ maxUses, expiresIn }),

    });

  },

  listAccessCodes() {

    return request<AccessCode[]>("/api/admin/access-codes");

  },

  deleteAccessCode(code: string) {

    return request<void>(`/api/admin/access-codes/${code}`, { method: "DELETE" });

  },

  getServiceInterruption() {

    return request<ServiceInterruption>("/api/admin/service-interruption");

  },

  updateServiceInterruption(data: Pick<ServiceInterruption, "enabled" | "title" | "message">) {

    return request<ServiceInterruption>("/api/admin/service-interruption", {

      method: "PUT",
      body: JSON.stringify(data),

    });

  },

};
