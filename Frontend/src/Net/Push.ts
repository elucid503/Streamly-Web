import { request } from "./Request";

export const pushAPI = {

  vapidKey() {

    return request<{ publicKey: string }>("/api/push/vapid");

  },

  upsertSubscription(subscription: { endpoint: string; keys: { p256dh: string; auth: string } }) {

    return request<void>("/api/push/subscription", {

      method: "PUT",
      body: JSON.stringify(subscription),

    });

  },

  deleteSubscription(endpoint: string) {

    return request<void>("/api/push/subscription", {

      method: "DELETE",
      body: JSON.stringify({ endpoint }),

    });

  },

};
