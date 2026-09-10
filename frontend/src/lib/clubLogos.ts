import type { Club } from '../types';

/**
 * Official club crests from luukhopman/football-logos (raw GitHub PNGs).
 * Every path below was verified against the repo tree via the GitHub API —
 * note the exact folder/file spellings (`Spain - LaLiga`, `Bayern Munich`,
 * `Olympique Marseille`). Served with a bone disc behind them in the UI.
 * Any club without an entry falls back to the initials crest.
 */
const G = 'https://raw.githubusercontent.com/luukhopman/football-logos/master/logos';

export const CLUB_CREST_URLS: Record<string, string> = {
  // Elite 12 — Super League + Champions Cup
  'LAL-BAR': `${G}/Spain%20-%20LaLiga/FC%20Barcelona.png`,
  'LAL-RMA': `${G}/Spain%20-%20LaLiga/Real%20Madrid.png`,
  'LAL-ATM': `${G}/Spain%20-%20LaLiga/Atl%C3%A9tico%20de%20Madrid.png`,
  'EPL-ARS': `${G}/England%20-%20Premier%20League/Arsenal%20FC.png`,
  'EPL-LIV': `${G}/England%20-%20Premier%20League/Liverpool%20FC.png`,
  'BUN-BAY': `${G}/Germany%20-%20Bundesliga/Bayern%20Munich.png`,
  'BUN-DOR': `${G}/Germany%20-%20Bundesliga/Borussia%20Dortmund.png`,
  'SEA-INT': `${G}/Italy%20-%20Serie%20A/Inter%20Milan.png`,
  'SEA-NAP': `${G}/Italy%20-%20Serie%20A/SSC%20Napoli.png`,
  'SEA-MIL': `${G}/Italy%20-%20Serie%20A/AC%20Milan.png`,
  'FL1-PSG': `${G}/France%20-%20Ligue%201/Paris%20Saint-Germain.png`,
  'EPL-TOT': `${G}/England%20-%20Premier%20League/Tottenham%20Hotspur.png`,
};

export function getClubCrestUrl(club: Pick<Club, 'club_id'> | null | undefined): string | null {
  if (!club) return null;
  return CLUB_CREST_URLS[club.club_id] ?? null;
}

/** Short-name lookup for payloads that only carry `short_name` (awards, ledgers). */
const CREST_BY_SHORT: Record<string, string> = {
  BAR: CLUB_CREST_URLS['LAL-BAR'],
  RMA: CLUB_CREST_URLS['LAL-RMA'],
  ATM: CLUB_CREST_URLS['LAL-ATM'],
  ARS: CLUB_CREST_URLS['EPL-ARS'],
  LIV: CLUB_CREST_URLS['EPL-LIV'],
  BAY: CLUB_CREST_URLS['BUN-BAY'],
  BVB: CLUB_CREST_URLS['BUN-DOR'],
  INT: CLUB_CREST_URLS['SEA-INT'],
  NAP: CLUB_CREST_URLS['SEA-NAP'],
  MIL: CLUB_CREST_URLS['SEA-MIL'],
  PSG: CLUB_CREST_URLS['FL1-PSG'],
  TOT: CLUB_CREST_URLS['EPL-TOT'],
};

export function getClubCrestUrlByShort(shortName: string | null | undefined): string | null {
  if (!shortName) return null;
  return CREST_BY_SHORT[shortName.toUpperCase()] ?? null;
}
