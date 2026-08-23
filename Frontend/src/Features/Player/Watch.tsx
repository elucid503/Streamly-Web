import { AlertTriangle, ArrowLeft } from "lucide-react";

import type { MultiviewStream } from "@/Features/Player/LiveStreamPane";
import { MULTIVIEW_MAX_STREAMS } from "@/Features/Player/MultiviewMenu";
import { VideoPlayer } from "@/Features/Player/VideoPlayer";
import { SettingsPanel } from "@/Features/User/Settings";
import { Button } from "@/UI/Button";

import { ModuleComponent } from "@/Core/Store";
import Net, { ApiError } from "@/Net";
import Stores from "@/Stores";
import type { Episode, IntroInfo, LiveChannel, LiveSourceProvider, NextEpisode, Season, StreamInfo, StreamQuality, SubtitleTrack, WatchHistoryItem } from "@/Types";
import { history, navigate, saveReturnPath, type NavigateFn } from "@/Utils/Navigation";
import { closestAvailableHeight, dedupeQualitiesByHeight, nextLowerQualityHeight } from "@/Utils/Player/Stream";
import { pickQualityByHeight, qualityPlaybackUrl, streamFromQuality, streamPlaybackUrl } from "@/Utils/Player/StreamClient";
import { parseWatchPath } from "@/Utils/Player/WatchRoute";

interface WatchPageProps {

  navigate: NavigateFn;

  watchPath: string;
  minimized?: boolean;
  onMinimize?: (path: string) => void;
  onReturn?: () => void;
  onDismiss?: () => void;
  onReadyChange?: (ready: boolean) => void;

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

  multiviewStreams: MultiviewStream[];
  multiviewChannels: LiveChannel[];
  multiviewLoading: boolean;

  /** Anonymized live source options + selection. */
  sourceProviders: LiveSourceProvider[];
  selectedSourceKey: string;
  sourceSwitching: boolean;

  /** Session subtitle preference; starts on and follows track heuristic. */
  subtitlesOn: boolean;

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

  multiviewStreams: [],
  multiviewChannels: [],
  multiviewLoading: false,

  sourceProviders: [],
  selectedSourceKey: "auto",
  sourceSwitching: false,

  subtitlesOn: true,

};

function episodeProgress(items: WatchHistoryItem[], showId: number, season: number, episode: number): number {

  const entry = items.find((item) => item.kind === "show" && item.mediaId === showId && item.season === season && item.episode === episode && !item.completed);
  return entry?.positionMs ?? 0;

}

function movieProgress(items: WatchHistoryItem[], movieId: number): number {

  const entry = items.find((item) => item.kind === "movie" && item.mediaId === movieId && !item.completed);

  return entry?.positionMs ?? 0;

}

export class WatchPage extends ModuleComponent<WatchPageProps, WatchPageState> {

  private progressDebounce: ReturnType<typeof setTimeout> | null = null;
  private lastProgressSave = 0;
  private liveActivityTimer: ReturnType<typeof setInterval> | null = null;

  private pendingProgress: { positionMs: number; durationMs: number } | null = null;

  private loadGen = 0;

  private failedQualityHeights = new Set<number>();

  private failedLiveSources = new Set<string>();

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

    this.watch(Stores.Auth);
    this.watch(Stores.Settings);

    this.lastPreferredHeight = this.preferredHeight();

