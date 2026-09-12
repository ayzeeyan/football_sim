import React, { useMemo } from 'react';
import { Eye, Zap, Lock, Star } from 'lucide-react';
import type { Fixture } from '../types';
import { cx, stripEmojis } from '../lib/format';
import { ClubCrest, FormPips } from './ui/ui';
import { soundManager } from '../audio/webAudio';

interface ScorerLine {
  minute: number;
  label: string;
  kind: 'goal' | 'penalty' | 'own_goal' | 'yellow' | 'red';
}

function sideLines(f: Fixture, side: 'home' | 'away'): ScorerLine[] {
  // Labels are built from structured fields only — never from the backend
  // Display string, which embeds verbose text ("Yellow Card: …", assists)
  // that would duplicate names and leak specifics into the feed.
  const minuteOf = (m: number) => `${m}'`;
  const nameOf = (id: string | undefined, fallback: string): string => {
    if (fallback && fallback.trim() !== '') return fallback;
    if (!id) return 'Unknown';
    const pools = [f.home_xi, f.away_xi, f.home_bench ?? [], f.away_bench ?? []];
    for (const pool of pools) {
      const hit = pool.find((r) => r.player_id === id);
      if (hit && hit.full_name) return hit.full_name;
    }
    return 'Unknown';
  };
  const isGoalKind = (t: string) =>
    t === 'goal' || t === 'penalty' || t === 'corner_goal' || t === 'free_kick_goal';
  const goals = new Map<string, { name: string; minutes: string[]; first: number; kind: 'goal' | 'penalty' | 'own_goal' }>();
  const cards: ScorerLine[] = [];
  const sentOff = new Set<string>();
  for (const e of f.events) {
    const pId = e.player?.player_id || e.player_id;
    if (e.type === 'red' && e.sent_off && pId) sentOff.add(pId);
  }
  for (const e of f.events) {
    if (isGoalKind(e.type)) {
      if (e.disallowed) continue;
      let eventSide = e.side;
      if (e.club_id) {
        eventSide = e.club_id === f.home.club_id ? 'home' : 'away';
      }
      if (eventSide !== side) continue;
      const pId = e.scorer?.player_id || e.player_id;
      const pName = e.scorer?.full_name || e.player_name || '';
      if (!pId && !pName) continue;
      const scorerKey = pId || pName;
      const kind = e.type === 'penalty' ? 'penalty' : 'goal';
      const g = goals.get(scorerKey) ?? { name: nameOf(pId, pName), minutes: [], first: e.minute, kind };
      g.minutes.push(minuteOf(e.minute));
      if (e.minute < g.first) g.first = e.minute;
      goals.set(scorerKey, g);
    } else if (e.type === 'own_goal') {
      if (e.disallowed) continue;
      // Own goals read under the benefiting side, red text, tagged (og)
      const benefiting = e.beneficiary ?? (e.side === 'home' ? 'away' : 'home');
      if (benefiting !== side) continue;
      const pId = e.scorer?.player_id || e.player_id;
      const pName = e.scorer?.full_name || e.player_name || '';
      const key = `og-${pId || pName}-${e.seq}`;
      goals.set(key, { name: nameOf(pId, pName), minutes: [minuteOf(e.minute)], first: e.minute, kind: 'own_goal' });
    } else if (e.type === 'yellow' || e.type === 'red') {
      let eventSide = e.side;
      if (e.club_id) {
        eventSide = e.club_id === f.home.club_id ? 'home' : 'away';
      }
      if (eventSide !== side) continue;
      const pId = e.player?.player_id || e.player_id;
      const pName = e.player?.full_name || e.player_name || '';
      // A second yellow is folded into the sending-off line
      if (pId && e.type === 'yellow' && sentOff.has(pId)) continue;
      cards.push({ minute: e.minute, label: `${nameOf(pId, pName)} ${minuteOf(e.minute)}`, kind: e.type });
    }
  }
  const lines: ScorerLine[] = [...goals.values()].map((g) => ({
    minute: g.first,
    label: `${g.name} ${g.minutes.join(', ')}${g.kind === 'penalty' ? ' (pen)' : ''}${g.kind === 'own_goal' ? ' (og)' : ''}`,
    kind: g.kind,
  }));
  return [...lines, ...cards].sort((a, b) => a.minute - b.minute);
}

