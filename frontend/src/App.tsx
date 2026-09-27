import React, { Suspense, lazy, useCallback, useEffect, useRef, useState } from 'react';
import type { BatchSimResult, Club, Fixture, SeasonAwards } from './types';
import { fetchCalendar, fetchFixtureSummaries, fetchInbox, fetchSeasonAwards, invalidateApiCache, simulateContinue, simulateMonth, simulateSeason, simulateWeek, type CalendarState } from './services/api';
import { slugFromTab, tabFromSlug, type TabId } from './lib/constants';
import { stripEmojis, setNumberFormat } from './lib/format';
import { filterUnreadCount, loadSettings, saveSettings, settingsRootClasses, type ViewerSettings } from './lib/settings';
import { useClubs } from './hooks/useClubs';
import { useToast } from './hooks/useToast';
import { TopBar } from './components/layout/TopBar';
import { AwardsModal } from './components/layout/AwardsModal';
import { AwardsCeremonyModal } from './components/career/AwardsCeremonyModal';
import { ToastHost } from './components/layout/ToastHost';
import { NewCareerModal } from './components/career/NewCareerModal';
import { SaveSlotsModal } from './components/career/SaveSlotsModal';
import { PlayerSheetProvider } from './components/clubs/PlayerSheet';
import { MatchweekDigestModal } from './components/postmatch/MatchweekDigestModal';
import { soundManager } from './audio/webAudio';
import { SettingsPanel } from './components/layout/SettingsPanel';

const HomeDashboardTab = lazy(() => import('./components/competitions/HomeDashboardTab').then((m) => ({ default: m.HomeDashboardTab })));
const SimulationCentreTab = lazy(() => import('./components/matches/SimulationCentreTab').then((m) => ({ default: m.SimulationCentreTab })));
const WonderkidLabTab = lazy(() => import('./components/wonderkids/WonderkidLabTab').then((m) => ({ default: m.WonderkidLabTab })));
const StandingsTab = lazy(() => import('./components/competitions/StandingsTab').then((m) => ({ default: m.StandingsTab })));
const CompetitionHubTab = lazy(() => import('./components/competitions/CompetitionHubTab').then((m) => ({ default: m.CompetitionHubTab })));
const InboxTab = lazy(() => import('./components/career/InboxTab').then((m) => ({ default: m.InboxTab })));
const HistoryTab = lazy(() => import('./components/competitions/HistoryTab').then((m) => ({ default: m.HistoryTab })));
const SquadTab = lazy(() => import('./components/clubs/SquadTab').then((m) => ({ default: m.SquadTab })));
const TransfersTab = lazy(() => import('./components/transfers/TransfersTab').then((m) => ({ default: m.TransfersTab })));
const PlayersTab = lazy(() => import('./components/clubs/PlayersTab').then((m) => ({ default: m.PlayersTab })));
const StatsCentreTab = lazy(() => import('./components/stats').then((m) => ({ default: m.StatsCentreTab })));

const ScreenLoading: React.FC = () => (
  <div className="p-12 text-center" role="status" aria-live="polite">
    <div className="inline-block animate-spin rounded-full h-8 w-8 border-2 border-brass border-t-transparent" />
    <p className="mt-3 text-[13px] font-mono text-sage">Loading screen…</p>
  </div>
);

