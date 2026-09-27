import { describe, expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';
import { TABS, tabFromSlug, slugFromTab } from '../../lib/constants';

const appSource = readFileSync(new URL('../../App.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const readApi = (name: string) => readFileSync(new URL(`../../services/api/${name}.ts`, import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const apiSource = ['competitions', 'world', 'transfers'].map(readApi).join('\n');
const topBar = readFileSync(new URL('../layout/TopBar.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');

describe('career home dashboard', () => {
  test('adds a Home tab as the default landing surface', () => {
    expect(TABS.map((t) => t.slug)).toContain('home');
    expect(tabFromSlug('home')).toBe(8);
    expect(slugFromTab(8)).toBe('home');
    expect(tabFromSlug('')).toBe(8);
  });

  test('wires Continue and the dashboard API', () => {
    expect(appSource).toContain('HomeDashboardTab');
    expect(appSource).toContain("handleMacroSim('continue')");
    expect(apiSource).toContain("'/sim/continue'");
    expect(apiSource).toContain("'/world/dashboard'");
    expect(topBar).toContain('Continue');
    expect(apiSource).toContain('deadline_day');
    expect(apiSource).toContain('/search?');
  });

  test('uses world-hub navigation labels without club-manager framing', () => {
    expect(TABS.map((t) => t.label)).toEqual([
      'Home', 'Match Centre', 'News', 'Tables', 'Competitions', 'History', 'Clubs', 'Players', 'Transfers', 'Wonderkids', 'Statistics',
    ]);
    expect(appSource).toContain('PlayersTab');
    expect(appSource).not.toContain('MANAGER COMMAND CENTER');
    expect(appSource).not.toContain('My Club');
  });
});
