import { apiFetch, invalidateApiCache, API_BASE } from './core';
import type { AwardsCeremony, InboxFeed, PlayerProfile, SeasonAwards } from '../../types';


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
    invalidateApiCache();
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

/** Restart the current season from matchweek 1 (all-time records kept). */
export function restartSeason(): Promise<{ status: string; message: string; current_matchweek: number; max_matchweeks: number }> {
  return apiFetch(
    '/season/restart',
    { method: 'POST' },
    { status: 'error', message: 'Failed to restart season.', current_matchweek: 1, max_matchweeks: 38 },
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

export async function replyInbox(itemId: string, choiceId: string): Promise<{ status: string; message?: string }> {
  try {
    const res = await fetch(`${API_BASE}/inbox/reply`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ item_id: itemId, choice_id: choiceId }),
    });
    const data = await res.json();
    if (!res.ok) return { status: 'error', message: data.detail || data.message || 'Could not reply.' };
    invalidateApiCache();
    return { status: data.status ?? 'success', message: data.message };
  } catch {
    return { status: 'error', message: 'Network error sending the reply.' };
  }
}

export async function markInboxRead(itemId?: string, all = false): Promise<{ unread: number; marked: number }> {
  // Network failures throw: callers must not clear local unread state when
  // the server never confirmed the read.
  const res = await fetch(`${API_BASE}/inbox/read`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ item_id: itemId ?? null, all }),
  });
  const data = await res.json();
  if (!res.ok) throw new Error(data.detail || data.message || 'Could not mark inbox messages as read.');
  invalidateApiCache();
  return { unread: data.unread ?? 0, marked: data.marked ?? 0 };
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