import React, { useCallback, useEffect, useState } from 'react';
import { Medal, Trophy, Star } from 'lucide-react';
import { fetchCareerHistory, fetchNXGN50, type CareerHistory, type HistoryResultRow, type SeasonHistoryRow } from '../../services/api';
import type { Fixture, NXGNPlayer } from '../../types';
import { Card, ClubCrest, EmptyState, LoadingState, PanelHeader } from '../ui/ui';
import { cx } from '../../lib/format';
import { downloadJSON } from '../../lib/export';
import { PlayerNameButton } from '../clubs/PlayerSheet';
import { PostMatchModal } from '../postmatch/PostMatchBroadcast';
import { soundManager } from '../../audio/webAudio';

function stageLabel(stage: string): string {
  return (stage || 'GROUP_STAGE').replace(/_/g, ' ').toLowerCase();
}

const Honour: React.FC<{ kicker: string; title: string; sub?: string; tone?: 'gold' | 'slate' }> = ({
  kicker,
  title,
  sub,
  tone = 'slate',
}) => (
  <div className={cx('rounded-xl border p-3.5 min-w-0 transition-colors', tone === 'gold' ? 'border-brass/50 bg-brass/[0.08]' : 'border-line bg-ink/40')}>
    <p className={cx('eyebrow', tone === 'gold' && '!text-brass')}>{kicker}</p>
    <p className="font-display text-[16.5px] font-bold text-bone mt-1 truncate" title={title}>
      {title}
    </p>
    {sub && <p className="text-[12px] text-sage font-mono mt-0.5 truncate">{sub}</p>}
  </div>
);

const SeasonCard: React.FC<{ row: SeasonHistoryRow }> = ({ row }) => (
  <div className="console-card p-5 space-y-4 border border-brass/30">
    <div className="flex items-center justify-between gap-3 pb-3 border-b border-line/60">
      <div className="flex items-center gap-3 min-w-0">
        {row.champion && <ClubCrest club={row.champion} size={44} />}
        <div className="min-w-0">
          <p className="eyebrow !text-brass">{row.season_name} {row.world ? 'European world' : 'Champion'}</p>
          <h3 className="font-display text-[22px] font-bold text-bone mt-0.5 truncate">
            {row.league_champions?.[0]?.club_name ?? row.champion?.club_name ?? 'League champions'}
          </h3>
          <p className="text-[12.5px] text-sage font-mono mt-0.5">
            {row.champion ? `${row.champion.pts} pts` : 'Champion archived'}
            {row.runner_up ? ` · Runner-up ${row.runner_up.club_name}` : ''}
          </p>
        </div>
      </div>
      <div className="w-10 h-10 rounded-xl bg-brass/15 border border-brass/40 flex items-center justify-center shrink-0">
        <Trophy size={20} className="text-brass" />
      </div>
    </div>
    {row.recap && (
      <p className="text-[14px] text-bone/85 leading-relaxed italic">{row.recap}</p>
    )}
    <div className="grid grid-cols-1 sm:grid-cols-2 gap-2.5">
      <Honour kicker="Champions League" title={row.ucl_champion?.club_name ?? 'Not decided'} sub={row.ucl_champion?.short_name} tone="gold" />
      <Honour kicker={row.europa_champion ? 'Europa League' : 'Cups & Europe'} title={row.europa_champion?.club_name ?? row.super_cup_champion?.club_name ?? 'Not decided'} sub={row.conference_champion?.club_name ?? row.super_cup_champion?.short_name} />
      <Honour
        kicker="Golden boot"
        title={row.top_scorer?.full_name ?? '—'}
        sub={row.top_scorer ? `${row.top_scorer.goals} goals` : undefined}
      />
      <Honour
        kicker="Playmaker"
        title={row.top_assister?.full_name ?? '—'}
        sub={row.top_assister ? `${row.top_assister.assists} assists` : undefined}
      />
      <Honour
        kicker="Player of the season"
        title={row.player_of_the_season?.full_name ?? '—'}
        sub={
          row.player_of_the_season
            ? `${row.player_of_the_season.short_name ?? ''} · ${row.player_of_the_season.goals ?? 0} G · ${row.player_of_the_season.assists ?? 0} A`
            : undefined
        }
      />
      <Honour
        kicker="Golden boy"
        title={row.golden_boy?.full_name ?? '—'}
        sub={row.golden_boy ? `${row.golden_boy.ovr} OVR` : undefined}
        tone="gold"
      />
    </div>
    {row.table && row.table.length > 0 && (
      <div className="border-t border-line/60 pt-3">
        <p className="eyebrow mb-2">Final Podium</p>
        <div className="space-y-1.5">
          {row.table.map((c, i) => (
            <div key={c.short_name} className="flex items-center justify-between text-[13px] font-mono px-3 py-1.5 rounded-lg bg-cardLight/50 border border-line/40">
              <span className="flex items-center gap-2 text-sage">
                <span className={cx('font-bold', i === 0 ? 'text-brass' : 'text-sage')}>{i + 1}.</span>
                <ClubCrest club={c} size={18} />
                <span className="text-bone font-semibold">{c.club_name}</span>
              </span>
              <span className="text-brass font-semibold">{c.pts} pts</span>
            </div>
          ))}
        </div>
      </div>
    )}
  </div>
);

