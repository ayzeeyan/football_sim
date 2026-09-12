import React, { useEffect, useMemo, useState } from 'react';
import { Volleyball, ArrowDownUp, Flag, Star, ArrowLeftRight, Target, Flame, Play, Pause, SkipBack, SkipForward, Film } from 'lucide-react';
import type { Fixture, MatchEventItem, MatchPlayerRow } from '../types';
import { cx, stripEmojis, rgbCss } from '../lib/format';
import { ClubCrest, Modal, ModalHeader } from './ui/ui';
import { PlayerNameButton } from './PlayerSheet';
import { soundManager } from '../audio/webAudio';

// XI order from the backend is always [GK, DEF x4, MID x3, FWD x3] (4-3-3).
// Index maps 1:1 onto FULL_PITCH_SLOTS — never reorder.
// viewBox 0 0 100 142. Row centres: GK 17, DEF 44, MID 75, FWD 106
// (gaps 27 / 31 / 31). Dot r=5.4; bubbles to −6.6; name baseline +12.4.
const FULL_PITCH_SLOTS: Array<[number, number]> = [
  [50, 17],
  [13, 44], [37.5, 44], [62.5, 44], [87, 44],
  [22, 75], [50, 75], [78, 75],
  [22, 106], [50, 106], [78, 106],
];

/** Max name-lane width in viewBox units, per slot (centred on the dot). */
const NAME_LANE_W = [28, 18, 18, 18, 18, 20, 20, 20, 20, 20, 20];

const DOT_R = 5.4;
const BUBBLE_R = 2.35;
const BUBBLE_OFF = 4.2;
const PILL_W = 11.6;
const PILL_H = 5.0;
const PILL_TOP = DOT_R - 1.8; // docks onto the bottom rim
const NAME_Y = PILL_TOP + PILL_H + 3.4; // 12.4 — dedicated clear lane

function initials(name: string): string {
  const parts = name.split(' ').filter(Boolean);
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
}

function surnameOf(full: string): string {
  const parts = full.trim().split(/\s+/).filter(Boolean);
  return parts[parts.length - 1] ?? full;
}

/** Inter at ~3.05px ≈ 0.56em per glyph in viewBox units. */
function fitSurname(name: string, maxWidth: number, fontSize = 3.05): string {
  const charW = fontSize * 0.56;
  const maxChars = Math.max(3, Math.floor(maxWidth / charW));
  if (name.length <= maxChars) return name;
  return `${name.slice(0, Math.max(2, maxChars - 1))}…`;
}

const TimelineRow: React.FC<{ event: MatchEventItem; fixture: Fixture }> = ({ event, fixture }) => {
  // Timestamps come from the structured minute — never from Display, which
  // embeds verbose backend text ("Yellow Card: …", assists) on instant sims.
  const stamp = `${event.minute}'`;
  const eventPlayerId = event.player_id || event.scorer?.player_id || event.player?.player_id;
  const eventPlayerName = event.player_name || event.scorer?.full_name || event.player?.full_name || 'Unknown';
  const eventPlayerPosition = event.scorer?.position || event.player?.position;
  if (event.type === 'goal' || event.type === 'penalty' || event.type === 'own_goal') {
    const isOg = event.type === 'own_goal';
    const beneficiary = isOg ? (event.beneficiary ?? (event.side === 'home' ? 'away' : 'home')) : event.side;
    const club = beneficiary === 'home' ? fixture.home : fixture.away;
    const title = isOg ? 'OWN GOAL' : event.type === 'penalty' ? 'PENALTY SCORED' : 'GOAL';
    return (
      <div className="rounded-xl overflow-hidden border border-brass/45">
        <div className="bg-brass px-4 py-3 text-center text-ink">
          <Volleyball size={18} className="mx-auto" />
          <p className="font-display font-semibold text-[17px] tracking-wide mt-1">{title}</p>
          <p className="font-mono font-bold text-[13px]">{stamp}</p>
        </div>
        <div className="bg-cardLight px-4 py-2 text-center font-mono text-[12.5px] font-semibold text-bone/85">
          {fixture.home.club_name} {event.home_score} – {event.away_score} {fixture.away.club_name}
        </div>
        <div className="bg-ink/60 px-4 py-3 flex items-center justify-between gap-3">
          <div className="min-w-0">
            <p className="font-semibold text-[15px] text-bone truncate">
              <PlayerNameButton playerId={eventPlayerId}>{eventPlayerName}</PlayerNameButton>
            </p>
            <p className="text-[12.5px] text-sage mt-0.5">
              {club.club_name}{eventPlayerPosition ? ` · ${eventPlayerPosition}` : ''}
              {!isOg && event.assister && <span className="block">Assist: {event.assister.full_name}</span>}
              {isOg && <span className="block">Turned into his own net</span>}
            </p>
          </div>
          <ClubCrest club={club} size={40} />
        </div>
      </div>
    );
  }

  if (event.type === 'sub') {
    const club = event.side === 'home' ? fixture.home : fixture.away;
    return (
      <div className="rounded-xl border border-line bg-ink/50 px-4 py-3">
        <div className="flex items-center justify-between gap-2">
          <p className="eyebrow !text-bone/85 flex items-center gap-2">
            <ArrowLeftRight size={12} className="text-sage" /> Substitution
          </p>
          <span className="font-mono text-[12px] font-semibold text-sage">{stamp}</span>
        </div>
        <div className="mt-2 grid grid-cols-[1fr_auto_1fr] items-center gap-2">
          <div className="min-w-0">
            <p className="text-[11px] font-mono uppercase tracking-[0.1em] text-sage">Off</p>
            <p className="font-semibold text-[14px] text-bone truncate">
              <PlayerNameButton playerId={event.player_out?.player_id}>{event.player_out?.full_name ?? '—'}</PlayerNameButton>
            </p>
            <p className="text-[12px] text-sage">{event.player_out?.position}</p>
          </div>
          <ArrowLeftRight size={14} className="text-brass shrink-0" />
          <div className="min-w-0 text-right">
            <p className="text-[11px] font-mono uppercase tracking-[0.1em] text-sage">On</p>
            <p className="font-semibold text-[14px] text-bone truncate">
              <PlayerNameButton playerId={event.player_in?.player_id}>{event.player_in?.full_name ?? '—'}</PlayerNameButton>
            </p>
            <p className="text-[12px] text-sage">{event.player_in?.position}</p>
          </div>
        </div>
        <p className="mt-2 text-[12px] text-sage">{club.club_name}</p>
      </div>
    );
  }

  if (event.type === 'penalty_miss') {
    const club = event.side === 'home' ? fixture.home : fixture.away;
    return (
      <div className="rounded-xl border border-line bg-ink/50 px-4 py-3">
        <div className="flex items-center justify-between gap-2">
          <p className="eyebrow !text-bone/85">Penalty missed</p>
          <span className="font-mono text-[12px] font-semibold text-sage">{stamp}</span>
        </div>
        <div className="mt-1.5">
          <p className="font-semibold text-[14px] text-bone truncate">
            <PlayerNameButton playerId={eventPlayerId}>{eventPlayerName}</PlayerNameButton>
          </p>
          <p className="text-[12px] text-sage">{club.club_name} · dragged it wide</p>
        </div>
      </div>
    );
  }

  if (event.type === 'var_review') {
    const club = event.side === 'home' ? fixture.home : fixture.away;
    const stood = event.outcome !== 'goal_disallowed' && event.decision !== 'goal_disallowed';
    return (
      <div className="rounded-xl border border-line bg-ink/50 px-4 py-3">
        <div className="flex items-center justify-between gap-2">
          <p className="eyebrow !text-bone/85 flex items-center gap-2">
            <Flag size={12} className="text-sage" />
            VAR check
          </p>
          <span className="font-mono text-[12px] font-semibold text-sage">{stamp}</span>
        </div>
        <p className="mt-1.5 text-[13px] text-sage">
          {club.club_name} · {stood ? 'goal stands' : `goal disallowed${event.reason ? ` (${event.reason})` : ''}`}
        </p>
      </div>
    );
  }

  const red = event.type === 'red';
  let club = event.side === 'home' ? fixture.home : fixture.away;
  if (event.club_id) {
    if (event.club_id === fixture.home.club_id) club = fixture.home;
    else if (event.club_id === fixture.away.club_id) club = fixture.away;
  }

  const playerId = event.player_id || event.player?.player_id;
  let playerName = event.player_name || event.player?.full_name;
  let playerPos = event.player?.position;

  if (!playerName || !playerPos) {
    const pools = [fixture.home_xi, fixture.away_xi, fixture.home_bench ?? [], fixture.away_bench ?? []];
    if (playerId) {
      for (const pool of pools) {
        const hit = pool.find((r) => r.player_id === playerId);
        if (hit) {
          if (!playerName) playerName = hit.full_name;
          if (!playerPos) playerPos = hit.position;
          break;
        }
      }
    }
  }
  const cleanPlayerName = stripEmojis(playerName ?? 'Unknown');

  const redDetail = red
    ? event.detail === 'second_yellow'
      ? 'second yellow'
      : event.detail === 'straight_red'
        ? 'straight red'
        : event.sent_off
          ? 'sent off'
          : ''
    : '';
  return (
    <div className="rounded-xl border border-line bg-ink/50 px-4 py-3">
      <div className="flex items-center justify-between gap-2">
        <p className="eyebrow !text-bone/85 flex items-center gap-2">
          <ArrowDownUp size={12} className="text-sage" />
          {red ? 'Red card' : 'Yellow card'}
          {event.sent_off && <span className="text-ember">· sent off</span>}
        </p>
        <span className="font-mono text-[12px] font-semibold text-sage">{stamp}</span>
      </div>
      <div className="mt-2 flex items-center gap-2.5">
        <span className={cx('w-3 h-4 rounded-[3px] shrink-0 inline-block', red ? 'bg-ember' : 'bg-brass')} />
        <div className="min-w-0">
          <p className="font-semibold text-[14px] text-bone truncate">
            <PlayerNameButton playerId={playerId}>{cleanPlayerName}</PlayerNameButton>
          </p>
          <p className="text-[12px] text-sage">{club.club_name}{playerPos ? ` · ${playerPos}` : ''}{redDetail ? ` · ${redDetail}` : ''}</p>
        </div>
      </div>
    </div>
  );
};

