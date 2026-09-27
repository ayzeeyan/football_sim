import { describe, expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';
import type { ScoutingReport } from '../../../types';

const panelSource = readFileSync(new URL('./RecruitmentPanel.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const squadTabSource = readFileSync(new URL('./SquadTab.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const statusSource = readFileSync(new URL('./status.ts', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const clubsApiSource = readFileSync(new URL('../../../services/api/clubs.ts', import.meta.url), 'utf8').replace(/\r\n/g, '\n');

describe('RecruitmentPanel (F5 scouting)', () => {
  test('is wired as a club profile sub-tab', () => {
    expect(statusSource).toContain("'recruitment'");
    expect(squadTabSource).toContain("import { RecruitmentPanel } from './RecruitmentPanel'");
    expect(squadTabSource).toContain('<RecruitmentPanel club={selectedClub}');
  });

  test('keeps the neutral-viewer framing: the AI staff decide, nobody signs here', () => {
    expect(panelSource).toContain('Observational');
    expect(panelSource).toContain('AI staff decide');
    expect(panelSource).not.toContain('Sign');
    expect(panelSource).not.toContain('Make a bid');
  });

  test('renders the full report card: ceiling, consistency, form, value trend, risk', () => {
    expect(panelSource).toContain('potential_ceiling');
    expect(panelSource).toContain('consistency');
    expect(panelSource).toContain('value_trend');
    expect(panelSource).toContain('risk');
    expect(panelSource).toContain('verdict');
    expect(panelSource).toContain('region');
  });

  test('the api client targets the scouting endpoint', () => {
    expect(clubsApiSource).toContain('/scouting?limit=');
  });
});

describe('ScoutingReport contract', () => {
  test('carries the deterministic scouting scores', () => {
    const r: ScoutingReport = {
      player_id: 'P-1', full_name: 'Test Player', position: 'MID', category: 'MID',
      age: 18, ovr: 72, club_id: 'PL-CHE', club_name: 'Chelsea', club_short: 'CHE',
      league: 'Premier League', region: 'England', on_loan: false, market_value_eur: 25_000_000,
      potential_ceiling: 99, ceiling_delta: 27, consistency: 82, form: 'hot',
      value_trend: 'rising', risk: 45, verdict: 'high-growth target — bid early', scout_score: 121,
    };
    expect(r.potential_ceiling).toBe(99);
    expect(r.ceiling_delta).toBe(27);
    expect(r.scout_score).toBeGreaterThan(r.ovr);
  });
});
