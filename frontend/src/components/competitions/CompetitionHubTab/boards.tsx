import React, { useMemo, useState } from 'react';
import { CalendarDays, Crown, Trophy } from 'lucide-react';
import { soundManager } from '../../../audio/webAudio';
import type { CompetitionDetail, CompetitionFixtureRow, CompetitionTableRow, NationsCupFixture, NationsCupResponse } from '../../../types';
import { formatGd, cx } from '../../../lib/format';
import { qualificationBand, qualificationBarClass, qualificationLabel, type QualBand } from '../../../lib/qualification';
import { Card, ClubCrest, ClubDot } from '../../ui/ui';
import { NationsMatchModal } from '../NationsMatchModal';
import { asClub, scoreLine, legLabel, isWatchable } from './helpers';
import { tieGroups, aggregateLine, tieWinnerId } from './ties';

export const NationsCupBoard: React.FC<{ data: NationsCupResponse; onRefresh?: () => void }> = ({ data, onRefresh }) => {
  const [matchFixture, setMatchFixture] = useState<NationsCupFixture | null>(null);
  const teams = useMemo(() => new Map(data.participants.map((team) => [team.id, team])), [data.participants]);
  const teamLabel = (id: string, compact?: { name: string; country: string } | null) => compact?.name ?? teams.get(id)?.name ?? id;

  return (
    <div className="space-y-4" data-international-competition="nations-cup">
      <Card>
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div className="min-w-0">
            <p className="eyebrow text-brass">International · {data.country}</p>
            <h3 className="mt-1 font-display text-[30px] font-semibold text-bone">{data.name}</h3>
            <p className="mt-1 text-[13px] text-sage">{data.stage} · {data.participants.length} national teams · results follow the shared season calendar</p>
          </div>
          {data.champion ? (
            <div className="flex items-center gap-3 border border-brass/40 bg-brass/[0.08] px-3 py-2">
              <Crown size={17} className="text-brass" aria-hidden="true" />
              <div><p className="text-[10px] font-mono uppercase tracking-[0.12em] text-brass">Champion</p><p className="text-[14px] font-semibold text-bone">{data.champion.name}</p></div>
            </div>
          ) : <div className="inline-flex items-center gap-2 rounded-full border border-pitchtone/25 bg-pitchtone/10 px-3 py-1.5 text-[11px] font-semibold text-pitchtone">Season underway</div>}
        </div>
      </Card>

      <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
        <section className="console-card min-w-0 overflow-hidden">
          <div className="flex items-center justify-between gap-3 border-b border-line bg-cardLight/50 px-4 py-3">
            <div><h4 className="text-[14px] font-semibold text-bone">Nations table</h4><p className="mt-0.5 text-[11px] text-sage">Group stage standings</p></div>
            <Trophy size={16} className="text-brass" aria-hidden="true" />
          </div>
          <div className="overflow-x-auto">
            <table className="w-full min-w-[500px] text-left text-[12px]">
              <thead className="table-head"><tr><th className="px-3 py-2">#</th><th className="px-3 py-2">Nation</th>{['P', 'W', 'D', 'L', 'GD', 'PTS'].map((head) => <th key={head} className="px-2 py-2 text-right">{head}</th>)}</tr></thead>
              <tbody className="divide-y divide-line/70">
                {data.table.length === 0 ? <tr><td colSpan={8} className="px-4 py-8 text-center text-sage">Standings appear after the opening international round.</td></tr> : data.table.map((row, index) => {
                  const team = teams.get(row.team_id);
                  const name = team?.name ?? row.name ?? row.team_id;
                  return <tr key={row.team_id} className={cx(index === 0 && 'bg-brass/[0.05]')}>
                    <td className="px-3 py-2.5 font-mono text-sage">{index + 1}</td>
                    <td className="px-3 py-2.5"><span className="block font-semibold text-bone">{name}</span><span className="text-[10px] text-sage">{team?.country ?? row.country ?? 'Europe'} · {team?.rating ?? row.rating ?? '—'} rating</span></td>
                    {[row.played, row.won, row.drawn, row.lost, row.goal_difference, row.points].map((value, cell) => <td key={cell} className={cx('px-2 py-2.5 text-right font-mono tabular-nums', cell === 5 ? 'font-bold text-bone' : 'text-sage')}>{cell === 4 && value > 0 ? `+${value}` : value}</td>)}
                  </tr>;
                })}
              </tbody>
            </table>
          </div>
        </section>

        <section className="console-card min-w-0 overflow-hidden">
          <div className="flex items-center justify-between gap-3 border-b border-line bg-cardLight/50 px-4 py-3">
            <div><h4 className="text-[14px] font-semibold text-bone">Fixtures and results</h4><p className="mt-0.5 text-[11px] text-sage">Open any match for its full international report</p></div>
            <CalendarDays size={16} className="text-brass" aria-hidden="true" />
          </div>
          <ul className="max-h-[34rem] divide-y divide-line/70 overflow-y-auto">
            {data.fixtures.length === 0 ? <li className="px-4 py-8 text-center text-[12px] text-sage">The international schedule has not been drawn yet.</li> : data.fixtures.map((fixture) => {
              const finished = fixture.status === 'finished';
              const home = teamLabel(fixture.home_id, fixture.home);
              const away = teamLabel(fixture.away_id, fixture.away);
              const line = finished && fixture.home_goals != null && fixture.away_goals != null ? `${fixture.home_goals}–${fixture.away_goals}` : 'vs';
              const pens = finished && fixture.home_penalties != null && fixture.away_penalties != null ? ` · Pens ${fixture.home_penalties}–${fixture.away_penalties}` : '';
              return <li key={fixture.id}>
                <button type="button" onClick={() => { soundManager.playClick(); setMatchFixture(fixture); }} className="grid w-full grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-3 px-4 py-3 text-left transition-colors hover:bg-cardLight/40">
                  <span className="min-w-[54px] text-[10px] font-mono text-sage">MW {fixture.matchweek}</span>
                  <span className="min-w-0"><span className="block truncate text-[12px] font-semibold text-bone">{home} <span className="px-1 font-mono text-brass">{line}</span> {away}</span><span className="block truncate text-[10px] text-sage">{fixture.stage}{pens}</span></span>
                  <span className={cx('text-[9px] font-bold uppercase tracking-[0.1em]', finished ? 'text-pitchtone' : 'text-sage/70')}>{finished ? 'FT' : 'Scheduled'}</span>
                </button>
              </li>;
            })}
          </ul>
        </section>
      </div>

      <section className="console-card overflow-hidden">
        <div className="border-b border-line bg-cardLight/50 px-4 py-3"><h4 className="text-[14px] font-semibold text-bone">National squads</h4><p className="mt-0.5 text-[11px] text-sage">Player lists are informational; club careers and statistics remain separate.</p></div>
        <div className="grid grid-cols-1 gap-2 p-3 sm:grid-cols-2 xl:grid-cols-3">
          {data.participants.map((team) => (
            <details key={team.id} className="rounded border border-line bg-ink/30">
              <summary className="flex cursor-pointer list-none items-center justify-between gap-3 px-3 py-2.5 marker:hidden">
                <span className="min-w-0"><strong className="block truncate text-[12px] text-bone">{team.name}</strong><span className="text-[10px] text-sage">{team.country}</span></span>
                <span className="shrink-0 text-right"><strong className="block font-mono text-[12px] text-brass">{team.rating}</strong><span className="text-[9px] text-sage">{team.players.length} players</span></span>
              </summary>
              <ul className="max-h-56 divide-y divide-line/60 overflow-y-auto border-t border-line px-3">
                {team.players.map((player) => <li key={player.player_id} className="flex items-center justify-between gap-2 py-1.5 text-[10px]"><span className="min-w-0 truncate text-bone">{player.full_name}<span className="text-sage"> · {player.position}</span></span><span className="shrink-0 font-mono text-sage">{player.ovr} · {player.age}</span></li>)}
              </ul>
            </details>
          ))}
        </div>
      </section>

      <section className="console-card overflow-hidden">
        <div className="border-b border-line bg-cardLight/50 px-4 py-3"><h4 className="text-[14px] font-semibold text-bone">Past winners</h4></div>
        {data.history.length === 0 ? <p className="px-4 py-6 text-[12px] text-sage">The national archive will fill in after the first season ends.</p> : <ul className="divide-y divide-line/70">{data.history.map((season) => <li key={season.season} className="flex flex-wrap items-center justify-between gap-2 px-4 py-2.5"><span className="font-mono text-[12px] text-brass">{season.season}</span><span className="text-[12px] font-semibold text-bone">{season.champion?.name ?? (season.champion_id ? teamLabel(season.champion_id) : 'No winner recorded')}</span></li>)}</ul>}
      </section>

      <NationsMatchModal fixture={matchFixture} onClose={() => setMatchFixture(null)} onPlayed={onRefresh} />
    </div>
  );
};

