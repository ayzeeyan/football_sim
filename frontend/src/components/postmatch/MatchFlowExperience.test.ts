import { describe, expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';
import { composeMatchStory } from '../../lib/matchStory';
import { eventSecondaryLine } from '../../lib/eventIdentity';

const app = readFileSync(new URL('../../App.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const centre = readFileSync(new URL('../matches/SimulationCentreTab.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const postMatch = readFileSync(new URL('./PostMatchBroadcast.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const actions = readFileSync(new URL('../prematch/FixtureActions.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const matchCard = readFileSync(new URL('../matches/MatchCard.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const tactics = readFileSync(new URL('../../lib/tactics.ts', import.meta.url), 'utf8').replace(/\r\n/g, '\n');

describe('simulation-first match flow', () => {
  test('the app uses the simulation centre and has no live socket dependency', () => {
    expect(app).toContain('SimulationCentreTab');
    expect(app).not.toContain('useMatchEngine');
    expect(app).not.toContain('matchSocket');
    expect(app).not.toContain('MatchdayTab');
    expect(centre).toContain('simulateFixture(fixture.id)');
    expect(centre).toContain('<PostMatchModal');
    expect(postMatch).not.toContain('MatchTickPayload');
  });

  test('fixture actions offer one instant simulation path and reports remain readable', () => {
    expect(actions).toContain('Simulate match');
    expect(actions).not.toMatch(/Watch Live|Visual Sim|Quick Sim|Sim to Result/);
    expect(matchCard).toContain('Match Centre');
    expect(matchCard).toContain('Simulate');
    expect(matchCard).not.toMatch(/Watch live|played live/i);
  });

  test('attacking position fit comes from declared roles, not player ids', () => {
    expect(tactics).not.toContain('WK_Reid_Randell_Libatan');
    expect(tactics).not.toContain('WK_Cliergy_Jave_Lanticse');
    expect(tactics).toContain('player.secondary_position');
  });

  test('match stories use recorded beats without claiming cause', () => {
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
  });

  test('assist identity is shown once in the event detail', () => {
    expect(eventSecondaryLine({
      minute: 43,
      display: "Serge Gnabry 43' (Assist: Harry Kane)",
      seq: 1,
      type: 'goal',
      side: 'home',
      assist_player_name: 'Harry Kane',
      player_name: 'Serge Gnabry',
    })).toBe('Assist: Harry Kane');
  });
});
