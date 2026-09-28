import React, { useMemo } from 'react';
import { Table2 } from 'lucide-react';
import type { Fixture } from '../../types';
import { fetchCompetition } from '../../services/api';
import { useAsyncData } from '../../hooks/useAsyncData';
import { cx, formatGd } from '../../lib/format';
import { ClubCrest, FormPips } from '../ui/ui';

/** FotMob-style league context: a standings window around the two clubs. */
export const LeagueContextCard: React.FC<{ fixture: Fixture }> = ({ fixture }) => {
  const { data: detail, loading } = useAsyncData(
    () => fetchCompetition(fixture.competition),
    [fixture.competition],
  );

  const window = useMemo(() => {
    const table = detail?.table ?? [];
    if (table.length < 2) return null;
    const homeIdx = table.findIndex((row) => row.club_id === fixture.home.club_id);
    const awayIdx = table.findIndex((row) => row.club_id === fixture.away.club_id);
    if (homeIdx < 0 || awayIdx < 0) return null;
    const lo = Math.max(0, Math.min(homeIdx, awayIdx) - 2);
    const hi = Math.min(table.length - 1, Math.max(homeIdx, awayIdx) + 2);
    return {
      rows: table.slice(lo, hi + 1).map((row, i) => ({ row, pos: lo + i + 1 })),
      homeIdx,
      awayIdx,
      gap: Math.abs(homeIdx - awayIdx),
    };
  }, [detail, fixture.home.club_id, fixture.away.club_id]);

  if (loading) {
    return (
      <section className="console-card p-4">
        <h3 className="match-section-title"><Table2 size={14} /> League context</h3>
        <p className="mt-3 text-[12px] text-sage">Loading standings…</p>
      </section>
    );
  }

  if (!window) return null;

  return (
    <section className="console-card overflow-hidden">
      <div className="flex items-center justify-between gap-3 border-b border-line bg-cardLight/50 px-4 py-3">
        <div>
          <h3 className="match-section-title"><Table2 size={14} /> League context</h3>
          <p className="mt-1 text-[10px] text-sage">
            {window.gap === 0 ? 'Level on position coming in' : `${window.gap + 1} places apart coming in`}
          </p>
        </div>
        <span className="font-mono text-[10px] uppercase tracking-[0.1em] text-sage">{detail?.name}</span>
      </div>
      <div className="overflow-x-auto">
      <table className="w-full text-left text-[12px]">
        <thead className="table-head">
          <tr>
            <th className="px-3 py-2">#</th>
            <th className="px-3 py-2">Club</th>
            <th className="px-2 py-2 text-right">P</th>
            <th className="px-2 py-2 text-right">GD</th>
            <th className="px-3 py-2 text-right">PTS</th>
            <th className="px-3 py-2 text-right">Form</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-line/70">
          {window.rows.map(({ row, pos }) => {
            const inFixture = pos - 1 === window.homeIdx || pos - 1 === window.awayIdx;
            return (
              <tr key={row.club_id} className={cx('transition-colors', inFixture ? 'bg-brass/[0.07]' : 'hover:bg-cardLight/50')}>
                <td className={cx('px-3 py-2 font-mono', inFixture ? 'font-bold text-brass' : 'text-sage')}>{pos}</td>
                <td className="px-3 py-2">
                  <span className="flex min-w-0 items-center gap-2">
                    <ClubCrest club={row.club} size={16} className="!border-0 !bg-transparent" />
                    <span className={cx('truncate', inFixture ? 'font-semibold text-bone' : 'text-bone/85')}>{row.club?.club_name ?? row.club_id}</span>
                  </span>
                </td>
                <td className="px-2 py-2 text-right font-mono text-sage">{row.played}</td>
                <td className="px-2 py-2 text-right font-mono text-sage">{formatGd(row.goal_difference)}</td>
                <td className={cx('px-3 py-2 text-right font-mono font-bold', inFixture ? 'text-brass' : 'text-bone')}>{row.points}</td>
                <td className="px-3 py-2 text-right">{row.form?.length ? <FormPips form={row.form} size="sm" /> : <span className="text-[10px] text-sage">—</span>}</td>
              </tr>
            );
          })}
        </tbody>
      </table>
      </div>
    </section>
  );
};
