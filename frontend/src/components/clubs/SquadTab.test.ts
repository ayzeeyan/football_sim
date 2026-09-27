import { describe, expect, test } from 'bun:test';
import type { Fixture, Player } from '../../types';
import {
  SQUAD_FATIGUE_FITNESS,
  isSquadExpiring,
  isSquadFatigued,
  isSquadInForm,
  isSquadUnavailable,
  isSquadUnhappy,
  matchesFixtureFilter,
  passesSquadStatus,
  squadStatusCounts,
} from './SquadTab';

function mockPlayer(overrides: Partial<Player> = {}): Player {
  return {
    player_id: 'p1',
    full_name: 'Test Player',
    position: 'ST',
    category: 'FWD',
    ovr: 75,
    age: 24,
    market_value_eur: 10_000_000,
    universe_wonderkid: false,
    player_source: 'dataset',
    club_id: 'EPL-ARS',
    goals: 0,
    assists: 0,
    appearances: 0,
    formatted_value: '€10.0M',
    wage_eur: 50_000,
    formatted_wage: '€50K',
    contract_years: 3,
    loyalty: 60,
    career_goals: 0,
    career_assists: 0,
    own_goals: 0,
    ...overrides,
  };
}

function mockFixture(competition: string): Fixture {
  return { competition } as Fixture;
}

describe('squad status helpers', () => {
  test('fatigue uses the single shared cutoff', () => {
    expect(SQUAD_FATIGUE_FITNESS).toBe(60);
    expect(isSquadFatigued(mockPlayer({ fitness: 59 }))).toBe(true);
    expect(isSquadFatigued(mockPlayer({ fitness: 60 }))).toBe(false);
    expect(isSquadFatigued(mockPlayer({ fitness: undefined }))).toBe(false);
  });

  test('status predicates cover each filter bucket', () => {
    expect(isSquadInForm(mockPlayer({ form: 4 }))).toBe(true);
    expect(isSquadInForm(mockPlayer({ form: 2, form_band: 'Excellent' }))).toBe(true);
    expect(isSquadInForm(mockPlayer({ form: 2 }))).toBe(false);
    expect(isSquadUnhappy(mockPlayer({ morale: 54 }))).toBe(true);
    expect(isSquadUnhappy(mockPlayer({ morale: 55 }))).toBe(false);
    expect(isSquadExpiring(mockPlayer({ contract_years: 1 }))).toBe(true);
    expect(isSquadExpiring(mockPlayer({ contract_years: 2 }))).toBe(false);
    expect(isSquadUnavailable(mockPlayer({ injured_matches: 2 }))).toBe(true);
    expect(isSquadUnavailable(mockPlayer({ suspended_matches: 1 }))).toBe(true);
    expect(isSquadUnavailable(mockPlayer())).toBe(false);
  });

  test('squadStatusCounts and passesSquadStatus agree', () => {
    const squad = [
      mockPlayer({ player_id: 'a', fitness: 40 }),
      mockPlayer({ player_id: 'b', contract_years: 1 }),
      mockPlayer({ player_id: 'c', injured_matches: 3 }),
      mockPlayer({ player_id: 'd' }),
    ];
    const counts = squadStatusCounts(squad);
    expect(counts.all).toBe(4);
    expect(counts.fatigued).toBe(1);
    expect(counts.expiring).toBe(1);
    expect(counts.unavailable).toBe(1);
    expect(counts.unhappy).toBe(0);
    expect(squad.filter((p) => passesSquadStatus(p, 'fatigued')).map((p) => p.player_id)).toEqual(['a']);
    expect(squad.filter((p) => passesSquadStatus(p, 'all'))).toHaveLength(4);
  });
});

describe('matchesFixtureFilter', () => {
  test('all returns every fixture', () => {
    expect(matchesFixtureFilter(mockFixture('fa-cup'), 'all', 'Premier League')).toBe(true);
  });

  test('league filter matches domestic leagues and the club league label', () => {
    expect(matchesFixtureFilter(mockFixture('premier-league'), 'league', 'Premier League')).toBe(true);
    expect(matchesFixtureFilter(mockFixture('la-liga'), 'league', 'La Liga')).toBe(true);
    expect(matchesFixtureFilter(mockFixture('fa-cup'), 'league', 'Premier League')).toBe(false);
    expect(matchesFixtureFilter(mockFixture('champions-league'), 'league', 'Premier League')).toBe(false);
  });

  test('cup filter takes any non-league non-UEFA competition, including new cups', () => {
    expect(matchesFixtureFilter(mockFixture('fa-cup'), 'cup', 'Premier League')).toBe(true);
    expect(matchesFixtureFilter(mockFixture('copa-del-rey'), 'cup', 'La Liga')).toBe(true);
    expect(matchesFixtureFilter(mockFixture('brand-new-cup'), 'cup', 'Serie A')).toBe(true);
    expect(matchesFixtureFilter(mockFixture('premier-league'), 'cup', 'Premier League')).toBe(false);
    expect(matchesFixtureFilter(mockFixture('europa-league'), 'cup', 'Premier League')).toBe(false);
  });

  test('UEFA filters match their competitions', () => {
    expect(matchesFixtureFilter(mockFixture('champions-league'), 'champions-league', 'Serie A')).toBe(true);
    expect(matchesFixtureFilter(mockFixture('ucl'), 'champions-league', 'Serie A')).toBe(true);
    expect(matchesFixtureFilter(mockFixture('europa-league'), 'europa-league', 'Serie A')).toBe(true);
    expect(matchesFixtureFilter(mockFixture('conference-league'), 'conference-league', 'Ligue 1')).toBe(true);
    expect(matchesFixtureFilter(mockFixture('champions-league'), 'europa-league', 'Serie A')).toBe(false);
  });
});
