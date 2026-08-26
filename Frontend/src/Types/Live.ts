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

export interface MatchedChannel {

  id: string;
  name: string;
  logo: string;

}

export interface SportsAlert {

  matchId: string;
  title: string;

}

export interface SportsTeamAlert {

  team: string;
  logo?: string;

}

export interface SportsAlertsList {

  matches: SportsAlert[];
  teams: SportsTeamAlert[];

}

export interface SportsMatch {

  id: string;
  title: string;
  category: string;
  league?: string;

  homeTeam?: string;
  awayTeam?: string;
  homeShortName?: string;
  awayShortName?: string;
  homeLogo?: string;
  awayLogo?: string;

  homeScore?: number;
  awayScore?: number;
  statusDetail?: string;
  /** Scoreboard lifecycle when known: pre / in / post. */
  status?: "pre" | "in" | "post" | string;
  delayed?: boolean;

  startsAt: number;
  live: boolean;

  /** Primary live TV/stream outlet from scoreboard data (e.g. "SNY"). */
  broadcast?: string;
  broadcasts?: string[];

  channel?: MatchedChannel;

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
