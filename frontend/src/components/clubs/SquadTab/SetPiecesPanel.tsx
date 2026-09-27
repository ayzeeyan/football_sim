import React, { useEffect, useState } from 'react';
import { Target } from 'lucide-react';
import type { Club, SetPiecePick } from '../../../types';
import { fetchClubSetPieces } from '../../../services/api';
import { Card, ClubCrest, EmptyState, LoadingState } from '../../ui/ui';

/**
 * Set-piece briefing (F10): who the match engine picks from the probable XI
 * for penalties, free kicks, corner delivery, and aerial targets — and the
 * reasoning behind each choice. Inspection only; the engine keeps its own
 * RNG-driven picks during play.
 */
export const SetPiecesPanel: React.FC<{ club: Club | null }> = ({ club }) => {
  const [picks, setPicks] = useState<{ penalty_taker?: SetPiecePick | null; free_kick_taker?: SetPiecePick | null; aerial_target?: SetPiecePick | null; corner_taker?: SetPiecePick | null } | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!club) return;
    let cancelled = false;
    setLoading(true);
    setError('');
    fetchClubSetPieces(club.club_id)
      .then((res) => {
        if (!cancelled) setPicks(res?.set_pieces ?? null);
      })
      .catch(() => {
        if (!cancelled) setError('The set-piece briefing could not be loaded.');
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [club?.club_id]);

  if (!club) return null;

  const rows: Array<{ label: string; pick?: SetPiecePick | null; note: string }> = [
    { label: 'Penalties', pick: picks?.penalty_taker, note: 'Strongest composite score among the XI' },
    { label: 'Free kicks', pick: picks?.free_kick_taker, note: 'Best dead-ball specialist (shooting 85+)' },
    { label: 'Aerial target', pick: picks?.aerial_target, note: 'Most likely corner header target' },
    { label: 'Corner taker', pick: picks?.corner_taker, note: 'Most likely delivery, never the aerial target' },
  ];

  return (
    <Card className="p-4 sm:p-5">
      <div className="flex items-center justify-between gap-3">
        <div>
          <p className="match-section-title">Set-piece briefing</p>
          <p className="mt-1 flex items-center gap-1.5 text-[12px] text-sage">
            <ClubCrest club={club} size={14} /> {club.club_name} · probable XI
          </p>
        </div>
        <span className="flex items-center gap-1 text-[10px] uppercase tracking-[0.12em] text-sage">
          <Target size={12} aria-hidden="true" /> Inspection
        </span>
      </div>

      {loading && <LoadingState message="Reading the engine's set-piece choices…" />}
      {error && <EmptyState message={error} />}

      {!loading && !error && (
        <div className="mt-4 grid grid-cols-1 gap-3 md:grid-cols-2">
          {rows.map((row) => (
            <div key={row.label} className="rounded-md border border-line bg-black/20 p-3">
              <p className="text-[10px] font-bold uppercase tracking-[0.12em] text-sage">{row.label}</p>
              {row.pick ? (
                <>
                  <p className="mt-1.5 text-[14px] font-semibold text-bone">
                    {row.pick.full_name}
                    <span className="ml-2 font-mono text-[11px] text-sage">
                      {row.pick.position} · {row.pick.ovr} OVR
                      {row.pick.confidence < 1 ? ` · ${(row.pick.confidence * 100).toFixed(0)}% likely` : ''}
                    </span>
                  </p>
                  <p className="mt-1 text-[11px] leading-relaxed text-sage">{row.pick.reason}</p>
                </>
              ) : (
                <p className="mt-1.5 text-[12px] text-sage">{row.note ? 'No qualified candidate in the probable XI.' : ''}</p>
              )}
            </div>
          ))}
        </div>
      )}

      <p className="mt-4 text-[11px] leading-relaxed text-sage/80">
        The engine draws these roles per match from weighted candidate lists; this briefing shows the most likely
        selections and the reasoning, not a locked-in lineup.
      </p>
    </Card>
  );
};
