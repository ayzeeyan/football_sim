import { describe, expect, test } from 'bun:test';
import { eventClock, eventKindLabel, eventPlayerLabel, eventSecondaryLine, eventAssistName, isKeyMoment } from './eventIdentity';
import { composeMatchStory, scorerLines } from './matchStory';
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

const gnabry: MatchEventItem = {
  minute: 43,
  display: "Serge Gnabry 43' (Assist: Harry Kane)",
  seq: 1,
  type: 'goal',
  side: 'home',
  player_name: 'Serge Gnabry',
  assist_player_name: 'Harry Kane',
  scorer: { player_id: 'P1', full_name: 'Serge Gnabry', position: 'RW', ovr: 84, age: 28, category: 'FWD', is_wk: false },
  assister: { player_id: 'P2', full_name: 'Harry Kane', position: 'ST', ovr: 90, age: 31, category: 'FWD', is_wk: false },
};

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

  test('clock ignores scorer names baked into display', () => {
    expect(eventClock({ minute: 27, display: "Christos Tzolis 27'", seq: 1, type: 'goal', side: 'home' })).toBe("27'");
    expect(eventClock({ minute: 51, display: "51'", seq: 2, type: 'goal', side: 'home' })).toBe("51'");
    expect(eventClock({ minute: 92, display: "90+2'", seq: 3, type: 'goal', side: 'away' })).toBe("90+2'");
    expect(eventClock(gnabry)).toBe("43'");
  });

  test('kind labels stay independent from display text', () => {
    expect(eventKindLabel({ minute: 27, display: "Christos Tzolis 27'", seq: 1, type: 'goal', side: 'home' })).toBe('Goal');
    expect(eventKindLabel({ minute: 40, display: "Red Card: X 40'", seq: 2, type: 'red', side: 'away' })).toBe('Red card');
  });

  test('assist stays metadata on the goal, not a second event label', () => {
    expect(eventAssistName(gnabry)).toBe('Harry Kane');
    expect(eventSecondaryLine(gnabry)).toBe('Assist: Harry Kane');
    expect(eventSecondaryLine(gnabry)).not.toContain('Serge Gnabry - assist');
  });

  test('long names stay a single label', () => {
    const event: MatchEventItem = {
      minute: 12,
      display: "12'",
      seq: 1,
      type: 'goal',
      side: 'home',
      player_name: 'Trent Alexander-Arnold',
      scorer: { player_id: 'TAA', full_name: 'Trent Alexander-Arnold', position: 'RB', ovr: 86, age: 26, category: 'DEF', is_wk: false },
    };
    expect(eventPlayerLabel(event)).toBe('Trent Alexander-Arnold');
    expect(eventClock(event)).toBe("12'");
  });

  test('substitution secondary line uses On for', () => {
    const event: MatchEventItem = {
      minute: 88,
      display: "88'",
      seq: 4,
      type: 'sub',
      side: 'home',
      player_in: { player_id: 'IN', full_name: 'Leroy Sané', position: 'RW', ovr: 84, age: 28, category: 'FWD', is_wk: false },
      player_out: { player_id: 'OUT', full_name: 'Serge Gnabry', position: 'RW', ovr: 84, age: 28, category: 'FWD', is_wk: false },
    };
    expect(eventSecondaryLine(event)).toBe('On for Serge Gnabry');
    expect(isKeyMoment(event)).toBe(true);
  });

  test('yellows are not key moments; reds, penalties, and injuries are', () => {
    expect(isKeyMoment(yellow({}))).toBe(false);
    expect(isKeyMoment({ minute: 63, display: "63'", seq: 2, type: 'red', side: 'away' })).toBe(true);
    expect(isKeyMoment({ minute: 40, display: "40'", seq: 3, type: 'penalty', side: 'home' })).toBe(true);
    expect(isKeyMoment({ minute: 51, display: "51'", seq: 4, type: 'injury', side: 'home', player_name: 'Jamal Musiala' })).toBe(true);
    expect(eventSecondaryLine(yellow({ player_name: 'Joshua Kimmich' }))).toBe('Yellow card');
    expect(eventSecondaryLine({ minute: 63, display: "63'", seq: 2, type: 'red', side: 'away' })).toBe('Red card');
  });

  test('scorer lines do not repeat the player name', () => {
    const lines = scorerLines(
      [
        { minute: 27, display: "Christos Tzolis 27'", seq: 1, type: 'goal', side: 'home', player_name: 'Christos Tzolis', scorer: { player_id: 'P1', full_name: 'Christos Tzolis', position: 'LW', ovr: 78, age: 24, category: 'FWD', is_wk: false } },
        { minute: 51, display: "Christos Tzolis 51'", seq: 2, type: 'goal', side: 'home', player_name: 'Christos Tzolis', scorer: { player_id: 'P1', full_name: 'Christos Tzolis', position: 'LW', ovr: 78, age: 24, category: 'FWD', is_wk: false } },
      ],
      'home',
    );
    expect(lines).toEqual(["Christos Tzolis 27', 51'"]);
    expect(lines[0]).not.toContain('Christos Tzolis Christos Tzolis');
  });

  test('match story stays factual and compact', () => {
    const story = composeMatchStory(
      { short_name: 'RMA', club_name: 'Real Madrid' } as never,
      { short_name: 'BAR', club_name: 'Barcelona' } as never,
      2,
      1,
      [{ kind: 'comeback', side: 'home' }, { kind: 'late_winner', minute: 78, player_name: 'Jude Bellingham', side: 'home' }],
    );
    expect(story).toContain('Real Madrid recovered from behind');
    expect(story).toContain('Jude Bellingham scoring the winner in the 78th minute');
    expect(story).not.toContain('caused');
    expect(story.split('. ').length).toBeLessThanOrEqual(4);
  });
});
