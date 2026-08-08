import { CachedImage } from "@/components/ui/CachedImage";

import { getLogoBackdrop, type LogoBackdrop } from "@/lib/logoBackdrop";
import { channelColor, channelInitial } from "@/lib/channelColor";
import type { LiveChannel } from "@/lib/types";

import { Component } from "react";

interface LiveLogoProps {

  channel: LiveChannel;

  className?: string;
  imgClassName?: string;
  rounded?: string;
  lazy?: boolean;

}

interface LiveLogoState {

  backdrop?: LogoBackdrop;

}

export class LiveLogo extends Component<LiveLogoProps, LiveLogoState> {

  state: LiveLogoState = {};

  private mounted = false;
  private visible = false;

  componentDidMount() {

    this.mounted = true;

  }

  componentDidUpdate(prev: LiveLogoProps) {

    if (prev.channel.logo !== this.props.channel.logo || prev.channel.enriched !== this.props.channel.enriched) {

      this.setState({ backdrop: undefined }, () => {

        if (this.visible) {

          this.loadBackdrop();

        }

      });

    }

  }

  componentWillUnmount() {

    this.mounted = false;

  }

  loadBackdrop = async () => {

    const src = this.props.channel.logo;

    if (!this.shouldUseBackdrop() || !src) {

      return;

    }

    const backdrop = await getLogoBackdrop(src);

    if (!this.mounted || this.props.channel.logo !== src) {

      return;

    }

    this.setState({ backdrop });

  };

  render() {

    const { channel, className, imgClassName, rounded, lazy = true } = this.props;

    const useBackdrop = this.shouldUseBackdrop();
    const backgroundColor = useBackdrop ? this.state.backdrop?.backgroundColor ?? "#ffffff" : undefined;

    return (

      <CachedImage

        className={className}
        src={channel.logo}
        alt={channel.name}
        imgClassName={imgClassName}
        rounded={rounded}
        fallback={<ChannelInitialFallback name={channel.name} />}
        lazy={lazy}
        onVisible={this.handleVisible}
        style={backgroundColor ? { backgroundColor } : undefined}

      />

    );

  }

  handleVisible = () => {

    this.visible = true;
    this.loadBackdrop();

  };

  shouldUseBackdrop = () => {

    const { channel } = this.props;

    return Boolean(channel.enriched && channel.logo);

  };

}

function ChannelInitialFallback({ name }: { name: string }) {

  return (

    <span className="flex size-full items-center justify-center text-sm font-semibold text-white"

      style={{ backgroundColor: channelColor(name) }}

    >

      {channelInitial(name)}

    </span>

  );

}
