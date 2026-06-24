import { api, ApiError } from "@/api/client";

import { VideoPlayer } from "@/components/player/VideoPlayer";
import { Button } from "@/components/ui/Button";
import { SettingsPanel } from "@/pages/SettingsPanel";

import { history, navigate, saveReturnPath, type NavigateFn } from "@/lib/navigation";
import { store } from "@/lib/store";

import type { Episode, IntroInfo, NextEpisode, Season, StreamInfo, StreamQuality, SubtitleTrack, WatchHistoryItem, } from "@/lib/types";
import { closestAvailableHeight, dedupeQualitiesByHeight, nextLowerQualityHeight, } from "@/lib/stream";
import { pickQualityByHeight, qualityPlaybackUrl, streamFromQuality, streamPlaybackUrl, } from "@/lib/streamClient";

import { parseWatchPath } from "@/lib/watchRoute";

import { Component } from "react";
import { AlertTriangle, ArrowLeft } from "lucide-react";

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

  ready: boolean;

  settingsOpen: boolean;

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

  settingsOpen: false,

};

function episodeProgress(items: WatchHistoryItem[], showId: number, season: number, episode: number): number {

  const entry = items.find((item) => item.kind === "show" && item.mediaId === showId && item.season === season && item.episode === episode && !item.completed);
  return entry?.positionMs ?? 0;

}

function movieProgress(items: WatchHistoryItem[], movieId: number): number {

  const entry = items.find((item) => item.kind === "movie" && item.mediaId === movieId && !item.completed);

  return entry?.positionMs ?? 0;

}

export class WatchPage extends Component<WatchPageProps, WatchPageState> {

  private progressDebounce: ReturnType<typeof setTimeout> | null = null;

  private lastProgressSave = 0;

  private pendingProgress: { positionMs: number; durationMs: number } | null = null;

  private unsubscribe = () => {};

  private loadGen = 0;

  private failedQualityHeights = new Set<number>();

  private userSelectedQuality = false;

  private lastPlaybackPositionMs = 0;

  private lastPreferredHeight = 1080;

  state: WatchPageState = {

    ...EMPTY_STATE,

    loading: true,
    error: "",

    ready: false,

    settingsOpen: false,

  };

  componentDidMount() {

    this.lastPreferredHeight = this.preferredHeight();

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

    if (this.pendingProgress) {

      void this.writeProgress(this.pendingProgress.positionMs, this.pendingProgress.durationMs);

    }

  }

