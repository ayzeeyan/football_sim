import { apiFetch } from './core';
import type { Club, SeasonAwards, TrophyCabinetClub, AllTimeRecordsData, NXGNPlayer } from '../../types';


export interface ScoringRaceRow {
  player_id?: string;
  full_name: string;
  position: string;
  ovr: number;
  goals: number;
  assists: number;
  club_name: string;
  club_short: string;
  is_wonderkid: boolean;
}

export function fetchScoringRace(): Promise<ScoringRaceRow[]> {
  return apiFetch<ScoringRaceRow[]>('/scoring-race', undefined, []);
}

export interface SeasonStatsRow extends ScoringRaceRow {
  player_id?: string;
  appearances?: number;
}

export interface PlayerOfTheWeek {
  player_id?: string;
  full_name: string;
  position: string;
  ovr: number;
  rating: number;
  goals: number;
  assists: number;
  club_name: string;
  club_short: string;
  is_wonderkid: boolean;
  matchweek: number;
}

export interface SeasonHistoryRow {
  season_name: string;
  world?: boolean;
  champion: { club_name: string; short_name: string; pts: number } | null;
  runner_up?: { club_name: string; short_name: string; pts: number } | null;
  league_champions?: Array<{ club_name: string; short_name: string; pts?: number; league?: string; competition_id?: string }>;
  europa_champion?: { club_name: string; short_name: string } | null;
  conference_champion?: { club_name: string; short_name: string } | null;
  ucl_champion: { club_name: string; short_name: string } | null;
  top_scorer: { full_name: string; goals: number; club_id?: string } | null;
  top_assister?: { full_name: string; assists: number; club_id?: string } | null;
  golden_boy: { full_name: string; ovr: number; goals?: number; assists?: number } | null;
  player_of_the_season?: {
    full_name: string;
    club_name?: string;
    short_name?: string;
    goals?: number;
    assists?: number;
    ovr?: number;
  } | null;
  super_cup_champion?: { club_name: string; short_name: string } | null;
  recap?: string;
  monthly_awards?: MonthlyAward[];
  table?: Array<{ club_name: string; short_name: string; pts: number; gd?: number; w?: number; d?: number; l?: number }>;
}

export interface CareerHistory {
  season_name: string;
  current_matchweek: number;
  max_matchweeks: number;
  season_phase: string;
  ucl_stage: string;
  super_cup_stage?: string;
  current: SeasonAwards | null;
  table: Array<{ club_name: string; short_name: string; pts: number; gd: number; p: number }>;
  past: SeasonHistoryRow[];
  trophy_cabinet?: TrophyCabinetClub[];
  all_time_records?: AllTimeRecordsData;
  recent_results?: HistoryResultRow[];
}

export interface HistoryResultRow {
  id: string;
  fixture_id?: string;
  matchweek: number;
  competition: string;
  stage?: string;
  status: string;
  home_id: string;
  away_id: string;
  home: { club_id: string; club_name: string; short_name: string };
  away: { club_id: string; club_name: string; short_name: string };
  home_goals: number;
  away_goals: number;
}

export function fetchCareerHistory(): Promise<CareerHistory> {
  return apiFetch<CareerHistory>('/season/history', undefined, {
    season_name: '2026-27',
    current_matchweek: 1,
    max_matchweeks: 44,
    season_phase: 'season',
    ucl_stage: 'GROUP_STAGE',
    current: null,
    table: [],
    past: [],
  });
}

export function fetchTrophies(): Promise<{ status: string; cabinet: TrophyCabinetClub[] }> {
  return apiFetch<{ status: string; cabinet: TrophyCabinetClub[] }>('/trophies', undefined, {
    status: 'fallback',
    cabinet: [],
  });
}

export function fetchRecords(): Promise<{ status: string; records: AllTimeRecordsData }> {
  return apiFetch<{ status: string; records: AllTimeRecordsData }>('/records', undefined, {
    status: 'fallback',
    records: {
      top_goalscorers: [],
      top_assisters: [],
      highest_scoring_match: { score: '—', total_goals: 0 },
      biggest_margin_victory: { score: '—', margin: 0 },
      single_match_goals_record: { player_name: '—', club_name: '—', goals: 0, fixture: '—' },
      highest_season_points: { club_name: '—', season_name: '—', points: 0 },
      wonderkid_milestones: {
        highest_ovr: { name: '—', ovr: 0, potential: 0 },
        top_prodigy_goals: { name: '—', goals: 0 },
      },
    },
  });
}

