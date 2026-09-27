import { describe, expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';
import { nextTabIndex } from './rovingTabindex';

describe('roving tablist keyboard navigation', () => {
  test('ArrowRight moves forward and wraps at the end', () => {
    expect(nextTabIndex('ArrowRight', 0, 5)).toBe(1);
    expect(nextTabIndex('ArrowRight', 4, 5)).toBe(0);
  });

  test('ArrowLeft moves backward and wraps at the start', () => {
    expect(nextTabIndex('ArrowLeft', 3, 5)).toBe(2);
    expect(nextTabIndex('ArrowLeft', 0, 5)).toBe(4);
  });

  test('Home and End jump to the first and last tab', () => {
    expect(nextTabIndex('Home', 3, 5)).toBe(0);
    expect(nextTabIndex('End', 0, 5)).toBe(4);
  });

  test('unknown keys do not navigate', () => {
    expect(nextTabIndex('a', 2, 5)).toBeNull();
    expect(nextTabIndex('Enter', 2, 5)).toBeNull();
    expect(nextTabIndex('Tab', 2, 5)).toBeNull();
  });

  test('empty tablists never navigate', () => {
    expect(nextTabIndex('ArrowRight', 0, 0)).toBeNull();
    expect(nextTabIndex('Home', 0, 0)).toBeNull();
  });

  test('single-tab lists stay put but still respond', () => {
    expect(nextTabIndex('ArrowRight', 0, 1)).toBe(0);
    expect(nextTabIndex('ArrowLeft', 0, 1)).toBe(0);
    expect(nextTabIndex('End', 0, 1)).toBe(0);
  });
});

// The three modal tab stacks must keep the ARIA tab contract wired to the
// shared helper: role=tablist container, role=tab buttons with aria-selected
// and roving tabindex, and the keydown handler attached.
describe('modal tab stacks keep the ARIA contract', () => {
  const read = (p: string) => readFileSync(new URL(p, import.meta.url), 'utf8').replace(/\r\n/g, '\n');

  test('post-match broadcast tablist', () => {
    const src = read('../components/postmatch/PostMatchBroadcast.tsx');
    expect(src).toContain('role="tablist"');
    expect(src).toContain('role="tab"');
    expect(src).toContain('aria-selected={tab === id}');
    expect(src).toContain('tabIndex={tab === id ? 0 : -1}');
    expect(src).toContain('onKeyDown={handlePostTabKey}');
    expect(src).toContain('nextTabIndex');
  });

  test('pre-match modal tablist', () => {
    const src = read('../components/prematch/PreMatchModal.tsx');
    expect(src).toContain('role="tablist"');
    expect(src).toContain('onKeyDown={handleFixtureTabKey}');
    expect(src).toContain('nextTabIndex');
  });

  test('transfers tablist', () => {
    const src = read('../components/transfers/TransfersTab.tsx');
    expect(src).toContain('role="tablist"');
    expect(src).toContain('onKeyDown={handleSubTabKeyDown}');
    expect(src).toContain('nextTabIndex');
    // The old handler treated every unknown key as ArrowLeft; the shared
    // helper must be the only navigation path.
    expect(src).not.toMatch(/event\.key === 'Home'\s*\n/);
  });
});
