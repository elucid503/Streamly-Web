import { Component } from "react";
import { api, ApiError } from "@/api/client";
import { history, navigate, saveReturnPath, type NavigateFn } from "@/lib/navigation";
import { parseWatchPath } from "@/lib/watchRoute";
import { VideoPlayer } from "@/components/player/VideoPlayer";
import { Button } from "@/components/ui/Button";
import { store } from "@/lib/store";
import {
  dedupeQualitiesByHeight,
  fetchStreamWithFallback,
  initialQualityAttempts,
  nextLowerQualityHeight,
} from "@/lib/stream";
import type {
  Episode,
  IntroInfo,
  NextEpisode,
  Season,
  StreamInfo,
  StreamQuality,
  SubtitleTrack,
} from "@/lib/types";

interface WatchPageProps {
  navigate: NavigateFn;
  watchPath: string;
}

interface WatchPageState {
  streamUrl: string;
  isHls: boolean;
  qualities: StreamQuality[];
  selectedHeight: number;
  subtitleTracks: SubtitleTrack[];
  title: string;
  subtitle: string;
  episodeTitle: string;
  description: string;
  intro: IntroInfo | null;
  nextEpisode: NextEpisode | null;
  startPositionMs: number;
  loading: boolean;
  error: string;
  kind: "movie" | "show" | "live";
  mediaId: number;
  season: number;
  episode: number;
  channelId: string;
  poster: string;
  historyPoster: string;
  seasons: Season[];
  menuEpisodes: Episode[];
  menuSeason: number;
  menuEpisodesLoading: boolean;
  episodeCache: Record<number, Episode[]>;
  streamGeneration: number;
  ready: boolean;
}

const EMPTY_STATE: Omit<WatchPageState, "loading" | "error" | "ready"> = {
  streamUrl: "",
  isHls: false,
  qualities: [],
  selectedHeight: 1080,
  subtitleTracks: [],
  title: "",
  subtitle: "",
  episodeTitle: "",
  description: "",
  intro: null,
  nextEpisode: null,
  startPositionMs: 0,
  kind: "movie",
  mediaId: 0,
  season: 0,
  episode: 0,
  channelId: "",
  poster: "",
  historyPoster: "",
  seasons: [],
  menuEpisodes: [],
  menuSeason: 1,
  menuEpisodesLoading: false,
  episodeCache: {},
  streamGeneration: 0,
};

export class WatchPage extends Component<WatchPageProps, WatchPageState> {
  private progressDebounce: ReturnType<typeof setTimeout> | null = null;
  private unsubscribe = () => {};
  private loadGen = 0;
  private failedQualityHeights = new Set<number>();

  state: WatchPageState = {
    ...EMPTY_STATE,
    loading: true,
    error: "",
    ready: false,
  };

  componentDidMount() {
    this.unsubscribe = store.subscribe(this.onStoreChange);
    this.tryLoad();
  }

  componentDidUpdate(prev: WatchPageProps) {
    if (prev.watchPath !== this.props.watchPath) {
      this.tryLoad();
    }
  }

  componentWillUnmount() {
    this.unsubscribe();
    if (this.progressDebounce) clearTimeout(this.progressDebounce);
  }

  onStoreChange = () => {
    this.forceUpdate();
  };

  tryLoad = () => {
    if (!store.isAuthenticated) {
      saveReturnPath(history.location.pathname);
      navigate("/auth");
      return;
    }
    this.load();
  };

  handleAuthFailure = () => {
    saveReturnPath(history.location.pathname);
    store.setUser(null);
    store.setSettings(null);
    navigate("/auth");
  };

  load = async () => {
    const gen = ++this.loadGen;
    const route = parseWatchPath(this.props.watchPath);
    this.failedQualityHeights.clear();

    this.setState({
      ...EMPTY_STATE,
      loading: true,
      error: "",
      ready: false,
    });

    if (!route.valid) {
      this.setState({ error: route.reason, loading: false });
      return;
    }

    try {
      if (route.kind === "movie") {
        await this.loadMovie(route.id, gen);
      } else if (route.kind === "show") {
        await this.loadEpisode(route.showId, route.season, route.episode, gen);
      } else {
        await this.loadLive(route.channelId, gen);
      }
    } catch (err) {
      if (gen !== this.loadGen) return;
      if (err instanceof ApiError && (err.status === 401 || err.status === 403)) {
        this.handleAuthFailure();
        return;
      }
      this.setState({
        error: err instanceof Error ? err.message : "failed to load stream",
        loading: false,
        ready: false,
      });
    }
  };

