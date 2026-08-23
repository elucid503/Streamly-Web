import { cn } from "./ClassNames";
import * as History from "./History";
import * as ChannelColor from "./Images/ChannelColor";
import { imageCache } from "./Images/Cache";
import * as LogoBackdrop from "./Images/LogoBackdrop";
import * as Navigation from "./Navigation";
import * as Platform from "./Platform";
import * as AlignmentClient from "./Player/AlignmentClient";
import { AudioTap } from "./Player/AudioTap";
import * as CtcAlign from "./Player/CtcAlign";
import * as Intro from "./Player/Intro";
import * as MediaSession from "./Player/MediaSession";
import * as Stream from "./Player/Stream";
import * as StreamClient from "./Player/StreamClient";
import * as SubtitleAlignment from "./Player/SubtitleAlignment";
import * as Vtt from "./Player/Vtt";
import * as WatchRoute from "./Player/WatchRoute";
import * as SportsAlerts from "./Sports/Alerts";
import * as SportsBackgrounds from "./Sports/Backgrounds";
import * as Time from "./Time";

export { cn } from "./ClassNames";
export { formatDuration, progressPercent } from "./Time";

export default class Utils {

  static readonly cn = cn;
  static readonly Time = Time;
  static readonly History = History;
  static readonly Navigation = Navigation;
  static readonly Platform = Platform;
  static readonly ImageCache = imageCache;
  static readonly ChannelColor = ChannelColor;
  static readonly LogoBackdrop = LogoBackdrop;
  static readonly AlignmentClient = AlignmentClient;
  static readonly AudioTap = AudioTap;
  static readonly CtcAlign = CtcAlign;
  static readonly Intro = Intro;
  static readonly MediaSession = MediaSession;
  static readonly Stream = Stream;
  static readonly StreamClient = StreamClient;
  static readonly SubtitleAlignment = SubtitleAlignment;
  static readonly Vtt = Vtt;
  static readonly WatchRoute = WatchRoute;
  static readonly SportsAlerts = SportsAlerts;
  static readonly SportsBackgrounds = SportsBackgrounds;

}
