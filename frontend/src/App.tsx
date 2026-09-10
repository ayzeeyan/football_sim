import React, { useCallback, useEffect, useState } from 'react';
import type { BatchSimResult, Club, Fixture, SeasonAwards } from './types';
import { fetchCalendar, fetchFavourite, fetchInbox, fetchSeasonAwards, setFavourite, simulateMonth, simulateRemaining, simulateSeason, simulateWeek, type CalendarState } from './services/api';
import { slugFromTab, tabFromSlug, type TabId } from './lib/constants';
import { stripEmojis } from './lib/format';
import { useClubs, useMatchEngine } from './hooks/useMatch';
import { useToast } from './hooks/useToast';
import { TopBar } from './components/layout/TopBar';
import { AwardsModal } from './components/layout/AwardsModal';
import { AwardsCeremonyModal } from './components/AwardsCeremonyModal';
import { ToastHost } from './components/layout/ToastHost';
import { MatchdayTab } from './components/MatchdayTab';
import { SquadTab } from './components/SquadTab';
import { WonderkidLabTab } from './components/WonderkidLabTab';
import { StandingsTab } from './components/StandingsTab';
import { TransfersTab } from './components/TransfersTab';
import { HistoryTab } from './components/HistoryTab';
import { ClubPickerModal } from './components/ClubPickerModal';
import { NewCareerModal } from './components/NewCareerModal';
import { InboxTab } from './components/InboxTab';
import { PlayerSheetProvider } from './components/PlayerSheet';
import { MatchweekDigestModal } from './components/MatchweekDigestModal';
import { soundManager } from './audio/webAudio';
import { matchWs } from './services/matchSocket';

