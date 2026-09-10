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

const EXACT_COORDS: Record<string, [number, number]> = {
  GK: [50, 91],
  LB: [17, 76],
  LWB: [13, 66],
  LCB: [38, 76],
  RCB: [62, 76],
  RB: [83, 76],
  RWB: [87, 66],
  LDM: [38, 62],
  CDM: [50, 62],
  RDM: [62, 62],
  LM: [18, 50],
  LCM: [38, 51],
  RCM: [62, 51],
  RM: [82, 50],
  LAM: [31, 38],
  CAM: [50, 38],
  RAM: [69, 38],
  LW: [17, 20],
  LF: [34, 24],
  CF: [50, 24],
  RF: [66, 24],
  RW: [83, 20],
};

const GENERIC_COORDS: Record<string, Array<[number, number]>> = {
  CB: [[38, 76], [62, 76], [50, 77]],
  CM: [[38, 51], [62, 51], [50, 51]],
  CDM: [[38, 62], [62, 62], [50, 62]],
  CAM: [[50, 38], [38, 39], [62, 39]],
  ST: [[50, 18], [39, 20], [61, 20]],
  CF: [[50, 24], [39, 25], [61, 25]],
};

function normalizedPosition(position: string): string {
  return position.trim().toUpperCase();
}

function fallbackCoords(player: Player, occurrence: number): [number, number] {
  switch (player.category) {
    case 'GK':
      return [50, 91];
    case 'DEF':
      return [[25, 76], [42, 76], [58, 76], [75, 76]][occurrence % 4] as [number, number];
    case 'MID':
      return [[31, 51], [50, 51], [69, 51]][occurrence % 3] as [number, number];
    default:
      return [[22, 21], [50, 18], [78, 21]][occurrence % 3] as [number, number];
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
      const distance = 5 + Math.floor(attempt / 2) * 4;
      nx = Math.max(8, Math.min(92, x + direction * distance));
      ny = y + Math.floor(attempt / 4) * 3;
      key = `${nx}:${ny}`;
      attempt += 1;
    }
    occupied.add(key);
    return [nx, ny];
  };

  return players.map((player) => {
    const pos = normalizedPosition(player.position);
    const occurrence = positionUse.get(pos) ?? 0;
    positionUse.set(pos, occurrence + 1);

    let coords: [number, number] | undefined;
    const generic = GENERIC_COORDS[pos];
    if (generic) {
      coords = generic[Math.min(occurrence, generic.length - 1)];
    } else if (EXACT_COORDS[pos]) {
      coords = EXACT_COORDS[pos];
    }

    if (!coords) {
      const catOccurrence = categoryUse.get(player.category) ?? 0;
      categoryUse.set(player.category, catOccurrence + 1);
      coords = fallbackCoords(player, catOccurrence);
    }

    const [x, y] = claim(coords[0], coords[1]);
    return { player, x, y, slot: pos || player.category };
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
