import type HLS from "hls.js";
import { createRef, type MouseEvent as ReactMouseEvent, type PointerEvent as ReactPointerEvent, type RefObject } from "react";
import { Airplay, ArrowLeft, Clapperboard, Expand, FastForward, Maximize, Minimize, Pause, Play, Rewind } from "lucide-react";

import { MiniPlayerControls } from "@/Features/Player/MiniPlayerControls";
import { EpisodeActions } from "@/Features/Player/EpisodeActions";
import { AmbienceLayer } from "@/Features/Player/AmbienceLayer";
import { EpisodePickerPanel } from "@/Features/Player/EpisodePickerPanel";
import { AdBreakOverlay } from "@/Features/Player/Broadcast/AdBreakOverlay";
import { PauseOverlay } from "@/Features/Player/PauseOverlay";
import { PlayerActionFeedbackOverlay, type PlayerActionFeedback, } from "@/Features/Player/PlayerActions";
import { PlayerOptionsMenu } from "@/Features/Player/PlayerOptionsMenu";
import { SeekTooltip } from "@/Features/Player/SeekTooltip";
import { SubtitleDisplay } from "@/Features/Player/Subtitles/SubtitleDisplay";
import { ControlButton, VolumeControl } from "@/Features/Player/VolumeControl";

import { bestSupportedHlsLevel, isHdrLevel, type HlsLevelLike } from "@/Features/Player/Playback/HlsLevels";
import { nativeFullscreenVideo, readPortrait, readStoredVolume } from "@/Features/Player/Playback/Browser";
import { ModuleComponent } from "@/Core/Store";
import { settings as settingsStore } from "@/Features/Settings/Store";
import { navigate } from "@/Utils/Navigation";
import { AdBreakDetector } from "@/Features/Player/Broadcast/AdBreak";
import { hasIntroWindow, isInIntroWindow } from "@/Features/Player/Playback/Intro";
import { isAirPlayActive, prepareVideoForAirPlay, shouldUseNativeHls, showAirPlayPicker, supportsAirPlayPicker, type WebKitPlaybackTargetAvailabilityEvent } from "@/Features/Player/Playback/AirPlay";
import { isProxiedStream, isWebPlayableUrl } from "@/Features/Player/Playback/StreamClient";
import { isMobile } from "@/Utils/Platform";
import { clearMediaSession, enableBackgroundAudio, setMediaSessionHandlers, setMediaSessionMetadata, setMediaSessionPlaybackState, setMediaSessionPosition } from "@/Features/Player/Playback/MediaSession";
import { ScreenWakeLock } from "@/Features/Player/Playback/WakeLock";
import { getLiveMediaArtwork } from "@/Features/Live/LogoBackdrop";
import { cn } from "@/Utils/ClassNames";
import { formatDuration } from "@/Utils/Time";
import type { Episode, Season } from "@/Features/Catalog/Types";
import type { IntroInfo, NextEpisode, StreamQuality, SubtitleTrack } from "@/Features/Player/Types";
import type { LiveSourceProvider } from "@/Features/Live/Types";

interface VideoPlayerProps {

  src: string;
  isHls: boolean;
  live?: boolean;
  lowLatency?: boolean;
  compact?: boolean;
  onReturn?: () => void;
  onDismiss?: () => void;

  title: string;
  subtitle?: string;
  episodeTitle?: string;
  description?: string;
  poster?: string;

  qualities?: StreamQuality[];
  selectedHeight?: number;
  preferredHeight?: number;
  subtitleTracks?: SubtitleTrack[];

  intro?: IntroInfo | null;
  nextEpisode?: NextEpisode | null;
  autoPlayNext?: boolean;
  skipIntroEnabled?: boolean;

  ambienceEnabled?: boolean;
  subtitlesEnabled?: boolean;
  startPositionMs?: number;

  onBack?: () => void;
  onSubtitlesEnabledChange?: (enabled: boolean) => void;
  onProgress?: (positionMs: number, durationMs: number) => void;
  onPlaybackStateChange?: (playing: boolean) => void;
  onEnded?: () => void;
  onNextEpisode?: () => void;
  onQualityChange?: (height: number, positionMs: number) => void;
  onOpenSettings?: () => void;
  onDurationReady?: (durationMs: number) => void;
  onPlaybackError?: (positionMs: number) => void;
  onFatalError?: () => void;

  /** Live TV: anonymized stream source switcher. */
  sourceProviders?: LiveSourceProvider[];
  selectedSourceKey?: string;
  sourceSwitching?: boolean;
  onSourceChange?: (key: string) => void;

  seasons?: Season[];
  episodes?: Episode[];
  currentSeason?: number;
  currentEpisode?: number;
  menuSeason?: number;
  episodesLoading?: boolean;
  onSeasonChange?: (season: number) => void;
  onEpisodeSelect?: (season: number, episode: number) => void;

  streamResolving?: boolean;

}

interface VideoPlayerState {

  playing: boolean;
  muted: boolean;
  volume: number;

  showControls: boolean;
  showOptions: boolean;
  showEpisodes: boolean;
  showSkipIntro: boolean;
  showUpNext: boolean;
  showUpNextMini: boolean;
  upNextCountdown: number;

  fullscreen: boolean;
  loading: boolean;
  seeking: boolean;
  holdPauseActive: boolean;

  activeSubtitleId: string | null;
  hlsSubtitleTracks: SubtitleTrack[];

  actionFeedback: PlayerActionFeedback | null;

  showEndTime: boolean;

  // Heights (in px) for which HDR content has been detected. Persists across  quality switches.
  hdrHeights: Set<number>;

  playbackPrimed: boolean;

  behindLive: boolean;

  portrait: boolean;

  adBreakOverlay: boolean;

  airplayAvailable: boolean;
  airplayActive: boolean;

}

export class VideoPlayer extends ModuleComponent<VideoPlayerProps, VideoPlayerState> {

  private videoRef = createRef<HTMLVideoElement>();
  private containerRef = createRef<HTMLDivElement>();
  private progressFillRef = createRef<HTMLDivElement>();
  private miniProgressFillRef = createRef<HTMLDivElement>();
  private bufferFillRef = createRef<HTMLDivElement>();
  private timeLabelRef = createRef<HTMLSpanElement>();
  private seekTooltipRef = createRef<SeekTooltip>();

  private hls: HLS | null = null;
  private adBreaks = new AdBreakDetector();
  private adDebugTimer: ReturnType<typeof setInterval> | null = null;

  private mobile = isMobile();
  private nativeAirPlay = shouldUseNativeHls();
  private wakeLock = new ScreenWakeLock();
  private portraitQuery: MediaQueryList | null = null;

  private controlsTimer: ReturnType<typeof setTimeout> | null = null;
  private waitingTimer: ReturnType<typeof setTimeout> | null = null;
  private liveFailoverTimer: ReturnType<typeof setTimeout> | null = null;
  private feedbackTimer: ReturnType<typeof setTimeout> | null = null;
  private audioProbeTimer: ReturnType<typeof setTimeout> | null = null;
  private sourceReadyTimer: ReturnType<typeof setTimeout> | null = null;
  private holdPauseTimer: ReturnType<typeof setTimeout> | null = null;
  private holdPausePointerId: number | null = null;
  private holdPauseWasPlaying = false;
  private suppressNextVideoClick = false;

  private lastProgressReport = 0;
  private lastUiUpdate = 0;
  private feedbackId = 0;
  private durationMs = 0;
  private durationReported = false;
  private playbackErrorReported = false;
  private hlsRecoveryAttempts = 0;
  private mediaSessionArtworkRequest = 0;

  private static readonly MAX_HLS_RECOVERIES = 1;
  private static readonly MAX_LIVE_HLS_RECOVERIES = 8;
  private static readonly SOURCE_READY_TIMEOUT_MS = 8_000;
  private static readonly LIVE_BUFFER_FAILOVER_MS = 10_000;
  private static readonly HOLD_PAUSE_DELAY_MS = 220;
  private static readonly UP_NEXT_VISIBLE_LEAD_MS = 150_000;
  private static readonly UP_NEXT_COUNTDOWN_LEAD_MS = 60_000;

