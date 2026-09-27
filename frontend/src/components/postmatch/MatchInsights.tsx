import React, { useMemo } from 'react';
import { Crosshair, Gauge, Lightbulb, Star, TrendingUp } from 'lucide-react';
import type { Fixture, MatchEventItem, MatchPlayerRow, ShotItem, TeamMatchStats } from '../../types';
import { rgbCss } from '../../lib/format';
import { PlayerPortrait } from '../ui/ui';

function pct(home: number, away: number): number {
  const total = home + away;
  return total > 0 ? Math.round((home / total) * 100) : 50;
}

function matchRead(fixture: Fixture, home: TeamMatchStats, away: TeamMatchStats): { title: string; detail: string } {
  const homeName = fixture.home.short_name;
  const awayName = fixture.away.short_name;
  const homeXg = home.xg ?? fixture.shot_map?.total_home_xg;
  const awayXg = away.xg ?? fixture.shot_map?.total_away_xg;
  const homeWon = fixture.home_goals != null && fixture.away_goals != null && fixture.home_goals > fixture.away_goals;
  const awayWon = fixture.home_goals != null && fixture.away_goals != null && fixture.away_goals > fixture.home_goals;
  const score = fixture.home_goals != null && fixture.away_goals != null
    ? ` The score finished ${fixture.home_goals}–${fixture.away_goals}.`
    : '';

  if (homeXg != null && awayXg != null && Number.isFinite(homeXg) && Number.isFinite(awayXg)) {
    const gap = Math.abs(homeXg - awayXg);
    if (gap < 0.2) {
      return {
        title: 'Chance quality was evenly matched',
        detail: `Both sides finished close on expected goals (${homeXg.toFixed(2)}–${awayXg.toFixed(2)}).${score}`,
      };
    }

    const leader = homeXg > awayXg ? homeName : awayName;
    const scoreWinner = homeWon ? homeName : awayWon ? awayName : null;
    const title = scoreWinner && scoreWinner !== leader
      ? `${leader} created the better chances`
      : `${leader} held the chance-quality edge`;
    const resultContext = scoreWinner && scoreWinner !== leader
      ? ` ${scoreWinner} still finished ahead on the scoreboard.`
      : scoreWinner === leader
        ? ` Their xG edge was reflected in the result.`
        : fixture.home_goals != null && fixture.away_goals != null
          ? ' The match ended level despite that edge.'
          : '';
    return {
      title,
      detail: `${homeName} ${homeXg.toFixed(2)} xG · ${awayName} ${awayXg.toFixed(2)} xG.${resultContext}`,
    };
  }

  if (home.on_target !== away.on_target) {
    const leader = home.on_target > away.on_target ? homeName : awayName;
    return {
      title: `${leader} put more shots on target`,
      detail: `${homeName} ${home.on_target} · ${awayName} ${away.on_target} shots on target.${score}`,
    };
  }

  if (Math.abs(home.possession - away.possession) >= 8) {
    const leader = home.possession > away.possession ? homeName : awayName;
    return {
      title: `${leader} had more of the ball`,
      detail: `${homeName} ${home.possession}% · ${awayName} ${away.possession}% possession; shots on target were level.${score}`,
    };
  }

  const hasStats = fixture.stats != null || [home, away].some((stats) =>
    stats.shots !== 0 || stats.on_target !== 0 || stats.possession !== 50 || stats.corners !== 0 || stats.passes !== 0 || stats.fouls !== 0 || stats.xg != null,
  );
  if (!hasStats) {
    return {
      title: 'Match statistics are unavailable',
      detail: 'There is not enough recorded data to compare the sides in this report.',
    };
  }

  return {
    title: 'No clear statistical edge',
    detail: `The sides were level on shots on target and close on possession (${home.possession}%–${away.possession}%).${score}`,
  };
}

