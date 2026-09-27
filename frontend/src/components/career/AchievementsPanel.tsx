import React, { useEffect, useState } from 'react';
import { Award, Lock } from 'lucide-react';
import type { AchievementsResponse } from '../../types';
import { fetchAchievements } from '../../services/api';
import { Card, EmptyState, LoadingState } from '../ui/ui';

/**
 * Milestone ledger (F6): the career-wide achievement store evaluated weekly
 * by the world itself — unbeaten runs, the treble, worst-to-champion, the
 * youngest-scorer record, 30-goal seasons, and mid-season sackings.
 * Observational: the world unlocks these; nobody spends points here.
 */
export const AchievementsPanel: React.FC = () => {
  const [data, setData] = useState<AchievementsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError('');
    fetchAchievements()
      .then((res) => {
        if (!cancelled) setData(res);
      })
      .catch(() => {
        if (!cancelled) setError('The milestone ledger could not be loaded.');
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  if (loading) return <LoadingState message="Loading milestone ledger…" />;
  if (error) return <EmptyState message={error} />;
  if (!data) return null;

  const unlockedByID = new Map<string, typeof data.achievements>();
  for (const entry of data.achievements) {
    const list = unlockedByID.get(entry.id) ?? [];
    list.push(entry);
    unlockedByID.set(entry.id, list);
  }

  return (
    <div className="space-y-4">
      {data.youngest_scorer_record && (
        <Card className="p-4">
          <p className="eyebrow !text-brass">World record</p>
          <h3 className="mt-1 font-display text-[20px] font-semibold text-bone">Youngest goalscorer</h3>
          <p className="mt-2 text-[13px] text-sage">
            <span className="font-semibold text-bone">{data.youngest_scorer_record.player_name}</span>{' '}
            scored at age {data.youngest_scorer_record.age} in {data.youngest_scorer_record.season}, week {data.youngest_scorer_record.matchweek}.
          </p>
        </Card>
      )}

      <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
        {data.definitions.map((def) => {
          const unlocked = unlockedByID.get(def.id) ?? [];
          const isUnlocked = def.unlocked && unlocked.length > 0;
          return (
            <Card key={def.id} className={isUnlocked ? 'p-4 border-brass/40' : 'p-4 opacity-80'}>
              <div className="flex items-start justify-between gap-3">
                <div className="flex items-center gap-2">
                  {isUnlocked ? (
                    <Award size={18} className="text-brass" aria-hidden="true" />
                  ) : (
                    <Lock size={16} className="text-sage" aria-hidden="true" />
                  )}
                  <h4 className="font-display text-[15px] font-semibold text-bone">{def.title}</h4>
                </div>
                <span className="rounded-full border border-line px-2 py-0.5 text-[9px] font-bold uppercase tracking-[0.12em] text-sage">
                  {def.kind}
                </span>
              </div>
              <p className="mt-2 text-[12px] leading-relaxed text-sage">{def.description}</p>
              {isUnlocked ? (
                <ul className="mt-3 space-y-1.5">
                  {unlocked.map((entry) => (
                    <li key={entry.unlock_key} className="text-[11px] text-sage">
                      <span className="font-semibold text-bone">{entry.subject_name}</span>
                      {' · '}
                      {entry.season} week {entry.matchweek}
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="mt-3 text-[11px] uppercase tracking-[0.1em] text-sage/70">Locked</p>
              )}
            </Card>
          );
        })}
      </div>

      {data.achievements.length === 0 && (
        <EmptyState message="No milestones unlocked yet — the ledger fills as the world plays." />
      )}
    </div>
  );
};