  state: VideoPlayerState = {

    playing: false,
    ...(this.mobile ? { volume: 1, muted: false } : readStoredVolume()),

    showControls: true,
    showOptions: false,
    showEpisodes: false,
    showSkipIntro: false,
    showUpNext: false,
    showUpNextMini: false,
    upNextCountdown: 0,

    fullscreen: false,
    portrait: readPortrait(),

    loading: true,
    seeking: false,
    holdPauseActive: false,

    activeSubtitleId: null,
    hlsSubtitleTracks: [],

    actionFeedback: null,

    showEndTime: false,

    hdrHeights: new Set(),

    playbackPrimed: false,

    behindLive: false,

    adBreakOverlay: false,

    airplayAvailable: this.nativeAirPlay && supportsAirPlayPicker(),
    airplayActive: false,

  };

  componentDidMount() {

    this.watch(settingsStore);

    this.attachSource();

    this.bindVideoEvents();

    document.addEventListener("keydown", this.onKeyDown);

    document.addEventListener("fullscreenchange", this.onFullscreenChange);

    this.syncSubtitlePreference();

    this.watchOrientation();

    this.setupBackgroundAudio();

    this.syncMediaSession();

  }

  componentDidUpdate(prev: VideoPlayerProps) {

    const srcChanged = prev.src !== this.props.src || prev.isHls !== this.props.isHls;
    const heightChanged = this.props.isHls && prev.selectedHeight !== this.props.selectedHeight;

    if (srcChanged) {

      this.unbindVideoEvents();

      this.destroyHls();

      this.attachSource();

      this.bindVideoEvents();

    } else if (heightChanged && this.hls) {

      this.applyHlsLevel(this.props.selectedHeight ?? 0, true);

    }

    if (prev.subtitleTracks !== this.props.subtitleTracks || prev.subtitlesEnabled !== this.props.subtitlesEnabled) {

      this.syncSubtitlePreference();

    }

    if (prev.title !== this.props.title || prev.subtitle !== this.props.subtitle || prev.episodeTitle !== this.props.episodeTitle || prev.poster !== this.props.poster) {

      this.syncMediaSession();

    }

    if (prev.intro !== this.props.intro) {

      const video = this.videoRef.current;
      if (video) this.checkSkipIntro(video.currentTime * 1000);

    }

    if (this.state.adBreakOverlay && !this.liveAdsEnabled()) {

      this.adBreaks.reset();

      this.setState({ adBreakOverlay: false }, () => this.applyAdAudio());

    }

    this.syncAdDebugLog();

  }

  componentWillUnmount() {

    this.unbindVideoEvents();

    this.destroyHls();

    this.clearTimers();

    document.removeEventListener("keydown", this.onKeyDown);
    document.removeEventListener("fullscreenchange", this.onFullscreenChange);
    document.removeEventListener("visibilitychange", this.onVisibilityChange);

    this.portraitQuery?.removeEventListener("change", this.onOrientationChange);

    window.visualViewport?.removeEventListener("resize", this.syncViewportHeight);
    window.visualViewport?.removeEventListener("scroll", this.syncViewportHeight);
    window.removeEventListener("resize", this.syncViewportHeight);

    this.wakeLock.disable();

    this.mediaSessionArtworkRequest += 1;
    clearMediaSession();

  }

  watchOrientation = () => {

    if (typeof window === "undefined") return;

    this.portraitQuery = window.matchMedia("(orientation: portrait)");

    this.setState({ portrait: this.portraitQuery.matches });

    this.portraitQuery.addEventListener("change", this.onOrientationChange);

    this.syncViewportHeight();

    window.visualViewport?.addEventListener("resize", this.syncViewportHeight);
    window.visualViewport?.addEventListener("scroll", this.syncViewportHeight);
    window.addEventListener("resize", this.syncViewportHeight);

  };

  syncViewportHeight = () => {

    const height = Math.max(
      window.innerHeight,
      window.visualViewport?.height ?? 0,
      document.documentElement.clientHeight,
    );

    document.documentElement.style.setProperty("--player-vvh", `${Math.round(height)}px`);

  };

  onOrientationChange = (event: MediaQueryListEvent) => {

    this.setState({ portrait: event.matches });

  };

  setupBackgroundAudio = () => {

    enableBackgroundAudio();

    document.addEventListener("visibilitychange", this.onVisibilityChange);

  };

  onVisibilityChange = () => {

    const video = this.videoRef.current;

    if (!video) return;

    if (document.visibilityState === "visible") {

      if (this.state.playing) this.wakeLock.enable();

      // Stall timers armed before the page froze would otherwise fire the
      // instant we resume and fail a stream that is actually fine.
      this.clearBuffering();

      return;

    }

    // AirPlay playback lives on the receiver — touching the local element while
    // backgrounded is what drops the stream a few seconds in.
    if (this.state.airplayActive) return;

    // iOS pauses the element when the PWA is backgrounded; resuming straight away keeps the audio running.
    if (this.state.playing && video.paused) {

      void video.play().catch(() => {});

    }

  };

  playFromMediaSession = () => {

    const video = this.videoRef.current;

    enableBackgroundAudio();

    if (video) void video.play().catch(() => {});

  };

  pauseFromMediaSession = () => {

    this.videoRef.current?.pause();

  };

  syncMediaSession = () => {

    const { title, subtitle, episodeTitle, poster, live, nextEpisode } = this.props;
    const artworkRequest = ++this.mediaSessionArtworkRequest;
    const metadata = {

      title: episodeTitle || title,
      artist: episodeTitle ? title : subtitle ?? "",
      album: live ? "Live TV" : subtitle ?? "",

    };

    setMediaSessionMetadata({

      ...metadata,
      artwork: live ? undefined : poster,

    });

    if (live && poster) {

      void getLiveMediaArtwork(poster, title).then((artwork) => {

        if (artworkRequest !== this.mediaSessionArtworkRequest) return;

        setMediaSessionMetadata({ ...metadata, artwork });

      });

    }

    setMediaSessionHandlers({

      onPlay: this.playFromMediaSession,
      onPause: this.pauseFromMediaSession,

      onSeekBackward: live ? undefined : () => this.seekBy(-10_000),
      onSeekForward: live ? undefined : () => this.seekBy(10_000),
      onSeekTo: live ? undefined : (positionMs) => this.seek(positionMs),

      onNextTrack: !live && nextEpisode ? () => this.props.onNextEpisode?.() : undefined,

    });

  };

  bindVideoEvents = () => {

    const video = this.videoRef.current;

    if (!video) return;

    video.addEventListener("timeupdate", this.onTimeUpdate);
    video.addEventListener("loadedmetadata", this.onLoadedMetadata);

    video.addEventListener("seeking", this.onSeeking);
    video.addEventListener("seeked", this.onSeeked);

    video.addEventListener("waiting", this.onWaiting);

    video.addEventListener("canplay", this.onCanPlay);
    video.addEventListener("playing", this.onPlaying);
    video.addEventListener("pause", this.onPause);

    video.addEventListener("volumechange", this.onVolumeChange);
    video.addEventListener("progress", this.onBufferProgress);

    prepareVideoForAirPlay(video);

    video.addEventListener("webkitplaybacktargetavailabilitychanged", this.onAirPlayAvailability);
    video.addEventListener("webkitcurrentplaybacktargetiswirelesschanged", this.onAirPlayTargetChange);

    this.setState({ airplayActive: isAirPlayActive(video) });

  };

  unbindVideoEvents = () => {

    const video = this.videoRef.current;

    if (!video) return;

    video.removeEventListener("timeupdate", this.onTimeUpdate);
    video.removeEventListener("loadedmetadata", this.onLoadedMetadata);

    video.removeEventListener("seeking", this.onSeeking);
    video.removeEventListener("seeked", this.onSeeked);

    video.removeEventListener("waiting", this.onWaiting);

    video.removeEventListener("canplay", this.onCanPlay);
    video.removeEventListener("playing", this.onPlaying);
    video.removeEventListener("pause", this.onPause);

    video.removeEventListener("volumechange", this.onVolumeChange);
    video.removeEventListener("progress", this.onBufferProgress);
    video.removeEventListener("error", this.onVideoError);

    video.removeEventListener("webkitplaybacktargetavailabilitychanged", this.onAirPlayAvailability);
    video.removeEventListener("webkitcurrentplaybacktargetiswirelesschanged", this.onAirPlayTargetChange);

  };

