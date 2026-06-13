import { Component, createRef, type RefObject } from "react";
import type Hls from "hls.js";
import {
  ArrowLeft,
  Clapperboard,
  Maximize,
  Minimize,
  Pause,
  Play,
  SkipForward,
} from "lucide-react";
import { AmbienceLayer } from "@/components/player/AmbienceLayer";
import { EpisodePickerPanel } from "@/components/player/EpisodePickerPanel";
import { PauseOverlay } from "@/components/player/PauseOverlay";
import {
  PlayerActionFeedbackOverlay,
  type PlayerActionFeedback,
} from "@/components/player/PlayerActionFeedbackOverlay";
import { PlayerOptionsMenu } from "@/components/player/PlayerOptionsMenu";
import { SubtitleDisplay } from "@/components/player/SubtitleDisplay";
import { ControlButton, VolumeControl } from "@/components/player/VolumeControl";
import { hasIntroWindow, isInIntroWindow } from "@/lib/intro";
import { cn, formatDuration } from "@/lib/utils";
import type { Episode, IntroInfo, NextEpisode, Season, StreamQuality, SubtitleTrack } from "@/lib/types";

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
  onDurationReady?: (durationMs: number) => void;
  onPlaybackError?: (positionMs: number) => void;
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
  upNextCountdown: number;
  fullscreen: boolean;
  loading: boolean;
  seeking: boolean;
  activeSubtitleId: string | null;
  hlsSubtitleTracks: SubtitleTrack[];
  actionFeedback: PlayerActionFeedback | null;
}

export class VideoPlayer extends Component<VideoPlayerProps, VideoPlayerState> {
  private videoRef = createRef<HTMLVideoElement>();
  private containerRef = createRef<HTMLDivElement>();
  private progressFillRef = createRef<HTMLDivElement>();
  private bufferFillRef = createRef<HTMLDivElement>();
  private timeLabelRef = createRef<HTMLSpanElement>();
  private hls: Hls | null = null;
  private controlsTimer: ReturnType<typeof setTimeout> | null = null;
  private upNextTimer: ReturnType<typeof setInterval> | null = null;
  private lastProgressReport = 0;
  private lastUiUpdate = 0;
  private waitingTimer: ReturnType<typeof setTimeout> | null = null;
  private feedbackTimer: ReturnType<typeof setTimeout> | null = null;
  private feedbackId = 0;
  private durationMs = 0;
  private durationReported = false;
  private playbackErrorReported = false;
  private hlsRecoveryAttempts = 0;
  private static readonly MAX_HLS_RECOVERIES = 3;

  state: VideoPlayerState = {
    playing: false,
    muted: false,
    volume: 1,
    showControls: true,
    showOptions: false,
    showEpisodes: false,
    showSkipIntro: false,
    showUpNext: false,
    upNextCountdown: 10,
    fullscreen: false,
    loading: true,
    seeking: false,
    activeSubtitleId: null,
    hlsSubtitleTracks: [],
    actionFeedback: null,
  };

  componentDidMount() {
    this.attachSource();
    this.bindVideoEvents();
    document.addEventListener("keydown", this.onKeyDown);
    document.addEventListener("fullscreenchange", this.onFullscreenChange);
    this.syncSubtitlePreference();
  }

