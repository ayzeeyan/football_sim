import React, { useMemo } from 'react';
import type { Player } from '../../types';
import { cx } from '../../lib/format';
import { FORMATION_SLOTS, UNIT_TONE_CLASSES, isOutOfPosition, pitchCoords, positionFit, resolveLineupFormation, resolveTacticalAssignments, unitForSlot } from '../../lib/tactics';
import { AlertTriangle } from 'lucide-react';
import { PlayerPortrait } from '../ui/ui';

interface FormationPitchProps {
  players: Player[];
  formation?: string;
  onPlayerClick?: (player: Player) => void;
  compact?: boolean;
  /** Explicit slot assignments (Tier B lineup editor). When provided they
   * render as-is instead of the automatic tactical resolution. */
  assignments?: Array<{ slot: string; player: Player }>;
}

interface PitchSlot {
  player: Player;
  x: number;
  y: number;
  slot: string;
  naturalPosition: string;
  positionFit: Player['position_fit'];
}

function assignFormationSlots(players: Player[], formation?: string): PitchSlot[] {
  // The backend-assigned tactical slots are ground truth for the shape the
  // XI actually fields; the declared formation only disambiguates or
  // provides the fallback when a payload carries no slot information.
  const effectiveFormation = resolveLineupFormation(formation, players);
  const formationSlots = new Set(FORMATION_SLOTS[effectiveFormation]);
  const usedSlots = new Set<string>();
  const placed: PitchSlot[] = [];

  for (const { player, naturalPosition, tacticalSlot, positionFit } of resolveTacticalAssignments(players, formation)) {
    const coords = pitchCoords(effectiveFormation, tacticalSlot);
    if (!coords || !formationSlots.has(tacticalSlot) || usedSlots.has(tacticalSlot)) {
      continue;
    }
    usedSlots.add(tacticalSlot);
    placed.push({
      player,
      x: coords[0],
      y: coords[1],
      slot: tacticalSlot,
      naturalPosition,
      positionFit,
    });
  }

  return placed;
}

