import React from 'react';
import { Flame, Swords } from 'lucide-react';
import type { Fixture } from '../../types';
import { fetchHeadToHead } from '../../services/api';
import { useAsyncData } from '../../hooks/useAsyncData';
import { rgbCss } from '../../lib/format';
import { prettyCompetitionName } from '../clubs/PlayerSheet';
import { ClubCrest } from '../ui/ui';

/** Broadcast-style rivalry record: aggregate split bar, derby heat, recent meetings. */
export const HeadToHeadPanel: React.FC<{ fixture: Fixture }> = ({ fixture }) => {
  const homeId = fixture.home.club_id;
  const awayId = fixture.away.club_id;
  const { data, loading, error } = useAsyncData(
    () => fetchHeadToHead(homeId, awayId),
    [homeId, awayId],
  );

  if (loading) {
    return (
      <section className="console-card p-6 text-[12px] text-sage" aria-live="polite">
        Loading rivalry record…
      </section>
    );
  }

  if (error || !data) {
    return (
      <section className="console-card p-6 text-[12px] text-sage">
        The rivalry record is unavailable right now. Reopen the match to try again.
      </section>
    );
  }

  const total = Math.max(1, data.matches_played);
  const segments = [
    { key: 'a', value: data.wins_a, color: rgbCss(data.club_a.primary_color), label: data.club_a.short_name },
    { key: 'd', value: data.draws, color: 'rgba(167, 184, 154, 0.45)', label: 'Draws' },
    { key: 'b', value: data.wins_b, color: rgbCss(data.club_b.primary_color), label: data.club_b.short_name },
  ];
  const hasHistory = data.matches_played > 0;
  const isDerby = !!data.derby_name;

  return (
    <section className="console-card overflow-hidden">
      <div className="match-section-title p-4">
        <Swords size={14} /> Rivalry record
        {data.derby_name && (
          <span className="ml-auto inline-flex items-center gap-1.5 rounded-full border border-ember/40 bg-ember/10 px-2.5 py-1 text-[10px] font-bold normal-case tracking-normal text-bone">
            <Flame size={11} className="text-ember" /> {data.derby_name}
          </span>
        )}
      </div>

      <div className="border-t border-white/[0.07] p-4 sm:p-5">
        <div className="flex items-center justify-between gap-3">
          <span className="flex min-w-0 items-center gap-2">
            <ClubCrest club={data.club_a} size={26} className="!border-0 !bg-transparent" />
            <strong className="truncate font-display text-[16px] text-bone">{data.club_a.club_name}</strong>
          </span>
          <span className="shrink-0 font-mono text-[11px] text-sage">{data.matches_played} meeting{data.matches_played === 1 ? '' : 's'}</span>
          <span className="flex min-w-0 items-center justify-end gap-2">
            <strong className="truncate font-display text-[16px] text-bone">{data.club_b.club_name}</strong>
            <ClubCrest club={data.club_b} size={26} className="!border-0 !bg-transparent" />
          </span>
        </div>

        {hasHistory ? (
          <>
            <div className="mt-4 grid grid-cols-3 text-center">
              <strong className="block font-display text-[30px] leading-none text-bone">{data.wins_a}</strong>
              <strong className="block font-display text-[30px] leading-none text-sage">{data.draws}</strong>
              <strong className="block font-display text-[30px] leading-none text-bone">{data.wins_b}</strong>
            </div>
            <div className="mt-3 flex h-2.5 overflow-hidden rounded-full border border-white/[0.08]" role="img" aria-label={`${data.wins_a} wins for ${data.club_a.short_name}, ${data.draws} draws, ${data.wins_b} wins for ${data.club_b.short_name}`}>
              {segments.map((segment) => (
                <span
                  key={segment.key}
                  className="h-full first:rounded-l-full last:rounded-r-full"
                  style={{ width: `${(segment.value / total) * 100}%`, backgroundColor: segment.color }}
                  title={`${segment.label}: ${segment.value}`}
                />
              ))}
            </div>
            <p className="mt-3 text-center font-mono text-[11px] text-sage">
              {data.goals_a} goals for {data.club_a.short_name} · {data.goals_b} for {data.club_b.short_name}
            </p>
          </>
        ) : (
          <p className="mt-4 text-[12px] text-sage">No previous meetings are recorded in this career.</p>
        )}

        {isDerby && (
          <div className="mt-5 rounded-md border border-white/[0.07] bg-black/15 p-3.5">
            <div className="flex items-center justify-between text-[11px]">
              <span className="font-semibold uppercase tracking-[0.12em] text-sage">Derby heat</span>
              <span className="font-mono font-bold text-bone">{data.derby_heat}/100</span>
            </div>
            <div className="mt-2 h-1.5 overflow-hidden rounded-full bg-white/[0.07]">
              <div
                className="h-full rounded-full bg-gradient-to-r from-brass to-ember"
                style={{ width: `${Math.min(100, Math.max(0, data.derby_heat))}%` }}
              />
            </div>
            <p className="mt-2 text-[11px] text-sage">
              {data.derby_heat >= 70 ? 'One spark from boiling over.' : data.derby_heat >= 40 ? 'Edge in the air.' : 'History to be written.'}
            </p>
          </div>
        )}
      </div>

      {hasHistory && data.recent_matches.length > 0 && (
        <div className="border-t border-white/[0.07]">
          <p className="eyebrow p-4 pb-2">Recorded meetings</p>
          <div className="divide-y divide-white/[0.07]">
            {data.recent_matches.map((row) => {
              const homeWon = row.winner === row.home_id;
              const awayWon = row.winner === row.away_id;
              return (
                <div key={row.id} className="grid grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-4 px-4 py-2.5 text-[12px]">
                  <span className="font-mono text-sage">MW {row.matchweek}</span>
                  <span className="truncate text-sage">{prettyCompetitionName(row.competition)} · {row.stage || 'League'}</span>
                  <strong className="rounded bg-black/20 px-2 py-1 font-mono text-bone">
                    <span className={homeWon ? 'text-brass' : ''}>{row.home_goals}</span>
                    <span className="mx-1 text-sage">–</span>
                    <span className={awayWon ? 'text-brass' : ''}>{row.away_goals}</span>
                  </strong>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </section>
  );
};
