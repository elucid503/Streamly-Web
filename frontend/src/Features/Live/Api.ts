import type { ChannelGuideEntry, LiveChannel, LiveSourceProvider } from "@/Features/Live/Types";
import { withIosProxyQuery } from "@/Features/Player/Playback/AirPlay";
import { request } from "@/Core/Request";

export const liveAPI = {

  channels() {

    return request<LiveChannel[]>("/api/live/channels");

  },

  popular(limit = 24) {

    return request<LiveChannel[]>(`/api/live/channels/popular?limit=${limit}`);

  },

  search(q: string) {

    return request<LiveChannel[]>(`/api/live/channels/search?q=${encodeURIComponent(q)}&limit=48`);

  },

  schedule() {

    return request<ChannelGuideEntry[]>("/api/live/schedule");

  },

  providers() {

    return request<LiveSourceProvider[]>("/api/live/providers");

  },

  stream(channelId: string, provider?: string) {

    const q = provider && provider !== "auto" ? `?provider=${encodeURIComponent(provider)}` : "";

    return request<{

      streamUrl: string;
      isHls: boolean;
      channel: LiveChannel;
      /** Anonymized public source key (auto/s1/…). */
      provider?: string;

    }>(withIosProxyQuery(`/api/live/channels/${channelId}/stream${q}`));

  },

};