export const HistoryTab: React.FC = () => {
  const [data, setData] = useState<CareerHistory | null>(null);
  const [loading, setLoading] = useState(true);
  const [subTab, setSubTab] = useState<'seasons' | 'cabinet' | 'records' | 'nxgn'>('seasons');
  const [nxgnList, setNxgnList] = useState<NXGNPlayer[]>([]);
  const [nxgnLoading, setNxgnLoading] = useState(false);
  const [openFixture, setOpenFixture] = useState<Fixture | null>(null);

  const openCompleted = (row: HistoryResultRow) => {
    const id = row.fixture_id || row.id;
    if (!id) return;
    soundManager.playClick();
    setOpenFixture({ ...row, id, fixture_id: id } as unknown as Fixture);
  };

  const load = useCallback(() => {
    fetchCareerHistory()
      .then(setData)
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const loadNxgn = useCallback(() => {
    if (nxgnList.length > 0) return;
    setNxgnLoading(true);
    fetchNXGN50()
      .then((res) => setNxgnList(res.rankings || []))
      .finally(() => setNxgnLoading(false));
  }, [nxgnList.length]);

  if (loading || !data) return <Card><LoadingState message="Loading dynasty records…" /></Card>;

  const cur = data.current;
  const past = [...data.past].reverse();
  const cabinet = data.trophy_cabinet ?? [];
  const records = data.all_time_records;

  return (
    <div className="space-y-5">
      <Card>
        <PanelHeader
          kicker="Multi-Decade Dynasty Archive"
          title="History & Records"
          subtitle="All-time honours, global Trophy Cabinet, record book milestones, and annual NXGN 50 wonderkid rankings."
          right={
            <div className="flex items-center gap-2">
              <div className="section-tabs border rounded-sm" role="tablist">
              {(['seasons', 'cabinet', 'records', 'nxgn'] as const).map((t) => (
                <button
                  key={t}
                  onClick={() => {
                    setSubTab(t);
                    if (t === 'nxgn') loadNxgn();
                  }}
                  className={cx(
                    'section-tab py-2 capitalize',
                    subTab === t ? 'section-tab-active' : ''
                  )}
                >
                  {t === 'seasons' ? 'Campaigns' : t === 'cabinet' ? 'Trophy Cabinet' : t === 'records' ? 'Record Book' : 'NXGN 50'}
                </button>
              ))}
              </div>
              <button
                type="button"
                onClick={() => downloadJSON('records', { seasons: data?.past ?? [], trophies: data?.trophy_cabinet ?? [], records: data?.all_time_records ?? [] })}
                className="min-h-9 border border-line bg-cardLight px-3 text-[12px] font-semibold text-sage hover:border-brass/30 hover:text-bone"
              >
                Export JSON
              </button>
            </div>
          }
        />
      </Card>

      {/* 1. SEASONS ARCHIVE VIEW */}
      {subTab === 'seasons' && (
        <>
          <Card>
            <div className="flex flex-wrap items-start justify-between gap-3 mb-4">
              <div>
                <p className="eyebrow !text-brass">{data.season_name} · in progress</p>
                <h3 className="font-display text-[24px] font-semibold text-bone mt-1">This season</h3>
                <p className="text-[13px] text-sage mt-1">
                  Matchweek {Math.min(data.current_matchweek, data.max_matchweeks)} of {data.max_matchweeks}
                  {' · '}Europe {stageLabel(data.ucl_stage)}
                </p>
              </div>
              <Trophy size={22} className="text-brass" />
            </div>

            {cur ? (
              <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-2.5">
                {(cur.league_champions && cur.league_champions.length > 0
                  ? cur.league_champions.map((champ) => (
                    <Honour
                      key={champ.competition_id || champ.league || champ.short_name}
                      kicker={champ.league || 'League champions'}
                      title={champ.club_name ?? '—'}
                      sub={champ.pts != null ? `${champ.pts} pts` : undefined}
                      tone="gold"
                    />
                  ))
                  : (
                    <Honour
                      kicker="Table leaders"
                      title={cur.super_league_champion?.club_name ?? '—'}
                      sub={cur.super_league_champion ? `${cur.super_league_champion.pts} pts` : undefined}
                      tone="gold"
                    />
                  ))}
                <Honour
                  kicker="Champions League"
                  title={cur.ucl_champion?.club_name ?? 'Still to be decided'}
                  sub={cur.ucl_champion?.short_name}
                />
                <Honour
                  kicker={cur.europa_champion || cur.conference_champion ? 'Europa & Conference' : 'Cups & Europe'}
                  title={cur.europa_champion?.club_name ?? cur.super_cup_champion?.club_name ?? 'Still to be decided'}
                  sub={cur.conference_champion?.club_name ?? cur.super_cup_champion?.short_name}
                />
                <Honour
                  kicker="Golden boot"
                  title={cur.top_scorer?.full_name ?? '—'}
                  sub={cur.top_scorer ? `${cur.top_scorer.goals} goals` : undefined}
                />
                <Honour
                  kicker="Playmaker"
                  title={cur.top_assister?.full_name ?? '—'}
                  sub={cur.top_assister ? `${cur.top_assister.assists} assists` : undefined}
                />
                <Honour
                  kicker="Player of the season"
                  title={cur.player_of_the_season?.full_name ?? '—'}
                  sub={
                    cur.player_of_the_season
                      ? `${cur.player_of_the_season.short_name} · ${cur.player_of_the_season.goals} G · ${cur.player_of_the_season.assists} A`
                      : undefined
                  }
                />
                <Honour
                  kicker="Golden boy"
                  title={cur.golden_boy?.full_name ?? '—'}
                  sub={cur.golden_boy ? `${cur.golden_boy.ovr} OVR · ${cur.golden_boy.goals} G` : undefined}
                  tone="gold"
                />
              </div>
            ) : (
              <EmptyState message="Play the opening matchweek and the races will appear here." className="py-10" />
            )}

            {data.table.length > 0 && (
              <div className="mt-4 grid grid-cols-1 sm:grid-cols-3 gap-2">
                {data.table.map((c, i) => (
                  <div key={c.short_name} className="flex items-center gap-2 px-3 py-2 rounded-lg border border-line bg-ink/40">
                    <span className="font-mono text-brass font-bold w-5">{i + 1}</span>
                    <span className="text-[13px] text-bone font-semibold truncate">{c.club_name}</span>
                    <span className="ml-auto font-mono text-[12px] text-sage">{c.pts} pts</span>
                  </div>
                ))}
              </div>
            )}

            {(data.recent_results?.length ?? 0) > 0 && (
              <div className="mt-5 border-t border-line/60 pt-4">
                <p className="match-section-title">Recent results</p>
                <div className="mt-2 divide-y divide-white/[0.07]">
                  {data.recent_results!.map((row) => (
                    <button
                      key={row.id}
                      type="button"
                      onClick={() => openCompleted(row)}
                      className="grid w-full grid-cols-[auto_minmax(0,1fr)_auto_minmax(0,1fr)] items-center gap-2 py-2 text-left text-[12px] hover:bg-white/[0.03]"
                    >
                      <span className="font-mono text-[10px] text-sage">MW{row.matchweek}</span>
                      <span className="flex min-w-0 items-center gap-1.5">
                        <ClubCrest club={row.home} size={16} />
                        <span className="truncate text-bone">{row.home.short_name}</span>
                      </span>
                      <strong className="font-mono tabular-nums text-brass">{row.home_goals}–{row.away_goals}</strong>
                      <span className="flex min-w-0 items-center justify-end gap-1.5">
                        <span className="truncate text-right text-bone">{row.away.short_name}</span>
                        <ClubCrest club={row.away} size={16} />
                      </span>
                    </button>
                  ))}
                </div>
              </div>
            )}
          </Card>

          <div className="flex items-center gap-2 text-[13px] text-sage">
            <Medal size={14} />
            <span>{past.length === 0 ? 'No completed seasons yet. Finish a campaign to archive it.' : `${past.length} completed ${past.length === 1 ? 'season' : 'seasons'}`}</span>
          </div>

          {past.length > 0 && (
            <div className="grid grid-cols-1 xl:grid-cols-2 gap-4">
              {past.map((row) => (
                <SeasonCard key={row.season_name} row={row} />
              ))}
            </div>
          )}
        </>
      )}

      {/* 2. TROPHY CABINET VIEW */}
      {subTab === 'cabinet' && (
        <div className="space-y-4">
          <Card>
            <div className="p-2 space-y-3">
              <div className="flex items-center justify-between pb-3 border-b border-line">
                <div>
                  <h3 className="font-display font-semibold text-lg text-bone">European Trophy Cabinet</h3>
                  <p className="text-xs text-sage">All-time titles won across the Top Five European world and legacy Super League dynasties.</p>
                </div>
                <Trophy size={24} className="text-brass" />
              </div>

              <div className="space-y-2.5">
                {cabinet.map((club, idx) => (
                  <div
                    key={club.club_id}
                    className="p-4 rounded-xl border border-line bg-cardLight/60 hover:bg-cardLight transition-all flex flex-wrap items-center justify-between gap-4 console-card-hover"
                  >
                    <div className="flex items-center gap-3.5 min-w-0">
                      <span className={cx('font-mono font-bold text-sm w-6', idx === 0 ? 'text-brass font-display text-[16px]' : 'text-sage')}>
                        {idx + 1}.
                      </span>
                      <ClubCrest club={club} size={40} />
                      <div className="min-w-0">
                        <p className="font-semibold text-[15.5px] text-bone truncate">{club.club_name}</p>
                        <p className="text-xs text-sage font-mono mt-0.5">
                          {club.ucl_count} UCL · {club.super_league_count} League · {club.super_cup_count} Super Cup
                        </p>
                        <p className="text-[11px] font-mono mt-1">
                          <span className="text-sage">Heritage {club.hist_total ?? '—'}</span>
                          <span className="text-sage mx-1.5">·</span>
                          <span className="text-brass font-semibold">Dynasty {club.career_total ?? 0}</span>
                        </p>
                      </div>
                    </div>

                    <div className="flex items-center gap-3">
                      <div className="text-right flex items-center gap-2">
                        <Trophy size={18} className="text-brass" />
                        <div>
                          <span className="text-2xl font-bold font-mono text-brass">{club.total_trophies}</span>
                          <span className="text-[10px] text-sage block -mt-1 font-mono uppercase tracking-wider">Trophies</span>
                        </div>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </Card>
        </div>
      )}

      {/* 3. ALL-TIME RECORD BOOK VIEW */}
      {subTab === 'records' && records && (
        <div className="space-y-5">
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
            <div className="p-4 rounded-xl bg-card border border-brass/35 space-y-1.5">
              <p className="eyebrow !text-brass">Single-Match Goals Record</p>
              <p className="text-lg font-bold text-bone">{records.single_match_goals_record.player_name}</p>
              <p className="text-xs text-sage font-mono">{records.single_match_goals_record.goals} goals · {records.single_match_goals_record.fixture}</p>
            </div>

            <div className="p-4 rounded-xl bg-card border border-line space-y-1.5">
              <p className="eyebrow">Highest Scoring Thriller</p>
              <p className="text-lg font-bold text-bone">{records.highest_scoring_match.score}</p>
              <p className="text-xs text-sage font-mono">{records.highest_scoring_match.total_goals} total goals</p>
            </div>

            <div className="p-4 rounded-xl bg-card border border-line space-y-1.5">
              <p className="eyebrow">Biggest Margin of Victory</p>
              <p className="text-lg font-bold text-bone">{records.biggest_margin_victory.score}</p>
              <p className="text-xs text-sage font-mono">+{records.biggest_margin_victory.margin} goal differential</p>
            </div>

            <div className="p-4 rounded-xl bg-card border border-line space-y-1.5">
              <p className="eyebrow">Highest Season Points</p>
              <p className="text-lg font-bold text-bone">{records.highest_season_points.club_name}</p>
              <p className="text-xs text-sage font-mono">{records.highest_season_points.points} pts ({records.highest_season_points.season_name})</p>
            </div>

            <div className="p-4 rounded-xl bg-card border border-brass/35 space-y-1.5">
              <p className="eyebrow !text-brass">Highest Rated Wonderkid</p>
              <p className="text-lg font-bold text-bone">{records.wonderkid_milestones.highest_ovr.name}</p>
              <p className="text-xs text-brass font-mono">{records.wonderkid_milestones.highest_ovr.ovr} OVR · {records.wonderkid_milestones.highest_ovr.potential} Potential</p>
            </div>

            <div className="p-4 rounded-xl bg-card border border-line space-y-1.5">
              <p className="eyebrow">Top Prodigy Goalscorer</p>
              <p className="text-lg font-bold text-bone">{records.wonderkid_milestones.top_prodigy_goals.name}</p>
              <p className="text-xs text-sage font-mono">{records.wonderkid_milestones.top_prodigy_goals.goals} career goals</p>
            </div>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-5">
            <Card>
              <p className="eyebrow mb-3">All-Time Top Scorers</p>
              <div className="space-y-1.5">
                {records.top_goalscorers.map((p, idx) => (
                  <div key={p.player_id} className="flex items-center justify-between p-2 rounded-lg bg-ink/30 text-xs font-mono">
                    <span className="flex items-center gap-2 truncate">
                      <span className="text-sage w-5">{idx + 1}.</span>
                      <PlayerNameButton playerId={p.player_id} className="font-semibold text-bone truncate">{p.full_name}</PlayerNameButton>
                      <span className="text-sage">({p.short_name})</span>
                      {p.is_wonderkid && <span className="text-[10px] px-1 py-0.2 rounded bg-brass/20 text-brass">WK</span>}
                    </span>
                    <span className="font-bold text-bone">{p.goals} G</span>
                  </div>
                ))}
              </div>
            </Card>

            <Card>
              <p className="eyebrow mb-3">All-Time Top Playmakers</p>
              <div className="space-y-1.5">
                {records.top_assisters.map((p, idx) => (
                  <div key={p.player_id} className="flex items-center justify-between p-2 rounded-lg bg-ink/30 text-xs font-mono">
                    <span className="flex items-center gap-2 truncate">
                      <span className="text-sage w-5">{idx + 1}.</span>
                      <PlayerNameButton playerId={p.player_id} className="font-semibold text-bone truncate">{p.full_name}</PlayerNameButton>
                      <span className="text-sage">({p.short_name})</span>
                      {p.is_wonderkid && <span className="text-[10px] px-1 py-0.2 rounded bg-brass/20 text-brass">WK</span>}
                    </span>
                    <span className="font-bold text-bone">{p.assists} A</span>
                  </div>
                ))}
              </div>
            </Card>
          </div>
        </div>
      )}

      {/* 4. NXGN 50 VIEW */}
      {subTab === 'nxgn' && (
        <div className="space-y-4">
          <Card>
            <div className="flex items-center justify-between pb-3 border-b border-line">
              <div>
                <h3 className="font-display font-semibold text-lg text-bone">NXGN 50 Scouting Dossier</h3>
                <p className="text-xs text-sage">Official scouting rankings of the top 50 teenage wonderkids in world football.</p>
              </div>
              <Star size={24} className="text-brass" />
            </div>

            {nxgnLoading ? (
              <LoadingState message="Compiling international scouting reports…" />
            ) : (
              <div className="space-y-2 mt-4">
                {nxgnList.map((wk) => (
                  <div
                    key={wk.player_id}
                    className="p-3.5 rounded-xl border border-line bg-cardLight/50 hover:bg-cardLight transition-colors flex flex-col md:flex-row md:items-center justify-between gap-3"
                  >
                    <div className="flex items-center gap-3 min-w-0">
                      <span className="font-mono font-bold text-sm text-brass w-7 shrink-0">#{wk.rank}</span>
                      <div className="min-w-0">
                        <div className="flex items-center gap-2 flex-wrap">
                          <PlayerNameButton playerId={wk.player_id} className="font-bold text-sm text-bone hover:text-brass">
                            {wk.full_name}
                          </PlayerNameButton>
                          <span className="text-xs text-sage font-mono">({wk.short_name} · {wk.position} · {wk.age}y)</span>
                          {wk.personality_title && (
                            <span className="text-[10px] font-mono px-1.5 py-0.5 rounded bg-brass/15 text-brass border border-brass/30">
                              {wk.personality_title}
                            </span>
                          )}
                        </div>
                        <p className="text-xs text-sage/90 mt-1 italic leading-relaxed">
                          "{wk.scout_verdict}"
                        </p>
                      </div>
                    </div>

                    <div className="flex items-center gap-4 shrink-0 font-mono text-xs text-right">
                      {wk.mentor_name && (
                        <div className="hidden sm:block text-left text-[11px] text-sage">
                          <span className="block text-bone/70">Mentor</span>
                          <span>{wk.mentor_name}</span>
                        </div>
                      )}
                      <div>
                        <span className="text-base font-bold text-bone">{wk.ovr}</span>
                        <span className="text-[10px] text-brass block">POT {wk.potential}</span>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </Card>
        </div>
      )}
      <PostMatchModal fixture={openFixture} onClose={() => setOpenFixture(null)} />
    </div>
  );
};
