import { useMemo } from 'react';
import { AlertTriangle, Star } from 'lucide-react';
import type { Club, MatchPlayerRow } from '../../types';
import { cx, rgbCss } from '../../lib/format';
import { FORMATION_SLOTS, isOutOfPosition, normalizeFormation, pitchCoords, resolveLineupFormation, resolveTacticalAssignments } from '../../lib/tactics';
import { ClubCrest, PlayerPortrait } from '../ui/ui';

function ratingClass(rating: number, played: boolean): string {
  if (!played || !rating) return 'text-sage';
  if (rating >= 8) return 'text-emerald-300';
  if (rating < 6) return 'text-red-300';
  return 'text-bone';
}

function ratingBadgeClass(rating: number, played: boolean): string {
  if (!played || !rating) return 'bg-[#35463d] text-sage ring-white/10';
  if (rating >= 8) return 'bg-[#1fbd72] text-[#051b10] ring-emerald-200/40';
  if (rating >= 7) return 'bg-[#8fcf58] text-[#10200a] ring-lime-100/35';
  if (rating >= 6) return 'bg-[#efd36c] text-[#251d05] ring-amber-100/35';
  return 'bg-[#e16658] text-white ring-red-200/35';
}

function contribution(player: MatchPlayerRow): string {
  const bits: string[] = [];
  if (player.match_goals) bits.push('⚽'.repeat(Math.min(player.match_goals, 3)));
  if (player.match_assists) bits.push(player.match_assists > 1 ? `${player.match_assists}A` : 'A');
  if (player.match_og) bits.push('OG');
  if (player.card === 'red') bits.push('🟥');
  else if (player.card === 'yellow') bits.push('🟨');
  if (!player.starter && player.played) bits.push('↑');
  if (player.off_minute != null && player.starter) bits.push('↓');
  return bits.join(' ');
}