export const CompetitionTable: React.FC<{
  rows: CompetitionTableRow[];
  european: boolean;
  leagueName?: string;
  onViewSquad?: (clubId: string) => void;
}> = ({ rows, european, leagueName, onViewSquad }) => {
  const firstRel = european ? -1 : rows.findIndex((_, i) => qualificationBand(leagueName || '', i + 1, rows.length) === 'rel');
  const legend = european
    ? [
        { cls: 'bg-brass', label: '1–8 · Round of 16 bye' },
        { cls: 'bg-[#8AB4C8]', label: '9–24 · Knockout play-off' },
        { cls: 'bg-sage/40', label: '25–36 · Eliminated' },
      ]
    : (['ucl', 'el', 'ecl', 'rel'] as const).map((band) => ({ cls: qualificationBarClass(band), label: qualificationLabel(band) }));
  return (
  <div className="console-card overflow-hidden">
    <div className="px-4 py-3 bg-cardLight/60 border-b border-line flex flex-wrap items-center justify-between gap-2">
      <h4 className="text-[14px] font-semibold text-bone">{european ? 'League Phase Standings' : 'League Standings'}</h4>
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-[10px] font-mono text-sage">
        {legend.map((chip) => (
          <span key={chip.label} className="inline-flex items-center gap-1.5">
            <i className={cx('h-2 w-2 rounded-full', chip.cls)} />
            {chip.label}
          </span>
        ))}
      </div>
    </div>
    <div className="overflow-x-auto">
      <table className="w-full text-left text-[13px]">
        <thead className="table-head">
          <tr>
            <th className="py-2.5 px-3">#</th>
            <th className="py-2.5 px-3">Club</th>
            {['P', 'W', 'D', 'L', 'GD'].map((h) => (
              <th key={h} className="py-2.5 px-2">{h}</th>
            ))}
            <th className="py-2.5 px-3">PTS</th>
            {european && <th className="py-2.5 px-2" title="UEFA-style coefficient points; seeds future Swiss pots">Coeff</th>}
          </tr>
        </thead>
        <tbody className="divide-y divide-line/70">
          {rows.map((row, idx) => {
            const band: QualBand = european
              ? (idx < 8 ? 'ucl' : idx < 24 ? 'el' : null)
              : qualificationBand(leagueName || '', idx + 1, rows.length);
            const stripe = european
              ? idx < 8
                ? 'bg-brass'
                : idx < 24
                  ? 'bg-[#8AB4C8]'
                  : 'bg-sage/20'
              : qualificationBarClass(band);
            const bandEdge = european
              ? idx === 8
                ? 'border-t-2 border-t-brass/40'
                : idx === 24
                  ? 'border-t-2 border-t-[#8AB4C8]/40'
                  : ''
              : idx === firstRel
                ? 'border-t-2 border-t-ember/40'
                : '';

            return (
              <tr
                key={row.club_id}
                className={cx('hover:bg-cardLight/50 transition-colors', bandEdge, (european ? idx < 8 : idx === 0) && 'bg-brass/[0.04]')}
              >
                <td className="py-2.5 px-3 font-bold">
                  <span className="flex items-center gap-2">
                    <span className={cx('w-1.5 h-4 rounded-full shadow-sm', stripe)} />
                    <span className="font-mono text-bone/85 text-[13px]">{idx + 1}</span>
                  </span>
                </td>
                <td className="py-2.5 px-3 font-semibold text-bone">
                  <button
                    type="button"
                    className="flex items-center gap-2.5 min-w-0 hover:text-brass transition-colors text-left"
                    onClick={() => onViewSquad?.(row.club_id)}
                  >
                    <ClubCrest club={asClub(row.club)} size={22} />
                    <span className="truncate">{row.club?.club_name ?? row.club_id}</span>
                  </button>
                </td>
                <td className="py-2.5 px-2 text-sage font-mono">{row.played}</td>
                <td className="py-2.5 px-2 text-bone font-semibold font-mono">{row.won}</td>
                <td className="py-2.5 px-2 text-sage font-mono">{row.drawn}</td>
                <td className="py-2.5 px-2 text-sage font-mono">{row.lost}</td>
                <td className={cx('py-2.5 px-2 font-mono', row.goal_difference > 0 ? 'text-pitchtone' : row.goal_difference < 0 ? 'text-ember' : 'text-bone/75')}>{formatGd(row.goal_difference)}</td>
                <td className="py-2.5 px-3 font-bold text-bone font-mono text-[14px]">{row.points}</td>
                {european && <td className="py-2.5 px-2 text-sage font-mono">{row.club?.coefficient ?? 0}</td>}
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  </div>
  );
};

// tieGroups clusters a round's fixture rows by tie (two legs share one
// tie_id; single legs stand alone), preserving backend order and sorting
// legs numerically. No fixture is ever dropped or duplicated.
export const TieGroupView: React.FC<{
  group: CompetitionFixtureRow[];
  onWatchFixture?: (fixture: CompetitionFixtureRow) => void;
  onOpenReport?: (fixture: CompetitionFixtureRow) => void;
  currentMatchweek?: number;
}> = ({ group, onWatchFixture, onOpenReport, currentMatchweek }) => {
  const agg = aggregateLine(group);
  const winnerId = tieWinnerId(group);
  const sideClass = (side: CompetitionFixtureRow, home: boolean): string => {
    const id = home ? side.home_id : side.away_id;
    if (winnerId && winnerId === id) return 'font-bold text-brass';
    if (home ? side.home : side.away) return 'text-bone';
    return 'text-sage/60';
  };
  return (
    <div className="m-2 overflow-hidden rounded-md border border-line bg-ink/40">
      {agg && (
        <p className="flex items-center justify-between gap-2 border-b border-line/70 bg-black/20 px-3 py-1.5 text-[11px] font-mono font-bold text-brass">
          <span className="uppercase tracking-[0.1em]">Aggregate</span>
          <span>{agg.replace('AGG ', '')}</span>
        </p>
      )}
      {group.map((tie) => (
        <div
          key={tie.id}
          className="border-b border-line/60 px-3 py-2 last:border-b-0 hover:bg-cardLight/40"
        >
          <div className="flex items-center justify-between gap-2">
            <span className="flex items-center gap-2 min-w-0">
              <ClubDot club={asClub(tie.home)} size={18} />
              <span className={cx('text-[12.5px] truncate', sideClass(tie, true))}>{tie.home?.short_name ?? 'TBD'}</span>
            </span>
            <span className="flex items-center gap-2 font-mono text-[12px] text-sage">
              {scoreLine(tie)}
              {tie.status === 'finished' && onOpenReport && (
                <button
                  type="button"
                  onClick={() => { soundManager.playClick(); onOpenReport(tie); }}
                  className="font-sans text-[11px] text-brass hover:text-bone"
                >
                  Report
                </button>
              )}
              {onWatchFixture && isWatchable(tie, currentMatchweek) && (
                <button
                  type="button"
                  onClick={() => { soundManager.playClick(); onWatchFixture(tie); }}
                  className="font-sans text-[11px] text-brass hover:text-bone"
                >
                  Match Centre
                </button>
              )}
            </span>
          </div>
          <div className="flex items-center justify-between gap-2 mt-1">
            <span className="flex items-center gap-2 min-w-0">
              <ClubDot club={asClub(tie.away)} size={18} />
              <span className={cx('text-[12.5px] truncate', sideClass(tie, false))}>{tie.away?.short_name ?? 'TBD'}</span>
            </span>
            <span className="text-[10px] font-mono uppercase text-sage">{legLabel(tie) || tie.stage}</span>
          </div>
        </div>
      ))}
    </div>
  );
};

export const KnockoutBoard: React.FC<{
  rounds: CompetitionDetail['rounds'];
  participants: CompetitionDetail['participants'];
  onWatchFixture?: (fixture: CompetitionFixtureRow) => void;
  onOpenReport?: (fixture: CompetitionFixtureRow) => void;
  currentMatchweek?: number;
  onViewSquad?: (clubId: string) => void;
}> = ({ rounds, participants, onWatchFixture, onOpenReport, currentMatchweek, onViewSquad }) => (
  <div className="panel-tight overflow-hidden">
    <div className="px-4 py-3 border-b border-line bg-cardLight/50">
      <h4 className="text-[14px] font-semibold text-bone">Knockout</h4>
    </div>
    <div className="p-4 grid gap-4 md:grid-cols-2 xl:grid-cols-3">
      {rounds.map((round) => (
        <div key={`${round.stage}-${round.fixture_ids.join(',')}`} className="border border-line bg-ink/40">
          <div className="flex items-center justify-between gap-2 border-b border-line bg-black/15 px-3 py-2">
            <p className="text-[11px] font-mono uppercase tracking-[0.12em] text-brass/90">{round.stage}</p>
            {(round.tie_ids?.length ?? 0) > 0 && (round.fixture_ids?.length ?? 0) > (round.tie_ids?.length ?? 0) && (
              <span className="rounded-full border border-line bg-ink/60 px-2 py-0.5 text-[9px] font-bold uppercase tracking-[0.1em] text-sage">Two legs</span>
            )}
          </div>
          <div className="divide-y divide-line/70">
            {tieGroups(round.ties ?? []).map((group) => (
              <TieGroupView key={group[0]?.tie_id ?? group[0]?.id} group={group} onWatchFixture={onWatchFixture} onOpenReport={onOpenReport} currentMatchweek={currentMatchweek} />
            ))}
            {(round.bye_ids ?? []).length > 0 && (
              <div className="px-3 py-2">
                <p className="text-[11px] font-mono uppercase tracking-[0.12em] text-brass mb-1.5">
                  Byes · straight through
                </p>
                <div className="flex flex-wrap gap-1.5">
                  {(round.bye_ids ?? []).map((clubId) => {
                    const club = participants.find((c) => c.club_id === clubId);
                    return (
                      <button
                        key={clubId}
                        type="button"
                        onClick={() => onViewSquad?.(clubId)}
                        title={club?.club_name ?? clubId}
                        className="flex items-center gap-1.5 px-2 py-1 border border-brass/40 bg-brass/[0.07] hover:bg-brass/[0.14] transition-colors"
                      >
                        <ClubDot club={asClub(club ?? null)} size={16} />
                        <span className="text-[12px] font-semibold text-bone">{club?.short_name ?? clubId}</span>
                      </button>
                    );
                  })}
                </div>
              </div>
            )}
          </div>
        </div>
      ))}
    </div>
  </div>
);
