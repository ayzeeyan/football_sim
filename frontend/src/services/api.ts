import type {
  AwardsCeremony,
  Club,
  CompletedTransfer,
  Fixture,
  FixturesResponse,
  GrowthMilestoneItem,
  InboxFeed,
  Player,
  PlayerProfile,
  ProdigyData,
  SeasonAwards,
  SuperLeagueState,
  TransferFeedItem,
  TransferNegotiation,
  UCLTournamentState,
  TrophyCabinetClub,
  AllTimeRecordsData,
  NXGNPlayer,
  ProdigyTimelineResponse,
  HeadToHeadData,
  ClubHistoryResponse,
  TransferRecordsData,
} from '../types';

const API_BASE = '/api';

async function apiFetch<T>(path: string, init?: RequestInit, fallback?: T): Promise<T> {
  try {
    const res = await fetch(`${API_BASE}${path}`, init);
    if (!res.ok) throw new Error(`HTTP ${res.status} for ${path}`);
    return (await res.json()) as T;
  } catch (err) {
    console.warn(`[API] ${path} failed`, err);
    if (fallback !== undefined) return fallback;
    throw err;
  }
}

// --- Clubs & squads ---------------------------------------------------------

export function fetchClubs(): Promise<Club[]> {
  return apiFetch<Club[]>('/clubs', undefined, []);
}

export function fetchClubSquad(clubId: string): Promise<Player[]> {
  return apiFetch<Player[]>(`/clubs/${clubId}/squad`, undefined, []);
}

export function fetchClubXi(clubId: string): Promise<Player[]> {
  return apiFetch<Player[]>(`/clubs/${clubId}/xi`, undefined, []);
}

export function fetchClubHistory(clubId: string): Promise<ClubHistoryResponse> {
  return apiFetch<ClubHistoryResponse>(`/clubs/${encodeURIComponent(clubId)}/history`, undefined, {
    club_id: clubId,
    club_name: clubId,
    short_name: clubId,
    primary_color: [200, 200, 200],
    history: [],
    trophies_summary: { super_league: 0, ucl: 0, super_cup: 0 },
  });
}

export function fetchHeadToHead(clubA: string, clubB: string): Promise<HeadToHeadData> {
  return apiFetch<HeadToHeadData>(`/h2h/${encodeURIComponent(clubA)}/${encodeURIComponent(clubB)}`, undefined, {
    club_a: { club_id: clubA, club_name: clubA, short_name: clubA, primary_color: [200, 200, 200] },
    club_b: { club_id: clubB, club_name: clubB, short_name: clubB, primary_color: [100, 100, 100] },
    matches_played: 0,
    wins_a: 0,
    wins_b: 0,
    draws: 0,
    goals_a: 0,
    goals_b: 0,
    derby_name: null,
    derby_heat: 50,
    recent_matches: [],
  });
}

// --- Prodigies / Wonderkids -------------------------------------------------

export function fetchProdigies(): Promise<ProdigyData[]> {
  return apiFetch<ProdigyData[]>('/prodigies', undefined, []);
}

export function fetchProdigyTimeline(playerId: string): Promise<ProdigyTimelineResponse> {
  return apiFetch<ProdigyTimelineResponse>(`/prodigies/${encodeURIComponent(playerId)}/timeline`, undefined, {
    player_id: playerId,
    full_name: playerId,
    age: 14,
    current_height_cm: 165,
    baseline_height_cm: 165,
    current_weight_kg: 55,
    baseline_weight_kg: 55,
    height_gain_cm: 0,
    weight_gain_kg: 0,
    ovr: 60,
    potential: 90,
    progression_history: [],
    milestones: [],
  });
}

export interface WonderkidsResponse {
  top_scorer: Player | null;
  top_assister: Player | null;
  top_ovr: Player | null;
  total_valuation: number;
  wonderkids: Player[];
}

export function fetchWonderkids(): Promise<WonderkidsResponse> {
  return apiFetch<WonderkidsResponse>('/wonderkids', undefined, {
    top_scorer: null,
    top_assister: null,
    top_ovr: null,
    total_valuation: 0,
    wonderkids: [],
  });
}

