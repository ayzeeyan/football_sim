import React, { useEffect, useMemo, useState } from 'react';
import type { Club, Fixture, MatchEventItem, MatchTickPayload } from '../types';
import { PitchCanvas } from './PitchCanvas';
import { Play, Pause, RotateCcw, ArrowLeftRight, Users, Sparkle, FastForward, Zap } from 'lucide-react';
import { soundManager } from '../audio/webAudio';
import { MATCH_SPEEDS } from '../lib/constants';
import { cx, stripEmojis } from '../lib/format';
import { Card, ClubCrest, FormPips, GhostButton, PanelHeader } from './ui/ui';
import { fetchFixtures, fetchWeekWatch, setFavourite, simulateRemaining, type WeekWatch } from '../services/api';
import { PlayerNameButton } from './PlayerSheet';
import { PreMatchModal } from './PreMatchModal';
import { getWeatherDetails } from './MatchCard';
import { HalfTimeDugout } from './HalfTimeDugout';

interface MatchdayTabProps {
  homeClub: Club | null;
  awayClub: Club | null;
  matchData: MatchTickPayload | null;
  onOpenClubModal: (target: 'home' | 'away') => void;
  onSwapTeams: () => void;
  onKickoff: () => void;
  onPause: () => void;
  onSetSpeed: (speed: number) => void;
  onReset: () => void;
  onNextFixture: () => void;
  onSeek70?: () => void;
  onSeekChance?: () => void;
  onJumpToFixture?: (f: Fixture) => void;
  onWeekAdvanced?: () => void;
}

function statusLabel(state: MatchTickPayload['state'], minute: number): string {
  if (state === 'NOT_STARTED') return 'Pre-match';
  if (state === 'FULL_TIME') return 'Full time';
  if (state === 'HALF_TIME') return "45' · Half-time";
  if (state === 'PAUSED') return `${Math.floor(minute)} min, paused`;
  return `${Math.floor(minute)} min`;
}

function eventLabel(e: MatchEventItem): string {
  const name = e.scorer?.full_name || e.player?.full_name || 'Unknown';
  const minute = e.display || `${e.minute}'`;
  if (e.type === 'own_goal') return `${minute} ${name} (og)`;
  if (e.type === 'penalty') return `${minute} ${name} (pen)`;
  if (e.type === 'penalty_miss') return `${minute} ${name} (missed pen)`;
  if (e.type === 'red') return `${minute} ${name} sent off`;
  if (e.type === 'yellow') return `${minute} ${name} booked`;
  if (e.type === 'sub') {
    const inn = e.player_in?.full_name ?? 'On';
    const out = e.player_out?.full_name ?? 'Off';
    return `${minute} ${inn} on for ${out}`;
  }
  return `${minute} ${name}`;
}

const LiveScorers: React.FC<{ events: MatchEventItem[]; home?: string; away?: string }> = ({ events }) => {
  const notable = events.filter((e) => e.type === 'goal' || e.type === 'penalty' || e.type === 'own_goal' || e.type === 'red' || e.type === 'sub');
  if (notable.length === 0) return null;
  const homeLines = notable.filter((e) => {
    if (e.type === 'own_goal') return (e.beneficiary ?? (e.side === 'home' ? 'away' : 'home')) === 'home';
    return e.side === 'home';
  });
  const awayLines = notable.filter((e) => {
    if (e.type === 'own_goal') return (e.beneficiary ?? (e.side === 'home' ? 'away' : 'home')) === 'away';
    return e.side === 'away';
  });
  return (
    <div className="grid grid-cols-2 gap-4 text-[12px] text-sage">
      <div className="text-left space-y-0.5">
        {homeLines.map((e, i) => (
          <div key={`h-${e.minute}-${i}`} className={e.type === 'red' ? 'text-ember' : 'text-bone/80'}>
            {eventLabel(e)}
          </div>
        ))}
      </div>
      <div className="text-right space-y-0.5">
        {awayLines.map((e, i) => (
          <div key={`a-${e.minute}-${i}`} className={e.type === 'red' ? 'text-ember' : 'text-bone/80'}>
            {eventLabel(e)}
          </div>
        ))}
      </div>
    </div>
  );
};

