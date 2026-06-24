import type HLS from "hls.js";
import { Component, createRef, type PointerEvent as ReactPointerEvent, type RefObject } from "react";
import { ArrowLeft, Clapperboard, Maximize, Minimize, Pause, Play, SkipForward } from "lucide-react";

import { AmbienceLayer } from "@/components/player/AmbienceLayer";
import { EpisodePickerPanel } from "@/components/player/EpisodePickerPanel";
import { PauseOverlay } from "@/components/player/PauseOverlay";

import { PlayerActionFeedbackOverlay, type PlayerActionFeedback, } from "@/components/player/PlayerActions";
import { PlayerOptionsMenu } from "@/components/player/PlayerOptionsMenu";
import { SubtitleDisplay } from "@/components/player/SubtitleDisplay";
import { ControlButton, VolumeControl } from "@/components/player/VolumeControl";

import { store } from "@/lib/store";
import { hasIntroWindow, isInIntroWindow } from "@/lib/intro";
import { isProxiedStream, isWebPlayableUrl } from "@/lib/streamClient";
import { cn, formatDuration } from "@/lib/utils";
import type { Episode, IntroInfo, NextEpisode, Season, StreamQuality, SubtitleTrack, } from "@/lib/types";

type HlsLevelLike = {

  attrs?: Record<string, string | undefined>;
  height?: number;
  videoCodec?: string;
  codecSet?: string;

};

const videoCodecFromLevel = (level: HlsLevelLike): string => {

  const explicit = level.videoCodec?.trim();
  if (explicit) return explicit;

  const codecs = (level.attrs?.["CODECS"] ?? level.codecSet ?? "").split(",");
  const video = codecs.find((codec) => /^(avc1|avc3|hvc1|hev1|dvh1|dvhe|av01|vp09)\./i.test(codec.trim()));

  return video?.trim() ?? "";

};

const isHdrLevel = (level: HlsLevelLike): boolean => {

  const videoRange = level.attrs?.["VIDEO-RANGE"];
  const codec = videoCodecFromLevel(level);

  return (
    videoRange === "PQ" ||
    videoRange === "HLG" ||
    /hvc1\.2\./i.test(codec) ||
    /hev1\.2\./i.test(codec) ||
    /dvh1\.|dvhe\./i.test(codec)
  );

};

const isHlsLevelSupported = (level: HlsLevelLike): boolean => {

  const codec = videoCodecFromLevel(level);

  if (!codec) return true;

  const mime = `video/mp4; codecs="${codec}"`;

  if (window.MediaSource?.isTypeSupported(mime)) return true;

  const video = document.createElement("video");

  return video.canPlayType(mime) !== "";

};

const bestSupportedHlsLevel = (levels: HlsLevelLike[], selectedHeight: number): { index: number; isExact: boolean } | null => {

  const supported = levels
    .map((level, index) => ({ level, index }))
    .filter(({ level }) => isHlsLevelSupported(level));

  if (supported.length === 0) return null;

  const capped = selectedHeight > 0
    ? supported.filter(({ level }) => (level.height ?? 0) > 0 && (level.height ?? 0) <= selectedHeight)
    : supported;

  const target = (capped.length > 0 ? capped : supported)
    .reduce((best, item) => ((item.level.height ?? 0) > (best.level.height ?? 0) ? item : best));

  return {

    index: target.index,
    isExact: selectedHeight <= 0 || (target.level.height ?? 0) === selectedHeight,

  };

};

interface VideoPlayerProps {

  src: string;
  isHls: boolean;
  live?: boolean;
  lowLatency?: boolean;

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
  onEnded?: () => void;
  onNextEpisode?: () => void;
  onQualityChange?: (height: number, positionMs: number) => void;
  onOpenSettings?: () => void;
  onDurationReady?: (durationMs: number) => void;
  onPlaybackError?: (positionMs: number) => void;
  onFatalError?: () => void;

  seasons?: Season[];
  episodes?: Episode[];
  currentSeason?: number;
  currentEpisode?: number;
  menuSeason?: number;
  episodesLoading?: boolean;
  onSeasonChange?: (season: number) => void;
  onEpisodeSelect?: (season: number, episode: number) => void;

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

