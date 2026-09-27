import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { CalendarDays, Crown, Flag, Globe2, Trophy } from 'lucide-react';
import type {
  Club,
  CompetitionClub,
  CompetitionDetail,
  CompetitionFixtureRow,
  CompetitionKind,
  CompetitionSummary,
  CompetitionTableRow,
  Fixture,
  NationsCupFixture,
  NationsCupResponse,
} from '../../types';
import { fetchClubs, fetchCompetition, fetchCompetitions, fetchFavourite, fetchNationsCup } from '../../services/api';
import { PostMatchModal } from '../postmatch/PostMatchBroadcast';
import { useAsyncData } from '../../hooks/useAsyncData';
import { formatGd, cx } from '../../lib/format';
import { qualificationBand, qualificationBarClass, qualificationLabel, type QualBand } from '../../lib/qualification';
import { soundManager } from '../../audio/webAudio';
import { Card, ClubCrest, ClubDot, ErrorState, LoadingState, PanelHeader } from '../ui/ui';
import { NationsMatchModal } from './NationsMatchModal';
import { usePlayerSheet } from '../clubs/PlayerSheet';

interface CompetitionHubTabProps {
  /** Opens the selected fixture in the instant simulation centre. */
  onWatchFixture?: (fixture: CompetitionFixtureRow) => void;
  currentMatchweek?: number;
  onViewSquad?: (clubId: string) => void;
}

function kindLabel(kind: CompetitionKind): string {
  if (kind === 'LEAGUE') return 'Domestic league';
  if (kind === 'DOMESTIC_CUP') return 'Domestic cup';
  if (kind === 'INTERNATIONAL') return 'National teams';
  return 'European';
}

function kindTone(kind: CompetitionKind): string {
  if (kind === 'LEAGUE') return 'text-[#A9CBDD]';
  if (kind === 'DOMESTIC_CUP') return 'text-[#A9CDBB]';
  return 'text-brass';
}

