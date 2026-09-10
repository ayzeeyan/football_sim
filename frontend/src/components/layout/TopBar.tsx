import React from 'react';
import { Volume2, VolumeX } from 'lucide-react';
import { TABS, type TabId } from '../../lib/constants';
import type { ConnectionStatus } from '../../services/matchSocket';
import { soundManager } from '../../audio/webAudio';
import { cx } from '../../lib/format';

interface TopBarProps {
  activeTab: TabId;
  onTab: (t: TabId) => void;
  wsStatus: ConnectionStatus;
  muted: boolean;
  onToggleMute: () => void;
  onOpenAwards: () => void;
  onNewCareer: () => void;
  inboxUnread?: number;
}

function statusCopy(s: ConnectionStatus): { live: boolean; label: string } {
  if (s === 'CONNECTED') return { live: true, label: 'Live' };
  if (s === 'CONNECTING' || s === 'RECONNECTING') return { live: false, label: s === 'RECONNECTING' ? 'Reconnecting' : 'Connecting' };
  return { live: false, label: 'Offline' };
}

export const TopBar: React.FC<TopBarProps> = ({ activeTab, onTab, wsStatus, muted, onToggleMute, onOpenAwards, onNewCareer, inboxUnread = 0 }) => {
  const status = statusCopy(wsStatus);
  return (
    <header
      className="sticky top-0 z-40 bg-obsidian/95 backdrop-blur border-b border-line"
      style={{ paddingTop: 'env(safe-area-inset-top)' }}
    >
      <div className="page-shell py-3 flex items-center justify-between gap-4">
        <div className="min-w-0">
          <h1 className="font-display text-[26px] leading-none font-semibold tracking-tight text-bone">
            Super League
          </h1>
          <p className="text-[13px] text-sage mt-1">Season 2026–27</p>
        </div>

        <nav className="hidden md:flex items-center gap-0.5 min-w-0" aria-label="Primary">
          {TABS.map((t) => (
            <button
              key={t.id}
              onClick={() => {
                soundManager.playClick();
                onTab(t.id);
              }}
              aria-current={activeTab === t.id ? 'page' : undefined}
              className={cx(
                'px-3 py-1.5 text-[14px] font-medium transition-colors',
                activeTab === t.id
                  ? 'bg-bone text-ink'
                  : 'text-sage hover:text-bone',
              )}
            >
              {t.label}
              {t.slug === 'inbox' && inboxUnread > 0 && (
                <span className="ml-1.5 font-mono text-[11px] text-brass">{inboxUnread}</span>
              )}
            </button>
          ))}
        </nav>

        <div className="flex items-center gap-2 shrink-0">
          <div className="flex items-center gap-2 text-[13px]" role="status">
            <span
              className={cx('w-2 h-2 rounded-full', status.live ? 'bg-ember' : 'bg-sage/50')}
              aria-hidden="true"
            />
            <span className={status.live ? 'text-ember font-semibold' : 'text-sage'}>{status.label}</span>
          </div>
          <button
            onClick={onToggleMute}
            aria-pressed={muted}
            aria-label={muted ? 'Unmute sound' : 'Mute sound'}
            className="p-2 text-sage hover:text-bone transition-colors"
          >
            {muted ? <VolumeX size={16} aria-hidden="true" /> : <Volume2 size={16} aria-hidden="true" />}
          </button>
          <button
            onClick={onNewCareer}
            className="px-3 py-1.5 text-[13px] font-semibold text-sage hover:text-bone border border-line hover:border-sage/50 transition-colors"
          >
            New career
          </button>
          <button
            onClick={onOpenAwards}
            className="px-3 py-1.5 text-[13px] font-semibold text-ink bg-bone hover:bg-[#fff6dc] transition-colors"
          >
            Honours
          </button>
        </div>
      </div>

      <nav className="md:hidden page-shell pb-3" aria-label="Primary">
        <div className="flex gap-1 overflow-x-auto overscroll-x-contain">
          {TABS.map((t) => (
            <button
              key={t.id}
              onClick={() => {
                soundManager.playClick();
                onTab(t.id);
              }}
              aria-current={activeTab === t.id ? 'page' : undefined}
              className={cx(
                'px-3 py-1.5 text-[14px] font-medium whitespace-nowrap transition-colors',
                activeTab === t.id ? 'bg-bone text-ink' : 'text-sage hover:text-bone',
              )}
            >
              {t.label}
              {t.slug === 'inbox' && inboxUnread > 0 && (
                <span className="ml-1.5 font-mono text-[11px] text-brass">{inboxUnread}</span>
              )}
            </button>
          ))}
        </div>
      </nav>
    </header>
  );
};
