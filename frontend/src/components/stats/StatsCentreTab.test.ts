import { describe, expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';
import { TABS, NAV_GROUPS } from '../../lib/constants';

const read = (p: string) => readFileSync(new URL(p, import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const tab = read('./StatsCentreTab.tsx');
const app = read('../../App.tsx');
const recordsApi = read('../../services/api/records.ts');

describe('Statistics centre (F3)', () => {
  test('registers a Statistics tab in the navigation', () => {
    const statsTab = TABS.find((t) => t.slug === 'stats');
    expect(statsTab).toBeDefined();
    expect(NAV_GROUPS.some((g) => g.slugs.includes('stats'))).toBe(true);
  });

  test('the app lazy-loads and renders the tab', () => {
    expect(app).toContain("import('./components/stats')");
    expect(app).toContain('activeTab === 10 && <StatsCentreTab');
  });

  test('the api client hits the advanced stats endpoint', () => {
    expect(recordsApi).toContain("'/season/stats/advanced'");
  });

  test('the leaderboard sorts and shows percentiles, zones, and ratings', () => {
    expect(tab).toContain('percentile');
    expect(tab).toContain('shots_box');
    expect(tab).toContain('avg_rating');
    expect(tab).toContain('aria-pressed={sort === key}');
  });

  test('team trends show possession, passing, xG, and territory', () => {
    expect(tab).toContain('avg_possession');
    expect(tab).toContain('avg_pass_accuracy');
    expect(tab).toContain('avg_xg_against');
    expect(tab).toContain('territory_attacking');
  });

  test('the shot-detail retention caveat is surfaced to the viewer', () => {
    expect(tab).toContain('shot_detail_note');
  });
});