const PitchDot: React.FC<{
  player: MatchPlayerRow;
  x: number;
  y: number;
  mode: 'rating' | 'age';
  nameMaxWidth: number;
  isMotm: boolean;
}> = ({ player, x, y, mode, nameMaxWidth, isMotm }) => {
  const hotPill = mode === 'rating' && player.rating >= 7.5;
  const shown = fitSurname(surnameOf(player.full_name), nameMaxWidth);
  return (
    <g transform={`translate(${x},${y})`}>
      <title>{player.full_name}</title>
      {isMotm && (
        <g aria-hidden="true">
          <circle r={9.4} fill="none" stroke="#C7A23A" strokeWidth={0.7} opacity={0.22} />
          <circle r={7.8} fill="none" stroke="#C7A23A" strokeWidth={1.6} opacity={0.38} />
        </g>
      )}
      <circle
        r={DOT_R}
        fill="#121714"
        stroke={player.is_wk ? '#C7A23A' : 'rgba(234,228,214,0.4)'}
        strokeWidth={player.is_wk ? 1.35 : 0.9}
      />
      <text
        y={0.35}
        textAnchor="middle"
        dominantBaseline="middle"
        fill="#EAE4D6"
        fontSize={3.7}
        fontWeight={700}
        fontFamily="Inter, sans-serif"
      >
        {initials(player.full_name)}
      </text>
      {player.match_goals > 0 && (
        <g transform={`translate(${BUBBLE_OFF},${-BUBBLE_OFF})`}>
          <circle r={BUBBLE_R} fill="#C7A23A" />
          <text
            y={0.55}
            textAnchor="middle"
            dominantBaseline="middle"
            fill="#0B0E0C"
            fontSize={3.1}
            fontWeight={800}
            fontFamily="'IBM Plex Mono', monospace"
          >
            {player.match_goals}
          </text>
        </g>
      )}
      {player.card && (
        <rect
          x={-DOT_R - 1.0}
          y={-DOT_R - 1.4}
          width={2.2}
          height={3.2}
          rx={0.5}
          fill={player.card === 'red' ? '#BE5A38' : '#C7A23A'}
        />
      )}
      {player.off_minute != null && (
        <g transform="translate(0, -7.2)">
          <rect
            x={-3.6}
            y={-1.8}
            width={7.2}
            height={3.6}
            rx={1.0}
            fill="#121714"
            stroke="#BE5A38"
            strokeWidth={0.5}
          />
          <text
            y={0.35}
            textAnchor="middle"
            dominantBaseline="middle"
            fill="#EAE4D6"
            fontSize={2.5}
            fontWeight={800}
            fontFamily="'IBM Plex Mono', monospace"
          >
            ↓{player.off_minute}'
          </text>
        </g>
      )}
      {player.match_assists > 0 && (
        <g transform={`translate(${PILL_W / 2 + BUBBLE_R + 0.2},${PILL_TOP + PILL_H / 2})`}>
          <circle r={BUBBLE_R} fill="#8AB4C8" />
          <text
            y={0.55}
            textAnchor="middle"
            dominantBaseline="middle"
            fill="#0B0E0C"
            fontSize={3.0}
            fontWeight={800}
            fontFamily="'IBM Plex Mono', monospace"
          >
            A
          </text>
        </g>
      )}
      {(player.match_og > 0 || player.match_pen_miss > 0) && (
        <g transform={`translate(${-(PILL_W / 2 + BUBBLE_R + 0.2)},${PILL_TOP + PILL_H / 2})`}>
          <circle
            r={BUBBLE_R}
            fill={player.match_og > 0 ? '#BE5A38' : 'none'}
            stroke={player.match_og > 0 ? 'none' : '#EAE4D6'}
            strokeWidth={0.7}
          />
          <text
            y={0.55}
            textAnchor="middle"
            dominantBaseline="middle"
            fill={player.match_og > 0 ? '#0B0E0C' : '#EAE4D6'}
            fontSize={player.match_og > 0 ? 2.4 : 3.1}
            fontWeight={800}
            fontFamily="'IBM Plex Mono', monospace"
          >
            {player.match_og > 0 ? 'OG' : 'x'}
          </text>
        </g>
      )}
      <g transform={`translate(0,${PILL_TOP + PILL_H / 2})`}>
        <rect
          x={-PILL_W / 2}
          y={-PILL_H / 2}
          width={PILL_W}
          height={PILL_H}
          rx={PILL_H / 2}
          fill={hotPill ? '#C7A23A' : '#263026'}
        />
        <text
          y={0.45}
          textAnchor="middle"
          dominantBaseline="middle"
          fill={hotPill ? '#0B0E0C' : '#EAE4D6'}
          fontSize={3.4}
          fontWeight={700}
          fontFamily="'IBM Plex Mono', monospace"
        >
          {mode === 'rating' ? player.rating.toFixed(1) : player.age}
        </text>
      </g>
      <clipPath id={`nl-${player.player_id}`}>
        <rect x={-nameMaxWidth / 2} y={NAME_Y - 3.1} width={nameMaxWidth} height={5} />
      </clipPath>
      <text
        y={NAME_Y}
        textAnchor="middle"
        fill="rgba(234,228,214,0.92)"
        fontSize={3.05}
        fontFamily="Inter, sans-serif"
        clipPath={`url(#nl-${player.player_id})`}
      >
        {shown}
      </text>
    </g>
  );
};