export const App: React.FC = () => {
  // Career home screen: the league table.
  const [settings, setSettings] = useState<ViewerSettings>(loadSettings);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [activeTab, setActiveTab] = useState<TabId>(() => {
    if (typeof window === 'undefined') return 8;
    const slug = window.location.hash.replace(/^#/, '');
    return tabFromSlug(slug || loadSettings().defaultTab);
  });
  const { clubs, reloadClubs } = useClubs();
  const [selectedFixtureId, setSelectedFixtureId] = useState<string | null>(null);
  const { toast, showToast } = useToast();

  // Persist viewer settings and apply their side effects. localStorage keeps
  // these out of the save format: they belong to the viewer, not the world.
  useEffect(() => {
    saveSettings(settings);
    setNumberFormat(settings.numberFormat);
    soundManager.muted = settings.soundMuted;
  }, [settings]);

  const [selectedClubId, setSelectedClubId] = useState<string | undefined>();
  const [awardsOpen, setAwardsOpen] = useState(false);
  const [ceremonyOpen, setCeremonyOpen] = useState(false);
  const [awardsData, setAwardsData] = useState<SeasonAwards | null>(null);
  const muted = settings.soundMuted;
  const [newCareerOpen, setNewCareerOpen] = useState(false);
  const [slotsOpen, setSlotsOpen] = useState(false);
  const [careerKey, setCareerKey] = useState(0);
  const [inboxUnread, setInboxUnread] = useState(0);
  const [calendar, setCalendar] = useState<CalendarState | null>(null);
  const [digestData, setDigestData] = useState<BatchSimResult | null>(null);
  const [digestOpen, setDigestOpen] = useState(false);
  const [simulating, setSimulating] = useState(false);
  const simLockRef = useRef(false);

  const handleOpenAwards = useCallback(() => {
    soundManager.playClick();
    fetchSeasonAwards().then((res) => {
      setAwardsData(res);
      setAwardsOpen(true);
    });
  }, []);

  const handleMacroSim = useCallback(async (mode: 'continue' | 'week' | 'month' | 'season') => {
    if (simLockRef.current) return;
    simLockRef.current = true;
    setSimulating(true);
    soundManager.playClick();
    showToast(mode === 'continue' ? 'Continuing…' : `Simulating ${mode}…`);
    try {
      const action = mode === 'week' ? simulateWeek : mode === 'month' ? simulateMonth : mode === 'continue' ? simulateContinue : simulateSeason;
      const result = await action();
      if (result.status !== 'success') {
        showToast(result.message || 'Simulation failed.');
        return;
      }
      setCareerKey((k) => k + 1);
      invalidateApiCache();
      await Promise.all([
        reloadClubs(),
        fetchCalendar().then(setCalendar),
        fetchInbox(1).then((feed) => setInboxUnread(filterUnreadCount(feed, settings))),
      ]);
      setDigestData(result);
      if ((mode === 'season' || result.stop_reason === 'season_event') && result.awards_ready) {
        setDigestOpen(false);
        setCeremonyOpen(true);
        showToast(result.champion ? `${result.champion} are champions. Awards ceremony.` : 'Season complete. Awards ceremony.');
      } else {
        setDigestOpen(true);
        showToast(result.continue_hint || result.message || `Advanced ${result.weeks_advanced} week${result.weeks_advanced === 1 ? '' : 's'}.`);
      }
    } catch (err: unknown) {
      showToast(err instanceof Error && err.message ? err.message : 'Simulation failed.');
    } finally {
      simLockRef.current = false;
      setSimulating(false);
    }
  }, [reloadClubs, showToast]);

  const watchClub = useCallback(async (c: Club) => {
    const schedule = await fetchFixtureSummaries();
    const fixture = schedule.fixtures.find((row) => row.status === 'scheduled' && (row.home.club_id === c.club_id || row.away.club_id === c.club_id));
    if (!fixture) {
      showToast(`No scheduled fixture is currently available for ${c.short_name}.`);
      return;
    }
    setSelectedFixtureId(fixture.id);
    setActiveTab(0);
    showToast(`Opened ${fixture.home.short_name} against ${fixture.away.short_name} in the Match Centre.`);
  }, [showToast]);

  const openFixture = useCallback(
    (f: Fixture) => {
      setSelectedFixtureId(f.id);
      setActiveTab(0);
      showToast(`Opened ${f.home.short_name} against ${f.away.short_name} in the Match Centre.`);
    },
    [showToast],
  );

  const viewSquadOf = useCallback(
    (c: Club) => {
      setSelectedClubId(c.club_id);
      setActiveTab(3);
    },
    [],
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


  // Auto-advance: when enabled, the world moves one matchweek every N
  // seconds unless a simulation is running or a modal is open.
  const autoAdvanceRef = useRef<() => void>(() => undefined);
  useEffect(() => {
    autoAdvanceRef.current = () => void handleMacroSim('week');
  });
  useEffect(() => {
    if (settings.autoAdvanceSeconds <= 0) return;
    const id = window.setInterval(() => {
      if (
        simLockRef.current ||
        simulating ||
        awardsOpen ||
        ceremonyOpen ||
        digestOpen ||
        newCareerOpen ||
        settingsOpen ||
        slotsOpen ||
        selectedFixtureId
      ) {
        return;
      }
      autoAdvanceRef.current();
    }, settings.autoAdvanceSeconds * 1000);
    return () => window.clearInterval(id);
  }, [settings.autoAdvanceSeconds, settingsOpen, slotsOpen, simulating, awardsOpen, ceremonyOpen, digestOpen, newCareerOpen, selectedFixtureId]);

  useEffect(() => {
    const onHash = () => setActiveTab(tabFromSlug(window.location.hash.replace(/^#/, '')));
    window.addEventListener('hashchange', onHash);
    return () => window.removeEventListener('hashchange', onHash);
  }, []);

  useEffect(() => {
    fetchInbox(1).then((feed) => setInboxUnread(filterUnreadCount(feed, settings))).catch(() => undefined);
  }, [careerKey, settings]);

  useEffect(() => {
    fetchCalendar().then(setCalendar).catch(() => undefined);
  }, [careerKey]);

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
        setAwardsOpen(false);
        setCeremonyOpen(false);
        setNewCareerOpen(false);
        setDigestOpen(false);
        return;
      }

      if (e.repeat) return;
      if (!simulating && !e.shiftKey && e.key.toLowerCase() === 'c') {
        e.preventDefault();
        void handleMacroSim('continue');
        return;
      }
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

    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [showToast, reloadClubs, handleMacroSim, simulating]);

  return (
    <PlayerSheetProvider>
    <div className={`min-h-screen bg-ink text-bone flex flex-col font-sans ${settingsRootClasses(settings)}`}>
      <a href="#main" className="skip-link">
        Skip to content
      </a>
      <TopBar
        activeTab={activeTab}
        onTab={handleTab}
        muted={muted}
        onToggleMute={() => {
          const isMut = !settings.soundMuted;
          setSettings((s) => ({ ...s, soundMuted: isMut }));
          showToast(isMut ? 'Sound off.' : 'Sound on.');
        }}
        onOpenSettings={() => {
          soundManager.playClick();
          setSettingsOpen(true);
        }}
        onOpenAwards={handleOpenAwards}
        onNewCareer={() => {
          soundManager.playClick();
          setNewCareerOpen(true);
        }}
        onOpenSlots={() => {
          soundManager.playClick();
          setSlotsOpen(true);
        }}
        onContinue={() => void handleMacroSim('continue')}
        onSimWeek={() => void handleMacroSim('week')}
        onSimMonth={() => void handleMacroSim('month')}
        onSimSeason={() => void handleMacroSim('season')}
        simulating={simulating}
        seasonName={calendar?.season_name}
        calendarLabel={calendar ? `MW ${Math.min(calendar.current_matchweek, calendar.max_matchweeks)}/${calendar.max_matchweeks} · ${calendar.month}${calendar.year ? ` ${calendar.year}` : ''}` : undefined}
        inboxUnread={inboxUnread}
        onOpenClub={(clubId) => {
          const club = clubs.find((c) => c.club_id === clubId);
          if (club) viewSquadOf(club);
        }}
        onOpenCompetition={() => setActiveTab(7)}
      />

      <main id="main" className="flex-1 min-w-0 page-shell py-4 sm:py-5 lg:ml-[216px] lg:w-[calc(100%-216px)] lg:py-6 scroll-mt-28">
        <Suspense fallback={<ScreenLoading />}>
          {activeTab === 8 && (
            <HomeDashboardTab
              key={`home-${careerKey}`}
              careerKey={careerKey}
              onOpenFixture={(fixture) => openFixture({ ...fixture, home: fixture.home as unknown as Club, away: fixture.away as unknown as Club } as Fixture)}
              onViewSquad={(clubId) => {
                const club = clubs.find((c) => c.club_id === clubId);
                if (club) viewSquadOf(club);
              }}
              onOpenInbox={() => setActiveTab(6)}
              onOpenTransfers={() => setActiveTab(4)}
              onOpenLeague={() => setActiveTab(2)}
              onOpenCompetitions={() => setActiveTab(7)}
            />
          )}
          {activeTab === 0 && (
            <SimulationCentreTab
              careerKey={careerKey}
              selectedFixtureId={selectedFixtureId}
              onSelectFixture={setSelectedFixtureId}
              onShowToast={showToast}
              onWeekAdvanced={() => {
                setCareerKey((k) => k + 1);
                void reloadClubs();
                fetchInbox(1).then((feed) => setInboxUnread(filterUnreadCount(feed, settings))).catch(() => undefined);
              }}
              onViewCompetition={() => setActiveTab(7)}
              onViewClub={(club) => viewSquadOf(club)}
            />
          )}
          {activeTab === 1 && <WonderkidLabTab key={`lab-${careerKey}`} onShowToast={(m) => showToast(stripEmojis(m))} />}
          {activeTab === 2 && (
            <StandingsTab
              key={`league-${careerKey}`}
              onWatchFixture={openFixture}
              onViewSquad={viewSquadOf}
              onShowToast={(m) => showToast(stripEmojis(m))}
              onOpenCeremony={() => setCeremonyOpen(true)}
              onSeasonTick={() => {
                fetchInbox(1).then((feed) => setInboxUnread(filterUnreadCount(feed, settings))).catch(() => undefined);
              }}
            />
          )}
          {activeTab === 7 && (
            <CompetitionHubTab
              key={`competitions-${careerKey}`}
              currentMatchweek={calendar?.current_matchweek}
              onWatchFixture={(fixture) => openFixture({ ...fixture, home: fixture.home as unknown as Club, away: fixture.away as unknown as Club } as Fixture)}
              onViewSquad={(clubId) => {
                const club = clubs.find((c) => c.club_id === clubId);
                if (club) viewSquadOf(club);
              }}
            />
          )}
          {activeTab === 6 && (
            <InboxTab key={`inbox-${careerKey}`} careerKey={careerKey} onUnread={setInboxUnread} />
          )}
          {activeTab === 5 && <HistoryTab key={`history-${careerKey}`} />}
          {activeTab === 10 && <StatsCentreTab key={`stats-${careerKey}`} careerKey={careerKey} />}
          {activeTab === 9 && (
            <PlayersTab
              key={`players-${careerKey}`}
              onViewClub={(clubId) => {
                const club = clubs.find((c) => c.club_id === clubId);
                if (club) viewSquadOf(club);
              }}
            />
          )}
          {activeTab === 3 && (
            <SquadTab
              key={`squads-${careerKey}`}
              clubs={clubs}
              initialClubId={selectedClubId}
              onWatchClub={watchClub}
              onWatchFixture={openFixture}
            />
          )}
          {activeTab === 4 && <TransfersTab key={`transfers-${careerKey}`} onShowToast={(m) => showToast(stripEmojis(m))} />}
        </Suspense>
      </main>

      <footer className="border-t border-line mt-auto min-w-0 lg:ml-[216px] lg:w-[calc(100%-216px)]">
        <div className="page-shell py-2.5 flex flex-wrap items-center justify-between gap-3 text-[11px] text-sage/70">
          <div className="flex items-center gap-3">
            <p>{calendar?.world ? 'Top Five Europe' : 'Legacy twelve-club save'}, {calendar?.season_name?.replace('-', '–') || '2026–27'}</p>
            <span className="text-line">•</span>
            <p>{calendar?.world ? '96 clubs · 38-week shared calendar' : 'Legacy calendar'}</p>
          </div>
          <p className="hidden sm:block">C Continue · W Week · M Month · Shift+S Season</p>
        </div>
      </footer>

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
          setActiveTab(8);
        }}
      />
      <SettingsPanel open={settingsOpen} settings={settings} onChange={setSettings} onClose={() => setSettingsOpen(false)} />
      <SaveSlotsModal open={slotsOpen} onClose={() => setSlotsOpen(false)} onToast={(m) => showToast(stripEmojis(m))} />
      <ToastHost message={toast} />
    </div>
    </PlayerSheetProvider>
  );
};
