import React, { useMemo } from 'react';
import type { Player } from '../types';
import { cx } from '../lib/format';
import { ovrTone } from '../lib/constants';

interface FormationPitchProps {
  players: Player[];
  onPlayerClick?: (player: Player) => void;
}

interface PitchSlot {
  player: Player;
  x: number;
  y: number;
  slot: string;
}

export const EXACT_COORDS: Record<string, [number, number]> = {
  GK: [50, 91],
  LB: [16, 75],
  LWB: [16, 68],
  LCB: [38, 77],
  CB: [50, 77],
  RCB: [62, 77],
  RB: [84, 75],
  RWB: [84, 68],
  LDM: [35, 60],
  CDM: [50, 58],
  RDM: [65, 60],
  LM: [18, 52],
  LCM: [30, 52],
  CM: [50, 58],
  CAM: [50, 42],
  RCM: [70, 52],
  RM: [82, 52],
  LAM: [31, 42],
  RAM: [69, 42],
  LW: [18, 20],
  LF: [34, 22],
  CF: [50, 22],
  RF: [66, 22],
  RW: [82, 20],
  ST: [50, 18],
};

const GENERIC_COORDS: Record<string, Array<{ slot: string; coords: [number, number] }>> = {
  CB: [
    { slot: 'LCB', coords: [38, 77] },
    { slot: 'RCB', coords: [62, 77] },
    { slot: 'CB', coords: [50, 77] },
  ],
  CM: [
    { slot: 'LCM', coords: [30, 52] },
    { slot: 'RCM', coords: [70, 52] },
    { slot: 'CM', coords: [50, 58] },
  ],
  CDM: [
    { slot: 'CDM', coords: [50, 58] },
    { slot: 'LDM', coords: [35, 60] },
    { slot: 'RDM', coords: [65, 60] },
  ],
  CAM: [
    { slot: 'CAM', coords: [50, 42] },
    { slot: 'LAM', coords: [31, 42] },
    { slot: 'RAM', coords: [69, 42] },
  ],
  ST: [
    { slot: 'ST', coords: [50, 18] },
    { slot: 'LF', coords: [34, 22] },
    { slot: 'RF', coords: [66, 22] },
  ],
  CF: [
    { slot: 'CF', coords: [50, 22] },
    { slot: 'LF', coords: [34, 22] },
    { slot: 'RF', coords: [66, 22] },
  ],
};

function normalizedPosition(position: string): string {
  return position.trim().toUpperCase();
}

function fallbackCoords(player: Player, occurrence: number): [number, number] {
  switch (player.category) {
    case 'GK':
      return [50, 91];
    case 'DEF':
      return [[16, 75], [38, 77], [62, 77], [84, 75]][occurrence % 4] as [number, number];
    case 'MID':
      return [[30, 52], [50, 58], [70, 52]][occurrence % 3] as [number, number];
    default:
      return [[18, 20], [50, 18], [82, 20]][occurrence % 3] as [number, number];
  }
}

function assignFormationSlots(players: Player[]): PitchSlot[] {
  const positionUse = new Map<string, number>();
  const categoryUse = new Map<string, number>();
  const occupied = new Set<string>();

  const claim = (x: number, y: number): [number, number] => {
    let nx = x;
    let ny = y;
    let key = `${nx}:${ny}`;
    let attempt = 0;
    while (occupied.has(key) && attempt < 8) {
      const direction = attempt % 2 === 0 ? -1 : 1;
      const distance = 6 + Math.floor(attempt / 2) * 5;
      nx = Math.max(12, Math.min(88, x + direction * distance));
      ny = y + Math.floor(attempt / 4) * 4;
      key = `${nx}:${ny}`;
      attempt += 1;
    }
    occupied.add(key);
    return [nx, ny];
  };

  return players.map((player) => {
    const explicitSlot = player.slot ? normalizedPosition(player.slot) : null;
    if (explicitSlot && EXACT_COORDS[explicitSlot]) {
      const [x, y] = claim(EXACT_COORDS[explicitSlot][0], EXACT_COORDS[explicitSlot][1]);
      return { player, x, y, slot: explicitSlot };
    }

    const pos = normalizedPosition(player.position);
    const occurrence = positionUse.get(pos) ?? 0;
    positionUse.set(pos, occurrence + 1);

    let coords: [number, number] | undefined;
    let assignedSlot = pos;

    const generic = GENERIC_COORDS[pos];
    if (generic) {
      const item = generic[Math.min(occurrence, generic.length - 1)];
      coords = item.coords;
      assignedSlot = item.slot;
    } else if (EXACT_COORDS[pos]) {
      coords = EXACT_COORDS[pos];
    }

    if (!coords) {
      const catOccurrence = categoryUse.get(player.category) ?? 0;
      categoryUse.set(player.category, catOccurrence + 1);
      coords = fallbackCoords(player, catOccurrence);
    }

    const [x, y] = claim(coords[0], coords[1]);
    return { player, x, y, slot: assignedSlot || player.category };
  });
}

export const FormationPitch: React.FC<FormationPitchProps> = ({ players, onPlayerClick }) => {
  const slots = useMemo(() => assignFormationSlots(players), [players]);

  return (
    <div className="relative min-h-[520px] overflow-hidden rounded-2xl border border-pitchtone/30 bg-[#173d31] shadow-inner">
      <div className="absolute inset-3 rounded-xl border border-bone/25 pointer-events-none" />
      <div className="absolute left-1/2 top-3 bottom-3 border-l border-bone/20 pointer-events-none" />
      <div className="absolute left-1/2 top-1/2 h-28 w-28 -translate-x-1/2 -translate-y-1/2 rounded-full border border-bone/20 pointer-events-none" />
      <div className="absolute left-1/2 top-3 h-16 w-40 -translate-x-1/2 border-x border-b border-bone/20 pointer-events-none" />
      <div className="absolute left-1/2 bottom-3 h-16 w-40 -translate-x-1/2 border-x border-t border-bone/20 pointer-events-none" />

      {slots.map(({ player, x, y, slot }) => (
        <button
          key={player.player_id}
          type="button"
          onClick={() => onPlayerClick?.(player)}
          className="absolute w-[104px] -translate-x-1/2 -translate-y-1/2 rounded-xl border border-line/90 bg-ink/90 px-2 py-2 text-center shadow-raised transition-transform hover:scale-105 hover:border-brass focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brass"
          style={{ left: `${x}%`, top: `${y}%` }}
          aria-label={`${player.full_name}, ${slot}, ${player.ovr} overall`}
        >
          <span className="inline-block rounded bg-cardLight px-1.5 py-0.5 font-mono text-[10px] font-bold text-sage">
            {slot}
          </span>
          <p className="mt-1 truncate text-[12px] font-semibold text-bone" title={player.full_name}>
            {player.full_name.split(' ').slice(-1)[0]}
          </p>
          <p className={cx('mt-0.5 font-mono text-[13px] font-bold', ovrTone(player.ovr))}>{player.ovr}</p>
        </button>
      ))}
    </div>
  );
};

export { assignFormationSlots };
