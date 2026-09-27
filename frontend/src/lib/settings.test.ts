import { describe, expect, test } from 'bun:test';
import {
  DEFAULT_SETTINGS,
  categoryAllowed,
  filterUnreadCount,
  formatDateParts,
  loadSettings,
  saveSettings,
  settingsRootClasses,
  type ViewerSettings,
} from './settings';

const KEY = 'football_sim.settings.v1';

// Bun's test environment has no DOM: install a minimal localStorage shim so
// the persistence path is exercised for real.
const storage = new Map<string, string>();
(globalThis as Record<string, unknown>).window = {
  localStorage: {
    getItem: (k: string) => storage.get(k) ?? null,
    setItem: (k: string, v: string) => storage.set(k, v),
    removeItem: (k: string) => storage.delete(k),
  },
};

function withCleanStorage(run: () => void) {
  storage.delete(KEY);
  run();
  storage.delete(KEY);
}

describe('viewer settings (F8)', () => {
  test('defaults load when nothing is stored', () => {
    withCleanStorage(() => {
      const s = loadSettings();
      expect(s).toEqual(DEFAULT_SETTINGS);
      expect(s.soundMuted).toBe(false);
      expect(s.autoAdvanceSeconds).toBe(0);
    });
  });

  test('settings survive a save/load round trip', () => {
    withCleanStorage(() => {
      const next: ViewerSettings = {
        ...DEFAULT_SETTINGS,
        density: 'compact',
        reducedMotion: true,
        soundMuted: true,
        numberFormat: 'full',
        dateFormat: 'mdy',
        defaultTab: 'tables',
        autoAdvanceSeconds: 30,
        notifyTransfers: false,
      };
      saveSettings(next);
      expect(loadSettings()).toEqual(next);
    });
  });

  test('malformed stored values fall back to defaults and clamp', () => {
    withCleanStorage(() => {
      storage.set(KEY, JSON.stringify({ density: 'bogus', autoAdvanceSeconds: 99999, soundMuted: 'yes' }));
      const s = loadSettings();
      expect(s.density).toBe('comfortable');
      expect(s.autoAdvanceSeconds).toBe(600);
      expect(s.soundMuted).toBe(false);
    });
  });

  test('corrupt storage is non-fatal', () => {
    withCleanStorage(() => {
      storage.set(KEY, '{not json');
      expect(loadSettings()).toEqual(DEFAULT_SETTINGS);
    });
  });

  test('root classes reflect density and motion preferences', () => {
    expect(settingsRootClasses({ ...DEFAULT_SETTINGS })).toBe('');
    expect(settingsRootClasses({ ...DEFAULT_SETTINGS, density: 'compact' })).toBe('density-compact');
    expect(settingsRootClasses({ ...DEFAULT_SETTINGS, reducedMotion: true })).toBe('reduce-motion');
    expect(settingsRootClasses({ ...DEFAULT_SETTINGS, density: 'compact', reducedMotion: true })).toBe('density-compact reduce-motion');
  });

  test('date format honours the preference', () => {
    expect(formatDateParts(2026, 9, 28, { ...DEFAULT_SETTINGS, dateFormat: 'dmy' })).toBe('28/09/2026');
    expect(formatDateParts(2026, 9, 28, { ...DEFAULT_SETTINGS, dateFormat: 'mdy' })).toBe('09/28/2026');
  });

  test('notification preferences gate inbox categories', () => {
    const s = { ...DEFAULT_SETTINGS, notifyTransfers: false, notifyMilestones: false };
    expect(categoryAllowed('transfer', s)).toBe(false);
    expect(categoryAllowed('watch', s)).toBe(false);
    expect(categoryAllowed('milestone', s)).toBe(false);
    expect(categoryAllowed('honour', s)).toBe(false);
    expect(categoryAllowed('match', s)).toBe(true);
    expect(categoryAllowed('wonderkid', s)).toBe(true);
    expect(categoryAllowed('system', s)).toBe(true);
  });

  test('unread badge counts only allowed categories', () => {
    const s = { ...DEFAULT_SETTINGS, notifyTransfers: false };
    const feed = {
      unread: 3,
      items: [
        { category: 'match', unread: true },
        { category: 'transfer', unread: true },
        { category: 'milestone', unread: true },
        { category: 'match', unread: false },
      ],
    };
    expect(filterUnreadCount(feed, s)).toBe(2);
    // No items fetched: fall back to the server total.
    expect(filterUnreadCount({ unread: 5, items: [] }, s)).toBe(5);
    expect(filterUnreadCount({ unread: 0, items: [] }, DEFAULT_SETTINGS)).toBe(0);
  });
});