  // Heights (in px) for which HDR content has been detected. Persists across  quality switches.
  hdrHeights: Set<number>;

}

export class VideoPlayer extends Component<VideoPlayerProps, VideoPlayerState> {

  private videoRef = createRef<HTMLVideoElement>();
  private containerRef = createRef<HTMLDivElement>();
  private progressFillRef = createRef<HTMLDivElement>();
  private bufferFillRef = createRef<HTMLDivElement>();
  private timeLabelRef = createRef<HTMLSpanElement>();

  private hls: HLS | null = null;

  private controlsTimer: ReturnType<typeof setTimeout> | null = null;
  private waitingTimer: ReturnType<typeof setTimeout> | null = null;
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

  private static readonly MAX_HLS_RECOVERIES = 1;
  private static readonly SOURCE_READY_TIMEOUT_MS = 8_000;
  private static readonly HOLD_PAUSE_DELAY_MS = 220;
  private static readonly UP_NEXT_VISIBLE_LEAD_MS = 150_000;
  private static readonly UP_NEXT_COUNTDOWN_LEAD_MS = 60_000;

  state: VideoPlayerState = {

    playing: false,
    muted: false,
    volume: 1,

    showControls: true,
    showOptions: false,
    showEpisodes: false,
    showSkipIntro: false,
    showUpNext: false,
    showUpNextMini: false,
    upNextCountdown: 0,

    fullscreen: false,
    loading: true,
    seeking: false,
    holdPauseActive: false,

    activeSubtitleId: null,
    hlsSubtitleTracks: [],

    actionFeedback: null,

    hdrHeights: new Set(),

  };

  componentDidMount() {

    this.attachSource();

    this.bindVideoEvents();

    document.addEventListener("keydown", this.onKeyDown);

    document.addEventListener("fullscreenchange", this.onFullscreenChange);

    this.syncSubtitlePreference();

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

    if (prev.intro !== this.props.intro) {

      const video = this.videoRef.current;
      if (video) this.checkSkipIntro(video.currentTime * 1000);

    }

  }