const LineupList: React.FC<{ title: string; rows: MatchTickPayload['home_coords'] }> = ({ title, rows }) => (
  <div>
    <div className="text-[13px] text-sage pb-2 mb-2 border-b border-line">{title}</div>
    <div className="space-y-1">
      {rows.map((item, idx) => (
        <div
          key={item.player.player_id ?? idx}
          className={cx(
            'flex justify-between items-center px-2.5 py-1.5 rounded-lg border',
            item.sent_off
              ? 'bg-ember/10 text-bone/50 border-ember/35'
              : item.player.universe_wonderkid
                ? 'bg-brass/10 text-bone border-brass/40'
                : 'bg-ink/40 text-bone/75 border-transparent',
          )}
        >
          <span className="flex items-center gap-2 truncate">
            <span className="text-sage font-mono font-semibold text-[11px] w-8">{item.player.position}</span>
            <span className="truncate text-[13px] font-medium">
              <PlayerNameButton playerId={item.player.player_id}>{item.player.full_name}</PlayerNameButton>
            </span>
            {item.player.universe_wonderkid && <Sparkle size={12} className="text-brass shrink-0" />}
            {item.sent_off && <span className="text-[11px] text-ember">Sent off</span>}
          </span>
          <span className="font-display font-semibold text-[14px] shrink-0 text-bone/80">{item.player.ovr}</span>
        </div>
      ))}
    </div>
  </div>
);

