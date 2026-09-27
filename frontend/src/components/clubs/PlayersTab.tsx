import React, { useEffect, useMemo, useState } from 'react';
import { Search } from 'lucide-react';
import { fetchSearch } from '../../services/api';
import { useAsyncData } from '../../hooks/useAsyncData';
import { usePlayerSheet } from '../clubs/PlayerSheet';
import { Card, ClubCrest, EmptyState, ErrorState, LoadingState, OvrBadge, PanelHeader } from '../ui/ui';
import { cx } from '../../lib/format';
import { soundManager } from '../../audio/webAudio';

type PosFilter = 'All' | 'GK' | 'DEF' | 'MID' | 'FWD';

interface PlayersTabProps {
  onViewClub: (clubId: string) => void;
}

/** Fetch window is larger than the visible page so client-side position filters
 *  still have rows to work with (the server truncates by limit before we filter). */
const SEARCH_FETCH_LIMIT = 120;
const SEARCH_DEBOUNCE_MS = 200;

export const PlayersTab: React.FC<PlayersTabProps> = ({ onViewClub }) => {
  const { openPlayer } = usePlayerSheet();
  const [query, setQuery] = useState('');
  const [debouncedQuery, setDebouncedQuery] = useState('');
  const [pos, setPos] = useState<PosFilter>('All');

  useEffect(() => {
    const handle = window.setTimeout(() => setDebouncedQuery(query), SEARCH_DEBOUNCE_MS);
    return () => window.clearTimeout(handle);
  }, [query]);

  const { data, loading, error, reload } = useAsyncData(
    () => fetchSearch(debouncedQuery.trim(), SEARCH_FETCH_LIMIT),
    [debouncedQuery],
  );

  const players = useMemo(() => {
    const rows = data?.players ?? [];
    if (pos === 'All') return rows;
    return rows.filter((row) => row.category === pos);
  }, [data, pos]);

  return (
    <div className="space-y-4">
      <Card>
        <PanelHeader
          kicker="World inspector"
          title="Players"
          subtitle="Browse the current save by rating, scoring, or name. Opening a player never takes control of their club."
        />
        <div className="mt-4 flex flex-wrap items-center gap-2">
          <div className="relative min-w-0 flex-1">
            <Search size={14} className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-sage" />
            <label className="sr-only" htmlFor="players-directory-search">Search players</label>
            <input
              id="players-directory-search"
              type="search"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Search a name, club or position…"
              className="field h-10 w-full pl-9 text-[13px]"
            />
          </div>
          {(['All', 'GK', 'DEF', 'MID', 'FWD'] as const).map((item) => (
            <button
              key={item}
              type="button"
              onClick={() => { soundManager.playClick(); setPos(item); }}
              aria-pressed={pos === item}
              className={cx('px-3 py-2 text-[12px] font-semibold border', pos === item ? 'bg-bone text-ink border-bone' : 'border-line text-sage hover:text-bone')}
            >
              {item}
            </button>
          ))}
        </div>
      </Card>

      {loading && <Card><LoadingState message="Loading the player directory…" /></Card>}
      {error && <Card><ErrorState message={error} onRetry={reload} /></Card>}
      {!loading && !error && players.length === 0 && <Card><EmptyState message="No players match that search." /></Card>}

      {!loading && !error && players.length > 0 && (
        <Card className="overflow-hidden !p-0">
          <div className="hidden md:grid grid-cols-[minmax(0,2fr)_70px_70px_70px_90px_70px] gap-2 table-head px-4 py-2">
            <span>Player</span><span>Pos</span><span>OVR</span><span>Age</span><span>Club</span><span>G / A</span>
          </div>
          <div className="divide-y divide-white/[0.06]">
            {players.map((player) => (
              <div
                key={player.player_id}
                className="grid w-full grid-cols-1 md:grid-cols-[minmax(0,2fr)_70px_70px_70px_90px_70px] items-center gap-2 px-4 py-2.5 hover:bg-white/[0.04]"
              >
                <button type="button" className="min-w-0 text-left" onClick={() => { soundManager.playClick(); openPlayer(player.player_id); }}>
                  <span className="block truncate font-semibold text-bone">{player.full_name}</span>
                  <span className="block text-[11px] text-sage md:hidden">{player.position} · {player.club_short || player.club_id}</span>
                </button>
                <span className="hidden text-[12px] text-sage md:block">{player.position}</span>
                <button type="button" className="justify-self-start" onClick={() => { soundManager.playClick(); openPlayer(player.player_id); }}>
                  <OvrBadge ovr={player.ovr} size="sm" />
                </button>
                <span className="hidden text-[12px] text-sage md:block">{player.age}</span>
                <button
                  type="button"
                  className="hidden min-w-0 items-center gap-1.5 text-left md:flex"
                  onClick={() => { soundManager.playClick(); onViewClub(player.club_id); }}
                >
                  <ClubCrest club={{ club_id: player.club_id, short_name: player.club_short, club_name: player.club_name }} size={18} className="!border-0 !bg-transparent" />
                  <span className="truncate text-[12px] text-bone">{player.club_short || player.club_id}</span>
                </button>
                <span className="hidden font-mono text-[12px] text-sage md:block">{player.goals} / {player.assists}</span>
              </div>
            ))}
          </div>
          <p className="px-4 py-3 text-[11px] text-sage">
            {pos !== 'All'
              ? `Showing ${players.length} ${pos} player${players.length === 1 ? '' : 's'}${data ? ` of ${data.players.length} matches` : ''}.`
              : 'Showing the highest-rated players in the current save. Search to inspect anyone.'}
          </p>
        </Card>
      )}
      {!loading && !error && data && data.clubs.length > 0 && query.trim().length >= 2 && (
        <Card>
          <p className="text-[10px] font-bold uppercase tracking-[0.14em] text-brass/70">Matching clubs</p>
          <div className="mt-3 grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-2">
            {data.clubs.map((club) => (
              <button
                key={club.club_id}
                type="button"
                onClick={() => { soundManager.playClick(); onViewClub(club.club_id); }}
                className="flex items-center gap-2 rounded-md border border-white/10 bg-black/15 px-3 py-2 text-left hover:border-brass/40"
              >
                <ClubCrest club={club} size={28} className="!border-0 !bg-transparent" />
                <span className="min-w-0">
                  <span className="block truncate text-[13px] font-semibold text-bone">{club.club_name}</span>
                  <span className="block text-[11px] text-sage">{club.league}</span>
                </span>
              </button>
            ))}
          </div>
        </Card>
      )}
    </div>
  );
};
