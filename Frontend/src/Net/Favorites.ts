import type { FavoriteItem } from "@/Types";

import { request } from "./Request";

export const favoritesAPI = {

  get() {

    return request<FavoriteItem[]>("/api/favorites");

  },

  upsert(item: Partial<FavoriteItem> & { kind: string; mediaId: number; title: string }) {

    return request<FavoriteItem>("/api/favorites", {

      method: "POST",
      body: JSON.stringify(item),

    });

  },

  delete(kind: FavoriteItem["kind"], key: number | string) {

    return request<void>(`/api/favorites/${kind}/${encodeURIComponent(String(key))}`, { method: "DELETE" });

  },

};