export interface TrainProdigyResponse {
  status: string;
  message: string;
  gains?: Record<string, unknown>;
  current_height?: number;
  current_weight?: number;
  height_gain?: number;
  weight_gain?: number;
  remaining_energy?: number;
  max_energy?: number;
}

export function trainProdigy(playerId: string, focus: string): Promise<TrainProdigyResponse> {
  return apiFetch<TrainProdigyResponse>(
    `/prodigies/${playerId}/train`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ focus }),
    },
    { status: 'error', message: 'Network error communicating with the incubator engine.' },
  );
}

export async function setProdigyPositionPath(playerId: string, position: string): Promise<{ status: string; message?: string; position_path?: string; position_xp?: number }> {  try {
    const res = await fetch(`${API_BASE}/prodigies/${encodeURIComponent(playerId)}/position-path`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ position }),
    });
    const data = await res.json();
    if (!res.ok) return { status: 'error', message: data.detail || 'Could not set position path.' };
    return data;
  } catch {
    return { status: 'error', message: 'Network error setting position path.' };
  }
}

export async function setProdigySchoolTrack(playerId: string, track: string): Promise<{ status: string; message?: string }> {
  try {
    const res = await fetch(`${API_BASE}/prodigies/${encodeURIComponent(playerId)}/school-track`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ track }),
    });
    const data = await res.json();
    if (!res.ok) return { status: 'error', message: data.detail || data.message || 'Could not set school track.' };
    return { status: 'success' };
  } catch {
    return { status: 'error', message: 'Network error setting school track.' };
  }
}

export function fetchGrowthMilestones(): Promise<GrowthMilestoneItem[]> {
  return apiFetch<GrowthMilestoneItem[]>('/growth/milestones', undefined, []);
}

export function fetchTrainingStatus(): Promise<{ training_energy: number; max_training_energy: number }> {
  return apiFetch('/training/status', undefined, { training_energy: 3, max_training_energy: 3 });
}

// --- Tournaments ------------------------------------------------------------

export function fetchSuperLeague(): Promise<SuperLeagueState> {
  return apiFetch<SuperLeagueState>('/super-league', undefined, {
    current_matchweek: 1,
    max_matchweeks: 44,
    season_phase: 'season',
    season_name: '2026-27',
    clubs: [],
    recent_results: [],
  });
}

export function fetchUCL(): Promise<UCLTournamentState> {
  return apiFetch<UCLTournamentState>('/ucl', undefined, {
    stage: 'GROUP_STAGE',
    group_a: [],
    group_b: [],
    semi_finals: {},
    quarter_finals: {},
    final: null,
    champion: null,
  });
}

export interface SimulateFixtureResponse {
  status: string;
  fixture_id?: string;
  home_goals?: number;
  away_goals?: number;
  ucl_event?: string | null;
  rolled_over?: boolean;
  is_finished?: boolean;
  champion?: string | null;
  message?: string;
}

export function fetchFixtures(matchweek?: number): Promise<FixturesResponse> {
  const q = matchweek ? `?matchweek=${matchweek}` : '';
  return apiFetch<FixturesResponse>(`/fixtures${q}`, undefined, {
    current_matchweek: 1,
    max_matchweeks: 44,
    season_phase: 'season',
    season_name: '2026-27',
    matchweek: matchweek ?? 1,
    fixtures: [],
    ucl_pending_ids: [],
  });
}

export async function simulateFixture(fixtureId: string): Promise<SimulateFixtureResponse> {
  try {
    const res = await fetch(`${API_BASE}/fixtures/${encodeURIComponent(fixtureId)}/simulate`, { method: 'POST' });
    const data = (await res.json()) as SimulateFixtureResponse;
    if (!res.ok) return { status: 'error', message: (data as { detail?: string }).detail || 'Could not simulate.' };
    return data;
  } catch {
    return { status: 'error', message: 'Network error simulating the fixture.' };
  }
}

