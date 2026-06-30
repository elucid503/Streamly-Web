import { imageCache } from "@/lib/imageCache";
import { cn } from "@/lib/utils";

import { Component, type CSSProperties, type ReactNode } from "react";

interface CachedImageProps {

  src?: string;
  alt: string;

  className?: string;
  imgClassName?: string;
  style?: CSSProperties;
  lazy?: boolean;
  onVisible?: () => void;

  fallback?: ReactNode;

  rounded?: string;

}

interface CachedImageState {

  loaded: boolean;
  failed: boolean;
  visible: boolean;

}

export class CachedImage extends Component<CachedImageProps, CachedImageState> {

  state: CachedImageState = {

    loaded: false,
    failed: false,
    visible: false,

  };

  private root?: HTMLDivElement | null;
  private observer?: IntersectionObserver;

  componentDidMount() {

    this.prepareLoad();

  }

  componentDidUpdate(prev: CachedImageProps) {

    if (prev.src !== this.props.src || prev.lazy !== this.props.lazy) {

      this.disconnectObserver();
      this.setState({ loaded: false, failed: false, visible: false }, () => this.prepareLoad());

    }

  }

  componentWillUnmount() {

    this.disconnectObserver();

  }

  prepareLoad = () => {

    const { lazy, src } = this.props;

    if (!src) {

      this.setState({ failed: true });

      return;

    }

    if (!lazy || imageCache.isLoaded(src)) {

      this.beginLoad();

      return;

    }

    this.observeVisibility();

  };

  observeVisibility = () => {

    if (!this.root || !("IntersectionObserver" in window)) {

      this.beginLoad();

      return;

    }

    this.observer = new IntersectionObserver((entries) => {

      if (!entries.some((entry) => entry.isIntersecting)) {

        return;

      }

      this.disconnectObserver();
      this.beginLoad();

    }, { rootMargin: "240px" });

    this.observer.observe(this.root);

  };

  disconnectObserver = () => {

    this.observer?.disconnect();
    this.observer = undefined;

  };

  load = () => {

    const { src } = this.props;

    if (!src) {

      this.setState({ failed: true });

      return;

    }

    if (imageCache.isLoaded(src)) {

      this.setState({ loaded: true, failed: false });

      return;

    }

    imageCache.preload(src).then(() => this.setState({ loaded: true, failed: false })).catch(() => this.setState({ failed: true, loaded: false }));

  };

  beginLoad = () => {

    this.props.onVisible?.();
    this.setState({ visible: true }, () => this.load());

  };

  render() {

    const { src, alt, className, imgClassName, style, fallback, rounded = "rounded-md" } = this.props;
    const { loaded, failed, visible } = this.state;

    if (!src || failed) {

      return (

        <div className={cn("flex items-center justify-center bg-surface-overlay", rounded, className)} style={style} ref={(node) => { this.root = node; }}>

          {fallback}

        </div>

      );

    }

    return (

      <div className={cn("relative overflow-hidden bg-surface-overlay", rounded, className)} style={style} ref={(node) => { this.root = node; }}>

        <div className={cn(

            "skeleton absolute inset-0 transition-opacity duration-500",
            loaded ? "opacity-0" : "opacity-100"

          )}

        />

        {visible && (

          <img className={cn(

              "h-full w-full object-cover transition-opacity duration-500 ease-out",
              loaded ? "opacity-100" : "opacity-0",
              imgClassName

            )}

            src={src}
            alt={alt}

            decoding="async"
            loading="lazy"

          />

        )}

      </div>

    );

  }

}