  onAirPlayAvailability = (event: Event) => {

    const availability = (event as WebKitPlaybackTargetAvailabilityEvent).availability;

    this.setState({

      airplayAvailable: availability === "available" || (this.nativeAirPlay && supportsAirPlayPicker()),
      airplayActive: isAirPlayActive(this.videoRef.current),

    });

  };

  onAirPlayTargetChange = () => {

    const active = isAirPlayActive(this.videoRef.current);

    if (active) {

      // Handing off makes the local element stall and often error; none of that
      // means the receiver failed, so drop the watchdogs rather than fail over.
      this.playbackErrorReported = false;
      this.clearBuffering();

    }

    this.setState({ airplayActive: active, loading: active ? false : this.state.loading });

  };

  openAirPlayPicker = () => {

    showAirPlayPicker(this.videoRef.current);

  };

  videoCrossOrigin = (): "anonymous" | "use-credentials" | undefined => {

    if (this.nativeAirPlay) return undefined;

    if (this.props.ambienceEnabled) return "anonymous";

    if (this.props.live && isProxiedStream(this.props.src)) return "use-credentials";

    return undefined;

  };

  onVideoError = () => {

    this.clearSourceReadyTimer();

    // While AirPlay owns playback the local element routinely errors (it has no
    // decoder attached). Reloading the source here kills the receiver's stream.
    if (this.state.airplayActive) {

      this.setState({ loading: false });

      return;

    }

    this.setState({ loading: false, playing: false });

    if (this.playbackErrorReported) return;

    this.playbackErrorReported = true;

    const video = this.videoRef.current;

    const currentMs = video && video.currentTime > 0 ? video.currentTime * 1000 : 0;
    const positionMs = currentMs || this.props.startPositionMs || 0;

    // VOD recovers by stepping down quality; live (no onPlaybackError) has no fallback, so  surface a fatal error instead of leaving the player stuck in a paused state.
    if (this.props.onPlaybackError) {

      this.props.onPlaybackError(positionMs);

      return;

    }

    this.props.onFatalError?.();

  };

  onLoadedMetadata = () => {

    if (this.props.live) return;

    const video = this.videoRef.current;

    if (!video) return;

    this.durationMs = (video.duration || 0) * 1000;

    this.updateProgressUI(video.currentTime * 1000, this.durationMs);

    this.updateBufferUI();

    if (this.durationMs > 0 && !this.durationReported) {

      this.durationReported = true;
      this.props.onDurationReady?.(this.durationMs);

    }

    this.checkSkipIntro(video.currentTime * 1000);

  };

  onTimeUpdate = () => {

    const video = this.videoRef.current;

    if (!video || this.state.seeking) return;

    if (this.props.live) {

      this.syncAdBreakOverlay();

      return;

    }

    const currentMs = video.currentTime * 1000;

    const durationMs = (video.duration || 0) * 1000;

    this.durationMs = durationMs;

    const now = performance.now();

    if (now - this.lastUiUpdate > 250) {

      this.lastUiUpdate = now;

      this.updateProgressUI(currentMs, durationMs);

      setMediaSessionPosition(durationMs, currentMs, video.playbackRate);
      this.updateBufferUI();

    }

    if (now - this.lastProgressReport > 2000) {

      this.lastProgressReport = now;
      this.props.onProgress?.(currentMs, durationMs);

    }

    this.checkSkipIntro(currentMs);

    this.checkCredits(currentMs, durationMs);

  };

  updateProgressUI = (currentMs: number, durationMs: number) => {

    const pct = durationMs > 0 ? (currentMs / durationMs) * 100 : 0;

    if (this.progressFillRef.current) {

      this.progressFillRef.current.style.width = `${pct}%`;

    }

    if (this.miniProgressFillRef.current) {

      this.miniProgressFillRef.current.style.width = `${pct}%`;

    }

    if (this.timeLabelRef.current) {

      if (this.state.showEndTime && durationMs > 0) {

        const remainingMs = Math.max(0, durationMs - currentMs);
        const endTime = new Date(Date.now() + remainingMs);
        const formatted = endTime.toLocaleTimeString([], { hour: "numeric", minute: "2-digit" });

        this.timeLabelRef.current.textContent = `Done at ${formatted}`;

      } else {

        this.timeLabelRef.current.textContent = `${formatDuration(currentMs)} / ${formatDuration(durationMs)}`;

      }

    }

  };

  onBufferProgress = () => {

    this.updateBufferUI();

  };

  updateBufferUI = () => {

    if (this.props.live || !this.bufferFillRef.current) return;

    const video = this.videoRef.current;

    if (!video || !Number.isFinite(video.duration) || video.duration <= 0 || video.buffered.length === 0) {

      this.bufferFillRef.current.style.width = "0%";
      return;

    }

    const duration = video.duration;

    let bufferedEnd = 0;

    for (let i = 0; i < video.buffered.length; i += 1) {

      const start = video.buffered.start(i);
      const end = video.buffered.end(i);

      if (video.currentTime >= start && video.currentTime <= end) {

        bufferedEnd = end;
        break;

      }

      bufferedEnd = Math.max(bufferedEnd, end);
    }

    const pct = Math.max(0, Math.min(100, (bufferedEnd / duration) * 100));

    this.bufferFillRef.current.style.width = `${pct}%`;

  };

  onSeeking = () => {

    this.setState({ seeking: true, loading: true });

  };

  onSeeked = () => {

    const video = this.videoRef.current;

    if (video) {

      this.updateProgressUI(video.currentTime * 1000, this.durationMs);

    }

    this.setState({ seeking: false }, () => this.syncAdBreakOverlay());

  };

  onWaiting = () => {

    if (!this.waitingTimer && !this.state.loading) {

      this.waitingTimer = setTimeout(() => {

        this.waitingTimer = null;
        this.setState({ loading: true });

      }, 600);

    }

    if (this.props.live && this.props.onPlaybackError && !this.liveFailoverTimer && !this.playbackErrorReported && !this.state.airplayActive) {

      this.liveFailoverTimer = setTimeout(() => {

        this.liveFailoverTimer = null;
        this.triggerLiveFailover();

      }, VideoPlayer.LIVE_BUFFER_FAILOVER_MS);

    }

  };

  triggerLiveFailover = () => {

    if (!this.props.live || this.playbackErrorReported) return;

    // A backgrounded page stalls on its own, and the receiver keeps playing
    // regardless — switching sources from here only breaks a working stream.
    if (this.state.airplayActive || document.visibilityState !== "visible") {

      this.clearBuffering();

      return;

    }

    this.playbackErrorReported = true;
    this.clearBuffering();
    this.props.onPlaybackError?.(0);

  };

  clearBuffering = () => {

    if (this.waitingTimer) {

      clearTimeout(this.waitingTimer);
      this.waitingTimer = null;

    }

    if (this.liveFailoverTimer) {

      clearTimeout(this.liveFailoverTimer);
      this.liveFailoverTimer = null;

    }

    if (this.state.loading && !this.state.seeking) {

      this.setState({ loading: false });

    }

    this.clearSourceReadyTimer();

  };

  onCanPlay = () => {

    this.clearBuffering();
    this.setState({ seeking: false, playbackPrimed: true });

  };

  onPlaying = () => {

    enableBackgroundAudio();

    this.clearBuffering();
    this.wakeLock.enable();
    this.setState({ playing: true, playbackPrimed: true });

    this.syncAdBreakOverlay();

    this.syncMediaSession();

    setMediaSessionPlaybackState("playing");
    this.props.onPlaybackStateChange?.(true);

  };

  onPause = () => {

    this.wakeLock.disable();

    this.setState({

      playing: false,
      behindLive: this.props.live ? true : this.state.behindLive,

    });

    setMediaSessionPlaybackState("paused");

    this.props.onPlaybackStateChange?.(false);

  };

  onVolumeChange = () => {

    const video = this.videoRef.current;

    if (!video || this.mobile) return;

    if (this.state.adBreakOverlay) return;

    try {

      localStorage.setItem("player:volume", String(video.volume));
      localStorage.setItem("player:muted", String(video.muted));

    } catch { /* storage unavailable */ }

    this.setState({ volume: video.volume, muted: video.muted });

  };

