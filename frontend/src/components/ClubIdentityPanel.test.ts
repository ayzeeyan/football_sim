import { describe, expect, test } from 'bun:test';
import React from 'react';
import type { Club } from '../types';
import { ClubIdentityPanel } from './ClubIdentityPanel';

function mockClub(overrides: Partial<Club> = {}): Club {
  return {
    club_id: 'LAL-RMA',
    club_name: 'Real Madrid',
    short_name: 'RMA',
    league: 'La Liga',
    country: 'Spain',
    home_stadium: 'Santiago Bernabéu',
    stadium_capacity: 81044,
    overall_team_rating: 89,
    squad_size: 24,
    squad_avg_ovr: 85.5,
    primary_color: [255, 255, 255],
    secondary_color: [0, 0, 0],
    p: 0,
    w: 0,
    d: 0,
    l: 0,
    gf: 0,
    ga: 0,
    gd: 0,
    pts: 0,
    form: [],
    manager: null,
    ...overrides,
  };
}

describe('ClubIdentityPanel', () => {
  test('returns null when identity is undefined', () => {
    const club = mockClub();
    const result = ClubIdentityPanel({ club });
    expect(result).toBeNull();
  });

  test('mounts and returns JSX element when identity is present', () => {
    const club = mockClub({
      identity: {
        reputation: 96,
        historical_prestige: 100,
        financial_power: 98,
        board_patience: 38,
        academy_quality: 88,
        recruitment_ambition: 100,
        youth_preference: 72,
        transfer_aggressiveness: 92,
        selling_tendency: 20,
      },
      finances: {
        transfer_budget: 120_000_000,
        balance: 250_000_000,
      },
      formatted_transfer_warchest: '€120.0M',
    });

    const element = ClubIdentityPanel({ club });
    expect(element).not.toBeNull();
    expect(React.isValidElement(element)).toBe(true);
  });
});