const LiveBench: React.FC<{ title: string; rows: NonNullable<MatchTickPayload['home_bench']> }> = ({ title, rows }) => {
  if (!rows.length) return null;
  return (
    <div className="mt-3">
      <p className="text-[11px] font-mono uppercase tracking-[0.1em] text-sage mb-1">{title}</p>
      <div className="space-y-0.5">
        {rows.map((row) => (
          <div key={row.player.player_id} className="flex items-center justify-between gap-2 text-[12.5px] py-0.5">
            <span className={cx('truncate', row.status === 'on' ? 'text-bone' : 'text-sage')}>
              <PlayerNameButton playerId={row.player.player_id}>{row.player.full_name}</PlayerNameButton>
              {row.on_minute != null && <span className="ml-1.5 font-mono text-brass">↑{row.on_minute}'</span>}
            </span>
            <span className="font-mono text-sage shrink-0">{row.status === 'on' ? 'on' : 'bench'}</span>
          </div>
        ))}
      </div>
    </div>
  );
};

export const MatchdayTab: React.FC<MatchdayTabProps> = ({
  homeClub,
  awayClub,
  matchData,
  onOpenClubModal,
  onSwapTeams,
  onKickoff,
  onPause,
  onSetSpeed,
  onReset,
  onNextFixture,
  onSeek70,
  onSeekChance,
  onJumpToFixture,
  onWeekAdvanced,
}) => {
  const [showLineups, setShowLineups] = useState(false);
  const [slateFixture, setSlateFixture] = useState<Fixture | null>(null);
  const [previewOpen, setPreviewOpen] = useState(false);
  const [watch, setWatch] = useState<WeekWatch | null>(null);
  const [simRestMsg, setSimRestMsg] = useState<string | null>(null);
  const [simRestBusy, setSimRestBusy] = useState(false);
  const autoSimFor = React.useRef<string | null>(null);
  const currentSpeed = matchData?.speed ?? 1;
  const matchState = matchData?.state ?? 'NOT_STARTED';
  const leagueFixture = matchData?.league_fixture;
  const decidedLeague = leagueFixture?.status === 'finished';
  const liveEvents = matchData?.match_events ?? [];
  const liveWeather = slateFixture?.weather || matchData?.weather;
  const favId = watch?.favourite_club_id ?? '';
  const isMyClubLive = !!favId && !!homeClub && !!awayClub &&
    (homeClub.club_id === favId || awayClub.club_id === favId);
  const canSeek = matchState === 'PLAYING' || matchState === 'PAUSED';

  const runSimRest = React.useCallback(async (excludeId: string) => {
    setSimRestBusy(true);
    try {
      const res = await simulateRemaining(excludeId);
      if (res.status === 'error') {
        setSimRestMsg('Could not sim the rest of the week. Try again.');
      } else {
        setSimRestMsg(res.played === 0 ? 'Nothing left to play this week.' : `${res.played} other games decided. Results stand.`);
        onWeekAdvanced?.();
      }
    } finally {
      setSimRestBusy(false);
    }
  }, [onWeekAdvanced]);

  // Watch-one week: keep the cup-jump offer fresh, and once my club's live
  // result stands at FT, auto-sim the rest (excluding that fixture).
  // Ticks keep flowing at FT, so guard on fixture id, not object identity.
  useEffect(() => {
    if (matchState !== 'FULL_TIME' || !decidedLeague || !leagueFixture) return;
    if (autoSimFor.current !== leagueFixture.id) {
      autoSimFor.current = leagueFixture.id;
      setSimRestMsg(null);
      let cancelled = false;
      fetchWeekWatch().then((w) => { if (!cancelled) setWatch(w); }).catch(() => undefined);
      void runSimRest(leagueFixture.id);
      return () => { cancelled = true; };
    }
  }, [matchState, decidedLeague, leagueFixture, runSimRest]);

  useEffect(() => {
    if (matchState === 'NOT_STARTED') {
      fetchWeekWatch().then(setWatch).catch(() => undefined);
    }
  }, [matchState]);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement | null;
      if (target && ['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName)) return;
      if (e.key === ' ' || e.code === 'Space') {
        e.preventDefault();
        if (matchState === 'PLAYING') onPause();
        else if (matchState !== 'HALF_TIME') onKickoff();
      } else if (e.key === '1') onSetSpeed(1);
      else if (e.key === '2') onSetSpeed(2);
      else if (e.key === '3') onSetSpeed(5);
      else if (e.key === '4') onSetSpeed(999);
      else if (e.key === '7' && canSeek) onSeek70?.();
      else if ((e.key === 'c' || e.key === 'C') && canSeek) onSeekChance?.();
      else if (e.key === 'l' || e.key === 'L') setShowLineups((v) => !v);
      else if (e.key === 'r' || e.key === 'R') onReset();
      else if ((e.key === 'n' || e.key === 'N') && matchState === 'FULL_TIME') onNextFixture();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [matchState, canSeek, onKickoff, onPause, onSetSpeed, onSeek70, onSeekChance, onReset, onNextFixture]);

  const stats = useMemo(
    () => [
      { label: 'Shots', value: `${matchData?.home_shots ?? 0} – ${matchData?.away_shots ?? 0}` },
      { label: 'On target', value: `${matchData?.home_shots_on_target ?? 0} – ${matchData?.away_shots_on_target ?? 0}` },
      { label: 'Corners', value: `${matchData?.home_corners ?? 0} – ${matchData?.away_corners ?? 0}` },
      {
        label: 'Field tilt',
        value:
          (matchData?.possession_momentum ?? 0) > 0
            ? `+${Math.round((matchData?.possession_momentum ?? 0) * 100)}%`
            : `${Math.round((matchData?.possession_momentum ?? 0) * 100)}%`,
      },
    ],
    [matchData],
  );

  const homePoss = matchData?.home_possession_pct ?? 50;
  const awayPoss = matchData?.away_possession_pct ?? 50;
  const isPlaying = matchState === 'PLAYING';

  useEffect(() => {
    if (!homeClub || !awayClub) {
      setSlateFixture(null);
      return;
    }
    let cancelled = false;
    fetchFixtures()
      .then((res) => {
        if (cancelled) return;
        const fx = res.fixtures.find(
          (f) => f.home.club_id === homeClub.club_id && f.away.club_id === awayClub.club_id,
        );
        setSlateFixture(fx ?? null);
      })
      .catch(() => {
        if (!cancelled) setSlateFixture(null);
      });
    return () => {
      cancelled = true;
    };
  }, [homeClub?.club_id, awayClub?.club_id]);

  return (
    <div className="space-y-4">
      {matchState === 'HALF_TIME' && matchData && <HalfTimeDugout match={matchData} home={homeClub?.short_name ?? 'Home'} away={awayClub?.short_name ?? 'Away'} />}
      {(homeClub || favId) && (
        <Card>
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div className="min-w-0">
              <p className="eyebrow">Watch-one week</p>
              <p className="text-[14px] text-bone mt-1">
                {favId
                  ? <>My club: <span className="font-semibold">{favId}</span>{isMyClubLive ? ' — live now.' : ' — this match is not mine.'}</>
                  : 'Lock a club to watch its match live; the rest sims at full time.'}
              </p>
            </div>
            <div className="flex items-center gap-2">
              {homeClub && favId !== homeClub.club_id && (
                <button
                  type="button"
                  onClick={() => { soundManager.playClick(); setFavourite(homeClub.club_id).then(() => fetchWeekWatch().then(setWatch).catch(() => undefined)).catch(() => undefined); }}
                  className="px-3 py-2 text-[13px] font-semibold text-bone border border-line bg-cardLight hover:bg-cardHover"
                >
                  Lock {homeClub.short_name} as my club
                </button>
              )}
              {favId === homeClub?.club_id && (
                <span className="px-3 py-2 text-[12px] font-mono text-brass border border-brass/40 bg-brass/10">My club locked</span>
              )}
            </div>
          </div>
        </Card>
      )}
      <div className="border border-line overflow-hidden">
        <div className="fascia grid grid-cols-1 md:grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] items-stretch">
          <button
            type="button"
            onClick={() => {
              soundManager.playClick();
              onOpenClubModal('home');
            }}
            className="flex items-center gap-3 px-4 py-3 text-left hover:bg-cardHover transition-colors min-w-0"
          >
            <ClubCrest club={homeClub} size={48} />
            <span className="min-w-0">
              <span className="block font-display text-[22px] leading-none font-semibold truncate">
                {homeClub?.club_name ?? 'Select home'}
              </span>
              <span className="block text-[13px] text-sage mt-1">
                {homeClub?.short_name}
                {(matchData?.home_manager || homeClub?.manager) && (
                  <span className="text-[11px] text-sage/80 block mt-0.5">
                    {matchData?.home_manager?.name ?? homeClub?.manager?.name} ({(matchData?.home_manager?.archetype_label ?? homeClub?.manager?.style)})
                    {matchData?.home_tactical_stance && matchData.home_tactical_stance !== 'NORMAL' && (
                      <span className="ml-1.5 px-1.5 py-0.5 rounded text-[10px] font-mono font-bold bg-brass/20 text-brass border border-brass/40">
                        {matchData.home_tactical_stance}
                      </span>
                    )}
                  </span>
                )}
              </span>
            </span>
          </button>

          <div className="flex flex-col items-center justify-center px-6 py-3 bg-obsidian">
            <div className="score-display text-[56px] leading-none text-bone">
              {matchData?.home_score ?? 0}
              <span className="text-sage/70 mx-2 text-[36px]">–</span>
              {matchData?.away_score ?? 0}
            </div>
            <p className="mt-1 text-[13px] font-semibold">
              {matchState === 'PLAYING' && <span className="text-ember mr-1.5">Live</span>}
              <span className="text-sage">{statusLabel(matchState, matchData?.minute ?? 0)}</span>
            </p>
            {leagueFixture && (
              <p className="text-[12px] text-sage mt-0.5">
                {decidedLeague ? 'Result stands' : 'This result will stand'}
              </p>
            )}
            <div className="flex flex-wrap items-center justify-center gap-1.5 mt-1.5">
              {liveWeather && (() => {
                const w = getWeatherDetails(liveWeather);
                return w ? (
                  <span className="px-2 py-0.5 rounded border border-line bg-cardLight text-bone/90 text-[10px] font-mono inline-flex items-center gap-1">
                    <span>{w.icon}</span> <span>{w.label}</span>
                  </span>
                ) : null;
              })()}
              {(slateFixture?.derby_name || (slateFixture?.is_derby && slateFixture?.derby)) && (() => {
                const title = slateFixture.derby_name || slateFixture.derby;
                const heat = slateFixture.derby_heat ?? 50;
                const isHigh = slateFixture.is_high_heat_derby || heat >= 70;
                return (
                  <span
                    className={cx(
                      'px-2 py-0.5 rounded border text-[10px] font-semibold uppercase tracking-[0.08em] inline-flex items-center gap-1',
                      isHigh
                        ? 'border-ember/70 bg-ember/15 text-ember animate-pulse shadow-[0_0_8px_rgba(224,86,36,0.3)]'
                        : 'border-ember/40 bg-ember/10 text-ember',
                    )}
                  >
                    <span>🔥</span> {title} · {heat}° Heat
                  </span>
                );
              })()}
            </div>
          </div>

          <button
            type="button"
            onClick={() => {
              soundManager.playClick();
              onOpenClubModal('away');
            }}
            className="flex items-center justify-end gap-3 px-4 py-3 text-right hover:bg-cardHover transition-colors min-w-0"
          >
            <span className="min-w-0">
              <span className="block font-display text-[22px] leading-none font-semibold truncate">
                {awayClub?.club_name ?? 'Select away'}
              </span>
              <span className="block text-[13px] text-sage mt-1">
                {awayClub?.short_name}
                {(matchData?.away_manager || awayClub?.manager) && (
                  <span className="text-[11px] text-sage/80 block mt-0.5">
                    {matchData?.away_tactical_stance && matchData.away_tactical_stance !== 'NORMAL' && (
                      <span className="mr-1.5 px-1.5 py-0.5 rounded text-[10px] font-mono font-bold bg-brass/20 text-brass border border-brass/40">
                        {matchData.away_tactical_stance}
                      </span>
                    )}
                    {matchData?.away_manager?.name ?? awayClub?.manager?.name} ({(matchData?.away_manager?.archetype_label ?? awayClub?.manager?.style)})
                  </span>
                )}
              </span>
            </span>
            <ClubCrest club={awayClub} size={48} />
          </button>
        </div>

        {liveEvents.length > 0 && (
          <div className="px-4 py-2 border-b border-line bg-dugout">
            <LiveScorers events={liveEvents} />
          </div>
        )}

        <PitchCanvas matchData={matchData} homeClub={homeClub} awayClub={awayClub} className="border-0" />
      </div>

      {matchState === 'NOT_STARTED' && slateFixture && slateFixture.status === 'scheduled' && (
        <Card>
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div className="min-w-0">
              <p className="eyebrow">Pre-match</p>
              <p className="text-[15px] text-bone mt-1 leading-relaxed">
                {slateFixture.preview?.kickoff_note ?? `${homeClub?.club_name} vs ${awayClub?.club_name}`}
              </p>
              <div className="mt-3 flex flex-wrap items-center gap-4">
                <FormPips form={slateFixture.preview?.home_form ?? homeClub?.form ?? []} size="sm" />
                <span className="text-sage text-[12px]">vs</span>
                <FormPips form={slateFixture.preview?.away_form ?? awayClub?.form ?? []} size="sm" />
              </div>
            </div>
            <button
              onClick={() => {
                soundManager.playClick();
                setPreviewOpen(true);
              }}
              className="px-4 py-2.5 bg-cardLight hover:bg-cardHover text-bone border border-line text-[13px] font-semibold"
            >
              Open pre-match centre
            </button>
          </div>
        </Card>
      )}

      {matchState !== 'NOT_STARTED' && ((matchData?.home_bench?.length ?? 0) > 0 || (matchData?.away_bench?.length ?? 0) > 0) && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Card>
            <LiveBench title={`${homeClub?.short_name ?? 'Home'} bench`} rows={matchData?.home_bench ?? []} />
          </Card>
          <Card>
            <LiveBench title={`${awayClub?.short_name ?? 'Away'} bench`} rows={matchData?.away_bench ?? []} />
          </Card>
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-4 items-start">
        <div className="lg:col-span-8 space-y-4 min-w-0">
          {showLineups && matchData && (
            <Card>
              <PanelHeader title="Lineups" />
              <div className="mt-4 grid grid-cols-1 md:grid-cols-2 gap-5">
                <div>
                  <LineupList title={homeClub?.club_name ?? 'Home'} rows={matchData.home_coords} />
                  <LiveBench title="Bench" rows={matchData.home_bench ?? []} />
                </div>
                <div>
                  <LineupList title={awayClub?.club_name ?? 'Away'} rows={matchData.away_coords} />
                  <LiveBench title="Bench" rows={matchData.away_bench ?? []} />
                </div>
              </div>
            </Card>
          )}

          <Card>
            <div className="flex items-end justify-between gap-3">
              <p className="text-[13px] text-sage">Possession</p>
              <p className="font-display text-[18px] font-semibold">
                {homePoss}% <span className="text-sage font-sans text-[13px] font-medium">{awayPoss}%</span>
              </p>
            </div>
            <div className="mt-2 w-full h-1.5 bg-line overflow-hidden flex">
              <div className="bg-bone h-full transition-[width] duration-300" style={{ width: `${homePoss}%` }} />
              <div className="bg-sage/40 h-full transition-[width] duration-300" style={{ width: `${awayPoss}%` }} />
            </div>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-px bg-line mt-4">
              {stats.map((s) => (
                <div key={s.label} className="py-3 px-3 bg-dugout text-center">
                  <div className="text-[12px] text-sage">{s.label}</div>
                  <div className="score-display text-[22px] text-bone mt-0.5">{s.value}</div>
                </div>
              ))}
            </div>
          </Card>
        </div>

        <div className="lg:col-span-4 min-w-0 lg:sticky lg:top-24 space-y-4">
          <Card>
            <PanelHeader title="Run the match" />
            <button
              onClick={isPlaying ? onPause : onKickoff}
              disabled={matchState === 'HALF_TIME'}
              className={cx(
                'mt-4 w-full px-6 py-3.5 font-semibold text-[15px] flex items-center justify-center gap-2 transition-colors',
                isPlaying
                  ? 'bg-cardLight hover:bg-cardHover text-bone border border-line'
                  : 'bg-bone hover:bg-[#fff6dc] text-ink',
              )}
            >
              {isPlaying ? (
                <>
                  <Pause size={17} aria-hidden="true" /> Pause match
                </>
              ) : matchState === 'FULL_TIME' ? (
                <>
                  <RotateCcw size={17} aria-hidden="true" /> {decidedLeague ? 'Friendly rematch' : 'Play rematch'}
                </>
              ) : (
                <>
                  <Play size={17} aria-hidden="true" /> {matchState === 'HALF_TIME' ? 'Choose in the dugout above' : matchState === 'PAUSED' ? 'Resume match' : 'Start kickoff'}
                </>
              )}
            </button>

            {matchState === 'FULL_TIME' && decidedLeague && (
              <p className="mt-2 text-[13px] text-sage">This result already stands. A rematch is exhibition only.</p>
            )}

            {matchState === 'FULL_TIME' && (
              <button
                onClick={() => {
                  soundManager.playWhistle();
                  onNextFixture();
                }}
                className="mt-2 w-full px-6 py-3 font-semibold text-[14px] flex items-center justify-center gap-2 transition-colors bg-cardLight hover:bg-cardHover text-bone border border-line"
              >
                <FastForward size={15} aria-hidden="true" /> Next fixture
              </button>
            )}

            {matchState === 'FULL_TIME' && decidedLeague && leagueFixture && (
              <div className="mt-3 border border-line bg-ink/40 p-3">
                <p className="eyebrow">Watch-one week</p>
                <p className="text-[13px] text-bone mt-1">Mine is decided. The rest of the slate sims without replaying this result.</p>
                {simRestMsg && <p className="text-[13px] text-sage mt-1">{simRestMsg}</p>}
                <button
                  type="button"
                  disabled={simRestBusy}
                  onClick={() => { soundManager.playWhistle(); void runSimRest(leagueFixture.id); }}
                  className="mt-2 w-full px-4 py-2.5 font-semibold text-[13px] bg-cardLight hover:bg-cardHover text-bone border border-line disabled:opacity-50"
                >
                  {simRestBusy ? 'Simming…' : 'Sim the rest of the week'}
                </button>
                {(watch?.same_week_cups?.length ?? 0) > 0 && (
                  <div className="mt-2">
                    <p className="text-[11px] font-mono uppercase tracking-[0.1em] text-sage mb-1">Same-week cup — jump in</p>
                    <div className="space-y-1">
                      {watch!.same_week_cups.slice(0, 3).map((cup) => (
                        <button
                          key={cup.id}
                          type="button"
                          onClick={() => { soundManager.playClick(); onJumpToFixture?.(cup); }}
                          className="w-full text-left px-3 py-2 border border-line bg-dugout hover:bg-cardHover text-[13px] text-bone"
                        >
                          {cup.home.short_name} vs {cup.away.short_name}
                          <span className="ml-2 font-mono text-[11px] text-sage">{cup.competition === 'ucl' ? 'Champions Cup' : 'Super Cup'} · MW {cup.matchweek}</span>
                        </button>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            )}

            <p className="text-[13px] text-sage mt-5 mb-2">Speed</p>
            <div className="grid grid-cols-4 gap-px bg-line" role="group" aria-label="Match speed">
              {MATCH_SPEEDS.map((s) => (
                <button
                  key={s.value}
                  onClick={() => {
                    soundManager.playClick();
                    onSetSpeed(s.value);
                  }}
                  aria-pressed={currentSpeed === s.value}
                  className={cx(
                    'px-2 py-2 font-display text-[16px] font-semibold transition-colors bg-dugout',
                    currentSpeed === s.value ? 'bg-bone text-ink' : 'text-sage hover:text-bone',
                  )}
                >
                  {s.label}
                </button>
              ))}
            </div>

            <p className="mt-3 text-[12px] text-sage">Space pause, 1–4 speed, 7 to 70′, C next chance, L lineups, R reset, N next</p>

            <div className="mt-2 grid grid-cols-2 gap-2">
              <button
                type="button"
                disabled={!canSeek}
                onClick={() => { soundManager.playClick(); onSeek70?.(); }}
                className="px-2 py-2 text-[13px] font-semibold border border-line bg-cardLight text-bone hover:bg-cardHover disabled:opacity-40 inline-flex items-center justify-center gap-1.5"
              >
                <FastForward size={14} aria-hidden="true" /> 70′
              </button>
              <button
                type="button"
                disabled={!canSeek}
                onClick={() => { soundManager.playClick(); onSeekChance?.(); }}
                className="px-2 py-2 text-[13px] font-semibold border border-line bg-cardLight text-bone hover:bg-cardHover disabled:opacity-40 inline-flex items-center justify-center gap-1.5"
              >
                <Zap size={14} aria-hidden="true" /> Next chance
              </button>
            </div>
            {!canSeek && matchState !== 'FULL_TIME' && matchState !== 'NOT_STARTED' && (
              <p className="mt-1.5 text-[12px] text-sage">Seeks unlock once the ball is rolling — the dugout still owns half-time.</p>
            )}

            <div className="mt-3 grid grid-cols-3 gap-2">
              <GhostButton active={showLineups} onClick={() => { soundManager.playClick(); setShowLineups((v) => !v); }} className="justify-center px-2">
                <Users size={14} aria-hidden="true" /> Lineups
              </GhostButton>
              <GhostButton onClick={() => { soundManager.playClick(); onReset(); }} className="justify-center px-2">
                <RotateCcw size={14} aria-hidden="true" /> Reset
              </GhostButton>
              <GhostButton onClick={() => { soundManager.playClick(); onSwapTeams(); }} className="justify-center px-2">
                <ArrowLeftRight size={14} aria-hidden="true" /> Swap
              </GhostButton>
            </div>
          </Card>

          <Card className="flex flex-col h-[480px]">
            <PanelHeader title="Commentary" />
            <div className="flex-1 min-h-0 overflow-y-auto mt-3 pr-1">
              {(!matchData?.commentary || matchData.commentary.length === 0) && (
                <div className="text-sage text-[14px] py-16">Start kickoff to hear the feed.</div>
              )}
              {matchData?.commentary?.map((item, idx) => {
                const isGoal = item.category === 'GOAL';
                const isFullTime = item.category === 'FULLTIME';
                return (
                  <div
                    key={`${item.timestamp}-${idx}`}
                    className={cx(
                      'py-2.5 border-b border-line/70 text-[14px] leading-relaxed',
                      isGoal || isFullTime ? 'text-bone' : 'text-bone/80',
                    )}
                  >
                    <span className="font-display font-semibold text-[15px] text-sage mr-2">{item.timestamp}</span>
                    {isGoal && <span className="text-ember font-semibold mr-1.5">Goal</span>}
                    {isFullTime && <span className="text-bone font-semibold mr-1.5">Full time</span>}
                    {item.is_wonderkid && <span className="text-bone/70 mr-1.5">U-14</span>}
                    {stripEmojis(item.text)}
                  </div>
                );
              })}
            </div>
          </Card>
        </div>
      </div>

      <PreMatchModal
        fixture={previewOpen ? slateFixture : null}
        onClose={() => setPreviewOpen(false)}
        onWatch={() => { setPreviewOpen(false); onKickoff(); }}
      />
    </div>
  );
};
