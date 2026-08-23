import { Component, createRef } from "react";
import type HLS from "hls.js";
import { Volume2, VolumeX, X } from "lucide-react";

import { isProxiedStream, isWebPlayableUrl } from "@/Utils/Player/StreamClient";
import { cn } from "@/Utils/ClassNames";

export interface MultiviewStream {

  channelId: string;
  name: string;
  streamUrl: string;
  isHls: boolean;
  logo?: string;
  pending?: boolean;
  error?: boolean;

}

interface LiveStreamPaneProps {

  stream: MultiviewStream;
  audioActive: boolean;
  volume: number;
  showLabel?: boolean;
  removable?: boolean;

  onSelectAudio: (channelId: string) => void;
  onRemove?: (channelId: string) => void;

}

interface LiveStreamPaneState {

  loading: boolean;
  error: boolean;

}

const isPendingStream = (stream: MultiviewStream) => !!stream.pending || !stream.streamUrl.trim();

export class LiveStreamPane extends Component<LiveStreamPaneProps, LiveStreamPaneState> {

  private videoRef = createRef<HTMLVideoElement>();
  private hls: HLS | null = null;
  private sourceGen = 0;

  state: LiveStreamPaneState = {

    loading: true,
    error: false,

  };

  componentDidMount() {

    if (isPendingStream(this.props.stream)) {

      this.setState({ loading: true, error: !!this.props.stream.error });
      return;

    }

    void this.attachSource();

  }

  componentDidUpdate(prev: LiveStreamPaneProps) {

    const prevPending = isPendingStream(prev.stream);
    const nextPending = isPendingStream(this.props.stream);

    if (nextPending) {

      if (!prevPending || prev.stream.error !== this.props.stream.error) {

        this.destroyHls();
        this.setState({ loading: !this.props.stream.error, error: !!this.props.stream.error });

      }

      return;

    }

    if (
      prevPending
      || prev.stream.streamUrl !== this.props.stream.streamUrl
      || prev.stream.isHls !== this.props.stream.isHls
    ) {

      void this.attachSource();
      return;

    }

    this.syncAudio();

  }

  componentWillUnmount() {

    this.destroyHls();

    const video = this.videoRef.current;

    if (video) {

      video.pause();
      video.removeAttribute("src");
      video.load();

    }

  }

  destroyHls = () => {

    if (!this.hls) return;

    this.hls.destroy();
    this.hls = null;

  };

  syncAudio = () => {

    const video = this.videoRef.current;

    if (!video) return;

    const { audioActive, volume } = this.props;

    video.muted = !audioActive;
    video.volume = audioActive ? volume : 0;

  };

  attachSource = async () => {

    const video = this.videoRef.current;
    const { stream } = this.props;
    const gen = ++this.sourceGen;

    if (!video || isPendingStream(stream) || !isWebPlayableUrl(stream.streamUrl)) {

      this.setState({ loading: false, error: true });
      return;

    }

    this.destroyHls();

    this.setState({ loading: true, error: false });

    video.pause();
    video.removeAttribute("src");

    const onReady = () => {

      if (gen !== this.sourceGen) return;

      this.syncAudio();
      video.play().catch(() => undefined);
      this.setState({ loading: false });

    };

    if (stream.isHls) {

      const { default: HlsConstructor } = await import("hls.js");

      if (gen !== this.sourceGen || this.videoRef.current !== video) return;

      if (!HlsConstructor.isSupported()) {

        if (video.canPlayType("application/vnd.apple.mpegurl")) {

          video.src = stream.streamUrl;
          video.addEventListener("loadedmetadata", onReady, { once: true });
          return;

        }

        this.setState({ loading: false, error: true });
        return;

      }

      const proxied = isProxiedStream(stream.streamUrl);

      this.hls = new HlsConstructor({

        enableWorker: true,
        lowLatencyMode: false,
        maxBufferLength: 30,
        maxMaxBufferLength: 60,
        backBufferLength: 0,
        liveSyncDurationCount: 3,
        xhrSetup: proxied ? (xhr) => { xhr.withCredentials = true; } : undefined,

      });

      this.hls.loadSource(stream.streamUrl);
      this.hls.attachMedia(video);

      this.hls.on(HlsConstructor.Events.MANIFEST_PARSED, onReady);

      this.hls.on(HlsConstructor.Events.ERROR, (_event, data) => {

        if (!data.fatal || gen !== this.sourceGen) return;

        this.setState({ loading: false, error: true });

      });

      return;

    }

    video.src = stream.streamUrl;
    video.addEventListener("loadedmetadata", onReady, { once: true });
    video.addEventListener("error", () => {

      if (gen !== this.sourceGen) return;

      this.setState({ loading: false, error: true });

    }, { once: true });

  };