  componentDidUpdate(prev: VideoPlayerProps) {
    if (prev.src !== this.props.src || prev.isHls !== this.props.isHls) {
      this.unbindVideoEvents();
      this.destroyHls();
      this.attachSource();
      this.bindVideoEvents();
    }
    if (
      prev.subtitleTracks !== this.props.subtitleTracks ||
      prev.subtitlesEnabled !== this.props.subtitlesEnabled
    ) {
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
    this.setState({ loading: false, playing: false });
    if (this.playbackErrorReported || !this.props.onPlaybackError) return;
    this.playbackErrorReported = true;
    const video = this.videoRef.current;
    const positionMs = video ? video.currentTime * 1000 : 0;
    this.props.onPlaybackError(positionMs);
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
    if (this.upNextTimer) clearInterval(this.upNextTimer);
    if (this.waitingTimer) clearTimeout(this.waitingTimer);
    if (this.feedbackTimer) clearTimeout(this.feedbackTimer);
    this.waitingTimer = null;
    this.feedbackTimer = null;
  };

  destroyHls = () => {
    if (this.hls) {
      this.hls.destroy();
      this.hls = null;
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
    if (!video || !src.trim()) {
      this.setState({ loading: false, playing: false });
      return;
    }

    this.durationReported = false;
    this.playbackErrorReported = false;
    this.hlsRecoveryAttempts = 0;
    this.setState({
      loading: true,
      showUpNext: false,
      showSkipIntro: false,
      seeking: false,
      hlsSubtitleTracks: [],
    });
    video.removeEventListener("error", this.onVideoError);
    video.addEventListener("error", this.onVideoError);

    const onReady = () => {
      if (!this.props.live && startPositionMs && startPositionMs > 0) {
        video.currentTime = startPositionMs / 1000;
      }
      video.volume = this.state.volume;
      video.muted = this.state.muted;
      video.play().catch(() => undefined);
      this.setState({ loading: false, playing: true });
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
      this.hls = new HlsConstructor({
        enableWorker: true,
        lowLatencyMode: !!lowLatency,
        maxBufferLength: lowLatency ? 12 : 45,
        maxMaxBufferLength: lowLatency ? 24 : 90,
        backBufferLength: 30,
        xhrSetup: (xhr) => {
          xhr.withCredentials = true;
        },
      });
      this.hls.loadSource(src);
      this.hls.attachMedia(video);
      this.hls.on(HlsConstructor.Events.MANIFEST_PARSED, onReady);
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
            return;
          }
          this.hls.recoverMediaError();
          return;
        }
        this.onVideoError();
      });
    } else {
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
    const { intro, nextEpisode, autoPlayNext } = this.props;
    if (!autoPlayNext || !nextEpisode || this.state.showUpNext) return;

    const creditsStart = intro?.creditsStartMs;
    const threshold = creditsStart && creditsStart > 0 ? creditsStart : durationMs - 30000;
    if (durationMs > 0 && currentMs >= threshold) {
      this.triggerUpNext();
    }
  };

  triggerUpNext = () => {
    this.setState({ showUpNext: true, upNextCountdown: 10 });
    if (this.upNextTimer) clearInterval(this.upNextTimer);
    this.upNextTimer = setInterval(() => {
      this.setState((s) => {
        if (s.upNextCountdown <= 1) {
          if (this.upNextTimer) clearInterval(this.upNextTimer);
          this.props.onNextEpisode?.();
          return { ...s, showUpNext: false };
        }
        return { ...s, upNextCountdown: s.upNextCountdown - 1 };
      });
    }, 1000);
  };

  togglePlay = () => {
    const video = this.videoRef.current;
    if (!video) return;
    if (video.paused) {
      video
        .play()
        .then(() => this.setState({ playing: true }))
        .catch(() => this.setState({ playing: false }));
    } else {
      video.pause();
      this.setState({ playing: false });
    }
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

  showActionFeedback = (
    feedback: Omit<PlayerActionFeedback, "id">,
  ) => {
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
        this.seekBy(-5000);
        this.showControlsTemporarily();
        break;
      case "ArrowRight":
        event.preventDefault();
        this.seekBy(5000);
        this.showControlsTemporarily();
        break;
      case "ArrowUp":
      case "ArrowDown": {
        event.preventDefault();
        const delta = event.key === "ArrowUp" ? 0.05 : -0.05;
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
      case " ":
      case "k":
      case "K":
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
    }, 3000);
  };

  toggleOptions = () => {
    this.setState((s) => ({
      showOptions: !s.showOptions,
      showEpisodes: s.showOptions ? s.showEpisodes : false,
    }));
  };

  toggleEpisodes = () => {
    this.setState((s) => ({
      showEpisodes: !s.showEpisodes,
      showOptions: s.showEpisodes ? s.showOptions : false,
      showControls: true,
    }));
  };

  render() {
    const {
      title,
      subtitle,
      episodeTitle,
      description,
      poster,
      qualities = [],
      selectedHeight = 1080,
      nextEpisode,
      onBack,
      ambienceEnabled,
      live,
      onQualityChange,
      seasons,
      episodes,
      currentSeason,
      currentEpisode,
      menuSeason,
      episodesLoading,
      onSeasonChange,
      onEpisodeSelect,
    } = this.props;
    const {
      playing,
      muted,
      volume,
      showControls,
      showOptions,
      showEpisodes,
      showSkipIntro,
      showUpNext,
      upNextCountdown,
      fullscreen,
      loading,
      seeking,
      activeSubtitleId,
      actionFeedback,
    } = this.state;

    const showPauseOverlay = !playing && !loading && !seeking && !showEpisodes;
    const qualityEnabled = !live && qualities.length > 0 && !!onQualityChange;
    const episodesEnabled = !live && !!onEpisodeSelect && (seasons?.length ?? 0) > 0;
    const episodeStill = !!episodeTitle;

    return (
      <div
        ref={this.containerRef}
        className="relative flex h-full w-full flex-col overflow-hidden bg-black"
        onMouseMove={this.showControlsTemporarily}
        onClick={this.showControlsTemporarily}
      >
        <AmbienceLayer
          videoRef={this.videoRef as RefObject<HTMLVideoElement>}
          enabled={!!ambienceEnabled}
        />

        <video
          ref={this.videoRef}
          className="relative z-10 h-full w-full object-contain [transform:translateZ(0)] [will-change:transform]"
          playsInline
          crossOrigin={ambienceEnabled ? "anonymous" : undefined}
          onEnded={this.onEnded}
          onClick={(e) => {
            e.stopPropagation();
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
          <div className="absolute inset-0 z-20 flex items-center justify-center bg-black/40">
            <div className="h-8 w-8 animate-spin rounded-full border-2 border-foreground/20 border-t-foreground" />
          </div>
        )}

        <PlayerActionFeedbackOverlay feedback={actionFeedback} />

        <div
          className={cn(
            "absolute top-4 right-4 left-4 z-30 flex items-start justify-between gap-4 transition-all duration-300",
            showControls ? "opacity-100" : "pointer-events-none opacity-0",
          )}
        >
          {onBack ? (
            <button
              onClick={(e) => {
                e.stopPropagation();
                onBack();
              }}
              className="flex shrink-0 items-center gap-2 rounded-md bg-black/50 px-3 py-1.5 text-xs backdrop-blur-sm hover:bg-black/70"
            >
              <ArrowLeft size={14} />
              Back
            </button>
          ) : (
            <div />
          )}

          <div className="max-w-[55%] min-w-0 rounded-md bg-black/50 px-3 py-1.5 text-right backdrop-blur-sm">
            <p className="truncate text-sm font-medium">{title}</p>
            {subtitle && (
              <p className="truncate text-xs text-foreground-muted">{subtitle}</p>
            )}
          </div>
        </div>

        {!live && showSkipIntro && (
          <button
            onClick={this.skipIntro}
            className="absolute right-6 bottom-20 z-20 flex animate-fade-in items-center gap-2 rounded-md border border-border bg-surface-raised/90 px-4 py-2 text-sm font-medium backdrop-blur-sm transition-colors hover:bg-surface-overlay"
          >
            <SkipForward size={14} />
            Skip Intro
          </button>
        )}

        {!live && showUpNext && nextEpisode && (
          <div className="absolute right-6 bottom-28 z-20 w-72 animate-fade-in rounded-lg border border-border bg-surface-raised/95 p-4 backdrop-blur-md">
              <p className="text-[11px] tracking-wide text-foreground-faint uppercase">Up Next</p>
              <p className="mt-1 text-sm font-medium">{nextEpisode.title}</p>
              <p className="text-xs text-foreground-muted">
                S{String(nextEpisode.season).padStart(2, "0")}E
                {String(nextEpisode.episode).padStart(2, "0")}
              </p>
              <div className="mt-3 flex gap-2">
                <button
                  onClick={() => this.setState({ showUpNext: false })}
                  className="flex-1 rounded-md border border-border px-3 py-1.5 text-xs transition-colors hover:bg-surface-overlay"
                >
                  Cancel
                </button>
                <button
                  onClick={() => this.props.onNextEpisode?.()}
                  className="flex-1 rounded-md bg-foreground px-3 py-1.5 text-xs text-surface transition-colors hover:bg-accent"
                >
                  Play ({upNextCountdown})
                </button>
              </div>
          </div>
        )}

        <div
          className={cn(
            "absolute inset-x-0 bottom-0 z-20 px-4 pb-4 transition-opacity duration-300 sm:px-6",
            showEpisodes
              ? "bg-gradient-to-t from-black/90 from-35% via-black/75 to-transparent pt-2"
              : "bg-gradient-to-t from-black/90 via-black/50 to-transparent pt-10",
            showControls || showEpisodes ? "opacity-100" : "pointer-events-none opacity-0",
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
            <div
              className="group mb-3 h-2 cursor-pointer rounded-full py-0.5"
              onClick={(e) => {
                e.stopPropagation();
                const rect = e.currentTarget.getBoundingClientRect();
                const pct = (e.clientX - rect.left) / rect.width;
                this.seek(pct * this.durationMs);
              }}
            >
              <div className="relative h-1 overflow-hidden rounded-full bg-white/20 transition-all duration-150 group-hover:h-1.5">
                <div
                  ref={this.bufferFillRef}
                  className="absolute left-0 h-full rounded-full bg-white/30"
                  style={{ width: "0%" }}
                />
                <div
                  ref={this.progressFillRef}
                  className="relative h-full rounded-full bg-foreground transition-colors group-hover:bg-accent"
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
                <span
                  ref={this.timeLabelRef}
                  className="text-xs text-foreground-muted tabular-nums"
                >
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
                subtitleTracks={this.allSubtitleTracks()}
                activeSubtitleId={activeSubtitleId}
                qualityEnabled={qualityEnabled}
                onToggle={this.toggleOptions}
                onClose={() => this.setState({ showOptions: false })}
                onQualityChange={(height) => {
                  const video = this.videoRef.current;
                  const positionMs = video ? video.currentTime * 1000 : 0;
                  this.setState({ showOptions: false });
                  onQualityChange?.(height, positionMs);
                }}
                onSubtitleChange={this.applySubtitleSelection}
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
