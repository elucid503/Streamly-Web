export interface LiveChannel {

  id: string;
  name: string;
  slug: string;
  code: string;
  logo: string;

  country: string;
  countryName?: string;
  category: string;
  categories?: string[];
  network?: string;
  owners?: string[];
  website?: string;
  enriched?: boolean;

}

/** Anonymized live stream source option (no upstream brand/domain). */
export interface LiveSourceProvider {

  key: string;
  label: string;
  description?: string;

}

export interface ProgramEntry {

  title: string;
  episodeTitle?: string;
  summary?: string;
  startsAt: number; // Unix seconds
  runtime: number; // minutes
  image?: string;
  season?: number;
  episode?: number;
  genres?: string[];
  rating?: string;
  network?: string;

}

export interface ChannelGuideEntry {

  channel: LiveChannel;
  current?: ProgramEntry;
  next?: ProgramEntry;
  upcoming?: ProgramEntry[];

}
