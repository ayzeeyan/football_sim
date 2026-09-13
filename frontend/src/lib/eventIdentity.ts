import type { Fixture, MatchEventItem, MatchMiniPlayer } from '../types';

function fromMini(mini?: MatchMiniPlayer | null): string {
  const name = mini?.full_name?.trim();
  return name || '';
}

export function eventPlayerId(e: MatchEventItem): string {
  return e.player?.player_id || e.scorer?.player_id || e.player_in?.player_id || e.player_id || '';
}

export function eventPlayerName(e: MatchEventItem, fixture?: Fixture | null): string {
  const nested = fromMini(e.player) || fromMini(e.scorer) || fromMini(e.player_in) || (e.player_name || '').trim();
  if (nested) return nested;
  const id = eventPlayerId(e);
  if (id && fixture) {
    const pools = [fixture.home_xi, fixture.away_xi, fixture.home_bench ?? [], fixture.away_bench ?? []];
    for (const pool of pools) {
      const hit = pool.find((r) => r.player_id === id);
      if (hit?.full_name) return hit.full_name;
    }
  }
  return id || '';
}

export function eventPlayerLabel(e: MatchEventItem, fixture?: Fixture | null): string {
  return eventPlayerName(e, fixture) || 'Unknown';
}
