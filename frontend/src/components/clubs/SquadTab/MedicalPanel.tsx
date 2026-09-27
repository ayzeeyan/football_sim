import React, { useEffect, useState } from 'react';
import { Activity } from 'lucide-react';
import type { Club, ClubMedicalResponse } from '../../../types';
import { fetchClubMedical } from '../../../services/api';
import { Card, EmptyState, LoadingState, OvrBadge } from '../../ui/ui';

/**
 * Club medical view (F11): current injuries with rehab roadmaps, the squad's
 * risk assessments from the injury-risk model, and the season's injury
 * history. Read-only — the medical staff report, nobody is treated by decree.
 */
export const MedicalPanel: React.FC<{ club: Club | null; onOpenPlayer: (id: string) => void }> = ({ club, onOpenPlayer }) => {
  const [data, setData] = useState<ClubMedicalResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!club) return;
    let cancelled = false;
    setLoading(true);
    setError('');
    fetchClubMedical(club.club_id)
      .then((res) => {
        if (!cancelled) setData(res);
      })
      .catch(() => {
        if (!cancelled) setError('The medical report could not be loaded.');
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [club?.club_id]);

  if (!club) return null;

  return (
    <div className="space-y-4">
      {loading && <LoadingState message="Reading the medical report…" />}
      {error && <EmptyState message={error} />}

      {data && !loading && !error && (
        <>
          <Card className="p-4 sm:p-5">
            <div className="flex items-center justify-between gap-3">
              <p className="match-section-title">Treatment room</p>
              <span className="flex items-center gap-1 text-[10px] uppercase tracking-[0.12em] text-sage">
                <Activity size={12} aria-hidden="true" /> {data.season}
              </span>
            </div>
            {data.injured.length === 0 ? (
              <p className="mt-3 text-[12px] text-sage">No current injuries — the full squad is available.</p>
            ) : (
              <div className="mt-3 space-y-3">
                {data.injured.map((row) => (
                  <div key={row.player_id} className="rounded-md border border-line bg-black/20 p-3">
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <button type="button" onClick={() => onOpenPlayer(row.player_id)} className="flex items-center gap-2 text-left font-semibold text-bone hover:text-brass">
                        {row.full_name}
                        <span className="font-mono text-[11px] text-sage">{row.position}</span>
                        <OvrBadge ovr={row.ovr} size="sm" />
                      </button>
                      <span className="text-[12px] text-brass">
                        {row.kind} · out {row.matches_out} {row.matches_out === 1 ? 'match' : 'matches'}
                      </span>
                    </div>
                    <div className="mt-2 flex flex-wrap gap-1.5">
                      {row.rehab.stages.map((stage, i) => (
                        <span key={stage.phase} className="rounded-full border border-line px-2 py-0.5 text-[10px] text-sage" title={stage.detail}>
                          {i + 1}. {stage.phase}
                        </span>
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </Card>

          <Card className="p-4 sm:p-5">
            <p className="match-section-title">Injury-risk assessments</p>
            <p className="mt-1 text-[11px] text-sage">Highest-risk available players, from load, age, fitness, and match density.</p>
            <div className="mt-3 overflow-x-auto">
              <table className="w-full min-w-[560px] text-left text-[12px]">
                <thead>
                  <tr className="border-b border-line text-[10px] uppercase tracking-[0.1em] text-sage">
                    <th className="py-2 pr-3">Player</th>
                    <th className="py-2 pr-3">Fitness</th>
                    <th className="py-2 pr-3">Risk</th>
                    <th className="py-2 pr-3">Factors</th>
                  </tr>
                </thead>
                <tbody>
                  {data.top_risks.map((row) => (
                    <tr key={row.player_id} className="border-b border-line/50">
                      <td className="py-2 pr-3">
                        <button type="button" onClick={() => onOpenPlayer(row.player_id)} className="font-semibold text-bone hover:text-brass">
                          {row.full_name}
                        </button>
                        <span className="ml-1.5 font-mono text-[10px] text-sage">{row.position}</span>
                      </td>
                      <td className="py-2 pr-3 font-mono text-sage">{row.fitness || '—'}</td>
                      <td className="py-2 pr-3">
                        <span className={row.assessment.risk_score >= 100 ? 'font-mono text-brass' : row.assessment.risk_score <= 50 ? 'font-mono text-emerald-400' : 'font-mono text-sage'}>
                          {row.assessment.risk_score}
                        </span>
                      </td>
                      <td className="py-2 pr-3 text-[11px] text-sage">
                        {row.assessment.factors.map((f) => f.label).join(' · ')}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </Card>

          <Card className="p-4 sm:p-5">
            <p className="match-section-title">Season injury history</p>
            <div className="mt-3 grid grid-cols-3 gap-3 text-center">
              <div className="rounded-md border border-line bg-black/20 p-3">
                <p className="font-mono text-[20px] font-bold text-bone">{data.history.total_injuries}</p>
                <p className="mt-1 text-[10px] uppercase tracking-[0.1em] text-sage">Injuries</p>
              </div>
              <div className="rounded-md border border-line bg-black/20 p-3">
                <p className="font-mono text-[20px] font-bold text-bone">{data.history.matches_lost}</p>
                <p className="mt-1 text-[10px] uppercase tracking-[0.1em] text-sage">Matches lost</p>
              </div>
              <div className="rounded-md border border-line bg-black/20 p-3">
                <p className="font-mono text-[20px] font-bold text-bone">
                  {data.history.by_severity.serious ?? 0}
                </p>
                <p className="mt-1 text-[10px] uppercase tracking-[0.1em] text-sage">Serious</p>
              </div>
            </div>
            <p className="mt-3 text-[11px] text-sage">
              Minor {data.history.by_severity.minor ?? 0} · Moderate {data.history.by_severity.moderate ?? 0} · Serious {data.history.by_severity.serious ?? 0}
            </p>
          </Card>
        </>
      )}
    </div>
  );
};