function KindIcon({ kind, size = 14 }: { kind: CompetitionKind; size?: number }) {
  if (kind === 'LEAGUE') return <Trophy size={size} aria-hidden="true" />;
  if (kind === 'DOMESTIC_CUP') return <Flag size={size} aria-hidden="true" />;
  return <Globe2 size={size} aria-hidden="true" />;
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

const COUNTRY_ORDER = ['England', 'Spain', 'Germany', 'Italy', 'France', 'Europe'] as const;

function isWatchable(fixture: CompetitionFixtureRow, currentMatchweek: number | undefined): boolean {
  return fixture.status === 'scheduled' && currentMatchweek != null && fixture.matchweek === currentMatchweek;
}

function groupByMatchweek(rows: CompetitionFixtureRow[]): Array<[number, CompetitionFixtureRow[]]> {
  const out = new Map<number, CompetitionFixtureRow[]>();
  for (const f of rows) {
    const list = out.get(f.matchweek) ?? [];
    list.push(f);
    out.set(f.matchweek, list);
  }
  return [...out.entries()].sort((a, b) => a[0] - b[0]);
}

export const CompetitionHubTab: React.FC<CompetitionHubTabProps> = ({ onWatchFixture, currentMatchweek, onViewSquad }) => {
  const { data: hub, loading, error: hubError, reload: reloadHub } = useAsyncData(fetchCompetitions, []);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [detail, setDetail] = useState<CompetitionDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [fixtureFilter, setFixtureFilter] = useState<'all' | 'results' | 'upcoming'>('all');
  const [pane, setPane] = useState<'overview' | 'table' | 'fixtures' | 'results' | 'stats' | 'history'>('overview');
  const [openFixture, setOpenFixture] = useState<Fixture | null>(null);
  const openCompleted = (row: CompetitionFixtureRow) => {
    const id = row.fixture_id || row.id;
    if (!id) return;
    setOpenFixture({ ...row, id, fixture_id: id } as unknown as Fixture);
  };
  const { openPlayer } = usePlayerSheet();

  const competitions = hub?.competitions ?? [];
  const grouped = useMemo(() => {
    const buckets = new Map<string, CompetitionSummary[]>();
    for (const c of competitions) {
      const country = c.country || 'Europe';
      const list = buckets.get(country) ?? [];
      list.push(c);
      buckets.set(country, list);
    }
    const seen = new Set<string>(COUNTRY_ORDER);
    const extra: string[] = [];
    for (const country of buckets.keys()) {
      if (!seen.has(country)) extra.push(country);
    }
    return [...COUNTRY_ORDER, ...extra].filter((country) => (buckets.get(country) ?? []).length > 0).map((country) => ({
      country,
      items: buckets.get(country) ?? [],
    }));
  }, [competitions]);

  // The observational club preference chooses the initial inspection league;
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
  const { data: nationsCup, loading: nationsLoading, error: nationsError, reload: reloadNations } = useAsyncData(
    () => activeId === 'nations-cup' ? fetchNationsCup() : Promise.resolve(null),
    [activeId],
  );

  const [detailError, setDetailError] = useState<string | null>(null);

  const loadDetail = useCallback(async (id: string) => {
    if (id === 'nations-cup') {
      setDetail(null);
      setDetailLoading(false);
      setDetailError(null);
      return;
    }
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

  if (loading) {
    return <Card><LoadingState message="Loading competitions…" /></Card>;
  }

  if (hubError || !hub) {
    return <Card><ErrorState message={hubError || 'The competition index returned no data.'} onRetry={reloadHub} /></Card>;
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
  const shownFixtures = pane === 'fixtures'
    ? upcoming
    : pane === 'results'
      ? results
      : fixtureFilter === 'results'
        ? results
        : fixtureFilter === 'upcoming'
          ? upcoming
          : (detail?.fixtures ?? []);

  return (
    <div className="space-y-4">
      <PanelHeader
        kicker="World"
        title="Competition Hub"
        subtitle="Domestic leagues, national cups, UEFA competitions and international fixtures share the season calendar."
      />

      <div className="grid grid-cols-1 xl:grid-cols-[240px_minmax(0,1fr)] gap-4">
        <aside className="space-y-4">
          {grouped.map(({ country, items }) => (
            <div key={country} className="panel-tight overflow-hidden rounded-md">
              <div className="px-3 py-2 border-b border-line bg-cardLight/50">
                <p className="text-[11px] font-mono uppercase tracking-[0.14em] text-brass/80">{country}</p>
              </div>
              <div className="divide-y divide-line/70">
                {items.map((c) => (
                  <button
                    key={c.id}
                    onClick={() => {
                      soundManager.playClick();
                      setSelectedId(c.id);
                      setFixtureFilter('all');
                      setPane('overview');
                    }}
                    aria-current={activeId === c.id ? 'true' : undefined}
                    className={cx(
                      'flex w-full items-center gap-2.5 px-3 py-2.5 text-left transition-colors',
                      activeId === c.id ? 'bg-bone text-ink [&_*]:!text-ink' : 'hover:bg-cardLight/60',
                    )}
                  >
                    <span className={cx('shrink-0', activeId === c.id ? '' : kindTone(c.kind))}>
                      <KindIcon kind={c.kind} />
                    </span>
                    <span className="min-w-0 flex-1">
                      <span className="flex items-center justify-between gap-2">
                        <span className="text-[13px] font-semibold truncate">{c.name}</span>
                        <span className="font-mono text-[11px] shrink-0">{c.participants}</span>
                      </span>
                      <span className="block text-[11px] mt-0.5 truncate">{c.stage}</span>
                    </span>
                  </button>
                ))}
              </div>
            </div>
          ))}
        </aside>

        <section className="space-y-4 min-w-0">
          {activeId === 'nations-cup' ? (
            nationsLoading || !nationsCup ? (
              nationsError ? <Card><ErrorState message={nationsError} onRetry={reloadNations} /></Card> : <Card><LoadingState message="Loading national teams…" /></Card>
            ) : (
              <NationsCupBoard data={nationsCup} onRefresh={reloadNations} />
            )
          ) : detailLoading || !detail ? (
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
                      <KindIcon kind={detail.kind} size={15} />
                      In progress
                    </div>
                  )}
                </div>
              </Card>

              <div className="section-tabs rounded-sm">
                {([
                  ['overview', 'Overview'],
                  ['table', detail.kind === 'DOMESTIC_CUP' ? 'Bracket' : 'Table'],
                  ['fixtures', 'Fixtures'],
                  ['results', 'Results'],
                  ['stats', 'Stats'],
                  ['history', 'History'],
                ] as const).map(([key, label]) => (
                  <button
                    key={key}
                    type="button"
                    onClick={() => { soundManager.playClick(); setPane(key); }}
                    className={cx(
                      'section-tab',
                      pane === key ? 'section-tab-active' : '',
                    )}
                  >
                    {label}
                  </button>
                ))}
              </div>

              {(pane === 'overview' || pane === 'table') && detail.pots && detail.pots.length > 0 && (
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

              {pane === 'overview' && detail.qualification_sources && Object.keys(detail.qualification_sources).length > 0 && (
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

              {(pane === 'overview' || pane === 'table') && detail.table.length > 0 && (
                <CompetitionTable
                  rows={detail.table}
                  european={detail.kind === 'EUROPEAN'}
                  leagueName={detail.name}
                  onViewSquad={onViewSquad}
                />
              )}

              {(pane === 'overview' || pane === 'table') && detail.rounds.length > 0 && (
                <KnockoutBoard rounds={detail.rounds} participants={detail.participants} onWatchFixture={onWatchFixture} onOpenReport={openCompleted} currentMatchweek={currentMatchweek} onViewSquad={onViewSquad} />
              )}

              {pane === 'stats' && (
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="panel-tight overflow-hidden">
                    <div className="px-4 py-3 border-b border-line bg-cardLight/50"><h4 className="text-[14px] font-semibold text-bone">Top scorers</h4></div>
                    <ul className="divide-y divide-line/70">
                      {(detail.scorers ?? []).length === 0 && <li className="px-4 py-6 text-[13px] text-sage">No competition goals yet.</li>}
                      {(detail.scorers ?? []).map((row, i, list) => {
                        const top = Math.max(...list.map((r) => r.goals), 1);
                        return (
                          <li key={row.player_id}>
                            <button type="button" onClick={() => openPlayer(row.player_id)} className="w-full px-4 py-2 flex items-center justify-between gap-3 text-left hover:bg-cardLight/60">
                              <span className="flex min-w-0 items-center gap-2.5">
                                <span className="w-5 shrink-0 text-right font-mono text-[11px] text-sage">{i + 1}</span>
                                <span className="text-[13px] text-bone truncate">{row.full_name}<span className="text-sage font-mono text-[11px]"> · {row.club_short}</span></span>
                              </span>
                              <span className="flex shrink-0 items-center gap-2">
                                <span className="hidden h-1.5 w-16 overflow-hidden rounded-full bg-white/[0.07] sm:block"><span className="block h-full rounded-full bg-brass/80" style={{ width: `${(row.goals / top) * 100}%` }} /></span>
                                <span className="w-8 text-right font-mono text-[13px] text-bone">{row.goals} G</span>
                              </span>
                            </button>
                          </li>
                        );
                      })}
                    </ul>
                  </div>
                  <div className="panel-tight overflow-hidden">
                    <div className="px-4 py-3 border-b border-line bg-cardLight/50"><h4 className="text-[14px] font-semibold text-bone">Assists</h4></div>
                    <ul className="divide-y divide-line/70">
                      {(detail.assisters ?? []).length === 0 && <li className="px-4 py-6 text-[13px] text-sage">No competition assists yet.</li>}
                      {(detail.assisters ?? []).map((row, i, list) => {
                        const top = Math.max(...list.map((r) => r.assists), 1);
                        return (
                          <li key={row.player_id}>
                            <button type="button" onClick={() => openPlayer(row.player_id)} className="w-full px-4 py-2 flex items-center justify-between gap-3 text-left hover:bg-cardLight/60">
                              <span className="flex min-w-0 items-center gap-2.5">
                                <span className="w-5 shrink-0 text-right font-mono text-[11px] text-sage">{i + 1}</span>
                                <span className="text-[13px] text-bone truncate">{row.full_name}<span className="text-sage font-mono text-[11px]"> · {row.club_short}</span></span>
                              </span>
                              <span className="flex shrink-0 items-center gap-2">
                                <span className="hidden h-1.5 w-16 overflow-hidden rounded-full bg-white/[0.07] sm:block"><span className="block h-full rounded-full bg-[#8AB4C8]/80" style={{ width: `${(row.assists / top) * 100}%` }} /></span>
                                <span className="w-8 text-right font-mono text-[13px] text-bone">{row.assists} A</span>
                              </span>
                            </button>
                          </li>
                        );
                      })}
                    </ul>
                  </div>
                </div>
              )}

              {pane === 'history' && (
                <div className="panel-tight overflow-hidden">
                  <div className="px-4 py-3 border-b border-line bg-cardLight/50"><h4 className="text-[14px] font-semibold text-bone">Archive</h4></div>
                  <ul className="divide-y divide-line/70">
                    {(detail.history ?? []).length === 0 && (
                      <li className="px-4 py-6 text-[13px] text-sage">
                        {detail.champion ? `${detail.champion.club_name} hold the current title. Previous seasons appear after rollover.` : 'No completed seasons archived yet.'}
                      </li>
                    )}
                    {(detail.history ?? []).map((h) => (
                      <li key={h.season_name} className="px-4 py-2.5 flex items-center justify-between gap-2">
                        <span className="font-mono text-brass text-[13px]">{h.season_name}</span>
                        <span className="text-[13px] text-bone">{h.champion?.club_name ?? h.champion_id ?? '—'}</span>
                      </li>
                    ))}
                  </ul>
                </div>
              )}

              {(pane === 'overview' || pane === 'fixtures' || pane === 'results') && <div className="panel-tight overflow-hidden">
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
                <div className="max-h-[28rem] overflow-y-auto">
                  {shownFixtures.length === 0 && (
                    <p className="px-4 py-6 text-[13px] text-sage">No fixtures in this filter.</p>
                  )}
                  {groupByMatchweek(shownFixtures).map(([mw, mwFixtures]) => (
                    <div key={mw}>
                      <p className="sticky top-0 z-[1] border-b border-line bg-[#0d2018]/95 px-4 py-1.5 text-[10px] font-mono uppercase tracking-[0.14em] text-brass/90 backdrop-blur-sm">Matchweek {mw}</p>
                      <ul className="divide-y divide-line/70">
                  {mwFixtures.map((f) => (
                    <li key={f.id} className="px-4 py-2.5 flex items-center gap-3">
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
                      {f.status === 'finished' && (
                        <button
                          type="button"
                          onClick={() => { soundManager.playClick(); openCompleted(f); }}
                          className="text-[12px] text-brass hover:text-bone shrink-0"
                        >
                          Report
                        </button>
                      )}
                      {onWatchFixture && isWatchable(f, currentMatchweek) && (
                        <button
                          onClick={() => { soundManager.playClick(); onWatchFixture(f); }}
                          className="text-[12px] text-brass hover:text-bone shrink-0"
                        >
                          Match Centre
                        </button>
                      )}
                    </li>
                  ))}
                      </ul>
                    </div>
                  ))}
                </div>
              </div>}
            </>
          )}
        </section>
      </div>
      <PostMatchModal fixture={openFixture} onClose={() => setOpenFixture(null)} />
    </div>
  );
};

const NationsCupBoard: React.FC<{ data: NationsCupResponse; onRefresh?: () => void }> = ({ data, onRefresh }) => {
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

const CompetitionTable: React.FC<{
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

// tieWinnerId resolves the side that won a finished tie: aggregate goals
// across legs, then penalties from the deciding leg. Unfinished ties and
// unresolvable shootouts yield null so no winner is ever invented.
export function tieWinnerId(group: CompetitionFixtureRow[]): string | null {
  if (group.length === 0) return null;
  const totals = new Map<string, number>();
  let pensWinner: string | null = null;
  let allFinished = true;
  for (const f of group) {
    if (f.status !== 'finished' || f.home_goals == null || f.away_goals == null) {
      allFinished = false;
      continue;
    }
    totals.set(f.home_id, (totals.get(f.home_id) ?? 0) + f.home_goals);
    totals.set(f.away_id, (totals.get(f.away_id) ?? 0) + f.away_goals);
    if (f.decided_by === 'penalties' && f.penalties && f.penalties.length >= 2) {
      pensWinner = f.penalties[0] > f.penalties[1] ? f.home_id : f.away_id;
    }
  }
  if (!allFinished) return null;
  const [a, b] = [...totals.entries()];
  if (!a) return null;
  if (!b) return a[0];
  if (a[1] !== b[1]) return a[1] > b[1] ? a[0] : b[0];
  return pensWinner;
}

const TieGroupView: React.FC<{
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

const KnockoutBoard: React.FC<{
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
              <TieGroupView key={group[0].tie_id ?? group[0].id} group={group} onWatchFixture={onWatchFixture} onOpenReport={onOpenReport} currentMatchweek={currentMatchweek} />
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