function ReportPitch({
  club,
  players,
  formation,
  motmId,
  onOpen,
}: {
  club: Club | null;
  players: MatchPlayerRow[];
  formation?: string;
  motmId?: string;
  onOpen: (id: string) => void;
}) {
  // Finished-match fixtures do not carry a formation field: recover the
  // shape from the tactical slots the report's XI actually played.
  const slots = useMemo(() => {
    const effectiveFormation = resolveLineupFormation(formation, players);
    const assigned = resolveTacticalAssignments(players, effectiveFormation);
    const allowed = new Set(FORMATION_SLOTS[effectiveFormation]);
    const used = new Set<string>();
    return assigned.flatMap(({ player, tacticalSlot }) => {
      const coords = pitchCoords(effectiveFormation, tacticalSlot);
      if (!coords || !allowed.has(tacticalSlot) || used.has(tacticalSlot)) return [];
      used.add(tacticalSlot);
      return [{ player, slot: tacticalSlot, x: coords[0], y: coords[1] }];
    });
  }, [players, formation]);

  const accent = rgbCss(club?.primary_color, '#F3E6C4');

  return (
    <div
      className="relative h-[390px] overflow-hidden rounded-lg border border-[#3b6b55] shadow-inner shadow-black/40 sm:h-[430px]"
      style={{ background: 'repeating-linear-gradient(0deg, #12422f 0%, #12422f 12.5%, #103b2b 12.5%, #103b2b 25%)' }}
    >
      <div className="pointer-events-none absolute inset-3 rounded-sm border border-bone/35" />
      <div className="pointer-events-none absolute left-3 right-3 top-1/2 border-t border-bone/30" />
      <div className="pointer-events-none absolute left-1/2 top-1/2 h-[64px] w-[64px] -translate-x-1/2 -translate-y-1/2 rounded-full border border-bone/30" />
      <div className="pointer-events-none absolute left-1/2 top-1/2 h-1.5 w-1.5 -translate-x-1/2 -translate-y-1/2 rounded-full bg-bone/35" />
      <div className="pointer-events-none absolute left-[24%] right-[24%] top-3 h-[19%] border border-t-0 border-bone/30" />
      <div className="pointer-events-none absolute left-[38%] right-[38%] top-3 h-[8%] border border-t-0 border-bone/30" />
      <div className="pointer-events-none absolute left-1/2 top-[15.5%] h-1.5 w-1.5 -translate-x-1/2 rounded-full bg-bone/35" />
      <div className="pointer-events-none absolute bottom-3 left-[24%] right-[24%] h-[19%] border border-b-0 border-bone/30" />
      <div className="pointer-events-none absolute bottom-3 left-[38%] right-[38%] h-[8%] border border-b-0 border-bone/30" />
      <div className="pointer-events-none absolute bottom-[15.5%] left-1/2 h-1.5 w-1.5 -translate-x-1/2 rounded-full bg-bone/35" />
      <div className="pointer-events-none absolute left-[43%] right-[43%] top-1 h-2 border-x border-b border-bone/30 bg-black/10" />
      <div className="pointer-events-none absolute bottom-1 left-[43%] right-[43%] h-2 border-x border-t border-bone/30 bg-black/10" />
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_50%_50%,rgba(255,255,255,.045),transparent_55%)]" />
      {slots.map(({ player, slot, x, y }) => {
        const rating = player.rating ?? 0;
        const surname = player.full_name.split(' ').slice(-1)[0];
        const notes = contribution(player);
        const isMotm = player.player_id === motmId;
        const outOfPosition = isOutOfPosition(player.position_fit);
        return (
          <button
            key={player.player_id}
            type="button"
            onClick={() => onOpen(player.player_id)}
            className="group absolute z-10 w-[82px] -translate-x-1/2 -translate-y-1/2 text-center focus-visible:z-20"
            style={{ left: `${x}%`, top: `${y}%` }}
            title={`${player.full_name} · ${slot} · ${rating ? `${rating.toFixed(1)} rating` : 'not rated'}${notes ? ` · ${notes}` : ''}`}
          >
            <span className="relative mx-auto block w-fit transition-transform group-hover:-translate-y-0.5 group-hover:scale-105">
              {isMotm ? <span className="absolute -inset-1.5 rounded-full border border-brass/80 shadow-[0_0_16px_rgba(243,230,196,.35)]" /> : null}
              <PlayerPortrait player={player} size={34} className="!rounded-full !border-2 shadow-lg shadow-black/35" />
              <span className={cx('absolute -bottom-1.5 -right-2 min-w-[30px] rounded-full px-1.5 py-0.5 font-mono text-[10px] font-extrabold tabular-nums ring-1', ratingBadgeClass(rating, !!player.played))}>
                {rating ? rating.toFixed(1) : '—'}
              </span>
              {isMotm ? <Star size={12} className="absolute -left-2 -top-2 fill-brass text-brass drop-shadow" /> : null}
              {player.card ? <span className={cx('absolute -right-2 -top-1 h-3.5 w-2.5 rounded-[2px] border border-black/20 shadow', player.card === 'red' ? 'bg-red-500' : 'bg-yellow-300')} /> : null}
              {outOfPosition ? <AlertTriangle size={11} className="absolute -bottom-2 -left-2 fill-amber-400 text-amber-900" /> : null}
            </span>
            <span className="mt-2.5 block truncate rounded bg-[#071810]/75 px-1.5 py-0.5 text-[10px] font-bold text-bone shadow-sm backdrop-blur-sm group-hover:bg-[#071810]">{surname}</span>
            <span className="mt-0.5 flex min-h-3 items-center justify-center gap-1 text-[9px] font-bold text-bone drop-shadow">
              {player.match_goals ? <span title={`${player.match_goals} goal${player.match_goals === 1 ? '' : 's'}`}>⚽{player.match_goals > 1 ? player.match_goals : ''}</span> : null}
              {player.match_assists ? <span className="rounded bg-sky-300 px-1 text-[8px] text-[#092131]" title={`${player.match_assists} assist${player.match_assists === 1 ? '' : 's'}`}>A{player.match_assists > 1 ? player.match_assists : ''}</span> : null}
              {player.off_minute != null ? <span className="rounded bg-black/55 px-1 text-[8px] text-sage">↓{player.off_minute}'</span> : null}
              {!player.match_goals && !player.match_assists && player.off_minute == null ? <span className="text-bone/55">{slot}</span> : null}
            </span>
          </button>
        );
      })}
      <div className="pointer-events-none absolute bottom-4 left-4 h-10 w-1 rounded-full opacity-70" style={{ backgroundColor: accent }} />
    </div>
  );
}

