import type { WatchHistoryItem } from "@/Types";

import { request } from "./Request";

export const historyAPI = {

  get(limit = 50, mediaId?: number) {

    const params = new URLSearchParams({ limit: String(limit) });

    if (mediaId != null) params.set("mediaId", String(mediaId));

    return request<WatchHistoryItem[]>(`/api/history?${params}`);

  },

  upsert(item: Partial<WatchHistoryItem> & { kind: string; mediaId: number; title: string }) {

    return request<WatchHistoryItem>("/api/history", {

      method: "POST",
      body: JSON.stringify(item),

    });

  },

  delete(id: string) {

    return request<void>(`/api/history/${id}`, { method: "DELETE" });

  },

};