  clearTimers = () => {

    if (this.controlsTimer) clearTimeout(this.controlsTimer);
    if (this.waitingTimer) clearTimeout(this.waitingTimer);
    if (this.liveFailoverTimer) clearTimeout(this.liveFailoverTimer);
    if (this.feedbackTimer) clearTimeout(this.feedbackTimer);
    if (this.audioProbeTimer) clearTimeout(this.audioProbeTimer);
    if (this.sourceReadyTimer) clearTimeout(this.sourceReadyTimer);
    if (this.holdPauseTimer) clearTimeout(this.holdPauseTimer);

    this.stopAdDebugLog();

    this.waitingTimer = null;
    this.liveFailoverTimer = null;
    this.feedbackTimer = null;
    this.audioProbeTimer = null;
    this.sourceReadyTimer = null;
    this.holdPauseTimer = null;
    this.holdPausePointerId = null;

  };

  clearSourceReadyTimer = () => {

    if (!this.sourceReadyTimer) return;

    clearTimeout(this.sourceReadyTimer);
    this.sourceReadyTimer = null;

  };

  hlsManifestHasAudio = (hls: HLS): boolean => {

    if ((hls.audioTracks?.length ?? 0) > 0) return true;

    return hls.levels.some((level) => {

      const codecs = level.attrs?.CODECS ?? "";
      return /mp4a\.|ac-3|ec-3|opus/i.test(codecs);

    });

  };

  applyHlsLevel = (selectedHeight: number, allowInexact = false) => {

    if (!this.hls || this.hls.levels.length === 0) return;

    const levels = this.hls.levels as HlsLevelLike[];
    const target = bestSupportedHlsLevel(levels, selectedHeight);

    if (!target) {

      this.onVideoError();
      return;

    }

    if (!this.props.live && selectedHeight > 0 && !target.isExact && !allowInexact) {

      this.onVideoError();
      return;

    }

    if (!this.props.live) {

      this.hls.currentLevel = target.index;
      this.hls.nextLevel = target.index;
      this.hls.autoLevelCapping = target.index;

    }

    this.ensureHlsAudio();

  };

  ensureHlsAudio = () => {

    if (!this.hls) return;

    const tracks = this.hls.audioTracks ?? [];

    if (tracks.length > 0) {

      const defaultIndex = tracks.findIndex((track) => track.default);
      const nextIndex = defaultIndex >= 0 ? defaultIndex : 0;

      if (this.hls.audioTrack !== nextIndex) {

        this.hls.audioTrack = nextIndex;

      }

      return;
    }

    if (!this.props.live && !this.hlsManifestHasAudio(this.hls)) {

      this.onVideoError();

    }

  };

  scheduleAudioProbe = () => {

    if (this.props.live || this.audioProbeTimer) return;

    this.audioProbeTimer = setTimeout(() => {

      this.audioProbeTimer = null;

      const video = this.videoRef.current;
      if (!video || video.paused || !this.hls) return;

      if (this.hlsManifestHasAudio(this.hls)) return;

      this.onVideoError();

    }, 3_500);

  };

  destroyHls = () => {

    if (this.hls) {

      this.hls.destroy();
      this.hls = null;

    }

    this.adBreaks.reset();

  };

  liveAdsEnabled = () => {

    return !!this.props.live;

  };

  syncAdDebugLog = () => {

    if (this.liveAdsEnabled()) {

      this.startAdDebugLog();

    } else {

      this.stopAdDebugLog();

    }

  };

  startAdDebugLog = () => {

    if (this.adDebugTimer) {

      return;

    }

    const log = () => {

      const playhead = this.videoRef.current?.currentTime;

      console.log("[ad-break]", this.adBreaks.debugLine(playhead));

    };

    log();
    this.adDebugTimer = setInterval(log, 15_000);

  };

  stopAdDebugLog = () => {

    if (!this.adDebugTimer) {

      return;

    }

    clearInterval(this.adDebugTimer);
    this.adDebugTimer = null;

  };

  applyAdAudio = () => {

    const video = this.videoRef.current;

    if (!video) {

      return;

    }

    const adMute = this.state.adBreakOverlay;
    video.muted = this.state.muted || adMute;

  };

  dismissAdBreak = () => {

    this.adBreaks.dismiss();

    this.setState({ adBreakOverlay: false }, () => this.applyAdAudio());

  };

  onHlsFragLoaded = (_event: string, data: { frag?: { duration?: number; sn?: number | string; type?: string; start?: number }; payload?: ArrayBuffer | Uint8Array }) => {

    if (!this.liveAdsEnabled() || !data.payload) {

      return;

    }

    const type = data.frag?.type;

    if (type === "audio" || type === "subtitle") {

      return;

    }

    const duration = data.frag?.duration ?? 0;
    const start = data.frag?.start;

    this.adBreaks.addFragment(data.payload, duration, data.frag?.sn, start);

    this.syncAdBreakOverlay();

  };

  syncAdBreakOverlay = () => {

    if (!this.liveAdsEnabled()) {

      return;

    }

    const overlay = this.adBreaks.visibleAt(this.videoRef.current?.currentTime ?? 0);

    this.setState((prev) => {

      if (overlay === prev.adBreakOverlay) {

        return null;

      }

      return { adBreakOverlay: overlay };

    }, () => this.applyAdAudio());

  };

  syncHlsSubtitles = () => {

    if (!this.hls) return;

    const tracks = this.hls.subtitleTracks ?? [];

    const hlsSubtitleTracks: SubtitleTrack[] = tracks.map((track, index) => ({

      id: `hls-${index}`,
      label: track.name || `Track ${index + 1}`,
      language: track.lang || "und",
      format: "vtt",
      proxyUrl: track.url || "",
      source: "hls",

    }));

    this.setState({ hlsSubtitleTracks }, () => {

      if (this.props.subtitlesEnabled && !this.state.activeSubtitleId) {

        this.syncSubtitlePreference();

      }

    });

  };

  preferredSubtitleTrackId = (): string | null => {

    if (!this.props.subtitlesEnabled) return null;

    const tracks = this.allSubtitleTracks();
    if (tracks.length === 0) return null;

    const external = tracks.find((track) => track.source !== "hls");

    return (external ?? tracks[0]).id;

  };

  syncSubtitlePreference = () => {

    const trackId = this.preferredSubtitleTrackId();

    if (trackId === this.state.activeSubtitleId) return;

    if (!this.props.subtitlesEnabled) {

      if (this.state.activeSubtitleId !== null) {

        this.applySubtitleSelection(null, false);

      }

      return;
    }

    if (trackId) {

      this.applySubtitleSelection(trackId, false);

    }

  };

  applySubtitleSelection = (trackId: string | null, persist = true) => {

    const allTracks = this.allSubtitleTracks();

    const track = allTracks.find((item) => item.id === trackId) ?? null;

    if (this.hls) {

      if (track?.source === "hls") {

        const index = this.state.hlsSubtitleTracks.findIndex((item) => item.id === trackId);

        this.hls.subtitleTrack = index;
        this.hls.subtitleDisplay = true;

      } else {

        this.hls.subtitleTrack = -1;
        this.hls.subtitleDisplay = false;

      }

    }

    this.setState({ activeSubtitleId: trackId });

    if (persist) {

      this.props.onSubtitlesEnabledChange?.(trackId !== null);

    }

  };

  allSubtitleTracks = (): SubtitleTrack[] => [

    ...(this.props.subtitleTracks ?? []),
    ...this.state.hlsSubtitleTracks,

  ];

  activeSubtitleTrack = (): SubtitleTrack | null => {

    const { activeSubtitleId } = this.state;
    if (!activeSubtitleId) return null;

    const track = this.allSubtitleTracks().find((item) => item.id === activeSubtitleId) ?? null;
    if (track?.source === "hls") return null;

    return track;

  };

