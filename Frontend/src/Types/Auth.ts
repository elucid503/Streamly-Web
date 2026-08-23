export interface User {

  id: string;
  email: string;

  isAdmin: boolean;

}

export interface UserSettings {

  preferredHeight: number;
  autoPlayNext: boolean;
  skipIntro: boolean;
  disablePauseOverlay: boolean;
  ambienceEnabled: boolean;
  subtitlesEnabled: boolean;
  proxyLiveStreams: boolean;
  detectLiveAds?: boolean;

}
