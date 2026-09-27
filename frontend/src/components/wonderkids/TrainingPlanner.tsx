import React, { useEffect, useState } from 'react';
import { ClipboardCheck, Dumbbell, Swords, Zap } from 'lucide-react';
import type { ProdigyData } from '../../types';
import { fetchTrainingProjection, trainProdigy, type TrainProdigyResponse, type TrainingProjection } from '../../services/api';
import { TRAINING_FOCUSES, type TrainingFocus } from '../../lib/constants';
import { cx } from '../../lib/format';
import { soundManager } from '../../audio/webAudio';

const FOCUS_ICONS: Record<TrainingFocus, React.ElementType> = {
  hypertrophy: Dumbbell,
  technical: Zap,
  tactical: Swords,
};

const FOCUS_BLURBS: Record<TrainingFocus, string> = {
  hypertrophy: 'Strength, stamina, and lean mass while the body develops.',
  technical: 'First touch, finishing, and composure reps.',
  tactical: 'Position-specific shape, scanning, and decision-making.',
};

export interface TrainingPlannerProps {
  /** The selected prodigy; when absent the planner renders the read-only
   *  staff projection for `playerId` instead (any squad player). */
  prodigy?: ProdigyData | null;
  playerId?: string | null;
  onShowToast: (msg: string) => void;
  /** Called after a successful session so the parent can refresh prodigy data. */
  onTrained?: () => void;
}

/**
 * Per-player training planner (Phase 3 F1). Prodigies get the interactive
 * planner: pick a regimen from TRAINING_FOCUSES, spend one of the week's
 * energy units, and see the recorded gains. Non-prodigies render the
 * read-only "what the staff do" projection from the backend.
 */
export const TrainingPlanner: React.FC<TrainingPlannerProps> = ({ prodigy, playerId, onShowToast, onTrained }) => {
  const [focus, setFocus] = useState<TrainingFocus>('technical');
  const [busy, setBusy] = useState(false);
  const [lastResult, setLastResult] = useState<TrainProdigyResponse | null>(null);
  const [projection, setProjection] = useState<TrainingProjection | null>(null);

  const effectiveId = prodigy?.player_id ?? playerId ?? null;
  const energy = prodigy?.training_energy ?? projection?.training_energy ?? 0;
  const maxEnergy = prodigy?.max_training_energy ?? projection?.max_training_energy ?? 3;
  const trainable = !!prodigy;

  useEffect(() => {
    setLastResult(null);
    if (!effectiveId || trainable) return;
    let cancelled = false;
    fetchTrainingProjection(effectiveId)
      .then((proj) => { if (!cancelled) setProjection(proj); })
      .catch(() => undefined);
    return () => { cancelled = true; };
  }, [effectiveId, trainable]);

  const runSession = async () => {
    if (!prodigy || busy) return;
    soundManager.playClick();
    setBusy(true);
    try {
      const res = await trainProdigy(prodigy.player_id, focus);
      if (res.status === 'error') {
        onShowToast(res.message || 'No training energy remaining this matchweek.');
      } else {
        setLastResult(res);
        onShowToast(res.message || 'Training session complete.');
        onTrained?.();
      }
    } catch {
      onShowToast('The training ground is unreachable right now.');
    } finally {
      setBusy(false);
    }
  };

  const staffFocus = projection?.focus ?? null;

  return (
    <div className="bg-ink/40 border border-line rounded-xl p-4 space-y-3" data-training-planner="true">
      <div className="flex items-center justify-between">
        <div className="text-[13px] font-semibold text-bone flex items-center gap-1.5">
          <ClipboardCheck size={15} className="text-brass" />
          <span>{trainable ? 'Training planner' : 'Managed by club staff'}</span>
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
          const active = trainable ? focus === value : staffFocus === value;
          return (
            <button
              key={value}
              type="button"
              disabled={!trainable || busy || energy <= 0}
              onClick={() => { soundManager.playClick(); setFocus(value); }}
              aria-pressed={active}
              title={FOCUS_BLURBS[value]}
              className={cx(
                'flex min-h-9 items-center gap-1.5 border px-2.5 text-[12px] font-semibold transition-colors',
                active ? 'border-brass/60 bg-brass/15 text-brass' : 'border-line bg-cardLight text-sage hover:border-brass/30 hover:text-bone',
                (!trainable || busy) && 'cursor-default opacity-80',
              )}
            >
              <Icon size={13} aria-hidden="true" /> {label}
            </button>
          );
        })}
      </div>

      {trainable ? (
        <>
          <p className="text-[12px] text-sage leading-relaxed">{FOCUS_BLURBS[focus]}</p>
          <button
            type="button"
            onClick={() => void runSession()}
            disabled={busy || energy <= 0}
            className="gold-btn w-full disabled:opacity-50"
          >
            {busy ? 'Running session…' : energy <= 0 ? 'No energy — advance the matchweek' : `Run ${TRAINING_FOCUSES.find((f) => f.value === focus)?.label ?? ''} session (1 energy)`}
          </button>
          {lastResult && lastResult.status !== 'error' && (
            <div className="rounded-md border border-brass/25 bg-brass/[0.06] p-2.5 text-[12px]" data-training-gains="true">
              <p className="text-[10px] font-bold uppercase tracking-[0.12em] text-brass">Session gains</p>
              <ul className="mt-1 space-y-0.5 text-bone">
                {lastResult.height_gain != null && lastResult.height_gain > 0 && <li>Height +{lastResult.height_gain.toFixed(1)} cm</li>}
                {lastResult.weight_gain != null && lastResult.weight_gain > 0 && <li>Weight +{lastResult.weight_gain.toFixed(1)} kg</li>}
                {lastResult.remaining_energy != null && <li className="text-sage">Energy left: {lastResult.remaining_energy}/{lastResult.max_energy ?? maxEnergy}</li>}
              </ul>
            </div>
          )}
        </>
      ) : (
        <div className="space-y-1.5 text-[12px] text-sage leading-relaxed">
          {projection ? (
            <>
              <p>{projection.rationale}</p>
              <ul className="list-disc pl-4 text-bone/90">
                {projection.projected_gains.map((gain) => <li key={gain}>{gain}</li>)}
              </ul>
              <p className="text-[11px] italic">This player is staff-managed; the plan above is a read-only projection.</p>
            </>
          ) : (
            <p>Club staff run occasional sessions, rotating hypertrophy, technical, and tactical work.</p>
          )}
        </div>
      )}
    </div>
  );
};
