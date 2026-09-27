import React, { useState } from 'react';
import { FlaskConical, RefreshCw } from 'lucide-react';
import type { Club, WhatIfClubDelta, WhatIfResult } from '../../types';
import { fetchWhatIf } from '../../services/api';
import { ClubCrest, PrimaryButton } from '../ui/ui';

/**
 * What-if sandbox (F4): resolves the fixture under a scratch seed on the
 * server, which deep-copies the inputs and never touches the real universe.
 * The panel is observational only — the recorded result always stands.
 */
export const WhatIfPanel: React.FC<{
  fixtureId: string;
  home: Club | null;
  away: Club | null;
}> = ({ fixtureId, home, away }) => {
  const [result, setResult] = useState<WhatIfResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  if (!fixtureId) return null;

  const run = async (seed: number) => {
    setLoading(true);
    setError('');
    try {
      const res = await fetchWhatIf(fixtureId, seed);
      if (!res) {
        setError('The sandbox could not resolve this fixture.');
      } else {
        setResult(res);
      }
    } catch {
      setError('The sandbox could not resolve this fixture.');
    } finally {
      setLoading(false);
    }
  };

  const hypo = result?.hypothetical;
  const actual = result?.actual;

  return (
    <section className="console-card p-4 sm:p-5" data-whatif-panel="true">
      <div className="flex items-center justify-between gap-3">
        <p className="match-section-title">What-if sandbox</p>
        <span className="flex items-center gap-1 text-[10px] uppercase tracking-[0.12em] text-sage">
          <FlaskConical size={12} aria-hidden="true" /> Simulation only
        </span>
      </div>
      <p className="mt-2 text-[12px] leading-relaxed text-sage">
        Re-resolve this fixture under a scratch seed. The recorded result stands; the sandbox never edits the real world.
      </p>

      {!result && !error && (
        <div className="mt-3">
          <PrimaryButton tone="cyan" onClick={() => void run(0)} disabled={loading}>
            <FlaskConical size={14} aria-hidden="true" /> {loading ? 'Resolving…' : 'Run what-if'}
          </PrimaryButton>
        </div>
      )}

      {error && (
        <div className="mt-3 flex flex-wrap items-center gap-3">
          <p className="text-[12px] text-brass">{error}</p>
          <PrimaryButton tone="cyan" onClick={() => void run(0)} disabled={loading}>
            <RefreshCw size={14} aria-hidden="true" /> Retry
          </PrimaryButton>
        </div>
      )}

      {hypo && (
        <div className="mt-3 space-y-3">
          <div className="flex items-center justify-between gap-3 rounded-md border border-line bg-black/20 p-3">
            <div className="flex items-center gap-2">
              <ClubCrest club={home} size={20} />
              <span className="text-[13px] font-bold text-bone">{home?.short_name || 'Home'}</span>
            </div>
            <div className="text-center">
              <p className="font-mono text-[18px] font-bold text-bone">
                {hypo.home_goals}–{hypo.away_goals}
              </p>
              <p className="text-[10px] uppercase tracking-[0.1em] text-sage">
                xG {hypo.home_xg.toFixed(2)} – {hypo.away_xg.toFixed(2)}
              </p>
              {hypo.decided_by && (
                <p className="text-[10px] uppercase tracking-[0.1em] text-brass">{hypo.decided_by.replace('_', ' ')}</p>
              )}
            </div>
            <div className="flex items-center gap-2">
              <span className="text-[13px] font-bold text-bone">{away?.short_name || 'Away'}</span>
              <ClubCrest club={away} size={20} />
            </div>
          </div>

          {actual && (
            <p className="text-[12px] text-sage">
              Recorded: <span className="font-mono font-semibold text-bone">{actual.home_goals}–{actual.away_goals}</span>
              {actual.home_xg > 0 || actual.away_xg > 0 ? ` (xG ${actual.home_xg.toFixed(2)} – ${actual.away_xg.toFixed(2)})` : ''}
              {' '}· Hypothetical: <span className="font-mono font-semibold text-bone">{hypo.home_goals}–{hypo.away_goals}</span>
            </p>
          )}

          {result?.table?.applicable && (
            <div className="rounded-md border border-line bg-black/20 p-3">
              <p className="text-[11px] font-bold uppercase tracking-[0.1em] text-sage">Hypothetical table movement</p>
              <div className="mt-2 space-y-1.5">
                {[result.table.home, result.table.away].map((delta) =>
                  delta ? <TableMoveRow key={delta.club_id} delta={delta} /> : null,
                )}
              </div>
            </div>
          )}

          <div className="flex flex-wrap items-center gap-3">
            <PrimaryButton tone="cyan" onClick={() => void run((result?.scratch_seed ?? 0) + 1)} disabled={loading}>
              <RefreshCw size={14} aria-hidden="true" /> {loading ? 'Resolving…' : 'Reroll scratch seed'}
            </PrimaryButton>
            <span className="text-[10px] uppercase tracking-[0.1em] text-sage">Seed {result?.scratch_seed ?? 0}</span>
          </div>
        </div>
      )}
    </section>
  );
};

const TableMoveRow: React.FC<{ delta: WhatIfClubDelta }> = ({ delta }) => {
  const moved = delta.after_pos !== delta.before_pos;
  const dir = delta.after_pos < delta.before_pos ? 'up' : 'down';
  return (
    <p className="flex items-center justify-between gap-3 text-[12px] text-sage">
      <span className="font-semibold text-bone">{delta.short_name}</span>
      <span className="font-mono">
        {delta.before_pos}. {delta.before_pts}pts ({delta.before_gd >= 0 ? '+' : ''}{delta.before_gd})
        {' → '}
        <span className={moved ? (dir === 'up' ? 'text-emerald-400' : 'text-brass') : 'text-bone'}>
          {delta.after_pos}. {delta.after_pts}pts ({delta.after_gd >= 0 ? '+' : ''}{delta.after_gd})
        </span>
      </span>
    </p>
  );
};
