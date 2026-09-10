import React from 'react';
import { Eye, Zap, ShieldAlert } from 'lucide-react';
import type { Fixture, Player } from '../types';
import { ovrTone, positionTone } from '../lib/constants';
import { cx } from '../lib/format';
import { soundManager } from '../audio/webAudio';
import { ClubCrest, FormPips, Modal, ModalHeader } from './ui/ui';
import { usePlayerSheet } from './PlayerSheet';
import { fixtureCompetitionLabel, getWeatherDetails } from './MatchCard';

interface PreMatchModalProps {
  fixture: Fixture | null;
  onClose: () => void;
  onWatch: (f: Fixture) => void;
  onSimulate?: (f: Fixture) => void;
  busy?: boolean;
}

function MissingLine({ players }: { players: Player[] }) {
  if (!players.length) return <p className="text-[12.5px] text-sage">Everyone available.</p>;
  return (
    <ul className="space-y-1">
      {players.map((p) => (
        <li key={p.player_id} className="text-[12.5px] text-ember font-semibold">
          {p.full_name} · {p.availability ?? ((p.injured_matches ?? 0) > 0 ? `injured ${p.injured_matches}` : `out ${p.suspended_matches}`)}
        </li>
      ))}
    </ul>
  );
}

function ProbableXi({ players, onOpen }: { players: Player[]; onOpen: (id: string) => void }) {
  return (
    <div className="grid grid-cols-1 gap-1">
      {players.map((p) => (
        <div key={p.player_id}>
          <button
            type="button"
            onClick={() => onOpen(p.player_id)}
            className="w-full flex items-center justify-between gap-2 px-2 py-1.5 text-left hover:bg-cardHover border border-transparent hover:border-line"
          >
            <span className="flex items-center gap-2 min-w-0">
              <span className={cx('text-[10px] font-mono font-semibold px-1.5 py-0.5', positionTone(p.category))}>
                {p.position}
              </span>
              <span className="truncate text-[13px] font-semibold text-bone">
                {p.full_name}
                {p.universe_wonderkid && <span className="ml-1.5 text-brass font-medium">U-14</span>}
              </span>
            </span>
            <span className={cx('font-mono font-bold text-[13px] shrink-0', ovrTone(p.ovr))}>{p.ovr}</span>
          </button>
          {p.grew_note && (
            <p className="px-2 pb-1 text-[11.5px] text-brass/90 font-medium">Growth spurt: {p.grew_note}</p>
          )}
        </div>
      ))}
    </div>
  );
}

