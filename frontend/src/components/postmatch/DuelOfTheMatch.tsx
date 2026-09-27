import React from 'react';
import { Swords } from 'lucide-react';
import type { Club, Fixture, MatchPlayerRow } from '../../types';
import { cx } from '../../lib/format';
import { ClubCrest, PlayerPortrait } from '../ui/ui';

function bestOfSide(rows: MatchPlayerRow[]): MatchPlayerRow | null {
  const rated = rows.filter((row) => row.played !== false && row.rating > 0);
  if (!rated.length) return null;
  const seed = rated[0];
  if (!seed) return null;
  return rated.reduce((best, row) => (row.rating > best.rating ? row : best), seed);
}

function ratingTone(rating: number): string {
  if (rating >= 8) return 'bg-[#1fbd72] text-[#051b10]';
  if (rating >= 7) return 'bg-[#8fcf58] text-[#10200a]';
  if (rating >= 6) return 'bg-[#efd36c] text-[#251d05]';
  return 'bg-[#e16658] text-white';
}

function SideCard({
  club,
  player,
  onOpenPlayer,
}: {
  club: Club | null;
  player: MatchPlayerRow | null;
  onOpenPlayer: (id: string) => void;
}) {
  if (!player) {
    return <p className="py-6 text-center text-[12px] text-sage">No rated player on this side.</p>;
  }
  return (
    <button
      type="button"
      onClick={() => onOpenPlayer(player.player_id)}
      className="flex min-w-0 flex-1 flex-col items-center gap-2 rounded-md border border-white/[0.08] bg-black/15 p-3 text-center transition-colors hover:border-brass/40"
    >
      <PlayerPortrait player={player} size={44} className="!rounded-full !border-2 shadow-lg shadow-black/35" />
      <span className="min-w-0">
        <strong className="block truncate text-[13px] text-bone">{player.full_name}</strong>
        <span className="mt-0.5 flex items-center justify-center gap-1.5 text-[10px] text-sage">
          <ClubCrest club={club} size={12} className="!border-0 !bg-transparent" />
          {club?.short_name} · {player.position}
        </span>
      </span>
      <span className="flex items-center gap-1.5 font-mono text-[10px] text-sage">
        {player.match_goals ? <span className="rounded bg-brass/20 px-1.5 py-0.5 font-bold text-brass">{player.match_goals} G</span> : null}
        {player.match_assists ? <span className="rounded bg-[#8AB4C8]/20 px-1.5 py-0.5 font-bold text-[#8AB4C8]">{player.match_assists} A</span> : null}
        {!player.match_goals && !player.match_assists ? <span>{player.minutes ?? 90} min</span> : null}
      </span>
      <span className={cx('rounded-full px-2.5 py-1 font-mono text-[13px] font-extrabold', ratingTone(player.rating))}>
        {player.rating.toFixed(1)}
      </span>
    </button>
  );
}

/** FotMob-style best-player-per-side duel card for the overview pane. */
export const DuelOfTheMatch = React.memo(function DuelOfTheMatch({
  fixture,
  homeRows,
  awayRows,
  onOpenPlayer,
}: {
  fixture: Fixture;
  homeRows: MatchPlayerRow[];
  awayRows: MatchPlayerRow[];
  onOpenPlayer: (id: string) => void;
}) {
  const homeBest = bestOfSide(homeRows);
  const awayBest = bestOfSide(awayRows);
  if (!homeBest && !awayBest) return null;
  return (
    <section className="console-card p-4">
      <h3 className="match-section-title"><Swords size={14} /> Duel of the match</h3>
      <p className="mt-1 text-[10px] text-sage">Each side's highest-rated performer</p>
      <div className="mt-3 flex items-stretch gap-2">
        <SideCard club={fixture.home} player={homeBest} onOpenPlayer={onOpenPlayer} />
        <span className="grid shrink-0 place-items-center font-display text-[13px] font-bold text-brass/70">VS</span>
        <SideCard club={fixture.away} player={awayBest} onOpenPlayer={onOpenPlayer} />
      </div>
    </section>
  );
});
