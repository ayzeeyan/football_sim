import React from 'react';
import type { TeamMatchStats } from '../../types';

function StatLine({ label, home, away, pct }: { label: string; home: number | string; away: number | string; pct?: number }) {
  const numericHome = typeof home === 'number' ? home : Number.parseFloat(String(home)) || 0;
  const numericAway = typeof away === 'number' ? away : Number.parseFloat(String(away)) || 0;
  const homeWidth = pct ?? (numericHome + numericAway > 0 ? (numericHome / (numericHome + numericAway)) * 100 : 50);
  return (
    <div className="py-2.5">
      <div className="flex items-center justify-between gap-3 text-[12px]">
        <strong className="w-12 shrink-0 text-left tabular-nums text-bone">{home}</strong>
        <span className="min-w-0 truncate text-center text-sage">{label}</span>
        <strong className="w-12 shrink-0 text-right tabular-nums text-bone">{away}</strong>
      </div>
      <div className="mt-1.5 flex h-1 overflow-hidden bg-white/10">
        <div className="bg-brass" style={{ width: `${Math.max(0, Math.min(100, homeWidth))}%` }} />
        <div className="flex-1 bg-pitchtone" />
      </div>
    </div>
  );
}

export const MatchStatComparison = React.memo(function MatchStatComparison({
  homeName,
  awayName,
  home,
  away,
  compact,
}: {
  homeName: string;
  awayName: string;
  home: TeamMatchStats;
  away: TeamMatchStats;
  compact?: boolean;
}) {
  const rows: Array<{ label: string; home: number | string; away: number | string; pct?: number; hide?: boolean }> = [
    { label: 'Expected goals', home: home.xg != null ? home.xg.toFixed(compact ? 1 : 2) : '—', away: away.xg != null ? away.xg.toFixed(compact ? 1 : 2) : '—', hide: home.xg == null && away.xg == null },
    { label: 'Possession', home: `${home.possession}%`, away: `${away.possession}%`, pct: home.possession },
    { label: 'Shots', home: home.shots, away: away.shots },
    { label: 'Shots on target', home: home.on_target, away: away.on_target },
    { label: 'Big chances', home: home.big_chances ?? 0, away: away.big_chances ?? 0, hide: compact || (!home.big_chances && !away.big_chances) },
    { label: 'Corners', home: home.corners, away: away.corners },
    { label: 'Pass accuracy', home: `${home.pass_accuracy}%`, away: `${away.pass_accuracy}%`, pct: home.pass_accuracy, hide: compact || (!home.pass_accuracy && !away.pass_accuracy) },
    { label: 'Fouls', home: home.fouls, away: away.fouls, hide: compact },
    { label: 'Yellow cards', home: home.yellows, away: away.yellows, hide: compact },
    { label: 'Red cards', home: home.reds, away: away.reds, hide: compact || (!home.reds && !away.reds) },
    { label: 'Saves', home: home.saves ?? 0, away: away.saves ?? 0, hide: compact || (!home.saves && !away.saves) },
  ];

  return (
    <div>
      <div className="grid grid-cols-3 text-center text-[12px]">
        <strong className="truncate text-bone">{homeName}</strong>
        <span className="text-[10px] uppercase tracking-[0.14em] text-sage">Match stats</span>
        <strong className="truncate text-bone">{awayName}</strong>
      </div>
      <div className="mt-1 divide-y divide-white/[0.07]">
        {rows.filter((row) => !row.hide).map((row) => (
          <StatLine key={row.label} label={row.label} home={row.home} away={row.away} pct={row.pct} />
        ))}
      </div>
    </div>
  );
});
