import { imageCache } from "@/lib/imageCache";
import { cn } from "@/lib/utils";

import { Component, type ReactNode } from "react";

interface CachedImageProps {

  src?: string
  alt: string;

  className?: string;
  imgClassName?: string;

  fallback?: ReactNode;

  rounded?: string;

}

interface CachedImageState {

  loaded: boolean;
  failed: boolean;

}

export class CachedImage extends Component<CachedImageProps, CachedImageState> {

  state: CachedImageState = {

    loaded: false,
    failed: false,

  };

  componentDidMount() {

    this.load();

  }

  componentDidUpdate(prev: CachedImageProps) {

    if (prev.src !== this.props.src) {

      this.setState({ loaded: false, failed: false }, () => this.load());

    }

  }

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

  render() {

    const { src, alt, className, imgClassName, fallback, rounded = "rounded-md" } = this.props;
    const { loaded, failed } = this.state;

    if (!src || failed) {

      return (

        <div className={cn("flex items-center justify-center bg-surface-overlay", rounded, className)}>

          {fallback}

        </div>

      );

    }

    return (

      <div className={cn("relative overflow-hidden bg-surface-overlay", rounded, className)}>

        <div className={cn(

            "skeleton absolute inset-0 transition-opacity duration-500",
            loaded ? "opacity-0" : "opacity-100"

          )}

        />

        <img className={cn(

            "h-full w-full object-cover transition-opacity duration-500 ease-out",
            loaded ? "opacity-100" : "opacity-0",
            imgClassName

          )}

          src={src}
          alt={alt}

          decoding="async"

        />

      </div>

    );

  }

}
