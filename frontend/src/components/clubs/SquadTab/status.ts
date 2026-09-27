import type { Club, Fixture, Player } from '../../../types';
import { prettyCompetitionName } from '../PlayerSheet';

export function formatWageBill(squad: Player[]): string {
  const annual = squad.reduce((sum, p) => sum + (p.wage_eur || 0) * 52, 0);
  if (annual >= 1_000_000_000) return `€${(annual / 1_000_000_000).toFixed(2)}B`;
  return `€${(annual / 1_000_000).toFixed(1)}M`;
}

export const CLUB_PROFILE_TABS = ['overview', 'squad', 'fixtures', 'transfers', 'finances', 'history', 'recruitment', 'set-pieces'] as const;
export type ClubProfileTab = (typeof CLUB_PROFILE_TABS)[number];

/** Single fatigue cutoff shared by status counts, filters, and row styling. */
export const SQUAD_FATIGUE_FITNESS = 60;
/** Morale below this counts as unhappy in filters and row styling. */
export const SQUAD_UNHAPPY_MORALE = 55;

export function isSquadFatigued(player: Player): boolean {
  return typeof player.fitness === 'number' && player.fitness < SQUAD_FATIGUE_FITNESS;
}

export function isSquadInForm(player: Player): boolean {
  return (player.form ?? 0) >= 4 || player.form_band === 'Excellent';
}

export function isSquadUnhappy(player: Player): boolean {
  return typeof player.morale === 'number' && player.morale < SQUAD_UNHAPPY_MORALE;
}

export function isSquadExpiring(player: Player): boolean {
  return player.contract_years <= 1;
}

export function isSquadUnavailable(player: Player): boolean {
  return (player.injured_matches ?? 0) > 0 || (player.suspended_matches ?? 0) > 0;
}

export type SquadStatusCounts = Record<SquadStatusFilter, number>;

export function squadStatusCounts(squad: Player[]): SquadStatusCounts {
  return {
    all: squad.length,
    in_form: squad.filter(isSquadInForm).length,
    fatigued: squad.filter(isSquadFatigued).length,
    unhappy: squad.filter(isSquadUnhappy).length,
    expiring: squad.filter(isSquadExpiring).length,
    unavailable: squad.filter(isSquadUnavailable).length,
  };
}

export function passesSquadStatus(player: Player, filter: SquadStatusFilter): boolean {
  switch (filter) {
    case 'all':
      return true;
    case 'in_form':
      return isSquadInForm(player);
    case 'fatigued':
      return isSquadFatigued(player);
    case 'unhappy':
      return isSquadUnhappy(player);
    case 'expiring':
      return isSquadExpiring(player);
    case 'unavailable':
      return isSquadUnavailable(player);
    default:
      return true;
  }
}

export const FIXTURE_COMP_FILTERS = [
  { id: 'all', label: 'All' },
  { id: 'league', label: 'League' },
  { id: 'cup', label: 'Domestic Cup' },
  { id: 'champions-league', label: 'UCL' },
  { id: 'europa-league', label: 'UEL' },
  { id: 'conference-league', label: 'UECL' },
] as const;

const LEAGUE_COMP_IDS = new Set(['premier-league', 'la-liga', 'bundesliga', 'serie-a', 'ligue-1', 'super-league']);
const UEFA_COMP_IDS = new Set(['champions-league', 'ucl', 'europa-league', 'conference-league', 'super-cup']);

/** Fixture competition filter; cup = anything that is not a domestic league or UEFA tie. */
export function matchesFixtureFilter(fixture: Fixture, filter: string, league: string): boolean {
  const comp = (fixture.competition || '').toLowerCase();
  if (filter === 'all') return true;
  if (filter === 'champions-league') return comp === 'champions-league' || comp === 'ucl';
  if (filter === 'europa-league') return comp === 'europa-league';
  if (filter === 'conference-league') return comp === 'conference-league';
  if (filter === 'cup') return !LEAGUE_COMP_IDS.has(comp) && !UEFA_COMP_IDS.has(comp) && prettyCompetitionName(comp) !== league;
  if (filter === 'league') return LEAGUE_COMP_IDS.has(comp) || prettyCompetitionName(comp) === league;
  return true;
}

export interface SquadTabProps {
  clubs: Club[];
  initialClubId?: string;
  onWatchClub: (club: Club) => void;
  onWatchFixture?: (fixture: Fixture, intent?: 'watch' | 'visual' | 'quick' | 'result') => void;
}

export type SquadViewMode = 'pitch' | 'table' | 'split';
export type SquadStatusFilter = 'all' | 'in_form' | 'fatigued' | 'unhappy' | 'expiring' | 'unavailable';