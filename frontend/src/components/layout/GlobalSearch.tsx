import React, { useEffect, useMemo, useRef, useState } from 'react';
import { Search } from 'lucide-react';
import { fetchSearch, type SearchResponse } from '../../services/api';
import { ClubCrest } from '../ui/ui';
import { cx } from '../../lib/format';
import { soundManager } from '../../audio/webAudio';

interface GlobalSearchProps {
  onOpenClub: (clubId: string) => void;
  onOpenPlayer: (playerId: string) => void;
  onOpenCompetition: (competitionId: string) => void;
}

export const GlobalSearch: React.FC<GlobalSearchProps> = ({ onOpenClub, onOpenPlayer, onOpenCompetition }) => {
  const [query, setQuery] = useState('');
  const [open, setOpen] = useState(false);
  const [results, setResults] = useState<SearchResponse | null>(null);
  const boxRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const q = query.trim();
    if (q.length < 2) {
      setResults(null);
      return;
    }
    const handle = window.setTimeout(() => {
      fetchSearch(q, 6)
        .then(setResults)
        .catch(() => setResults(null));
    }, 160);
    return () => window.clearTimeout(handle);
  }, [query]);

  useEffect(() => {
    const onDoc = (event: MouseEvent) => {
      if (!boxRef.current?.contains(event.target as Node)) setOpen(false);
    };
    document.addEventListener('mousedown', onDoc);
    return () => document.removeEventListener('mousedown', onDoc);
  }, []);

  const empty = useMemo(() => {
    if (!results) return true;
    return results.clubs.length + results.players.length + results.competitions.length === 0;
  }, [results]);

  return (
    <div ref={boxRef} className="relative min-w-0 w-[min(100%,16rem)]">
      <label className="sr-only" htmlFor="world-search">Search players, clubs and competitions</label>
      <Search size={14} className="pointer-events-none absolute left-2.5 top-1/2 -translate-y-1/2 text-sage" aria-hidden="true" />
      <input
        id="world-search"
        type="search"
        value={query}
        onChange={(e) => {
          setQuery(e.target.value);
          setOpen(true);
        }}
        onFocus={() => setOpen(true)}
        placeholder="Search world…"
        className="field h-9 w-full pl-8 pr-3 text-[12px]"
        autoComplete="off"
      />
      {open && query.trim().length >= 2 && (
        <div className="absolute right-0 z-50 mt-1 w-[min(24rem,calc(100vw-2rem))] overflow-hidden rounded-md border border-white/10 bg-[#0b1a13] shadow-2xl">
          {empty && <p className="px-3 py-4 text-[12px] text-sage">No matching players, clubs or competitions.</p>}
          {results && results.clubs.length > 0 && (
            <section>
              <p className="px-3 pt-2 text-[10px] font-bold uppercase tracking-[0.14em] text-brass/70">Clubs</p>
              {results.clubs.map((club) => (
                <button
                  key={club.club_id}
                  type="button"
                  className="flex w-full items-center gap-2 px-3 py-2 text-left hover:bg-white/[0.05]"
                  onClick={() => {
                    soundManager.playClick();
                    onOpenClub(club.club_id);
                    setOpen(false);
                    setQuery('');
                  }}
                >
                  <ClubCrest club={club} size={22} className="!border-0 !bg-transparent" />
                  <span className="min-w-0">
                    <span className="block truncate text-[12px] font-semibold text-bone">{club.club_name}</span>
                    <span className="block text-[10px] text-sage">{club.league} · {club.ovr} OVR</span>
                  </span>
                </button>
              ))}
            </section>
          )}
          {results && results.players.length > 0 && (
            <section>
              <p className="px-3 pt-2 text-[10px] font-bold uppercase tracking-[0.14em] text-brass/70">Players</p>
              {results.players.map((player) => (
                <button
                  key={player.player_id}
                  type="button"
                  className={cx('flex w-full items-center justify-between gap-2 px-3 py-2 text-left hover:bg-white/[0.05]')}
                  onClick={() => {
                    soundManager.playClick();
                    onOpenPlayer(player.player_id);
                    setOpen(false);
                    setQuery('');
                  }}
                >
                  <span className="min-w-0">
                    <span className="block truncate text-[12px] font-semibold text-bone">{player.full_name}</span>
                    <span className="block text-[10px] text-sage">{player.club_short || player.club_id} · {player.position}</span>
                  </span>
                  <span className="font-mono text-[11px] text-brass">{player.ovr}</span>
                </button>
              ))}
            </section>
          )}
          {results && results.competitions.length > 0 && (
            <section className="pb-1">
              <p className="px-3 pt-2 text-[10px] font-bold uppercase tracking-[0.14em] text-brass/70">Competitions</p>
              {results.competitions.map((comp) => (
                <button
                  key={comp.id}
                  type="button"
                  className="flex w-full items-center justify-between gap-2 px-3 py-2 text-left hover:bg-white/[0.05]"
                  onClick={() => {
                    soundManager.playClick();
                    onOpenCompetition(comp.id);
                    setOpen(false);
                    setQuery('');
                  }}
                >
                  <span className="truncate text-[12px] font-semibold text-bone">{comp.name}</span>
                  <span className="text-[10px] uppercase text-sage">{comp.kind.replace(/_/g, ' ')}</span>
                </button>
              ))}
            </section>
          )}
        </div>
      )}
    </div>
  );
};
