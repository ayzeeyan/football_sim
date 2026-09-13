import { describe, expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';
import { TABS, tabFromSlug, slugFromTab } from '../lib/constants';
import { tieGroups, aggregateLine } from './CompetitionHubTab';
import type { CompetitionFixtureRow } from '../types';

const appSource = readFileSync(new URL('../App.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const apiSource = readFileSync(new URL('../services/api.ts', import.meta.url), 'utf8').replace(/\r\n/g, '\n');

describe('competition hub navigation', () => {
  test('adds a competitions tab without replacing league compatibility', () => {
    const slugs = TABS.map((t) => t.slug);
    expect(slugs).toContain('league');
    expect(slugs).toContain('competitions');
    expect(tabFromSlug('competitions')).toBe(7);
    expect(slugFromTab(7)).toBe('competitions');
    expect(tabFromSlug('cup')).toBe(2);
  });

  test('app mounts the hub and clients call the competition APIs', () => {
    expect(appSource).toContain('CompetitionHubTab');
    expect(appSource).toContain('activeTab === 7');
    expect(apiSource).toContain("apiFetch<CompetitionsResponse>('/competitions'");
    expect(apiSource).toContain('`/competitions/${encodeURIComponent(id)}`');
		expect(appSource).toContain('onWatchFixture');
  });
});

function legRow(over: Partial<CompetitionFixtureRow> & { id: string }): CompetitionFixtureRow {
  return {
    fixture_id: over.id,
    matchweek: 30,
    competition: 'champions-league',
    stage: 'Round of 16',
    status: 'finished',
    home_id: 'A',
    away_id: 'B',
    home: null,
    away: null,
    home_goals: 0,
    away_goals: 0,
    leg: null,
    tie_id: null,
    ...over,
  };
}

describe('two-legged tie display', () => {
  test('tieGroups clusters legs by tie_id and sorts legs numerically', () => {
    const l2 = legRow({ id: 'L2', tie_id: 'T', leg: 2 });
    const l1 = legRow({ id: 'L1', tie_id: 'T', leg: 1 });
    const single = legRow({ id: 'S', tie_id: 'S' });
    const groups = tieGroups([l2, l1, single]);
    expect(groups.length).toBe(2);
    expect(groups[0].map((f) => f.id)).toEqual(['L1', 'L2']);
    expect(groups[1].map((f) => f.id)).toEqual(['S']);
  });

  test('aggregateLine sums per club across swapped venues', () => {
    // Leg 1: ARS 2–0 CHE at ARS. Leg 2: CHE 1–0 ARS at CHE. Aggregate ARS 2–1.
    const l1 = legRow({ id: 'L1', tie_id: 'T', leg: 1, home_id: 'EPL-ARS', away_id: 'EPL-CHE', home_goals: 2, away_goals: 0, home: { club_id: 'EPL-ARS', club_name: 'Arsenal', short_name: 'ARS' } as never, away: { club_id: 'EPL-CHE', club_name: 'Chelsea', short_name: 'CHE' } as never });
    const l2 = legRow({ id: 'L2', tie_id: 'T', leg: 2, home_id: 'EPL-CHE', away_id: 'EPL-ARS', home_goals: 1, away_goals: 0, home: { club_id: 'EPL-CHE', club_name: 'Chelsea', short_name: 'CHE' } as never, away: { club_id: 'EPL-ARS', club_name: 'Arsenal', short_name: 'ARS' } as never });
    expect(aggregateLine([l1, l2])).toBe('AGG ARS 2–1 CHE');
  });

  test('aggregateLine yields null unless both legs are finished', () => {
    const l1 = legRow({ id: 'L1', tie_id: 'T', leg: 1, home_goals: 2, away_goals: 0 });
    const pending = legRow({ id: 'L2', tie_id: 'T', leg: 2, status: 'scheduled' });
    expect(aggregateLine([l1, pending])).toBeNull();
    expect(aggregateLine([l1])).toBeNull();
    expect(aggregateLine([])).toBeNull();
  });
});
