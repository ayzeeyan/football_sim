import { describe, expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';

const panelSource = readFileSync(new URL('./SettingsPanel.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const appSource = readFileSync(new URL('../../App.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const topBarSource = readFileSync(new URL('./TopBar.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const formatSource = readFileSync(new URL('../../lib/format.ts', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const cssSource = readFileSync(new URL('../../index.css', import.meta.url), 'utf8').replace(/\r\n/g, '\n');

describe('SettingsPanel (F8 viewer settings)', () => {
  test('exposes every required preference', () => {
    expect(panelSource).toContain("set('density'");
    expect(panelSource).toContain("set('reducedMotion'");
    expect(panelSource).toContain("set('soundMuted'");
    expect(panelSource).toContain("set('numberFormat'");
    expect(panelSource).toContain("set('dateFormat'");
    expect(panelSource).toContain("set('defaultTab'");
    expect(panelSource).toContain("set('autoAdvanceSeconds'");
    expect(panelSource).toContain('notifyMatch');
    expect(panelSource).toContain('notifyTransfers');
    expect(panelSource).toContain('notifyMilestones');
    expect(panelSource).toContain('notifyWonderkids');
  });

  test('states the persistence choice and why', () => {
    expect(panelSource).toContain('localStorage');
    expect(panelSource).toContain('survive career resets');
  });

  test('is reachable from the top bar and wired in App', () => {
    expect(topBarSource).toContain('onOpenSettings');
    expect(topBarSource).toContain('Viewer settings');
    expect(appSource).toContain('<SettingsPanel open={settingsOpen}');
    expect(appSource).toContain('onOpenSettings');
  });

  test('App persists settings and applies their side effects', () => {
    expect(appSource).toContain('saveSettings(settings)');
    expect(appSource).toContain('setNumberFormat(settings.numberFormat)');
    expect(appSource).toContain('soundManager.muted = settings.soundMuted');
    expect(appSource).toContain('settingsRootClasses(settings)');
    expect(appSource).toContain('filterUnreadCount(feed, settings)');
    expect(appSource).toContain('settings.autoAdvanceSeconds');
  });

  test('the landing tab honours the default-tab preference', () => {
    expect(appSource).toContain('loadSettings().defaultTab');
  });

  test('currency formatting respects the preference', () => {
    expect(formatSource).toContain('setNumberFormat');
    expect(formatSource).toContain("notation: numberFormatMode === 'full' ? 'standard' : 'compact'");
  });

  test('density and motion preferences have CSS backing', () => {
    expect(cssSource).toContain('.reduce-motion *');
    expect(cssSource).toContain('.density-compact .console-card');
  });
});
