import React, { useEffect, useState } from 'react';
import { ClipboardList } from 'lucide-react';
import type { Club, ScoutingReport } from '../../../types';
import { fetchClubScouting } from '../../../services/api';
import { Card, ClubCrest, EmptyState, LoadingState, OvrBadge } from '../../ui/ui';
import { downloadCSV } from '../../../lib/export';
import { t } from '../../../i18n';

/**
 * Recruitment desk (F5): the AI scouting department's deterministic shortlist
 * for one club. Observational only — the staff report on the market; nobody
 * signs anything here.
 */
export const RecruitmentPanel: React.FC<{ club: Club | null; onOpenPlayer: (id: string) => void }> = ({ club, onOpenPlayer }) => {
  const [shortlist, setShortlist] = useState<ScoutingReport[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!club) return;
    let cancelled = false;
    setLoading(true);
    setError('');
    fetchClubScouting(club.club_id)
      .then((res) => {
        if (!cancelled) setShortlist(res?.shortlist ?? []);
      })
      .catch(() => {
        if (!cancelled) setError('The scouting desk could not compile a shortlist.');
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
    <Card className="p-4 sm:p-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p className="match-section-title">Recruitment desk</p>
          <p className="mt-1 text-[12px] text-sage">
            The {club.short_name} scouting department's current shortlist — ranked by ceiling, consistency, and risk. Observational: the AI staff decide any actual moves.
          </p>
        </div>
        {shortlist.length > 0 && (
          <button
            type="button"
            onClick={() =>
              downloadCSV(`scouting-${club.club_id.toLowerCase()}`, shortlist.map((r) => ({
                player: r.full_name, position: r.position, age: r.age, ovr: r.ovr,
                ceiling: r.potential_ceiling, consistency: r.consistency, form: r.form,
                value_trend: r.value_trend, risk: r.risk, verdict: r.verdict,
                club: r.club_short, region: r.region, value_eur: r.market_value_eur,
              })))
            }
            className="min-h-9 border border-line bg-cardLight px-3 text-[12px] font-semibold text-sage hover:border-brass/30 hover:text-bone"
          >
            {t('action.exportCsv')}
          </button>
        )}
      </div>

      {loading && <LoadingState message="Compiling shortlist…" />}
      {error && <EmptyState message={error} />}
      {!loading && !error && shortlist.length === 0 && (
        <EmptyState message="No shortlist available for this club yet." />
      )}

      {shortlist.length > 0 && (
        <div className="mt-4 overflow-x-auto">
          <table className="w-full min-w-[860px] text-left text-[12px]">
            <thead>
              <tr className="border-b border-line text-[10px] uppercase tracking-[0.1em] text-sage">
                <th className="py-2 pr-3">Player</th>
                <th className="py-2 pr-3">Pos</th>
                <th className="py-2 pr-3">Age</th>
                <th className="py-2 pr-3">OVR</th>
                <th className="py-2 pr-3">Ceiling</th>
                <th className="py-2 pr-3">Consist.</th>
                <th className="py-2 pr-3">Form</th>
                <th className="py-2 pr-3">Value</th>
                <th className="py-2 pr-3">Risk</th>
                <th className="py-2 pr-3">Verdict</th>
              </tr>
            </thead>
            <tbody>
              {shortlist.map((r) => (
                <tr key={r.player_id} className="border-b border-line/50 align-middle">
                  <td className="py-2 pr-3">
                    <button type="button" onClick={() => onOpenPlayer(r.player_id)} className="text-left font-semibold text-bone hover:text-brass">
                      {r.full_name}
                    </button>
                    <span className="mt-0.5 flex items-center gap-1 text-[10px] text-sage">
                      <ClubCrest club={{ club_id: r.club_id, club_name: r.club_name, short_name: r.club_short }} size={12} />
                      {r.club_short} · {r.region}
                    </span>
                  </td>
                  <td className="py-2 pr-3 text-sage">{r.position}</td>
                  <td className="py-2 pr-3 text-sage">{r.age}</td>
                  <td className="py-2 pr-3"><OvrBadge ovr={r.ovr} size="sm" /></td>
                  <td className="py-2 pr-3">
                    <span className="font-mono font-semibold text-bone">{r.potential_ceiling}</span>
                    {r.ceiling_delta > 0 && <span className="ml-1 text-[10px] text-emerald-400">+{r.ceiling_delta}</span>}
                  </td>
                  <td className="py-2 pr-3 font-mono text-sage">{r.consistency}</td>
                  <td className="py-2 pr-3">
                    <span className={r.form === 'hot' ? 'text-emerald-400' : r.form === 'cold' ? 'text-brass' : 'text-sage'}>{r.form}</span>
                  </td>
                  <td className="py-2 pr-3">
                    <span className="font-mono text-sage">€{Math.round(r.market_value_eur / 1_000_000)}M</span>
                    <span className="ml-1 text-[10px] text-sage">({r.value_trend})</span>
                  </td>
                  <td className="py-2 pr-3">
                    <span className={r.risk >= 70 ? 'font-mono text-brass' : r.risk <= 35 ? 'font-mono text-emerald-400' : 'font-mono text-sage'}>{r.risk}</span>
                  </td>
                  <td className="py-2 pr-3 text-sage">
                    <span className="flex items-center gap-1"><ClipboardList size={11} aria-hidden="true" />{r.verdict}</span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </Card>
  );
};