    this.tryLoad();

  }

  componentDidUpdate(prev: WatchPageProps, prevState: WatchPageState) {

    if (prev.watchPath !== this.props.watchPath) {

      this.tryLoad();

    }

    if (prevState.ready !== this.state.ready) {

      this.props.onReadyChange?.(this.state.ready);

    }

    const nextPreferred = this.preferredHeight();

    if (nextPreferred !== this.lastPreferredHeight) {

      this.lastPreferredHeight = nextPreferred;
      this.userSelectedQuality = false;
      this.ensurePreferredQuality(this.lastPlaybackPositionMs);

    }

  }

  componentWillUnmount() {

    if (this.progressDebounce) clearTimeout(this.progressDebounce);
    if (this.liveActivityTimer) clearInterval(this.liveActivityTimer);

    if (this.pendingProgress) {

      void this.writeProgress(this.pendingProgress.positionMs, this.pendingProgress.durationMs);

    }

  }

  tryLoad = () => {

    if (!Stores.Auth.isAuthenticated) {

      saveReturnPath(history.location.pathname);

      navigate("/auth");

      return;

    }

    this.load();

  };

  handleAuthFailure = () => {

    saveReturnPath(history.location.pathname);

    Stores.Auth.setUser(null);

    Stores.Settings.setSettings(null);

    navigate("/auth");

  };

  load = async () => {

    const gen = ++this.loadGen;

    this.stopLiveActivity();
    this.props.onReadyChange?.(false);

    const route = parseWatchPath(this.props.watchPath);

    this.failedQualityHeights.clear();
    this.userSelectedQuality = false;

    const routeKind = route.valid
      ? route.kind === "movie"
        ? "movie" as const
        : route.kind === "show"
          ? "show" as const
          : "live" as const
      : this.state.kind;

    this.setState({

      ...EMPTY_STATE,

      kind: routeKind,
      mediaId: route.valid && route.kind === "movie" ? (route.id || 0) : route.valid && route.kind === "show" ? (route.showId || 0) : 0,
      season: route.valid && route.kind === "show" ? (route.season || 1) : 0,
      episode: route.valid && route.kind === "show" ? (route.episode || 1) : 0,
      channelId: route.valid && route.kind === "live" ? (route.channelId || "") : "",

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

  preferredHeight = (): number => Stores.Settings.settings?.preferredHeight ?? 1080;

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

      Net.Stream.movie(id),
      Net.History.get(5, id).catch(() => []),

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

      const details = await Net.Catalog.movieDetails(id);

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

      const subtitles = await Net.Stream.movieSubtitles(id);

      if (gen !== this.loadGen) return;

      this.setState({ subtitleTracks: subtitles });

    } catch {

      /* subtitles are optional */

    }

  };

  loadEpisode = async (showId: number, season: number, episode: number, gen: number) => {

    // Drop prior show menu state immediately so async loads cannot reuse old seasons/cache.
    this.setState({

      seasons: [],
      episodeCache: {},
      menuEpisodes: [],
      menuEpisodesLoading: true,
      menuSeason: season,

    });

    const menuPromise = this.loadMenuData(showId, season);

    const [streamData, historyItems] = await Promise.all([

      Net.Stream.episode(showId, season, episode),
      Net.History.get(30, showId).catch(() => []),

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

        subtitle: `S${season} E${episode}`,
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

        Net.Catalog.showDetails(showId),
        Net.Catalog.episodeDetails(showId, season, episode).catch(() => null),

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

      const subtitles = await Net.Stream.episodeSubtitles(showId, season, episode);

      if (gen !== this.loadGen) return;

      this.setState({ subtitleTracks: subtitles });

    } catch {

      /* subtitles are optional */

    }

  };

  loadNextEpisode = async (showId: number, season: number, episode: number, gen: number) => {

    try {

      const next = await Net.Stream.nextEpisode(showId, season, episode);

      if (gen !== this.loadGen) return;

      if (next) this.setState({ nextEpisode: next });

    } catch {

      /* up-next is optional */

    }

  };

  loadMenuEpisodes = async (season: number) => {

    const { mediaId, kind, ready, episodeCache, seasons } = this.state;

    if (!ready || kind !== "show") return;

    const cached = episodeCache[season];

    if (cached && seasons.length > 0) {

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

      seasons: data.seasons,

      menuEpisodes: data.episodes,
      menuEpisodesLoading: false,

      episodeCache: { ...prev.episodeCache, [season]: data.episodes },

    }));

  };

  loadMenuData = async (showId: number, season: number): Promise<{ seasons: Season[]; episodes: Episode[] } | null> => {

    try {

      const [seasons, episodes] = await Promise.all([

        Net.Catalog.showSeasons(showId).catch(() => []),
        Net.Catalog.seasonEpisodes(showId, season),

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

      if (prev.episodeCache[season]) {

        return { seasons: data.seasons } as Pick<WatchPageState, "seasons">;

      }

      return {

        seasons: data.seasons,
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

  loadLive = async (channelId: string, gen: number, providerKey = "auto") => {

    this.failedLiveSources.clear();

    const [stream, providers] = await Promise.all([
      Net.Live.stream(channelId, providerKey === "auto" ? undefined : providerKey),
      this.state.sourceProviders.length > 0
        ? Promise.resolve(this.state.sourceProviders)
        : Net.Live.providers().catch(() => [] as LiveSourceProvider[]),
    ]);

    if (gen !== this.loadGen) return;

    if (!stream.streamUrl?.trim()) {

      throw new Error("no stream available for this channel");

    }

    const channelTitle = stream.channel?.name?.trim() || `Channel ${channelId}`;

    const selected = stream.provider?.trim() || providerKey || "auto";

    this.setState({

      streamUrl: stream.streamUrl,

      isHls: true,

      title: channelTitle,
      subtitle: stream.channel?.category ?? "",

      intro: null,
      nextEpisode: null,

      startPositionMs: 0,

      loading: false,
      ready: true,

      kind: "live",

      mediaId: 0,
      channelId: channelId,

      poster: stream.channel?.logo,

      // Reset multiview when switching the primary live channel.
      multiviewStreams: [],
      multiviewChannels: [],
      multiviewLoading: false,

      sourceProviders: providers ?? [],
      selectedSourceKey: selected,
      sourceSwitching: false,

    });

  };

  handleSourceChange = async (key: string, opts?: { failover?: boolean }) => {

    const { channelId, kind, selectedSourceKey } = this.state;

    if (kind !== "live" || !channelId || key === selectedSourceKey) return;

    if (!opts?.failover) {

      this.failedLiveSources.clear();

    }

    const gen = ++this.loadGen;

    this.setState({ sourceSwitching: true, selectedSourceKey: key, streamUrl: "", ready: false, error: "" });

    try {

      const stream = await Net.Live.stream(channelId, key === "auto" ? undefined : key);

      if (gen !== this.loadGen) return;

      if (!stream.streamUrl?.trim()) {

        throw new Error("no stream available from this source");

      }

      this.setState({

        streamUrl: stream.streamUrl,
        isHls: true,
        ready: true,
        sourceSwitching: false,
        selectedSourceKey: stream.provider?.trim() || key,
        error: "",

      });

    } catch (err) {

      if (gen !== this.loadGen) return;

      if (err instanceof ApiError && (err.status === 401 || err.status === 403)) {

        this.handleAuthFailure();
        return;

      }

      if (opts?.failover) {

        this.failedLiveSources.add(key);
        this.setState({ sourceSwitching: false });
        this.handleLiveFailover();
        return;

      }

      this.setState({

        sourceSwitching: false,
        ready: false,
        error: err instanceof Error ? err.message : "failed to switch source",

      });

    }

  };

  /** Advance to the next anonymized live source after soft recovery / stall. */
  handleLiveFailover = () => {

    const { kind, channelId, sourceProviders, selectedSourceKey, sourceSwitching } = this.state;

    if (kind !== "live" || !channelId || sourceSwitching) return;

    const keys = sourceProviders.map((provider) => provider.key).filter((key) => key !== "auto");

    if (keys.length === 0) {

      this.handleFatalError();
      return;

    }

    const current = selectedSourceKey === "auto" ? (keys[0] ?? "auto") : selectedSourceKey;

    this.failedLiveSources.add(current);

    const next = keys.find((key) => !this.failedLiveSources.has(key));

    if (!next) {

      this.handleFatalError();
      return;

    }

    void this.handleSourceChange(next, { failover: true });

  };

  startLiveActivity = () => {

    this.stopLiveActivity();
    void this.writeProgress(0, 0);

    this.liveActivityTimer = setInterval(() => {

      void this.writeProgress(0, 0);

    }, 60_000);

  };

  stopLiveActivity = () => {

    if (!this.liveActivityTimer) return;

    clearInterval(this.liveActivityTimer);
    this.liveActivityTimer = null;

  };

  handleMultiviewSearch = async (query: string) => {

    if (this.state.kind !== "live") return;

    this.setState({ multiviewLoading: true });

    try {

      const channels = query.trim()
        ? await Net.Live.search(query.trim())
        : await Net.Live.popular(48);

      this.setState({ multiviewChannels: channels ?? [], multiviewLoading: false });

    } catch {

      this.setState({ multiviewLoading: false });

    }

  };

  private multiviewLoadGen = new Map<string, number>();

  bumpMultiviewGen = (channelId: string) => {

    const next = (this.multiviewLoadGen.get(channelId) ?? 0) + 1;
    this.multiviewLoadGen.set(channelId, next);
    return next;

  };

  handleMultiviewToggle = async (channel: LiveChannel) => {

    if (this.state.kind !== "live") return;

    const { channelId, multiviewStreams } = this.state;

    if (channel.id === channelId) return;

    const existing = multiviewStreams.find((s) => s.channelId === channel.id);

    if (existing) {

      this.bumpMultiviewGen(channel.id);

      this.setState({

        multiviewStreams: multiviewStreams.filter((s) => s.channelId !== channel.id),

      });

      return;

    }

    // Primary + additional streams cap.
    if (1 + multiviewStreams.length >= MULTIVIEW_MAX_STREAMS) return;

    const loadGen = this.bumpMultiviewGen(channel.id);

    // Optimistically open the pane so layout/selection update immediately.
    const pending: MultiviewStream = {

      channelId: channel.id,
      name: channel.name,
      streamUrl: "",
      isHls: true,
      logo: channel.logo,
      pending: true,

    };

    this.setState((s) => {

      if (s.channelId !== channelId) return null;

      if (s.multiviewStreams.some((m) => m.channelId === channel.id)) return null;

      if (1 + s.multiviewStreams.length >= MULTIVIEW_MAX_STREAMS) return null;

      return { multiviewStreams: [...s.multiviewStreams, pending] };

    });

    try {

      const stream = await Net.Live.stream(channel.id);

      if (this.multiviewLoadGen.get(channel.id) !== loadGen) return;

      if (!stream.streamUrl?.trim()) {

        this.setState((s) => ({

          multiviewStreams: s.multiviewStreams.map((m) =>

            m.channelId === channel.id
              ? { ...m, pending: false, error: true, streamUrl: "" }
              : m

          ),

        }));

        return;

      }

      const ready: MultiviewStream = {

        channelId: channel.id,
        name: stream.channel?.name?.trim() || channel.name,
        streamUrl: stream.streamUrl,
        isHls: stream.isHls !== false,
        logo: stream.channel?.logo || channel.logo,
        pending: false,

      };

      this.setState((s) => {

        if (s.channelId !== channelId) return null;

        if (!s.multiviewStreams.some((m) => m.channelId === channel.id)) return null;

        return {

          multiviewStreams: s.multiviewStreams.map((m) =>

            m.channelId === channel.id ? ready : m

          ),

        };

      });

    } catch {

      if (this.multiviewLoadGen.get(channel.id) !== loadGen) return;

      this.setState((s) => ({

        multiviewStreams: s.multiviewStreams.map((m) =>

          m.channelId === channel.id
            ? { ...m, pending: false, error: true, streamUrl: "" }
            : m

        ),

      }));

    }

  };

  handleMultiviewRemove = (removeId: string) => {

    this.bumpMultiviewGen(removeId);

    const { channelId, multiviewStreams, streamUrl, poster } = this.state;

    // Removing a secondary pane.
    if (removeId !== channelId) {

      this.setState({

        multiviewStreams: multiviewStreams.filter((m) => m.channelId !== removeId),

      });

      return;

    }

    // Removing the primary pane: promote the first ready secondary stream.
    const remaining = multiviewStreams.filter((m) => m.channelId !== removeId);
    const nextPrimary = remaining.find((m) => m.streamUrl.trim() && !m.error) ?? remaining[0];

    if (!nextPrimary) {

      this.setState({ multiviewStreams: [] });
      return;

    }

    const rest = remaining.filter((m) => m.channelId !== nextPrimary.channelId);

    this.setState({

      channelId: nextPrimary.channelId,
      streamUrl: nextPrimary.streamUrl || streamUrl,
      isHls: nextPrimary.isHls,
      title: nextPrimary.name,
      subtitle: "",
      poster: nextPrimary.logo || poster,
      multiviewStreams: rest,

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

    const { kind, mediaId, title, poster, historyPoster, season, episode, episodeTitle, channelId, ready } = this.state;

    if (!ready) return;

    const completed = durationMs > 0 && positionMs / durationMs > 0.9;

    try {

      await Net.History.upsert({

        kind,
        mediaId: kind === "live" ? 0 : mediaId,

        title,
        poster: kind === "show" ? historyPoster || poster : poster,

        season,
        episode,
        episodeTitle: kind === "show" ? episodeTitle : undefined,

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

  handleSubtitlesEnabledChange = (enabled: boolean) => {

    this.setState({ subtitlesOn: enabled });

  };

  loadIntro = async (durationMs: number) => {

    const { kind, mediaId, season, episode, ready } = this.state;

    if (!ready || durationMs <= 0) return;

    try {

      const intro = kind === "movie" ? await Net.Stream.movieIntro(mediaId, durationMs) : kind === "show" ? await Net.Stream.episodeIntro(mediaId, season, episode, durationMs) : null;

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

    const { kind, mediaId } = this.state;

    const destination = kind === "movie" && mediaId
      ? `/movie/${mediaId}`
      : kind === "show" && mediaId
        ? `/show/${mediaId}`
        : "/";

    if (this.props.onMinimize) {

      this.props.onMinimize(destination);

    } else if (kind === "movie" && mediaId) {

      this.props.navigate(`/movie/${mediaId}`);

    } else if (kind === "show" && mediaId) {

      this.props.navigate(`/show/${mediaId}`);

    } else {

      this.props.navigate("/");

    }

  };

  render() {

    const { streamUrl, isHls, qualities, selectedHeight, subtitleTracks, title, subtitle, episodeTitle, description, poster, intro, nextEpisode, startPositionMs, loading, error, ready, seasons, menuEpisodes, menuSeason, menuEpisodesLoading, season, episode, kind, mediaId, channelId, settingsOpen, multiviewStreams, multiviewChannels, multiviewLoading, sourceProviders, selectedSourceKey, sourceSwitching } = this.state;
    const settings = Stores.Settings.settings;

    const streamResolving = loading;
    const fatalError = error && !streamResolving && !ready;

    if (fatalError) {

      return (

        <div className="relative flex h-full min-h-full flex-col items-center justify-center gap-5 bg-black px-6">

          <button onClick={this.handleBack}
            className="absolute left-4 top-[calc(env(safe-area-inset-top,0px)+1rem)] flex h-8 items-center gap-2 rounded-md border border-border-subtle bg-surface/80 px-3 text-xs text-foreground backdrop-blur-md transition-colors hover:bg-surface-overlay"
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

      <div className={this.props.minimized ? "relative overflow-hidden bg-black" : "relative h-full min-h-full overflow-hidden bg-black"}>

        <VideoPlayer

          key={kind === "live" ? `live-${channelId}` : `${kind}-${mediaId}-${season}-${episode}`}
          src={ready ? streamUrl : ""}

          isHls={isHls}

          live={this.state.kind === "live"}
          lowLatency={false}

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
          autoPlayNext={this.state.kind !== "live"}
          skipIntroEnabled={this.state.kind !== "live"}

          ambienceEnabled={settings?.ambienceEnabled ?? true}
          subtitlesEnabled={this.state.kind !== "live" && this.state.subtitlesOn}

          onBack={this.handleBack}
          onNextEpisode={this.handleNextEpisode}
          onSubtitlesEnabledChange={this.handleSubtitlesEnabledChange}
          onSeasonChange={kind === "show" ? this.loadMenuEpisodes : undefined}
          onProgress={this.state.kind === "live" ? undefined : this.saveProgress}
          onPlaybackStateChange={this.state.kind === "live" ? (playing) => {

            if (playing) this.startLiveActivity();
            else this.stopLiveActivity();

          } : undefined}
          onEpisodeSelect={kind === "show" ? this.handleEpisodeSelect : undefined}
          onDurationReady={this.state.kind === "live" ? undefined : this.loadIntro}
          onQualityChange={this.state.kind === "live" ? undefined : this.handleQualityChange}
          onOpenSettings={() => this.setState({ settingsOpen: true })}
          onPlaybackError={this.state.kind === "live" ? () => this.handleLiveFailover() : this.handlePlaybackError}
          onFatalError={this.handleFatalError}
          compact={this.props.minimized}
          onReturn={this.props.onReturn}
          onDismiss={this.props.onDismiss}

          startPositionMs={this.state.kind === "live" ? 0 : startPositionMs}

          seasons={kind === "show" ? seasons : undefined}
          episodes={kind === "show" ? menuEpisodes : undefined}

          currentSeason={kind === "show" ? season : undefined}
          currentEpisode={kind === "show" ? episode : undefined}

          menuSeason={kind === "show" ? menuSeason : undefined}

          episodesLoading={kind === "show" ? menuEpisodesLoading : undefined}

          primaryChannelId={kind === "live" ? channelId : undefined}
          multiviewStreams={kind === "live" ? multiviewStreams : undefined}
          multiviewChannels={kind === "live" ? multiviewChannels : undefined}
          multiviewLoading={kind === "live" ? multiviewLoading : undefined}
          onMultiviewSearch={kind === "live" ? this.handleMultiviewSearch : undefined}
          onMultiviewToggle={kind === "live" ? this.handleMultiviewToggle : undefined}
          onMultiviewRemove={kind === "live" ? this.handleMultiviewRemove : undefined}

          sourceProviders={kind === "live" ? sourceProviders : undefined}
          selectedSourceKey={kind === "live" ? selectedSourceKey : undefined}
          sourceSwitching={kind === "live" ? sourceSwitching : undefined}
          onSourceChange={kind === "live" ? (key) => void this.handleSourceChange(key) : undefined}

          streamResolving={streamResolving || sourceSwitching}

        />

        <SettingsPanel open={settingsOpen} onClose={() => this.setState({ settingsOpen: false })} />

      </div>

    );

  }

}
