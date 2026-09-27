import { describe, expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';

const squadSource = readFileSync(new URL('./SquadTab.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const playerSource = readFileSync(new URL('./PlayerSheet.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const apiSource = readFileSync(new URL('../../services/api.ts', import.meta.url), 'utf8').replace(/\r\n/g, '\n');

describe('club and player profile surfaces', () => {
  test('club page exposes profile navigation rather than a squad-only view', () => {
    for (const tab of ['overview', 'squad', 'fixtures', 'transfers', 'finances', 'history']) {
      expect(squadSource).toContain(`'${tab}'`);
    }
    expect(squadSource).not.toContain('My Club');
    expect(squadSource).not.toContain('My Squad');
    expect(squadSource).not.toContain('My Budget');
    expect(apiSource).toContain('/profile');
    expect(apiSource).toContain('/fixtures');
    expect(apiSource).toContain('/transfers');
  });

  test('player sheet has career and history as first-class tabs', () => {
    expect(playerSource).toContain("['career', 'Career'");
    expect(playerSource).toContain("['history', 'History'");
    expect(playerSource).toContain('Season-by-season');
    expect(playerSource).toContain('Transfer history');
    expect(playerSource).toContain('Contract history');
  });
});