  requestStream = async (height: number): Promise<StreamInfo> => {
    const { kind, mediaId, season, episode } = this.state;
    if (kind === "movie") return api.movieStream(mediaId, height);
    if (kind === "show") return api.episodeStream(mediaId, season, episode, height);
    throw new Error("no stream available");
  };

  applyStream = (stream: StreamInfo, requestedHeight: number, positionMs: number) => {
    this.setState((prev) => ({
      ...prev,
      streamUrl: stream.proxyUrl,
      isHls: stream.isHls,
      qualities: dedupeQualitiesByHeight(stream.qualities ?? prev.qualities),
      selectedHeight: stream.selectedHeight ?? requestedHeight,
      startPositionMs: Math.floor(positionMs),
      streamGeneration: prev.streamGeneration + 1,
      error: "",
    }));
  };

  switchStream = async (height: number, positionMs: number) => {
    const stream = await this.requestStream(height);
    if (!stream?.proxyUrl) throw new Error("no stream available");
    this.applyStream(stream, height, positionMs);
  };

  loadMovie = async (id: number, gen: number) => {
    const preferredHeight = store.settings?.preferredHeight;
    const [details, historyItems, subtitles] = await Promise.all([
      api.movieDetails(id),
      api.getHistory().catch(() => []),
      api.movieSubtitles(id).catch(() => []),
    ]);

    if (gen !== this.loadGen) return;

    this.setState({ kind: "movie", mediaId: id });
    const { stream } = await fetchStreamWithFallback(
      initialQualityAttempts(preferredHeight),
      (height) => api.movieStream(id, height),
    );

    if (gen !== this.loadGen) return;

    const progress = historyItems.find((h) => h.kind === "movie" && h.mediaId === id);

    this.setState((prev) => ({
      streamUrl: stream.proxyUrl,
      isHls: stream.isHls,
      qualities: dedupeQualitiesByHeight(stream.qualities ?? []),
      selectedHeight: stream.selectedHeight ?? preferredHeight ?? 1080,
      streamGeneration: prev.streamGeneration + 1,
      subtitleTracks: subtitles,
      title: details.title,
      subtitle: details.year,
      episodeTitle: "",
      description: details.description,
      intro: null,
      nextEpisode: null,
      startPositionMs: progress && !progress.completed ? progress.positionMs : 0,
      loading: false,
      ready: true,
      kind: "movie",
      mediaId: id,
      poster: details.poster,
      historyPoster: details.poster,
    }));
  };

  loadEpisode = async (showId: number, season: number, episode: number, gen: number) => {
    const preferredHeight = store.settings?.preferredHeight;
    const [details, next, historyItems, subtitles, episodeDetails, seasons, menuEpisodes] =
      await Promise.all([
        api.showDetails(showId),
        api.nextEpisode(showId, season, episode).catch(() => null),
        api.getHistory().catch(() => []),
        api.episodeSubtitles(showId, season, episode).catch(() => []),
        api.episodeDetails(showId, season, episode).catch(() => null),
        api.showSeasons(showId).catch(() => []),
        api.seasonEpisodes(showId, season).catch(() => []),
      ]);

    if (gen !== this.loadGen) return;

    this.setState({ kind: "show", mediaId: showId, season, episode });
    const { stream } = await fetchStreamWithFallback(
      initialQualityAttempts(preferredHeight),
      (height) => api.episodeStream(showId, season, episode, height),
    );

    if (gen !== this.loadGen) return;

    const progress = historyItems.find(
      (h) =>
        h.kind === "show" &&
        h.mediaId === showId &&
        h.season === season &&
        h.episode === episode,
    );

    this.setState((prev) => ({
      streamUrl: stream.proxyUrl,
      isHls: stream.isHls,
      qualities: dedupeQualitiesByHeight(stream.qualities ?? []),
      selectedHeight: stream.selectedHeight ?? preferredHeight ?? 1080,
      streamGeneration: prev.streamGeneration + 1,
      subtitleTracks: subtitles,
      title: details.title,
      subtitle: `Season ${season}, Episode ${episode}`,
      episodeTitle: episodeDetails?.title ?? "",
      description: episodeDetails?.description || details.description,
      intro: null,
      nextEpisode: next,
      startPositionMs: progress && !progress.completed ? progress.positionMs : 0,
      loading: false,
      ready: true,
      kind: "show",
      mediaId: showId,
      season,
      episode,
      poster: episodeDetails?.poster || details.poster,
      historyPoster: details.poster,
      seasons,
      menuEpisodes,
      menuSeason: season,
      menuEpisodesLoading: false,
      episodeCache: menuEpisodes.length > 0 ? { [season]: menuEpisodes } : {},
    }));
  };

