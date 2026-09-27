import { describe, expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';

const source = readFileSync(new URL('./App.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');

describe('screen-level code splitting', () => {
  test('loads every primary product screen lazily behind an accessible fallback', () => {
    const screens = [
      'HomeDashboardTab',
      'SimulationCentreTab',
      'WonderkidLabTab',
      'StandingsTab',
      'CompetitionHubTab',
      'InboxTab',
      'HistoryTab',
      'SquadTab',
      'PlayersTab',
      'TransfersTab',
    ];
    for (const screen of screens) {
      expect(source).toContain(`const ${screen} = lazy(`);
    }
    expect(source).toContain('<Suspense fallback={<ScreenLoading />}>');
    expect(source).toContain('role="status" aria-live="polite"');
  });
});
