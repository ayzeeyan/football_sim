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

export function eventAssistId(e: MatchEventItem): string {
  return e.assist_player_id || e.assister?.player_id || '';
}

export function eventAssistName(e: MatchEventItem): string {
  return (e.assist_player_name || e.assister?.full_name || '').trim();
}

/** Clock stamp only. Instant reports used to bake the scorer name into `display`. */
export function eventClock(e: MatchEventItem): string {
  const display = (e.display || '').trim();
  const extra = display.match(/(\d+\+\d+)'/);
  if (extra) return `${extra[1]}'`;
  const simple = display.match(/(\d+)'/);
  if (simple) return `${simple[1]}'`;
  if (typeof e.minute === 'number') return `${e.minute}'`;
  return '';
}

export function eventKindLabel(e: MatchEventItem): string {
  switch (e.type) {
    case 'goal':
    case 'corner_goal':
    case 'free_kick_goal':
      return 'Goal';
    case 'penalty':
      return 'Penalty';
    case 'own_goal':
      return 'Own goal';
    case 'penalty_miss':
      return 'Penalty missed';
    case 'red':
      return 'Red card';
    case 'yellow':
      return 'Yellow card';
    case 'sub':
      return 'Substitution';
    case 'injury':
      return 'Injury';
    case 'var_review':
      return e.outcome === 'goal_disallowed' ? 'Goal disallowed' : 'VAR';
    default:
      return 'Event';
  }
}

export function eventIcon(e: MatchEventItem): string {
  switch (e.type) {
    case 'goal':
    case 'corner_goal':
    case 'free_kick_goal':
    case 'penalty':
      return '⚽';
    case 'own_goal':
      return '⚽';
    case 'penalty_miss':
      return '🚫';
    case 'red':
      return '🟥';
    case 'yellow':
      return '🟨';
    case 'sub':
      return '🔁';
    case 'injury':
      return '✚';
    case 'var_review':
      return '📺';
    default:
      return '•';
  }
}

export function isScoringEvent(e: MatchEventItem): boolean {
  return ['goal', 'penalty', 'own_goal', 'corner_goal', 'free_kick_goal'].includes(e.type) && !e.disallowed;
}

export function isKeyMoment(e: MatchEventItem): boolean {
  if (e.disallowed) return false;
  if (['goal', 'penalty', 'own_goal', 'corner_goal', 'free_kick_goal', 'red', 'penalty_miss', 'injury'].includes(e.type)) return true;
  if (e.type === 'var_review' && e.outcome === 'goal_disallowed') return true;
  if (e.type === 'sub' && e.minute >= 80) return true;
  return false;
}

export function eventSecondaryLine(e: MatchEventItem): string {
  if (e.type === 'sub') {
    const outgoing = e.player_out?.full_name?.trim();
    return outgoing ? `On for ${outgoing}` : 'Substitution';
  }
  if (e.type === 'own_goal') return 'Own goal';
  if (e.type === 'penalty') return 'Penalty';
  if (e.type === 'penalty_miss') return 'Penalty missed';
  if (e.type === 'red') return e.detail === 'second_yellow' ? 'Second yellow' : 'Red card';
  if (e.type === 'yellow') return 'Yellow card';
  if (e.type === 'injury') return (e.detail || 'Injury').replace(/_/g, ' ');
  if (e.type === 'var_review') {
    if (e.reason) return e.reason.replace(/_/g, ' ');
    return eventKindLabel(e);
  }
  const assist = eventAssistName(e);
  if (assist) return `Assist: ${assist}`;
  return eventKindLabel(e);
}

export function clubForEvent(event: MatchEventItem, homeName?: string, awayName?: string): string {
  if (event.club_name) return event.club_name;
  const side = event.type === 'own_goal'
    ? (event.beneficiary === 'home' || event.beneficiary === 'away' ? event.beneficiary : event.side === 'home' ? 'away' : 'home')
    : event.side;
  return side === 'away' ? (awayName || 'Away') : (homeName || 'Home');
}