export const MatchSnapshot = React.memo(function MatchSnapshot({ fixture, home, away }: { fixture: Fixture; home: TeamMatchStats; away: TeamMatchStats }) {
  const read = matchRead(fixture, home, away);
  const homeXg = home.xg ?? fixture.shot_map?.total_home_xg;
  const awayXg = away.xg ?? fixture.shot_map?.total_away_xg;
  const cards = [
    { label: 'Expected goals', home: homeXg?.toFixed(2) ?? '—', away: awayXg?.toFixed(2) ?? '—', edge: homeXg != null && awayXg != null ? pct(homeXg, awayXg) : 50 },
    { label: 'Shots on target', home: home.on_target, away: away.on_target, edge: pct(home.on_target, away.on_target) },
    { label: 'Possession', home: `${home.possession}%`, away: `${away.possession}%`, edge: home.possession },
  ];
  return (
    <section className="grid grid-cols-1 gap-2 sm:grid-cols-3" aria-label="Match snapshot">
      {cards.map((card) => (
        <div key={card.label} className="rounded-md border border-white/[0.08] bg-black/15 p-3">
          <p className="text-center text-[9px] font-bold uppercase tracking-[0.14em] text-sage">{card.label}</p>
          <div className="mt-2 flex items-center justify-between font-display text-[20px] font-bold text-bone"><span>{card.home}</span><span>{card.away}</span></div>
          <div className="mt-2 flex h-1.5 overflow-hidden rounded-full bg-white/10"><span className="bg-bone" style={{ width: `${card.edge}%` }} /><span className="flex-1 bg-[#69aee5]" /></div>
          <div className="mt-1 flex justify-between text-[9px] text-sage"><span>{fixture.home.short_name}</span><span>{fixture.away.short_name}</span></div>
        </div>
      ))}
      <div className="flex items-start gap-3 rounded-md border border-brass/20 bg-brass/[0.06] p-3 sm:col-span-3">
        <Lightbulb size={15} className="mt-0.5 shrink-0 text-brass" aria-hidden="true" />
        <div className="min-w-0">
          <p className="text-[9px] font-bold uppercase tracking-[0.14em] text-brass">Match read</p>
          <p className="mt-1 text-[12px] font-semibold text-bone">{read.title}</p>
          <p className="mt-0.5 text-[11px] leading-relaxed text-sage">{read.detail}</p>
        </div>
      </div>
    </section>
  );
});

function buildMomentum(shots: ShotItem[], events: MatchEventItem[]): Array<{ home: number; away: number }> {
  const bins = Array.from({ length: 9 }, () => ({ home: 0, away: 0 }));
  if (shots.length) {
    shots.forEach((shot) => {
      const index = Math.min(8, Math.max(0, Math.floor(shot.minute / 10)));
      const weight = 0.25 + Math.max(0, shot.xg) * 4 + (shot.outcome === 'goal' ? 0.8 : 0);
      bins[index][shot.team] += weight;
    });
    return bins;
  }
  events.forEach((event) => {
    const index = Math.min(8, Math.max(0, Math.floor(event.minute / 10)));
    const weight = ['goal', 'penalty', 'own_goal', 'corner_goal', 'free_kick_goal'].includes(event.type) ? 2.5 : event.type === 'red' ? 1.5 : 0.4;
    bins[index][event.side] += weight;
  });
  return bins;
}