export function simulateRemaining(excludeFixtureId?: string): Promise<{ status: string; played: number; is_finished: boolean; champion?: string | null; excluded_fixture_id?: string }> {
  return apiFetch(
    '/fixtures/simulate-remaining',
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(excludeFixtureId ? { exclude_fixture_id: excludeFixtureId } : {}),
    },
    { status: 'error', played: 0, is_finished: false },
  );
}

export interface WeekWatch {
  favourite_club_id: string;
  matchweek: number;
  fixture: Fixture | null;
  same_week_cups: Fixture[];
}

export function fetchFavourite(): Promise<{ favourite_club_id: string }> {
  return apiFetch<{ favourite_club_id: string }>('/favourite', undefined, { favourite_club_id: '' });
}

export async function setFavourite(clubId: string): Promise<{ favourite_club_id: string }> {
  try {
    const res = await fetch(`${API_BASE}/favourite`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ club_id: clubId }),
    });
    const data = await res.json();
    return { favourite_club_id: data.favourite_club_id ?? clubId };
  } catch {
    return { favourite_club_id: clubId };
  }
}

export function fetchWeekWatch(): Promise<WeekWatch> {
  return apiFetch<WeekWatch>('/week/watch', undefined, {
    favourite_club_id: '',
    matchweek: 1,
    fixture: null,
    same_week_cups: [],
  });
}

export function fixtureByClubs(fixtures: Fixture[], homeId: string, awayId: string): Fixture | undefined {
  return fixtures.find((f) => f.home.club_id === homeId && f.away.club_id === awayId);
}

export function fetchUclFixtures(): Promise<Fixture[]> {
  return apiFetch<Fixture[]>('/ucl/fixtures', undefined, []);
}

export function fetchFixture(fixtureId: string): Promise<Fixture | null> {
  return apiFetch<Fixture | null>(`/fixtures/${encodeURIComponent(fixtureId)}`, undefined, null);
}

export interface ProdigyDrawRow {
  full_name: string;
  position: string;
  age: number;
  club_id: string;
  club_name: string;
  short_name: string;
}

export interface ProdigyDraw {
  homes: Record<string, string>;
  draw: ProdigyDrawRow[];
  shuffle?: boolean;
  from_last?: boolean;
}

export function previewCareerShuffle(): Promise<ProdigyDraw> {
  return apiFetch<ProdigyDraw>('/career/preview-shuffle', undefined, { homes: {}, draw: [] });
}

export function fetchDefaultHomes(): Promise<ProdigyDraw> {
  return apiFetch<ProdigyDraw>('/career/default-homes', undefined, { homes: {}, draw: [] });
}

export async function startNewCareer(shuffle: boolean, homes?: Record<string, string>): Promise<{
  status: string;
  message: string;
  season_name?: string;
  current_matchweek?: number;
  max_matchweeks?: number;
  draw?: ProdigyDrawRow[];
}> {
  try {
    const res = await fetch(`${API_BASE}/career/new`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ shuffle, homes: homes ?? null }),
    });
    const data = await res.json();
    if (!res.ok) return { status: 'error', message: data.detail || 'Could not start a new career.' };
    return data;
  } catch {
    return { status: 'error', message: 'Network error starting a new career.' };
  }
}

export function resetSeason(): Promise<{ status: string; message: string; current_matchweek: number; max_matchweeks: number }> {
  return apiFetch(
    '/season/reset',
    { method: 'POST' },
    { status: 'error', message: 'Failed to reset season.', current_matchweek: 1, max_matchweeks: 44 },
  );
}

export function fetchAwardsCeremony(): Promise<AwardsCeremony | null> {
  return apiFetch<AwardsCeremony | null>('/season/awards/ceremony', undefined, null);
}

export function fetchPlayerProfile(playerId: string): Promise<PlayerProfile | null> {
  return apiFetch<PlayerProfile | null>(`/players/${encodeURIComponent(playerId)}`, undefined, null);
}

