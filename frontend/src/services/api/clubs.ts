import { apiFetch, invalidateApiCache } from './core';
import type { Club, Fixture, Player, HeadToHeadData, ClubHistoryResponse, ClubProfile, ClubTransferActivity, ClubScoutingResponse, ClubSetPiecesResponse, ClubMedicalResponse, LineupOverride } from '../../types';


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

export function fetchClubProfile(clubId: string): Promise<ClubProfile | null> {
  return apiFetch<ClubProfile | null>(`/clubs/${encodeURIComponent(clubId)}/profile`, undefined, null);
}

export function fetchClubFixtures(clubId: string): Promise<Fixture[]> {
  return apiFetch<{ fixtures?: Fixture[] }>(`/clubs/${encodeURIComponent(clubId)}/fixtures`, undefined, { fixtures: [] }).then(
    (data) => data.fixtures ?? [],
  );
}

export function fetchClubTransfers(clubId: string): Promise<ClubTransferActivity> {
  return apiFetch<ClubTransferActivity>(`/clubs/${encodeURIComponent(clubId)}/transfers`, undefined, {
    arrivals: [],
    departures: [],
    loans_in: [],
    loans_out: [],
    spent: 0,
    received: 0,
    net_spend: 0,
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

export function fetchClubScouting(clubId: string, limit = 12): Promise<ClubScoutingResponse | null> {
  return apiFetch<ClubScoutingResponse | null>(
    `/clubs/${encodeURIComponent(clubId)}/scouting?limit=${limit}`,
    undefined,
    null,
  );
}



export function fetchClubSetPieces(clubId: string): Promise<ClubSetPiecesResponse | null> {
  return apiFetch<ClubSetPiecesResponse | null>(
    `/clubs/${encodeURIComponent(clubId)}/set-pieces`,
    undefined,
    null,
  );
}


export function fetchClubMedical(clubId: string): Promise<ClubMedicalResponse | null> {
  return apiFetch<ClubMedicalResponse | null>(
    `/clubs/${encodeURIComponent(clubId)}/medical`,
    undefined,
    null,
  );
}


export async function setClubLineup(
  clubId: string,
  formation: string,
  players: Record<string, string>,
): Promise<{ status: string; message?: string; override?: LineupOverride }> {
  try {
    const res = await fetch(`/api/clubs/${encodeURIComponent(clubId)}/lineup`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ formation, players }),
    });
    const data = await res.json();
    if (!res.ok) return { status: 'error', message: data.detail || 'The lineup was rejected.' };
    invalidateApiCache();
    return data;
  } catch {
    return { status: 'error', message: 'Network error saving the lineup.' };
  }
}

export async function clearClubLineup(clubId: string): Promise<{ status: string }> {
  try {
    const res = await fetch(`/api/clubs/${encodeURIComponent(clubId)}/lineup`, { method: 'DELETE' });
    invalidateApiCache();
    return { status: res.ok ? 'success' : 'error' };
  } catch {
    return { status: 'error' };
  }
}

