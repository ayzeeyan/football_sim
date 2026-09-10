import React from 'react';
import { Trophy } from 'lucide-react';
import type { Club } from '../types';
import type { SuperCupState, SuperCupTie } from '../services/api';
import { Card, ClubCrest, LoadingState, PanelHeader } from './ui/ui';
import { cx } from '../lib/format';

const PLAY_IN_KEYS = ['sc_pi_1', 'sc_pi_2', 'sc_pi_3', 'sc_pi_4'] as const;
const QF_KEYS = ['sc_qf_1', 'sc_qf_2', 'sc_qf_3', 'sc_qf_4'] as const;
const SF_KEYS = ['sc_sf_1', 'sc_sf_2'] as const;
const PLAY_IN_LABELS = ['5 vs 12', '6 vs 11', '7 vs 10', '8 vs 9'];

function isWinner(club: Club | null | undefined, winner: Club | null | undefined): boolean {
  return !!(club && winner && club.club_id === winner.club_id);
}

function scoreBits(tie: SuperCupTie | null | undefined): { home: string; away: string; note: string } {
  if (!tie?.leg1) return { home: '', away: '', note: '' };
  const note =
    tie.decided_by === 'penalties' && tie.penalties
      ? `${tie.penalties[0]}–${tie.penalties[1]} pens`
      : tie.decided_by === 'extra_time'
        ? 'AET'
        : '';
  return { home: String(tie.leg1[0]), away: String(tie.leg1[1]), note };
}

const ClubRow: React.FC<{
  club: Club | null;
  placeholder?: string;
  score?: string;
  won?: boolean;
  empty?: boolean;
}> = ({ club, placeholder, score, won, empty }) => (
  <div
    className={cx(
      'flex items-center gap-2 px-2.5 py-1.5 min-h-[34px]',
      won && 'bg-brass/[0.10]',
      empty && 'opacity-55',
    )}
  >
    {club ? (
      <ClubCrest club={club} size={22} />
    ) : (
      <span className="w-[22px] h-[22px] rounded-md border border-dashed border-line shrink-0" />
    )}
    <span className={cx('flex-1 truncate text-[12.5px] font-semibold', club ? 'text-bone' : 'text-sage')}>
      {club?.short_name ?? placeholder ?? 'TBD'}
    </span>
    {score !== undefined && score !== '' && (
      <span className={cx('font-mono text-[12px] font-bold tabular-nums', won ? 'text-brass' : 'text-sage')}>{score}</span>
    )}
  </div>
);

const BracketMatch: React.FC<{
  kicker: string;
  home: Club | null;
  away: Club | null;
  homeLabel?: string;
  awayLabel?: string;
  tie?: SuperCupTie | null;
  highlight?: boolean;
}> = ({ kicker, home, away, homeLabel, awayLabel, tie, highlight }) => {
  const bits = scoreBits(tie);
  const decided = !!tie?.winner || !!tie?.leg1;
  return (
    <div
      className={cx(
        'h-full min-h-[88px] flex flex-col justify-center rounded-xl border bg-ink/50 overflow-hidden',
        highlight ? 'border-brass/50' : 'border-line',
      )}
    >
      <p className="px-2.5 pt-1.5 pb-0.5 text-[10px] font-mono uppercase tracking-[0.12em] text-sage truncate">{kicker}</p>
      <ClubRow club={home} placeholder={homeLabel} score={bits.home} won={isWinner(home, tie?.winner)} empty={!home && !decided} />
      <div className="h-px bg-line/80" />
      <ClubRow club={away} placeholder={awayLabel} score={bits.away} won={isWinner(away, tie?.winner)} empty={!away && !decided} />
      {bits.note && <p className="px-2.5 pb-1.5 text-[10px] font-mono text-brass">{bits.note}</p>}
    </div>
  );
};

const ByeSlot: React.FC<{ seed: number; club: Club | null }> = ({ seed, club }) => (
  <div className="h-full min-h-[44px] flex items-center gap-2 px-2.5 rounded-xl border border-brass/30 bg-brass/[0.06]">
    <span className="font-mono text-[10px] text-brass font-bold w-6 shrink-0">S{seed}</span>
    {club ? <ClubCrest club={club} size={20} /> : <span className="w-5 h-5 rounded-md border border-dashed border-line shrink-0" />}
    <span className="truncate text-[12.5px] font-semibold text-bone">{club?.short_name ?? 'Bye'}</span>
    <span className="ml-auto text-[10px] font-mono uppercase tracking-[0.1em] text-brass">Bye</span>
  </div>
);