export const MatchMomentum = React.memo(function MatchMomentum({ fixture, events }: { fixture: Fixture; events: MatchEventItem[] }) {
  const bins = useMemo(() => buildMomentum(fixture.shot_map?.shots ?? [], events), [fixture.shot_map, events]);
  const maximum = Math.max(1, ...bins.flatMap((bin) => [bin.home, bin.away]));
  const hasSignal = bins.some((bin) => bin.home > 0 || bin.away > 0);
  return (
    <section className="console-card p-4">
      <div className="flex items-start justify-between gap-3">
        <div><h3 className="match-section-title"><TrendingUp size={14} /> Match momentum</h3><p className="mt-1 text-[10px] text-sage">Attacking pressure by 10-minute interval</p></div>
        <div className="flex gap-3 text-[10px] text-sage"><span className="inline-flex items-center gap-1"><i className="h-2 w-2 rounded-full bg-bone" />{fixture.home.short_name}</span><span className="inline-flex items-center gap-1"><i className="h-2 w-2 rounded-full bg-[#69aee5]" />{fixture.away.short_name}</span></div>
      </div>
      {hasSignal ? (
        <div className="mt-4">
          <div className="grid h-28 grid-cols-9 items-center gap-1 border-y border-white/[0.07] py-2">
            {bins.map((bin, index) => {
              const homeHeight = Math.max(bin.home > 0 ? 5 : 0, (bin.home / maximum) * 45);
              const awayHeight = Math.max(bin.away > 0 ? 5 : 0, (bin.away / maximum) * 45);
              return (
                <div key={index} className="relative flex h-full flex-col items-stretch justify-center">
                  <span className="mb-px self-stretch rounded-t-sm bg-bone/90" style={{ height: `${homeHeight}%` }} />
                  <span className="h-px bg-white/20" />
                  <span className="mt-px self-stretch rounded-b-sm bg-[#69aee5]/90" style={{ height: `${awayHeight}%` }} />
                </div>
              );
            })}
          </div>
          <div className="mt-1 grid grid-cols-4 text-[9px] text-sage"><span>0'</span><span className="text-center">30'</span><span className="text-center">60'</span><span className="text-right">90'</span></div>
        </div>
      ) : <p className="py-8 text-center text-[12px] text-sage">No pressure events were stored for this match.</p>}
    </section>
  );
});

function shooterName(shot: ShotItem): string {
  return shot.shooter?.full_name || 'Unknown player';
}

/**
 * Cumulative expected-goals flow, rendered from the backend-built
 * shot_map.xg_flow series (matchreport.go) rather than a rebuilt proxy.
 */
export const XGFlow = React.memo(function XGFlow({ fixture }: { fixture: Fixture }) {
  const flow = fixture.shot_map?.xg_flow ?? [];
  if (flow.length < 2) {
    return <p className="rounded-md border border-white/[0.08] bg-black/15 p-6 text-center text-[12px] text-sage">No expected-goals flow was recorded for this match.</p>;
  }
  const lastMinute = Math.max(90, flow[flow.length - 1].minute);
  const maxXg = Math.max(0.5, ...flow.map((point) => Math.max(point.home_xg, point.away_xg)));
  const width = 100;
  const height = 46;
  const x = (minute: number) => (Math.min(minute, lastMinute) / lastMinute) * width;
  const y = (value: number) => height - (value / maxXg) * (height - 4) - 2;
  const line = (key: 'home_xg' | 'away_xg') => flow.map((point, index) => `${index === 0 ? 'M' : 'L'}${x(point.minute).toFixed(2)},${y(point[key]).toFixed(2)}`).join(' ');
  const homeColor = rgbCss(fixture.home.primary_color, '#F3E6C4');
  const awayColor = rgbCss(fixture.away.primary_color, '#69AEE5');
  const homeTotal = fixture.shot_map?.total_home_xg ?? flow[flow.length - 1].home_xg;
  const awayTotal = fixture.shot_map?.total_away_xg ?? flow[flow.length - 1].away_xg;

  return (
    <section className="console-card p-4" data-xg-flow="true">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h3 className="match-section-title"><Gauge size={14} /> xG flow</h3>
          <p className="mt-1 text-[10px] text-sage">Cumulative expected goals across the 90 minutes</p>
        </div>
        <div className="flex gap-3 text-[10px] text-sage">
          <span className="inline-flex items-center gap-1"><i className="h-2 w-2 rounded-full" style={{ background: homeColor }} />{fixture.home.short_name} {homeTotal.toFixed(2)}</span>
          <span className="inline-flex items-center gap-1"><i className="h-2 w-2 rounded-full" style={{ background: awayColor }} />{fixture.away.short_name} {awayTotal.toFixed(2)}</span>
        </div>
      </div>
      <svg viewBox={`0 0 ${width} ${height}`} className="mt-3 block h-28 w-full" role="img" aria-label={`Cumulative expected goals: ${fixture.home.short_name} ${homeTotal.toFixed(2)}, ${fixture.away.short_name} ${awayTotal.toFixed(2)}`}>
        <line x1="0" y1={height - 2} x2={width} y2={height - 2} stroke="rgba(255,255,255,.15)" strokeWidth=".4" />
        <line x1={x(45)} y1="0" x2={x(45)} y2={height} stroke="rgba(255,255,255,.12)" strokeWidth=".4" strokeDasharray="1.5 1.5" />
        <path d={line('away_xg')} fill="none" stroke={awayColor} strokeWidth="1.1" strokeLinejoin="round" strokeLinecap="round" />
        <path d={line('home_xg')} fill="none" stroke={homeColor} strokeWidth="1.1" strokeLinejoin="round" strokeLinecap="round" />
      </svg>
      <div className="mt-1 grid grid-cols-4 text-[9px] text-sage"><span>0'</span><span className="text-center">30'</span><span className="text-center">60'</span><span className="text-right">{lastMinute}'</span></div>
    </section>
  );
});

