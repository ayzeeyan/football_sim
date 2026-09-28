import React, { useEffect, useState } from 'react';
import { ClipboardCheck, Dumbbell, Swords, Zap } from 'lucide-react';
import { fetchTrainingProjection, type TrainingProjection } from '../../services/api';
import { TRAINING_FOCUSES, type TrainingFocus } from '../../lib/constants';
import { cx } from '../../lib/format';

const FOCUS_ICONS: Record<TrainingFocus, React.ElementType> = {
  hypertrophy: Dumbbell,
  technical: Zap,
  tactical: Swords,
};

export interface TrainingPlannerProps {
  /** The player whose staff-run plan is projected. */
  playerId?: string | null;
}

/**
 * Read-only staff training plan (F1). Every player is machine-managed: club
 * staff decide the regimen, and this panel projects what they do with the
 * week's training energy. The viewer never directs a session.
 */
export const TrainingPlanner: React.FC<TrainingPlannerProps> = ({ playerId }) => {
  const [projection, setProjection] = useState<TrainingProjection | null>(null);

  useEffect(() => {
    setProjection(null);
    if (!playerId) return;
    let cancelled = false;
    fetchTrainingProjection(playerId)
      .then((proj) => { if (!cancelled) setProjection(proj); })
      .catch(() => undefined);
    return () => { cancelled = true; };
  }, [playerId]);

  const energy = projection?.training_energy ?? 0;
  const maxEnergy = projection?.max_training_energy ?? 3;
  const staffFocus = projection?.focus ?? null;

  return (
    <div className="bg-ink/40 border border-line rounded-xl p-4 space-y-3" data-training-planner="true">
      <div className="flex items-center justify-between">
        <div className="text-[13px] font-semibold text-bone flex items-center gap-1.5">
          <ClipboardCheck size={15} className="text-brass" />
          <span>Managed by club staff</span>
        </div>
        <span className="text-[12px] font-mono text-sage font-semibold" aria-label="Training energy">
          {energy}/{maxEnergy} sessions
        </span>
      </div>

      <div className="flex h-1.5 overflow-hidden rounded-full bg-white/10" aria-hidden="true">
        <span className="bg-brass" style={{ width: `${maxEnergy > 0 ? (energy / maxEnergy) * 100 : 0}%` }} />
      </div>

      <div className="flex flex-wrap gap-1.5" role="group" aria-label="Training regimens">
        {TRAINING_FOCUSES.map(({ value, label }) => {
          const Icon = FOCUS_ICONS[value];
          const active = staffFocus === value;
          return (
            <span
              key={value}
              aria-current={active ? 'true' : undefined}
              className={cx(
                'flex min-h-9 items-center gap-1.5 border px-2.5 text-[12px] font-semibold',
                active ? 'border-brass/60 bg-brass/15 text-brass' : 'border-line bg-cardLight text-sage opacity-70',
              )}
            >
              <Icon size={13} aria-hidden="true" /> {label}
            </span>
          );
        })}
      </div>

      <div className="space-y-1.5 text-[12px] text-sage leading-relaxed">
        {projection ? (
          <>
            <p>{projection.rationale}</p>
            <ul className="list-disc pl-4 text-bone/90">
              {projection.projected_gains.map((gain) => <li key={gain}>{gain}</li>)}
            </ul>
            <p className="text-[11px] italic">Every player is staff-managed; the plan above is a read-only projection.</p>
          </>
        ) : (
          <p>Club staff run occasional sessions, rotating hypertrophy, technical, and tactical work.</p>
        )}
      </div>
    </div>
  );
};
