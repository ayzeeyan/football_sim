import { describe, expect, test } from 'bun:test';
import { eventPlayerLabel } from './eventIdentity';
import type { MatchEventItem } from '../types';

function yellow(over: Partial<MatchEventItem>): MatchEventItem {
  return {
    minute: 41,
    display: "41'",
    seq: 1,
    type: 'yellow',
    side: 'away',
    ...over,
  };
}

describe('event identity', () => {
  test('prefers nested player name over Unknown', () => {
    expect(eventPlayerLabel(yellow({ player: { player_id: 'P1', full_name: 'Pedro Porro', position: 'RB', ovr: 84, age: 25, category: 'DEF', is_wk: false } }))).toBe('Pedro Porro');
  });

  test('uses flat player_name when nested name is missing', () => {
    expect(eventPlayerLabel(yellow({ player_id: 'P1', player_name: 'Pedro Porro', player: { player_id: 'P1', full_name: '', position: 'RB', ovr: 84, age: 25, category: 'DEF', is_wk: false } }))).toBe('Pedro Porro');
  });

  test('Unknown is only the last fallback', () => {
    expect(eventPlayerLabel(yellow({}))).toBe('Unknown');
  });
});
