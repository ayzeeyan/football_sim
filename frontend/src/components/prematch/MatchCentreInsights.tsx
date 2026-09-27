import React from 'react';
import { Activity, ArrowRight, Sparkles, Swords, TrendingUp } from 'lucide-react';
import type { Club, Fixture, Player } from '../../types';
import { ClubCrest, FormPips, OvrBadge } from '../ui/ui';

function formScore(form: string[] = []): number {
  return form.slice(-5).reduce((total, result) => total + (result === 'W' ? 3 : result === 'D' ? 1 : 0), 0);
}

function clamp(value: number, min: number, max: number): number {
  return Math.max(min, Math.min(max, value));
}

export function matchPrediction(fixture: Fixture): { home: number; draw: number; away: number } {
  const preview = fixture.preview;
  const homeOvr = preview?.home_xi_avg || fixture.home.overall_team_rating || 75;
  const awayOvr = preview?.away_xi_avg || fixture.away.overall_team_rating || 75;
  const homeForm = formScore(preview?.home_form ?? fixture.home.form ?? []);
  const awayForm = formScore(preview?.away_form ?? fixture.away.form ?? []);
  const ratingEdge = (homeOvr - awayOvr) * 1.8;
  const formEdge = (homeForm - awayForm) * 0.9;
  const rawHome = clamp(42 + ratingEdge + formEdge, 16, 72);
  const rawAway = clamp(31 - ratingEdge - formEdge, 12, 68);
  const draw = clamp(100 - rawHome - rawAway, 18, 34);
  const scale = 100 / (rawHome + rawAway + draw);
  const home = Math.round(rawHome * scale);
  const away = Math.round(rawAway * scale);
  return { home, draw: 100 - home - away, away };
}

function predictionRead(fixture: Fixture, prediction: ReturnType<typeof matchPrediction>): { headline: string; detail: string } {
  const outcomes = [
    { label: fixture.home.short_name, probability: prediction.home },
    { label: 'a draw', probability: prediction.draw },
    { label: fixture.away.short_name, probability: prediction.away },
  ].sort((left, right) => right.probability - left.probability);
  const [leader, runnerUp] = outcomes;
  const gap = leader.probability - runnerUp.probability;
  const headline = leader.label === 'a draw'
    ? 'The draw leads the model'
    : `${leader.label} ${gap < 8 ? 'have a narrow edge' : 'are favored'}`;

  const preview = fixture.preview;
  const homeXiRating = preview?.home_xi_avg || 0;
  const awayXiRating = preview?.away_xi_avg || 0;
  const homeRating = homeXiRating || fixture.home.overall_team_rating || 0;
  const awayRating = awayXiRating || fixture.away.overall_team_rating || 0;
  const ratingGap = homeRating - awayRating;
  const homeForm = preview?.home_form ?? fixture.home.form ?? [];
  const awayForm = preview?.away_form ?? fixture.away.form ?? [];
  const formGap = formScore(homeForm) - formScore(awayForm);
  const ratingImpact = homeRating && awayRating ? Math.abs(ratingGap) * 1.8 : 0;
  const formImpact = homeForm.length && awayForm.length ? Math.abs(formGap) * 0.9 : 0;
  const drivers: string[] = [];

  if (ratingImpact >= 1) {
    const stronger = ratingGap > 0 ? fixture.home.short_name : fixture.away.short_name;
    const ratingLabel = homeXiRating && awayXiRating ? 'expected XI strength' : 'team strength';
    drivers.push(`${stronger} lead on ${ratingLabel} by ${Math.abs(ratingGap).toFixed(1)} points`);
  }
  if (formImpact >= 1) {
    const inForm = formGap > 0 ? fixture.home.short_name : fixture.away.short_name;
    const inFormPoints = formGap > 0 ? formScore(homeForm) : formScore(awayForm);
    const otherPoints = formGap > 0 ? formScore(awayForm) : formScore(homeForm);
    drivers.push(`${inForm} lead recent form ${inFormPoints}–${otherPoints} on points`);
  }

  const detail = drivers.length
    ? `${drivers.join('; ')}. The top outcome is only ${gap} percentage point${gap === 1 ? '' : 's'} ahead of the next.`
    : homeRating && awayRating && homeForm.length && awayForm.length
      ? 'Recent form and team strength are closely matched, leaving home advantage as the main model lean.'
      : 'There is little team-strength or recent-form detail available, so home advantage is the main model lean.';
  return { headline, detail };
}