export const App: React.FC = () => {
  // Career home screen: the league table.
  const [activeTab, setActiveTab] = useState<TabId>(() =>
    typeof window === 'undefined' ? 2 : tabFromSlug(window.location.hash.replace(/^#/, '')),
  );
  const { clubs, homeClub, awayClub, setHome, setAway, swap, applyClubs, reloadClubs } = useClubs();
  const { matchData, status, kickoff, pause, setSpeed, reset, seek70, seekChance } = useMatchEngine();
  const { toast, showToast } = useToast();

  const [modalOpen, setModalOpen] = useState(false);
  const [modalTarget, setModalTarget] = useState<'home' | 'away'>('home');
  const [awardsOpen, setAwardsOpen] = useState(false);
  const [ceremonyOpen, setCeremonyOpen] = useState(false);
  const [awardsData, setAwardsData] = useState<SeasonAwards | null>(null);
  const [muted, setMuted] = useState(false);
  const [newCareerOpen, setNewCareerOpen] = useState(false);
  const [careerKey, setCareerKey] = useState(0);
  const [inboxUnread, setInboxUnread] = useState(0);
  const [calendar, setCalendar] = useState<CalendarState | null>(null);
  const [digestData, setDigestData] = useState<BatchSimResult | null>(null);
  const [digestOpen, setDigestOpen] = useState(false);
  const [simulating, setSimulating] = useState(false);

  const handleSelectClub = useCallback(
    (selected: Club) => {
      if (modalTarget === 'home') setHome(selected);
      else setAway(selected);
    },
    [modalTarget, setHome, setAway],
  );

  const handleOpenAwards = useCallback(() => {
    soundManager.playClick();
    fetchSeasonAwards().then((res) => {
      setAwardsData(res);
      setAwardsOpen(true);
    });
  }, []);

  const handleMacroSim = useCallback(async (mode: 'week' | 'month' | 'season') => {
    if (simulating) return;
    setSimulating(true);
    soundManager.playClick();
    showToast(`Simulating ${mode}…`);
    try {
      const action = mode === 'week' ? simulateWeek : mode === 'month' ? simulateMonth : simulateSeason;
      const result = await action();
      if (result.status !== 'success') {
        showToast(result.message || 'Simulation failed.');
        return;
      }
      setDigestData(result);
      setCareerKey((k) => k + 1);
      await Promise.all([
        reloadClubs(),
        fetchCalendar().then(setCalendar),
        fetchInbox(1).then((feed) => setInboxUnread(feed.unread)),
      ]);
      if (mode === 'season' && result.awards_ready) {
        setDigestOpen(false);
        setCeremonyOpen(true);
        showToast(result.champion ? `${result.champion} are champions. Awards ceremony.` : 'Season complete. Awards ceremony.');
      } else {
        setDigestOpen(true);
        showToast(result.message || `Advanced ${result.weeks_advanced} week${result.weeks_advanced === 1 ? '' : 's'}.`);
      }
    } catch {
      showToast('Simulation failed.');
    } finally {
      setSimulating(false);
    }
  }, [reloadClubs, showToast, simulating]);

  const watchClub = useCallback(
    (c: Club) => {
      setHome(c);
      setActiveTab(0);
      showToast(`Now showing ${c.club_name}.`);
    },
    [setHome, showToast],
  );

  const watchFixture = useCallback(
    (f: Fixture) => {
      applyClubs(f.home, f.away);
      setActiveTab(0);
      showToast(`Now showing ${f.home.short_name} against ${f.away.short_name}.`);
    },
    [applyClubs, showToast],
  );

  /** Full-time follow-on: the broadcast moves itself to a fresh random fixture and kicks off. */
  const handleNextFixture = useCallback(() => {
    if (clubs.length < 2) return;
    const home = clubs[Math.floor(Math.random() * clubs.length)];
    let away = clubs[Math.floor(Math.random() * clubs.length)];
    if (away.club_id === home.club_id) {
      away = clubs[(clubs.indexOf(home) + 1) % clubs.length];
    }
    setHome(home);
    setAway(away);
    showToast(`Next up: ${home.short_name} against ${away.short_name}.`);
    matchWs.sendCommand('kickoff');
  }, [clubs, setHome, setAway, showToast]);

  const viewSquadOf = useCallback(
    (c: Club) => {
      setHome(c);
      setActiveTab(3);
    },
    [setHome],
  );

  const handleTab = useCallback((id: TabId) => {
    setActiveTab(id);
  }, []);

  useEffect(() => {
    const slug = slugFromTab(activeTab);
    if (window.location.hash.replace(/^#/, '') !== slug) {
      window.history.replaceState(null, '', `#${slug}`);
    }
  }, [activeTab]);

  useEffect(() => {
    const onHash = () => setActiveTab(tabFromSlug(window.location.hash.replace(/^#/, '')));
    window.addEventListener('hashchange', onHash);
    return () => window.removeEventListener('hashchange', onHash);
  }, []);

  useEffect(() => {
    fetchInbox(1).then((feed) => setInboxUnread(feed.unread)).catch(() => undefined);
  }, [careerKey]);

  useEffect(() => {
    fetchCalendar().then(setCalendar).catch(() => undefined);
  }, [careerKey]);

  // Favourite watch club: first opened live match pins it when unset.
  // Server also pins home on set_clubs; this mirrors to localStorage-backed API.
  useEffect(() => {
    if (!homeClub) return;
    let cancelled = false;
    fetchFavourite()
      .then((f) => {
        if (cancelled) return;
        if (!f.favourite_club_id && homeClub) {
          void setFavourite(homeClub.club_id).catch(() => undefined);
        }
      })
      .catch(() => undefined);
    return () => { cancelled = true; };
  }, [homeClub?.club_id]);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement;
      if (
        target &&
        (target.tagName === 'INPUT' ||
          target.tagName === 'TEXTAREA' ||
          target.tagName === 'SELECT' ||
          target.isContentEditable)
      ) {
        return;
      }

      if (e.key === 'Escape') {
        setModalOpen(false);
        setAwardsOpen(false);
        setCeremonyOpen(false);
        setNewCareerOpen(false);
        setDigestOpen(false);
        return;
      }

      if (e.repeat) return;
      if (!simulating && !e.shiftKey && e.key.toLowerCase() === 'w') {
        e.preventDefault();
        void handleMacroSim('week');
        return;
      }
      if (!simulating && !e.shiftKey && e.key.toLowerCase() === 'm') {
        e.preventDefault();
        void handleMacroSim('month');
        return;
      }
      if (!simulating && e.shiftKey && e.key.toLowerCase() === 's') {
        e.preventDefault();
        void handleMacroSim('season');
        return;
      }

      if (e.key === ' ' || e.code === 'Space') {
        // Matchday owns playback shortcuts; never sim the slate from its dugout.
        if (activeTab === 0 || matchData?.state === 'HALF_TIME') return;
        e.preventDefault();
        soundManager.playClick();
        showToast('Simulating matchweek…');
        simulateRemaining()
          .then((res) => {
            if (res.status === 'success') {
              showToast(`Simulated ${res.played} fixtures.`);
              setCareerKey((k) => k + 1);
              void reloadClubs();
            } else {
              showToast(res.status || 'Simulation complete.');
            }
          })
          .catch(() => {
            showToast('Simulation failed.');
          });
        return;
      }

      if (activeTab === 0) return;
      if (e.key === '1') {
        setSpeed(1);
        showToast('Speed: 1x');
      } else if (e.key === '2') {
        setSpeed(2);
        showToast('Speed: 2x');
      } else if (e.key === '3') {
        setSpeed(5);
        showToast('Speed: 5x');
      } else if (e.key === '4') {
        setSpeed(999);
        showToast('Speed: Instant');
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [showToast, reloadClubs, setSpeed, activeTab, matchData?.state, handleMacroSim]);

  return (
    <PlayerSheetProvider>
    <div className="min-h-screen bg-ink text-bone flex flex-col font-sans">
      <a href="#main" className="skip-link">
        Skip to content
      </a>
      {status !== 'CONNECTED' && (
        <div
          className="bg-ember/15 border-b border-ember/40 text-center py-2 px-4 text-[13px] font-semibold text-[#D89A84]"
          role="status"
          aria-live="polite"
        >
          {status === 'CONNECTING' || status === 'RECONNECTING'
            ? 'Connecting to the match engine… live matches pause until the link returns.'
            : 'Match engine offline. Tables still load; live kickoff needs the engine.'}
        </div>
      )}

      <TopBar
        activeTab={activeTab}
        onTab={handleTab}
        wsStatus={status}
        muted={muted}
        onToggleMute={() => {
          const isMut = soundManager.toggleMute();
          setMuted(isMut);
          showToast(isMut ? 'Sound off.' : 'Sound on.');
        }}
        onOpenAwards={handleOpenAwards}
        onNewCareer={() => {
          soundManager.playClick();
          setNewCareerOpen(true);
        }}
        onSimWeek={() => void handleMacroSim('week')}
        onSimMonth={() => void handleMacroSim('month')}
        onSimSeason={() => void handleMacroSim('season')}
        simulating={simulating}
        seasonName={calendar?.season_name}
        calendarLabel={calendar ? `MW ${Math.min(calendar.current_matchweek, calendar.max_matchweeks)}/${calendar.max_matchweeks} · ${calendar.month}${calendar.year ? ` ${calendar.year}` : ''}` : undefined}
        inboxUnread={inboxUnread}
      />

      <main id="main" className="flex-1 page-shell py-5 sm:py-6 lg:py-8 scroll-mt-28">
        {activeTab === 0 && (
          <MatchdayTab
            homeClub={homeClub}
            awayClub={awayClub}
            matchData={matchData}
            onOpenClubModal={(target) => {
              setModalTarget(target);
              setModalOpen(true);
            }}
            onSwapTeams={swap}
            onKickoff={kickoff}
            onPause={pause}
            onSetSpeed={setSpeed}
            onReset={reset}
            onNextFixture={handleNextFixture}
            onSeek70={seek70}
            onSeekChance={seekChance}
            onJumpToFixture={watchFixture}
            onWeekAdvanced={() => {
              setCareerKey((k) => k + 1);
              fetchInbox(1).then((feed) => setInboxUnread(feed.unread)).catch(() => undefined);
            }}
          />
        )}
        {activeTab === 1 && <WonderkidLabTab key={`lab-${careerKey}`} onShowToast={(m) => showToast(stripEmojis(m))} />}
        {activeTab === 2 && (
          <StandingsTab
            key={`league-${careerKey}`}
            onWatchFixture={watchFixture}
            onViewSquad={viewSquadOf}
            onShowToast={(m) => showToast(stripEmojis(m))}
            onOpenCeremony={() => setCeremonyOpen(true)}
            onSeasonTick={() => {
              fetchInbox(1).then((feed) => setInboxUnread(feed.unread)).catch(() => undefined);
            }}
          />
        )}
        {activeTab === 6 && (
          <InboxTab key={`inbox-${careerKey}`} careerKey={careerKey} onUnread={setInboxUnread} />
        )}
        {activeTab === 5 && <HistoryTab key={`history-${careerKey}`} />}
        {activeTab === 3 && <SquadTab key={`squads-${careerKey}`} clubs={clubs} onWatchClub={watchClub} />}
        {activeTab === 4 && <TransfersTab key={`transfers-${careerKey}`} onShowToast={(m) => showToast(stripEmojis(m))} />}
      </main>

      <footer className="border-t border-line mt-auto">
        <div className="page-shell py-4 flex flex-wrap items-center justify-between gap-3 text-[13px] text-sage">
          <div className="flex items-center gap-3">
            <p>European Super League, {calendar?.season_name?.replace('-', '–') || '2026–27'}</p>
            <span className="text-line">•</span>
            <p>12 clubs · 44-week Super League</p>
          </div>

          <div
            className="flex items-center gap-2 text-[11px] font-mono text-sage bg-cardLight/70 border border-line px-3 py-1.5 rounded-full shadow-sm select-none"
            title="Global Spectator Shortcuts"
          >
            <span className="flex items-center gap-1.5">
              <kbd className="px-1.5 py-0.5 rounded bg-ink/70 border border-line text-bone font-semibold text-[10px]">W</kbd> Week
            </span>
            <span className="text-line">•</span>
            <span className="flex items-center gap-1.5">
              <kbd className="px-1.5 py-0.5 rounded bg-ink/70 border border-line text-bone font-semibold text-[10px]">M</kbd> Month
            </span>
            <span className="text-line">•</span>
            <span className="flex items-center gap-1.5">
              <kbd className="px-1.5 py-0.5 rounded bg-ink/70 border border-line text-bone font-semibold text-[10px]">⇧S</kbd> Season
            </span>
            <span className="text-line">•</span>
            <span className="flex items-center gap-1.5">
              <kbd className="px-1.5 py-0.5 rounded bg-ink/70 border border-line text-bone font-semibold text-[10px]">Esc</kbd> Close
            </span>
          </div>
        </div>
      </footer>

      <ClubPickerModal
        isOpen={modalOpen}
        target={modalTarget}
        clubs={clubs}
        onSelectClub={handleSelectClub}
        onClose={() => setModalOpen(false)}
      />

      <AwardsModal
        open={awardsOpen}
        data={awardsData}
        onClose={() => setAwardsOpen(false)}
        onWatchCeremony={() => {
          setAwardsOpen(false);
          setCeremonyOpen(true);
        }}
      />
      <AwardsCeremonyModal open={ceremonyOpen} onClose={() => setCeremonyOpen(false)} />
      <MatchweekDigestModal open={digestOpen} data={digestData} onClose={() => setDigestOpen(false)} />
      <NewCareerModal
        open={newCareerOpen}
        onClose={() => setNewCareerOpen(false)}
        onStarted={(msg) => {
          showToast(stripEmojis(msg));
          setCareerKey((n) => n + 1);
          void reloadClubs();
          setActiveTab(2);
        }}
      />
      <ToastHost message={toast} />
    </div>
    </PlayerSheetProvider>
  );
};