export const ShotMap = React.memo(function ShotMap({ fixture }: { fixture: Fixture }) {
  const shots = fixture.shot_map?.shots ?? [];
  const homeColor = rgbCss(fixture.home.primary_color, '#F3E6C4');
  const awayColor = rgbCss(fixture.away.primary_color, '#69AEE5');
  if (!shots.length) return <p className="rounded-md border border-white/[0.08] bg-black/15 p-8 text-center text-[12px] text-sage">Shot locations were not recorded for this result.</p>;
  return (
    <section className="console-card overflow-hidden">
      <div className="flex items-center justify-between border-b border-white/[0.08] p-4">
        <div><h3 className="match-section-title"><Crosshair size={14} /> Shot map</h3><p className="mt-1 text-[10px] text-sage">Dot size reflects chance quality</p></div>
        <div className="flex items-center gap-3 text-[10px] text-sage"><span>{fixture.home.short_name} {fixture.shot_map?.total_home_xg.toFixed(2)} xG</span><span>{fixture.away.short_name} {fixture.shot_map?.total_away_xg.toFixed(2)} xG</span></div>
      </div>
      <div className="bg-[#123d2b] p-3">
        <svg viewBox="0 0 100 62" className="block w-full" role="img" aria-label="Match shot map">
          <rect x="1" y="1" width="98" height="60" rx="1" fill="none" stroke="rgba(243,230,196,.4)" strokeWidth=".6" />
          <line x1="50" y1="1" x2="50" y2="61" stroke="rgba(243,230,196,.35)" strokeWidth=".6" />
          <circle cx="50" cy="31" r="9" fill="none" stroke="rgba(243,230,196,.35)" strokeWidth=".6" />
          <rect x="1" y="14" width="17" height="34" fill="none" stroke="rgba(243,230,196,.35)" strokeWidth=".6" />
          <rect x="82" y="14" width="17" height="34" fill="none" stroke="rgba(243,230,196,.35)" strokeWidth=".6" />
          {shots.map((shot, index) => {
            const x = Math.max(3, Math.min(97, shot.x * 100));
            const y = Math.max(3, Math.min(59, shot.y * 62));
            const radius = Math.max(1.3, Math.min(4.2, 1.2 + shot.xg * 6));
            const color = shot.team === 'home' ? homeColor : awayColor;
            return (
              <g key={`${shot.minute}-${index}`}>
                {shot.outcome === 'goal' && <circle cx={x} cy={y} r={radius + 1.5} fill="none" stroke="#fff3c4" strokeWidth=".8" />}
                <circle cx={x} cy={y} r={radius} fill={shot.outcome === 'goal' ? '#fff3c4' : color} fillOpacity={shot.outcome === 'miss' ? .42 : .9} stroke={color} strokeWidth=".8" />
                <title>{`${shot.minute}' ${shooterName(shot)} · ${shot.outcome} · ${shot.xg.toFixed(2)} xG`}</title>
              </g>
            );
          })}
        </svg>
      </div>
      <div className="flex flex-wrap items-center justify-between gap-3 border-t border-white/[0.08] px-4 py-2 text-[10px] text-sage">
        <span>○ Goal &nbsp; ● Saved / missed</span><span>{shots.length} recorded shots</span>
      </div>
    </section>
  );
});

