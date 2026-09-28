import React, { useCallback, useEffect, useRef, useState } from 'react';
import type { Club, Fixture, LeagueChange } from '../../types';
import { fetchSuperLeague, fetchFixtureSummaries, fetchFixture, simulateFixture, simulateRemaining, resetSeason, restartSeason, fetchScoringRace, fetchSeasonStats, fetchCalendar, fetchSuperCup, fetchWeekWatch, fetchLeagueChanges, type WeekWatch } from '../../services/api';
import type { ScoringRaceRow, SeasonStats, CalendarState, SuperCupState } from '../../services/api';
import type { SuperLeagueState, FixturesResponse } from '../../types';
import { Trophy, RotateCcw, Users, Zap, CalendarDays } from 'lucide-react';
import { soundManager } from '../../audio/webAudio';
import { formatGd, cx, stripEmojis } from '../../lib/format';
import { Card, ClubCrest, ConfirmBar, FormPips, LoadingState, OvrBadge, PanelHeader, PrimaryButton, ProgressBar } from '../ui/ui';
import { qualificationBand, qualificationBarClass } from '../../lib/qualification';
import { LEAGUES_5 } from '../../lib/constants';
import { downloadCSV } from '../../lib/export';
import { t } from '../../i18n';
import { MatchCard } from '../matches/MatchCard';
import { PostMatchModal } from '../postmatch/PostMatchBroadcast';
import { PreMatchModal } from '../prematch/PreMatchModal';
import { usePlayerSheet } from '../clubs/PlayerSheet';
import { UclTournamentTab } from './UclTournamentTab';
import { CalendarStrip } from '../prematch/CalendarStrip';
import { SuperCupBoard } from './SuperCupBoard';

interface StandingsTabProps {
  onWatchFixture: (fixture: Fixture) => void;
  onViewSquad: (club: Club) => void;
  onShowToast: (msg: string) => void;
  onOpenCeremony: () => void;
  onSeasonTick?: () => void;
}

function positionMarker(pos: number): string {
  if (pos === 1) return 'bg-brass';
  if (pos <= 4) return 'bg-[#8AB4C8]';
  if (pos <= 6) return 'bg-pitchtone';
  return 'bg-sage/30';
}

// cupBadge shortens any cup competition id for the same-week jump buttons.
// Legacy ids keep their historic badges; world cups map by family.
export function cupBadge(competition: string): string {
  switch (competition) {
    case 'ucl':
      return 'CU';
    case 'super-cup':
      return 'SC';
    case 'champions-league':
      return 'UCL';
    case 'europa-league':
      return 'UEL';
    case 'conference-league':
      return 'UECL';
    case 'fa-cup':
      return 'FAC';
    case 'efl-cup':
      return 'EFL';
    case 'copa-del-rey':
      return 'CDR';
    case 'dfb-pokal':
      return 'DFB';
    case 'coppa-italia':
      return 'CI';
    case 'coupe-de-france':
      return 'CDF';
    default:
      return 'Cup';
  }
}

// Table narrative strip (F8): pure math off the current table — title pace,
// Europe race, relegation scrap. Derived from props/state only, so the text
// moves whenever the table moves. No resim.
export function tableNarrative(clubs: Club[], maxMatchweeks: number = 38, world = true): { title: string; europe: string; scrap: string } {
  if (clubs.length === 0) return { title: 'No table yet.', europe: 'No table yet.', scrap: 'No table yet.' };
  const played = clubs[0]?.p ?? 0;
  if (played === 0) {
    return {
      title: 'Title pace: too early — opening series decides the first gaps.',
      europe: world ? 'Europe race: too early — top sides reach UEFA competitions.' : 'Europe race: too early — 2nd–4th reach the Champions Cup.',
      scrap: 'Relegation scrap: too early — the bottom three are all level.',
    };
  }
  const [first, second] = [clubs[0], clubs[1]];
  const gap = (first?.pts ?? 0) - (second?.pts ?? 0);
  const pace = ((first?.pts ?? 0) / Math.max(1, played)) * (maxMatchweeks || 38);
  const title = gap <= 0
    ? `Title pace: level at the top — ${first?.short_name ?? '?'} and ${second?.short_name ?? '?'} on ${first?.pts ?? 0} (${Math.round(pace)}-pt pace).`
    : `Title pace: ${first?.short_name} lead by ${gap} (${first?.pts} pts, ${Math.round(pace)}-pt pace).`;
  const contenders = clubs.slice(1, 5);
  const spread = (contenders[0]?.pts ?? 0) - (contenders[contenders.length - 1]?.pts ?? 0);
  const europe = spread === 0
    ? `Europe race: ${contenders.map((c) => c.short_name).join(', ')} all level on ${contenders[0]?.pts ?? 0} for 2nd–4th.`
    : `Europe race: ${contenders.map((c) => c.short_name).join(', ')} split by ${spread} for 2nd–4th.`;
  const bottom = clubs.slice(-3);
  const bSpread = (bottom[0]?.pts ?? 0) - (bottom[bottom.length - 1]?.pts ?? 0);
  const scrap = bSpread === 0
    ? `Relegation scrap: ${bottom.map((c) => c.short_name).join(', ')} all on ${bottom[0]?.pts ?? 0}.`
    : `Relegation scrap: ${bottom.map((c) => c.short_name).join(', ')} split by ${bSpread} at the bottom.`;
  return { title, europe, scrap };
}