  componentWillUnmount() {

    this.unbindVideoEvents();

    this.destroyHls();

    this.clearTimers();

    document.removeEventListener("keydown", this.onKeyDown);
    document.removeEventListener("fullscreenchange", this.onFullscreenChange);

  }

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

  };

  onVideoError = () => {

    this.clearSourceReadyTimer();

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

    if (this.props.live) return;

    const video = this.videoRef.current;

    if (!video || this.state.seeking) return;

    const currentMs = video.currentTime * 1000;

    const durationMs = (video.duration || 0) * 1000;

    this.durationMs = durationMs;

    const now = performance.now();

    if (now - this.lastUiUpdate > 250) {

      this.lastUiUpdate = now;

      this.updateProgressUI(currentMs, durationMs);
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

    if (this.progressFillRef.current) {

      const pct = durationMs > 0 ? (currentMs / durationMs) * 100 : 0;
      this.progressFillRef.current.style.width = `${pct}%`;

    }

    if (this.timeLabelRef.current) {

      this.timeLabelRef.current.textContent = `${formatDuration(currentMs)} / ${formatDuration(durationMs)}`;

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

    this.setState({ seeking: false });

  };

  onWaiting = () => {

    if (this.waitingTimer || this.state.loading) return;

    this.waitingTimer = setTimeout(() => {

      this.waitingTimer = null;
      this.setState({ loading: true });

    }, 600);

  };

  clearBuffering = () => {

    if (this.waitingTimer) {

      clearTimeout(this.waitingTimer);
      this.waitingTimer = null;

    }

    if (this.state.loading && !this.state.seeking) {

      this.setState({ loading: false });

    }

    this.clearSourceReadyTimer();

  };

  onCanPlay = () => {

    this.clearBuffering();
    this.setState({ seeking: false });

  };

  onPlaying = () => {

    this.clearBuffering();
    this.setState({ playing: true });

  };

  onPause = () => {

    this.setState({ playing: false });

  };

  onVolumeChange = () => {

    const video = this.videoRef.current;

    if (!video) return;

    this.setState({ volume: video.volume, muted: video.muted });

  };

  clearTimers = () => {

    if (this.controlsTimer) clearTimeout(this.controlsTimer);
    if (this.waitingTimer) clearTimeout(this.waitingTimer);
    if (this.feedbackTimer) clearTimeout(this.feedbackTimer);
    if (this.audioProbeTimer) clearTimeout(this.audioProbeTimer);
    if (this.sourceReadyTimer) clearTimeout(this.sourceReadyTimer);
    if (this.holdPauseTimer) clearTimeout(this.holdPauseTimer);

    this.waitingTimer = null;
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
      this.hls = null

    }

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

    this.setState({

      loading: true,

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

    const onReady = () => {

      if (!this.props.live && startPositionMs && startPositionMs > 0) {

        video.currentTime = startPositionMs / 1000;

      }

      video.volume = this.state.volume;
      video.muted = this.state.muted;

      // Don't mark loading:false here — onCanPlay/onPlaying handle that so the spinner stays visible through any initial seek without oscillating.
      video.play().catch(() => this.setState({ loading: false, playing: false }));

      this.onLoadedMetadata();
      this.syncHlsSubtitles();

    };

    if (isHls) {

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

      const proxied = isProxiedStream(src);

      this.hls = new HlsConstructor({

        enableWorker: true,

        lowLatencyMode: !!lowLatency,

        maxBufferLength: lowLatency ? 12 : 45,
        maxMaxBufferLength: lowLatency ? 24 : 90,

        backBufferLength: 30,

        xhrSetup: proxied ? (xhr) => { xhr.withCredentials = true; } : undefined,

      });

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

          if (++this.hlsRecoveryAttempts > VideoPlayer.MAX_HLS_RECOVERIES) {

            this.onVideoError();
            return;

          }

          this.hls.startLoad();

          return;
        }

        if (data.type === HlsConstructor.ErrorTypes.MEDIA_ERROR) {

          if (++this.hlsRecoveryAttempts > VideoPlayer.MAX_HLS_RECOVERIES) {

            this.onVideoError();
            return

          }

          this.hls.recoverMediaError();

          return;

        }

        this.onVideoError();

      });

    } else {

      if (isProxiedStream(src)) {

        video.crossOrigin = this.props.ambienceEnabled ? "anonymous" : "use-credentials";

      } else if (this.props.ambienceEnabled) {

        video.crossOrigin = "anonymous";

      } else {

        video.removeAttribute("crossorigin");

      }

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

  togglePlay = () => {

    const video = this.videoRef.current;

    if (!video) return;

    if (video.paused) {

      video.play().then(() => this.setState({ playing: true })).catch(() => this.setState({ playing: false }));

    } else {

      video.pause();
      this.setState({ playing: false });

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

    const video = this.videoRef.current;
    if (!video) return;

    const clamped = Math.max(0, Math.min(volume, 1));

    video.volume = clamped;
    video.muted = clamped === 0;

    this.setState({ volume: clamped, muted: video.muted });

  };

  toggleMute = () => {

    const video = this.videoRef.current;

    if (!video) return;

    if (video.muted || video.volume === 0) {

      const nextVolume = video.volume > 0 ? video.volume : 0.8;

      video.muted = false;
      video.volume = nextVolume;

    } else {

      video.muted = true;

    }

    this.setState({ muted: video.muted, volume: video.volume });

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

  showControlsTemporarily = () => {

    this.setState({ showControls: true });

    if (this.controlsTimer) clearTimeout(this.controlsTimer);

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

    const { title, subtitle, episodeTitle, description, poster, qualities = [], selectedHeight = 1080, preferredHeight, nextEpisode, onBack, ambienceEnabled, live, onQualityChange, onOpenSettings, seasons, episodes, currentSeason, currentEpisode, menuSeason, episodesLoading, onSeasonChange, onEpisodeSelect, } = this.props;
    const { playing, muted, volume, showControls, showOptions, showEpisodes, showSkipIntro, showUpNext, showUpNextMini, upNextCountdown, fullscreen, loading, seeking, holdPauseActive, activeSubtitleId, actionFeedback, hdrHeights, } = this.state;

    const showPauseOverlay = !playing && !loading && !seeking && !holdPauseActive && !showEpisodes && store.settings?.disablePauseOverlay !== true;

    const qualityEnabled = !live && qualities.length > 0 && !!onQualityChange;
    const episodesEnabled = !live && !!onEpisodeSelect && !!onSeasonChange;

    const episodeStill = !!episodeTitle;

    return (

      <div className="relative flex h-full w-full flex-col bg-black"

        ref={this.containerRef}
        onMouseMove={this.showControlsTemporarily}
        onClick={this.showControlsTemporarily}

      >
        <AmbienceLayer

          videoRef={this.videoRef as RefObject<HTMLVideoElement>}
          enabled={!!ambienceEnabled}

        />

        <video className="relative z-10 h-full w-full object-contain"

          ref={this.videoRef}
          playsInline
          crossOrigin={ambienceEnabled ? "anonymous" : undefined}

          onEnded={this.onEnded}
          onPointerDown={this.beginHoldPause}
          onPointerUp={this.endHoldPause}
          onPointerCancel={this.cancelHoldPause}
          onClick={(e) => {

            e.stopPropagation();

            if (this.suppressNextVideoClick) {

              this.suppressNextVideoClick = false;
              return;

            }

            this.togglePlay();

          }}

        />

        <SubtitleDisplay

          videoRef={this.videoRef}
          track={showPauseOverlay || showEpisodes ? null : this.activeSubtitleTrack()}

        />

        <PauseOverlay

          visible={showPauseOverlay}
          poster={poster}
          still={episodeStill}

          title={title}
          subtitle={subtitle}
          episodeTitle={episodeTitle}
          description={description}

          onResume={this.togglePlay}
          pausedAt={this.videoRef.current ? this.videoRef.current.currentTime : 0}
          totalDuration={this.videoRef.current ? this.videoRef.current.duration : 0}

        />

        {loading && (

          <div className="absolute inset-0 z-20 flex items-center justify-center bg-surface/60 backdrop-blur-md">

            <div className="h-8 w-8 animate-spin rounded-full border-2 border-foreground/20 border-t-foreground" />

          </div>

        )}

        <PlayerActionFeedbackOverlay feedback={actionFeedback} />

        <div className={cn(

            "absolute top-4 right-4 left-4 z-30 flex items-start justify-between gap-4 transition-opacity duration-300",
            showControls ? "opacity-100" : "pointer-events-none opacity-0"

          )}

        >

          {onBack ? (

            <button onClick={(e) => {

                e.stopPropagation();

                onBack();

              }} className="flex shrink-0 items-center gap-2 rounded-md border border-border-subtle bg-surface/80 px-3 py-1.5 text-xs backdrop-blur-md transition-colors hover:bg-surface-overlay" >

              <ArrowLeft size={14} />

              Back

            </button>

          ) : (

            <div />

          )}

          <div className="max-w-[55%] min-w-0 rounded-md border border-border-subtle bg-surface/80 px-3 py-1.5 text-right backdrop-blur-md">

            <p className="truncate text-sm font-medium">

              {title}

            </p>

            {subtitle && (

              <p className="truncate text-xs text-foreground-muted">

                {subtitle}

              </p>

            )}

          </div>

        </div>

        {!live && showSkipIntro && (

          <button onClick={(e) => {

              e.stopPropagation();

              this.skipIntro();

            }} className="pointer-events-auto absolute right-6 bottom-20 z-40 flex animate-fade-in items-center gap-2 rounded-md border border-border-subtle bg-surface/80 px-4 py-2 text-sm font-medium backdrop-blur-md transition-colors hover:bg-surface-overlay" >

            <SkipForward size={14} />

            Skip Intro

          </button>

        )}

        {!live && showUpNextMini && !showUpNext && nextEpisode && (

          <button onClick={(e) => {

              e.stopPropagation();

              this.props.onNextEpisode?.();

            }} className="pointer-events-auto absolute right-6 bottom-28 z-40 flex animate-fade-in items-center gap-2 rounded-md border border-border-subtle bg-surface/80 px-4 py-2 text-sm font-medium backdrop-blur-md transition-colors hover:bg-surface-overlay" >

            <SkipForward size={14} />

            {nextEpisode.season !== currentSeason ? "Next Season" : "Next Episode"}

          </button>

        )}

        {!live && showUpNext && nextEpisode && (

          <div className="pointer-events-auto absolute right-6 bottom-28 z-40 w-72 animate-fade-in rounded-lg border border-border-subtle bg-surface/80 p-4 backdrop-blur-md">

            <p className="text-[11px] tracking-wide text-foreground-faint uppercase">

              Up Next

            </p>

            <p className="mt-1 text-sm font-medium">

              {nextEpisode.title}

            </p>

            <p className="text-xs text-foreground-muted">

              S{String(nextEpisode.season).padStart(2, "0")}E
              {String(nextEpisode.episode).padStart(2, "0")}

            </p>

            <div className="mt-3 flex gap-2">

              <button onClick={() => this.setState({ showUpNext: false, showUpNextMini: false })} className="flex-1 rounded-md border border-border px-3 py-1.5 text-xs transition-colors hover:bg-surface-overlay" >

                Cancel

              </button>

              <button onClick={() => this.props.onNextEpisode?.()} className="flex-1 rounded-md bg-foreground px-3 py-1.5 text-xs text-surface transition-colors hover:bg-accent" >

                {upNextCountdown > 0 ? `Play (${upNextCountdown})` : "Play"}

              </button>

            </div>

          </div>

        )}

        <div className={cn(

            "absolute inset-x-0 bottom-0 z-20 px-4 pb-4 transition-opacity duration-300 sm:px-6",
            showEpisodes ? "pt-2" : "pt-10",
            showControls || showEpisodes ? "opacity-100" : "pointer-events-none opacity-0"

        )}

        >

          {episodesEnabled && onSeasonChange && onEpisodeSelect && (

            <EpisodePickerPanel

              open={showEpisodes}
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

          )}

          {!live && (

            <div className="group mb-3 h-2 cursor-pointer rounded-full py-0.5" onClick={(e) => {

                e.stopPropagation();

                const rect = e.currentTarget.getBoundingClientRect();

                const pct = (e.clientX - rect.left) / rect.width;

                this.seek(pct * this.durationMs);

              }} >

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

          )}

          <div className="flex items-center justify-between">

            <div className="flex items-center gap-3">

              <ControlButton onClick={this.togglePlay}>

                {playing ? <Pause size={18} /> : <Play size={18} />}

              </ControlButton>

              <VolumeControl

                volume={volume}
                muted={muted}

                onVolumeChange={this.setVolume}
                onToggleMute={this.toggleMute}

              />

              {live ? (

                <span className="flex items-center gap-1.5 text-[10px] font-medium tracking-wider text-red-400">

                  <span className="relative flex h-1.5 w-1.5">

                    <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-red-500 opacity-60" />

                    <span className="relative inline-flex h-1.5 w-1.5 rounded-full bg-red-500" />

                  </span>

                  LIVE

                </span>

              ) : (

                <span ref={this.timeLabelRef} className="text-xs text-foreground-muted tabular-nums" >

                  0:00 / 0:00

                </span>

              )}

            </div>

            <div className="flex items-center gap-1">

              {episodesEnabled && (

                <ControlButton

                  onClick={this.toggleEpisodes}
                  className={showEpisodes ? "bg-white/10" : undefined}
                  aria-label="Browse episodes"

                >
                  <Clapperboard size={18} />

                </ControlButton>

              )}

              <PlayerOptionsMenu

                open={showOptions}
                qualities={qualities}
                selectedHeight={selectedHeight}
                preferredHeight={preferredHeight}
                hdrHeights={hdrHeights}

                subtitleTracks={this.allSubtitleTracks()}
                activeSubtitleId={activeSubtitleId}
                qualityEnabled={qualityEnabled}

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
                onOpenSettings={onOpenSettings}

              />

              <ControlButton onClick={this.toggleFullscreen}>

                {fullscreen ? <Minimize size={18} /> : <Maximize size={18} />}

              </ControlButton>

            </div>

          </div>

        </div>

      </div>

    );

  }

}