  render() {

    const { stream, audioActive, showLabel = true, removable, onSelectAudio, onRemove } = this.props;
    const { loading, error } = this.state;

    return (

      <div className="group relative flex h-full min-h-0 min-w-0 items-center justify-center overflow-hidden bg-black">

        <video

          ref={this.videoRef}
          className="h-full w-full object-contain object-center"
          playsInline
          muted={!audioActive}

        />

        {(loading || stream.pending) && !error && !stream.error && (

          <div className="absolute inset-0 z-10 flex items-center justify-center bg-black/50">

            <div className="h-6 w-6 animate-spin rounded-full border-2 border-white/20 border-t-white" />

          </div>

        )}

        {(error || stream.error) && !loading && !stream.pending && (

          <div className="absolute inset-0 z-10 flex items-center justify-center bg-black/70 px-3 text-center text-xs text-foreground-muted">

            Unable to load {stream.name}

          </div>

        )}

        <div className="absolute top-2 left-1/2 z-40 flex -translate-x-1/2 items-center gap-1.5">

          {showLabel && (

            <div className="pointer-events-none max-w-[10rem] truncate rounded-md border border-border-subtle bg-surface/80 px-2 py-1 text-[11px] font-medium text-foreground backdrop-blur-md sm:max-w-[14rem]">

              {stream.name}

            </div>

          )}

          <button

            type="button"
            onClick={(e) => {

              e.stopPropagation();
              onSelectAudio(stream.channelId);

            }}
            className={cn(

              "flex size-8 shrink-0 items-center justify-center rounded-md backdrop-blur-md transition-colors",
              audioActive
                ? "bg-accent text-black"
                : "border border-border-subtle bg-surface/80 text-foreground/90 hover:bg-surface-overlay hover:text-foreground"

            )}
            aria-label={audioActive ? `Audio from ${stream.name}` : `Route audio to ${stream.name}`}

          >

            {audioActive ? <Volume2 size={14} /> : <VolumeX size={14} />}

          </button>

          {removable && onRemove && (

            <button

              type="button"
              onClick={(e) => {

                e.stopPropagation();
                onRemove(stream.channelId);

              }}
              className="flex size-8 shrink-0 items-center justify-center rounded-md border border-border-subtle bg-surface/80 text-foreground/90 backdrop-blur-md transition-colors hover:bg-surface-overlay hover:text-foreground"
              aria-label={`Remove ${stream.name}`}

            >

              <X size={14} />

            </button>

          )}

        </div>

      </div>

    );

  }

}

export function multiviewLayout(count: number): {

  className: string;
  spanFirst?: boolean;
  fiveCol?: boolean;

} {

  switch (count) {

    case 2:

      return { className: "grid-cols-2 grid-rows-1" };

    case 3:

      return { className: "grid-cols-2 grid-rows-2", spanFirst: true };

    case 4:

      return { className: "grid-cols-2 grid-rows-2" };

    case 5:

      return { className: "grid-cols-6 grid-rows-2", fiveCol: true };

    case 6:

      return { className: "grid-cols-3 grid-rows-2" };

    default:

      return { className: "grid-cols-1 grid-rows-1" };

  }

}

export function paneSpanClass(index: number, layout: ReturnType<typeof multiviewLayout>): string {

  if (layout.spanFirst && index === 0) return "col-span-2";

  if (layout.fiveCol) {

    // Top row: two panes spanning 3 cols each; bottom: three spanning 2.
    if (index < 2) return "col-span-3";

    return "col-span-2";

  }

  return "";

}