export function fetchInbox(limit = 80): Promise<InboxFeed> {
  return apiFetch<InboxFeed>(`/inbox?limit=${limit}`, undefined, {
    unread: 0,
    items: [],
    season_name: '2026-27',
    current_matchweek: 1,
  });
}

export async function markInboxRead(itemId?: string, all = false): Promise<{ unread: number; marked: number }> {
  try {
    const res = await fetch(`${API_BASE}/inbox/read`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ item_id: itemId ?? null, all }),
    });
    const data = await res.json();
    return { unread: data.unread ?? 0, marked: data.marked ?? 0 };
  } catch {
    return { unread: 0, marked: 0 };
  }
}

export function fetchSeasonAwards(): Promise<SeasonAwards> {
  return apiFetch<SeasonAwards>('/season/awards', undefined, {
    super_league_champion: null,
    super_league_runner_up: null,
    ucl_champion: null,
    top_scorer: null,
    top_assister: null,
    golden_boy: null,
    player_of_the_season: null,
  });
}

// --- Transfers --------------------------------------------------------------

export interface ExpiringContract {
  player_id: string;
  full_name: string;
  position: string;
  ovr: number;
  club_name: string;
  club_short: string;
  formatted_wage: string;
  loyalty: number;
  is_wonderkid: boolean;
}

export interface WarchestRow {
  club_name: string;
  club_short: string;
  manager_name: string;
  tactic: string;
  focus: string;
  budget_eur: number;
  formatted_budget: string;
  wage_bill_eur: number;
}

export interface TransfersResponse {
  window_name: string;
  is_window_open: boolean;
  season_phase: 'season' | 'transfer_window';
  window_day: number;
  window_week?: number;
  max_window_weeks?: number;
  active_negotiations: TransferNegotiation[];
  transfer_feed: TransferFeedItem[];
  completed_transfers: CompletedTransfer[];
  expiring_contracts: ExpiringContract[];
  warchests: WarchestRow[];
}

const EMPTY_TRANSFERS: TransfersResponse = {
  window_name: 'Summer Window',
  is_window_open: false,
  season_phase: 'season',
  window_day: 1,
  window_week: 1,
  max_window_weeks: 12,
  active_negotiations: [],
  transfer_feed: [],
  completed_transfers: [],
  expiring_contracts: [],
  warchests: [],
};

export function fetchTransfers(): Promise<TransfersResponse> {
  return apiFetch<TransfersResponse>('/transfers', undefined, EMPTY_TRANSFERS);
}

export function submitTransferBid(buyerId: string, sellerId: string, playerId: string): Promise<TransferNegotiation | null> {
  return apiFetch<TransferNegotiation | null>(
    '/transfers/bid',
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ buyer_id: buyerId, seller_id: sellerId, player_id: playerId }),
    },
    null,
  );
}

export async function advanceMarket(): Promise<TransfersResponse> {
  try {
    const res = await fetch(`${API_BASE}/transfers/advance`, { method: 'POST' });
    const data = (await res.json()) as TransfersResponse & { detail?: string };
    if (!res.ok) return { ...EMPTY_TRANSFERS, window_name: data.detail || 'The window opens when the season ends.' };
    return data;
  } catch {
    return EMPTY_TRANSFERS;
  }
}

export function fetchTransferRecords(): Promise<TransferRecordsData> {
  return apiFetch<TransferRecordsData>('/transfers/records', undefined, {
    top_signings: [],
    net_spend: {},
    total_transfers_count: 0,
  });
}

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
  champion: { club_name: string; short_name: string; pts: number } | null;
  runner_up?: { club_name: string; short_name: string; pts: number } | null;
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
  chapter?: string;
  league: number;
  ucl: number;
  super_cup: number;
  current: boolean;
}

export interface CalendarState {
  current_matchweek: number;
  max_matchweeks: number;
  phase: string;
  month: string;
  chapter?: string;
  this_week?: number;
  next_cup_night?: number | null;
  weeks: CalendarWeek[];
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
