import { apiFetch, invalidateApiCache, API_BASE } from './core';
import type { Fixture, WorldDashboard } from '../../types';

export interface WorldSearchClub {
  club_id: string;
  club_name: string;
  short_name: string;
  league: string;
  ovr: number;
  pts: number;
  primary_color?: [number, number, number];
}

export interface WorldSearchPlayer {
  player_id: string;
  full_name: string;
  position: string;
  ovr: number;
  age: number;
  category?: string;
  club_id: string;
  club_name?: string;
  club_short?: string;
  goals: number;
  assists: number;
  is_wonderkid?: boolean;
}

export interface WorldSearchCompetition {
  id: string;
  name: string;
  kind: string;
  country: string;
  stage: string;
  participants?: number;
}

export interface SearchResponse {
  query: string;
  clubs: WorldSearchClub[];
  players: WorldSearchPlayer[];
  competitions: WorldSearchCompetition[];
}

export function fetchSearch(query = '', limit = 8): Promise<SearchResponse> {
  const params = new URLSearchParams();
  if (query) params.set('q', query);
  params.set('limit', String(limit));
  return apiFetch<SearchResponse>(`/search?${params.toString()}`, undefined, {
    query,
    clubs: [],
    players: [],
    competitions: [],
  });
}

export function fetchWorldDashboard(): Promise<WorldDashboard> {
  return apiFetch<WorldDashboard>('/world/dashboard', undefined, {
    world: false,
    season_name: '2026-27',
    season_phase: 'season',
    current_matchweek: 1,
    max_matchweeks: 38,
    league_leaders: [],
    europe: { stage: '', name: 'UEFA Champions League', competition_id: 'champions-league' },
    top_scorer: null,
    biggest_transfers: [],
    injuries: [],
    sackings: [],
    wonderkids: [],
    upcoming_fixtures: [],
    headlines: [],
    transfer_window: { open: false, type: 'CLOSED', week: 0, weeks: 0 },
    unread_inbox: 0,
    power_rankings: [],
    loan_watch: [],
  });
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

// Observational viewing preference only (neutral-viewer framing); consumed
// by the watchlist panel (Phase 3 F2).
export async function setFavourite(clubId: string): Promise<{ favourite_club_id: string }> {
  try {
    const res = await fetch(`${API_BASE}/favourite`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ club_id: clubId }),
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.detail || data.message || 'Could not save favourite club.');
    invalidateApiCache();
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