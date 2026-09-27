import React from 'react';
import type { Club, Fixture, MatchEventItem } from '../../types';
import { ClubCrest } from '../ui/ui';
import { prettyCompetitionName } from '../clubs/PlayerSheet';
import { scorerLines } from '../../lib/matchStory';
import { CalendarDays, MapPin, Users } from 'lucide-react';

export const MatchResultHeader = React.memo(function MatchResultHeader({
  fixture,
  home,
  away,
  homeScore,
  awayScore,
  events,
}: {
  fixture?: Fixture | null;
  home: Club | null;
  away: Club | null;
  homeScore: number;
  awayScore: number;
  events: MatchEventItem[];
}) {
  const decided = fixture?.decided_by === 'penalties'
    ? 'Penalties'
    : fixture?.decided_by === 'extra_time'
      ? 'After extra time'
      : 'Full time';
  const homeScorers = scorerLines(events, 'home');
  const awayScorers = scorerLines(events, 'away');
  const ht = fixture?.ht_home != null && fixture?.ht_away != null
    ? `HT ${fixture.ht_home}–${fixture.ht_away}`
    : '';

  return (
    <header className="relative shrink-0 overflow-hidden border-b border-white/10 bg-gradient-to-br from-[#194330] via-[#0c2519] to-[#07140e] px-4 py-4 sm:px-7 sm:py-5">
      <div className="pointer-events-none absolute inset-0 opacity-50 [background-image:linear-gradient(rgba(255,255,255,.025)_1px,transparent_1px),linear-gradient(90deg,rgba(255,255,255,.025)_1px,transparent_1px)] [background-size:32px_32px]" />
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_50%_110%,rgba(243,230,196,.15),transparent_48%)]" />
      <div className="relative flex flex-wrap items-center justify-between gap-2 text-[10px] font-semibold uppercase tracking-[0.14em] text-sage">
        <p className="min-w-0 truncate"><span className="text-brass">{fixture ? prettyCompetitionName(fixture.competition) : 'Match result'}</span>{fixture?.stage ? <span> · {fixture.stage.replace(/_/g, ' ')}</span> : null}</p>
        <p className="shrink-0 rounded-full border border-white/10 bg-black/25 px-3 py-1 text-bone">{decided}</p>
      </div>

      <div className="relative mx-auto mt-4 grid max-w-4xl grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] items-start gap-3 sm:gap-8">
        <div className="flex min-w-0 flex-col items-center text-center">
          <ClubCrest club={home} size={70} className="!border-0 !bg-transparent" />
          <h2 className="mt-2 max-w-full truncate font-display text-[20px] font-bold text-bone sm:text-[26px]">{home?.club_name}</h2>
          <div className="mt-2 min-h-[32px] max-w-full space-y-0.5 text-[10px] text-sage">{homeScorers.map((line) => <p key={line} className="truncate">{line}</p>)}</div>
        </div>
        <div className="pt-3 text-center">
          <p className="text-[9px] font-bold uppercase tracking-[0.18em] text-sage">Full time</p>
          <div className="score-display mt-2 whitespace-nowrap text-[46px] font-bold leading-none text-bone sm:text-[58px]">{homeScore}<span className="mx-2 text-brass/45">–</span>{awayScore}</div>
          <div className="mt-2 space-y-0.5 text-[10px] text-sage">
            {fixture?.decided_by === 'penalties' ? <p>{fixture.penalties?.[0] ?? 0}–{fixture.penalties?.[1] ?? 0} on penalties</p> : null}
            {ht ? <p>{ht}</p> : null}
          </div>
        </div>
        <div className="flex min-w-0 flex-col items-center text-center">
          <ClubCrest club={away} size={70} className="!border-0 !bg-transparent" />
          <h2 className="mt-2 max-w-full truncate font-display text-[20px] font-bold text-bone sm:text-[26px]">{away?.club_name}</h2>
          <div className="mt-2 min-h-[32px] max-w-full space-y-0.5 text-[10px] text-sage">{awayScorers.map((line) => <p key={line} className="truncate">{line}</p>)}</div>
        </div>
      </div>

      <div className="relative mx-auto mt-4 flex w-fit max-w-full flex-wrap justify-center gap-x-5 gap-y-2 rounded-full border border-white/[0.08] bg-black/20 px-4 py-2 text-[10px] text-sage">
        <span className="inline-flex items-center gap-1.5"><MapPin size={11} /><span className="max-w-[230px] truncate">{fixture?.preview?.venue || home?.home_stadium || 'Venue not supplied'}</span></span>
        <span className="inline-flex items-center gap-1.5"><CalendarDays size={11} />{fixture?.date_label || (fixture ? `Matchweek ${fixture.matchweek}` : 'Final')}</span>
        {fixture?.attendance ? <span className="hidden items-center gap-1.5 sm:inline-flex"><Users size={11} />{fixture.attendance.toLocaleString()}</span> : null}
        {fixture?.referee ? <span className="hidden md:inline">Referee · {fixture.referee}</span> : null}
      </div>
    </header>
  );
});