  onStoreChange = () => {

    const nextPreferred = this.preferredHeight();
    const preferredChanged = nextPreferred !== this.lastPreferredHeight;

    this.lastPreferredHeight = nextPreferred;

    this.forceUpdate();

    if (preferredChanged) {

      this.userSelectedQuality = false;
      this.ensurePreferredQuality(this.lastPlaybackPositionMs);

    }

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
    this.userSelectedQuality = false;

    this.setState({

      ...EMPTY_STATE,

      loading: true,
      error: "",

      ready: false,

    });

    if (!route.valid) {

      this.setState({ error: route.reason || "unknwon error", loading: false });

      return;

    }

    try {

      if (route.kind === "movie") {

        await this.loadMovie(route.id || 0, gen);

      } else if (route.kind === "show") {

        await this.loadEpisode(route.showId || 0, route.season || 1, route.episode || 1, gen);

      } else {

        await this.loadLive(route.channelId || "0", gen);

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

  preferredHeight = (): number => store.settings?.preferredHeight ?? 1080;

  resolvedPreferredHeight = (qualities: StreamQuality[]): number => {

    return closestAvailableHeight(qualities, this.preferredHeight()) ?? this.preferredHeight();

  };

  ensurePreferredQuality = (positionMs: number) => {

    if (this.userSelectedQuality) return;

    const { kind, qualities, selectedHeight, ready } = this.state;

    if (!ready || kind === "live") return;

    const resolved = this.resolvedPreferredHeight(qualities);

    if (resolved === selectedHeight) return;

    this.switchStream(resolved, positionMs);

  };

  mergeQualities = (incoming: StreamQuality[] | undefined, previous: StreamQuality[]): StreamQuality[] => {

    return dedupeQualitiesByHeight(incoming ?? previous);

  };

  applyStream = (stream: StreamInfo, requestedHeight: number, positionMs: number) => {

    const playbackUrl = streamPlaybackUrl(stream);

    if (!playbackUrl) throw new Error("no stream available");

    this.setState((prev) => {

      const qualities = this.mergeQualities(stream.qualities, prev.qualities);
      const resolvedHeight = this.userSelectedQuality ? (stream.selectedHeight ?? requestedHeight) : this.resolvedPreferredHeight(qualities);

      return {

        ...prev,

        streamUrl: playbackUrl,

        isHls: stream.isHls,

        qualities,
        selectedHeight: resolvedHeight,

        startPositionMs: Math.floor(positionMs),

        error: "",

      };

    });

  };

  switchStream = (height: number, positionMs: number) => {

    const { qualities, streamUrl } = this.state;

    const quality = pickQualityByHeight(qualities, height);

    if (!quality || !qualityPlaybackUrl(quality)) return;

    const nextUrl = qualityPlaybackUrl(quality);

    if (quality.isHls && nextUrl === streamUrl) {

      this.userSelectedQuality = true;

      this.setState({

        selectedHeight: height,
        startPositionMs: Math.floor(positionMs),
        error: "",

      });

      return;

    }

    this.applyStream(streamFromQuality(qualities, quality, height), height, positionMs);

  };

  loadMovie = async (id: number, gen: number) => {

    const [streamData, historyItems] = await Promise.all([

      api.movieStream(id),
      api.getHistory(5, id).catch(() => []),

    ]);

    if (gen !== this.loadGen) return;

    const qualities = dedupeQualitiesByHeight(streamData.qualities ?? []);

    if (qualities.length === 0) throw new Error("no stream available");

    const resolvedHeight = this.resolvedPreferredHeight(qualities);
    const selectedQuality = pickQualityByHeight(qualities, resolvedHeight) ?? qualities[0];
    const playbackUrl = qualityPlaybackUrl(selectedQuality);

    if (!playbackUrl) throw new Error("no stream available");

    const startPositionMs = movieProgress(historyItems, id);

    this.setState({

      streamUrl: playbackUrl,

      isHls: selectedQuality.isHls,

      qualities,
      selectedHeight: resolvedHeight,

      startPositionMs,

      loading: false,
      ready: true,

      kind: "movie",
      mediaId: id,

      subtitle: "",

    });

    void this.enrichMovie(id, gen);

    void this.loadMovieSubtitles(id, gen);

  };

  enrichMovie = async (id: number, gen: number) => {

    try {

      const details = await api.movieDetails(id);

      if (gen !== this.loadGen) return;

      this.setState({

        title: details.title,
        subtitle: details.year,
        description: details.description,

        poster: details.poster,
        historyPoster: details.poster,

      });

    } catch {

      /* metadata is optional for playback */

    }

  };

  loadMovieSubtitles = async (id: number, gen: number) => {

    try {

      const subtitles = await api.movieSubtitles(id);

      if (gen !== this.loadGen) return;

      this.setState({ subtitleTracks: subtitles });

    } catch {

      /* subtitles are optional */

    }

  };

  loadEpisode = async (showId: number, season: number, episode: number, gen: number) => {

    const menuPromise = this.loadMenuData(showId, season);

    const [streamData, historyItems] = await Promise.all([

      api.episodeStream(showId, season, episode),
      api.getHistory(30, showId).catch(() => []),

    ]);

    if (gen !== this.loadGen) return;

    const qualities = dedupeQualitiesByHeight(streamData.qualities ?? []);

    if (qualities.length === 0) throw new Error("no stream available");

    const resolvedHeight = this.resolvedPreferredHeight(qualities);
    const selectedQuality = pickQualityByHeight(qualities, resolvedHeight) ?? qualities[0];
    const playbackUrl = qualityPlaybackUrl(selectedQuality);

    if (!playbackUrl) throw new Error("no stream available");

    const startPositionMs = episodeProgress(historyItems, showId, season, episode);

    this.setState(

      {

        streamUrl: playbackUrl,

        isHls: selectedQuality.isHls,

        qualities,
        selectedHeight: resolvedHeight,

        startPositionMs,

        loading: false,
        ready: true,

        kind: "show",
        mediaId: showId,

        season,
        episode,

        subtitle: `Season ${season}, Episode ${episode}`,
        menuSeason: season,

      },

      () => {

        void this.loadMenuEpisodes(season);

      }

    );

    void this.enrichEpisode(showId, season, episode, gen);

    void this.loadEpisodeSubtitles(showId, season, episode, gen);

    void this.loadNextEpisode(showId, season, episode, gen);

    void this.applyMenuData(showId, season, menuPromise);

  };

  enrichEpisode = async (showId: number, season: number, episode: number, gen: number) => {

    try {

      const [details, episodeDetails] = await Promise.all([

        api.showDetails(showId),
        api.episodeDetails(showId, season, episode).catch(() => null),

      ]);

      if (gen !== this.loadGen) return;

      this.setState({

        title: details.title,
        episodeTitle: episodeDetails?.title ?? "",

        description: episodeDetails?.description || details.description,

        poster: episodeDetails?.poster || details.poster,
        historyPoster: details.poster,

      });

    } catch {

      /* metadata is optional for playback */

    }

  };

  loadEpisodeSubtitles = async (showId: number, season: number, episode: number, gen: number) => {

    try {

      const subtitles = await api.episodeSubtitles(showId, season, episode);

      if (gen !== this.loadGen) return;

      this.setState({ subtitleTracks: subtitles });

    } catch {

      /* subtitles are optional */

    }

  };

  loadNextEpisode = async (showId: number, season: number, episode: number, gen: number) => {

    try {

      const next = await api.nextEpisode(showId, season, episode);

      if (gen !== this.loadGen) return;

      if (next) this.setState({ nextEpisode: next });

    } catch {

      /* up-next is optional */

    }

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

    const data = await this.loadMenuData(mediaId, season);

    if (!data) {

      this.setState({ menuEpisodes: [], menuEpisodesLoading: false });
      return;

    }

    if (this.state.mediaId !== mediaId) return;

    this.setState((prev) => ({

      seasons: prev.seasons.length > 0 ? prev.seasons : data.seasons,

      menuEpisodes: data.episodes,
      menuEpisodesLoading: false,

      episodeCache: { ...prev.episodeCache, [season]: data.episodes },

    }));

  };

  loadMenuData = async (showId: number, season: number): Promise<{ seasons: Season[]; episodes: Episode[] } | null> => {

    try {

      const [seasons, episodes] = await Promise.all([

        this.state.seasons.length > 0 ? Promise.resolve(this.state.seasons) : api.showSeasons(showId).catch(() => []),
        api.seasonEpisodes(showId, season),

      ]);

      return { seasons, episodes };

    } catch {

      return null;

    }

  };

  applyMenuData = async (showId: number, season: number, dataPromise: Promise<{ seasons: Season[]; episodes: Episode[] } | null>) => {

    const data = await dataPromise;

    if (!data || this.state.mediaId !== showId || this.state.kind !== "show") return;

    this.setState((prev) => {

      if (prev.episodeCache[season]) return null;

      return {

        seasons: prev.seasons.length > 0 ? prev.seasons : data.seasons,
        menuEpisodes: prev.menuSeason === season ? data.episodes : prev.menuEpisodes,
        menuEpisodesLoading: prev.menuSeason === season ? false : prev.menuEpisodesLoading,
        episodeCache: { ...prev.episodeCache, [season]: data.episodes },

      };

    });

  };

  handleEpisodeSelect = (season: number, episode: number) => {

    const { mediaId, kind } = this.state;

    if (kind !== "show") return;

    this.props.navigate(`/watch/show/${mediaId}/${season}/${episode}`);

  };

  loadLive = async (daddyId: string, gen: number) => {

    const stream = await api.liveStream(daddyId);

    if (gen !== this.loadGen) return;

    const playbackUrl = streamPlaybackUrl(stream);

    if (!playbackUrl) {

      throw new Error("no stream available for this channel");

    }

    const channelTitle = stream.channel.name?.trim() || stream.channel.slug?.trim() || `Channel ${daddyId}`;

    this.setState({

      streamUrl: playbackUrl,

      isHls: true,

      title: channelTitle,
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

    this.lastPlaybackPositionMs = positionMs;

    if (this.progressDebounce) clearTimeout(this.progressDebounce);

    this.pendingProgress = { positionMs, durationMs };

    const now = Date.now();
    const elapsed = now - this.lastProgressSave;
    const wait = Math.max(500, 3000 - elapsed);

    this.progressDebounce = setTimeout(() => {

      const pending = this.pendingProgress;

      if (!pending) return;

      this.pendingProgress = null;
      this.lastProgressSave = Date.now();

      void this.writeProgress(pending.positionMs, pending.durationMs);

    }, wait);

  };

  writeProgress = async (positionMs: number, durationMs: number) => {

    const { kind, mediaId, title, poster, historyPoster, season, episode, channelId, ready } = this.state;

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

  };

  handleNextEpisode = () => {

    const { nextEpisode, mediaId } = this.state;

    if (!nextEpisode) return;

    this.props.navigate(`/watch/show/${mediaId}/${nextEpisode.season}/${nextEpisode.episode}`);

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

      const intro = kind === "movie" ? await api.movieIntro(mediaId, durationMs) : kind === "show" ? await api.episodeIntro(mediaId, season, episode, durationMs) : null;

      if (intro) this.setState({ intro });

    } catch {

      /* intro metadata is optional */

    }

  };

  handleQualityChange = (height: number, positionMs: number) => {

    const { ready, kind, selectedHeight } = this.state;

    if (!ready || kind === "live" || height === selectedHeight) return;

    this.userSelectedQuality = true;
    this.failedQualityHeights.clear();

    this.switchStream(height, positionMs);

  };

  handlePlaybackError = (positionMs: number) => {

    const { ready, kind, selectedHeight, qualities } = this.state;

    if (!ready || kind === "live") return;

    this.failedQualityHeights.add(selectedHeight);

    let remaining = qualities;
    let nextHeight = nextLowerQualityHeight(remaining, selectedHeight);

    while (nextHeight !== null && this.failedQualityHeights.has(nextHeight)) {

      remaining = remaining.filter((q) => q.height !== nextHeight);

      nextHeight = nextLowerQualityHeight(remaining, nextHeight);

    }

    if (nextHeight === null || this.failedQualityHeights.size > qualities.length + 2) {

      this.handleFatalError();

      return;

    }

    this.switchStream(nextHeight, positionMs);

  };

  handleFatalError = () => {

    if (this.state.error) return;

    const message = this.state.kind === "live"
      ? "This channel is unavailable right now. It may be offline or between broadcasts."
      : "Playback failed. This title may be temporarily unavailable.";

    this.setState({ error: message, ready: false });

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

    const { streamUrl, isHls, qualities, selectedHeight, subtitleTracks, title, subtitle, episodeTitle, description, poster, intro, nextEpisode, startPositionMs, loading, error, ready, seasons, menuEpisodes, menuSeason, menuEpisodesLoading, season, episode, kind, mediaId, channelId, settingsOpen } = this.state;
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

        <div className="relative flex h-screen flex-col items-center justify-center gap-5 bg-black px-6">

          <button onClick={this.handleBack}
            className="absolute left-4 top-[calc(env(safe-area-inset-top,0px)+1rem)] flex items-center gap-2 rounded-md border border-border-subtle bg-surface/80 px-3 py-1.5 text-xs text-foreground backdrop-blur-md transition-colors hover:bg-surface-overlay"
          >

            <ArrowLeft size={14} />
            Back

          </button>

          <AlertTriangle size={32} className="text-foreground-faint" />

          <p className="max-w-sm text-center text-sm text-foreground-muted">

            {error || "Unable to start playback."}

          </p>

          <div className="flex gap-3">

            <Button onClick={this.load}>

              Retry

            </Button>

            <Button variant="outline" onClick={this.handleBack}>

              Go back

            </Button>

          </div>

        </div>

      );

    }

    return (

      <div className="relative h-screen overflow-hidden bg-black">

        <VideoPlayer

          key={kind === "live" ? `live-${channelId}` : `${kind}-${mediaId}-${season}-${episode}`}
          src={streamUrl}

          isHls={isHls}

          live={this.state.kind === "live"}
          lowLatency={this.state.kind === "live"}

          title={title || subtitle}
          subtitle={title ? subtitle : undefined}
          episodeTitle={episodeTitle}
          description={description}
          poster={poster}

          qualities={qualities}
          selectedHeight={selectedHeight}
          preferredHeight={settings?.preferredHeight ?? 1080}

          subtitleTracks={subtitleTracks}

          intro={this.state.kind === "live" ? null : intro}
          nextEpisode={this.state.kind === "live" ? null : nextEpisode}
          autoPlayNext={this.state.kind !== "live" && (settings?.autoPlayNext ?? true)}
          skipIntroEnabled={this.state.kind !== "live" && (settings?.skipIntro ?? true)}

          ambienceEnabled={settings?.ambienceEnabled ?? true}
          subtitlesEnabled={settings?.subtitlesEnabled ?? false}

          onBack={this.handleBack}
          onNextEpisode={this.handleNextEpisode}
          onSubtitlesEnabledChange={this.handleSubtitlesEnabledChange}
          onSeasonChange={kind === "show" ? this.loadMenuEpisodes : undefined}
          onProgress={this.state.kind === "live" ? undefined : this.saveProgress}
          onEpisodeSelect={kind === "show" ? this.handleEpisodeSelect : undefined}
          onDurationReady={this.state.kind === "live" ? undefined : this.loadIntro}
          onQualityChange={this.state.kind === "live" ? undefined : this.handleQualityChange}
          onOpenSettings={() => this.setState({ settingsOpen: true })}
          onPlaybackError={this.state.kind === "live" ? undefined : this.handlePlaybackError}
          onFatalError={this.handleFatalError}

          startPositionMs={this.state.kind === "live" ? 0 : startPositionMs}

          seasons={kind === "show" ? seasons : undefined}
          episodes={kind === "show" ? menuEpisodes : undefined}

          currentSeason={kind === "show" ? season : undefined}
          currentEpisode={kind === "show" ? episode : undefined}

          menuSeason={kind === "show" ? menuSeason : undefined}

          episodesLoading={kind === "show" ? menuEpisodesLoading : undefined}

        />

        <SettingsPanel open={settingsOpen} onClose={() => this.setState({ settingsOpen: false })} />

      </div>

    );

  }

}