export function fixtureCompetitionLabel(f: Fixture): string {
  if (f.competition === 'ucl') {
    const leg = f.leg ? ` · Leg ${f.leg}` : '';
    return `Champions Cup · ${f.stage}${leg} · MW ${f.matchweek}`;
  }
  if (f.competition === 'super-cup') {
    return `Super Cup · ${f.stage} · MW ${f.matchweek}`;
  }
  const derby = f.derby ? ` · ${f.derby}` : '';
  return `${f.stage || 'Super League'} · MW ${f.matchweek}${derby}`;
}

export function getWeatherDetails(weather?: string | null): { icon: string; label: string } | null {
  if (!weather) return null;
  const w = weather.toLowerCase();
  if (w.includes('rain') || w.includes('wet') || w.includes('shower')) return { icon: '🌧️', label: 'Rainy' };
  if (w.includes('snow') || w.includes('blizzard') || w.includes('freez') || w.includes('ice')) return { icon: '❄️', label: 'Snow' };
  if (w.includes('wind') || w.includes('gale') || w.includes('breeze') || w.includes('gust')) return { icon: '💨', label: 'Windy' };
  if (w.includes('sun') || w.includes('clear')) return { icon: '☀️', label: 'Sunny' };
  if (w.includes('cloud') || w.includes('overcast')) return { icon: '⛅', label: 'Overcast' };
  return { icon: '⛅', label: weather };
}

interface MatchCardProps {
  fixture: Fixture;
  onWatch: (f: Fixture) => void;
  onSimulate: (f: Fixture) => void;
  onOpen: (f: Fixture) => void;
  busyId: string | null;
  eyebrow?: string;
}

