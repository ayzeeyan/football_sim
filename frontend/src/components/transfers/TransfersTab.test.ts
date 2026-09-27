import { describe, expect, test } from 'bun:test';
import { canStartNextSeason } from './TransfersTab';

describe('transfer center primary action', () => {
  test('starts the next season only after the completed summer window', () => {
    expect(canStartNextSeason({
      is_window_open: false,
      window_type: 'SUMMER',
      season_phase: 'transfer_window',
      window_week: 12,
      max_window_weeks: 12,
    })).toBe(true);

    expect(canStartNextSeason({
      is_window_open: true,
      window_type: 'SUMMER',
      season_phase: 'transfer_window',
      window_week: 12,
      max_window_weeks: 12,
    })).toBe(false);
    expect(canStartNextSeason({
      is_window_open: false,
      window_type: 'CLOSED',
      season_phase: 'season',
      window_week: 0,
      max_window_weeks: 12,
    })).toBe(false);
    expect(canStartNextSeason({
      is_window_open: false,
      window_type: 'WINTER',
      season_phase: 'season',
      window_week: 4,
      max_window_weeks: 4,
    })).toBe(false);
  });
});