export const FormationPitch: React.FC<FormationPitchProps> = ({ players, formation, onPlayerClick, compact = false, assignments }) => {
  const slots = useMemo(() => {
    if (assignments && assignments.length > 0) {
      return assignments.map(({ slot, player }) => {
        const coords = pitchCoords(formation, slot) ?? [50, 50];
        return {
          player,
          x: coords[0],
          y: coords[1],
          slot,
          naturalPosition: player.position,
          positionFit: positionFit(player, slot),
        };
      });
    }
    return assignFormationSlots(players, formation);
  }, [players, formation, assignments]);

  return (
    <div
      className={cx(
        'relative overflow-hidden rounded-lg border border-[#3b6b55] shadow-inner shadow-black/40',
        compact ? 'h-[390px] sm:h-[430px]' : 'min-h-[430px]',
      )}
      style={{ background: 'repeating-linear-gradient(0deg, #154734 0%, #154734 12.5%, #113e2d 12.5%, #113e2d 25%)' }}
    >
      <div className="pointer-events-none absolute inset-2 rounded-sm border border-bone/35" />
      <div className="pointer-events-none absolute left-2 right-2 top-1/2 border-t border-bone/30" />
      <div className={cx('pointer-events-none absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 rounded-full border border-bone/30', compact ? 'h-16 w-16' : 'h-24 w-24')} />
      <div className="pointer-events-none absolute left-1/2 top-1/2 h-1.5 w-1.5 -translate-x-1/2 -translate-y-1/2 rounded-full bg-bone/35" />
      <div className="pointer-events-none absolute left-[24%] right-[24%] top-2 h-[19%] border border-t-0 border-bone/30" />
      <div className="pointer-events-none absolute left-[38%] right-[38%] top-2 h-[8%] border border-t-0 border-bone/30" />
      <div className="pointer-events-none absolute left-1/2 top-[15.5%] h-1.5 w-1.5 -translate-x-1/2 rounded-full bg-bone/35" />
      <div className="pointer-events-none absolute bottom-2 left-[24%] right-[24%] h-[19%] border border-b-0 border-bone/30" />
      <div className="pointer-events-none absolute bottom-2 left-[38%] right-[38%] h-[8%] border border-b-0 border-bone/30" />
      <div className="pointer-events-none absolute bottom-[15.5%] left-1/2 h-1.5 w-1.5 -translate-x-1/2 rounded-full bg-bone/35" />
      <div className="pointer-events-none absolute left-[43%] right-[43%] top-0.5 h-2 border-x border-b border-bone/30 bg-black/10" />
      <div className="pointer-events-none absolute bottom-0.5 left-[43%] right-[43%] h-2 border-x border-t border-bone/30 bg-black/10" />
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_50%_50%,rgba(255,255,255,.05),transparent_58%)]" />

      {slots.map(({ player, x, y, slot, naturalPosition, positionFit }) => {
        const outOfPosition = isOutOfPosition(positionFit);
        const unit = unitForSlot(slot);
        const tone = UNIT_TONE_CLASSES[unit];
        const surname = player.full_name.split(' ').slice(-1)[0];
        const isFormGood = (player.form ?? 0) >= 4 || player.form_band === 'Excellent';
        const isTired = typeof player.fitness === 'number' && player.fitness < 65;
        const isUnhappy = typeof player.morale === 'number' && player.morale < 55;

        return (
          <button
            key={player.player_id}
            type="button"
            onClick={() => onPlayerClick?.(player)}
            className={cx('group absolute z-10 -translate-x-1/2 -translate-y-1/2 text-center transition-transform hover:z-20 hover:scale-105 focus-visible:z-20 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brass', compact ? 'w-[64px]' : 'w-[78px]')}
            style={{ left: `${x}%`, top: `${y}%` }}
            aria-label={`${player.full_name}, natural ${naturalPosition}, playing ${slot}, ${positionFit}, ${player.ovr} overall`}
            title={`${player.full_name} · Natural: ${naturalPosition} · Playing: ${slot} · ${positionFit}${outOfPosition ? ' · Out of position' : ''}`}
          >
            <span className="relative mx-auto block w-fit transition-transform group-hover:-translate-y-0.5">
              <PlayerPortrait player={player} size={compact ? 26 : 32} className={cx('!rounded-full !border-2 shadow-lg shadow-black/40', tone.ring)} />
              <span className={cx('absolute -bottom-1.5 -right-2 min-w-[28px] rounded-full border px-1.5 py-0.5 font-mono text-[9px] font-extrabold shadow-md', player.ovr >= 88 ? 'border-amber-200/50 bg-brass text-ink' : player.ovr >= 82 ? 'border-emerald-200/40 bg-[#3e9d68] text-white' : player.ovr >= 76 ? 'border-sky-200/35 bg-[#3d6b8a] text-white' : 'border-white/15 bg-[#263b31] text-bone')}>
                {player.ovr}
              </span>
              <span className={cx('absolute -right-2 -top-1 h-2.5 w-2.5 rounded-full border border-black/40', isUnhappy ? 'bg-purple-400' : isTired ? 'bg-amber-400' : isFormGood ? 'bg-emerald-400' : 'bg-sage/60')} title={isUnhappy ? 'Unhappy' : isTired ? 'Fatigued' : isFormGood ? 'In form' : 'Available'} />
              {outOfPosition ? <AlertTriangle size={11} className="absolute -bottom-2 -left-2 fill-amber-400 text-amber-950" /> : null}
            </span>
            <span className="mt-2 flex items-center justify-center gap-1 rounded bg-[#06160f]/85 px-1 py-0.5 shadow-md backdrop-blur-sm">
              <span className={cx('shrink-0 rounded px-[3px] font-mono text-[8px] font-bold uppercase tracking-[0.06em]', tone.pill)}>{slot}</span>
              <span className="min-w-0 truncate text-[10px] font-bold text-bone group-hover:text-brass">{surname}</span>
            </span>
          </button>
        );
      })}
    </div>
  );
};

export { assignFormationSlots };