export function fetchNXGN50(): Promise<{ status: string; rankings: NXGNPlayer[] }> {
  return apiFetch<{ status: string; rankings: NXGNPlayer[] }>('/nxgn50', undefined, {
    status: 'fallback',
    rankings: [],
  });
}

export interface MonthlyAward {
  month: string;
  through_mw: number;
  player_id?: string;
  full_name: string;
  position: string;
  ovr: number;
  rating: number;
  apps: number;
  goals: number;
  club_name: string;
  club_short: string;
  is_wonderkid: boolean;
}

export interface SeasonStats {
  scorers: SeasonStatsRow[];
  assisters: SeasonStatsRow[];
  player_of_the_week: PlayerOfTheWeek | null;
  monthly_awards?: MonthlyAward[];
  history: SeasonHistoryRow[];
}

export function fetchSeasonStats(): Promise<SeasonStats> {
  return apiFetch<SeasonStats>('/season/stats', undefined, {
    scorers: [],
    assisters: [],
    player_of_the_week: null,
    monthly_awards: [],
    history: [],
  });
}

export interface CalendarWeek {
  matchweek: number;
  phase: string;
  month: string;
  year?: number;
  chapter?: string;
  league: number;
  ucl: number;
  super_cup: number;
  current: boolean;
}

export interface CalendarState {
  current_matchweek: number;
  max_matchweeks: number;
  season_name?: string;
  phase: string;
  month: string;
  year?: number;
  chapter?: string;
  this_week?: number;
  next_cup_night?: number | null;
  weeks: CalendarWeek[];
  world?: boolean;
}

export function fetchCalendar(): Promise<CalendarState> {
  return apiFetch<CalendarState>('/calendar', undefined, {
    current_matchweek: 1,
    max_matchweeks: 44,
    phase: 'Opening series',
    month: 'August',
    weeks: [],
  });
}

export interface SuperCupTie {
  home: Club | null;
  away: Club | null;
  leg1: [number, number] | null;
  winner: Club | null;
  decided_by?: string | null;
  penalties?: [number, number] | null;
}

export interface SuperCupState {
  stage: string;
  byes: Club[];
  play_in: Record<string, SuperCupTie>;
  quarter_finals: Record<string, SuperCupTie>;
  semi_finals: Record<string, SuperCupTie>;
  final: {
    team1: Club | null;
    team2: Club | null;
    score: [number, number] | null;
    winner: Club | null;
    decided_by?: string | null;
    penalties?: [number, number] | null;
  } | null;
  champion: Club | null;
}

export function fetchSuperCup(): Promise<SuperCupState> {
  return apiFetch<SuperCupState>('/super-cup', undefined, {
    stage: 'PLAY_IN',
    byes: [],
    play_in: {},
    quarter_finals: {},
    semi_finals: {},
    final: null,
    champion: null,
  });
}

export interface AdvancedPlayerStatRow {
  player_id: string;
  full_name: string;
  position: string;
  category: string;
  club_id: string;
  club_short: string;
  ovr: number;
  age: number;
  appearances: number;
  minutes: number;
  goals: number;
  assists: number;
  avg_rating: number;
  shots: number;
  xg: number;
  shots_box: number;
  shots_outside: number;
  xg_box: number;
  xg_outside: number;
  percentile: number;
}

export interface AdvancedClubStatRow {
  club_id: string;
  club_name: string;
  short_name: string;
  matches: number;
  avg_possession: number;
  avg_pass_accuracy: number;
  avg_shots: number;
  avg_xg: number;
  avg_xg_against: number;
  territory_defensive: number;
  territory_midfield: number;
  territory_attacking: number;
}

export interface AdvancedSeasonStats {
  season_name: string;
  players: AdvancedPlayerStatRow[];
  clubs: AdvancedClubStatRow[];
  shot_detail_note: string;
}

/** Statistics centre aggregation (Phase 3 F3). */
export function fetchAdvancedSeasonStats(): Promise<AdvancedSeasonStats> {
  return apiFetch<AdvancedSeasonStats>('/season/stats/advanced', undefined, {
    season_name: '',
    players: [],
    clubs: [],
    shot_detail_note: '',
  });
}
