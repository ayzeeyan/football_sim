/**
 * Shared roving-tabindex keyboard navigation for ARIA tab lists.
 * Extracted from the three modal tab stacks (PreMatchModal,
 * PostMatchBroadcast, TransfersTab) so the behaviour is identical and
 * unit-testable: ArrowLeft/ArrowRight move with wrap-around, Home/End jump
 * to the first/last tab, every other key is ignored.
 */
export const TAB_NAV_KEYS = ['ArrowLeft', 'ArrowRight', 'Home', 'End'] as const;

/** Returns the next tab index for a tablist keydown, or null when the key
 *  does not navigate. `currentIndex` may be -1 when the event target is not
 *  one of the tabs; navigation then still works from Home/End. */
export function nextTabIndex(key: string, currentIndex: number, tabCount: number): number | null {
  if (tabCount <= 0) return null;
  switch (key) {
    case 'Home':
      return 0;
    case 'End':
      return tabCount - 1;
    case 'ArrowRight':
      return ((Math.max(currentIndex, 0)) + 1) % tabCount;
    case 'ArrowLeft':
      return ((Math.max(currentIndex, 0)) - 1 + tabCount) % tabCount;
    default:
      return null;
  }
}

/** Focuses and activates the tab at `index` inside the tablist element. */
export function focusTabAt(tablist: Element, index: number): void {
  const tabs = Array.from(tablist.querySelectorAll<HTMLButtonElement>('[role="tab"]'));
  const tab = tabs[index];
  if (!tab) return;
  tab.focus();
  tab.click();
}