function BenchList({ rows }: { rows: MatchPlayerRow[] }) {
  if (!rows.length) return null;
  return (
    <div className="mt-3 space-y-1">
      <p className="text-[11px] font-mono uppercase tracking-[0.12em] text-sage">Substitutes</p>
      {rows.map((p) => (
        <div key={p.player_id} className="flex items-center justify-between gap-2 text-[12.5px] py-1 border-b border-line/60 last:border-0">
          <span className={cx('truncate font-semibold', p.played === false ? 'text-sage' : 'text-bone')}>
            <PlayerNameButton playerId={p.player_id}>{p.full_name}</PlayerNameButton>
            {p.on_minute != null && <span className="ml-1.5 font-mono font-normal text-brass">↑{p.on_minute}'</span>}
            {p.card === 'yellow' && <span className="ml-1.5 text-brass">Y</span>}
            {p.card === 'red' && <span className="ml-1.5 text-ember">R</span>}
          </span>
          <span className="font-mono text-sage shrink-0">
            {p.played === false ? 'unused' : `${p.minutes}' · ${p.rating != null ? p.rating.toFixed(1) : '—'}`}
          </span>
        </div>
      ))}
    </div>
  );
}

const LineupsPitch: React.FC<{ fixture: Fixture; mode: 'rating' | 'age' }> = ({ fixture, mode }) => {
  const motmId = fixture.motm?.player_id ?? null;
  return (
    <div className="space-y-3">
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {(
          [
            { club: fixture.home, xi: fixture.home_xi, bench: fixture.home_bench ?? [] },
            { club: fixture.away, xi: fixture.away_xi, bench: fixture.away_bench ?? [] },
          ] as const
        ).map(({ club, xi, bench }) => (
          <div key={club.club_id} className="min-w-0">
            <div className="flex items-center justify-between gap-2 mb-2">
              <span className="flex items-center gap-2 min-w-0">
                <ClubCrest club={club} size={30} />
                <span className="font-semibold text-[15px] text-bone truncate">{club.club_name}</span>
              </span>
              <span className="font-mono text-[11px] text-sage border border-line rounded-md px-2 py-0.5">4-3-3</span>
            </div>
            <div className="overflow-x-auto">
              <svg
                viewBox="0 0 100 142"
                className="w-full min-w-[280px] rounded-xl border border-line"
                style={{ background: 'linear-gradient(to bottom, #12251A, #0E1A12)' }}
                role="img"
                aria-label={`${club.club_name} lineup`}
              >
                <rect x={1.5} y={1.5} width={97} height={139} fill="none" stroke="rgba(234,228,214,0.5)" strokeWidth={0.55} />
                <line x1={1.5} y1={71} x2={98.5} y2={71} stroke="rgba(234,228,214,0.35)" strokeWidth={0.55} />
                <circle cx={50} cy={71} r={10} fill="none" stroke="rgba(234,228,214,0.35)" strokeWidth={0.55} />
                <circle cx={50} cy={71} r={0.8} fill="rgba(234,228,214,0.5)" />
                <rect x={24} y={1.5} width={52} height={16} fill="none" stroke="rgba(234,228,214,0.45)" strokeWidth={0.55} />
                <rect x={34} y={1.5} width={32} height={7} fill="none" stroke="rgba(234,228,214,0.4)" strokeWidth={0.5} />
                <rect x={24} y={124.5} width={52} height={16} fill="none" stroke="rgba(234,228,214,0.35)" strokeWidth={0.55} />
                <rect x={34} y={133.5} width={32} height={7} fill="none" stroke="rgba(234,228,214,0.3)" strokeWidth={0.5} />
                {xi.slice(0, 11).map((p, i) => (
                  <PitchDot
                    key={p.player_id}
                    player={p}
                    x={FULL_PITCH_SLOTS[i][0]}
                    y={FULL_PITCH_SLOTS[i][1]}
                    mode={mode}
                    nameMaxWidth={NAME_LANE_W[i]}
                    isMotm={motmId === p.player_id}
                  />
                ))}
              </svg>
            </div>
            <BenchList rows={bench} />
          </div>
        ))}
      </div>
      <p className="text-[11px] text-sage font-mono leading-relaxed">
        Gold ring: U-14. Brass glow: man of the match. Name ↓minute: subbed off. Bench ↑minute: came on. Unused substitutes sit without a rating.
      </p>
    </div>
  );
};

const STAT_ROWS: Array<{ key: keyof NonNullable<Fixture['stats']>['home']; label: string; suffix?: string }> = [
  { key: 'xg', label: 'Expected goals' },
  { key: 'shots', label: 'Shots' },
  { key: 'on_target', label: 'Shots on target' },
  { key: 'possession', label: 'Possession', suffix: '%' },
  { key: 'corners', label: 'Corners' },
  { key: 'passes', label: 'Passes' },
  { key: 'pass_accuracy', label: 'Pass accuracy', suffix: '%' },
  { key: 'fouls', label: 'Fouls' },
  { key: 'yellows', label: 'Yellow cards' },
  { key: 'reds', label: 'Red cards' },
];

function StatDualBar({ label, home, away, suffix }: { label: string; home: number; away: number; suffix?: string }) {
  const total = home + away || 1;
  const hPct = (home / total) * 100;
  const aPct = (away / total) * 100;
  const fmt = (n: number) => (Number.isInteger(n) ? String(n) : n.toFixed(2));
  return (
    <div className="space-y-1">
      <div className="flex items-center justify-between text-[13px] font-mono">
        <span className={cx('font-bold', home > away ? 'text-brass' : 'text-bone/80')}>{fmt(home)}{suffix ?? ''}</span>
        <span className="text-sage font-sans font-medium text-[12.5px]">{label}</span>
        <span className={cx('font-bold', away > home ? 'text-brass' : 'text-bone/80')}>{fmt(away)}{suffix ?? ''}</span>
      </div>
      <div className="grid grid-cols-2 gap-1.5">
        <div className="h-1.5 rounded-full bg-line/50 overflow-hidden flex justify-end">
          <div className="h-full rounded-full bg-bone" style={{ width: `${hPct}%` }} />
        </div>
        <div className="h-1.5 rounded-full bg-line/50 overflow-hidden">
          <div className="h-full rounded-full bg-brass" style={{ width: `${aPct}%` }} />
        </div>
      </div>
    </div>
  );
}

function RatingsTable({ rows, clubName, kicker }: { rows: MatchPlayerRow[]; clubName: string; kicker?: string }) {
  const sorted = [...rows].sort((a, b) => (b.rating ?? -1) - (a.rating ?? -1));
  return (
    <div className="panel-tight overflow-hidden">
      <div className="px-3 py-2 border-b border-line">
        <p className="text-[12px] font-semibold text-bone truncate">{clubName}</p>
        {kicker && <p className="text-[11px] font-mono text-sage">{kicker}</p>}
      </div>
      <table className="w-full text-left text-[12.5px]">
        <thead className="table-head">
          <tr>
            <th className="py-2 px-3">Player</th>
            <th className="py-2 px-2">Min</th>
            <th className="py-2 px-2">G</th>
            <th className="py-2 px-2">A</th>
            <th className="py-2 px-3">Rt</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-line/70">
          {sorted.map((p) => (
            <tr key={p.player_id} className={p.played === false ? 'opacity-50' : undefined}>
              <td className="py-1.5 px-3 text-bone font-semibold truncate">
                <PlayerNameButton playerId={p.player_id}>{p.full_name}</PlayerNameButton>
                {p.on_minute != null && <span className="ml-1.5 text-sage font-mono font-normal">↑{p.on_minute}</span>}
                {p.off_minute != null && p.card !== 'red' && <span className="ml-1.5 text-sage font-mono font-normal">↓{p.off_minute}</span>}
                {p.card === 'red' && <span className="ml-1.5 text-ember">sent off</span>}
                {p.card === 'yellow' && <span className="ml-1.5 text-brass">booked</span>}
              </td>
              <td className="py-1.5 px-2 font-mono text-sage">{p.minutes ?? (p.played === false ? 0 : 90)}</td>
              <td className="py-1.5 px-2 font-mono text-bone">{p.match_goals || '—'}</td>
              <td className="py-1.5 px-2 font-mono text-sage">{p.match_assists || '—'}</td>
              <td className={cx('py-1.5 px-3 font-mono font-bold', (p.rating ?? 0) >= 8 ? 'text-brass' : 'text-bone')}>
                {p.rating != null ? p.rating.toFixed(1) : '—'}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

const ShotMapPitch: React.FC<{ fixture: Fixture }> = ({ fixture }) => {
  const shotMap = fixture.shot_map;
  if (!shotMap || !shotMap.shots || shotMap.shots.length === 0) {
    return (
      <div className="py-12 text-center text-sage text-[13px] border border-line rounded-xl bg-ink/30">
        Shot coordinate telemetry not available for this fixture.
      </div>
    );
  }

  const hCol = rgbCss(fixture.home.primary_color, '#C47A3A');
  const aCol = rgbCss(fixture.away.primary_color, '#3D6B8A');

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-2 gap-4 p-4 rounded-xl bg-cardLight border border-line">
        <div className="flex items-center gap-3">
          <ClubCrest club={fixture.home} size={32} />
          <div>
            <p className="text-xs text-sage font-mono uppercase">{fixture.home.short_name} Expected Goals</p>
            <p className="text-xl font-bold font-mono text-bone">{shotMap.total_home_xg.toFixed(2)} <span className="text-xs font-normal text-sage">xG</span></p>
          </div>
        </div>
        <div className="flex items-center justify-end gap-3 text-right">
          <div>
            <p className="text-xs text-sage font-mono uppercase">{fixture.away.short_name} Expected Goals</p>
            <p className="text-xl font-bold font-mono text-bone">{shotMap.total_away_xg.toFixed(2)} <span className="text-xs font-normal text-sage">xG</span></p>
          </div>
          <ClubCrest club={fixture.away} size={32} />
        </div>
      </div>

      <div className="relative rounded-2xl overflow-hidden border border-line bg-[#16422C]">
        <svg viewBox="0 0 100 64" className="w-full h-auto block select-none">
          <rect x="2" y="2" width="96" height="60" fill="none" stroke="rgba(213, 228, 194, 0.4)" strokeWidth="0.6" />
          <line x1="50" y1="2" x2="50" y2="62" stroke="rgba(213, 228, 194, 0.4)" strokeWidth="0.6" />
          <circle cx="50" cy="32" r="9" fill="none" stroke="rgba(213, 228, 194, 0.4)" strokeWidth="0.6" />
          <circle cx="50" cy="32" r="0.8" fill="rgba(213, 228, 194, 0.6)" />

          <rect x="2" y="15" width="16" height="34" fill="none" stroke="rgba(213, 228, 194, 0.4)" strokeWidth="0.6" />
          <rect x="2" y="23" width="6" height="18" fill="none" stroke="rgba(213, 228, 194, 0.4)" strokeWidth="0.6" />
          <circle cx="11" cy="32" r="0.7" fill="rgba(213, 228, 194, 0.6)" />

          <rect x="82" y="15" width="16" height="34" fill="none" stroke="rgba(213, 228, 194, 0.4)" strokeWidth="0.6" />
          <rect x="92" y="23" width="6" height="18" fill="none" stroke="rgba(213, 228, 194, 0.4)" strokeWidth="0.6" />
          <circle cx="89" cy="32" r="0.7" fill="rgba(213, 228, 194, 0.6)" />

          {shotMap.shots.map((s, idx) => {
            const cxVal = Math.max(4, Math.min(96, s.x * 100));
            const cyVal = Math.max(4, Math.min(60, s.y * 64));
            const r = Math.max(1.6, Math.min(4.5, (s.xg || 0.2) * 6));
            const isGoal = s.outcome === 'goal';
            const isSave = s.outcome === 'save';

            const fillCol = isGoal ? '#F3E6C4' : isSave ? '#38BDF8' : 'rgba(213, 228, 194, 0.45)';
            const strokeCol = s.team === 'home' ? hCol : aCol;

            return (
              <g key={idx} className="cursor-pointer">
                {isGoal && (
                  <circle cx={cxVal} cy={cyVal} r={r + 1.8} fill="none" stroke="#F3E6C4" strokeWidth="0.5" strokeDasharray="1.2 1" />
                )}
                {s.is_wonderkid && (
                  <circle cx={cxVal} cy={cyVal} r={r + 2.6} fill="none" stroke="#FACC15" strokeWidth="0.4" />
                )}
                <circle
                  cx={cxVal}
                  cy={cyVal}
                  r={r}
                  fill={fillCol}
                  stroke={strokeCol}
                  strokeWidth={isGoal ? 1.0 : 0.6}
                />
                <title>{`${s.minute}' - ${s.shooter?.full_name ?? 'Player'} (${s.outcome.toUpperCase()}) · ${s.xg.toFixed(2)} xG`}</title>
              </g>
            );
          })}
        </svg>

        <div className="p-3 bg-ink/80 border-t border-line flex flex-wrap items-center justify-between gap-4 text-xs font-mono text-sage">
          <div className="flex items-center gap-4">
            <span className="flex items-center gap-1.5"><span className="w-2.5 h-2.5 rounded-full bg-brass border border-white inline-block" /> Goal</span>
            <span className="flex items-center gap-1.5"><span className="w-2.5 h-2.5 rounded-full bg-[#38BDF8] inline-block" /> Saved</span>
            <span className="flex items-center gap-1.5"><span className="w-2.5 h-2.5 rounded-full bg-sage/40 inline-block" /> Miss / Block</span>
          </div>
          <div className="flex items-center gap-3 text-[11px]">
            <span>Dot Size = Shot Quality (xG)</span>
            <span className="text-brass">Golden Ring = Wonderkid</span>
          </div>
        </div>
      </div>

      {shotMap.xg_flow && shotMap.xg_flow.length > 1 && (
        <div className="p-4 rounded-xl bg-cardLight border border-line">
          <p className="eyebrow mb-3">Cumulative Match Momentum (xG Flow)</p>
          <div className="relative h-28 w-full">
            <svg viewBox="0 0 100 50" preserveAspectRatio="none" className="w-full h-full overflow-visible">
              <line x1="0" y1="12.5" x2="100" y2="12.5" stroke="rgba(255,255,255,0.05)" strokeWidth="0.5" />
              <line x1="0" y1="25" x2="100" y2="25" stroke="rgba(255,255,255,0.05)" strokeWidth="0.5" />
              <line x1="0" y1="37.5" x2="100" y2="37.5" stroke="rgba(255,255,255,0.05)" strokeWidth="0.5" />
              <line x1="50" y1="0" x2="50" y2="50" stroke="rgba(255,255,255,0.1)" strokeWidth="0.5" strokeDasharray="2 2" />

              {(() => {
                const maxVal = Math.max(2.5, shotMap.total_home_xg, shotMap.total_away_xg);
                const hPts = shotMap.xg_flow.map(pt => `${(pt.minute / 90) * 100},${50 - (pt.home_xg / maxVal) * 46}`).join(' ');
                const aPts = shotMap.xg_flow.map(pt => `${(pt.minute / 90) * 100},${50 - (pt.away_xg / maxVal) * 46}`).join(' ');
                return (
                  <>
                    <polyline points={hPts} fill="none" stroke={hCol} strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" />
                    <polyline points={aPts} fill="none" stroke={aCol} strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" />
                  </>
                );
              })()}
            </svg>
          </div>
          <div className="flex justify-between text-[11px] font-mono text-sage mt-2">
            <span>0' Kickoff</span>
            <span>45' Halftime</span>
            <span>90' Fulltime</span>
          </div>
        </div>
      )}
    </div>
  );
};

const HeatmapTerritoryView: React.FC<{ fixture: Fixture }> = ({ fixture }) => {
  const hm = fixture.touch_heatmap;
  if (!hm) {
    return (
      <div className="py-12 text-center text-sage text-[13px] border border-line rounded-xl bg-ink/30">
        Touch spatial territory data not available for this fixture.
      </div>
    );
  }

  const hCol = rgbCss(fixture.home.primary_color, '#C47A3A');
  const aCol = rgbCss(fixture.away.primary_color, '#3D6B8A');

  return (
    <div className="space-y-6">
      <div className="rounded-2xl overflow-hidden border border-line bg-[#133824] p-2">
        <svg viewBox="0 0 100 64" className="w-full h-auto block select-none">
          <rect x="2" y="2" width="96" height="60" fill="none" stroke="rgba(213, 228, 194, 0.3)" strokeWidth="0.6" />
          <line x1="50" y1="2" x2="50" y2="62" stroke="rgba(213, 228, 194, 0.3)" strokeWidth="0.6" />
          <circle cx="50" cy="32" r="9" fill="none" stroke="rgba(213, 228, 194, 0.3)" strokeWidth="0.6" />

          <line x1="33.3" y1="2" x2="33.3" y2="62" stroke="rgba(213, 228, 194, 0.15)" strokeWidth="0.5" strokeDasharray="1 1" />
          <line x1="66.6" y1="2" x2="66.6" y2="62" stroke="rgba(213, 228, 194, 0.15)" strokeWidth="0.5" strokeDasharray="1 1" />

          {hm.home_points.map(([x, y], idx) => (
            <circle
              key={`h-${idx}`}
              cx={Math.max(3, Math.min(97, x * 100))}
              cy={Math.max(3, Math.min(61, y * 64))}
              r={1.8}
              fill={hCol}
              opacity={0.38}
            />
          ))}

          {hm.away_points.map(([x, y], idx) => (
            <circle
              key={`a-${idx}`}
              cx={Math.max(3, Math.min(97, x * 100))}
              cy={Math.max(3, Math.min(61, y * 64))}
              r={1.8}
              fill={aCol}
              opacity={0.38}
            />
          ))}
        </svg>

        <div className="p-3 flex items-center justify-between text-xs font-mono text-sage">
          <span className="flex items-center gap-1.5"><span className="w-2.5 h-2.5 rounded-full inline-block" style={{ backgroundColor: hCol }} /> {fixture.home.short_name} Density</span>
          <span className="flex items-center gap-1.5"><span className="w-2.5 h-2.5 rounded-full inline-block" style={{ backgroundColor: aCol }} /> {fixture.away.short_name} Density</span>
        </div>
      </div>

      <div className="p-5 rounded-xl bg-cardLight border border-line space-y-4">
        <p className="eyebrow">Spatial Territory & Field Tilt (%)</p>

        <div className="space-y-3">
          <div>
            <div className="flex justify-between text-xs font-mono text-bone mb-1">
              <span>{fixture.home.short_name} Def. Third: {hm.home_zones.defensive}%</span>
              <span>{fixture.away.short_name} Att. Third: {hm.away_zones.attacking}%</span>
            </div>
            <div className="h-2 w-full rounded-full bg-ink overflow-hidden flex">
              <div style={{ width: `${hm.home_zones.defensive}%`, backgroundColor: hCol }} />
              <div className="flex-1 bg-ink/80" />
              <div style={{ width: `${hm.away_zones.attacking}%`, backgroundColor: aCol }} />
            </div>
          </div>

          <div>
            <div className="flex justify-between text-xs font-mono text-bone mb-1">
              <span>Midfield Battle: {hm.home_zones.midfield}%</span>
              <span>Midfield Battle: {hm.away_zones.midfield}%</span>
            </div>
            <div className="h-2 w-full rounded-full bg-ink overflow-hidden flex">
              <div style={{ width: `${hm.home_zones.midfield}%`, backgroundColor: hCol }} />
              <div className="flex-1 bg-ink/80" />
              <div style={{ width: `${hm.away_zones.midfield}%`, backgroundColor: aCol }} />
            </div>
          </div>

          <div>
            <div className="flex justify-between text-xs font-mono text-bone mb-1">
              <span>{fixture.home.short_name} Att. Third: {hm.home_zones.attacking}%</span>
              <span>{fixture.away.short_name} Def. Third: {hm.away_zones.defensive}%</span>
            </div>
            <div className="h-2 w-full rounded-full bg-ink overflow-hidden flex">
              <div style={{ width: `${hm.home_zones.attacking}%`, backgroundColor: hCol }} />
              <div className="flex-1 bg-ink/80" />
              <div style={{ width: `${hm.away_zones.defensive}%`, backgroundColor: aCol }} />
            </div>
          </div>
        </div>

        <div className="pt-3 border-t border-line grid grid-cols-3 gap-3 text-center text-xs font-mono">
          <div className="p-2 rounded-lg bg-ink/40">
            <p className="text-sage text-[10px]">LEFT FLANK</p>
            <p className="font-bold text-bone mt-0.5">{hm.home_zones.left}% / {hm.away_zones.left}%</p>
          </div>
          <div className="p-2 rounded-lg bg-ink/40">
            <p className="text-sage text-[10px]">CENTRAL CHANNEL</p>
            <p className="font-bold text-bone mt-0.5">{hm.home_zones.center}% / {hm.away_zones.center}%</p>
          </div>
          <div className="p-2 rounded-lg bg-ink/40">
            <p className="text-sage text-[10px]">RIGHT FLANK</p>
            <p className="font-bold text-bone mt-0.5">{hm.home_zones.right}% / {hm.away_zones.right}%</p>
          </div>
        </div>
      </div>
    </div>
  );
};

const PressConferenceView: React.FC<{ fixture: Fixture }> = ({ fixture }) => {
  const press = fixture.press_conference;
  const hMgr = fixture.home_manager || fixture.home.manager;
  const aMgr = fixture.away_manager || fixture.away.manager;

  return (
    <div className="space-y-6">
      <div className="p-5 rounded-2xl bg-gradient-to-r from-cardLight via-ink to-cardLight border border-brass/40 shadow-raised text-center">
        <span className="px-2.5 py-1 rounded-full bg-brass/10 border border-brass/30 text-[11px] font-mono font-bold text-brass uppercase tracking-wider">
          Post-Match Media Centre
        </span>
        <h3 className="font-display font-semibold text-xl text-bone mt-2.5 leading-snug">
          "{press?.headline ?? `${fixture.home.short_name} and ${fixture.away.short_name} conclude tactical encounter`}"
        </h3>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
        <div className="p-5 rounded-xl bg-cardLight border border-line flex flex-col justify-between">
          <div>
            <div className="flex items-center gap-3 mb-3">
              <ClubCrest club={fixture.home} size={36} />
              <div>
                <p className="font-bold text-[15px] text-bone leading-tight">{hMgr?.name ?? `${fixture.home.short_name} Manager`}</p>
                <p className="text-xs text-brass font-mono">
                  {hMgr?.archetype_label ?? hMgr?.tactic ?? 'Tactician'} · {fixture.home.short_name}
                </p>
              </div>
            </div>
            <blockquote className="text-sm text-bone/90 italic border-l-2 border-brass/60 pl-3.5 py-1">
              "{press?.home_quote ?? 'The squad gave their all on the pitch today. We will analyze the tape and keep striving for consistency.'}"
            </blockquote>
          </div>
          {hMgr?.press_intensity && (
            <div className="mt-4 pt-3 border-t border-line/60 flex items-center justify-between text-[11px] font-mono text-sage">
              <span>Press: {hMgr.press_intensity}</span>
              <span>Tempo: {hMgr.tempo}</span>
              <span>Line: {hMgr.line_height}</span>
            </div>
          )}
        </div>

        <div className="p-5 rounded-xl bg-cardLight border border-line flex flex-col justify-between">
          <div>
            <div className="flex items-center gap-3 mb-3">
              <ClubCrest club={fixture.away} size={36} />
              <div>
                <p className="font-bold text-[15px] text-bone leading-tight">{aMgr?.name ?? `${fixture.away.short_name} Manager`}</p>
                <p className="text-xs text-brass font-mono">
                  {aMgr?.archetype_label ?? aMgr?.tactic ?? 'Tactician'} · {fixture.away.short_name}
                </p>
              </div>
            </div>
            <blockquote className="text-sm text-bone/90 italic border-l-2 border-brass/60 pl-3.5 py-1">
              "{press?.away_quote ?? 'Every match at this elite level is decided by fine margins. We respect our opponents and prepare for the next battle.'}"
            </blockquote>
          </div>
          {aMgr?.press_intensity && (
            <div className="mt-4 pt-3 border-t border-line/60 flex items-center justify-between text-[11px] font-mono text-sage">
              <span>Press: {aMgr.press_intensity}</span>
              <span>Tempo: {aMgr.tempo}</span>
              <span>Line: {aMgr.line_height}</span>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

interface HighlightItem {
  id: string;
  minute: number;
  type: 'goal' | 'penalty' | 'own_goal' | 'red' | 'save' | 'woodwork';
  title: string;
  commentary: string;
  team: 'home' | 'away';
  player_name: string;
  player_id?: string;
  assister_name?: string;
  start: [number, number];
  end: [number, number];
  xg?: number;
  home_score: number;
  away_score: number;
}

const HighlightsReelView: React.FC<{ fixture: Fixture }> = ({ fixture }) => {
  const hCol = rgbCss(fixture.home.primary_color);
  const aCol = rgbCss(fixture.away.primary_color);

  const highlights = useMemo<HighlightItem[]>(() => {
    const list: HighlightItem[] = [];
    let runningHome = 0;
    let runningAway = 0;

    const sortedEvents = [...fixture.events].sort((a, b) => a.minute - b.minute || a.seq - b.seq);

    for (const e of sortedEvents) {
      if (e.type === 'goal' || e.type === 'penalty' || e.type === 'own_goal') {
        const isOg = e.type === 'own_goal';
        const teamSide = isOg ? (e.beneficiary ?? (e.side === 'home' ? 'away' : 'home')) : e.side;
        if (teamSide === 'home') runningHome++;
        else runningAway++;

        const isHome = teamSide === 'home';
        const matchingShot = fixture.shot_map?.shots?.find(
          (s) => Math.abs(s.minute - e.minute) <= 1 && s.outcome === 'goal',
        );

        const startX = matchingShot ? matchingShot.x * 100 : (isHome ? 72 : 28);
        const startY = matchingShot ? matchingShot.y * 64 : (24 + (e.minute % 16));
        const endX = isHome ? 98 : 2;
        const endY = 28 + (e.minute % 8);

        list.push({
          id: `hl-goal-${e.minute}-${e.seq}`,
          minute: e.minute,
          type: e.type,
          title: isOg ? 'OWN GOAL!' : e.type === 'penalty' ? 'PENALTY GOAL!' : 'GOAL!',
          commentary: isOg
            ? `${e.scorer?.full_name ?? 'The defender'} inadvertently deflects the ball into his own net!`
            : `${e.scorer?.full_name ?? 'The attacker'} finishes with sublime precision! ${e.assister ? `Unselfish assist from ${e.assister.full_name}.` : 'Clinical solo effort.'}`,
          team: teamSide,
          player_name: e.scorer?.full_name ?? 'Player',
          player_id: e.scorer?.player_id,
          assister_name: e.assister?.full_name,
          start: [Math.max(6, Math.min(94, startX)), Math.max(6, Math.min(58, startY))],
          end: [endX, endY],
          xg: matchingShot?.xg ?? 0.45,
          home_score: runningHome,
          away_score: runningAway,
        });
      } else if (e.type === 'red') {
        const isHome = e.side === 'home';
        list.push({
          id: `hl-red-${e.minute}-${e.seq}`,
          minute: e.minute,
          type: 'red',
          title: 'RED CARD DISMISSAL!',
          commentary: `The referee brandishes red for ${e.player?.full_name ?? 'the defender'} following a serious infringement!`,
          team: e.side,
          player_name: e.player?.full_name ?? 'Player',
          player_id: e.player?.player_id,
          start: [isHome ? 42 : 58, 32],
          end: [isHome ? 42 : 58, 32],
          home_score: runningHome,
          away_score: runningAway,
        });
      } else if (e.type === 'penalty_miss') {
        const isHome = e.side === 'home';
        list.push({
          id: `hl-penmiss-${e.minute}-${e.seq}`,
          minute: e.minute,
          type: 'woodwork',
          title: 'PENALTY SQUANDERED!',
          commentary: `${e.scorer?.full_name ?? 'The taker'} steps up but cannot find the target! Massive let-off!`,
          team: e.side,
          player_name: e.scorer?.full_name ?? 'Player',
          player_id: e.scorer?.player_id,
          start: [isHome ? 88 : 12, 32],
          end: [isHome ? 98 : 2, 18],
          home_score: runningHome,
          away_score: runningAway,
        });
      }
    }

    if (fixture.shot_map?.shots) {
      for (const s of fixture.shot_map.shots) {
        if (s.outcome === 'save' && (s.xg >= 0.2 || list.length < 3)) {
          const isHome = s.team === 'home';
          list.push({
            id: `hl-save-${s.minute}-${s.x}`,
            minute: s.minute,
            type: 'save',
            title: 'SPECTACULAR SAVE!',
            commentary: `Point-blank stop! Goalkeeper reacts with cat-like reflexes to deny ${s.shooter?.full_name ?? 'the attack'}!`,
            team: s.team,
            player_name: s.shooter?.full_name ?? 'Attacker',
            player_id: s.shooter && 'player_id' in s.shooter ? (s.shooter as { player_id?: string }).player_id : undefined,
            start: [Math.max(6, Math.min(94, s.x * 100)), Math.max(6, Math.min(58, s.y * 64))],
            end: [isHome ? 97 : 3, 32],
            xg: s.xg,
            home_score: runningHome,
            away_score: runningAway,
          });
        }
      }
    }

    list.sort((a, b) => a.minute - b.minute);

    if (list.length === 0) {
      list.push({
        id: 'hl-baseline',
        minute: 45,
        type: 'save',
        title: 'TACTICAL STANDOFF',
        commentary: 'A fiercely contested tactical battle with defensive blocks dominating the rhythm.',
        team: 'home',
        player_name: fixture.home.short_name,
        start: [50, 32],
        end: [50, 32],
        home_score: fixture.home_goals ?? 0,
        away_score: fixture.away_goals ?? 0,
      });
    }

    return list;
  }, [fixture]);

  const [currentIndex, setCurrentIndex] = useState(0);
  const [isPlaying, setIsPlaying] = useState(false);

  useEffect(() => {
    if (!isPlaying) return;
    const timer = setInterval(() => {
      setCurrentIndex((prev) => (prev + 1) % highlights.length);
    }, 3200);
    return () => clearInterval(timer);
  }, [isPlaying, highlights.length]);

  const active = highlights[currentIndex] ?? highlights[0];
  const activeClub = active.team === 'home' ? fixture.home : fixture.away;
  const isGoal = active.type === 'goal' || active.type === 'penalty' || active.type === 'own_goal';
  const isRed = active.type === 'red';

  return (
    <div className="space-y-5">
      <div className="p-4 rounded-xl bg-cardLight border border-line flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <Film size={18} className="text-brass" />
          <span className="font-display font-semibold text-bone text-[16px]">Match Highlights Reel</span>
          <span className="text-[12px] font-mono text-sage ml-1">
            ({currentIndex + 1} of {highlights.length})
          </span>
        </div>

        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => {
              soundManager.playClick();
              setCurrentIndex((prev) => (prev - 1 + highlights.length) % highlights.length);
            }}
            className="p-2 rounded-lg bg-ink/50 border border-line hover:border-sage text-bone transition-colors"
            title="Previous highlight"
          >
            <SkipBack size={15} />
          </button>

          <button
            type="button"
            onClick={() => {
              soundManager.playClick();
              setIsPlaying((v) => !v);
            }}
            className={cx(
              'px-3.5 py-1.5 rounded-lg font-semibold text-[13px] inline-flex items-center gap-1.5 transition-colors border',
              isPlaying
                ? 'bg-amber-500/20 text-amber-300 border-amber-500/40'
                : 'bg-brass text-ink border-brass hover:bg-brass/90',
            )}
          >
            {isPlaying ? (
              <>
                <Pause size={14} /> Pause Reel
              </>
            ) : (
              <>
                <Play size={14} fill="currentColor" /> Play Reel
              </>
            )}
          </button>

          <button
            type="button"
            onClick={() => {
              soundManager.playClick();
              setCurrentIndex((prev) => (prev + 1) % highlights.length);
            }}
            className="p-2 rounded-lg bg-ink/50 border border-line hover:border-sage text-bone transition-colors"
            title="Next highlight"
          >
            <SkipForward size={15} />
          </button>
        </div>
      </div>

      <div className="flex items-center gap-2 overflow-x-auto pb-1 scrollbar-thin">
        {highlights.map((item, idx) => {
          const isSel = idx === currentIndex;
          const isItemGoal = item.type === 'goal' || item.type === 'penalty' || item.type === 'own_goal';
          const isItemRed = item.type === 'red';
          return (
            <button
              key={item.id}
              type="button"
              onClick={() => {
                soundManager.playClick();
                setCurrentIndex(idx);
                setIsPlaying(false);
              }}
              className={cx(
                'px-3 py-1.5 rounded-lg border text-left shrink-0 transition-all font-mono text-[12px] flex items-center gap-2',
                isSel
                  ? 'border-brass bg-brass/15 text-bone font-bold shadow-[0_0_10px_rgba(199,162,58,0.2)]'
                  : 'border-line/60 bg-cardBg/60 text-sage hover:border-line hover:text-bone',
              )}
            >
              <span>{item.minute}'</span>
              <span>{isItemGoal ? '⚽' : isItemRed ? '🟥' : '🧤'}</span>
              <span className="truncate max-w-[120px]">{item.player_name.split(' ').pop()}</span>
            </button>
          );
        })}
      </div>

      <div className="relative rounded-2xl overflow-hidden border border-line bg-[#133E29] shadow-2xl">
        <svg viewBox="0 0 100 64" className="w-full h-auto block select-none">
          <defs>
            <marker
              id="arrow-gold"
              viewBox="0 0 10 10"
              refX="6"
              refY="5"
              markerWidth="4"
              markerHeight="4"
              orient="auto-start-reverse"
            >
              <path d="M 0 1 L 10 5 L 0 9 z" fill="#F3E6C4" />
            </marker>
            <marker
              id="arrow-blue"
              viewBox="0 0 10 10"
              refX="6"
              refY="5"
              markerWidth="4"
              markerHeight="4"
              orient="auto-start-reverse"
            >
              <path d="M 0 1 L 10 5 L 0 9 z" fill="#38BDF8" />
            </marker>
            <linearGradient id="goal-gradient" x1="0%" y1="0%" x2="100%" y2="0%">
              <stop offset="0%" stopColor="#C7A23A" stopOpacity="0.8" />
              <stop offset="100%" stopColor="#FFF" stopOpacity="0.9" />
            </linearGradient>
          </defs>

          <rect x="2" y="2" width="96" height="60" fill="none" stroke="rgba(213, 228, 194, 0.4)" strokeWidth="0.6" />
          <line x1="50" y1="2" x2="50" y2="62" stroke="rgba(213, 228, 194, 0.4)" strokeWidth="0.6" />
          <circle cx="50" cy="32" r="9" fill="none" stroke="rgba(213, 228, 194, 0.4)" strokeWidth="0.6" />
          <circle cx="50" cy="32" r="0.8" fill="rgba(213, 228, 194, 0.6)" />

          <rect x="2" y="15" width="16" height="34" fill="none" stroke="rgba(213, 228, 194, 0.4)" strokeWidth="0.6" />
          <rect x="2" y="23" width="6" height="18" fill="none" stroke="rgba(213, 228, 194, 0.4)" strokeWidth="0.6" />
          <circle cx="11" cy="32" r="0.7" fill="rgba(213, 228, 194, 0.6)" />

          <rect x="82" y="15" width="16" height="34" fill="none" stroke="rgba(213, 228, 194, 0.4)" strokeWidth="0.6" />
          <rect x="92" y="23" width="6" height="18" fill="none" stroke="rgba(213, 228, 194, 0.4)" strokeWidth="0.6" />
          <circle cx="89" cy="32" r="0.7" fill="rgba(213, 228, 194, 0.6)" />

          {isRed ? (
            <g>
              <circle cx={active.start[0]} cy={active.start[1]} r={6.5} fill="rgba(239, 68, 68, 0.2)" stroke="#EF4444" strokeWidth="0.8" className="animate-pulse" />
              <rect x={active.start[0] - 2} y={active.start[1] - 3} width="4" height="6" rx="0.8" fill="#EF4444" stroke="#FFF" strokeWidth="0.4" />
              <text x={active.start[0]} y={active.start[1] + 7} fill="#FFF" fontSize="2.8" fontWeight="bold" textAnchor="middle">
                FOUL & RED
              </text>
            </g>
          ) : (
            <g>
              <line
                x1={active.start[0]}
                y1={active.start[1]}
                x2={active.end[0]}
                y2={active.end[1]}
                stroke={isGoal ? 'url(#goal-gradient)' : '#38BDF8'}
                strokeWidth={isGoal ? '1.2' : '0.9'}
                strokeDasharray={isGoal ? '3 1.5' : '2 1'}
                markerEnd={isGoal ? 'url(#arrow-gold)' : 'url(#arrow-blue)'}
              />

              <circle
                cx={active.start[0]}
                cy={active.start[1]}
                r={3}
                fill={active.team === 'home' ? hCol : aCol}
                stroke="#FFF"
                strokeWidth="0.8"
              />
              <text
                x={active.start[0]}
                y={active.start[1] - 4.2}
                fill="#FFF"
                fontSize="2.9"
                fontWeight="bold"
                textAnchor="middle"
              >
                {active.player_name}
              </text>

              <g transform={`translate(${active.end[0]}, ${active.end[1]})`}>
                {isGoal ? (
                  <>
                    <circle r="4.8" fill="none" stroke="#FACC15" strokeWidth="0.8" className="animate-ping opacity-75" />
                    <circle r="3" fill="#FACC15" stroke="#FFF" strokeWidth="0.6" />
                    <text y="0.9" textAnchor="middle" fontSize="3.2" fill="#000" fontWeight="bold">⚽</text>
                  </>
                ) : (
                  <>
                    <circle r="3.8" fill="none" stroke="#38BDF8" strokeWidth="0.6" className="animate-pulse" />
                    <circle r="2.6" fill="#38BDF8" stroke="#FFF" strokeWidth="0.5" />
                    <text y="0.8" textAnchor="middle" fontSize="2.8" fill="#FFF">🧤</text>
                  </>
                )}
              </g>
            </g>
          )}
        </svg>

        <div className="p-4 bg-ink/90 backdrop-blur-md border-t border-line flex flex-col md:flex-row items-start md:items-center justify-between gap-3">
          <div className="flex items-center gap-3">
            <ClubCrest club={activeClub} size={38} />
            <div>
              <div className="flex items-center gap-2">
                <span className="px-2 py-0.5 rounded bg-brass text-ink font-mono font-bold text-[11px]">
                  {active.minute}'
                </span>
                <span className={cx(
                  'px-2 py-0.5 rounded font-bold text-[10px] uppercase tracking-wider',
                  isGoal ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/40' :
                  isRed ? 'bg-ember/20 text-ember border border-ember/40' :
                  'bg-sky-500/20 text-sky-300 border border-sky-500/40',
                )}>
                  {active.title}
                </span>
                {active.xg && active.xg > 0 && (
                  <span className="text-[11px] font-mono text-sage">
                    {active.xg.toFixed(2)} xG
                  </span>
                )}
              </div>
              <p className="text-[14px] text-bone font-medium mt-1 leading-snug">
                "{active.commentary}"
              </p>
            </div>
          </div>

          <div className="text-right shrink-0">
            <div className="font-mono text-[11px] text-sage uppercase">Tally at {active.minute}'</div>
            <div className="font-mono font-bold text-[18px] text-bone">
              {fixture.home.short_name} {active.home_score} – {active.away_score} {fixture.away.short_name}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

interface MatchDetailModalProps {
  fixture: Fixture | null;
  onClose: () => void;
}

export const MatchDetailModal: React.FC<MatchDetailModalProps> = ({ fixture, onClose }) => {
  const [tab, setTab] = useState<'timeline' | 'lineups' | 'stats' | 'tactics' | 'press' | 'highlights'>('timeline');
  const [lineupMode, setLineupMode] = useState<'rating' | 'age'>('rating');

  const timeline = useMemo(() => {
    if (!fixture) return [];
    return [...fixture.events].sort((a, b) => a.minute - b.minute || a.seq - b.seq);
  }, [fixture]);

  return (
    <Modal open={!!fixture} onClose={onClose} maxWidth="max-w-6xl">
      {fixture && (
        <>
          <ModalHeader
            title={
              <span>
                {fixture.competition === 'ucl'
                  ? 'Champions Cup'
                  : fixture.competition === 'super-cup'
                    ? 'Super Cup'
                    : 'Super League'}
                {' · '}
                {fixture.competition === 'ucl' && fixture.stage !== 'Final'
                  ? `${fixture.stage}${fixture.leg ? ` · Leg ${fixture.leg}` : ''} · MW ${fixture.matchweek}`
                  : fixture.competition === 'ucl'
                    ? `Final · MW ${fixture.matchweek}`
                    : fixture.competition === 'super-cup'
                      ? `${fixture.stage} · MW ${fixture.matchweek}`
                      : `${fixture.stage || 'League'} · MW ${fixture.matchweek}`}
                {fixture.derby ? ` · ${fixture.derby}` : ''} · Full-time
              </span>
            }
            subtitle={`${fixture.home.home_stadium}${fixture.attendance ? ` · ${fixture.attendance.toLocaleString()} in` : ''}${fixture.referee ? ` · ${fixture.referee}` : ''} · ${fixture.method === 'live' ? 'Played live' : 'Simulated'} · Result stands`}
            onClose={onClose}
          />
          <div className="flex-1 min-h-0 overflow-y-auto">
          <div className="px-6 pt-4">
            <div className="flex items-center justify-between gap-3">
              <div className="flex items-center gap-3 flex-1 min-w-0">
                <ClubCrest club={fixture.home} size={46} />
                <span className="font-semibold text-[16px] text-bone leading-tight truncate">{fixture.home.club_name}</span>
              </div>
              <div className="text-center shrink-0 px-2">
                <div className="score-display text-[40px] leading-none text-bone">
                  {fixture.home_goals}<span className="text-sage mx-2 text-[26px]">–</span>{fixture.away_goals}
                </div>
                <p className="text-[11px] text-sage font-medium mt-1">
                  {fixture.decided_by === 'penalties'
                    ? `AET · ${fixture.penalties?.[0] ?? 0}–${fixture.penalties?.[1] ?? 0} pens`
                    : fixture.decided_by === 'extra_time'
                      ? 'After extra time'
                      : 'Full-time'}
                </p>
                {typeof fixture.ht_home === 'number' && (
                  <p className="text-[11px] font-mono text-sage/80">HT {fixture.ht_home}–{fixture.ht_away}</p>
                )}
              </div>
              <div className="flex items-center justify-end gap-3 flex-1 min-w-0">
                <span className="font-semibold text-[16px] text-bone leading-tight truncate text-right">{fixture.away.club_name}</span>
                <ClubCrest club={fixture.away} size={46} />
              </div>
            </div>
            {fixture.motm && (
              <div className="mt-2 mx-auto w-fit flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-brass/10 border border-brass/45 text-[12px] font-semibold text-brass">
                <Star size={12} fill="currentColor" /> Man of the match:{' '}
                <PlayerNameButton playerId={fixture.motm.player_id} className="text-brass font-semibold">
                  {fixture.motm.full_name} · {fixture.motm.rating.toFixed(1)}
                </PlayerNameButton>
              </div>
            )}
          </div>

          <div className="px-6 mt-3 grid grid-cols-3 sm:grid-cols-6 border-y border-line text-center" role="tablist" aria-label="Match detail">
            {(['timeline', 'lineups', 'stats', 'tactics', 'press', 'highlights'] as const).map((t) => (
              <button
                key={t}
                role="tab"
                aria-selected={tab === t}
                onClick={() => setTab(t)}
                className={cx(
                  'py-3 text-[11px] sm:text-[12px] font-bold uppercase tracking-[0.1em] transition-colors border-b-2 -mb-px truncate px-1',
                  tab === t ? 'text-bone border-bone' : 'text-sage border-transparent hover:text-bone',
                )}
              >
                {t === 'timeline'
                  ? 'Timeline'
                  : t === 'lineups'
                    ? 'Lineups'
                    : t === 'stats'
                      ? 'Stats'
                      : t === 'tactics'
                        ? 'Tactics & xG'
                        : t === 'press'
                          ? 'Press Centre'
                          : 'Highlights Reel'}
              </button>
            ))}
          </div>

          <div className="p-6 pt-5">
            {tab === 'timeline' && (
              <div className="space-y-3">
                <div className="flex items-center gap-2 text-[12px] font-mono text-sage">
                  <Flag size={13} /> <span>Kick-off · 0'</span>
                </div>
                {timeline.length === 0 && (
                  <p className="text-center text-sage text-[13px] py-10">A quiet game — no goals or cards, just football.</p>
                )}
                {timeline.map((e, i) => (
                  <TimelineRow key={`${e.minute}-${e.seq}-${i}`} event={e} fixture={fixture} />
                ))}
                <div className="flex items-center gap-2 text-[12px] font-mono text-sage">
                  <Flag size={13} /> <span>{fixture.decided_by ? 'Full-time · 120' : "Full-time · 90'"}</span>
                </div>
              </div>
            )}

            {tab === 'lineups' && (
              <div className="space-y-4">
                <div className="flex gap-1.5" role="group" aria-label="Lineup detail">
                  {(['rating', 'age'] as const).map((m) => (
                    <button
                      key={m}
                      onClick={() => setLineupMode(m)}
                      aria-pressed={lineupMode === m}
                      className={cx(
                        'px-4 py-1.5 rounded-lg text-[13px] font-semibold capitalize transition-colors border',
                        lineupMode === m ? 'bg-brass text-ink border-brass' : 'bg-transparent text-sage hover:text-bone border-line',
                      )}
                    >
                      {m === 'rating' ? 'Performance' : 'Age'}
                    </button>
                  ))}
                </div>
                <LineupsPitch fixture={fixture} mode={lineupMode} />
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 pt-2">
                  <RatingsTable rows={fixture.home_xi} clubName={fixture.home.club_name} kicker="Starting XI" />
                  <RatingsTable rows={fixture.away_xi} clubName={fixture.away.club_name} kicker="Starting XI" />
                  {(fixture.home_bench?.length ?? 0) > 0 && (
                    <RatingsTable rows={fixture.home_bench ?? []} clubName={fixture.home.club_name} kicker="Bench" />
                  )}
                  {(fixture.away_bench?.length ?? 0) > 0 && (
                    <RatingsTable rows={fixture.away_bench ?? []} clubName={fixture.away.club_name} kicker="Bench" />
                  )}
                </div>
              </div>
            )}

            {tab === 'stats' && fixture.stats && (
              <div>
                <div className="flex items-center justify-between mb-4">
                  <ClubCrest club={fixture.home} size={30} />
                  <p className="eyebrow">Match stats</p>
                  <ClubCrest club={fixture.away} size={30} />
                </div>
                <div className="space-y-3.5">
                  {STAT_ROWS.map((row) => {
                    const h = Number(fixture.stats!.home[row.key] ?? 0);
                    const a = Number(fixture.stats!.away[row.key] ?? 0);
                    return <StatDualBar key={row.key} label={row.label} home={h} away={a} suffix={row.suffix} />;
                  })}
                </div>
                {(fixture.head_to_head?.length ?? 0) > 0 && (
                  <div className="mt-6 pt-4 border-t border-line">
                    <p className="eyebrow mb-2">Previous meetings</p>
                    <div className="space-y-1.5">
                      {fixture.head_to_head!.map((h) => (
                        <p key={h.id} className="font-mono text-[12px] text-sage">
                          MW {h.matchweek} · {h.competition === 'ucl' ? 'Champions Cup' : h.competition === 'super-cup' ? 'Super Cup' : 'League'} · {h.home_goals}–{h.away_goals}
                        </p>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            )}

            {tab === 'tactics' && (
              <div className="space-y-8">
                <ShotMapPitch fixture={fixture} />
                <div className="pt-6 border-t border-line">
                  <HeatmapTerritoryView fixture={fixture} />
                </div>
              </div>
            )}

            {tab === 'press' && (
              <PressConferenceView fixture={fixture} />
            )}

            {tab === 'highlights' && (
              <HighlightsReelView fixture={fixture} />
            )}
          </div>
        </div>
        </>
      )}
    </Modal>
  );
};
