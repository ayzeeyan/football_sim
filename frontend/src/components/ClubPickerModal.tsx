import React, { useMemo, useState } from 'react';
import type { Club } from '../types';
import { Search, Shield } from 'lucide-react';
import { soundManager } from '../audio/webAudio';
import { LEAGUE_FILTER_ALL } from '../lib/constants';
import { cx } from '../lib/format';
import { ClubCrest, Modal, ModalHeader } from './ui/ui';

interface ClubPickerModalProps {
  isOpen: boolean;
  target: 'home' | 'away';
  clubs: Club[];
  onSelectClub: (club: Club) => void;
  onClose: () => void;
}

export const ClubPickerModal: React.FC<ClubPickerModalProps> = ({ isOpen, target, clubs, onSelectClub, onClose }) => {
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedLeague, setSelectedLeague] = useState<string>('All');

  const filteredClubs = useMemo(() => {
    const q = searchQuery.trim().toLowerCase();
    return clubs.filter((c) => {
      const matchesLeague = selectedLeague === 'All' || c.league === selectedLeague;
      if (!matchesLeague) return false;
      if (q === '') return true;
      return c.club_name.toLowerCase().includes(q) || c.short_name.toLowerCase().includes(q);
    });
  }, [clubs, selectedLeague, searchQuery]);

  return (
    <Modal open={isOpen} onClose={onClose}>
      <ModalHeader
        title={
          <span className="flex items-center gap-2">
            <Shield className="text-brass" size={18} />
            <span>Pick the {target} side to follow</span>
          </span>
        }
        subtitle="You are in the stands — this only changes which fixture is on your screen."
        onClose={onClose}
      />

      <div className="px-4 py-3 border-b border-line bg-dugout flex flex-wrap items-center justify-between gap-3">
        <div className="flex gap-1.5 overflow-x-auto" role="group" aria-label="League filter">
          {LEAGUE_FILTER_ALL.map((lg) => (
            <button
              key={lg}
              onClick={() => {
                soundManager.playClick();
                setSelectedLeague(lg);
              }}
              aria-pressed={selectedLeague === lg}
              className={cx(
                'px-3 py-1.5 rounded-lg text-[13px] font-semibold transition-colors whitespace-nowrap border',
                selectedLeague === lg ? 'bg-brass text-ink border-brass' : 'bg-cardLight text-sage hover:text-bone border-line',
              )}
            >
              {lg}
            </button>
          ))}
        </div>
        <div className="relative w-full sm:w-64 min-w-0">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-sage" size={14} aria-hidden="true" />
          <label className="sr-only" htmlFor="club-search">
            Search clubs
          </label>
          <input
            id="club-search"
            name="club-search"
            type="search"
            autoComplete="off"
            spellCheck={false}
            placeholder="Search clubs…"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="field w-full pl-9 pr-3 py-2 placeholder-sage/70"
          />
        </div>
      </div>

      <div className="flex-1 overflow-y-auto overscroll-contain p-4 grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-2.5 min-h-0">
        {filteredClubs.length === 0 ? (
          <div className="col-span-full py-16 text-center text-sage text-[13px]">No clubs match that search. Try a different name.</div>
        ) : (
          filteredClubs.map((club) => (
            <button
              key={club.club_id}
              onClick={() => {
                soundManager.playClick();
                onSelectClub(club);
                onClose();
              }}
              className="p-3 bg-cardLight/60 hover:bg-cardLight border border-line hover:border-sage/40 rounded-xl flex items-center justify-between transition-colors text-left group"
            >
              <span className="flex items-center gap-3 min-w-0">
                <ClubCrest club={club} size={44} />
                <span className="min-w-0">
                  <span className="block font-semibold text-[13px] text-bone group-hover:text-brass transition-colors truncate">{club.club_name}</span>
                  <span className="block text-[11px] text-sage font-mono">{club.league} · OVR {club.overall_team_rating}</span>
                </span>
              </span>
              <span className="font-mono text-[11px] px-2 py-0.5 rounded-md bg-ink/60 border border-line text-sage shrink-0">
                {club.short_name}
              </span>
            </button>
          ))
        )}
      </div>
    </Modal>
  );
};