export const StandingsTab: React.FC<StandingsTabProps> = ({ onWatchFixture, onViewSquad, onShowToast, onOpenCeremony, onSeasonTick }) => {
  const { openPlayer } = usePlayerSheet();
  const [league, setLeague] = useState<SuperLeagueState | null>(null);
  const [fixtures, setFixtures] = useState<FixturesResponse | null>(null);
  const [race, setRace] = useState<ScoringRaceRow[]>([]);
  const [stats, setStats] = useState<SeasonStats | null>(null);
  const [calendar, setCalendar] = useState<CalendarState | null>(null);
  const [superCup, setSuperCup] = useState<SuperCupState | null>(null);
  const [viewMw, setViewMw] = useState<number | null>(null);
  const [loading, setLoading] = useState(true);
  const [busyId, setBusyId] = useState<string | null>(null);
  const [simulatingAll, setSimulatingAll] = useState(false);
  const [resetting, setResetting] = useState(false);
  const [confirmReset, setConfirmReset] = useState(false);
  const [confirmRestart, setConfirmRestart] = useState(false);
  const [restarting, setRestarting] = useState(false);
  const [openFixture, setOpenFixture] = useState<Fixture | null>(null);
  const [previewFixture, setPreviewFixture] = useState<Fixture | null>(null);
  const [watch, setWatch] = useState<WeekWatch | null>(null);
  const [leagueSelector, setLeagueSelector] = useState<string>(LEAGUES_5[0]);
  const [leagueChanges, setLeagueChanges] = useState<Record<string, LeagueChange>>({});
  const fixturesRef = useRef<HTMLDivElement>(null);

  const loadAll = useCallback(async (mw?: number) => {
    const [lg, fx, rc, st, cal, sc] = await Promise.all([
      fetchSuperLeague(leagueSelector),
      fetchFixtureSummaries(mw),
      fetchScoringRace(),
      fetchSeasonStats(),
      fetchCalendar(),
      fetchSuperCup(),
    ]);
    setLeague(lg);
    setFixtures(fx);
    setRace(rc);
    setStats(st);
    setCalendar(cal);
    setSuperCup(sc);
    setViewMw((prev) => prev ?? fx.current_matchweek);
  }, [leagueSelector]);

  useEffect(() => {
    loadAll().finally(() => setLoading(false));
    fetchWeekWatch().then(setWatch).catch(() => undefined);
    fetchLeagueChanges()
      .then((res) => setLeagueChanges(Object.fromEntries(res.moves.map((m) => [m.club_id, m]))))
      .catch(() => undefined);
  }, [loadAll]);

  useEffect(() => {
    if (viewMw == null && fixtures) setViewMw(fixtures.current_matchweek);
  }, [fixtures, viewMw]);

  const reloadFixtures = useCallback(async (mw: number) => {
    const fx = await fetchFixtureSummaries(mw);
    setFixtures(fx);
  }, []);

  /** After a result changes the table, refresh standings + fixtures together. */
  const reloadAfterResult = useCallback(
    async (mw: number) => {
      const [lg, fx, rc, st] = await Promise.all([
        fetchSuperLeague(leagueSelector),
        fetchFixtureSummaries(mw),
        fetchScoringRace(),
        fetchSeasonStats(),
      ]);
      setLeague(lg);
      setFixtures(fx);
      setRace(rc);
      setStats(st);
    },
    [leagueSelector],
  );

  const handleSimulateOne = async (f: Fixture) => {
    soundManager.playWhistle();
    setBusyId(f.id);
    try {
      const res = await simulateFixture(f.id);
      if (res.status === 'error') {
        onShowToast(res.message || 'That result already stands and cannot be replayed.');
      } else {
        onShowToast(`${f.home.short_name} ${res.home_goals} – ${res.away_goals} ${f.away.short_name}. Result stands.`);
        if (res.ucl_event) onShowToast(res.ucl_event);
        if (res.rolled_over && res.is_finished) {
          onShowToast(`Season complete. ${res.champion ?? 'The leaders'} take the title.`);
        } else if (res.rolled_over) {
          onShowToast('Matchweek complete. The table moves on.');
        }
      }
      await reloadAfterResult(viewMw ?? f.matchweek);
      onSeasonTick?.();
    } finally {
      setBusyId(null);
    }
  };

  const handleSimulateRemaining = async () => {
    soundManager.playWhistle();
    setSimulatingAll(true);
    try {
      const res = await simulateRemaining();
      onShowToast(res.played === 0 ? 'Nothing left to play this week.' : `${res.played} games decided this week. Results stand.`);
      if (res.is_finished) onShowToast(`Season complete. ${res.champion ?? 'The leaders'} take the title.`);
      await reloadAfterResult(viewMw ?? fixtures?.current_matchweek ?? 1);
      onSeasonTick?.();
    } catch {
      onShowToast('Could not simulate the remaining games. Try again.');
    } finally {
      setSimulatingAll(false);
    }
  };

  const handleReset = async () => {
    soundManager.playWhistle();
    setResetting(true);
    try {
      const res = await resetSeason();
      onShowToast(stripEmojis(res.message));
      if (res.league_changes && res.league_changes.length > 0) {
        onShowToast(`${res.league_changes.length} clubs crossed league boundaries in the closed pyramid — badged Up/Down in the tables.`);
      }
      setViewMw(1);
      await loadAll(1);
    } catch {
      onShowToast('Could not reset the season. Try again.');
    } finally {
      setResetting(false);
    }
  };

  const handleRestart = async () => {
    soundManager.playWhistle();
    setRestarting(true);
    try {
      const res = await restartSeason();
      onShowToast(stripEmojis(res.message));
      setViewMw(1);
      await loadAll(1);
    } catch {
      onShowToast('Could not restart the season. Try again.');
    } finally {
      setRestarting(false);
    }
  };

  if (loading || !league) return <Card><LoadingState message="Loading the table…" /></Card>;

  const isFinished = league.current_matchweek > league.max_matchweeks;
  const champion = league.clubs[0];
  const shownMw = Math.min(viewMw ?? league.current_matchweek, league.max_matchweeks);
  const weekFixtures = fixtures && fixtures.matchweek === shownMw ? fixtures.fixtures : [];
  const unplayed = weekFixtures.filter((f) => f.status === 'scheduled').length;

  return (
    <div className="space-y-5">
      {isFinished && champion && (
        <div className="panel-pad border-brass/50 flex flex-wrap items-center justify-between gap-4 animate-fade-in">
          <div className="flex items-center gap-4">
            <div className="w-16 h-16 rounded-xl bg-brass flex items-center justify-center text-ink">
              <Trophy size={34} />
            </div>
            <div>
              <p className="eyebrow !text-brass">Champions · {league.max_matchweeks} of {league.max_matchweeks} matchweeks</p>
              <h3 className="font-display text-[30px] font-semibold text-bone mt-1">{champion.club_name} {league.world && champion.league ? `win ${champion.league}` : league.world ? 'top their league' : 'win the Super League'}</h3>
              <p className="text-[14px] text-sage font-normal mt-0.5">
                {champion.pts} points · {champion.w} wins · {formatGd(champion.gd)} goal difference
              </p>
            </div>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <PrimaryButton tone="brass" onClick={() => setConfirmReset(true)} disabled={resetting}>
              <RotateCcw size={15} aria-hidden="true" /> {resetting ? 'Starting…' : 'Start Next Season'}
            </PrimaryButton>
            <PrimaryButton tone="cyan" onClick={() => { soundManager.playClick(); onOpenCeremony(); }}>
              <Trophy size={15} aria-hidden="true" /> Awards Night
            </PrimaryButton>
          </div>
        </div>
      )}

      {confirmReset && (
        <ConfirmBar
          message="Start a new season? Current table, cup ties and window deals will be archived and reset."
          confirmLabel="Start New Season"
          busyLabel="Starting…"
          busy={resetting}
          onConfirm={() => {
            setConfirmReset(false);
            void handleReset();
          }}
          onCancel={() => setConfirmReset(false)}
        />
      )}


      {confirmRestart && (
        <ConfirmBar
          message="Restart this season from matchweek 1? This season's results, table and records are wiped; all-time history is kept."
          confirmLabel="Restart Season"
          busyLabel="Restarting…"
          busy={restarting}
          onConfirm={() => {
            setConfirmRestart(false);
            void handleRestart();
          }}
          onCancel={() => setConfirmRestart(false)}
        />
      )}

      <Card>
        {league.world && (
          <div className="flex flex-wrap items-center gap-1.5 px-4 pt-4 sm:px-5" role="group" aria-label="Domestic league selector">
            {LEAGUES_5.map((name) => (
              <button
                key={name}
                type="button"
                onClick={() => { soundManager.playClick(); setLeagueSelector(name); }}
                aria-pressed={leagueSelector === name}
                className={cx(
                  'min-h-9 border px-3 text-[12px] font-semibold transition-colors',
                  leagueSelector === name
                    ? 'border-brass/60 bg-brass/15 text-brass'
                    : 'border-line bg-cardLight text-sage hover:border-brass/30 hover:text-bone',
                )}
              >
                {name}
              </button>
            ))}
            <button
              type="button"
              onClick={() => downloadCSV(`standings-${leagueSelector.toLowerCase().replace(/ /g, '-')}`, league.clubs.map((c, i) => ({ position: i + 1, club: c.club_name, league: c.league, played: c.p, won: c.w, drawn: c.d, lost: c.l, goals_for: c.gf, goals_against: c.ga, goal_difference: c.gd, points: c.pts })))}
              className="min-h-9 border border-line bg-cardLight px-3 text-[12px] font-semibold text-sage hover:border-brass/30 hover:text-bone"
            >
              {t('action.exportCsv')}
            </button>
          </div>
        )}
        <PanelHeader
          kicker={isFinished ? 'Season complete' : `Matchweek ${Math.min(league.current_matchweek, league.max_matchweeks)} of ${league.max_matchweeks}`}
          title={league.world && league.clubs[0]?.league ? `${league.clubs[0].league} table` : 'League table'}
          subtitle={
            league.world
              ? 'Selected domestic league — home and away once. Cups and Europe share the same authoritative world slate. Decided games lock.'
              : 'Quadruple round-robin, 44 games each. Champions Cup and Super Cup sit on the same slate. Decided games lock.'
          }
          right={
            !isFinished ? (
              <div className="flex flex-wrap items-center gap-2">
                <PrimaryButton
                  tone="brass"
                  onClick={() => fixturesRef.current?.scrollIntoView({ behavior: 'smooth', block: 'start' })}
                >
                  <CalendarDays size={15} aria-hidden="true" /> This Week’s Fixtures
                </PrimaryButton>
                <PrimaryButton tone="blue" onClick={() => setConfirmRestart(true)} disabled={restarting}>
                  <RotateCcw size={15} aria-hidden="true" /> Restart Season
                </PrimaryButton>
              </div>
            ) : (
              <PrimaryButton tone="brass" onClick={() => setConfirmReset(true)} disabled={resetting}>
                <RotateCcw size={15} aria-hidden="true" /> Start New Season
              </PrimaryButton>
            )
          }
        />
      </Card>

      <Card>
        <p className="eyebrow mb-2">Table narrative · matchweek {Math.min(league.current_matchweek, league.max_matchweeks)}</p>
        {(() => {
          const n = tableNarrative(league.clubs, league.max_matchweeks, league.world);
          return (
            <div className="grid grid-cols-1 md:grid-cols-3 gap-2">
              {[
                { label: 'Title pace', text: n.title },
                { label: 'Europe race', text: n.europe },
                { label: 'Relegation scrap', text: n.scrap },
              ].map((s) => (
                <div key={s.label} className="rounded-xl border border-line bg-ink/40 px-3.5 py-3">
                  <p className="text-[11px] font-mono uppercase tracking-[0.1em] text-brass">{s.label}</p>
                  <p className="text-[13.5px] text-bone mt-1 leading-relaxed">{s.text}</p>
                </div>
              ))}
            </div>
          );
        })()}
      </Card>

      {calendar && (
        <CalendarStrip
          calendar={calendar}
          viewMw={shownMw}
          onPickWeek={(mw) => {
            soundManager.playClick();
            setViewMw(mw);
            void reloadFixtures(mw);
          }}
        />
      )}

      {(stats?.player_of_the_week || race.length > 0) && (
        <div className="grid grid-cols-1 xl:grid-cols-3 gap-4">
          {stats?.player_of_the_week && (
            <button
              type="button"
              className="text-left"
              onClick={() => stats.player_of_the_week?.player_id && openPlayer(stats.player_of_the_week.player_id)}
            >
              <Card>
                <p className="eyebrow">Player of the week · MW {stats.player_of_the_week.matchweek}</p>
                <p className="font-display text-[22px] font-semibold text-bone mt-2 truncate">{stats.player_of_the_week.full_name}</p>
                <p className="text-[13px] text-sage font-mono mt-1">
                  {stats.player_of_the_week.club_short} · {stats.player_of_the_week.position} · {stats.player_of_the_week.rating} rating
                </p>
                <p className="text-[13px] text-bone/80 mt-2 font-mono">
                  {stats.player_of_the_week.goals} G · {stats.player_of_the_week.assists} A
                  {stats.player_of_the_week.is_wonderkid ? ' · U-17' : ''}
                </p>
              </Card>
            </button>
          )}
          <Card className={stats?.player_of_the_week ? 'xl:col-span-2' : 'xl:col-span-3'}>
            <div className="flex items-center justify-between gap-2 mb-3">
              <p className="eyebrow">Golden boot race · all players</p>
              <span className="font-mono text-[11px] text-sage">Top 5</span>
            </div>
            <div className="grid grid-cols-2 md:grid-cols-5 gap-2.5">
              {(stats?.scorers?.slice(0, 5) ?? race).map((r, i) => (
                <button
                  type="button"
                  key={r.player_id || r.full_name}
                  onClick={() => r.player_id && openPlayer(r.player_id)}
                  className={cx('p-3 rounded-xl border text-left', i === 0 ? 'border-brass/50 bg-brass/[0.07]' : 'border-line bg-ink/40')}
                >
                  <div className="flex items-baseline justify-between gap-2">
                    <span className={cx('font-display font-semibold text-[22px]', i === 0 ? 'text-brass' : 'text-bone')}>{i + 1}</span>
                    <span className="font-mono font-bold text-[15px] text-bone">{r.goals}<span className="text-sage font-semibold text-[12px]"> G · {r.assists} A</span></span>
                  </div>
                  <p className="font-semibold text-[13px] text-bone truncate mt-1" title={r.full_name}>
                    {r.full_name} {r.is_wonderkid && <span className="text-brass">· U-17</span>}
                  </p>
                  <p className="text-[11.5px] text-sage font-mono truncate">{r.club_short} · {r.position}</p>
                </button>
              ))}
            </div>
          </Card>
        </div>
      )}

      {stats?.monthly_awards && stats.monthly_awards.length > 0 && (
        <Card>
          <p className="eyebrow mb-3">Player of the month</p>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-2.5">
            {stats.monthly_awards.map((m) => (
              <button
                type="button"
                key={m.month}
                onClick={() => m.player_id && openPlayer(m.player_id)}
                className="p-3 rounded-xl border border-line bg-ink/40 text-left"
              >
                <p className="eyebrow !text-brass">{m.month}</p>
                <p className="font-semibold text-[14px] text-bone truncate mt-1">{m.full_name}</p>
                <p className="font-mono text-[12px] text-sage mt-0.5">{m.club_short} · {m.rating} avg · {m.apps} apps</p>
              </button>
            ))}
          </div>
        </Card>
      )}

      {stats && stats.assisters.length > 0 && (
        <Card>
          <div className="flex items-center justify-between gap-2 mb-3">
            <p className="eyebrow">Playmaker race · assists</p>
            <span className="font-mono text-[11px] text-sage">Top 5</span>
          </div>
          <div className="grid grid-cols-2 md:grid-cols-5 gap-2.5">
            {stats.assisters.slice(0, 5).map((r, i) => (
              <button
                type="button"
                key={`a-${r.player_id || r.full_name}`}
                onClick={() => r.player_id && openPlayer(r.player_id)}
                className={cx('p-3 rounded-xl border text-left', i === 0 ? 'border-pitchtone/50 bg-pitchtone/[0.07]' : 'border-line bg-ink/40')}
              >
                <div className="flex items-baseline justify-between gap-2">
                  <span className={cx('font-display font-semibold text-[22px]', i === 0 ? 'text-pitchtone' : 'text-bone')}>{i + 1}</span>
                  <span className="font-mono font-bold text-[15px] text-bone">{r.assists}<span className="text-sage font-semibold text-[12px]"> A · {r.goals} G</span></span>
                </div>
                <p className="font-semibold text-[13px] text-bone truncate mt-1">{r.full_name}</p>
                <p className="text-[11.5px] text-sage font-mono truncate">{r.club_short} · {r.position}</p>
              </button>
            ))}
          </div>
        </Card>
      )}

      {stats && stats.history.length > 0 && (
        <Card>
          <p className="eyebrow mb-3">Honours roll · previous seasons</p>
          <div className="space-y-2">
            {stats.history.slice().reverse().map((h) => (
              <div key={h.season_name} className="flex flex-wrap items-center justify-between gap-2 px-3 py-2 rounded-lg border border-line bg-ink/40 text-[13px]">
                <span className="font-mono text-brass font-semibold">{h.season_name}</span>
                <span className="text-bone">{h.champion?.club_name ?? '—'} · League</span>
                <span className="text-sage">{h.ucl_champion?.club_name ?? '—'} · Europe</span>
                <span className="font-mono text-sage">{h.top_scorer ? `${h.top_scorer.full_name} ${h.top_scorer.goals} G` : ''}</span>
              </div>
            ))}
          </div>
        </Card>
      )}

      <div className="console-card overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-[15px]">
            <thead className="table-head">
              <tr>
                {['Pos', 'Club', 'OVR', 'P', 'W', 'D', 'L', 'GF', 'GA', 'GD', 'PTS', 'Form'].map((h) => (
                  <th key={h} className="py-3.5 px-4 text-[12px]">{h}</th>
                ))}
                <th className="py-3.5 px-4 text-center text-[12px]">Open</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-line/70">
              {league.clubs.map((c, idx) => {
                const pos = idx + 1;
                const change = leagueChanges[c.club_id];
                const band = league.world ? qualificationBand(c.league, pos, league.clubs.length) : null;
                const bar = league.world ? qualificationBarClass(band) : positionMarker(pos);
                return (
                  <tr key={c.club_id} className={cx('hover:bg-cardLight/60 transition-colors', pos === 1 && 'bg-brass/[0.06]')}>
                    <td className="py-3.5 px-4 font-bold">
                      <span className="flex items-center gap-2.5">
                        <span className={cx('w-1.5 h-6 rounded-full shadow-sm', bar)} title={band === 'ucl' ? 'Champions League' : band === 'el' ? 'Europa League' : band === 'ecl' ? 'Conference League' : band === 'rel' ? 'Relegation' : undefined} />
                        <span className="font-mono text-[15px] text-bone/85">{pos}</span>
                      </span>
                    </td>
                    <td className="py-3.5 px-4 font-semibold text-bone">
                      <span className="flex items-center gap-3">
                        <ClubCrest club={c} size={28} />
                        <span className="text-[15px]">{c.club_name}</span>
                        {change && (
                          <span
                            title={change.direction === 'promoted' ? `Promoted from ${change.from_league} last season` : `Relegated from ${change.from_league} last season`}
                            className={cx('rounded px-1 py-0.5 text-[9px] font-bold uppercase tracking-wide', change.direction === 'promoted' ? 'bg-pitchtone/20 text-pitchtone' : 'bg-ember/20 text-ember')}
                          >
                            {change.direction === 'promoted' ? 'Up' : 'Down'}
                          </span>
                        )}
                        <span className="text-sage font-mono text-[12px]">[{c.short_name}]</span>
                      </span>
                    </td>
                    <td className="py-3.5 px-4">
                      <OvrBadge ovr={c.overall_team_rating} size="sm" />
                    </td>
                    <td className="py-3.5 px-4 text-bone/70 font-mono">{c.p}</td>
                    <td className="py-3.5 px-4 text-bone font-semibold font-mono">{c.w}</td>
                    <td className="py-3.5 px-4 text-sage font-mono">{c.d}</td>
                    <td className="py-3.5 px-4 text-sage font-mono">{c.l}</td>
                    <td className="py-3.5 px-4 text-sage font-mono">{c.gf}</td>
                    <td className="py-3.5 px-4 text-sage font-mono">{c.ga}</td>
                    <td className="py-3.5 px-4 font-semibold font-mono">
                      <span className={c.gd > 0 ? 'text-[#A9CDBB]' : c.gd < 0 ? 'text-[#D89A84]' : 'text-sage'}>{formatGd(c.gd)}</span>
                    </td>
                    <td className="py-3.5 px-4 font-bold text-bone font-mono text-[16px]">{c.pts}</td>
                    <td className="py-3.5 px-4">
                      <FormPips form={c.form ?? []} size="sm" />
                    </td>
                    <td className="py-3.5 px-4 text-center">
                      <span className="inline-flex items-center gap-1.5">
                        <button onClick={() => { soundManager.playClick(); onViewSquad(c); }} className="px-3 py-1.5 bg-transparent hover:bg-cardLight text-sage hover:text-bone rounded-lg border border-line text-[12.5px] font-semibold inline-flex items-center gap-1.5 transition-colors">
                          <Users size={13} /> Squad
                        </button>
                      </span>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
        <div className="px-5 py-3.5 bg-ink/40 border-t border-line flex flex-wrap items-center gap-x-6 gap-y-2 text-[13px] font-mono text-sage">
          <span className="flex items-center gap-2"><span className="w-2 h-2 rounded-full bg-brass" /> 1st takes the title</span>
          {league.world ? (
            <>
              <span className="flex items-center gap-2"><span className="w-2 h-2 rounded-full bg-brass" /> Champions League</span>
              <span className="flex items-center gap-2"><span className="w-2 h-2 rounded-full bg-[#8AB4C8]" /> Europa League</span>
              <span className="flex items-center gap-2"><span className="w-2 h-2 rounded-full bg-pitchtone" /> Conference League</span>
              <span className="flex items-center gap-2"><span className="w-2 h-2 rounded-full bg-ember" /> Relegation</span>
              {Object.keys(leagueChanges).length > 0 && (
                <>
                  <span className="flex items-center gap-2"><span className="rounded bg-pitchtone/20 px-1 text-[9px] font-bold uppercase text-pitchtone">Up</span> Promoted this season</span>
                  <span className="flex items-center gap-2"><span className="rounded bg-ember/20 px-1 text-[9px] font-bold uppercase text-ember">Down</span> Relegated this season</span>
                </>
              )}
            </>
          ) : (
            <>
              <span className="flex items-center gap-2"><span className="w-2 h-2 rounded-full bg-[#8AB4C8]" /> 2nd–4th reach the Champions Cup</span>
              <span className="flex items-center gap-2"><span className="w-2 h-2 rounded-full bg-pitchtone" /> Super Cup: seeds 1–4 bye, 5–12 play in</span>
            </>
          )}
        </div>
      </div>

      <div ref={fixturesRef} className="scroll-mt-36">
        <Card>
          <PanelHeader
            kicker={isFinished ? 'Season complete' : `Matchweek ${shownMw} of ${league.max_matchweeks}`}
            title={calendar?.weeks.find((w) => w.matchweek === shownMw)?.chapter ?? `Matchweek ${shownMw}`}
            subtitle={
              league.world
                ? 'Domestic league plus same-week cups and Europe on one list. Simulate one game, or the rest of the week. European ties play home and away.'
                : 'Super League and Champions Cup sit on one list. Simulate one game, or the rest of the week. Cup legs play in order.'
            }
            right={
              <div className="flex items-center gap-2">
                <select
                  value={shownMw}
                  onChange={(e) => {
                    const mw = Number(e.target.value);
                    setViewMw(mw);
                    reloadFixtures(mw);
                  }}
                  aria-label="Select matchweek"
                  className="field px-3 py-2 font-semibold font-mono"
                >
                  {Array.from({ length: league.max_matchweeks }, (_, i) => i + 1).map((mw) => (
                    <option key={mw} value={mw}>MW {mw}{mw === league.current_matchweek ? ' · now' : ''}</option>
                  ))}
                </select>
                {!isFinished && shownMw === league.current_matchweek && unplayed > 0 && (
                  <PrimaryButton tone="brass" onClick={handleSimulateRemaining} disabled={simulatingAll}>
                    <Zap size={15} aria-hidden="true" /> {simulatingAll ? 'Simulating…' : `Simulate remaining (${unplayed})`}
                  </PrimaryButton>
                )}
              </div>
            }
          />
          {weekFixtures.length > 0 && (
            <div className="mt-5 mb-4 p-4 rounded-xl border border-line bg-ink/40">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <p className="eyebrow">Fixture focus</p>
                  <p className="text-[13px] text-bone mt-1">
                    {watch?.fixture
                      ? <>Viewing fixture: <span className="font-semibold">{watch.fixture.home.short_name} vs {watch.fixture.away.short_name}</span>{watch.fixture.status === 'finished' ? ' (decided)' : ''}.</>
                      : 'Select a club to open its next fixture in the Match Centre.'}
                  </p>
                </div>
                <div className="flex flex-wrap items-center gap-2">
                  {watch?.fixture && watch.fixture.status === 'scheduled' && (
                    <PrimaryButton
                      tone="brass"
                      onClick={() => { soundManager.playClick(); onWatchFixture(watch.fixture!); }}
                    >
                      Open Match Centre
                    </PrimaryButton>
                  )}
                  {(watch?.same_week_cups?.length ?? 0) > 0 && (
                    <span className="font-mono text-[11px] text-sage">{watch!.same_week_cups.length} same-week cup{watch!.same_week_cups.length === 1 ? '' : 's'}</span>
                  )}
                </div>
              </div>
              {(watch?.same_week_cups?.length ?? 0) > 0 && (
                <div className="mt-2 flex flex-wrap gap-2">
                  {watch!.same_week_cups.slice(0, 4).map((cup) => (
                    <button
                      key={cup.id}
                      type="button"
                      onClick={() => { soundManager.playClick(); onWatchFixture(cup); }}
                      className="px-3 py-1.5 border border-line bg-dugout hover:bg-cardHover text-[12.5px] text-bone"
                    >
                      Jump: {cup.home.short_name} vs {cup.away.short_name} ({cupBadge(cup.competition)})
                    </button>
                  ))}
                </div>
              )}
            </div>
          )}
          {weekFixtures.length > 0 && (
            <div className="mt-5 mb-4 p-4 rounded-xl border border-line bg-ink/40">
              <div className="flex flex-wrap items-end justify-between gap-3 mb-3">
                <div>
                  <p className="eyebrow">Week progress</p>
                  <p className="font-display text-[28px] font-semibold text-bone leading-none mt-1">
                    {weekFixtures.length - unplayed}
                    <span className="text-sage text-[18px] font-sans font-semibold"> / {weekFixtures.length}</span>
                  </p>
                  <p className="text-[13px] text-sage mt-1.5">
                    {weekFixtures.length - unplayed === weekFixtures.length ? 'All games decided.' : `${unplayed} still to play.`}
                  </p>
                </div>
                <div className="text-right font-mono text-[12px] text-sage space-y-1">
                  <p>
                    League {weekFixtures.filter((f) => (f.competition === 'super-league' || ['premier-league', 'la-liga', 'bundesliga', 'serie-a', 'ligue-1'].includes(f.competition)) && f.status === 'finished').length}
                    /{weekFixtures.filter((f) => f.competition === 'super-league' || ['premier-league', 'la-liga', 'bundesliga', 'serie-a', 'ligue-1'].includes(f.competition)).length}
                  </p>
                  <p>
                    Europe {weekFixtures.filter((f) => ['ucl', 'champions-league', 'europa-league', 'conference-league'].includes(f.competition) && f.status === 'finished').length}
                    /{weekFixtures.filter((f) => ['ucl', 'champions-league', 'europa-league', 'conference-league'].includes(f.competition)).length || 0}
                  </p>
                  <p>
                    Cups {weekFixtures.filter((f) => ['super-cup', 'fa-cup', 'efl-cup', 'copa-del-rey', 'dfb-pokal', 'coppa-italia', 'coupe-de-france'].includes(f.competition) && f.status === 'finished').length}
                    /{weekFixtures.filter((f) => ['super-cup', 'fa-cup', 'efl-cup', 'copa-del-rey', 'dfb-pokal', 'coppa-italia', 'coupe-de-france'].includes(f.competition)).length || 0}
                  </p>
                </div>
              </div>
              <ProgressBar
                pct={weekFixtures.length ? ((weekFixtures.length - unplayed) / weekFixtures.length) * 100 : 0}
                className="h-2"
              />
            </div>
          )}
          <div className="grid grid-cols-1 xl:grid-cols-2 gap-4 mt-5">
            {weekFixtures.map((f) => (
              <MatchCard
                key={f.id}
                fixture={f}
                busyId={busyId}
                onOpenMatchCentre={(fx) => {
                  soundManager.playClick();
                  onWatchFixture(fx);
                }}
                onSimulate={handleSimulateOne}
                onOpen={(fx) => {
                  if (fx.status === 'finished') {
                    setOpenFixture(fx);
                    return;
                  }
                  void fetchFixture(fx.id)
                    .then((full) => setPreviewFixture(full ?? fx))
                    .catch(() => setPreviewFixture(fx));
                }}
              />
            ))}
          </div>
        </Card>
      </div>

      {league.world ? (
        <Card>
          <p className="text-[13px] text-sage leading-relaxed">
            Cups and Europe are in the <span className="text-bone font-semibold">Competitions</span> tab for this 96-club career — domestic cups plus three UEFA league phases and two-legged knockouts share the calendar above.
          </p>
        </Card>
      ) : (
        <>
          <UclTournamentTab
            key={`cup-${league.current_matchweek}-${weekFixtures.filter((f) => f.competition === 'ucl' && f.status === 'finished').length}`}
            boardOnly
            onWatchFixture={onWatchFixture}
            onShowToast={onShowToast}
          />

          <SuperCupBoard
            data={superCup}
            loading={!superCup}
          />
        </>
      )}

      <PostMatchModal fixture={openFixture} onClose={() => setOpenFixture(null)} />
      <PreMatchModal
        fixture={previewFixture}
        onClose={() => setPreviewFixture(null)}
        busy={busyId === previewFixture?.id}
        onSimulate={async (fx) => {
          await handleSimulateOne(fx);
          setPreviewFixture(null);
        }}
      />
    </div>
  );
};