  attachSource = async () => {

    const video = this.videoRef.current;

    const { src, isHls, lowLatency, startPositionMs } = this.props;

    if (!video || !src.trim() || !isWebPlayableUrl(src)) {

      if (src.trim() && !isWebPlayableUrl(src)) {

        this.onVideoError();

      } else {

        this.setState({ loading: false, playing: false });

      }

      return;
    }

    this.durationReported = false;
    this.playbackErrorReported = false;

    this.hlsRecoveryAttempts = 0;

    this.adBreaks.reset();
    this.syncAdDebugLog();

    this.setState({

      loading: true,
      playbackPrimed: false,
      behindLive: false,
      adBreakOverlay: false,

      portrait: readPortrait(),

      showUpNext: false,
      showUpNextMini: false,
      showSkipIntro: false,

      seeking: false,
      holdPauseActive: false,

      hlsSubtitleTracks: [],

    });

    this.clearSourceReadyTimer();

    this.sourceReadyTimer = setTimeout(() => {

      const current = this.videoRef.current;

      if (current === video && this.state.loading) {

        this.onVideoError();

      }

    }, VideoPlayer.SOURCE_READY_TIMEOUT_MS);

    video.removeEventListener("error", this.onVideoError);

    video.addEventListener("error", this.onVideoError);

    const proxied = isProxiedStream(src);

    if (this.nativeAirPlay) {

      video.removeAttribute("crossorigin");

    } else if (this.props.live) {

      if (this.props.ambienceEnabled) {

        video.crossOrigin = "anonymous";

      } else if (proxied) {

        video.crossOrigin = "use-credentials";

      } else {

        video.removeAttribute("crossorigin");

      }

    } else if (this.props.ambienceEnabled || proxied) {

      video.crossOrigin = proxied && !this.props.ambienceEnabled ? "use-credentials" : "anonymous";

    } else {

      video.removeAttribute("crossorigin");

    }

    const onReady = () => {

      if (!this.props.live && startPositionMs && startPositionMs > 0) {

        video.currentTime = startPositionMs / 1000;

      }

      video.volume = this.state.volume;
      video.muted = this.state.muted || this.state.adBreakOverlay;

      // Don't mark loading:false here — onCanPlay/onPlaying handle that so the spinner stays visible through any initial seek without oscillating.
      video.play().catch(() => this.setState({ loading: false, playing: false }));

      this.onLoadedMetadata();
      this.syncHlsSubtitles();

    };

    if (isHls) {

      prepareVideoForAirPlay(video);

      if (this.nativeAirPlay || shouldUseNativeHls(video)) {

        video.src = src;
        video.addEventListener("loadedmetadata", onReady, { once: true });

        return;

      }

      const { default: HlsConstructor } = await import("hls.js");

      if (this.props.src !== src || this.videoRef.current !== video) return;

      if (!HlsConstructor.isSupported()) {

        if (video.canPlayType("application/vnd.apple.mpegurl")) {

          video.src = src;
          video.addEventListener("loadedmetadata", onReady, { once: true });

        } else {

          this.onVideoError();

        }

        return;
      }

      const live = !!this.props.live;
      const maxRecoveries = live ? VideoPlayer.MAX_LIVE_HLS_RECOVERIES : VideoPlayer.MAX_HLS_RECOVERIES;

      // Live TV here is regular ~10s HLS (not LL-HLS); match ntv's buffer profile.
      this.hls = new HlsConstructor({

        enableWorker: true,

        lowLatencyMode: !!lowLatency && !live,

        maxBufferLength: live ? 30 : lowLatency ? 12 : 45,
        maxMaxBufferLength: live ? 60 : lowLatency ? 24 : 90,

        backBufferLength: live ? 0 : 30,
        liveSyncDurationCount: live ? 3 : undefined,

        xhrSetup: proxied ? (xhr) => { xhr.withCredentials = true; } : undefined,

      });

      this.hls.on(HlsConstructor.Events.FRAG_LOADED, this.onHlsFragLoaded);

      this.hls.loadSource(src);

      this.hls.attachMedia(video);

      this.hls.on(HlsConstructor.Events.MANIFEST_PARSED, () => {

        const hls = this.hls;

        if (hls && hls.levels.length > 0) {

          const selectedHeight = this.props.selectedHeight ?? 0;

          this.applyHlsLevel(selectedHeight);

          if (selectedHeight > 0) {

            const levels = hls.levels as HlsLevelLike[];
            const isHdr = levels.some((level) => isHdrLevel(level));

            if (isHdr) {

              this.setState((prev) => ({

                hdrHeights: new Set([...prev.hdrHeights, selectedHeight]),

              }));

            }

          }

        }

        this.ensureHlsAudio();

        onReady();

        this.scheduleAudioProbe();

      });

      this.hls.on(HlsConstructor.Events.AUDIO_TRACKS_UPDATED, this.ensureHlsAudio);

      this.hls.on(HlsConstructor.Events.SUBTITLE_TRACKS_UPDATED, this.syncHlsSubtitles);

      this.hls.on(HlsConstructor.Events.ERROR, (_, data) => {

        if (!data.fatal || !this.hls) return;

        if (data.type === HlsConstructor.ErrorTypes.NETWORK_ERROR) {

          if (++this.hlsRecoveryAttempts > maxRecoveries) {

            this.onVideoError();
            return;

          }

          this.hls.startLoad();

          return;
        }

        if (data.type === HlsConstructor.ErrorTypes.MEDIA_ERROR) {

          if (++this.hlsRecoveryAttempts > maxRecoveries) {

            this.onVideoError();
            return;

          }

          this.hls.recoverMediaError();

          return;

        }

        this.onVideoError();

      });

    } else {

      prepareVideoForAirPlay(video);

      video.src = src;

      video.addEventListener("loadedmetadata", onReady, { once: true });
    }

  };

  checkSkipIntro = (currentMs: number) => {

    const { intro, skipIntroEnabled } = this.props;

    if (!skipIntroEnabled || !hasIntroWindow(intro)) {

      if (this.state.showSkipIntro) this.setState({ showSkipIntro: false });
      return;

    }

    const inIntro = isInIntroWindow(intro, currentMs);

    if (inIntro !== this.state.showSkipIntro) {

      this.setState({ showSkipIntro: inIntro });

    }

  };

  checkCredits = (currentMs: number, durationMs: number) => {

    const { nextEpisode, autoPlayNext, intro } = this.props;

    if (!autoPlayNext || !nextEpisode || durationMs <= 0) return;

    const visibleAt = durationMs - VideoPlayer.UP_NEXT_VISIBLE_LEAD_MS;
    const countdownAt = durationMs - VideoPlayer.UP_NEXT_COUNTDOWN_LEAD_MS;

    if (currentMs < visibleAt) {

      if (this.state.showUpNext || this.state.showUpNextMini) {

        this.setState({ showUpNext: false, showUpNextMini: false, upNextCountdown: 0 });

      }

      return;

    }

    const creditsStartMs = intro?.creditsStartMs;
    const inCredits = creditsStartMs != null && currentMs >= creditsStartMs;
    const inCountdown = currentMs >= countdownAt;
    const showFull = inCredits || inCountdown;

    if (showFull) {

      const secondsLeft = inCountdown ? Math.max(0, Math.ceil((durationMs - currentMs) / 1000)) : 0;

      if (!this.state.showUpNext) {

        this.setState({ showUpNext: true, showUpNextMini: false, upNextCountdown: secondsLeft });

      } else if (this.state.upNextCountdown !== secondsLeft) {

        this.setState({ upNextCountdown: secondsLeft });

      }

    } else {

      if (!this.state.showUpNextMini) {

        this.setState({ showUpNextMini: true, showUpNext: false, upNextCountdown: 0 });

      }

    }

  };

  liveEdgePosition = (): number | null => {

    const video = this.videoRef.current;

    if (this.hls?.liveSyncPosition != null) {

      return this.hls.liveSyncPosition;

    }

    if (!video || video.seekable.length === 0) return null;

    return video.seekable.end(video.seekable.length - 1);

  };

  jumpToLive = () => {

    const video = this.videoRef.current;

    if (!video || !this.props.live) return;

    const edge = this.liveEdgePosition();

    if (edge == null || !Number.isFinite(edge)) return;

    video.currentTime = Math.max(0, edge - 0.25);

  };

  jumpToLiveAndPlay = () => {

    const video = this.videoRef.current;

    if (!video || !this.props.live) return;

    this.jumpToLive();

    if (video.paused) {

      video.play()
        .then(() => this.setState({ playing: true, behindLive: false }))
        .catch(() => this.setState({ playing: false }));

    } else {

      this.setState({ behindLive: false });

    }

  };

