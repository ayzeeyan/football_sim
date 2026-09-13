import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Crown, Flag, Globe2, Trophy } from 'lucide-react';
import type {
  CompetitionClub,
  CompetitionDetail,
  CompetitionFixtureRow,
  CompetitionKind,
  CompetitionSummary,
  CompetitionTableRow,
} from '../types';
import { fetchClubs, fetchCompetition, fetchCompetitions, fetchFavourite } from '../services/api';
import { useAsyncData } from '../hooks/useAsyncData';
import { formatGd, cx } from '../lib/format';
import { soundManager } from '../audio/webAudio';
import { Card, ClubDot, LoadingState, PanelHeader } from './ui/ui';
import type { Club } from '../types';

interface CompetitionHubTabProps {
  /** Only a scheduled fixture on the active matchweek can enter live play. */
  onWatchFixture?: (fixture: CompetitionFixtureRow) => void;
  currentMatchweek?: number;
  onViewSquad?: (clubId: string) => void;
}

function kindLabel(kind: CompetitionKind): string {
  if (kind === 'LEAGUE') return 'Domestic league';
  if (kind === 'DOMESTIC_CUP') return 'Domestic cup';
  return 'European';
}

function kindTone(kind: CompetitionKind): string {
  if (kind === 'LEAGUE') return 'text-[#A9CBDD]';
  if (kind === 'DOMESTIC_CUP') return 'text-[#A9CDBB]';
  return 'text-brass';
}

function asClub(club: CompetitionClub | null | undefined): Club | null {
  if (!club) return null;
  return club as unknown as Club;
}

function scoreLine(f: CompetitionFixtureRow): string {
  if (f.status !== 'finished' || f.home_goals == null || f.away_goals == null) return '–';
  const pens = f.decided_by === 'penalties' && f.penalties && f.penalties.length >= 2
    ? ` (${f.penalties[0]}–${f.penalties[1]} pens)`
    : f.decided_by === 'extra_time'
      ? ' AET'
      : '';
  return `${f.home_goals}–${f.away_goals}${pens}`;
}

function legLabel(f: CompetitionFixtureRow): string {
  if (f.leg === 1) return 'Leg 1';
  if (f.leg === 2) return 'Leg 2 · decider';
  return '';
}

const LEAGUE_BY_COMPETITION: Record<string, string> = {
  'premier-league': 'Premier League',
  'la-liga': 'La Liga',
  'bundesliga': 'Bundesliga',
  'serie-a': 'Serie A',
  'ligue-1': 'Ligue 1',
};

const KIND_ORDER: CompetitionKind[] = ['LEAGUE', 'DOMESTIC_CUP', 'EUROPEAN'];

function isWatchable(fixture: CompetitionFixtureRow, currentMatchweek: number | undefined): boolean {
  return fixture.status === 'scheduled' && currentMatchweek != null && fixture.matchweek === currentMatchweek;
}

