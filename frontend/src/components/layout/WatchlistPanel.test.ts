import { describe, expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';

const read = (p: string) => readFileSync(new URL(p, import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const panel = read('./WatchlistPanel.tsx');
const worldApi = read('../../services/api/world.ts');
const dashboard = read('../competitions/HomeDashboardTab.tsx');
const clubPanels = read('../clubs/SquadTab/panels.tsx');
const playerSheet = read('../clubs/PlayerSheet/PlayerSheetModal.tsx');

describe('Watchlist (F2)', () => {
  test('the api client covers list and toggle', () => {
    expect(worldApi).toContain("apiFetch<WatchlistResponse>('/watchlist'");
    expect(worldApi).toContain('toggleWatchlist');
    expect(worldApi).toContain("invalidateApiCache('/watchlist')");
  });

  test('the panel lists clubs, players, and competitions with un-watch', () => {
    expect(panel).toContain('fetchWatchlist()');
    expect(panel).toContain('toggleWatchlist(entity, entry.id)');
    expect(panel).toContain('data-watchlist-panel="true"');
  });

  test('the home dashboard hosts the panel', () => {
    expect(dashboard).toContain('<WatchlistPanel');
  });

  test('club and player surfaces expose watch toggles', () => {
    expect(clubPanels).toContain('<WatchToggle entity="club"');
    expect(playerSheet).toContain('<WatchToggle entity="player"');
  });

  test('the panel keeps the observational framing', () => {
    expect(panel).toContain('Purely observational');
    expect(panel).toContain('digest');
    // No management verbs appear in user-facing copy.
    for (const banned of ['sign a player', 'transfer bid', 'pick the XI']) {
      expect(panel.toLowerCase()).not.toContain(banned);
    }
  });
});
