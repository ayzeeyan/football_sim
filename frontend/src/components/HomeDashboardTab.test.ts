import { describe, expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';
import { TABS, tabFromSlug, slugFromTab } from '../lib/constants';

const appSource = readFileSync(new URL('../App.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const apiSource = readFileSync(new URL('../services/api.ts', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const topBar = readFileSync(new URL('./layout/TopBar.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');

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
  });
});
