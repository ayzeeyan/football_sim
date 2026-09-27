import React, { useEffect, useId, useMemo, useState } from 'react';
import type { Club, Fixture } from '../../types';
import { Modal } from '../ui/ui';
import { usePlayerSheet } from '../clubs/PlayerSheet';
import { Activity, ArrowRight, BarChart3, List, Table2, Users } from 'lucide-react';
import { soundManager } from '../../audio/webAudio';
import { cx } from '../../lib/format';
import { composeMatchStory, normalizeTeamStats, otherResultsFor } from '../../lib/matchStory';
import { fetchFixture, fetchFixtureSummaries } from '../../services/api';
import { MatchResultHeader } from './MatchResultHeader';
import { KeyMoments, MatchTimeline } from './MatchEvents';
import { MatchStatComparison } from './MatchStatComparison';
import { PlayerOfMatch } from './PlayerOfMatch';
import { LineupView } from './LineupView';
import { CompetitionImpact } from './CompetitionImpact';
import { OtherResults } from './OtherResults';
import { MatchMomentum, MatchSnapshot, ShotMap, TerritoryCard, TopPerformers, XGFlow } from './MatchInsights';
import { DuelOfTheMatch } from './DuelOfTheMatch';
import { PressConference } from './PressConference';

type PostTab = 'overview' | 'stats' | 'lineups' | 'events' | 'table';

const EMPTY_RESULTS: Fixture[] = [];
const EMPTY_EVENTS: Fixture['events'] = [];
const EMPTY_FACTS: NonNullable<Fixture['story_facts']> = [];
const EMPTY_PLAYERS: Fixture['home_xi'] = [];

export interface PostMatchBroadcastProps {
  fixture?: Fixture | null;
  homeClub: Club | null;
  awayClub: Club | null;
  otherResults?: Fixture[];
  onContinueWorld?: () => void;
  onBackToMatches?: () => void;
  onViewCompetition?: () => void;
  onViewClub?: (club: Club) => void;
  onOpenPlayer: (id: string) => void;
  onOpenFixture?: (id: string) => void;
  onRetry?: () => void;
  loadError?: boolean;
  /** Inside a fillViewport Modal the pane scrolls itself; inline the page scrolls. */
  bounded?: boolean;
}

const TABS: Array<{ id: PostTab; label: string; icon: React.ElementType }> = [
  { id: 'overview', label: 'Overview', icon: Activity },
  { id: 'stats', label: 'Stats', icon: BarChart3 },
  { id: 'lineups', label: 'Lineups', icon: Users },
  { id: 'events', label: 'Events', icon: List },
  { id: 'table', label: 'Table', icon: Table2 },
];