function RatingsList({
  title,
  rows,
  motmId,
  onOpen,
}: {
  title: string;
  rows: MatchPlayerRow[];
  motmId?: string;
  onOpen: (id: string) => void;
}) {
  const starters = rows.filter((row) => row.starter);
  const subs = rows.filter((row) => !row.starter && row.played);
  const render = (player: MatchPlayerRow) => {
    const rating = player.rating ?? 0;
    const isMotm = player.player_id === motmId;
    return (
      <button
        key={player.player_id}
        type="button"
        onClick={() => onOpen(player.player_id)}
        className="grid w-full grid-cols-[2.5rem_minmax(0,1fr)_auto] items-center gap-2 py-1.5 text-left hover:bg-white/[0.03]"
      >
        <span className={cx('font-mono text-[12px] font-bold tabular-nums', ratingClass(rating, !!player.played), isMotm && 'text-brass')}>
          {rating ? rating.toFixed(1) : '—'}
        </span>
        <span className="flex min-w-0 items-center gap-1.5">
          {isMotm ? <Star size={11} className="shrink-0 fill-brass text-brass" /> : null}
          <span className="event-row-name text-[12px] font-semibold text-bone">{player.full_name}</span>
        </span>
        <span className="shrink-0 text-[11px] text-sage">{contribution(player)}</span>
      </button>
    );
  };

  return (
    <section className="min-w-0">
      <h3 className="font-display text-[16px] font-bold text-bone">{title}</h3>
      {starters.length ? <div className="mt-2 divide-y divide-white/[0.07]">{starters.map(render)}</div> : <p className="mt-3 text-[12px] text-sage">Starting XI not stored.</p>}
      {subs.length ? (
        <>
          <p className="match-section-title mt-4">Substitutes</p>
          <div className="mt-1 divide-y divide-white/[0.07]">{subs.map(render)}</div>
        </>
      ) : null}
    </section>
  );
}

export function LineupView({
  home,
  away,
  homeRows,
  awayRows,
  homeFormation,
  awayFormation,
  motmId,
  onOpenPlayer,
}: {
  home: Club | null;
  away: Club | null;
  homeRows: MatchPlayerRow[];
  awayRows: MatchPlayerRow[];
  homeFormation?: string;
  awayFormation?: string;
  motmId?: string;
  onOpenPlayer: (id: string) => void;
}) {
  const homeXI = homeRows.filter((row) => row.starter);
  const awayXI = awayRows.filter((row) => row.starter);
  const resolvedHomeFormation = resolveLineupFormation(homeFormation, homeXI);
  const resolvedAwayFormation = resolveLineupFormation(awayFormation, awayXI);
  const sideHeader = (club: Club | null, rows: MatchPlayerRow[], formation?: string) => {
    const rated = rows.filter((row) => row.played !== false && row.rating > 0);
    const average = rated.length ? rated.reduce((sum, row) => sum + row.rating, 0) / rated.length : 0;
    return (
      <div className="mb-2 flex items-center justify-between gap-3 rounded-md border border-white/[0.08] bg-black/15 px-3 py-2">
        <span className="flex min-w-0 items-center gap-2"><ClubCrest club={club} size={28} className="!border-0 !bg-transparent" /><span className="min-w-0"><strong className="block truncate text-[13px] text-bone">{club?.club_name || 'Team'}</strong><small className="block text-[9px] uppercase tracking-[0.12em] text-sage">Starting XI</small></span></span>
        <span className="flex items-center gap-2 text-right"><span><strong className="block text-[12px] text-bone">{normalizeFormation(formation)}</strong><small className="block text-[9px] text-sage">Formation</small></span>{average ? <span className={cx('rounded px-2 py-1 font-mono text-[11px] font-bold ring-1', ratingBadgeClass(average, true))}>{average.toFixed(1)}</span> : null}</span>
      </div>
    );
  };
  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-center justify-between gap-2"><div><p className="match-section-title">Starting lineups</p><p className="mt-1 text-[11px] text-sage">Select a player for their full profile. Rating colours run from red to elite green.</p></div><div className="flex items-center gap-3 text-[9px] text-sage"><span className="inline-flex items-center gap-1"><i className="h-2.5 w-2.5 rounded-full bg-[#1fbd72]" />8.0+</span><span className="inline-flex items-center gap-1"><i className="h-2.5 w-2.5 rounded-full bg-[#8fcf58]" />7.0+</span><span className="inline-flex items-center gap-1"><i className="h-2.5 w-2.5 rounded-full bg-[#efd36c]" />6.0+</span></div></div>
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <div className="min-w-0">
          {sideHeader(home, homeXI.length ? homeXI : homeRows, resolvedHomeFormation)}
          <ReportPitch club={home} players={homeXI.length ? homeXI : homeRows} formation={resolvedHomeFormation} motmId={motmId} onOpen={onOpenPlayer} />
        </div>
        <div className="min-w-0">
          {sideHeader(away, awayXI.length ? awayXI : awayRows, resolvedAwayFormation)}
          <ReportPitch club={away} players={awayXI.length ? awayXI : awayRows} formation={resolvedAwayFormation} motmId={motmId} onOpen={onOpenPlayer} />
        </div>
      </div>
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <RatingsList title={home?.club_name || 'Home'} rows={homeRows} motmId={motmId} onOpen={onOpenPlayer} />
        <RatingsList title={away?.club_name || 'Away'} rows={awayRows} motmId={motmId} onOpen={onOpenPlayer} />
      </div>
    </div>
  );
}
