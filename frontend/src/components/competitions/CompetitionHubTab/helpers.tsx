import { Flag, Globe2, Trophy } from 'lucide-react';
import type { Club, CompetitionClub, CompetitionFixtureRow, CompetitionKind } from '../../../types';

export interface CompetitionHubTabProps {
  /** Opens the selected fixture in the instant simulation centre. */
  onWatchFixture?: (fixture: CompetitionFixtureRow) => void;
  currentMatchweek?: number;
  onViewSquad?: (clubId: string) => void;
}

export function kindLabel(kind: CompetitionKind): string {
  if (kind === 'LEAGUE') return 'Domestic league';
  if (kind === 'DOMESTIC_CUP') return 'Domestic cup';
  if (kind === 'INTERNATIONAL') return 'National teams';
  return 'European';
}

export function kindTone(kind: CompetitionKind): string {
  if (kind === 'LEAGUE') return 'text-[#A9CBDD]';
  if (kind === 'DOMESTIC_CUP') return 'text-[#A9CDBB]';
  return 'text-brass';
}

export function KindIcon({ kind, size = 14 }: { kind: CompetitionKind; size?: number }) {
  if (kind === 'LEAGUE') return <Trophy size={size} aria-hidden="true" />;
  if (kind === 'DOMESTIC_CUP') return <Flag size={size} aria-hidden="true" />;
  return <Globe2 size={size} aria-hidden="true" />;
}

export function asClub(club: CompetitionClub | null | undefined): Club | null {
  if (!club) return null;
  return club as unknown as Club;
}

export function scoreLine(f: CompetitionFixtureRow): string {
  if (f.status !== 'finished' || f.home_goals == null || f.away_goals == null) return '–';
  const pens = f.decided_by === 'penalties' && f.penalties && f.penalties.length >= 2
    ? ` (${f.penalties[0]}–${f.penalties[1]} pens)`
    : f.decided_by === 'extra_time'
      ? ' AET'
      : '';
  return `${f.home_goals}–${f.away_goals}${pens}`;
}

export function legLabel(f: CompetitionFixtureRow): string {
  if (f.leg === 1) return 'Leg 1';
  if (f.leg === 2) return 'Leg 2 · decider';
  return '';
}

export const LEAGUE_BY_COMPETITION: Record<string, string> = {
  'premier-league': 'Premier League',
  'la-liga': 'La Liga',
  'bundesliga': 'Bundesliga',
  'serie-a': 'Serie A',
  'ligue-1': 'Ligue 1',
};

export const COUNTRY_ORDER = ['England', 'Spain', 'Germany', 'Italy', 'France', 'Europe'] as const;

export function isWatchable(fixture: CompetitionFixtureRow, currentMatchweek: number | undefined): boolean {
  return fixture.status === 'scheduled' && currentMatchweek != null && fixture.matchweek === currentMatchweek;
}

export function groupByMatchweek(rows: CompetitionFixtureRow[]): Array<[number, CompetitionFixtureRow[]]> {
  const out = new Map<number, CompetitionFixtureRow[]>();
  for (const f of rows) {
    const list = out.get(f.matchweek) ?? [];
    list.push(f);
    out.set(f.matchweek, list);
  }
  return [...out.entries()].sort((a, b) => a[0] - b[0]);
}
