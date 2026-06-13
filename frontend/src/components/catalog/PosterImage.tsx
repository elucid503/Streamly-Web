import { CachedImage } from "@/components/ui/CachedImage";

import { Component } from "react";
import { Film } from "lucide-react";

interface PosterImageProps {

  src?: string;

  alt: string;

  className?: string;

}

export class PosterImage extends Component<PosterImageProps> {

  render() {

    const { src, alt, className } = this.props;

    return (

      <CachedImage className={className}

        src={src}
        alt={alt}

        rounded="rounded-none"
        fallback={<Film size={24} strokeWidth={1.5} className="text-foreground-faint" />}

      />

    );

  }

}