  togglePlay = () => {

    const video = this.videoRef.current;

    if (!video) return;

    if (video.paused) {

      video.play()
        .then(() => this.setState({ playing: true }))
        .catch(() => this.setState({ playing: false }));

    } else {

      video.pause();
      this.setState({ playing: false, behindLive: this.props.live ? true : this.state.behindLive });

    }

  };

  beginHoldPause = (event: ReactPointerEvent<HTMLVideoElement>) => {

    if (event.pointerType === "mouse" && event.button !== 0) return;
    if (this.holdPauseTimer || this.holdPausePointerId !== null) return;

    const video = this.videoRef.current;

    if (!video || video.paused || this.state.loading || this.state.seeking) return;

    this.holdPausePointerId = event.pointerId;
    this.holdPauseWasPlaying = !video.paused;

    try {

      event.currentTarget.setPointerCapture(event.pointerId);

    } catch {

      /* pointer capture is optional */

    }

    this.holdPauseTimer = setTimeout(() => {

      this.holdPauseTimer = null;

      const current = this.videoRef.current;

      if (!current || !this.holdPauseWasPlaying || current.paused) return;

      this.suppressNextVideoClick = true;
      this.setState({ holdPauseActive: true, showControls: false });
      current.pause();

    }, VideoPlayer.HOLD_PAUSE_DELAY_MS);

  };

  endHoldPause = (event?: ReactPointerEvent<HTMLVideoElement>) => {

    if (event && this.holdPausePointerId !== event.pointerId) return;

    if (this.holdPauseTimer) {

      clearTimeout(this.holdPauseTimer);
      this.holdPauseTimer = null;

    }

    const wasActive = this.state.holdPauseActive;
    const shouldResume = wasActive && this.holdPauseWasPlaying;

    this.holdPausePointerId = null;
    this.holdPauseWasPlaying = false;

    if (!wasActive) return;

    event?.preventDefault();
    event?.stopPropagation();

    this.setState({ holdPauseActive: false });

    if (shouldResume) {

      this.videoRef.current?.play().then(() => this.setState({ playing: true })).catch(() => this.setState({ playing: false }));

    }

  };

  cancelHoldPause = (event: ReactPointerEvent<HTMLVideoElement>) => {

    this.endHoldPause(event);

  };

  setVolume = (volume: number) => {

    const clamped = Math.max(0, Math.min(volume, 1));
    const muted = clamped === 0;

    try {

      localStorage.setItem("player:volume", String(clamped));
      localStorage.setItem("player:muted", String(muted));

    } catch { /* storage unavailable */ }

    this.setState({ volume: clamped, muted });

    const video = this.videoRef.current;

    if (video) {

      video.volume = clamped;
      video.muted = muted || this.state.adBreakOverlay;

    }

  };

  toggleMute = () => {

    const { muted, volume } = this.state;

    if (muted || volume === 0) {

      const nextVolume = volume > 0 ? volume : 0.8;

      this.setVolume(nextVolume);

    } else {

      const video = this.videoRef.current;

      try {

        localStorage.setItem("player:muted", "true");

      } catch { /* storage unavailable */ }

      this.setState({ muted: true });

      if (video) video.muted = true;

    }

  };

  seek = (ms: number) => {

    const video = this.videoRef.current;

    if (!video || !Number.isFinite(ms)) return;

    const duration = Number.isFinite(video.duration) && video.duration > 0 ? video.duration * 1000 : this.durationMs;
    const clamped = Math.max(0, duration > 0 ? Math.min(ms, duration) : ms);

    video.currentTime = clamped / 1000;

    this.updateProgressUI(clamped, this.durationMs);

    this.setState({ seeking: true, loading: true });

  };

  scrubberRatioFromEvent = (e: ReactMouseEvent<HTMLElement>) => {

    const rect = e.currentTarget.getBoundingClientRect();

    if (rect.width <= 0) return 0;

    return Math.max(0, Math.min(1, (e.clientX - rect.left) / rect.width));

  };

  onScrubberMove = (e: ReactMouseEvent<HTMLDivElement>) => {

    if (this.props.live || this.durationMs <= 0) return;

    this.seekTooltipRef.current?.update(this.scrubberRatioFromEvent(e), this.durationMs);

  };

  onScrubberLeave = () => {

    this.seekTooltipRef.current?.hide();

  };

  seekBy = (deltaMs: number) => {

    if (this.props.live) return;

    const video = this.videoRef.current;

    if (!video) return;

    this.seek(video.currentTime * 1000 + deltaMs);

    this.showActionFeedback({

      kind: "seek",
      direction: deltaMs < 0 ? -1 : 1,

      label: `${Math.abs(deltaMs / 1000)}s`,

    });

  };

  toggleTimeDisplay = () => {

    this.setState(

      (s) => ({ showEndTime: !s.showEndTime }),
      () => {

        const video = this.videoRef.current;
        if (video) this.updateProgressUI(video.currentTime * 1000, this.durationMs);

      },

    );

  };

  showActionFeedback = (feedback: Omit<PlayerActionFeedback, "id">) => {

    if (this.feedbackTimer) clearTimeout(this.feedbackTimer);

    this.setState({ actionFeedback: { ...feedback, id: ++this.feedbackId } });

    this.feedbackTimer = setTimeout(() => {

      this.feedbackTimer = null;
      this.setState({ actionFeedback: null });

    }, 500);

  };

  onKeyDown = (event: KeyboardEvent) => {

    const target = event.target as HTMLElement | null;

    const tagName = target?.tagName;

    if (tagName === "INPUT" || tagName === "TEXTAREA" || target?.isContentEditable) return;

    switch (event.key) {

      case "ArrowLeft":

        event.preventDefault();

        this.seekBy(-5_000); // 5s backward
        this.showControlsTemporarily();

        break;

      case "ArrowRight":

        event.preventDefault();

        this.seekBy(5_000); // 5s forward
        this.showControlsTemporarily();

        break;

      case "ArrowUp":
      case "ArrowDown": {

        if (this.mobile) break;

        event.preventDefault();

        const delta = event.key === "ArrowUp" ? 0.05 : -0.05; // 5% volume change in either direction
        const nextVolume = Math.max(0, Math.min(this.state.volume + delta, 1));

        this.setVolume(nextVolume);

        this.showActionFeedback({

          kind: "volume",

          direction: delta > 0 ? 1 : -1,
          label: `${Math.round(nextVolume * 100)}%`,

        });

        this.showControlsTemporarily();

        break;

      }

      case " ": // space
      case "k": // k for "keyboard play/pause"
      case "K": // shift + k for "keyboard play/pause"

        event.preventDefault();

        this.togglePlay();

        this.showControlsTemporarily();

        break;

      default:

        break;

    }

  };

  skipIntro = () => {

    const { intro } = this.props;

    if (intro?.introEndMs) {

      this.seek(intro.introEndMs);
      this.setState({ showSkipIntro: false });

    }

  };

  onEnded = () => {

    const { nextEpisode, autoPlayNext, onEnded, onNextEpisode } = this.props;

    this.setState({ playing: false });

    if (autoPlayNext && nextEpisode) {

      onNextEpisode?.();

    } else {

      onEnded?.();

    }

  };

  onFullscreenChange = () => {

    this.setState({ fullscreen: !!document.fullscreenElement });

  };

  toggleFullscreen = () => {

    const el = this.containerRef.current;

    if (!el) return;

    if (!document.fullscreenElement) {

      el.requestFullscreen();

    } else {

      document.exitFullscreen();

    }

  };

  enterNativeFullscreen = () => {

    const video = this.videoRef.current;

    if (!video) return;

    const native = nativeFullscreenVideo(video);

    if (!native) {

      if (typeof video.requestFullscreen === "function") void video.requestFullscreen().catch(() => {});

      return;

    }

    // iOS refuses the call until metadata lands, which is a beat after hls.js attaches the manifest.
    if (native.readyState < HTMLMediaElement.HAVE_METADATA) {

      native.addEventListener("loadedmetadata", this.openNativeFullscreen, { once: true });

      return;

    }

    this.openNativeFullscreen();

  };

  openNativeFullscreen = () => {

    nativeFullscreenVideo(this.videoRef.current)?.webkitEnterFullscreen?.();

  };

