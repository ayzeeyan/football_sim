import React from 'react';
import { Star } from 'lucide-react';
import type { Club, MatchPlayerRow } from '../../types';
import { ClubCrest, PlayerPortrait } from '../ui/ui';

export function PlayerOfMatch({
  motm,
  home,
  away,
  onOpenPlayer,
}: {
  motm?: (MatchPlayerRow & { side?: string }) | null;
  home: Club | null;
  away: Club | null;
  onOpenPlayer: (id: string) => void;
}) {
  if (!motm) {
    return <p className="mt-3 text-[12px] text-sage">Official selection is not available.</p>;
  }
  const club = motm.side === 'away' ? away : home;
  const bits = [
    motm.match_goals ? `${motm.match_goals} Goal${motm.match_goals === 1 ? '' : 's'}` : '',
    motm.match_assists ? `${motm.match_assists} Assist${motm.match_assists === 1 ? '' : 's'}` : '',
  ].filter(Boolean);

  return (
    <button type="button" onClick={() => onOpenPlayer(motm.player_id)} className="group relative mt-3 flex w-full min-w-0 items-center gap-3 overflow-hidden rounded-md border border-brass/25 bg-gradient-to-r from-brass/10 to-transparent p-3 text-left hover:border-brass/50">
      <div className="absolute right-2 top-2 opacity-20"><ClubCrest club={club} size={50} className="!border-0 !bg-transparent" /></div>
      <PlayerPortrait player={motm} size={52} />
      <div className="min-w-0 flex-1">
        <p className="flex items-center gap-1.5 truncate font-display text-[17px] font-bold text-bone group-hover:text-brass"><Star size={12} className="shrink-0 fill-brass text-brass" /><span className="truncate">{motm.full_name}</span></p>
        <p className="truncate text-[10px] text-sage">{club?.club_name || motm.position} · {motm.position}</p>
        <p className="mt-1 text-[10px] font-semibold text-bone/80">{bits.length ? bits.join(' · ') : `${motm.minutes ?? 90} minutes`}</p>
      </div>
      <span className="relative rounded-md bg-brass px-2.5 py-1.5 font-mono text-[19px] font-bold tabular-nums text-ink">{(motm.rating ?? 0).toFixed(1)}</span>
    </button>
  );
}
