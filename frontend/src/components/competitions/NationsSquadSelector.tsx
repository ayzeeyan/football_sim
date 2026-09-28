import React, { useEffect, useMemo, useState } from 'react';
import { Check, RotateCcw, ShieldCheck } from 'lucide-react';
import type { NationsSquadSelection } from '../../types';
import { clearNationsSquad, fetchNationsSquadSelection, setNationsSquad } from '../../services/api';
import { cx } from '../../lib/format';
import { soundManager } from '../../audio/webAudio';
import { Modal, ModalHeader } from '../ui/ui';

/**
 * Tier B national squad selection (B6): the viewer picks one nation's
 * 23-player squad from the eligible pool. Eligibility is enforced by the
 * backend — only players whose original club country matches the nation are
 * listed — and the AI selection stays the default until the viewer opts in.
 */
export const NationsSquadSelector: React.FC<{
  teamId: string | null;
  teamName?: string;
  onClose: () => void;
  onChanged?: () => void;
}> = ({ teamId, teamName, onClose, onChanged }) => {
  const [selection, setSelection] = useState<NationsSquadSelection | null>(null);
  const [picked, setPicked] = useState<Set<string>>(new Set());
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!teamId) {
      setSelection(null);
      return;
    }
    let cancelled = false;
    setError(null);
    fetchNationsSquadSelection(teamId)
      .then((payload) => {
        if (cancelled || !payload) return;
        setSelection(payload);
        setPicked(new Set(payload.eligible_pool.filter((row) => row.selected).map((row) => row.player_id)));
      })
      .catch(() => {
        if (!cancelled) setError('Could not load the eligible pool.');
      });
    return () => { cancelled = true; };
  }, [teamId]);

  const squadSize = selection?.squad_size ?? 23;
  const goalkeepers = useMemo(
    () => (selection?.eligible_pool ?? []).filter((row) => picked.has(row.player_id) && row.category === 'GK').length,
    [selection, picked],
  );
  const complete = picked.size === squadSize && goalkeepers >= 1;

  const toggle = (playerId: string) => {
    soundManager.playClick();
    setPicked((prev) => {
      const next = new Set(prev);
      if (next.has(playerId)) {
        next.delete(playerId);
      } else if (next.size < squadSize) {
        next.add(playerId);
      }
      return next;
    });
  };

  const save = async () => {
    if (!teamId || busy || !complete) return;
    setBusy(true);
    setError(null);
    try {
      const res = await setNationsSquad(teamId, [...picked].sort());
      if (res.status === 'error') {
        setError(res.message ?? 'Could not set the squad.');
      } else {
        soundManager.playWhistle();
        onChanged?.();
        onClose();
      }
    } finally {
      setBusy(false);
    }
  };

  const reset = async () => {
    if (!teamId || busy) return;
    setBusy(true);
    setError(null);
    try {
      const res = await clearNationsSquad(teamId);
      if (res.status === 'error') {
        setError(res.message ?? 'Could not clear the squad.');
      } else {
        soundManager.playClick();
        onChanged?.();
        onClose();
      }
    } finally {
      setBusy(false);
    }
  };

  const pool = selection?.eligible_pool ?? [];

  return (
    <Modal open={teamId != null} onClose={onClose} maxWidth="max-w-2xl" labelledBy="nations-squad-title">
      <div data-nations-squad-selector="true">
        <ModalHeader
          title={<span id="nations-squad-title">{teamName ?? selection?.team.name ?? 'National squad'}</span>}
          subtitle={`Pick exactly ${squadSize} eligible players · at least one goalkeeper`}
          onClose={onClose}
        />
        <div className="border-b border-line bg-cardLight/50 px-4 py-2.5">
          <div className="flex items-center justify-between gap-3">
            <p className="text-[12px] text-sage">
              <span className={cx('font-mono font-bold', picked.size === squadSize ? 'text-pitchtone' : 'text-brass')}>
                {picked.size}/{squadSize}
              </span>{' '}
              called up · <span className={cx('font-mono', goalkeepers >= 1 ? 'text-pitchtone' : 'text-red-400')}>{goalkeepers} GK</span>
              {selection?.viewer_selected ? ' · viewer-selected' : ' · AI-selected'}
            </p>
            <div className="flex items-center gap-2">
              {selection?.viewer_selected && (
                <button
                  type="button"
                  onClick={() => void reset()}
                  disabled={busy}
                  className="inline-flex items-center gap-1 rounded-md border border-line bg-cardLight px-2.5 py-1.5 text-[11px] font-semibold text-sage hover:border-brass/40 hover:text-bone"
                >
                  <RotateCcw size={12} aria-hidden="true" /> Back to AI
                </button>
              )}
              <button
                type="button"
                onClick={() => void save()}
                disabled={busy || !complete}
                className="inline-flex items-center gap-1 rounded-md border border-pitchtone/50 bg-pitchtone/15 px-2.5 py-1.5 text-[11px] font-semibold text-pitchtone hover:bg-pitchtone/25 disabled:opacity-40"
              >
                <Check size={12} aria-hidden="true" /> Announce squad
              </button>
            </div>
          </div>
        </div>
        {error && <p className="px-4 py-2 text-[12px] text-red-400">{error}</p>}
        <ul className="max-h-[26rem] divide-y divide-line/60 overflow-y-auto" data-nations-squad-pool="true">
          {pool.length === 0 && <li className="px-4 py-8 text-center text-[12px] text-sage">Loading the eligible pool…</li>}
          {pool.map((row) => {
            const on = picked.has(row.player_id);
            return (
              <li key={row.player_id}>
                <button
                  type="button"
                  onClick={() => toggle(row.player_id)}
                  aria-pressed={on}
                  className="flex w-full items-center justify-between gap-3 px-4 py-2 text-left hover:bg-cardLight/40"
                >
                  <span className="min-w-0">
                    <span className="block truncate text-[12px] font-semibold text-bone">
                      {row.full_name}
                      {row.category === 'GK' && <ShieldCheck size={11} className="ml-1 inline text-brass" aria-label="Goalkeeper" />}
                    </span>
                    <span className="block truncate text-[10px] text-sage">{row.position} · {row.club_name}</span>
                  </span>
                  <span className="flex shrink-0 items-center gap-2 font-mono text-[11px] text-sage">
                    {row.ovr} · {row.age}
                    <span className={cx('inline-flex h-4 w-4 items-center justify-center rounded border', on ? 'border-pitchtone bg-pitchtone/25 text-pitchtone' : 'border-line text-transparent')}>
                      <Check size={11} aria-hidden="true" />
                    </span>
                  </span>
                </button>
              </li>
            );
          })}
        </ul>
      </div>
    </Modal>
  );
};