  showControlsTemporarily = () => {

    this.setState({ showControls: true });

    if (this.controlsTimer) clearTimeout(this.controlsTimer);

    if (this.props.streamResolving) {

      return;

    }

    this.controlsTimer = setTimeout(() => {

      if (this.state.playing) this.setState({ showControls: false });

    }, 3_000);

  };

  toggleOptions = () => {

    this.setState((s) => ({

      showOptions: !s.showOptions,
      showEpisodes: s.showOptions ? s.showEpisodes : false,

    }));

  };

  closeOptionsFromOutside = (event: PointerEvent) => {

    if (event.target === this.videoRef.current) {

      this.suppressNextVideoClick = true;

    }

    this.setState({ showOptions: false });

  };

  toggleEpisodes = () => {

    this.setState((s) => {

      const nextShowEpisodes = !s.showEpisodes;

      if (nextShowEpisodes) {

        const season = this.props.menuSeason ?? this.props.currentSeason ?? 1;
        this.props.onSeasonChange?.(season);

      }

      return {

        showEpisodes: nextShowEpisodes,
        showOptions: s.showEpisodes ? s.showOptions : false,
        showControls: true,

      };

    });

  };

  render() {

    const { title, subtitle, episodeTitle, description, poster, qualities = [], selectedHeight = 1080, preferredHeight, nextEpisode, onBack, ambienceEnabled, live, compact, onReturn, onDismiss, onQualityChange, onOpenSettings, seasons, episodes, currentSeason, currentEpisode, menuSeason, episodesLoading, onSeasonChange, onEpisodeSelect, streamResolving, sourceProviders, selectedSourceKey, sourceSwitching, onSourceChange, } = this.props;
    const { playing, muted, volume, showControls, showOptions, showEpisodes, showSkipIntro, showUpNext, showUpNextMini, upNextCountdown, fullscreen, loading, seeking, holdPauseActive, activeSubtitleId, actionFeedback, hdrHeights, playbackPrimed, behindLive, portrait, adBreakOverlay, airplayAvailable, airplayActive, } = this.state;

    const resolving = !!streamResolving;
    const portraitMobile = portrait && this.mobile;

    // Mobile menus are full-screen sheets, so the chrome hides behind them; desktop menus sit inside the control bar and die with it.
    const menuOpen = showOptions || showEpisodes;
    const overlayMenuOpen = this.mobile && menuOpen;
    const anchoredMenuOpen = !this.mobile && menuOpen;

    const controlsPinned = resolving || loading || seeking;
    const effectiveShowControls = (showControls || controlsPinned || anchoredMenuOpen) && !overlayMenuOpen;

    const showAdBreakOverlay = !compact && adBreakOverlay && !!live;
    const showPauseOverlay = !playing && !loading && !seeking && !holdPauseActive && !showEpisodes && !showOptions && !resolving && playbackPrimed && !!this.props.src.trim() && settingsStore.settings?.disablePauseOverlay !== true && !showAdBreakOverlay;

    const qualityEnabled = !live && qualities.length > 0 && !!onQualityChange;
    const sourceEnabled = !!live && (sourceProviders?.length ?? 0) > 0 && !!onSourceChange;
    const episodesEnabled = !live && !!onEpisodeSelect && !!onSeasonChange;

    const episodePicker = episodesEnabled ? (

      <EpisodePickerPanel

        open={showEpisodes}
        compact={this.mobile}
        seasons={seasons ?? []}
        episodes={episodes ?? []}

        currentSeason={currentSeason}
        currentEpisode={currentEpisode}
        menuSeason={menuSeason ?? currentSeason ?? 1}
        episodesLoading={episodesLoading}

        onClose={() => this.setState({ showEpisodes: false })}
        onSeasonChange={onSeasonChange}
        onEpisodeSelect={(season, episode) => {

          this.setState({ showEpisodes: false });

          onEpisodeSelect(season, episode);

        }}

      />

    ) : null;

    const pauseLayout = live ? "live" as const : episodeTitle ? "episode" as const : "movie" as const;

    const videoHandlers = {

      onEnded: this.onEnded,
      onPointerDown: this.beginHoldPause,
      onPointerUp: this.endHoldPause,
      onPointerCancel: this.cancelHoldPause,
      onClick: (e: ReactMouseEvent<HTMLVideoElement>) => {

        e.stopPropagation();

        if (this.suppressNextVideoClick) {

          this.suppressNextVideoClick = false;
          return;

        }

        if (this.mobile) {

          this.showControlsTemporarily();
          return;

        }

        this.togglePlay();

      },

    };

    return (

      <div className={cn("relative flex w-full flex-col bg-black", !compact && "player-portrait-band h-full min-h-full")}

        ref={this.containerRef}
        onMouseMove={compact ? undefined : this.showControlsTemporarily}
        onClick={compact ? undefined : this.showControlsTemporarily}

      >

        <div className={cn("relative w-full", compact ? "aspect-video overflow-hidden" : "min-h-0 flex-1")}>

        {!compact && (

          <AmbienceLayer

            videoRef={this.videoRef as RefObject<HTMLVideoElement>}
            enabled={!!ambienceEnabled && !airplayActive && !this.nativeAirPlay}

          />

        )}

        <video

          className={cn("relative z-10 h-full w-full object-contain object-center", showAdBreakOverlay && "scale-105 blur-2xl")}
          ref={this.videoRef}
          playsInline
          disablePictureInPicture
          disableRemotePlayback={false}
          poster={poster}
          crossOrigin={this.videoCrossOrigin()}
          {...videoHandlers}

        />

        {!compact && (

          <SubtitleDisplay

            videoRef={this.videoRef}
            compact={this.mobile}
            track={showPauseOverlay || showAdBreakOverlay || showEpisodes ? null : this.activeSubtitleTrack()}

          />

        )}

        {!compact && <PauseOverlay

          visible={showPauseOverlay}
          poster={poster}
          layout={pauseLayout}

          title={title}
          subtitle={subtitle}
          episodeTitle={episodeTitle}
          description={description}

          onResume={this.togglePlay}
          pausedAt={this.videoRef.current ? this.videoRef.current.currentTime : 0}
          totalDuration={this.videoRef.current ? this.videoRef.current.duration : 0}
          simplified={this.mobile}

        />}

        {!compact && <AdBreakOverlay

          visible={showAdBreakOverlay}
          poster={poster}
          title={title}
          subtitle={subtitle}

          onDismiss={this.dismissAdBreak}
          simplified={this.mobile}

        />}

        {(loading || resolving) && (

          <div className="absolute inset-0 z-20 flex flex-col items-center justify-center gap-3 bg-surface/70 backdrop-blur-xl">

            <div className="h-9 w-9 animate-spin rounded-full border-2 border-foreground/20 border-t-foreground" />

            <p className="max-w-xs text-center text-sm text-foreground-muted">

              {streamResolving ? "Resolving stream…" : "Loading…"}

            </p>

          </div>

        )}

        {!compact && <PlayerActionFeedbackOverlay feedback={actionFeedback} />}

        {compact && (

          <button type="button" onClick={(event) => { event.stopPropagation(); onReturn?.(); }} className="absolute inset-0 z-40" aria-label="Return to full player" />

        )}

        {!compact && <div className={cn(

            "player-chrome-safe pointer-events-none absolute right-3 left-3 z-30 grid grid-cols-[1fr_auto_1fr] items-start gap-2 transition-opacity duration-300 sm:right-4 sm:left-4 sm:gap-4",
            portraitMobile ? "player-top-portrait" : "top-[max(1rem,calc(env(safe-area-inset-top,0px)+0.5rem))]",
            effectiveShowControls ? "opacity-100" : "opacity-0"

          )}

        >

          {onBack ? (

            <button onClick={(e) => {

                e.stopPropagation();

                onBack();

              }} className="pointer-events-auto flex shrink-0 items-center gap-2 justify-self-start px-1 py-1 text-sm text-foreground transition-colors hover:text-accent" >

              <ArrowLeft size={16} />

              Back

            </button>

          ) : (

            <div />

          )}

          <div className="pointer-events-none min-w-0 max-w-[min(100%,28rem)] justify-self-center px-2 text-center">

            <p className="truncate text-sm font-medium">

              {title} {episodeTitle ? <span className="text-foreground-muted">{subtitle}</span> : null}

            </p>

            {subtitle && (

              <p className="truncate text-xs text-foreground-muted">

                {episodeTitle || subtitle}

              </p>

            )}

          </div>

          <button
            type="button"
            onClick={(e) => {

              e.stopPropagation();
              navigate("/");

            }}
            className="pointer-events-auto justify-self-end px-1 py-1 text-sm font-semibold tracking-tight text-foreground transition-colors hover:text-accent"
            aria-label="Streamly Web"
          >

            {portrait ? (

              <img src="/Streamly.svg" alt="" className="h-7 w-7" />

            ) : (

              <>
                Streamly <span className="font-light text-foreground-muted">Web</span>
              </>

            )}

          </button>

        </div>}

        {!compact && !live && !menuOpen && (

          <EpisodeActions
            showSkipIntro={showSkipIntro}
            showUpNext={showUpNext}
            showUpNextMini={showUpNextMini}
            upNextCountdown={upNextCountdown}
            portraitMobile={portraitMobile}
            nextEpisode={nextEpisode}
            currentSeason={currentSeason}
            onSkipIntro={this.skipIntro}
            onNextEpisode={this.props.onNextEpisode}
            onCancel={() => this.setState({ showUpNext: false, showUpNextMini: false })}
          />

        )}

        {!compact && <div className={cn(

            "pointer-events-none absolute inset-x-0 z-20 transition-opacity duration-300",
            this.mobile ? "player-bottom-chrome" : "px-6 pb-4",
            portraitMobile && !showEpisodes ? "player-bottom-portrait" : "bottom-0",
            this.mobile ? (showEpisodes ? "pt-2" : "pt-3") : "pt-10",
            effectiveShowControls || showEpisodes ? "opacity-100" : "opacity-0"

        )}

        >

          {this.mobile && (

            <div className={showEpisodes ? "pointer-events-auto" : undefined}>

              {episodePicker}

            </div>

          )}

          {!live && (

            <div className="relative">

              {!this.mobile && episodePicker && (

                <div className={cn(

                    "absolute bottom-full left-1/2 z-30 mb-5 w-[96vw] -translate-x-1/2",
                    showEpisodes && "pointer-events-auto"

                  )}

                >

                  {episodePicker}

                </div>

              )}

            <div className={cn(

                "group relative mb-3 h-2 cursor-pointer rounded-full py-0.5",
                (effectiveShowControls || showEpisodes) && "pointer-events-auto"

              )} onClick={(e) => {

                e.stopPropagation();

                this.seek(this.scrubberRatioFromEvent(e) * this.durationMs);

              }} onMouseMove={this.onScrubberMove} onMouseLeave={this.onScrubberLeave} >

              <SeekTooltip

                ref={this.seekTooltipRef}

              />

              <div className="relative h-1 overflow-hidden rounded-full bg-white/20 transition-all duration-150 group-hover:h-1.5">

                <div className="absolute left-0 h-full rounded-full bg-white/30"

                  ref={this.bufferFillRef}
                  style={{ width: "0%" }}

                />

                <div className="relative h-full rounded-full bg-foreground transition-colors group-hover:bg-accent"

                  ref={this.progressFillRef}
                  style={{ width: "0%" }}

                />

              </div>

            </div>

            </div>

          )}

          <div className={cn(

              "relative flex items-center justify-between",
              (effectiveShowControls || showEpisodes) && "pointer-events-auto"

            )}
          >

            <div className="flex items-center gap-1.5">

              <ControlButton onClick={this.togglePlay}>

                {playing ? <Pause size={20} /> : <Play size={20} />}

              </ControlButton>

              {!live && !portraitMobile && (

                <ControlButton onClick={() => this.seekBy(-10_000)} aria-label="Seek back 10 seconds">

                  <Rewind size={20} />

                </ControlButton>

              )}

              {!live && !portraitMobile && (

                <ControlButton onClick={() => this.seekBy(10_000)} aria-label="Seek forward 10 seconds">

                  <FastForward size={20} />

                </ControlButton>

              )}

              {!this.mobile && (

                <VolumeControl

                  volume={volume}
                  muted={muted}

                  onVolumeChange={this.setVolume}
                  onToggleMute={this.toggleMute}

                />

              )}

              {(airplayAvailable || airplayActive) && (

                <ControlButton

                  onClick={this.openAirPlayPicker}
                  className={airplayActive ? "bg-white/15 text-accent" : undefined}
                  aria-label={airplayActive ? "AirPlay connected" : "AirPlay"}

                >

                  <Airplay size={20} />

                </ControlButton>

              )}

              {live && (

                behindLive ? (

                  <button type="button" onClick={(e) => {

                      e.stopPropagation();

                      this.jumpToLiveAndPlay();

                    }} className="pointer-events-auto flex items-center gap-1.5 rounded-md px-1.5 py-0.5 text-[10px] font-medium tracking-wider text-foreground-muted transition-colors hover:text-foreground" aria-label="Jump to live" >

                    <span className="h-1.5 w-1.5 rounded-full bg-foreground-muted" />

                    Jump To Live

                  </button>

                ) : (

                  <span className="flex items-center gap-1.5 text-[10px] font-medium tracking-wider text-red-400">

                    <span className="relative flex h-1.5 w-1.5">

                      <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-red-500 opacity-60" />

                      <span className="relative inline-flex h-1.5 w-1.5 rounded-full bg-red-500" />

                    </span>

                    LIVE

                  </span>

                )

              )}

            </div>

            {!live && (

              <span ref={this.timeLabelRef} className="pointer-events-auto absolute left-1/2 -translate-x-1/2 cursor-pointer select-none text-xs text-foreground-muted tabular-nums transition-colors hover:text-foreground" onClick={(e) => { e.stopPropagation(); this.toggleTimeDisplay(); }} >

                0:00 / 0:00

              </span>

            )}

            <div className="flex items-center gap-1">

              {episodesEnabled && (

                <ControlButton

                  onClick={this.toggleEpisodes}
                  className={showEpisodes ? "bg-white/15" : undefined}
                  aria-label="Browse episodes"

                >
                  <Clapperboard size={20} />

                </ControlButton>

              )}

              {(qualityEnabled || sourceEnabled) && (

                <PlayerOptionsMenu

                  open={showOptions}
                  compact={this.mobile}
                  qualities={qualities}
                  selectedHeight={selectedHeight}
                  preferredHeight={preferredHeight}
                  hdrHeights={hdrHeights}

                  subtitleTracks={live ? [] : this.allSubtitleTracks()}
                  activeSubtitleId={activeSubtitleId}
                  qualityEnabled={qualityEnabled}

                  sourceProviders={sourceEnabled ? sourceProviders : undefined}
                  selectedSourceKey={selectedSourceKey}
                  sourceSwitching={sourceSwitching}
                  onSourceChange={sourceEnabled ? onSourceChange : undefined}

                  onToggle={this.toggleOptions}
                  onClose={() => this.setState({ showOptions: false })}
                  onOutsideClose={this.closeOptionsFromOutside}
                  onQualityChange={(height) => {

                    const video = this.videoRef.current;

                    const positionMs = video ? video.currentTime * 1000 : 0;

                    this.setState({ showOptions: false });

                    onQualityChange?.(height, positionMs);

                  }}

                  onSubtitleChange={this.applySubtitleSelection}
                  onOpenSettings={live ? undefined : onOpenSettings}

                />

              )}

              {this.mobile ? (

                <ControlButton onClick={this.enterNativeFullscreen} aria-label="Full screen">

                  <Expand size={20} />

                </ControlButton>

              ) : (

                <ControlButton onClick={this.toggleFullscreen}>

                  {fullscreen ? <Minimize size={20} /> : <Maximize size={20} />}

                </ControlButton>

              )}

            </div>

          </div>

        </div>}

        </div>

        {compact && (

          <MiniPlayerControls
            live={live}
            playing={playing}
            progressRef={this.miniProgressFillRef}
            onDismiss={onDismiss}
            onReturn={onReturn}
            onTogglePlay={this.togglePlay}
            onSeek={(event) => this.seek(this.scrubberRatioFromEvent(event) * this.durationMs)}
          />

        )}

      </div>

    );

  }

}
