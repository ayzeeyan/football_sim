import { describe, expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';

const selectorSource = readFileSync(new URL('./NationsSquadSelector.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const boardsSource = readFileSync(new URL('./CompetitionHubTab/boards.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const competitionsApiSource = readFileSync(new URL('../../services/api/competitions.ts', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const typesSource = readFileSync(new URL('../../types/index.ts', import.meta.url), 'utf8').replace(/\r\n/g, '\n');

describe('NationsSquadSelector (Tier B6 national squad selection)', () => {
  test('is surfaced from the Nations Cup squads section', () => {
    expect(boardsSource).toContain("import { NationsSquadSelector } from '../NationsSquadSelector'");
    expect(boardsSource).toContain('Select squad');
    expect(boardsSource).toContain('<NationsSquadSelector');
  });

  test('enforces the rigid squad contract in the UI', () => {
    expect(selectorSource).toContain('picked.size === squadSize');
    expect(selectorSource).toContain('goalkeepers >= 1');
    expect(selectorSource).toContain('disabled={busy || !complete}');
    expect(selectorSource).toContain('next.size < squadSize');
  });

  test('offers the AI fallback', () => {
    expect(selectorSource).toContain('Back to AI');
    expect(selectorSource).toContain('clearNationsSquad');
  });

  test('keeps the eligibility framing: the pool comes from the server', () => {
    expect(selectorSource).toContain('eligible_pool');
    expect(selectorSource).toContain('viewer_selected');
  });

  test('the api client targets the squad endpoints', () => {
    expect(competitionsApiSource).toContain('/competitions/nations-cup/teams/');
    expect(competitionsApiSource).toContain('export function fetchNationsSquadSelection');
    expect(competitionsApiSource).toContain('export async function setNationsSquad');
    expect(competitionsApiSource).toContain('export async function clearNationsSquad');
  });

  test('the selection payload is typed', () => {
    expect(typesSource).toContain('export interface NationsSquadSelection');
    expect(typesSource).toContain('export interface NationsEligiblePlayer');
  });
});