export const CompetitionHubTab: React.FC<CompetitionHubTabProps> = ({ onWatchFixture, currentMatchweek, onViewSquad }) => {
  const { data: hub, loading } = useAsyncData(fetchCompetitions, []);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [detail, setDetail] = useState<CompetitionDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [fixtureFilter, setFixtureFilter] = useState<'all' | 'results' | 'upcoming'>('all');

  const competitions = hub?.competitions ?? [];
  const grouped = useMemo(() => {
    const buckets: Record<CompetitionKind, CompetitionSummary[]> = {
      LEAGUE: [],
      DOMESTIC_CUP: [],
      EUROPEAN: [],
    };
    for (const c of competitions) buckets[c.kind]?.push(c);
    return buckets;
  }, [competitions]);

  // Favourite-club league wins the default selection so the hub opens on
  // the watched domestic league instead of always premier-league.
  const [defaultCompId, setDefaultCompId] = useState<string | null>(null);
  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const [fav, clubList] = await Promise.all([fetchFavourite(), fetchClubs()]);
        if (cancelled) return;
        const club = clubList.find((c) => c.club_id === fav.favourite_club_id);
        if (!club) return;
        const match = competitions.find((c) => c.kind === 'LEAGUE' && LEAGUE_BY_COMPETITION[c.id] === club.league);
        if (match) setDefaultCompId(match.id);
      } catch {
        // Keep the hub default; favourites are a nicety, not a gate.
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [competitions]);

  const activeId = selectedId ?? defaultCompId ?? competitions[0]?.id ?? null;

  const [detailError, setDetailError] = useState<string | null>(null);

  const loadDetail = useCallback(async (id: string) => {
    setDetailLoading(true);
    setDetailError(null);
    try {
      const next = await fetchCompetition(id);
      setDetail(next);
    } catch {
      setDetail(null);
      setDetailError(`Could not load ${id}. Check the engine and retry.`);
    } finally {
      setDetailLoading(false);
    }
  }, []);

  useEffect(() => {
    if (!activeId) {
      setDetail(null);
      return;
    }
    void loadDetail(activeId);
  }, [activeId, loadDetail]);

  if (loading || !hub) {
    return <Card><LoadingState message="Loading competitions…" /></Card>;
  }

  if (!hub.world || competitions.length === 0) {
    return (
      <div className="space-y-5">
        <PanelHeader
          kicker="World"
          title="Competition Hub"
          subtitle="This save is the original 12-club Super League. Start a new career to open the Top Five European world."
        />
        <Card>
          <p className="text-[14px] text-sage leading-relaxed">
            Legacy careers keep their Super League, Champions Cup and Super Cup boards on the League tab.
            They are not converted into the 96-club calendar.
          </p>
        </Card>
      </div>
    );
  }

  const upcoming = (detail?.fixtures ?? []).filter((f) => f.status !== 'finished');
  const results = (detail?.fixtures ?? []).filter((f) => f.status === 'finished');
  const shownFixtures = fixtureFilter === 'results' ? results : fixtureFilter === 'upcoming' ? upcoming : (detail?.fixtures ?? []);

  return (
    <div className="space-y-5">
      <PanelHeader
        kicker="World"
        title="Competition Hub"
        subtitle="Domestic leagues, national cups and the three UEFA competitions share one calendar."
      />

      <div className="grid grid-cols-1 xl:grid-cols-[280px_minmax(0,1fr)] gap-5">
        <aside className="space-y-4">
          {KIND_ORDER.map((kind) => (
            <div key={kind} className="panel-tight overflow-hidden">
              <div className="px-3 py-2 border-b border-line bg-cardLight/50">
                <p className={cx('text-[11px] font-mono uppercase tracking-[0.14em]', kindTone(kind))}>{kindLabel(kind)}</p>
              </div>
              <div className="divide-y divide-line/70">
                {grouped[kind].map((c) => (
                  <button
                    key={c.id}
                    onClick={() => {
                      soundManager.playClick();
                      setSelectedId(c.id);
                      setFixtureFilter('all');
                    }}
                    className={cx(
                      'w-full text-left px-3 py-2.5 hover:bg-cardLight/60 transition-colors',
                      activeId === c.id && 'bg-bone/10',
                    )}
                  >
                    <div className="flex items-center justify-between gap-2">
                      <span className="text-[13px] font-semibold text-bone truncate">{c.name}</span>
                      <span className="font-mono text-[11px] text-sage shrink-0">{c.participants}</span>
                    </div>
                    <p className="text-[12px] text-sage mt-0.5 truncate">{c.country} · {c.stage}</p>
                  </button>
                ))}
              </div>
            </div>
          ))}
        </aside>

        <section className="space-y-4 min-w-0">
          {detailLoading || !detail ? (
            detailError && activeId ? (
              <Card>
                <p className="text-[13px] text-sage">{detailError}</p>
                <button
                  type="button"
                  onClick={() => {
                    soundManager.playClick();
                    void loadDetail(activeId);
                  }}
                  className="mt-3 px-4 py-2 rounded-lg bg-brass text-ink text-[13px] font-bold hover:bg-[#D4AF4D] transition-colors"
                >
                  Retry
                </button>
              </Card>
            ) : (
              <Card><LoadingState message="Loading competition…" /></Card>
            )
          ) : (
            <>
              <Card>
                <div className="flex flex-wrap items-start justify-between gap-4">
                  <div>
                    <p className={cx('eyebrow', kindTone(detail.kind))}>{kindLabel(detail.kind)} · {detail.country}</p>
                    <h3 className="font-display text-[30px] font-semibold text-bone mt-1">{detail.name}</h3>
                    <p className="text-[14px] text-sage mt-1">
                      {detail.stage} · {detail.participants.length} clubs · prestige {detail.prestige}
                    </p>
                  </div>
                  {detail.champion ? (
                    <div className="flex items-center gap-3 border border-brass/40 bg-brass/[0.08] px-3 py-2">
                      <Crown size={16} className="text-brass" />
                      <div>
                        <p className="text-[11px] font-mono uppercase tracking-[0.12em] text-brass">Champion</p>
                        <p className="text-[14px] font-semibold text-bone">{detail.champion.club_name}</p>
                      </div>
                    </div>
                  ) : (
                    <div className="flex items-center gap-2 text-sage text-[13px]">
                      {detail.kind === 'EUROPEAN' ? <Globe2 size={15} /> : detail.kind === 'LEAGUE' ? <Trophy size={15} /> : <Flag size={15} />}
                      In progress
                    </div>
                  )}
                </div>
              </Card>

              {detail.pots && detail.pots.length > 0 && (
                <div className="panel-tight overflow-hidden">
                  <div className="px-4 py-3 border-b border-line bg-cardLight/50">
                    <h4 className="text-[14px] font-semibold text-bone">League-phase pots</h4>
                  </div>
                  <div className="p-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
                    {detail.pots.map((pot, i) => (
                      <div key={i} className="border border-line bg-ink/40 px-3 py-2">
                        <p className="text-[11px] font-mono uppercase tracking-[0.12em] text-brass mb-1.5">Pot {i + 1}</p>
                        <ul className="space-y-1">
                          {pot.map((clubId) => {
                            const club = detail.participants.find((c) => c.club_id === clubId);
                            return (
                              <li key={clubId} className="flex items-center gap-2 min-w-0">
                                <ClubDot club={asClub(club ?? null)} size={16} />
                                <span className="text-[12px] text-bone truncate">{club?.short_name ?? clubId}</span>
                              </li>
                            );
                          })}
                        </ul>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {detail.qualification_sources && Object.keys(detail.qualification_sources).length > 0 && (
                <div className="panel-tight overflow-hidden">
                  <div className="px-4 py-3 border-b border-line bg-cardLight/50">
                    <h4 className="text-[14px] font-semibold text-bone">Qualification source</h4>
                  </div>
                  <ul className="divide-y divide-line/70 max-h-64 overflow-y-auto">
                    {detail.participants.map((club) => (
                      <li key={club.club_id} className="px-4 py-2 flex items-center justify-between gap-3">
                        <span className="flex items-center gap-2 min-w-0">
                          <ClubDot club={asClub(club)} />
                          <span className="text-[13px] text-bone truncate">{club.club_name}</span>
                        </span>
                        <span className="text-[12px] text-sage font-mono shrink-0">
                          {detail.qualification_sources?.[club.club_id] || 'Allocated'}
                        </span>
                      </li>
                    ))}
                  </ul>
                </div>
              )}

              {detail.table.length > 0 && (
                <CompetitionTable
                  rows={detail.table}
                  european={detail.kind === 'EUROPEAN'}
                  onViewSquad={onViewSquad}
                />
              )}

              {detail.rounds.length > 0 && (
                <KnockoutBoard rounds={detail.rounds} participants={detail.participants} onWatchFixture={onWatchFixture} currentMatchweek={currentMatchweek} onViewSquad={onViewSquad} />
              )}

              <div className="panel-tight overflow-hidden">
                <div className="px-4 py-3 border-b border-line bg-cardLight/50 flex flex-wrap items-center justify-between gap-2">
                  <h4 className="text-[14px] font-semibold text-bone">Fixtures and results</h4>
                  <div className="flex gap-1">
                    {(['all', 'upcoming', 'results'] as const).map((key) => (
                      <button
                        key={key}
                        onClick={() => { soundManager.playClick(); setFixtureFilter(key); }}
                        className={cx(
                          'px-2 py-1 text-[12px] font-medium capitalize',
                          fixtureFilter === key ? 'bg-bone text-ink' : 'text-sage hover:text-bone',
                        )}
                      >
                        {key}
                      </button>
                    ))}
                  </div>
                </div>
                <ul className="divide-y divide-line/70 max-h-[28rem] overflow-y-auto">
                  {shownFixtures.length === 0 && (
                    <li className="px-4 py-6 text-[13px] text-sage">No fixtures in this filter.</li>
                  )}
                  {shownFixtures.map((f) => (
                    <li key={f.id} className="px-4 py-2.5 flex items-center gap-3">
                      <span className="font-mono text-[11px] text-sage w-10 shrink-0">MW{f.matchweek}</span>
                      <span className="flex-1 min-w-0 flex items-center justify-between gap-2">
                        <span className="flex items-center gap-2 min-w-0">
                          <ClubDot club={asClub(f.home)} />
                          <span className="text-[13px] text-bone truncate">{f.home?.short_name ?? f.home_id}</span>
                        </span>
                        <span className="font-mono text-[13px] font-semibold text-bone tabular-nums shrink-0">{scoreLine(f)}</span>
                        <span className="flex items-center gap-2 min-w-0 justify-end">
                          <span className="text-[13px] text-bone truncate">{f.away?.short_name ?? f.away_id}</span>
                          <ClubDot club={asClub(f.away)} />
                        </span>
                      </span>
                      <span className="hidden sm:block text-[11px] font-mono text-sage w-32 truncate">{f.stage}{legLabel(f) ? ` · ${legLabel(f)}` : ''}</span>
                      {onWatchFixture && isWatchable(f, currentMatchweek) && (
                        <button
                          onClick={() => { soundManager.playClick(); onWatchFixture(f); }}
                          className="text-[12px] text-brass hover:text-bone shrink-0"
                        >
                          Watch live
                        </button>
                      )}
                    </li>
                  ))}
                </ul>
              </div>
            </>
          )}
        </section>
      </div>
    </div>
  );
};

const CompetitionTable: React.FC<{
  rows: CompetitionTableRow[];
  european: boolean;
  onViewSquad?: (clubId: string) => void;
}> = ({ rows, european, onViewSquad }) => (
  <div className="panel-tight overflow-hidden">
    <div className="px-4 py-3 bg-cardLight/60 border-b border-line flex items-center justify-between">
      <h4 className="text-[14px] font-semibold text-bone">{european ? 'League phase' : 'Standings'}</h4>
      {european && <span className="text-[11px] font-mono text-sage">Top sides earn byes · two-legged knockouts</span>}
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
          {rows.map((row, idx) => (
            <tr
              key={row.club_id}
              className={cx('hover:bg-cardLight/50 transition-colors', european && idx < 4 && 'bg-brass/[0.05]')}
            >
              <td className="py-2.5 px-3 font-bold font-mono text-bone/85">{idx + 1}</td>
              <td className="py-2.5 px-3 font-semibold text-bone">
                <button
                  className="flex items-center gap-2 min-w-0"
                  onClick={() => onViewSquad?.(row.club_id)}
                >
                  <ClubDot club={asClub(row.club)} />
                  <span className="truncate">{row.club?.club_name ?? row.club_id}</span>
                </button>
              </td>
              <td className="py-2.5 px-2 text-sage font-mono">{row.played}</td>
              <td className="py-2.5 px-2 text-bone font-semibold font-mono">{row.won}</td>
              <td className="py-2.5 px-2 text-sage font-mono">{row.drawn}</td>
              <td className="py-2.5 px-2 text-sage font-mono">{row.lost}</td>
              <td className="py-2.5 px-2 font-mono text-bone/75">{formatGd(row.goal_difference)}</td>
              <td className="py-2.5 px-3 font-bold text-bone font-mono text-[14px]">{row.points}</td>
              {european && <td className="py-2.5 px-2 text-sage font-mono">{row.club?.coefficient ?? 0}</td>}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  </div>
);

// tieGroups clusters a round's fixture rows by tie (two legs share one
// tie_id; single legs stand alone), preserving backend order and sorting
// legs numerically. No fixture is ever dropped or duplicated.
export function tieGroups(ties: CompetitionFixtureRow[]): CompetitionFixtureRow[][] {
  const order: string[] = [];
  const byId = new Map<string, CompetitionFixtureRow[]>();
  for (const t of ties) {
    const k = t.tie_id ?? t.id;
    const arr = byId.get(k);
    if (arr) arr.push(t);
    else {
      byId.set(k, [t]);
      order.push(k);
    }
  }
  return order.map((k) => {
    const arr = byId.get(k) ?? [];
    return [...arr].sort((a, b) => (a.leg ?? 0) - (b.leg ?? 0) || (a.id < b.id ? -1 : a.id > b.id ? 1 : 0));
  });
}

// aggregateLine renders a finished two-legged tie's real aggregate (summed
// per club, since venues swap). Unfinished or degenerate groups yield null
// so no manufactured scoreline is ever shown.
export function aggregateLine(group: CompetitionFixtureRow[]): string | null {
  if (group.length !== 2) return null;
  const [l1, l2] = group;
  if (l1.status !== 'finished' || l2.status !== 'finished') return null;
  if (l1.home_goals == null || l1.away_goals == null || l2.home_goals == null || l2.away_goals == null) return null;
  const totals = new Map<string, { goals: number; short: string }>();
  const add = (id: string, short: string | undefined, g: number) => {
    const cur = totals.get(id) ?? { goals: 0, short: short ?? id };
    cur.goals += g;
    totals.set(id, cur);
  };
  add(l1.home_id, l1.home?.short_name, l1.home_goals);
  add(l1.away_id, l1.away?.short_name, l1.away_goals);
  add(l2.home_id, l2.home?.short_name, l2.home_goals);
  add(l2.away_id, l2.away?.short_name, l2.away_goals);
  if (totals.size !== 2) return null;
  const [x, y] = [...totals.entries()].sort((a, b) => (a[0] < b[0] ? -1 : 1));
  return `AGG ${x[1].short} ${x[1].goals}–${y[1].goals} ${y[1].short}`;
}

const TieGroupView: React.FC<{
  group: CompetitionFixtureRow[];
  onWatchFixture?: (fixture: CompetitionFixtureRow) => void;
  currentMatchweek?: number;
}> = ({ group, onWatchFixture, currentMatchweek }) => {
  const agg = aggregateLine(group);
  return (
    <div>
      {agg && <p className="px-3 pt-2 text-[11px] font-mono font-bold text-brass">{agg}</p>}
      {group.map((tie) => (
        <div
          key={tie.id}
          className="px-3 py-2 hover:bg-cardLight/40"
        >
          <div className="flex items-center justify-between gap-2">
            <span className="flex items-center gap-2 min-w-0">
              <ClubDot club={asClub(tie.home)} size={18} />
              <span className="text-[12.5px] text-bone truncate">{tie.home?.short_name ?? 'TBD'}</span>
            </span>
            <span className="flex items-center gap-2 font-mono text-[12px] text-sage">
              {scoreLine(tie)}{legLabel(tie) ? ` · ${legLabel(tie)}` : ''}
              {onWatchFixture && isWatchable(tie, currentMatchweek) && (
                <button
                  type="button"
                  onClick={() => { soundManager.playClick(); onWatchFixture(tie); }}
                  className="font-sans text-[11px] text-brass hover:text-bone"
                >
                  Watch live
                </button>
              )}
            </span>
          </div>
          <div className="flex items-center justify-between gap-2 mt-1">
            <span className="flex items-center gap-2 min-w-0">
              <ClubDot club={asClub(tie.away)} size={18} />
              <span className="text-[12.5px] text-bone truncate">{tie.away?.short_name ?? 'TBD'}</span>
            </span>
            <span className="text-[10px] font-mono uppercase text-sage">{tie.status}{legLabel(tie) ? ` · ${legLabel(tie)}` : ''}</span>
          </div>
        </div>
      ))}
    </div>
  );
};

const KnockoutBoard: React.FC<{
  rounds: CompetitionDetail['rounds'];
  participants: CompetitionDetail['participants'];
  onWatchFixture?: (fixture: CompetitionFixtureRow) => void;
  currentMatchweek?: number;
  onViewSquad?: (clubId: string) => void;
}> = ({ rounds, participants, onWatchFixture, currentMatchweek, onViewSquad }) => (
  <div className="panel-tight overflow-hidden">
    <div className="px-4 py-3 border-b border-line bg-cardLight/50">
      <h4 className="text-[14px] font-semibold text-bone">Knockout</h4>
    </div>
    <div className="p-4 grid gap-4 md:grid-cols-2 xl:grid-cols-3">
      {rounds.map((round) => (
        <div key={`${round.stage}-${round.fixture_ids.join(',')}`} className="border border-line bg-ink/40">
          <p className="px-3 py-2 text-[11px] font-mono uppercase tracking-[0.12em] text-sage border-b border-line">
            {round.stage}{(round.tie_ids?.length ?? 0) > 0 && (round.fixture_ids?.length ?? 0) > (round.tie_ids?.length ?? 0) ? ' · home & away' : ''}
          </p>
          <div className="divide-y divide-line/70">
            {tieGroups(round.ties ?? []).map((group) => (
              <TieGroupView key={group[0].tie_id ?? group[0].id} group={group} onWatchFixture={onWatchFixture} currentMatchweek={currentMatchweek} />
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
