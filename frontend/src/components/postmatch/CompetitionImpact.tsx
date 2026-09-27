import type { Club, Fixture, NearbyStanding, TableImpact } from '../../types';
import { prettyCompetitionName } from '../clubs/PlayerSheet';
import { cx } from '../../lib/format';

function TableSlice({
  nearby,
  highlight,
}: {
  nearby: NearbyStanding[];
  highlight: string[];
}) {
  return (
    <div className="mt-3 divide-y divide-white/[0.07]">
      <div className="grid grid-cols-[2.25rem_minmax(0,1fr)_3.5rem_3rem] gap-2 py-1 text-[10px] uppercase tracking-[0.12em] text-sage">
        <span>Pos</span><span>Club</span><span className="text-right">Pts</span><span className="text-right">GD</span>
      </div>
      {nearby.map((row) => {
        const active = highlight.includes(row.club_id);
        return (
          <div
            key={row.club_id}
            className={cx('grid grid-cols-[2.25rem_minmax(0,1fr)_3.5rem_3rem] gap-2 py-1.5 text-[12px]', active && 'bg-brass/10')}
          >
            <span className="tabular-nums text-sage">{row.pos}</span>
            <span className={cx('truncate', active ? 'font-semibold text-bone' : 'text-bone')}>{row.short_name}</span>
            <strong className="text-right tabular-nums text-bone">{row.pts}</strong>
            <span className="text-right tabular-nums text-sage">{row.gd >= 0 ? `+${row.gd}` : row.gd}</span>
          </div>
        );
      })}
    </div>
  );
}

export function CompetitionImpact({
  fixture,
  home,
  away,
}: {
  fixture?: Fixture | null;
  home: Club | null;
  away: Club | null;
}) {
  const impact = fixture?.table_impact as TableImpact | null | undefined;
  if (impact?.applicable) {
    const ids = [impact.home.club_id, impact.away.club_id];
    return (
      <div>
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          {[impact.home, impact.away].map((row) => (
            <p key={row.club_id} className="min-w-0 text-[13px] text-bone">
              <span className="font-semibold">{row.short_name}</span>
              <span className="text-sage"> {row.before_pos}{posSuffix(row.before_pos)} → {row.after_pos}{posSuffix(row.after_pos)}</span>
              <span className="block text-[11px] text-sage">{row.before_pts} pts → {row.after_pts} pts</span>
            </p>
          ))}
        </div>
        {impact.nearby?.length ? (
          <TableSlice nearby={impact.nearby} highlight={ids} />
        ) : (
          <p className="mt-3 text-[12px] text-sage">Nearby standings were not stored for this result.</p>
        )}
      </div>
    );
  }

  const copy = fixture?.competition_impact || fixture?.night?.story;
  if (!copy && !fixture?.night?.aggregate) {
    return <p className="mt-3 text-[12px] text-sage">No competition table applies to this fixture.</p>;
  }
  return (
    <div className="mt-3 space-y-2 text-[13px] text-bone">
      <p className="text-[10px] font-bold uppercase tracking-[0.14em] text-sage">
        {prettyCompetitionName(fixture?.competition || '')}
        {fixture?.stage ? ` · ${fixture.stage.replace(/_/g, ' ')}` : ''}
      </p>
      {copy ? <p>{copy}</p> : null}
      {fixture?.night?.aggregate ? <p className="text-sage">Aggregate {fixture.night.aggregate}</p> : null}
      {!copy && home && away ? <p>{home.club_name} {fixture?.home_goals}–{fixture?.away_goals} {away.club_name}.</p> : null}
    </div>
  );
}

function posSuffix(n: number): string {
  const v = n % 100;
  if (v >= 11 && v <= 13) return 'th';
  switch (n % 10) {
    case 1: return 'st';
    case 2: return 'nd';
    case 3: return 'rd';
    default: return 'th';
  }
}
