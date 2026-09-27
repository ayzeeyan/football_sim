import { describe, expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';
import type { SetPieceInspection } from '../../../types';

const panelSource = readFileSync(new URL('./SetPiecesPanel.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const squadTabSource = readFileSync(new URL('./SquadTab.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const statusSource = readFileSync(new URL('./status.ts', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const clubsApiSource = readFileSync(new URL('../../../services/api/clubs.ts', import.meta.url), 'utf8').replace(/\r\n/g, '\n');

describe('SetPiecesPanel (F10 set-piece inspection)', () => {
  test('is wired as a club profile sub-tab', () => {
    expect(statusSource).toContain("'set-pieces'");
    expect(squadTabSource).toContain("import { SetPiecesPanel } from './SetPiecesPanel'");
    expect(squadTabSource).toContain('<SetPiecesPanel club={selectedClub}');
  });

  test('shows all four set-piece roles with reasons', () => {
    expect(panelSource).toContain('Penalties');
    expect(panelSource).toContain('Free kicks');
    expect(panelSource).toContain('Aerial target');
    expect(panelSource).toContain('Corner taker');
    expect(panelSource).toContain('row.pick.reason');
  });

  test('surfaces the probabilistic confidence for weighted picks', () => {
    expect(panelSource).toContain('confidence');
    expect(panelSource).toContain('% likely');
  });

  test('keeps the inspection-only framing', () => {
    expect(panelSource).toContain('Inspection');
    expect(panelSource).toContain('not a locked-in lineup');
    expect(panelSource).not.toContain('Assign');
    expect(panelSource).not.toContain('Choose taker');
  });

  test('the api client targets the set-pieces endpoint', () => {
    expect(clubsApiSource).toContain('/set-pieces');
  });
});

describe('SetPieceInspection contract', () => {
  test('carries the four picks with confidence and reason', () => {
    const ins: SetPieceInspection = {
      penalty_taker: { player_id: 'P1', full_name: 'Star Striker', position: 'ST', ovr: 90, confidence: 1, reason: 'highest penalty score (96)' },
      free_kick_taker: { player_id: 'P2', full_name: 'Dead Ball', position: 'CM', ovr: 86, confidence: 1, reason: 'best dead-ball specialist (shooting 88, threshold 85)' },
      aerial_target: { player_id: 'P3', full_name: 'Tall CB', position: 'CB', ovr: 78, confidence: 0.42, reason: 'highest squared aerial score' },
      corner_taker: { player_id: 'P2', full_name: 'Dead Ball', position: 'CM', ovr: 86, confidence: 0.35, reason: 'highest delivery weighting' },
    };
    expect(ins.penalty_taker?.confidence).toBe(1);
    expect(ins.aerial_target && ins.aerial_target.confidence < 1).toBe(true);
    expect(ins.corner_taker?.player_id).not.toBe(ins.aerial_target?.player_id);
  });
});
