import React from 'react';
import {
  Activity, Archive, BarChart3, CalendarDays, CircleDot, History, Home, Inbox, Medal, Plus,
  Settings2, Sparkles, Trophy, Users, UserRound, Volume2, VolumeX, WalletCards, Globe2,
} from 'lucide-react';
import { NAV_GROUPS, TABS, type TabId } from '../../lib/constants';
import { soundManager } from '../../audio/webAudio';
import { cx } from '../../lib/format';
import { GlobalSearch } from './GlobalSearch';
import { usePlayerSheet } from '../clubs/PlayerSheet';

interface TopBarProps {
  activeTab: TabId;
  onTab: (t: TabId) => void;
  muted: boolean;
  onToggleMute: () => void;
  onOpenSettings: () => void;
  onOpenAwards: () => void;
  onNewCareer: () => void;
  onOpenSlots: () => void;
  onSimWeek: () => void;
  onSimMonth: () => void;
  onSimSeason: () => void;
  onContinue?: () => void;
  simulating?: boolean;
  seasonName?: string;
  calendarLabel?: string;
  inboxUnread?: number;
  onOpenClub?: (clubId: string) => void;
  onOpenPlayer?: (playerId: string) => void;
  onOpenCompetition?: (competitionId: string) => void;
}

const TAB_ICONS: Record<TabId, React.ElementType> = {
  8: Home, 0: CircleDot, 7: Trophy, 2: Medal, 3: Users, 9: UserRound,
  4: WalletCards, 6: Inbox, 5: History, 1: Sparkles, 10: BarChart3,
};

