import { describe, expect, test } from 'bun:test';
import { qualificationBand } from './qualification';

describe('qualification bands', () => {
  test('Premier League 20: 1–4 UCL, 5 EL, 6 ECL, 18–20 relegation', () => {
    expect(qualificationBand('Premier League', 1, 20)).toBe('ucl');
    expect(qualificationBand('Premier League', 4, 20)).toBe('ucl');
    expect(qualificationBand('Premier League', 5, 20)).toBe('el');
    expect(qualificationBand('Premier League', 6, 20)).toBe('ecl');
    expect(qualificationBand('Premier League', 7, 20)).toBeNull();
    expect(qualificationBand('Premier League', 17, 20)).toBeNull();
    expect(qualificationBand('Premier League', 18, 20)).toBe('rel');
    expect(qualificationBand('Premier League', 20, 20)).toBe('rel');
  });

  test('Bundesliga 18: last three relegated, top four UCL', () => {
    expect(qualificationBand('Bundesliga', 4, 18)).toBe('ucl');
    expect(qualificationBand('Bundesliga', 16, 18)).toBe('rel');
    expect(qualificationBand('Bundesliga', 15, 18)).toBeNull();
  });

  test('Ligue 1 uses three Champions League places', () => {
    expect(qualificationBand('Ligue 1', 3, 18)).toBe('ucl');
    expect(qualificationBand('Ligue 1', 4, 18)).toBe('el');
    expect(qualificationBand('Ligue 1', 5, 18)).toBe('ecl');
    expect(qualificationBand('Ligue 1', 16, 18)).toBe('rel');
  });
});
