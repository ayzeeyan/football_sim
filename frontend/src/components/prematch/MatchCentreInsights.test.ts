import { describe, expect, test } from 'bun:test';
import type { Fixture } from '../../types';
import { matchPrediction } from './MatchCentreInsights';

function fixture(homeRating: number, awayRating: number): Fixture {
  return {
    home: { overall_team_rating: homeRating, form: ['W', 'W', 'D', 'W', 'L'] },
    away: { overall_team_rating: awayRating, form: ['L', 'D', 'L', 'W', 'L'] },
  } as Fixture;
}

describe('pre-match prediction', () => {
  test('always returns a complete, bounded probability split', () => {
    const result = matchPrediction(fixture(92, 64));
    expect(result.home + result.draw + result.away).toBe(100);
    expect(result.home).toBeLessThanOrEqual(72);
    expect(result.away).toBeGreaterThanOrEqual(10);
    expect(result.home).toBeGreaterThan(result.away);
  });

  test('gives the stronger away side the larger win share', () => {
    const result = matchPrediction(fixture(68, 91));
    expect(result.away).toBeGreaterThan(result.home);
  });
});
