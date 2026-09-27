import { describe, expect, test } from 'bun:test';
import type { Player } from '../../types';
import { FORMATION_PITCH_COORDS, FORMATION_SLOTS, REPORT_PITCH_COORDS } from '../../lib/tactics';
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
  test('every supported formation owns eleven unique renderable slots', () => {
    Object.entries(FORMATION_SLOTS).forEach(([formation, slots]) => {
      expect(slots).toHaveLength(11);
      expect(new Set(slots).size).toBe(11);
      expect(new Set(slots.map((slot) => FORMATION_PITCH_COORDS[slot].join(':'))).size).toBe(11);
      expect(new Set(slots.map((slot) => REPORT_PITCH_COORDS[slot].join(':'))).size).toBe(11);
      slots.forEach((slot) => {
        expect(FORMATION_PITCH_COORDS[slot], `${formation} formation coordinate for ${slot}`).toBeDefined();
        expect(REPORT_PITCH_COORDS[slot], `${formation} report coordinate for ${slot}`).toBeDefined();
      });
    });
  });

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
      player('Bantol', 'CAM', 'FWD'),
      player('CentralMid', 'CM', 'MID'),
    ], '4-3-3 Attack');
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

  test('two right-backs with assigned slots do not occupy the same coordinates', () => {
    const slots = assignFormationSlots([
      { ...player('Cancelo', 'RB', 'DEF'), starting_slot: 'LB' },
      { ...player('Kounde', 'RB', 'DEF'), starting_slot: 'RB' },
    ]);
    expect(slots[0].x).not.toBe(slots[1].x);
    expect(slots.find((s) => s.player.player_id === 'Cancelo')!.x).toBeLessThan(50);
    expect(slots.find((s) => s.player.player_id === 'Kounde')!.x).toBeGreaterThan(50);
  });

  test('uses the backend tactical slot rather than duplicating a natural position', () => {
    const slots = assignFormationSlots([
      { ...player('CB-left', 'CB', 'DEF'), starting_slot: 'LCB' },
      { ...player('CB-right', 'CB', 'DEF'), starting_slot: 'RCB' },
    ]);

    expect(slots.map((slot) => slot.slot)).toEqual(['LCB', 'RCB']);
    expect(slots[0].x).toBe(38);
    expect(slots[1].x).toBe(62);
  });

  test('Cantalejo legacy payload is inferred as CAM in 4-2-3-1 regardless of row order', () => {
    const slots = assignFormationSlots([
      player('Natural-LW', 'LW', 'FWD'),
      player('WK_Maverick_Cantalejo', 'CAM', 'FWD'),
      player('Natural-RW', 'RW', 'FWD'),
    ], '4-2-3-1');
    const cantalejo = slots.find((slot) => slot.player.player_id === 'WK_Maverick_Cantalejo');

    expect(cantalejo?.slot).toBe('CAM');
    expect(cantalejo?.x).toBe(50);
    expect(cantalejo?.y).toBe(38);
  });

  test('new tactical_slot wins over natural position and reports the playing lane', () => {
    const slots = assignFormationSlots([
      { ...player('Emergency-CAM', 'CAM', 'FWD'), tactical_slot: 'RW', position_fit: 'Emergency' },
    ], '4-2-3-1');

    expect(slots[0].slot).toBe('RW');
    expect(slots[0].x).toBeGreaterThan(50);
    expect(slots[0].naturalPosition).toBe('CAM');
    expect(slots[0].positionFit).toBe('Emergency');
  });

  test('malformed extra players do not steal a slot or nudge occupied coordinates', () => {
    const xi = [
      { ...player('GK', 'GK', 'GK'), tactical_slot: 'GK' },
      { ...player('LB', 'LB', 'DEF'), tactical_slot: 'LB' },
      { ...player('LCB', 'CB', 'DEF'), tactical_slot: 'LCB' },
      { ...player('RCB', 'CB', 'DEF'), tactical_slot: 'RCB' },
      { ...player('RB', 'RB', 'DEF'), tactical_slot: 'RB' },
      { ...player('LDM', 'CDM', 'MID'), tactical_slot: 'LDM' },
      { ...player('RDM', 'CDM', 'MID'), tactical_slot: 'RDM' },
      { ...player('LW', 'LW', 'FWD'), tactical_slot: 'LW' },
      { ...player('CAM', 'CAM', 'FWD'), tactical_slot: 'CAM' },
      { ...player('RW', 'RW', 'FWD'), tactical_slot: 'RW' },
      { ...player('ST', 'ST', 'FWD'), tactical_slot: 'ST' },
    ];
    const extras = [
      player('Extra-CB-1', 'CB', 'DEF'),
      player('Extra-CB-2', 'CB', 'DEF'),
      player('Extra-RW', 'RW', 'FWD'),
    ];

    const slots = assignFormationSlots([...xi, ...extras], '4-2-3-1');
    const reversed = assignFormationSlots([...extras, ...xi], '4-2-3-1');

    expect(slots).toHaveLength(11);
    expect(new Set(slots.map((slot) => slot.slot)).size).toBe(11);
    expect(new Set(slots.map((slot) => `${slot.x}:${slot.y}`)).size).toBe(11);
    expect(slots.find((slot) => slot.slot === 'RW')?.player.player_id).toBe('RW');
    expect(slots.find((slot) => slot.slot === 'LCB')?.x).toBe(38);
    expect(slots.find((slot) => slot.slot === 'RCB')?.x).toBe(62);
    expect(reversed.map((slot) => `${slot.slot}:${slot.x}:${slot.y}`).sort()).toEqual(
      slots.map((slot) => `${slot.slot}:${slot.x}:${slot.y}`).sort(),
    );
  });
});