function handlePostTabKey(event: React.KeyboardEvent<HTMLDivElement>) {
  const tabs = Array.from(event.currentTarget.querySelectorAll<HTMLButtonElement>('[role="tab"]'));
  const index = tabs.indexOf(event.target as HTMLButtonElement);
  if (index < 0 || !['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return;
  event.preventDefault();
  const next = event.key === 'Home' ? 0 : event.key === 'End' ? tabs.length - 1 : (index + (event.key === 'ArrowRight' ? 1 : -1) + tabs.length) % tabs.length;
  tabs[next].focus();
  tabs[next].click();
}

export const PostMatchBroadcast: React.FC<PostMatchBroadcastProps> = ({
  fixture, homeClub, awayClub, otherResults = EMPTY_RESULTS, onContinueWorld, onBackToMatches, onViewCompetition, onViewClub, onOpenPlayer, onOpenFixture, onRetry, loadError,
  bounded = false,
}) => {
  const [tab, setTab] = useState<PostTab>('overview');
  const tabsId = useId().replace(/:/g, '');
  const home = fixture?.home ?? homeClub;
  const away = fixture?.away ?? awayClub;
  const homeScore = fixture?.home_goals ?? 0;
  const awayScore = fixture?.away_goals ?? 0;
  const events = fixture?.events ?? EMPTY_EVENTS;
  const motm = fixture?.motm;
  const facts = fixture?.story_facts ?? EMPTY_FACTS;
  const summary = useMemo(
    () => composeMatchStory(home, away, homeScore, awayScore, facts, events),
    [home, away, homeScore, awayScore, facts, events],
  );
  const homeStats = useMemo(() => normalizeTeamStats(fixture?.stats?.home), [fixture?.stats?.home]);
  const awayStats = useMemo(() => normalizeTeamStats(fixture?.stats?.away), [fixture?.stats?.away]);
  const impact = fixture?.table_impact;
  const tableTabLabel = impact?.applicable ? 'Table' : 'Competition';
  const homeRows = useMemo(() => [...(fixture?.home_xi ?? EMPTY_PLAYERS), ...(fixture?.home_bench ?? EMPTY_PLAYERS).filter((p) => p.played)], [fixture?.home_xi, fixture?.home_bench]);
  const awayRows = useMemo(() => [...(fixture?.away_xi ?? EMPTY_PLAYERS), ...(fixture?.away_bench ?? EMPTY_PLAYERS).filter((p) => p.played)], [fixture?.away_xi, fixture?.away_bench]);
  const allRows = useMemo(() => [...homeRows, ...awayRows], [homeRows, awayRows]);
  const related = useMemo(() => otherResultsFor(fixture, otherResults), [fixture, otherResults]);

  useEffect(() => { setTab('overview'); }, [fixture?.id]);

  if (loadError && !fixture) {
    return (
      <section data-post-match="true" className="console-hero p-10 text-center">
        <p className="match-section-title">Match report unavailable</p>
        <p className="mt-3 text-[13px] text-sage">The finished fixture could not be loaded.</p>
        <div className="mt-6 flex flex-wrap justify-center gap-2">
          {onRetry && <button type="button" className="gold-btn" onClick={onRetry}>Retry</button>}
          {onBackToMatches && <button type="button" className="px-4 py-2 text-[12px] font-semibold border border-line bg-cardLight text-bone hover:bg-cardHover" onClick={onBackToMatches}>Back to Matches</button>}
        </div>
      </section>
    );
  }

  if (!fixture) {
    return (
      <section data-post-match="true" className="console-hero p-10 text-center">
        <p className="match-section-title">Match report unavailable</p>
        <p className="mt-3 text-[13px] text-sage">This fixture does not have a stored report.</p>
        <div className="mt-6 flex flex-wrap justify-center gap-2">
          {onRetry && <button type="button" className="gold-btn" onClick={onRetry}>Retry</button>}
          {onBackToMatches && <button type="button" className="px-4 py-2 text-[12px] font-semibold border border-line bg-cardLight text-bone hover:bg-cardHover" onClick={onBackToMatches}>Back to Matches</button>}
        </div>
      </section>
    );
  }

  if (!fixture.events?.length && fixture.report_summary) {
    const summary = fixture.report_summary;
    return (
      <section data-post-match="true" className="console-hero flex flex-col border border-brass/25 bg-[#0a1811] rounded-lg p-6 sm:p-8">
        <MatchResultHeader fixture={fixture} home={home} away={away} homeScore={summary.home_goals} awayScore={summary.away_goals} events={events} />
        <div className="mt-4 rounded-md border border-line bg-black/20 p-4" data-archived-summary="true">
          <p className="text-[10px] font-bold uppercase tracking-[0.14em] text-brass/80">Archived match summary</p>
          <p className="mt-1 text-[11px] text-sage">Full match detail is retained for recent fixtures; this older result keeps the archival record.</p>
          <div className="mt-3 grid grid-cols-2 gap-3 text-[12px] sm:grid-cols-4">
            <div><p className="text-sage">Half-time</p><p className="font-mono text-bone">{summary.ht_home}–{summary.ht_away}</p></div>
            <div><p className="text-sage">Possession</p><p className="font-mono text-bone">{summary.possession_home}%–{summary.possession_away}%</p></div>
            <div><p className="text-sage">Shots</p><p className="font-mono text-bone">{summary.shots_home}–{summary.shots_away}</p></div>
            <div><p className="text-sage">xG</p><p className="font-mono text-bone">{summary.xg_home.toFixed(2)}–{summary.xg_away.toFixed(2)}</p></div>
          </div>
          {(summary.scorers?.length ?? 0) > 0 && (
            <div className="mt-3 border-t border-white/[0.08] pt-3">
              <p className="text-[10px] font-bold uppercase tracking-[0.12em] text-sage">Goals</p>
              <ul className="mt-1 space-y-0.5 text-[12px] text-bone">
                {summary.scorers?.map((scorer, i) => (
                  <li key={`${scorer.player_id}-${i}`}>
                    <button type="button" className="hover:text-brass" onClick={() => onOpenPlayer(scorer.player_id)}>
                      {scorer.player_name} {scorer.minute}&prime;{scorer.type === 'own_goal' ? ' (og)' : ''}
                    </button>
                  </li>
                ))}
              </ul>
            </div>
          )}
          {summary.motm && (
            <p className="mt-3 border-t border-white/[0.08] pt-3 text-[12px] text-sage">
              Player of the match:{' '}
              <button type="button" className="font-semibold text-bone hover:text-brass" onClick={() => onOpenPlayer(summary.motm!.player_id)}>
                {summary.motm.full_name}
              </button>
              {summary.motm.rating ? ` (${summary.motm.rating})` : ''}
            </p>
          )}
        </div>
        {onBackToMatches && (
          <div className="mt-4 flex flex-wrap gap-2">
            <button type="button" onClick={onBackToMatches} className="px-4 py-2 text-[12px] font-semibold border border-line bg-cardLight text-bone hover:bg-cardHover">Back to Matches</button>
          </div>
        )}
      </section>
    );
  }

  return (
    <section
      data-post-match="true"
      className={cx(
        'flex flex-col border border-brass/25 bg-[#0a1811] shadow-2xl shadow-black/30 rounded-lg',
        bounded && 'min-h-0 flex-1 overflow-hidden rounded-none shadow-2xl',
      )}
    >
      <MatchResultHeader fixture={fixture} home={home} away={away} homeScore={homeScore} awayScore={awayScore} events={events} />

      <div className="section-tabs shrink-0 px-2 sm:px-4" role="tablist" aria-label="Post-match report" onKeyDown={handlePostTabKey}>
        {TABS.map(({ id, label, icon: Icon }) => (
          <button key={id} id={`${tabsId}-tab-${id}`} type="button" role="tab" aria-selected={tab === id} aria-controls={`${tabsId}-panel`} tabIndex={tab === id ? 0 : -1} onClick={() => setTab(id)} className={cx('section-tab flex items-center gap-1.5', tab === id && 'section-tab-active')}>
            <Icon size={13} />{id === 'table' ? tableTabLabel : label}
          </button>
        ))}
      </div>

      <div
        id={`${tabsId}-panel`}
        role="tabpanel"
        aria-labelledby={`${tabsId}-tab-${tab}`}
        tabIndex={0}
        className={cx(
          'p-4 focus-visible:outline focus-visible:outline-2 focus-visible:outline-brass sm:p-5',
          bounded && 'min-h-0 flex-1 overflow-y-auto overscroll-contain',
        )}
      >
        {tab === 'overview' && (
          <div className="grid grid-cols-1 gap-4 xl:grid-cols-12">
            <div className="space-y-4 xl:col-span-8">
              {fixture && <MatchSnapshot fixture={fixture} home={homeStats} away={awayStats} />}
              <section className="console-card p-4 sm:p-5">
                <div className="flex items-center justify-between gap-3"><p className="match-section-title">Match story</p><span className="text-[10px] uppercase tracking-[0.12em] text-sage">Report</span></div>
                <p className="mt-3 text-[15px] leading-relaxed text-bone">{summary}</p>
              </section>
              {fixture && <MatchMomentum fixture={fixture} events={events} />}
              {fixture && <PressConference fixture={fixture} home={home} away={away} />}
              <section className="console-card p-4 sm:p-5">
                <p className="match-section-title">Key moments</p>
                <KeyMoments events={events} fixture={fixture} home={home} away={away} onOpenPlayer={onOpenPlayer} />
              </section>
            </div>
            <aside className="space-y-4 xl:col-span-4">
              <section className="console-card p-4">
                <p className="match-section-title">Player of the match</p>
                <PlayerOfMatch motm={motm} home={home} away={away} onOpenPlayer={onOpenPlayer} />
              </section>
              <DuelOfTheMatch fixture={fixture} homeRows={homeRows} awayRows={awayRows} onOpenPlayer={onOpenPlayer} />
              <section className="console-card p-4">
                <p className="match-section-title">At a glance</p>
                <div className="mt-3"><MatchStatComparison homeName={home?.short_name || 'Home'} awayName={away?.short_name || 'Away'} home={homeStats} away={awayStats} compact /></div>
              </section>
              <section className="console-card p-4">
                <p className="match-section-title">Other results</p>
                <OtherResults results={related} onOpenFixture={onOpenFixture} />
              </section>
              <section className="console-card p-4">
                <p className="match-section-title">{impact?.applicable ? 'Table impact' : 'Competition impact'}</p>
                <div className="mt-3"><CompetitionImpact fixture={fixture} home={home} away={away} /></div>
              </section>
            </aside>
          </div>
        )}

        {tab === 'stats' && (
          <div className="grid grid-cols-1 gap-4 xl:grid-cols-12">
            <section className="console-card p-4 sm:p-5 xl:col-span-5"><MatchStatComparison homeName={home?.short_name || 'Home'} awayName={away?.short_name || 'Away'} home={homeStats} away={awayStats} /></section>
            <div className="space-y-4 xl:col-span-7">
              {fixture && <ShotMap fixture={fixture} />}
              {fixture && <XGFlow fixture={fixture} />}
              {fixture && <MatchMomentum fixture={fixture} events={events} />}
            </div>
            {fixture && <div className="xl:col-span-7"><TopPerformers fixture={fixture} rows={allRows} onOpenPlayer={onOpenPlayer} /></div>}
            {fixture && <div className="xl:col-span-5"><TerritoryCard fixture={fixture} /></div>}
          </div>
        )}

        {tab === 'lineups' && (
          <LineupView
            home={home}
            away={away}
            homeRows={homeRows}
            awayRows={awayRows}
            homeFormation={fixture?.home_formation}
            awayFormation={fixture?.away_formation}
            motmId={motm?.player_id}
            onOpenPlayer={onOpenPlayer}
          />
        )}

        {tab === 'events' && (
          <div className="grid grid-cols-1 gap-4 xl:grid-cols-12">
            <section className="console-card p-4 xl:col-span-8"><MatchTimeline events={events} fixture={fixture} home={home} away={away} onOpenPlayer={onOpenPlayer} /></section>
            {fixture && <div className="xl:col-span-4"><TopPerformers fixture={fixture} rows={allRows} onOpenPlayer={onOpenPlayer} /></div>}
          </div>
        )}

        {tab === 'table' && (
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            <section className="console-card p-4 sm:p-5"><p className="match-section-title">{impact?.applicable ? 'Updated table' : 'Competition outcome'}</p><div className="mt-3"><CompetitionImpact fixture={fixture} home={home} away={away} /></div></section>
            <section className="console-card p-4 sm:p-5"><p className="match-section-title">Other results</p><OtherResults results={related} onOpenFixture={onOpenFixture} /></section>
          </div>
        )}
      </div>

      <div className="shrink-0 border-t border-white/10 bg-black/15 p-3 sm:px-5">
        <div className="flex flex-wrap items-center gap-2">
          {onContinueWorld && (
            <button type="button" onClick={() => { soundManager.playWhistle(); onContinueWorld(); }} className="gold-btn">
              <span>Continue World</span><ArrowRight size={15} />
            </button>
          )}
          {onBackToMatches && (
            <button type="button" onClick={onBackToMatches} className="px-4 py-2 text-[12px] font-semibold border border-line bg-cardLight text-bone hover:bg-cardHover">
              Back to Matches
            </button>
          )}
          {onViewCompetition && (
            <button type="button" onClick={onViewCompetition} className="px-4 py-2 text-[12px] font-semibold border border-line bg-cardLight text-bone hover:bg-cardHover">
              View Competition
            </button>
          )}
          {onViewClub && home && (
            <button type="button" onClick={() => onViewClub(home)} className="px-4 py-2 text-[12px] font-semibold border border-line bg-cardLight text-bone hover:bg-cardHover">
              View {home.short_name}
            </button>
          )}
          {onViewClub && away && (
            <button type="button" onClick={() => onViewClub(away)} className="px-4 py-2 text-[12px] font-semibold border border-line bg-cardLight text-bone hover:bg-cardHover">
              View {away.short_name}
            </button>
          )}
        </div>
      </div>
    </section>
  );
};

export const PostMatchModal: React.FC<{ fixture: Fixture | null; onClose: () => void; otherResults?: Fixture[] }> = ({ fixture, onClose, otherResults }) => {
  const { openPlayer } = usePlayerSheet();
  const [full, setFull] = useState<Fixture | null>(fixture);
  const [related, setRelated] = useState<Fixture[]>(otherResults ?? []);
  const [loadError, setLoadError] = useState(false);

  useEffect(() => {
    if (!fixture) {
      setFull(null);
      setLoadError(false);
      setRelated(otherResults ?? []);
      return;
    }
    setFull(fixture);
    setRelated(otherResults ?? []);
    if (fixture.status !== 'finished') return;
    let cancelled = false;
    fetchFixture(fixture.id)
      .then((row) => {
        if (cancelled) return;
        if (row) setFull(row);
        else setLoadError(true);
      })
      .catch(() => { if (!cancelled) setLoadError(true); });
    fetchFixtureSummaries(fixture.matchweek)
      .then((res) => {
        if (cancelled) return;
        setRelated(res.fixtures.filter((row) => row.id !== fixture.id && row.status === 'finished'));
      })
      .catch(() => undefined);
    return () => { cancelled = true; };
  }, [fixture?.id]);

  const retry = () => {
    const id = full?.id ?? fixture?.id;
    if (!id) return;
    setLoadError(false);
    fetchFixture(id).then((row) => { if (row) setFull(row); else setLoadError(true); }).catch(() => setLoadError(true));
  };

  const openRelated = (id: string) => {
    setLoadError(false);
    fetchFixture(id)
      .then((row) => { if (row) setFull(row); else setLoadError(true); })
      .catch(() => setLoadError(true));
  };

  return (
    <Modal open={!!fixture} onClose={onClose} maxWidth="max-w-7xl" fillViewport>
      {fixture && (
        <PostMatchBroadcast
          fixture={full}
          homeClub={full?.home ?? fixture.home}
          awayClub={full?.away ?? fixture.away}
          otherResults={related}
          onBackToMatches={onClose}
          onOpenPlayer={openPlayer}
          onOpenFixture={openRelated}
          onRetry={retry}
          loadError={loadError}
          bounded
        />
      )}
    </Modal>
  );
};