export const SuperCupBoard: React.FC<{ data: SuperCupState | null; loading?: boolean }> = ({ data, loading }) => {
  if (loading || !data) return <Card><LoadingState message="Loading the Super Cup…" /></Card>;

  const playIn = PLAY_IN_KEYS.map((k) => data.play_in[k] ?? null);
  const qfDrawn = QF_KEYS.map((k) => data.quarter_finals[k] ?? null);
  const sf = SF_KEYS.map((k) => data.semi_finals[k] ?? null);
  const byes = data.byes;

  const qf: Array<SuperCupTie | null> = qfDrawn.some(Boolean)
    ? qfDrawn
    : [
        { home: byes[0] ?? null, away: playIn[3]?.winner ?? null, leg1: null, winner: null },
        { home: byes[1] ?? null, away: playIn[2]?.winner ?? null, leg1: null, winner: null },
        { home: byes[2] ?? null, away: playIn[1]?.winner ?? null, leg1: null, winner: null },
        { home: byes[3] ?? null, away: playIn[0]?.winner ?? null, leg1: null, winner: null },
      ];

  const qfAwayLabel = [
    playIn[3]?.winner ? undefined : 'Winner · 8 vs 9',
    playIn[2]?.winner ? undefined : 'Winner · 7 vs 10',
    playIn[1]?.winner ? undefined : 'Winner · 6 vs 11',
    playIn[0]?.winner ? undefined : 'Winner · 5 vs 12',
  ];

  const finalHome = data.final?.team1 ?? sf[0]?.winner ?? null;
  const finalAway = data.final?.team2 ?? sf[1]?.winner ?? null;
  const finalTie: SuperCupTie | null = data.final
    ? {
        home: data.final.team1,
        away: data.final.team2,
        leg1: data.final.score,
        winner: data.final.winner,
        decided_by: data.final.decided_by,
        penalties: data.final.penalties,
      }
    : null;

  const stage = (data.stage || 'PLAY_IN').replace(/_/g, ' ').toLowerCase();

  return (
    <div className="space-y-4">
      <Card>
        <PanelHeader
          kicker={`Stage · ${stage}`}
          title="Super Cup"
          subtitle="Seeds 1–4 receive a bye. Play-in winners join them in the quarter-finals. Single legs, extra time if needed."
        />
      </Card>

      {data.champion && (
        <div className="panel-pad border-pitchtone/45 flex items-center gap-3">
          <div className="w-12 h-12 rounded-full bg-pitchtone/20 border border-pitchtone/40 flex items-center justify-center">
            <Trophy size={22} className="text-pitchtone" />
          </div>
          <div>
            <p className="eyebrow !text-pitchtone">Super Cup holders</p>
            <p className="font-display text-[22px] font-semibold text-bone">{data.champion.club_name}</p>
          </div>
        </div>
      )}

      <div className="panel-tight p-4 overflow-x-auto">
        <div className="min-w-[980px]">
          <div className="grid grid-cols-4 gap-3 mb-3 px-1">
            {['Play-in · MW 5', 'Quarter-finals · MW 12', 'Semi-finals · MW 20', 'Final · MW 26'].map((label) => (
              <p key={label} className="eyebrow">{label}</p>
            ))}
          </div>

          <div
            className="grid gap-x-6 gap-y-2"
            style={{
              gridTemplateColumns: '1.15fr 1fr 1fr 1.05fr',
              gridTemplateRows: 'repeat(8, minmax(46px, auto))',
            }}
          >
            {/* Path 1: seed 1 + 8v9 → QF1 → SF1 */}
            <div style={{ gridColumn: 1, gridRow: '1 / 2' }}><ByeSlot seed={1} club={byes[0] ?? null} /></div>
            <div style={{ gridColumn: 1, gridRow: '2 / 3' }}>
              <BracketMatch kicker={PLAY_IN_LABELS[3]} home={playIn[3]?.home ?? null} away={playIn[3]?.away ?? null} tie={playIn[3]} />
            </div>
            <div style={{ gridColumn: 2, gridRow: '1 / 3' }} className="relative pl-3">
              <span className="absolute left-0 top-1/4 bottom-1/4 w-3 border-l border-t border-b border-line/80 rounded-l-md" aria-hidden />
              <BracketMatch kicker="QF 1" home={qf[0]?.home ?? null} away={qf[0]?.away ?? null} awayLabel={qfAwayLabel[0]} homeLabel="Seed 1" tie={qf[0]} />
            </div>

            <div style={{ gridColumn: 1, gridRow: '3 / 4' }}><ByeSlot seed={2} club={byes[1] ?? null} /></div>
            <div style={{ gridColumn: 1, gridRow: '4 / 5' }}>
              <BracketMatch kicker={PLAY_IN_LABELS[2]} home={playIn[2]?.home ?? null} away={playIn[2]?.away ?? null} tie={playIn[2]} />
            </div>
            <div style={{ gridColumn: 2, gridRow: '3 / 5' }} className="relative pl-3">
              <span className="absolute left-0 top-1/4 bottom-1/4 w-3 border-l border-t border-b border-line/80 rounded-l-md" aria-hidden />
              <BracketMatch kicker="QF 2" home={qf[1]?.home ?? null} away={qf[1]?.away ?? null} awayLabel={qfAwayLabel[1]} homeLabel="Seed 2" tie={qf[1]} />
            </div>

            <div style={{ gridColumn: 3, gridRow: '1 / 5' }} className="relative pl-3">
              <span className="absolute left-0 top-[20%] bottom-[20%] w-3 border-l border-t border-b border-line/80 rounded-l-md" aria-hidden />
              <BracketMatch
                kicker="Semi-final 1"
                home={sf[0]?.home ?? qf[0]?.winner ?? null}
                away={sf[0]?.away ?? qf[1]?.winner ?? null}
                homeLabel="Winner QF 1"
                awayLabel="Winner QF 2"
                tie={sf[0]}
              />
            </div>

            {/* Path 2: seed 3 + 6v11 → QF3 → SF2 */}
            <div style={{ gridColumn: 1, gridRow: '5 / 6' }}><ByeSlot seed={3} club={byes[2] ?? null} /></div>
            <div style={{ gridColumn: 1, gridRow: '6 / 7' }}>
              <BracketMatch kicker={PLAY_IN_LABELS[1]} home={playIn[1]?.home ?? null} away={playIn[1]?.away ?? null} tie={playIn[1]} />
            </div>
            <div style={{ gridColumn: 2, gridRow: '5 / 7' }} className="relative pl-3">
              <span className="absolute left-0 top-1/4 bottom-1/4 w-3 border-l border-t border-b border-line/80 rounded-l-md" aria-hidden />
              <BracketMatch kicker="QF 3" home={qf[2]?.home ?? null} away={qf[2]?.away ?? null} awayLabel={qfAwayLabel[2]} homeLabel="Seed 3" tie={qf[2]} />
            </div>

            <div style={{ gridColumn: 1, gridRow: '7 / 8' }}><ByeSlot seed={4} club={byes[3] ?? null} /></div>
            <div style={{ gridColumn: 1, gridRow: '8 / 9' }}>
              <BracketMatch kicker={PLAY_IN_LABELS[0]} home={playIn[0]?.home ?? null} away={playIn[0]?.away ?? null} tie={playIn[0]} />
            </div>
            <div style={{ gridColumn: 2, gridRow: '7 / 9' }} className="relative pl-3">
              <span className="absolute left-0 top-1/4 bottom-1/4 w-3 border-l border-t border-b border-line/80 rounded-l-md" aria-hidden />
              <BracketMatch kicker="QF 4" home={qf[3]?.home ?? null} away={qf[3]?.away ?? null} awayLabel={qfAwayLabel[3]} homeLabel="Seed 4" tie={qf[3]} />
            </div>

            <div style={{ gridColumn: 3, gridRow: '5 / 9' }} className="relative pl-3">
              <span className="absolute left-0 top-[20%] bottom-[20%] w-3 border-l border-t border-b border-line/80 rounded-l-md" aria-hidden />
              <BracketMatch
                kicker="Semi-final 2"
                home={sf[1]?.home ?? qf[2]?.winner ?? null}
                away={sf[1]?.away ?? qf[3]?.winner ?? null}
                homeLabel="Winner QF 3"
                awayLabel="Winner QF 4"
                tie={sf[1]}
              />
            </div>

            <div style={{ gridColumn: 4, gridRow: '1 / 9' }} className="relative pl-3 flex flex-col justify-center">
              <span className="absolute left-0 top-[30%] bottom-[30%] w-3 border-l border-t border-b border-brass/40 rounded-l-md" aria-hidden />
              <div className="flex flex-col items-center gap-3">
                <div className="w-10 h-10 rounded-full bg-brass flex items-center justify-center text-ink">
                  <Trophy size={18} />
                </div>
                <BracketMatch
                  kicker="Final"
                  home={finalHome}
                  away={finalAway}
                  homeLabel="Winner SF 1"
                  awayLabel="Winner SF 2"
                  tie={finalTie}
                  highlight
                />
                {data.champion && (
                  <p className="text-[12px] font-semibold text-brass text-center">
                    {data.champion.club_name} lift the Super Cup
                  </p>
                )}
              </div>
            </div>
          </div>
        </div>
        <p className="mt-3 text-[11px] font-mono text-sage">Scroll sideways on a narrow screen. Brass names have gone through.</p>
      </div>
    </div>
  );
};
