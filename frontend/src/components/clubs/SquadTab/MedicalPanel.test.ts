import { describe, expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';
import type { ClubMedicalResponse, InjuryRecord } from '../../../types';

const panelSource = readFileSync(new URL('./MedicalPanel.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const sheetSource = readFileSync(new URL('../PlayerSheet/PlayerSheetModal.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const squadTabSource = readFileSync(new URL('./SquadTab.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const statusSource = readFileSync(new URL('./status.ts', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const clubsApiSource = readFileSync(new URL('../../../services/api/clubs.ts', import.meta.url), 'utf8').replace(/\r\n/g, '\n');

describe('MedicalPanel (F11 medical view)', () => {
  test('is wired as a club profile sub-tab', () => {
    expect(statusSource).toContain("'medical'");
    expect(squadTabSource).toContain("import { MedicalPanel } from './MedicalPanel'");
    expect(squadTabSource).toContain('<MedicalPanel club={selectedClub}');
  });

  test('renders the treatment room with rehab roadmaps', () => {
    expect(panelSource).toContain('Treatment room');
    expect(panelSource).toContain('rehab.stages');
    expect(panelSource).toContain('out {row.matches_out}');
  });

  test('renders risk assessments with itemised factors', () => {
    expect(panelSource).toContain('risk_score');
    expect(panelSource).toContain('factors');
    expect(panelSource).toContain('load, age, fitness, and match density');
  });

  test('renders the season history summary by severity', () => {
    expect(panelSource).toContain('Season injury history');
    expect(panelSource).toContain('by_severity');
    expect(panelSource).toContain('Matches lost');
  });

  test('keeps the read-only framing', () => {
    expect(panelSource).toContain('Read-only');
    expect(panelSource).toContain('nobody is treated by decree');
  });

  test('the api client targets the medical endpoint', () => {
    expect(clubsApiSource).toContain('/medical');
  });
});

describe('Player sheet medical record', () => {
  test('renders the per-player injury history when present', () => {
    expect(sheetSource).toContain('Medical record');
    expect(sheetSource).toContain('injury_history');
    expect(sheetSource).toContain('rec.severity');
  });
});

describe('Medical contract types', () => {
  test('injury records and the club view round-trip', () => {
    const rec: InjuryRecord = {
      season: '2026-27', matchweek: 5, kind: 'hamstring tear',
      severity: 'moderate', matches_out: 7, fixture_id: 'FX-1',
    };
    expect(rec.severity).toBe('moderate');
    const view: ClubMedicalResponse = {
      club_id: 'PL-ARS', club_name: 'Arsenal', season: '2026-27',
      injured: [{
        player_id: 'P1', full_name: 'Test Player', position: 'CM', ovr: 82,
        kind: 'hamstring tear', matches_out: 7,
        rehab: { kind: 'hamstring tear', severity: 'moderate', matches_out: 7, stages: [{ phase: 'Acute care', detail: 'First week' }] },
      }],
      top_risks: [{
        player_id: 'P2', full_name: 'Another', position: 'ST', ovr: 88, fitness: 62,
        assessment: { risk_score: 96, factors: [{ label: 'fitness below 60', mult: 1.5 }] },
      }],
      history: { season: '2026-27', total_injuries: 1, by_severity: { moderate: 1 }, matches_lost: 7 },
    };
    expect(view.injured[0]?.rehab.stages[0]?.phase).toBe('Acute care');
    expect(view.top_risks[0]?.assessment.risk_score).toBe(96);
  });
});
