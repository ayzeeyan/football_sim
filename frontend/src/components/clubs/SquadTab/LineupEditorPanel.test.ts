import { describe, expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';
import type { LineupOverride } from '../../../types';

const panelSource = readFileSync(new URL('./LineupEditorPanel.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const squadTabSource = readFileSync(new URL('./SquadTab.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const statusSource = readFileSync(new URL('./status.ts', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const clubsApiSource = readFileSync(new URL('../../../services/api/clubs.ts', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const pitchSource = readFileSync(new URL('../../matches/FormationPitch.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const agentsSource = readFileSync(new URL('../../../../../AGENTS.md', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const readmeSource = readFileSync(new URL('../../../../../README.md', import.meta.url), 'utf8').replace(/\r\n/g, '\n');

describe('LineupEditorPanel (Tier B, B1)', () => {
  test('is wired as a club profile sub-tab', () => {
    expect(statusSource).toContain("'lineup'");
    expect(squadTabSource).toContain("import { LineupEditorPanel } from './LineupEditorPanel'");
    expect(squadTabSource).toContain('<LineupEditorPanel club={selectedClub}');
  });

  test('edits through the rigid tactical-slot contract', () => {
    expect(panelSource).toContain('FORMATION_SLOTS');
    expect(panelSource).toContain('positionFit');
    expect(panelSource).toContain('Assign every slot before saving');
  });

  test('offers all four formations and an AI fallback', () => {
    expect(panelSource).toContain("'4-3-3'");
    expect(panelSource).toContain("'4-3-3 Attack'");
    expect(panelSource).toContain("'4-2-3-1'");
    expect(panelSource).toContain("'4-4-2'");
    expect(panelSource).toContain('Back to AI');
    expect(panelSource).toContain('falls back to the AI XI');
  });

  test('renders explicit assignments on the pitch', () => {
    expect(pitchSource).toContain('assignments?: Array<{ slot: string; player: Player }>');
    expect(panelSource).toContain('assignments={pitchAssignments}');
  });

  test('the api client covers set and clear', () => {
    expect(clubsApiSource).toContain('/lineup');
    expect(clubsApiSource).toContain('method: \'DELETE\'');
  });

  test('AGENTS.md invariant #4 and README framing are amended in the same change', () => {
    expect(agentsSource).toContain('directed control (Tier B)');
    expect(agentsSource).toContain('Club.LineupOverride');
    expect(agentsSource).not.toContain('does not imply human club management');
    expect(readmeSource).toContain('football-world viewer with directed control');
    expect(readmeSource).toContain('POST /api/clubs/{club_id}/lineup');
  });
});

describe('LineupOverride contract', () => {
  test('carries the formation and slot map', () => {
    const override: LineupOverride = {
      formation: '4-3-3',
      players: {
        GK: 'P1', LB: 'P2', LCB: 'P3', RCB: 'P4', RB: 'P5',
        LCM: 'P6', CM: 'P7', RCM: 'P8', LW: 'P9', ST: 'P10', RW: 'P11',
      },
    };
    expect(Object.keys(override.players)).toHaveLength(11);
    expect(override.formation).toBe('4-3-3');
  });
});
