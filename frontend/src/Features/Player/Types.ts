export interface StreamQuality {

  label: string;
  height: number;
  isHls: boolean;
  url: string;
  proxyUrl?: string;

}

export interface StreamInfo {

  qualities: StreamQuality[];
  url: string;
  proxyUrl?: string;
  isHls: boolean;

  selectedHeight?: number;

}

export interface SubtitleTrack {

  id: string;
  label: string;
  language: string;
  format: string;

  proxyUrl: string;
  source?: "file" | "hls" | "febbox" | "subdl";

}

export interface IntroInfo {

  introStartMs?: number;
  introEndMs?: number;
  creditsStartMs?: number;

}

export interface NextEpisode {

  season: number;
  episode: number;
  title: string;

}
