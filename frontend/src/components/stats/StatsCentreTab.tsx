import React, { useMemo, useState } from 'react';
import { BarChart3, TrendingUp } from 'lucide-react';
import { fetchAdvancedSeasonStats, type AdvancedPlayerStatRow } from '../../services/api';
import { useAsyncData } from '../../hooks/useAsyncData';
import { usePlayerSheet } from '../clubs/PlayerSheet';
import { Card, ErrorState, LoadingState, PanelHeader } from '../ui/ui';
import { cx } from '../../lib/format';

type SortKey = 'percentile' | 'goals' | 'assists' | 'avg_rating' | 'xg';

const SORTS: Array<{ key: SortKey; label: string }> = [
  { key: 'percentile', label: 'Contribution percentile' },
  { key: 'goals', label: 'Goals' },
  { key: 'assists', label: 'Assists' },
  { key: 'avg_rating', label: 'Average rating' },
  { key: 'xg', label: 'Expected goals' },
];

/**
 * Season statistics centre (Phase 3 F3): per-player season aggregates with
 * percentile ranks and shot-zone splits, plus per-club possession, passing,
 * xG, and territory trends. All data is aggregated by the backend from
 * resolved fixtures and authoritative player ledgers.
 */
export const StatsCentreTab: React.FC<{ careerKey: number }> = ({ careerKey }) => {
  const { openPlayer } = usePlayerSheet();
  const { data, loading, error, reload } = useAsyncData(fetchAdvancedSeasonStats, [careerKey]);
  const [sort, setSort] = useState<SortKey>('percentile');

  const players = useMemo(() => {
    const rows = [...(data?.players ?? [])];
    rows.sort((a, b) => {
      const av = a[sort];
      const bv = b[sort];
      if (av !== bv) return bv > av ? 1 : -1;
      return a.player_id.localeCompare(b.player_id);
    });
    return rows.slice(0, 25);
  }, [data?.players, sort]);

  if (loading) return <Card><LoadingState message="Aggregating the season…" /></Card>;
  if (error || !data) return <Card><ErrorState message={error || 'No statistics available.'} onRetry={reload} /></Card>;

  return (
    <div className="space-y-4 animate-fade-in" data-stats-centre="true">
      <Card>
        <PanelHeader
          kicker={`Season aggregates · ${data.season_name}`}
          title="Statistics centre"
          subtitle="Percentile ranks, shot-zone splits, and team trends compiled from every resolved fixture. Neutral viewing data only."
        />
      </Card>

      <Card>
        <div className="flex flex-wrap items-center justify-between gap-2 pb-3">
          <h3 className="font-display text-[16px] font-bold text-bone flex items-center gap-1.5">
            <BarChart3 size={15} className="text-brass" aria-hidden="true" /> Player leaderboard
          </h3>
          <div className="flex flex-wrap gap-1.5" role="group" aria-label="Sort players by">
            {SORTS.map(({ key, label }) => (
              <button
                key={key}
                type="button"
                onClick={() => setSort(key)}
                aria-pressed={sort === key}
                className={cx(
                  'min-h-9 border px-2.5 text-[12px] font-semibold transition-colors',
                  sort === key ? 'border-brass/60 bg-brass/15 text-brass' : 'border-line bg-cardLight text-sage hover:text-bone',
                )}
              >
                {label}
              </button>
            ))}
          </div>
        </div>
        {players.length === 0 ? (
          <p className="py-6 text-center text-[13px] text-sage">No appearances recorded yet — statistics appear once the season is underway.</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-[12px]">
              <caption className="sr-only">Season player statistics, ranked by {SORTS.find((s) => s.key === sort)?.label}</caption>
              <thead className="text-sage font-mono uppercase text-[10px] border-b border-line">
                <tr>
                  <th scope="col" className="px-2 py-2 text-left">Player</th>
                  <th scope="col" className="px-2 py-2 text-right">Apps</th>
                  <th scope="col" className="px-2 py-2 text-right">Min</th>
                  <th scope="col" className="px-2 py-2 text-right">G</th>
                  <th scope="col" className="px-2 py-2 text-right">A</th>
                  <th scope="col" className="px-2 py-2 text-right">Rating</th>
                  <th scope="col" className="px-2 py-2 text-right">Shots (box)</th>
                  <th scope="col" className="px-2 py-2 text-right">xG (box)</th>
                  <th scope="col" className="px-2 py-2 text-right">Pct</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-line/60">
                {players.map((row) => (
                  <PlayerStatLine key={row.player_id} row={row} onOpenPlayer={openPlayer} />
                ))}
              </tbody>
            </table>
          </div>
        )}
        <p className="mt-2 text-[11px] text-sage">{data.shot_detail_note}</p>
      </Card>

      <Card>
        <h3 className="font-display text-[16px] font-bold text-bone flex items-center gap-1.5 pb-3">
          <TrendingUp size={15} className="text-brass" aria-hidden="true" /> Team trends
        </h3>
        {data.clubs.length === 0 ? (
          <p className="py-6 text-center text-[13px] text-sage">No finished fixtures yet.</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-[12px]">
              <caption className="sr-only">Per-club season trends: possession, passing, shots, expected goals, and territory</caption>
              <thead className="text-sage font-mono uppercase text-[10px] border-b border-line">
                <tr>
                  <th scope="col" className="px-2 py-2 text-left">Club</th>
                  <th scope="col" className="px-2 py-2 text-right">P</th>
                  <th scope="col" className="px-2 py-2 text-right">Poss%</th>
                  <th scope="col" className="px-2 py-2 text-right">Pass%</th>
                  <th scope="col" className="px-2 py-2 text-right">Shots</th>
                  <th scope="col" className="px-2 py-2 text-right">xG</th>
                  <th scope="col" className="px-2 py-2 text-right">xGA</th>
                  <th scope="col" className="px-2 py-2 text-left">Territory (D/M/A)</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-line/60">
                {data.clubs.map((club) => (
                  <tr key={club.club_id}>
                    <td className="px-2 py-2 font-semibold text-bone">{club.short_name || club.club_name}</td>
                    <td className="px-2 py-2 text-right font-mono text-sage">{club.matches}</td>
                    <td className="px-2 py-2 text-right font-mono text-sage">{club.avg_possession.toFixed(0)}</td>
                    <td className="px-2 py-2 text-right font-mono text-sage">{club.avg_pass_accuracy.toFixed(0)}</td>
                    <td className="px-2 py-2 text-right font-mono text-sage">{club.avg_shots.toFixed(1)}</td>
                    <td className="px-2 py-2 text-right font-mono text-bone">{club.avg_xg.toFixed(2)}</td>
                    <td className="px-2 py-2 text-right font-mono text-sage">{club.avg_xg_against.toFixed(2)}</td>
                    <td className="px-2 py-2">
                      <span className="flex h-2 w-40 overflow-hidden rounded-full bg-white/10" aria-hidden="true">
                        <span className="bg-[#5B7FA6]" style={{ width: `${club.territory_defensive}%` }} />
                        <span className="bg-[#8E86C8]" style={{ width: `${club.territory_midfield}%` }} />
                        <span className="bg-pitchtone" style={{ width: `${club.territory_attacking}%` }} />
                      </span>
                      <span className="mt-0.5 block font-mono text-[10px] text-sage">
                        {club.territory_defensive.toFixed(0)} / {club.territory_midfield.toFixed(0)} / {club.territory_attacking.toFixed(0)}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>
    </div>
  );
};

const PlayerStatLine: React.FC<{ row: AdvancedPlayerStatRow; onOpenPlayer: (id: string) => void }> = ({ row, onOpenPlayer }) => (
  <tr className="hover:bg-white/[0.03]">
    <td className="px-2 py-2">
      <button type="button" onClick={() => onOpenPlayer(row.player_id)} className="text-left hover:text-brass">
        <span className="block font-semibold text-bone">{row.full_name}</span>
        <span className="block text-[10px] text-sage">{row.club_short} · {row.position} · {row.ovr} OVR</span>
      </button>
    </td>
    <td className="px-2 py-2 text-right font-mono text-sage">{row.appearances}</td>
    <td className="px-2 py-2 text-right font-mono text-sage">{row.minutes}</td>
    <td className="px-2 py-2 text-right font-mono text-bone">{row.goals}</td>
    <td className="px-2 py-2 text-right font-mono text-bone">{row.assists}</td>
    <td className="px-2 py-2 text-right font-mono text-sage">{row.avg_rating > 0 ? row.avg_rating.toFixed(1) : '—'}</td>
    <td className="px-2 py-2 text-right font-mono text-sage">{row.shots} ({row.shots_box})</td>
    <td className="px-2 py-2 text-right font-mono text-sage">{row.xg.toFixed(2)} ({row.xg_box.toFixed(2)})</td>
    <td className="px-2 py-2 text-right">
      <span className="inline-flex items-center gap-1.5">
        <span className="h-1.5 w-12 overflow-hidden rounded-full bg-white/10" aria-hidden="true">
          <span className="block h-full bg-brass" style={{ width: `${row.percentile}%` }} />
        </span>
        <span className="font-mono text-bone">{row.percentile}</span>
      </span>
    </td>
  </tr>
);
