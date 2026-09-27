import React, { useCallback, useEffect, useMemo, useState } from 'react';
import type { CompetitionDetail, CompetitionFixtureRow, CompetitionSummary, Fixture } from '../../../types';
import { fetchClubs, fetchCompetition, fetchCompetitions, fetchFavourite, fetchNationsCup } from '../../../services/api';
import { PostMatchModal } from '../../postmatch/PostMatchBroadcast';
import { useAsyncData } from '../../../hooks/useAsyncData';
import { Crown } from 'lucide-react';
import { cx } from '../../../lib/format';
import { soundManager } from '../../../audio/webAudio';
import { Card, ClubDot, ErrorState, LoadingState, PanelHeader } from '../../ui/ui';
import { usePlayerSheet } from '../../clubs/PlayerSheet';

import { COUNTRY_ORDER, KindIcon, LEAGUE_BY_COMPETITION, asClub, groupByMatchweek, isWatchable, kindLabel, kindTone, legLabel, scoreLine, type CompetitionHubTabProps } from './helpers';
import { CompetitionTable, KnockoutBoard, NationsCupBoard } from './boards';

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