  loadMenuEpisodes = async (season: number) => {
    const { mediaId, kind, ready, episodeCache } = this.state;
    if (!ready || kind !== "show") return;

    const cached = episodeCache[season];
    if (cached) {
      this.setState({ menuSeason: season, menuEpisodes: cached, menuEpisodesLoading: false });
      return;
    }

    this.setState({ menuEpisodesLoading: true, menuSeason: season });
    try {
      const episodes = await api.seasonEpisodes(mediaId, season);
      if (this.state.mediaId !== mediaId) return;
      this.setState((prev) => ({
        menuEpisodes: episodes,
        menuEpisodesLoading: false,
        episodeCache: { ...prev.episodeCache, [season]: episodes },
      }));
    } catch {
      this.setState({ menuEpisodes: [], menuEpisodesLoading: false });
    }
  };

  handleEpisodeSelect = (season: number, episode: number) => {
    const { mediaId, kind } = this.state;
    if (kind !== "show") return;
    this.props.navigate(`/watch/show/${mediaId}/${season}/${episode}`);
  };

  loadLive = async (daddyId: string, gen: number) => {
    const stream = await api.liveStream(daddyId);
    if (gen !== this.loadGen) return;
    if (!stream?.proxyUrl) {
      throw new Error("no stream available for this channel");
    }

    this.setState({
      streamUrl: stream.proxyUrl,
      isHls: true,
      title: stream.channel.name,
      subtitle: stream.channel.category,
      intro: null,
      nextEpisode: null,
      startPositionMs: 0,
      loading: false,
      ready: true,
      kind: "live",
      mediaId: 0,
      channelId: daddyId,
      poster: stream.channel.logo,
    });
  };

  saveProgress = (positionMs: number, durationMs: number) => {
    if (this.progressDebounce) clearTimeout(this.progressDebounce);
    this.progressDebounce = setTimeout(async () => {
      const { kind, mediaId, title, poster, historyPoster, season, episode, channelId, ready } =
        this.state;
      if (!ready) return;

      const completed = durationMs > 0 && positionMs / durationMs > 0.9;

      try {
        await api.upsertHistory({
          kind,
          mediaId: kind === "live" ? 0 : mediaId,
          title,
          poster: kind === "show" ? historyPoster || poster : poster,
          season,
          episode,
          channelId,
          positionMs: Math.floor(positionMs),
          durationMs: Math.floor(durationMs),
          completed,
        });
      } catch {
        /* ignore */
      }
    }, 2000);
  };

  handleNextEpisode = () => {
    const { nextEpisode, mediaId } = this.state;
    if (!nextEpisode) return;
    this.props.navigate(
      `/watch/show/${mediaId}/${nextEpisode.season}/${nextEpisode.episode}`,
    );
  };

  handleSubtitlesEnabledChange = async (enabled: boolean) => {
    try {
      const updated = await api.updateSettings({ subtitlesEnabled: enabled });
      store.setSettings(updated);
      localStorage.setItem("streamly:subtitlesEnabled", enabled ? "1" : "0");
    } catch {
      // Preference still applies for this session via player state.
    }
  };

  loadIntro = async (durationMs: number) => {
    const { kind, mediaId, season, episode, ready } = this.state;
    if (!ready || durationMs <= 0) return;

    try {
      const intro =
        kind === "movie"
          ? await api.movieIntro(mediaId, durationMs)
          : kind === "show"
            ? await api.episodeIntro(mediaId, season, episode, durationMs)
            : null;
      if (intro) this.setState({ intro });
    } catch {
      /* intro metadata is optional */
    }
  };

  handleQualityChange = async (height: number, positionMs: number) => {
    const { ready, kind, selectedHeight } = this.state;
    if (!ready || kind === "live" || height === selectedHeight) return;

    try {
      await this.switchStream(height, positionMs);
      this.failedQualityHeights.clear();
    } catch {
      /* keep current stream on quality switch failure */
    }
  };