export const PreMatchModal: React.FC<PreMatchModalProps> = ({ fixture, onClose, onWatch, onSimulate, busy }) => {
  const { openPlayer } = usePlayerSheet();
  const preview = fixture?.preview;
  const finished = fixture?.status === 'finished';
  const weatherInfo = getWeatherDetails(fixture?.weather);
  const derbyTitle = fixture?.derby_name || (fixture?.is_derby && fixture?.derby ? fixture.derby : null);
  const derbyHeat = typeof fixture?.derby_heat === 'number' ? fixture.derby_heat : 50;
  const isHighHeat = fixture?.is_high_heat_derby || derbyHeat >= 70;

  return (
    <Modal open={!!fixture} onClose={onClose} maxWidth="max-w-4xl">
      {fixture && (
        <>
          <ModalHeader
            title={fixtureCompetitionLabel(fixture)}
            subtitle={`${preview?.venue ?? fixture.home.home_stadium}${preview?.capacity ? ` · ${preview.capacity.toLocaleString()}` : ''}`}
            onClose={onClose}
          />
          <div className="flex-1 min-h-0 overflow-y-auto">
            <div className="px-6 pt-4 pb-5 space-y-5">
              {(weatherInfo || derbyTitle) && (
                <div className="flex items-center justify-between flex-wrap gap-2 pb-3 border-b border-line">
                  <div className="flex items-center gap-2 flex-wrap">
                    {weatherInfo && (
                      <span className="px-2.5 py-1 rounded-md border border-line bg-cardLight text-bone text-[12px] font-mono inline-flex items-center gap-1.5 shadow-sm">
                        <span>{weatherInfo.icon}</span>
                        <span>{weatherInfo.label}</span>
                      </span>
                    )}
                    {derbyTitle && (
                      <span
                        className={cx(
                          'px-2.5 py-1 rounded-md border font-semibold text-[11px] uppercase tracking-[0.08em] inline-flex items-center gap-1.5 transition-all',
                          isHighHeat
                            ? 'border-ember/70 bg-ember/15 text-ember animate-pulse shadow-[0_0_12px_rgba(224,86,36,0.3)]'
                            : 'border-ember/40 bg-ember/10 text-ember',
                        )}
                      >
                        <span>🔥</span>
                        <span>{derbyTitle} · {derbyHeat}° Heat</span>
                      </span>
                    )}
                  </div>
                  <span className="text-[11px] font-mono text-sage">Match Conditions</span>
                </div>
              )}

              {preview?.kickoff_note && (
                <p className="text-[15px] text-bone/90 leading-relaxed">{preview.kickoff_note}</p>
              )}

              <div className="flex items-center justify-between gap-3">
                <div className="flex items-center gap-3 flex-1 min-w-0">
                  <ClubCrest club={fixture.home} size={52} />
                  <div className="min-w-0">
                    <p className="font-semibold text-[16px] text-bone truncate">{fixture.home.club_name}</p>
                    <p className="text-[12px] text-sage font-mono mt-0.5">
                      {preview?.home_pos ? `${preview.home_pos}th · ${preview.home_pts} pts` : fixture.home.short_name}
                      {preview ? ` · XI ${preview.home_xi_avg}` : ''}
                    </p>
                    <div className="mt-1.5">
                      <FormPips form={preview?.home_form ?? fixture.home.form ?? []} size="sm" />
                    </div>
                  </div>
                </div>
                <span className="text-sage font-display text-[22px] font-semibold shrink-0 px-2">vs</span>
                <div className="flex items-center justify-end gap-3 flex-1 min-w-0">
                  <div className="min-w-0 text-right">
                    <p className="font-semibold text-[16px] text-bone truncate">{fixture.away.club_name}</p>
                    <p className="text-[12px] text-sage font-mono mt-0.5">
                      {preview?.away_pos ? `${preview.away_pos}th · ${preview.away_pts} pts` : fixture.away.short_name}
                      {preview ? ` · XI ${preview.away_xi_avg}` : ''}
                    </p>
                    <div className="mt-1.5 flex justify-end">
                      <FormPips form={preview?.away_form ?? fixture.away.form ?? []} size="sm" />
                    </div>
                  </div>
                  <ClubCrest club={fixture.away} size={52} />
                </div>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {[
                  { club: fixture.home, manager: fixture.home.manager, missing: preview?.home_missing ?? [], xi: preview?.home_xi ?? [] },
                  { club: fixture.away, manager: fixture.away.manager, missing: preview?.away_missing ?? [], xi: preview?.away_xi ?? [] },
                ].map(({ club, manager, missing, xi }) => (
                  <div key={club.club_id} className="border border-line bg-ink/40 p-3">
                    <div className="flex items-center justify-between gap-2 mb-2">
                      <p className="font-semibold text-[14px] text-bone truncate">{club.club_name}</p>
                      <span className="font-mono text-[11px] text-sage border border-line px-2 py-0.5">4-3-3</span>
                    </div>
                    {manager && (
                      <p className="text-[12.5px] text-sage mb-2">
                        {manager.name} · {manager.tactic} · {manager.style}
                      </p>
                    )}
                    <p className="text-[11px] font-mono uppercase tracking-[0.1em] text-sage mb-1">Probable XI</p>
                    <ProbableXi players={xi} onOpen={openPlayer} />
                    <div className="mt-3 pt-2 border-t border-line">
                      <p className="text-[11px] font-mono uppercase tracking-[0.1em] text-sage mb-1 flex items-center gap-1.5">
                        <ShieldAlert size={11} /> Missing
                      </p>
                      <MissingLine players={missing} />
                    </div>
                  </div>
                ))}
              </div>

              {(fixture.head_to_head?.length ?? 0) > 0 && (
                <div>
                  <p className="eyebrow mb-2">Head to head</p>
                  <div className="space-y-1">
                    {fixture.head_to_head!.map((h) => (
                      <p key={h.id} className="font-mono text-[12px] text-sage">
                        MW {h.matchweek} · {h.competition === 'ucl' ? 'Champions Cup' : h.competition === 'super-cup' ? 'Super Cup' : 'League'} · {h.home_goals}–{h.away_goals}
                      </p>
                    ))}
                  </div>
                </div>
              )}

              {!finished && (
                <div className="flex items-center gap-2 pt-1">
                  <button
                    onClick={() => {
                      soundManager.playClick();
                      onWatch(fixture);
                    }}
                    className="flex-1 px-4 py-3 bg-bone hover:bg-[#fff6dc] text-ink text-[14px] font-semibold inline-flex items-center justify-center gap-1.5"
                  >
                    <Eye size={14} /> Watch live
                  </button>
                  {onSimulate && (
                    <button
                      onClick={() => onSimulate(fixture)}
                      disabled={busy}
                      className="flex-1 px-4 py-3 bg-cardLight hover:bg-cardHover text-bone border border-line text-[14px] font-semibold inline-flex items-center justify-center gap-1.5 disabled:opacity-50"
                    >
                      <Zap size={14} /> {busy ? 'Simulating…' : 'Simulate'}
                    </button>
                  )}
                </div>
              )}
            </div>
          </div>
        </>
      )}
    </Modal>
  );
};
