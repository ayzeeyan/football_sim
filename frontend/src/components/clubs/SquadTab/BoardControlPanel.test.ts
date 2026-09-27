import { describe, expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';

const panelSource = readFileSync(new URL('./BoardControlPanel.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const squadTabSource = readFileSync(new URL('./SquadTab.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const statusSource = readFileSync(new URL('./status.ts', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const clubsApiSource = readFileSync(new URL('../../../services/api/clubs.ts', import.meta.url), 'utf8').replace(/\r\n/g, '\n');

describe('BoardControlPanel (Tier B4)', () => {
  test('is wired as a club profile sub-tab', () => {
    expect(statusSource).toContain("'board'");
    expect(squadTabSource).toContain("import { BoardControlPanel } from './BoardControlPanel'");
    expect(squadTabSource).toContain('<BoardControlPanel club={selectedClub}');
  });

  test('sets budget, wage cap, and objective', () => {
    expect(panelSource).toContain('Transfer budget');
    expect(panelSource).toContain('Wage cap');
    expect(panelSource).toContain('Board objective');
    expect(panelSource).toContain('setClubBoard(club.club_id');
  });

  test('offers only the closed objective set', () => {
    expect(panelSource).toContain("'Title challenge'");
    expect(panelSource).toContain("'Survival'");
    expect(panelSource).toContain("'Competitive finish'");
  });

  test('states the binding invariants', () => {
    expect(panelSource).toContain('never exceed the');
    expect(panelSource).toContain('never sit below the committed wage bill');
  });

  test('the api client targets the board endpoint', () => {
    expect(clubsApiSource).toContain('/board');
    expect(clubsApiSource).toContain('budget <= balance and cap >= wage bill');
  });
});