  handlePlaybackError = async (positionMs: number) => {
    const { ready, kind, selectedHeight, qualities } = this.state;
    if (!ready || kind === "live") return;

    this.failedQualityHeights.add(selectedHeight);
    let remaining = qualities;
    let nextHeight = nextLowerQualityHeight(remaining, selectedHeight);
    while (nextHeight !== null && this.failedQualityHeights.has(nextHeight)) {
      remaining = remaining.filter((q) => q.height !== nextHeight);
      nextHeight = nextLowerQualityHeight(remaining, nextHeight);
    }
    if (nextHeight === null || this.failedQualityHeights.size > qualities.length + 2) return;

    try {
      await this.switchStream(nextHeight, positionMs);
    } catch {
      this.failedQualityHeights.add(nextHeight);
      await this.handlePlaybackError(positionMs);
    }
  };

  handleBack = () => {
    if (window.history.length > 1) {
      history.back();
      return;
    }
    const { kind, mediaId } = this.state;
    if (kind === "movie" && mediaId) {
      this.props.navigate(`/movie/${mediaId}`);
    } else if (kind === "show" && mediaId) {
      this.props.navigate(`/show/${mediaId}`);
    } else {
      this.props.navigate("/");
    }
  };

  render() {
    const {
      streamUrl,
      isHls,
      qualities,
      selectedHeight,
      subtitleTracks,
      title,
      subtitle,
      episodeTitle,
      description,
      poster,
      intro,
      nextEpisode,
      startPositionMs,
      loading,
      error,
      ready,
      seasons,
      menuEpisodes,
      menuSeason,
      menuEpisodesLoading,
      season,
      episode,
      kind,
      streamGeneration,
    } = this.state;
    const settings = store.settings;

    if (loading) {
      return (
        <div className="flex h-screen items-center justify-center bg-black">
          <div className="h-8 w-8 animate-spin rounded-full border-2 border-foreground/20 border-t-foreground" />
        </div>
      );
    }

    if (error || !ready || !streamUrl) {
      return (
        <div className="flex h-screen flex-col items-center justify-center gap-4 bg-black px-6">
          <p className="text-center text-sm text-foreground-muted">
            {error || "unable to start playback"}
          </p>
          <div className="flex gap-3">
            <Button variant="outline" onClick={this.load}>
              Retry
            </Button>
            <Button variant="outline" onClick={() => this.props.navigate("/")}>
              Go home
            </Button>
          </div>
        </div>
      );
    }

    return (
      <div className="relative h-screen overflow-hidden bg-black">
        <VideoPlayer
          key={`${streamUrl}-${selectedHeight}-${streamGeneration}`}
          src={streamUrl}
          isHls={isHls}
          live={this.state.kind === "live"}
          lowLatency={this.state.kind === "live"}
          title={title}
          subtitle={subtitle}
          episodeTitle={episodeTitle}
          description={description}
          poster={poster}
          qualities={qualities}
          selectedHeight={selectedHeight}
          subtitleTracks={subtitleTracks}
          intro={this.state.kind === "live" ? null : intro}
          nextEpisode={this.state.kind === "live" ? null : nextEpisode}
          autoPlayNext={this.state.kind !== "live" && (settings?.autoPlayNext ?? true)}
          skipIntroEnabled={this.state.kind !== "live" && (settings?.skipIntro ?? true)}
          ambienceEnabled={settings?.ambienceEnabled ?? true}
          subtitlesEnabled={settings?.subtitlesEnabled ?? false}
          onSubtitlesEnabledChange={this.handleSubtitlesEnabledChange}
          startPositionMs={this.state.kind === "live" ? 0 : startPositionMs}
          onBack={this.handleBack}
          onProgress={this.state.kind === "live" ? undefined : this.saveProgress}
          onNextEpisode={this.handleNextEpisode}
          onQualityChange={
            this.state.kind === "live" ? undefined : this.handleQualityChange
          }
          onDurationReady={this.state.kind === "live" ? undefined : this.loadIntro}
          onPlaybackError={
            this.state.kind === "live" ? undefined : this.handlePlaybackError
          }
          seasons={kind === "show" ? seasons : undefined}
          episodes={kind === "show" ? menuEpisodes : undefined}
          currentSeason={kind === "show" ? season : undefined}
          currentEpisode={kind === "show" ? episode : undefined}
          menuSeason={kind === "show" ? menuSeason : undefined}
          episodesLoading={kind === "show" ? menuEpisodesLoading : undefined}
          onSeasonChange={kind === "show" ? this.loadMenuEpisodes : undefined}
          onEpisodeSelect={kind === "show" ? this.handleEpisodeSelect : undefined}
        />
      </div>
    );
  }
}