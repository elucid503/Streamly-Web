import { Component } from "react";
import { Film } from "lucide-react";
import { CachedImage } from "@/components/ui/CachedImage";

interface PosterImageProps {
  src?: string;
  alt: string;
  className?: string;
}

export class PosterImage extends Component<PosterImageProps> {
  render() {
    const { src, alt, className } = this.props;
    return (
      <CachedImage
        src={src}
        alt={alt}
        className={className}
        rounded="rounded-none"
        fallback={<Film size={24} strokeWidth={1.5} className="text-foreground-faint" />}
      />
    );
  }
}