/**
 * Viewer settings (F8), persisted in localStorage.
 *
 * Why localStorage and not the career save: these are preferences of the
 * person looking at the world, not world state. Keeping them out of the
 * save format avoids save-version churn for non-simulation data, and they
 * survive career resets, new careers, and server restarts by construction.
 */

export type SettingsDensity = 'comfortable' | 'compact';
export type NumberFormatMode = 'compact' | 'full';
export type DateFormatMode = 'dmy' | 'mdy';

export interface ViewerSettings {
  /** Spacing scale for cards and tables. */
  density: SettingsDensity;
  /** Suppresses animations and transitions regardless of OS preference. */
  reducedMotion: boolean;
  /** Sound mute, applied to the shared soundManager. */
  soundMuted: boolean;
  /** Compact (€25M) or full (€25,000,000) currency rendering. */
  numberFormat: NumberFormatMode;
  /** Day-first or month-first date rendering. */
  dateFormat: DateFormatMode;
  /** Tab slug opened on a fresh load (empty = Home). */
  defaultTab: string;
  /** Auto-advance one matchweek every N seconds; 0 disables. */
  autoAdvanceSeconds: number;
  /** Notification preferences: which inbox categories surface as unread. */
  notifyMatch: boolean;
  notifyTransfers: boolean;
  notifyMilestones: boolean;
  notifyWonderkids: boolean;
}

export const DEFAULT_SETTINGS: ViewerSettings = {
  density: 'comfortable',
  reducedMotion: false,
  soundMuted: false,
  numberFormat: 'compact',
  dateFormat: 'dmy',
  defaultTab: '',
  autoAdvanceSeconds: 0,
  notifyMatch: true,
  notifyTransfers: true,
  notifyMilestones: true,
  notifyWonderkids: true,
};

const STORAGE_KEY = 'football_sim.settings.v1';

/** Category buckets the notification preferences gate. */
export function categoryAllowed(category: string, s: ViewerSettings): boolean {
  switch (category) {
    case 'match':
    case 'cup':
    case 'race':
      return s.notifyMatch;
    case 'transfer':
    case 'watch':
      return s.notifyTransfers;
    case 'milestone':
    case 'honour':
    case 'dugout':
      return s.notifyMilestones;
    case 'wonderkid':
    case 'youth':
    case 'nxgn':
      return s.notifyWonderkids;
    default:
      return true;
  }
}

function coerceString(value: unknown, allowed: readonly string[], fallback: string): string {
  return typeof value === 'string' && (allowed as readonly unknown[]).includes(value) ? value : fallback;
}

function coerceBool(value: unknown, fallback: boolean): boolean {
  return typeof value === 'boolean' ? value : fallback;
}

function coerceNumber(value: unknown, fallback: number, min: number, max: number): number {
  const n = typeof value === 'number' && Number.isFinite(value) ? Math.round(value) : fallback;
  return Math.min(max, Math.max(min, n));
}

/** Loads settings, merging over defaults and clamping anything malformed. */
export function loadSettings(): ViewerSettings {
  if (typeof window === 'undefined') return { ...DEFAULT_SETTINGS };
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return { ...DEFAULT_SETTINGS };
    const parsed = JSON.parse(raw) as Partial<ViewerSettings>;
    return {
      density: coerceString(parsed.density, ['comfortable', 'compact'], DEFAULT_SETTINGS.density) as SettingsDensity,
      reducedMotion: coerceBool(parsed.reducedMotion, DEFAULT_SETTINGS.reducedMotion),
      soundMuted: coerceBool(parsed.soundMuted, DEFAULT_SETTINGS.soundMuted),
      numberFormat: coerceString(parsed.numberFormat, ['compact', 'full'], DEFAULT_SETTINGS.numberFormat) as NumberFormatMode,
      dateFormat: coerceString(parsed.dateFormat, ['dmy', 'mdy'], DEFAULT_SETTINGS.dateFormat) as DateFormatMode,
      defaultTab: typeof parsed.defaultTab === 'string' ? parsed.defaultTab.slice(0, 32) : DEFAULT_SETTINGS.defaultTab,
      autoAdvanceSeconds: coerceNumber(parsed.autoAdvanceSeconds, DEFAULT_SETTINGS.autoAdvanceSeconds, 0, 600),
      notifyMatch: coerceBool(parsed.notifyMatch, DEFAULT_SETTINGS.notifyMatch),
      notifyTransfers: coerceBool(parsed.notifyTransfers, DEFAULT_SETTINGS.notifyTransfers),
      notifyMilestones: coerceBool(parsed.notifyMilestones, DEFAULT_SETTINGS.notifyMilestones),
      notifyWonderkids: coerceBool(parsed.notifyWonderkids, DEFAULT_SETTINGS.notifyWonderkids),
    };
  } catch {
    return { ...DEFAULT_SETTINGS };
  }
}

/** Persists settings; failures (private mode, quota) are non-fatal. */
export function saveSettings(settings: ViewerSettings): void {
  if (typeof window === 'undefined') return;
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(settings));
  } catch {
    // Storage unavailable: settings stay session-only.
  }
}

/** Root element classes for the density and motion preferences. */
export function settingsRootClasses(s: ViewerSettings): string {
  const classes: string[] = [];
  if (s.density === 'compact') classes.push('density-compact');
  if (s.reducedMotion) classes.push('reduce-motion');
  return classes.join(' ');
}

/**
 * Counts unread inbox items the viewer still wants to be notified about.
 * The count reflects the fetched page; the server total is used when the
 * page cannot express the preference (no items fetched).
 */
export function filterUnreadCount(
  feed: { unread: number; items: Array<{ category: string; unread: boolean }> },
  s: ViewerSettings,
): number {
  if (!feed) return 0;
  if (!Array.isArray(feed.items) || feed.items.length === 0) return feed.unread;
  return feed.items.filter((item) => item.unread && categoryAllowed(item.category, s)).length;
}

/** Formats a date-like triple honouring the date-format preference. */
export function formatDateParts(year: number, month: number, day: number, s: ViewerSettings): string {
  const mm = String(month).padStart(2, '0');
  const dd = String(day).padStart(2, '0');
  return s.dateFormat === 'mdy' ? `${mm}/${dd}/${year}` : `${dd}/${mm}/${year}`;
}