export const TopBar = React.memo<TopBarProps>(function TopBar({
  activeTab, onTab, muted, onToggleMute, onOpenSettings, onOpenAwards, onNewCareer, onOpenSlots,
  onSimWeek, onSimMonth, onSimSeason, onContinue, simulating = false,
  seasonName = '2026-27', calendarLabel, inboxUnread = 0,
  onOpenClub, onOpenPlayer, onOpenCompetition,
}) {
  const playerSheet = usePlayerSheet();
  const activate = (id: TabId) => {
    soundManager.playClick();
    onTab(id);
  };

  return (
    <>
      <aside className="career-sidebar hidden lg:flex" aria-label="Career navigation">
        <button type="button" onClick={() => activate(8)} className="career-club-lockup text-left">
          <span className="career-club-crest grid h-[46px] w-[46px] place-items-center rounded-full border border-brass/30 bg-brass/10 text-brass"><Globe2 size={24} /></span>
          <span className="min-w-0">
            <span className="block font-display text-[20px] font-bold leading-none text-bone truncate">
              Football World
            </span>
            <span className="block mt-1 text-[10px] font-semibold uppercase tracking-[0.16em] text-brass/70 truncate">
              Neutral career simulation
            </span>
          </span>
        </button>

        <nav className="career-nav" aria-label="World sections">
          {NAV_GROUPS.map((group) => (
            <div key={group.label} className="mb-1">
              <div className="px-4 pb-1.5 pt-3 text-[9px] font-semibold uppercase tracking-[0.18em] text-sage/50">{group.label}</div>
              {group.slugs.map((slug) => {
                const tab = TABS.find((t) => t.slug === slug)!;
                const Icon = TAB_ICONS[tab.id];
                const active = tab.id === activeTab;
                return (
                  <button key={tab.id} type="button" onClick={() => activate(tab.id)} aria-current={active ? 'page' : undefined}
                    className={cx('career-nav-item', active && 'career-nav-item-active')}>
                    <Icon size={17} strokeWidth={active ? 2.2 : 1.7} aria-hidden="true" />
                    <span>{tab.label}</span>
                    {tab.slug === 'inbox' && inboxUnread > 0 && <span className="nav-count">{inboxUnread}</span>}
                  </button>
                );
              })}
            </div>
          ))}
        </nav>

        <div className="mt-auto border-t border-white/[0.07] p-4 space-y-2">
          <button type="button" onClick={onOpenAwards} className="sidebar-utility"><Trophy size={15} /> Honours</button>
          <button type="button" onClick={onNewCareer} className="sidebar-utility"><Plus size={15} /> New career</button>
          <button type="button" onClick={onOpenSlots} className="sidebar-utility"><Archive size={15} /> Save slots</button>
        </div>
      </aside>

      <header className="career-topbar sticky top-0 z-40 lg:ml-[216px]" style={{ paddingTop: 'env(safe-area-inset-top)' }}>
        <div className="px-4 sm:px-6 lg:px-7 min-h-[66px] flex flex-wrap items-center justify-between gap-2 py-2">
          <div className="flex min-w-0 items-center gap-3">
            <div className="lg:hidden grid h-9 w-9 shrink-0 place-items-center rounded-full border border-brass/30 bg-brass/10 text-brass"><Globe2 size={19} /></div>
            <div className="min-w-0">
              <div className="flex items-center gap-2 text-[10px] font-semibold uppercase tracking-[0.16em] text-sage/60">
                <CalendarDays size={12} /><span className="truncate">{calendarLabel || `Season ${seasonName.replace('-', '–')}`}</span>
              </div>
              <h1 className="mt-0.5 truncate font-display text-[18px] font-semibold leading-none text-bone">
                {TABS.find((tab) => tab.id === activeTab)?.label || 'Career'}
              </h1>
            </div>
          </div>

          <div className="flex min-w-0 flex-1 flex-wrap items-center justify-end gap-2">
            {onOpenClub && onOpenCompetition && (
              <div className="hidden min-w-0 xl:block">
                <GlobalSearch
                  onOpenClub={onOpenClub}
                  onOpenPlayer={onOpenPlayer ?? playerSheet.openPlayer}
                  onOpenCompetition={onOpenCompetition}
                />
              </div>
            )}
            <div className="hidden md:flex simulation-cluster shrink-0" role="group" aria-label="Simulation controls">
              <button disabled={simulating} onClick={onSimWeek} title="Simulate one week (W)">Week</button>
              <button disabled={simulating} onClick={onSimMonth} title="Simulate one month (M)">Month</button>
              <button disabled={simulating} onClick={onSimSeason} title="Simulate season (Shift+S)">Season</button>
            </div>
            <button type="button" onClick={onToggleMute} aria-pressed={muted} aria-label={muted ? 'Unmute sound' : 'Mute sound'} className="icon-button hidden sm:grid shrink-0">
              {muted ? <VolumeX size={17} /> : <Volume2 size={17} />}
            </button>
            <button type="button" onClick={onOpenSettings} aria-label="Viewer settings" className="icon-button hidden sm:grid shrink-0">
              <Settings2 size={17} />
            </button>
            {onContinue && (
              <button disabled={simulating} onClick={onContinue} className="continue-button shrink-0">
                <Activity size={16} /><span>{simulating ? 'Advancing…' : 'Continue'}</span><kbd>C</kbd>
              </button>
            )}
          </div>
        </div>

        <nav className="lg:hidden flex gap-1.5 overflow-x-auto border-t border-white/[0.06] px-3 py-2.5 no-scrollbar" aria-label="Mobile career navigation">
          {TABS.map((tab) => (
            <button
              key={tab.id}
              type="button"
              onClick={() => activate(tab.id)}
              aria-current={activeTab === tab.id ? 'page' : undefined}
              className={cx(
                'relative flex min-h-10 shrink-0 items-center gap-2 border px-3 text-[12px] font-semibold transition-colors',
                activeTab === tab.id
                  ? 'border-brass/50 bg-bone text-ink shadow-md'
                  : 'border-transparent text-sage hover:border-white/10 hover:bg-white/[0.04] hover:text-bone',
              )}
            >
              {React.createElement(TAB_ICONS[tab.id], { size: 15, strokeWidth: activeTab === tab.id ? 2.2 : 1.8, 'aria-hidden': true })}
              <span>{tab.label}</span>
              {tab.slug === 'inbox' && inboxUnread > 0 && (
                <span className={cx('min-w-5 rounded-full px-1.5 py-0.5 text-center text-[10px] font-bold', activeTab === tab.id ? 'bg-ink/10 text-ink' : 'bg-ember text-white')}>
                  {inboxUnread}
                </span>
              )}
            </button>
          ))}
</nav>
    </header>
    </>
  );
});
