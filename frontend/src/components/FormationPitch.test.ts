import { describe, expect, test } from 'bun:test';
import type { Player } from '../types';
import { assignFormationSlots } from './FormationPitch';

function player(player_id: string, position: string, category: Player['category']): Player {
  return {
    player_id,
    full_name: player_id,
    position,
    ovr: 80,
    age: 24,
    market_value_eur: 50_000_000,
    universe_wonderkid: false,
    player_source: 'headline',
    club_id: 'TEST',
    goals: 0,
    assists: 0,
    appearances: 0,
    category,
    formatted_value: '€50M',
    wage_eur: 100_000,
    formatted_wage: '€100K/wk',
    contract_years: 3,
    loyalty: 70,
    career_goals: 0,
    career_assists: 0,
    own_goals: 0,
  };
}

describe('formation pitch slot assignment', () => {
  test('right and left wide players remain on their true sides', () => {
    const slots = assignFormationSlots([
      player('Saka', 'RW', 'FWD'),
      player('LeftWinger', 'LW', 'FWD'),
    ]);
    const rw = slots.find((slot) => slot.player.player_id === 'Saka');
    const lw = slots.find((slot) => slot.player.player_id === 'LeftWinger');

    expect(rw).toBeDefined();
    expect(lw).toBeDefined();
    expect(rw!.x).toBeGreaterThan(50);
    expect(lw!.x).toBeLessThan(50);
  });

  test('CAM is central and more advanced than CM', () => {
    const slots = assignFormationSlots([
      player('Bantol', 'CAM', 'MID'),
      player('CentralMid', 'CM', 'MID'),
    ]);
    const cam = slots.find((slot) => slot.player.player_id === 'Bantol');
    const cm = slots.find((slot) => slot.player.player_id === 'CentralMid');

    expect(cam).toBeDefined();
    expect(cm).toBeDefined();
    expect(cam!.x).toBe(50);
    expect(cam!.y).toBeLessThan(cm!.y);
  });

  test('two generic centre-backs receive separate slots without changing positions', () => {
    const first = player('CB-A', 'CB', 'DEF');
    const second = player('CB-B', 'CB', 'DEF');
    const slots = assignFormationSlots([first, second]);

    expect(slots[0].x).not.toBe(slots[1].x);
    expect(slots[0].y).toBe(slots[1].y);
    expect(first.position).toBe('CB');
    expect(second.position).toBe('CB');
  });

  test('explicit player.slot maps 1:1 to non-colliding tactical coordinates for 11 starters', () => {
    const startingEleven = [
      { ...player('P-GK', 'GK', 'GK'), slot: 'GK' },
      { ...player('P-LB', 'LB', 'DEF'), slot: 'LB' },
      { ...player('P-LCB', 'CB', 'DEF'), slot: 'LCB' },
      { ...player('P-RCB', 'CB', 'DEF'), slot: 'RCB' },
      { ...player('P-RB', 'RB', 'DEF'), slot: 'RB' },
      { ...player('P-LCM', 'CM', 'MID'), slot: 'LCM' },
      { ...player('P-CAM', 'CAM', 'MID'), slot: 'CAM' },
      { ...player('P-RCM', 'CM', 'MID'), slot: 'RCM' },
      { ...player('P-LW', 'LW', 'FWD'), slot: 'LW' },
      { ...player('P-ST', 'ST', 'FWD'), slot: 'ST' },
      { ...player('P-RW', 'RW', 'FWD'), slot: 'RW' },
    ];

    const slots = assignFormationSlots(startingEleven);
    expect(slots).toHaveLength(11);

    const positions = new Set<string>();
    for (const s of slots) {
      const key = `${s.x}:${s.y}`;
      expect(positions.has(key)).toBe(false);
      positions.add(key);
    }
  });
});
