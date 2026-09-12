import { describe, expect, it } from 'bun:test';
import {
  FALLBACK_PALETTES,
  getPlayerFallbackColors,
  getPlayerInitials,
  getPlayerPortraitUrl,
  hashPlayerId,
} from './playerPortraits';

describe('PlayerPortrait Fallback', () => {
  it('never returns fake placeholder photo URLs for unmapped players', () => {
    expect(getPlayerPortraitUrl('P01412')).toBeNull();
    expect(getPlayerPortraitUrl('WK_Venjamin_Valerio')).toBeNull();
    expect(getPlayerPortraitUrl(null)).toBeNull();
    expect(getPlayerPortraitUrl(undefined)).toBeNull();
  });

  it('produces identical deterministic palette across runs for the same player ID', () => {
    const p1 = 'WK_Venjamin_Valerio';
    const first = getPlayerFallbackColors(p1);
    for (let i = 0; i < 50; i++) {
      const repeated = getPlayerFallbackColors(p1);
      expect(repeated).toEqual(first);
    }
  });

  it('generates deterministic hashes and valid palette indexes', () => {
    const ids = ['P00001', 'P00002', 'WK_Izyan_Levin_Bantol', 'WK_Jhed_Anthony_Guinita', 'P99999'];
    for (const id of ids) {
      const hash = hashPlayerId(id);
      expect(hash).toBeGreaterThanOrEqual(0);
      const colors = getPlayerFallbackColors(id);
      expect(FALLBACK_PALETTES).toContain(colors as any);
      expect(colors.bg).toBeTruthy();
      expect(colors.text).toBeTruthy();
      expect(colors.border).toBeTruthy();
    }
  });

  it('extracts honest initials from player names without spoofing', () => {
    expect(getPlayerInitials('Venjamin Valerio')).toBe('VV');
    expect(getPlayerInitials('Jhed Anthony Guinita')).toBe('JG');
    expect(getPlayerInitials('Pedri')).toBe('PE');
    expect(getPlayerInitials('')).toBe('?');
    expect(getPlayerInitials(null)).toBe('?');
  });
});