function ProbabilityBar({ fixture }: { fixture: Fixture }) {
  const prediction = matchPrediction(fixture);
  const read = predictionRead(fixture, prediction);
  return (
    <div>
      <div className="grid grid-cols-3 gap-2 text-center">
        <div><strong className="block font-display text-[25px] text-bone">{prediction.home}%</strong><span className="text-[10px] uppercase tracking-[0.12em] text-sage">{fixture.home.short_name}</span></div>
        <div><strong className="block font-display text-[25px] text-sage">{prediction.draw}%</strong><span className="text-[10px] uppercase tracking-[0.12em] text-sage">Draw</span></div>
        <div><strong className="block font-display text-[25px] text-bone">{prediction.away}%</strong><span className="text-[10px] uppercase tracking-[0.12em] text-sage">{fixture.away.short_name}</span></div>
      </div>
      <div role="img" className="mt-3 flex h-2 overflow-hidden rounded-full bg-white/10" aria-label={`${fixture.home.short_name} win ${prediction.home}%, draw ${prediction.draw}%, ${fixture.away.short_name} win ${prediction.away}%`}>
        <span aria-hidden="true" className="bg-bone" style={{ width: `${prediction.home}%` }} />
        <span aria-hidden="true" className="bg-sage/45" style={{ width: `${prediction.draw}%` }} />
        <span aria-hidden="true" className="bg-[#69aee5]" style={{ width: `${prediction.away}%` }} />
      </div>
      <div className="mt-3 rounded-md border border-brass/15 bg-brass/[0.045] px-3 py-2.5">
        <p className="text-[10px] font-bold uppercase tracking-[0.12em] text-brass">{read.headline}</p>
        <p className="mt-1 text-[11px] leading-relaxed text-sage">{read.detail}</p>
      </div>
      <p className="mt-2 text-[10px] leading-relaxed text-sage">Model estimate from expected XI strength, recent form and home advantage.</p>
    </div>
  );
}

