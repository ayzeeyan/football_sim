import React from 'react';
import { cx } from '../../lib/format';
import { ATTACKING_ROLE_GROUPS, positionFit } from '../../lib/tactics';

export interface AttackingRoleMapPlayer {
  player_id: string;
  position: string;
  secondary_position?: string;
}

interface AttackingRoleMapProps {
  player: AttackingRoleMapPlayer;
}

const FIT_TONES: Record<string, string> = {
  Natural: 'border-brass/45 bg-brass/[0.10] text-brass',
  Good: 'border-emerald-400/30 bg-emerald-400/[0.08] text-emerald-200',
  Acceptable: 'border-sky-300/25 bg-sky-300/[0.07] text-sky-200',
  Emergency: 'border-amber-300/25 bg-amber-300/[0.06] text-amber-200',
};

/** A compact overview of a player's fit across the game's attacking tactical slots. */
export const AttackingRoleMap: React.FC<AttackingRoleMapProps> = ({ player }) => {
  const primaryPosition = player.position.trim().toUpperCase();
  const secondaryPosition = player.secondary_position?.trim().toUpperCase();

  return (
    <section
      aria-label="Attacking role map"
      className="rounded-xl border border-line bg-ink/45 p-3 sm:p-4"
    >
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <p className="text-[10px] font-semibold uppercase tracking-[0.14em] text-brass/75">Tactical versatility</p>
          <h3 className="mt-1 font-display text-[16px] font-semibold text-bone">Attacking role map</h3>
          <p className="mt-1 text-[11px] text-sage">Fit across all supported attacking roles.</p>
        </div>
        <div className="flex flex-wrap gap-1.5" aria-label="Player positions">
          <span className="rounded-md border border-brass/30 bg-brass/[0.06] px-2 py-1 font-mono text-[10px] font-semibold text-brass">
            Natural · {primaryPosition || '—'}
          </span>
          {secondaryPosition && secondaryPosition !== primaryPosition ? (
            <span className="rounded-md border border-white/10 bg-white/[0.03] px-2 py-1 font-mono text-[10px] text-sage">
              Secondary · {secondaryPosition}
            </span>
          ) : null}
        </div>
      </div>

      <div className="mt-3 grid gap-2 sm:grid-cols-3">
        {ATTACKING_ROLE_GROUPS.map(({ label, slots }) => (
          <div key={label} className="rounded-lg border border-white/[0.07] bg-black/15 p-2.5">
            <h4 className="mb-2 text-[9px] font-semibold uppercase tracking-[0.12em] text-sage/80">{label}</h4>
            <ul className="grid grid-cols-2 gap-1.5" aria-label={label}>
              {slots.map((slot) => {
                const fit = positionFit(player, slot);
                return (
                  <li
                    key={slot}
                    aria-label={`${slot}${slot === 'CF' ? ', common forward role' : ''}: ${fit} fit`}
                    className={cx(
                      'flex min-w-0 items-center justify-between gap-1 rounded-md border px-2 py-1.5',
                      FIT_TONES[fit] ?? FIT_TONES.Emergency,
                    )}
                  >
                    <span className="font-mono text-[11px] font-bold">{slot}</span>
                    <span className="truncate text-[9px] font-semibold">{fit}</span>
                  </li>
                );
              })}
            </ul>
          </div>
        ))}
      </div>
    </section>
  );
};

export default AttackingRoleMap;
