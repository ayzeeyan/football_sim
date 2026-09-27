import { describe, expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';
import type { AchievementsResponse } from '../../types';

const panelSource = readFileSync(new URL('./AchievementsPanel.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const historySource = readFileSync(new URL('../competitions/HistoryTab.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const recordsApiSource = readFileSync(new URL('../../services/api/records.ts', import.meta.url), 'utf8').replace(/\r\n/g, '\n');

describe('AchievementsPanel (F6 milestone ledger)', () => {
  test('is surfaced as the Milestones sub-tab of History', () => {
    expect(historySource).toContain("'achievements'");
    expect(historySource).toContain('Milestones');
    expect(historySource).toContain("import { AchievementsPanel } from '../career/AchievementsPanel'");
    expect(historySource).toContain("subTab === 'achievements' && <AchievementsPanel />");
  });

  test('renders the full catalogue with locked and unlocked states', () => {
    expect(panelSource).toContain('definitions.map');
    expect(panelSource).toContain('unlockedByID');
    expect(panelSource).toContain('Locked');
    expect(panelSource).toContain('unlock_key');
  });

  test('surfaces the youngest-scorer world record', () => {
    expect(panelSource).toContain('youngest_scorer_record');
    expect(panelSource).toContain('Youngest goalscorer');
  });

  test('keeps the neutral-viewer framing: the world unlocks, nobody spends', () => {
    expect(panelSource).toContain('the world unlocks');
    expect(panelSource).not.toContain('Earn');
    expect(panelSource).not.toContain('Your achievements');
  });

  test('the api client targets the achievements endpoint', () => {
    expect(recordsApiSource).toContain("'/achievements'");
  });
});

describe('AchievementsResponse contract', () => {
  test('carries definitions, ledger entries, and the record', () => {
    const res: AchievementsResponse = {
      definitions: [
        { id: 'unbeaten_20', kind: 'club', title: 'The Unbeaten March', description: '20 unbeaten.', unlocked: true },
        { id: 'treble', kind: 'club', title: 'The Treble', description: 'Three trophies.', unlocked: false },
      ],
      achievements: [
        {
          id: 'unbeaten_20', kind: 'club', title: 'The Unbeaten March', description: 'A club strung together 20 league matches without defeat.',
          subject_id: 'PL-ARS', subject_name: 'Arsenal', season: '2026-27', matchweek: 20,
          unlock_key: 'unbeaten_20:PL-ARS:20',
        },
      ],
      youngest_scorer_record: {
        player_id: 'P00001', player_name: 'Test Prodigy', club_id: 'LAL-RMA', age: 16, season: '2026-27', matchweek: 7,
      },
    };
    expect(res.definitions[0]?.unlocked).toBe(true);
    expect(res.achievements[0]?.unlock_key).toBe('unbeaten_20:PL-ARS:20');
    expect(res.youngest_scorer_record?.age).toBe(16);
  });
});
