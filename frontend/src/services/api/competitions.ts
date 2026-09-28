import { apiFetch, invalidateApiCache, API_BASE } from './core';
import type { CompetitionDetail, CompetitionsResponse, Fixture, FixturesResponse, SuperLeagueState, UCLTournamentState, NationsCupResponse, BatchSimResult } from '../../types';


// --- Tournaments ------------------------------------------------------------

/** Domestic-league standings. Pass a league name ("La Liga") or competition
 *  ID ("la-liga") to select a table; omit for the default compatibility view. */
export function fetchSuperLeague(league?: string): Promise<SuperLeagueState> {
  const query = league ? `?league=${encodeURIComponent(league)}` : '';
  return apiFetch<SuperLeagueState>(`/super-league${query}`, undefined, {
    current_matchweek: 1,
    max_matchweeks: 44,
    season_phase: 'season',
    season_name: '2026-27',
    clubs: [],
    recent_results: [],
    world: false,
  });
}

export function fetchCompetitions(): Promise<CompetitionsResponse> {
  return apiFetch<CompetitionsResponse>('/competitions', undefined, { world: false, competitions: [] });
}

/** National-team competition data is separate from club fixtures and standings. */
export function fetchNationsCup(): Promise<NationsCupResponse> {
  return apiFetch<NationsCupResponse>('/competitions/nations-cup');
}


export function fetchNationsFixture(fixtureId: string): Promise<Fixture | null> {
  return apiFetch<Fixture | null>(`/competitions/nations-cup/fixtures/${encodeURIComponent(fixtureId)}`, undefined, null);
}

/** Plays one scheduled international fixture on demand and returns the full payload. */
export async function simulateNationsFixture(fixtureId: string): Promise<Fixture | null> {
  try {
    const res = await fetch(`${API_BASE}/competitions/nations-cup/fixtures/${encodeURIComponent(fixtureId)}/simulate`, { method: 'POST' });
    if (!res.ok) return null;
    const data = (await res.json()) as Fixture;
    invalidateApiCache();
    return data;
  } catch {
    return null;
  }
}

export function fetchCompetition(id: string): Promise<CompetitionDetail | null> {
  return apiFetch<CompetitionDetail>(`/competitions/${encodeURIComponent(id)}`);
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
  report_ready?: boolean;
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

/** Loads card-ready fixture rows without full previews or match reports. */
export function fetchFixtureSummaries(matchweek?: number): Promise<FixturesResponse> {
  const params = new URLSearchParams({ summary: '1' });
  if (matchweek != null) params.set('matchweek', String(matchweek));
  return apiFetch<FixturesResponse>(`/fixtures?${params.toString()}`, undefined, {
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
    invalidateApiCache();
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

const macroFallback = (mode: 'week' | 'month' | 'season' | 'continue'): BatchSimResult => ({
  status: 'error',
  mode,
  season_name: '2026-27',
  season_phase: 'season',
  current_matchweek: 1,
  digests: [],
  weeks_advanced: 0,
  played: 0,
  skipped: 0,
  season_finished: false,
  awards_ready: false,
  message: 'Macro simulation failed.',
});

export async function simulateWeek(): Promise<BatchSimResult> {
  try {
    return await apiFetch<BatchSimResult>('/sim/week', { method: 'POST' });
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : 'Macro simulation failed.';
    return { ...macroFallback('week'), message: msg };
  }
}

export async function simulateMonth(): Promise<BatchSimResult> {
  try {
    return await apiFetch<BatchSimResult>('/sim/month', { method: 'POST' });
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : 'Macro simulation failed.';
    return { ...macroFallback('month'), message: msg };
  }
}

export async function simulateSeason(): Promise<BatchSimResult> {
  try {
    return await apiFetch<BatchSimResult>('/sim/season', { method: 'POST' });
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : 'Macro simulation failed.';
    return { ...macroFallback('season'), message: msg };
  }
}

export async function simulateContinue(): Promise<BatchSimResult> {
  try {
    return await apiFetch<BatchSimResult>('/sim/continue', { method: 'POST' });
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : 'Continue failed.';
    return { ...macroFallback('continue'), message: msg };
  }
}