export const TopPerformers = React.memo(function TopPerformers({ fixture, rows, onOpenPlayer }: { fixture: Fixture; rows: MatchPlayerRow[]; onOpenPlayer: (id: string) => void }) {
  const ranked = [...rows].filter((row) => row.played !== false && row.rating > 0).sort((a, b) => b.rating - a.rating).slice(0, 5);
  return (
    <section className="console-card p-4">
      <h3 className="match-section-title"><Star size={14} /> Top performers</h3>
      {ranked.length ? <div className="mt-3 divide-y divide-white/[0.07]">{ranked.map((player, index) => {
        const side = fixture.home_xi.some((row) => row.player_id === player.player_id) || fixture.home_bench?.some((row) => row.player_id === player.player_id) ? 'home' : 'away';
        const club = side === 'home' ? fixture.home : fixture.away;
        return (
          <button key={player.player_id} type="button" onClick={() => onOpenPlayer(player.player_id)} className="grid w-full grid-cols-[auto_auto_minmax(0,1fr)_auto] items-center gap-2 py-2 text-left hover:bg-white/[0.03]">
            <span className="w-4 text-[10px] font-bold text-sage">{index + 1}</span>
            <PlayerPortrait player={player} size={30} />
            <span className="min-w-0"><strong className="block truncate text-[12px] text-bone">{player.full_name}</strong><span className="block truncate text-[10px] text-sage">{club.short_name} · {player.position}{player.match_goals ? ` · ${player.match_goals} G` : ''}{player.match_assists ? ` · ${player.match_assists} A` : ''}</span></span>
            <strong className="rounded bg-brass px-2 py-1 font-mono text-[12px] text-ink">{player.rating.toFixed(1)}</strong>
          </button>
        );
      })}</div> : <p className="mt-3 text-[12px] text-sage">Player ratings were not stored.</p>}
    </section>
  );
});

export const TerritoryCard = React.memo(function TerritoryCard({ fixture }: { fixture: Fixture }) {
  const heatmap = fixture.touch_heatmap;
  if (!heatmap) return null;
  const rows = [
    { label: 'Attacking third', home: heatmap.home_zones.attacking, away: heatmap.away_zones.attacking },
    { label: 'Central channel', home: heatmap.home_zones.center, away: heatmap.away_zones.center },
    { label: 'Midfield', home: heatmap.home_zones.midfield, away: heatmap.away_zones.midfield },
  ];
  return (
    <section className="console-card p-4">
      <h3 className="match-section-title"><Gauge size={14} /> Territory</h3>
      <div className="mt-3 space-y-3">{rows.map((row) => (
        <div key={row.label}><div className="mb-1 flex justify-between text-[11px]"><strong className="text-bone">{row.home}%</strong><span className="text-sage">{row.label}</span><strong className="text-bone">{row.away}%</strong></div><div className="flex h-1.5 overflow-hidden rounded-full bg-white/10"><span className="bg-bone" style={{ width: `${pct(row.home, row.away)}%` }} /><span className="flex-1 bg-[#69aee5]" /></div></div>
      ))}</div>
      <div className="mt-3 flex justify-between text-[9px] text-sage"><span>{fixture.home.short_name}</span><span>{fixture.away.short_name}</span></div>
    </section>
  );
});
