import { apiFetch, invalidateApiCache, API_BASE } from './core';
import type { GrowthMilestoneItem, Player, ProdigyData, ProdigyTimelineResponse, ProdigyWatchRow } from '../../types';


// --- Prodigies / Wonderkids -------------------------------------------------

export function fetchProdigies(): Promise<ProdigyData[]> {
  return apiFetch<ProdigyData[]>('/prodigies', undefined, []);
}

export function fetchProdigyWatch(): Promise<{ season_name: string; matchweek: number; rankings: ProdigyWatchRow[] }> {
  return apiFetch<{ season_name: string; matchweek: number; rankings: ProdigyWatchRow[] }>(
    '/prodigies/watch',
    undefined,
    { season_name: '2026-27', matchweek: 1, rankings: [] },
  );
}

export function fetchProdigyTimeline(playerId: string): Promise<ProdigyTimelineResponse> {
  return apiFetch<ProdigyTimelineResponse>(`/prodigies/${encodeURIComponent(playerId)}/timeline`, undefined, {
    player_id: playerId,
    full_name: playerId,
    age: 17,
    current_height_cm: 165,
    baseline_height_cm: 165,
    current_weight_kg: 55,
    baseline_weight_kg: 55,
    height_gain_cm: 0,
    weight_gain_kg: 0,
    ovr: 60,
    potential: 94,
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
  ovr?: number;
  gains?: Record<string, unknown>;
  current_height?: number;
  current_weight?: number;
  height_gain?: number;
  weight_gain?: number;
  remaining_energy?: number;
  max_energy?: number;
}

// Consumed by the prodigy training planner (Phase 3 F1); focus values come
// from TRAINING_FOCUSES in lib/constants.ts.
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
    invalidateApiCache();
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
    invalidateApiCache();
    return { status: 'success' };
  } catch {
    return { status: 'error', message: 'Network error setting school track.' };
  }
}

export function fetchGrowthMilestones(): Promise<GrowthMilestoneItem[]> {
  return apiFetch<GrowthMilestoneItem[]>('/growth/milestones', undefined, []);
}

// Consumed by the prodigy training planner (Phase 3 F1) to show energy spend.
export function fetchTrainingStatus(): Promise<{ training_energy: number; max_training_energy: number }> {
  return apiFetch('/training/status', undefined, { training_energy: 3, max_training_energy: 3 });
}
export interface TrainingProjection {
  player_id: string;
  player_name: string;
  club_id: string;
  focus: 'hypertrophy' | 'technical' | 'tactical';
  rationale: string;
  projected_gains: string[];
  trainable: boolean;
  training_energy: number;
  max_training_energy: number;
}

/** Read-only "what the staff do" projection for any player (Phase 3 F1). */
export function fetchTrainingProjection(playerId: string): Promise<TrainingProjection | null> {
  return apiFetch<TrainingProjection | null>(`/training/projection/${encodeURIComponent(playerId)}`, undefined, null);
}


// Tier B (B2): train any squad player, not just the twelve prodigies. The
// backend registers a growth profile on demand for untracked players.
export function trainPlayer(playerId: string, focus: string): Promise<TrainProdigyResponse> {
  return apiFetch<TrainProdigyResponse>(
    `/players/${encodeURIComponent(playerId)}/train`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ focus }),
    },
    { status: 'error', message: 'Network error running the training session.' },
  );
}

