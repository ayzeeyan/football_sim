import React from 'react';
import { Settings2 } from 'lucide-react';
import type { ViewerSettings } from '../../lib/settings';
import { formatDateParts } from '../../lib/settings';
import { formatEUR } from '../../lib/format';
import { Modal, ModalHeader } from '../ui/ui';

/**
 * Viewer settings (F8): density, motion, sound, number/date format, default
 * tab, auto-advance, and notification preferences. Persisted in
 * localStorage — these belong to the person watching the world, not to the
 * world itself, so they survive career resets without touching the save
 * format.
 */
export const SettingsPanel: React.FC<{
  open: boolean;
  settings: ViewerSettings;
  onChange: (next: ViewerSettings) => void;
  onClose: () => void;
}> = ({ open, settings, onChange, onClose }) => {
  const set = <K extends keyof ViewerSettings>(key: K, value: ViewerSettings[K]) => {
    onChange({ ...settings, [key]: value });
  };

  return (
    <Modal open={open} onClose={onClose} maxWidth="max-w-2xl">
      <ModalHeader
        title={
          <span className="flex items-center gap-2">
            <Settings2 size={18} aria-hidden="true" /> Viewer settings
          </span>
        }
        subtitle="Preferences persist in this browser and survive career resets."
        onClose={onClose}
      />
      <div className="space-y-5 p-4 sm:p-5">
        <section>
          <p className="match-section-title">Display</p>
          <div className="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-2">
            <label className="flex items-center justify-between gap-3 rounded-md border border-line bg-black/20 px-3 py-2.5">
              <span className="text-[12px] font-semibold text-bone">Density</span>
              <select
                value={settings.density}
                onChange={(e) => set('density', e.target.value as ViewerSettings['density'])}
                className="border border-line bg-ink px-2 py-1 text-[12px] text-bone"
                aria-label="Density"
              >
                <option value="comfortable">Comfortable</option>
                <option value="compact">Compact</option>
              </select>
            </label>
            <label className="flex items-center justify-between gap-3 rounded-md border border-line bg-black/20 px-3 py-2.5">
              <span className="text-[12px] font-semibold text-bone">Reduced motion</span>
              <input
                type="checkbox"
                checked={settings.reducedMotion}
                onChange={(e) => set('reducedMotion', e.target.checked)}
                aria-label="Reduced motion"
              />
            </label>
            <label className="flex items-center justify-between gap-3 rounded-md border border-line bg-black/20 px-3 py-2.5">
              <span className="text-[12px] font-semibold text-bone">Currency</span>
              <select
                value={settings.numberFormat}
                onChange={(e) => set('numberFormat', e.target.value as ViewerSettings['numberFormat'])}
                className="border border-line bg-ink px-2 py-1 text-[12px] text-bone"
                aria-label="Currency format"
              >
                <option value="compact">Compact (€25M)</option>
                <option value="full">Full (€25,000,000)</option>
              </select>
            </label>
            <label className="flex items-center justify-between gap-3 rounded-md border border-line bg-black/20 px-3 py-2.5">
              <span className="text-[12px] font-semibold text-bone">Dates</span>
              <select
                value={settings.dateFormat}
                onChange={(e) => set('dateFormat', e.target.value as ViewerSettings['dateFormat'])}
                className="border border-line bg-ink px-2 py-1 text-[12px] text-bone"
                aria-label="Date format"
              >
                <option value="dmy">Day first (28/09/2026)</option>
                <option value="mdy">Month first (09/28/2026)</option>
              </select>
            </label>
          </div>
          <p className="mt-2 text-[11px] text-sage">
            Preview: {formatEUR(25_000_000)} · {formatDateParts(2026, 9, 28, settings)}
          </p>
        </section>

        <section>
          <p className="match-section-title">Sound &amp; flow</p>
          <div className="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-2">
            <label className="flex items-center justify-between gap-3 rounded-md border border-line bg-black/20 px-3 py-2.5">
              <span className="text-[12px] font-semibold text-bone">Mute sound</span>
              <input
                type="checkbox"
                checked={settings.soundMuted}
                onChange={(e) => set('soundMuted', e.target.checked)}
                aria-label="Mute sound"
              />
            </label>
            <label className="flex items-center justify-between gap-3 rounded-md border border-line bg-black/20 px-3 py-2.5">
              <span className="text-[12px] font-semibold text-bone">Auto-advance (seconds)</span>
              <input
                type="number"
                min={0}
                max={600}
                step={5}
                value={settings.autoAdvanceSeconds}
                onChange={(e) => set('autoAdvanceSeconds', Math.max(0, Math.min(600, Number(e.target.value) || 0)))}
                className="w-20 border border-line bg-ink px-2 py-1 text-right text-[12px] text-bone"
                aria-label="Auto-advance seconds"
              />
            </label>
          </div>
          <p className="mt-2 text-[11px] text-sage">
            {settings.autoAdvanceSeconds > 0
              ? `The world advances one matchweek every ${settings.autoAdvanceSeconds}s while no modal is open.`
              : 'Auto-advance is off; the world only moves when you press a simulation control.'}
          </p>
        </section>

        <section>
          <p className="match-section-title">Default landing tab</p>
          <select
            value={settings.defaultTab}
            onChange={(e) => set('defaultTab', e.target.value)}
            className="mt-3 w-full border border-line bg-ink px-2 py-2 text-[12px] text-bone"
            aria-label="Default landing tab"
          >
            <option value="">Home</option>
            <option value="matches">Match Centre</option>
            <option value="news">News</option>
            <option value="tables">Tables</option>
            <option value="competitions">Competitions</option>
            <option value="history">History</option>
            <option value="clubs">Clubs</option>
            <option value="players">Players</option>
            <option value="transfers">Transfers</option>
            <option value="wonderkids">Wonderkids</option>
            <option value="stats">Statistics</option>
          </select>
        </section>

        <section>
          <p className="match-section-title">Notifications</p>
          <p className="mt-1 text-[11px] text-sage">Which inbox categories count toward the unread badge.</p>
          <div className="mt-3 grid grid-cols-1 gap-2 sm:grid-cols-2">
            {(
              [
                ['notifyMatch', 'Match reports, cups, and races'],
                ['notifyTransfers', 'Transfers and watchlist digests'],
                ['notifyMilestones', 'Milestones, honours, and dugout news'],
                ['notifyWonderkids', 'Wonderkids, youth, and NXGN'],
              ] as Array<[keyof ViewerSettings, string]>
            ).map(([key, label]) => (
              <label key={key} className="flex items-center justify-between gap-3 rounded-md border border-line bg-black/20 px-3 py-2.5">
                <span className="text-[12px] text-bone">{label}</span>
                <input
                  type="checkbox"
                  checked={settings[key] as boolean}
                  onChange={(e) => set(key, e.target.checked as ViewerSettings[typeof key])}
                  aria-label={label}
                />
              </label>
            ))}
          </div>
        </section>
      </div>
    </Modal>
  );
};
