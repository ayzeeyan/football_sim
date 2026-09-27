import { describe, expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';
import type { WhatIfResult } from '../../types';

const panelSource = readFileSync(new URL('./WhatIfPanel.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const postMatchSource = readFileSync(new URL('../postmatch/PostMatchBroadcast.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const preMatchSource = readFileSync(new URL('../prematch/PreMatchModal.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const worldApiSource = readFileSync(new URL('../../services/api/world.ts', import.meta.url), 'utf8').replace(/\r\n/g, '\n');

describe('WhatIfPanel (F4 what-if sandbox)', () => {
  test('keeps the neutral-viewer framing: the recorded result stands', () => {
    expect(panelSource).toContain('recorded result stands');
    expect(panelSource).toContain('never edits the real world');
    expect(panelSource).not.toContain('replay the match');
  });

  test('fetches the sandbox endpoint with a rerollable scratch seed', () => {
    expect(panelSource).toContain('fetchWhatIf(fixtureId, seed)');
    expect(panelSource).toContain('scratch_seed');
    expect(worldApiSource).toContain('/whatif?seed=');
  });

  test('renders hypothetical scoreline, xG, and table movement', () => {
    expect(panelSource).toContain('hypo.home_goals');
    expect(panelSource).toContain('toFixed(2)');
    expect(panelSource).toContain('Hypothetical table movement');
    expect(panelSource).toContain('before_pos');
    expect(panelSource).toContain('after_pos');
  });

  test('is surfaced from both the post-match and pre-match modals', () => {
    expect(postMatchSource).toContain("import { WhatIfPanel } from '../matches/WhatIfPanel'");
    expect(postMatchSource).toContain('<WhatIfPanel fixtureId=');
    expect(preMatchSource).toContain("import { WhatIfPanel } from '../matches/WhatIfPanel'");
    expect(preMatchSource).toContain('<WhatIfPanel fixtureId=');
  });
});

describe('WhatIfResult contract', () => {
  test('hypothetical scoreline carries goals, xG, and table deltas', () => {
    const res: WhatIfResult = {
      fixture_id: 'FX-1',
      competition: 'premier-league',
      matchweek: 4,
      scratch_seed: 7,
      home_id: 'PL-ARS',
      away_id: 'PL-CHE',
      hypothetical: { home_goals: 2, away_goals: 1, home_xg: 1.84, away_xg: 0.92 },
      table: {
        applicable: true,
        home: { club_id: 'PL-ARS', short_name: 'ARS', before_pos: 3, before_pts: 9, before_gd: 5, after_pos: 2, after_pts: 12, after_gd: 6 },
        away: { club_id: 'PL-CHE', short_name: 'CHE', before_pos: 8, before_pts: 5, before_gd: 0, after_pos: 9, after_pts: 5, after_gd: -1 },
      },
    };
    expect(res.hypothetical.home_goals).toBe(2);
    expect(res.table?.home?.after_pts).toBe(12);
    expect(res.table?.away?.after_pos).toBe(9);
  });
});
