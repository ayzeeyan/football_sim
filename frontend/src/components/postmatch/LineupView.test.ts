import { describe, expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';

// Source-text assertions: the lineup components are hook-bound and cannot be
// invoked outside a renderer, so the wiring is pinned by reading the source
// (the established pattern for hook-component tests in this repo).

const read = (path: string) => readFileSync(path, 'utf8');

describe('post-match lineup formation recovery', () => {
  test('the lineups tab resolves the played formation from XI slots, never a bare 4-3-3 default', () => {
    const source = read('src/components/postmatch/LineupView.tsx');
    expect(source).toContain('resolveLineupFormation(homeFormation, homeXI)');
    expect(source).toContain('resolveLineupFormation(awayFormation, awayXI)');
    expect(source).not.toContain('inferFormationFromSlots(homeXI) ?? homeFormation');
  });

  test('archived fixtures pass their summary formation through to the lineups tab', () => {
    const source = read('src/components/postmatch/PostMatchBroadcast.tsx');
    expect(source).toContain('fixture?.home_formation ?? fixture?.report_summary?.home_formation');
    expect(source).toContain('fixture?.away_formation ?? fixture?.report_summary?.away_formation');
  });

  test('archived lineups show an explicit note instead of a silent empty pitch', () => {
    const source = read('src/components/postmatch/LineupView.tsx');
    expect(source).toContain('ArchivedLineupNote');
    expect(source).toContain('Lineup archived');
  });

  test('history and competition hub load the full fixture before opening a report', () => {
    const history = read('src/components/competitions/HistoryTab.tsx');
    const hub = read('src/components/competitions/CompetitionHubTab/CompetitionHubTab.tsx');
    expect(history).toContain('const detail = await fetchFixture(id)');
    expect(hub).toContain('const detail = await fetchFixture(id)');
    // The fetched detail wins over the bare summary row.
    expect(history).toContain('setOpenFixture(detail ??');
    expect(hub).toContain('setOpenFixture(detail ??');
  });
});