export function MatchPulseCard({ fixture }: { fixture: Fixture }) {
  const preview = fixture.preview;
  const rows = [
    { label: 'Expected XI', home: preview?.home_xi_avg ? preview.home_xi_avg.toFixed(1) : '—', away: preview?.away_xi_avg ? preview.away_xi_avg.toFixed(1) : '—' },
    { label: 'League position', home: preview?.home_pos ? `#${preview.home_pos}` : '—', away: preview?.away_pos ? `#${preview.away_pos}` : '—' },
    { label: 'Points', home: preview ? preview.home_pts : '—', away: preview ? preview.away_pts : '—' },
    { label: 'Goal difference', home: preview?.home_gd != null ? `${preview.home_gd >= 0 ? '+' : ''}${preview.home_gd}` : '—', away: preview?.away_gd != null ? `${preview.away_gd >= 0 ? '+' : ''}${preview.away_gd}` : '—' },
  ];
  return (
    <section className="console-card overflow-hidden lg:col-span-5">
      <div className="border-b border-white/[0.08] p-4">
        <h3 className="match-section-title"><Sparkles size={14} /> Match pulse</h3>
        <div className="mt-4"><ProbabilityBar fixture={fixture} /></div>
      </div>
      <div className="p-4">
        <div className="grid grid-cols-[1fr_auto_1fr] text-center text-[11px] font-bold text-bone">
          <span>{fixture.home.short_name}</span><span className="px-3 text-[9px] uppercase tracking-[0.14em] text-sage">Compare</span><span>{fixture.away.short_name}</span>
        </div>
        <div className="mt-2 divide-y divide-white/[0.07]">
          {rows.map((row) => (
            <div key={row.label} className="grid grid-cols-[1fr_auto_1fr] items-center py-2 text-[12px]">
              <strong className="text-center text-bone">{row.home}</strong>
              <span className="min-w-[110px] text-center text-sage">{row.label}</span>
              <strong className="text-center text-bone">{row.away}</strong>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

export function FormGuide({ fixture }: { fixture: Fixture }) {
  const preview = fixture.preview;
  return (
    <section className="console-card p-4 lg:col-span-7">
      <h3 className="match-section-title"><TrendingUp size={14} /> Form guide</h3>
      <div className="mt-4 space-y-4">
        {[
          { club: fixture.home, form: preview?.home_form ?? fixture.home.form ?? [], pos: preview?.home_pos, pts: preview?.home_pts },
          { club: fixture.away, form: preview?.away_form ?? fixture.away.form ?? [], pos: preview?.away_pos, pts: preview?.away_pts },
        ].map(({ club, form, pos, pts }) => (
          <div key={club.club_id} className="grid grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-3 rounded-md border border-white/[0.07] bg-black/15 p-3">
            <ClubCrest club={club} size={34} className="!border-0 !bg-transparent" />
            <div className="min-w-0">
              <p className="truncate text-[13px] font-bold text-bone">{club.club_name}</p>
              <div className="mt-1.5"><FormPips form={form} size="sm" /></div>
            </div>
            <div className="text-right"><strong className="block font-display text-[18px] text-bone">{pos ? `#${pos}` : '—'}</strong><span className="text-[10px] text-sage">{pts ?? '—'} pts</span></div>
          </div>
        ))}
      </div>
    </section>
  );
}

export function PlayersToWatch({ fixture, onOpen }: { fixture: Fixture; onOpen: (id: string) => void }) {
  const preview = fixture.preview;
  const players: Array<{ player?: Player | null; club: Club }> = [
    { player: preview?.home_key_player ?? preview?.home_xi?.[0], club: fixture.home },
    { player: preview?.away_key_player ?? preview?.away_xi?.[0], club: fixture.away },
  ];
  return (
    <section className="console-card p-4 lg:col-span-7">
      <h3 className="match-section-title"><Activity size={14} /> Players to watch</h3>
      <div className="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2">
        {players.map(({ player, club }) => player ? (
          <button key={player.player_id} type="button" onClick={() => onOpen(player.player_id)} className="group flex min-w-0 items-center gap-3 rounded-md border border-white/[0.08] bg-black/15 p-3 text-left hover:border-brass/35 hover:bg-white/[0.035]">
            <OvrBadge ovr={player.ovr} size="lg" />
            <span className="min-w-0 flex-1">
              <span className="block truncate font-display text-[17px] font-bold text-bone group-hover:text-brass">{player.full_name}</span>
              <span className="block truncate text-[11px] text-sage">{club.short_name} · {player.position} · {player.goals} G · {player.assists} A</span>
            </span>
            <ArrowRight size={14} className="shrink-0 text-sage group-hover:text-brass" />
          </button>
        ) : <p key={club.club_id} className="p-3 text-[12px] text-sage">Player data unavailable.</p>)}
      </div>
    </section>
  );
}

export function HeadToHeadSummary({ fixture }: { fixture: Fixture }) {
  const meetings = fixture.head_to_head ?? [];
  let homeWins = 0;
  let awayWins = 0;
  let draws = 0;
  meetings.forEach((row) => {
    if (row.home_goals === row.away_goals) { draws += 1; return; }
    const winnerId = (row.home_goals ?? 0) > (row.away_goals ?? 0) ? row.home_id : row.away_id;
    if (winnerId === fixture.home.club_id) homeWins += 1;
    else if (winnerId === fixture.away.club_id) awayWins += 1;
  });
  return (
    <section className="console-card p-4 lg:col-span-5">
      <h3 className="match-section-title"><Swords size={14} /> Head-to-head</h3>
      {meetings.length ? (
        <>
          <div className="mt-4 grid grid-cols-3 text-center">
            <div><strong className="block font-display text-[28px] text-bone">{homeWins}</strong><span className="text-[10px] text-sage">{fixture.home.short_name} wins</span></div>
            <div><strong className="block font-display text-[28px] text-sage">{draws}</strong><span className="text-[10px] text-sage">Draws</span></div>
            <div><strong className="block font-display text-[28px] text-bone">{awayWins}</strong><span className="text-[10px] text-sage">{fixture.away.short_name} wins</span></div>
          </div>
          <p className="mt-4 border-t border-white/[0.07] pt-3 text-center text-[11px] text-sage">Last {meetings.length} recorded meeting{meetings.length === 1 ? '' : 's'} in this career.</p>
        </>
      ) : <p className="mt-4 text-[12px] text-sage">No previous meetings are recorded in this career.</p>}
    </section>
  );
}
