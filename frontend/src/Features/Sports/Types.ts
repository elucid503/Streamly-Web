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
