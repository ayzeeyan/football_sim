import React, { useCallback, useEffect, useState } from 'react';
import { Eye, EyeOff, Trophy, User } from 'lucide-react';
import { fetchWatchlist, toggleWatchlist, type WatchlistEntry, type WatchlistResponse } from '../../services/api';
import { cx } from '../../lib/format';
import { soundManager } from '../../audio/webAudio';

const EMPTY: WatchlistResponse = { clubs: [], players: [], competitions: [] };

export interface WatchlistPanelProps {
  onOpenClub?: (clubId: string) => void;
  onOpenPlayer?: (playerId: string) => void;
  onShowToast: (msg: string) => void;
}

/**
 * Multi-entity watchlist panel (Phase 3 F2). Purely observational: entries
 * filter what the weekly watch digest reports. Toggles come from the club
 * page and player sheet; this panel lists and un-watches.
 */
export const WatchlistPanel: React.FC<WatchlistPanelProps> = ({ onOpenClub, onOpenPlayer, onShowToast }) => {
  const [list, setList] = useState<WatchlistResponse>(EMPTY);
  const [busyId, setBusyId] = useState<string | null>(null);

  const reload = useCallback(() => {
    fetchWatchlist().then(setList).catch(() => undefined);
  }, []);

  useEffect(() => { reload(); }, [reload]);

  const remove = async (entity: 'club' | 'player' | 'competition', entry: WatchlistEntry) => {
    soundManager.playClick();
    setBusyId(entry.id);
    try {
      const res = await toggleWatchlist(entity, entry.id);
      if (res.status === 'error') {
        onShowToast('Could not update the watchlist.');
      } else {
        onShowToast(`${entry.name} removed from the watchlist.`);
        reload();
      }
    } finally {
      setBusyId(null);
    }
  };

  const total = list.clubs.length + list.players.length + list.competitions.length;

  const row = (entity: 'club' | 'player' | 'competition', entry: WatchlistEntry, open?: () => void) => (
    <li key={`${entity}-${entry.id}`} className="flex items-center justify-between gap-2 rounded-md px-2 py-1.5 hover:bg-white/[0.04]">
      <button type="button" onClick={open} className="min-w-0 flex-1 text-left" disabled={!open}>
        <span className="block truncate text-[12px] font-semibold text-bone">{entry.name}</span>
        <span className="block truncate text-[10px] text-sage">
          {entity === 'club' ? (entry.short_name ?? 'Club') : entity === 'player' ? `${entry.position ?? ''} · ${entry.club_short ?? ''} · ${entry.ovr ?? ''} OVR` : (entry.kind ?? 'Competition')}
        </span>
      </button>
      <button
        type="button"
        onClick={() => void remove(entity, entry)}
        disabled={busyId === entry.id}
        aria-label={`Remove ${entry.name} from the watchlist`}
        className="shrink-0 rounded-md border border-line bg-cardLight p-1.5 text-sage hover:border-brass/40 hover:text-bone"
      >
        <EyeOff size={13} aria-hidden="true" />
      </button>
    </li>
  );

  return (
    <section className="console-card p-5" data-watchlist-panel="true">
      <div className="flex items-center justify-between gap-2">
        <h3 className="font-display text-[16px] font-bold text-bone flex items-center gap-1.5">
          <Eye size={15} className="text-brass" aria-hidden="true" /> Watchlist
        </h3>
        <span className="text-[10px] font-semibold uppercase tracking-[0.12em] text-sage">{total} watched</span>
      </div>
      <p className="mt-1 text-[11px] text-sage">Watched clubs and players feed the weekly digest in your inbox.</p>
      {total === 0 ? (
        <p className="mt-3 text-[12px] text-sage">
          Nothing watched yet. Open a club page or player sheet and press the eye icon to follow them.
        </p>
      ) : (
        <div className="mt-3 space-y-3">
          {list.clubs.length > 0 && (
            <div>
              <p className="mb-1 flex items-center gap-1 text-[10px] font-bold uppercase tracking-[0.12em] text-sage"><Trophy size={11} aria-hidden="true" /> Clubs</p>
              <ul className="space-y-0.5">{list.clubs.map((c) => row('club', c, onOpenClub ? () => onOpenClub(c.id) : undefined))}</ul>
            </div>
          )}
          {list.players.length > 0 && (
            <div>
              <p className="mb-1 flex items-center gap-1 text-[10px] font-bold uppercase tracking-[0.12em] text-sage"><User size={11} aria-hidden="true" /> Players</p>
              <ul className="space-y-0.5">{list.players.map((p) => row('player', p, onOpenPlayer ? () => onOpenPlayer(p.id) : undefined))}</ul>
            </div>
          )}
          {list.competitions.length > 0 && (
            <div>
              <p className="mb-1 text-[10px] font-bold uppercase tracking-[0.12em] text-sage">Competitions</p>
              <ul className="space-y-0.5">{list.competitions.map((c) => row('competition', c))}</ul>
            </div>
          )}
        </div>
      )}
    </section>
  );
};

/** Small toggle button used on club pages and player sheets. */
export const WatchToggle: React.FC<{
  entity: 'club' | 'player' | 'competition';
  id: string;
  onShowToast: (msg: string) => void;
  className?: string;
}> = ({ entity, id, onShowToast, className }) => {
  const [watched, setWatched] = useState<boolean | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    let cancelled = false;
    fetchWatchlist().then((list) => {
      if (cancelled) return;
      const entries = entity === 'club' ? list.clubs : entity === 'player' ? list.players : list.competitions;
      setWatched(entries.some((e) => e.id === id));
    }).catch(() => undefined);
    return () => { cancelled = true; };
  }, [entity, id]);

  const toggle = async () => {
    if (busy) return;
    soundManager.playClick();
    setBusy(true);
    try {
      const res = await toggleWatchlist(entity, id);
      if (res.status === 'error') {
        onShowToast('Could not update the watchlist.');
      } else {
        setWatched(res.watched);
        onShowToast(res.watched ? 'Added to the watchlist.' : 'Removed from the watchlist.');
      }
    } finally {
      setBusy(false);
    }
  };

  return (
    <button
      type="button"
      onClick={() => void toggle()}
      disabled={busy || watched === null}
      aria-pressed={watched ?? false}
      aria-label={watched ? 'Remove from watchlist' : 'Add to watchlist'}
      title={watched ? 'Remove from watchlist' : 'Add to watchlist'}
      className={cx(
        'inline-flex min-h-9 items-center gap-1.5 border px-2.5 text-[12px] font-semibold transition-colors',
        watched ? 'border-brass/60 bg-brass/15 text-brass' : 'border-line bg-cardLight text-sage hover:border-brass/30 hover:text-bone',
        className,
      )}
    >
      {watched ? <Eye size={13} aria-hidden="true" /> : <EyeOff size={13} aria-hidden="true" />}
      {watched === null ? '…' : watched ? 'Watching' : 'Watch'}
    </button>
  );
};