export const MatchCard: React.FC<MatchCardProps> = ({ fixture: f, onWatch, onSimulate, onOpen, busyId, eyebrow }) => {
  const finished = f.status === 'finished';
  const busy = busyId === f.id;
  const homeLines = useMemo(() => (finished ? sideLines(f, 'home') : []), [f, finished]);
  const awayLines = useMemo(() => (finished ? sideLines(f, 'away') : []), [f, finished]);

  const openReport = () => {
    soundManager.playClick();
    onOpen(f);
  };

  const night = f.night;
  const european = f.competition === 'ucl' || f.competition === 'super-cup';
  const isCuLeg2 = f.competition === 'ucl' && f.leg === 2;
  const greyed = night?.group_status === 'qualified' || night?.group_status === 'eliminated';
  const weatherInfo = getWeatherDetails(f.weather);
  const derbyTitle = f.derby_name || (f.is_derby && f.derby ? f.derby : null);
  const derbyHeat = typeof f.derby_heat === 'number' ? f.derby_heat : 50;
  const isHighHeat = f.is_high_heat_derby || derbyHeat >= 70;
  const shellClass = cx(
    'panel-tight p-5 transition-colors',
    european && 'bg-[#101814] border-brass/30',
    greyed && 'opacity-80',
    finished ? 'hover:border-sage/40' : 'hover:border-sage/30',
  );

  const body = (
    <>
      <div className="flex items-center justify-between gap-2">
        <p className={cx('eyebrow', european && '!text-brass')}>{eyebrow ?? fixtureCompetitionLabel(f)}</p>
        <span className="flex items-center gap-1.5 flex-wrap justify-end">
          {weatherInfo && (
            <span
              className="px-2 py-0.5 rounded-md border border-line bg-cardLight/80 text-bone/90 text-[10px] font-mono inline-flex items-center gap-1 shadow-sm"
              title={`Conditions: ${weatherInfo.label}`}
            >
              <span>{weatherInfo.icon}</span>
              <span>{weatherInfo.label}</span>
            </span>
          )}
          {derbyTitle && (
            <span
              className={cx(
                'px-2 py-0.5 rounded-md border font-semibold text-[10px] uppercase tracking-[0.08em] inline-flex items-center gap-1 transition-all',
                isHighHeat
                  ? 'border-ember/70 bg-ember/15 text-ember animate-pulse shadow-[0_0_10px_rgba(224,86,36,0.25)]'
                  : 'border-ember/40 bg-ember/10 text-ember',
              )}
            >
              <span>🔥</span>
              <span>{derbyTitle} · {derbyHeat}° Heat</span>
            </span>
          )}
          {night?.badge && (
            <span className="px-2 py-0.5 border border-brass/40 bg-brass/10 text-brass font-semibold text-[10px] uppercase tracking-[0.08em]">
              {night.badge}
            </span>
          )}
          {night?.aggregate && !finished && (
            <span className="px-2 py-0.5 border border-ember/40 bg-ember/10 text-ember font-semibold text-[10px]">
              {night.aggregate}
            </span>
          )}
          {night?.group_status === 'must_win' && (
            <span className="px-2 py-0.5 border border-ember/40 bg-ember/10 text-ember font-semibold text-[10px] uppercase tracking-[0.08em]">
              Must-win
            </span>
          )}
          {night?.group_status === 'qualified' && (
            <span className="px-2 py-0.5 border border-sage/40 text-sage font-semibold text-[10px] uppercase tracking-[0.08em]">
              Through
            </span>
          )}
          <span className={cx('font-mono text-[12px] font-semibold', finished ? 'text-bone/80' : 'text-sage')}>
            {finished ? 'Full-time' : 'Scheduled'}
          </span>
        </span>
      </div>
      {night?.story && !finished && (
        <p className="mt-2 text-[13px] text-bone/80 leading-snug">{night.story}</p>
      )}
      {isCuLeg2 && !finished && (
        <div className="mt-2 text-center space-y-0.5">
          {night?.leg1_label && (
            <p className="text-[12px] font-mono text-[#A9CBDD]">{night.leg1_label}</p>
          )}
          <p className="text-[11.5px] text-sage">Away goals don’t count — extra time if level.</p>
        </div>
      )}

      <div className="mt-4 flex items-center justify-between gap-3">
        <div className="flex flex-col items-center gap-2 flex-1 min-w-0">
          <ClubCrest club={f.home} size={64} />
          <span className="font-semibold text-[15px] text-bone text-center leading-tight truncate w-full">{f.home.club_name}</span>
        </div>

        <div className="score-display text-[44px] leading-none text-bone shrink-0 px-1 text-center">
          {finished ? (
            <>
              {f.home_goals}<span className="text-sage mx-3 text-[28px]">–</span>{f.away_goals}
              {f.decided_by === 'penalties' && f.penalties && (
                <p className="font-sans font-semibold text-[11px] text-brass mt-1 tracking-normal">
                  {f.penalties[0]}–{f.penalties[1]} pens
                </p>
              )}
              {f.decided_by === 'extra_time' && (
                <p className="font-sans font-semibold text-[11px] text-sage mt-1 tracking-normal">AET</p>
              )}
            </>
          ) : (
            <span className="text-sage/50 text-[30px] font-sans font-semibold">vs</span>
          )}
        </div>

        <div className="flex flex-col items-center gap-2 flex-1 min-w-0">
          <ClubCrest club={f.away} size={64} />
          <span className="font-semibold text-[15px] text-bone text-center leading-tight truncate w-full">{f.away.club_name}</span>
        </div>
      </div>

      {finished ? (
        <div>
          <p className="text-center text-[12px] text-sage font-medium mt-3">
            {f.decided_by === 'penalties' ? 'After extra time · penalties' : f.decided_by === 'extra_time' ? 'After extra time' : 'Full-time'}
            {typeof f.ht_home === 'number' && typeof f.ht_away === 'number' ? ` · HT ${f.ht_home}-${f.ht_away}` : ''}
          </p>
          {isCuLeg2 && (
            <div className="mt-1.5 text-center space-y-0.5">
              {night?.leg1_label && (
                <p className="text-[11.5px] font-mono text-[#A9CBDD]">{night.leg1_label}</p>
              )}
              <p className="text-[11px] text-sage">Away goals don’t count — extra time if level.</p>
            </div>
          )}
          {(homeLines.length > 0 || awayLines.length > 0) && (
            <div className="grid grid-cols-2 gap-3 mt-3 pt-3 border-t border-line">
              <div className="space-y-1">
                {homeLines.map((l, i) => (
                  <p key={i} className={cx('text-[12.5px] leading-snug flex items-center gap-1.5', (l.kind === 'red' || l.kind === 'own_goal') ? 'text-ember font-medium' : 'text-bone/75')}>
                    {l.kind === 'yellow' && (
                      <span className="w-2.5 h-3 rounded-[2px] shrink-0 inline-block bg-brass" />
                    )}
                    {l.kind === 'red' && (
                      <span className="w-2.5 h-3 rounded-[2px] shrink-0 inline-block bg-ember" />
                    )}
                    <span>{stripEmojis(l.label)}</span>
                  </p>
                ))}
              </div>
              <div className="space-y-1 text-right">
                {awayLines.map((l, i) => (
                  <p key={i} className={cx('text-[12.5px] leading-snug flex items-center justify-end gap-1.5', (l.kind === 'red' || l.kind === 'own_goal') ? 'text-ember font-medium' : 'text-bone/75')}>
                    <span>{stripEmojis(l.label)}</span>
                    {l.kind === 'yellow' && (
                      <span className="w-2.5 h-3 rounded-[2px] shrink-0 inline-block bg-brass" />
                    )}
                    {l.kind === 'red' && (
                      <span className="w-2.5 h-3 rounded-[2px] shrink-0 inline-block bg-ember" />
                    )}
                  </p>
                ))}
              </div>
            </div>
          )}
          {f.motm && (
            <p className="mt-3 flex items-center justify-center gap-1.5 text-[12px] font-semibold text-brass">
              <Star size={12} fill="currentColor" aria-hidden="true" /> Man of the match: {f.motm.full_name} {f.motm.rating.toFixed(1)}
            </p>
          )}
          {f.head_to_head && f.head_to_head.length > 0 && (
            <p className="text-center text-[11px] text-sage font-mono mt-2">
              Last meeting: {f.head_to_head[0].home_goals}–{f.head_to_head[0].away_goals} · MW {f.head_to_head[0].matchweek}
            </p>
          )}
          <p className="text-center text-[12px] text-[#A9CBDD] font-semibold mt-2">Open report for timeline, lineups and stats</p>
        </div>
      ) : (
        <div className="mt-4 pt-3 border-t border-line space-y-3">
          {f.preview?.kickoff_note && (
            <p className="text-center text-[13px] text-bone/80 leading-snug">{f.preview.kickoff_note}</p>
          )}
          <div className="grid grid-cols-2 gap-3">
            <div className="flex justify-center">
              <FormPips form={f.preview?.home_form ?? f.home.form} size="sm" />
            </div>
            <div className="flex justify-center">
              <FormPips form={f.preview?.away_form ?? f.away.form} size="sm" />
            </div>
          </div>
          {((f.preview?.home_missing.length ?? 0) > 0 || (f.preview?.away_missing.length ?? 0) > 0) && (
            <p className="text-center text-[12px] text-ember">
              Missing
              {f.preview!.home_missing.length > 0 ? ` · ${f.home.short_name} ${f.preview!.home_missing.map((p) => p.full_name.split(' ').slice(-1)[0]).join(', ')}` : ''}
              {f.preview!.away_missing.length > 0 ? ` · ${f.away.short_name} ${f.preview!.away_missing.map((p) => p.full_name.split(' ').slice(-1)[0]).join(', ')}` : ''}
            </p>
          )}
          {f.head_to_head && f.head_to_head.length > 0 && (
            <p className="text-center text-[12px] text-sage font-mono">
              Last meeting · MW {f.head_to_head[0].matchweek}: {f.head_to_head[0].home_goals}–{f.head_to_head[0].away_goals}
            </p>
          )}
          <div className="flex items-center gap-2">
          <button
            onClick={() => {
              soundManager.playClick();
              onOpen(f);
            }}
            className="flex-1 px-3 py-2.5 bg-cardLight hover:bg-cardHover text-bone border border-line text-[14px] font-semibold transition-colors"
          >
            Preview
          </button>
          <button
            onClick={() => {
              soundManager.playClick();
              onWatch(f);
            }}
            className="flex-1 px-3 py-2.5 bg-bone hover:bg-[#fff6dc] text-ink text-[14px] font-semibold transition-colors inline-flex items-center justify-center gap-1.5"
          >
            <Eye size={14} aria-hidden="true" /> Watch live
          </button>
          <button
            onClick={() => onSimulate(f)}
            disabled={busy}
            className="flex-1 px-3 py-2.5 bg-cardLight hover:bg-cardHover text-bone border border-line text-[14px] font-semibold transition-colors inline-flex items-center justify-center gap-1.5 disabled:opacity-50"
          >
            <Zap size={14} aria-hidden="true" /> {busy ? 'Simulating…' : 'Simulate'}
          </button>
          </div>
        </div>
      )}

      {finished && (
        <p className="mt-3 pt-2 border-t border-line flex items-center justify-center gap-1.5 text-[11px] font-mono text-sage/70 uppercase tracking-[0.1em]">
          <Lock size={11} aria-hidden="true" /> Result stands · {f.method === 'live' ? 'played live' : 'simulated'}
        </p>
      )}
    </>
  );

  if (finished) {
    return (
      <button
        type="button"
        className={cx(shellClass, 'w-full text-left')}
        onClick={openReport}
        aria-label={`Open ${f.home.club_name} against ${f.away.club_name} report`}
      >
        {body}
      </button>
    );
  }

  return <article className={shellClass}>{body}</article>;
};
