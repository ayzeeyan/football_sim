import React from 'react';
import type { BatchSimResult } from '../types';
import { Modal, ModalHeader } from './ui/ui';
import { usePlayerSheet } from './PlayerSheet';

interface MatchweekDigestModalProps {
  open: boolean;
  data: BatchSimResult | null;
  onClose: () => void;
}

export const MatchweekDigestModal: React.FC<MatchweekDigestModalProps> = ({ open, data, onClose }) => {
  const { openPlayer } = usePlayerSheet();
  if (!data) return null;
  const digest = data.digests?.[data.digests.length - 1] ?? null;
  const title = digest ? `Matchweek ${digest.matchweek} digest` : 'Universe advanced';

  return (
    <Modal open={open} onClose={onClose} maxWidth="max-w-5xl">
      <ModalHeader
        title={title}
        subtitle={
          digest
            ? `${digest.calendar_label} · ${data.weeks_advanced > 1 ? `${data.weeks_advanced} weeks simulated · showing latest` : `${digest.played} fixtures resolved`}`
            : data.message
        }
        onClose={onClose}
      />
      <div className="overflow-y-auto p-5 space-y-5">
        {!digest ? (
          <div className="border border-line bg-cardLight/50 p-5 text-[14px] text-sage">
            {data.message || 'The autonomous universe advanced without a league matchweek digest.'}
          </div>
        ) : (
          <>
            {digest.upset_of_the_week && (
              <div className="border border-brass/40 bg-brass/[0.07] p-4">
                <p className="eyebrow !text-brass">Upset of the week</p>
                <p className="mt-1 text-bone font-semibold">
                  {digest.upset_of_the_week.home_short} {digest.upset_of_the_week.home_goals}–{digest.upset_of_the_week.away_goals} {digest.upset_of_the_week.away_short}
                </p>
                {digest.upset_of_the_week.upset_label && <p className="text-[12px] text-sage mt-1">{digest.upset_of_the_week.upset_label}</p>}
              </div>
            )}

            <section>
              <div className="flex items-center justify-between mb-2">
                <h3 className="font-display font-semibold text-bone">Results</h3>
                <span className="text-[11px] font-mono text-sage">{digest.results.length} matches</span>
              </div>
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-2">
                {(digest.results ?? []).map((match) => (
                  <article key={match.fixture_id} className="border border-line bg-cardLight/45 p-3">
                    <div className="flex items-center gap-3">
                      <span className="text-[12px] text-sage w-12 text-right">{match.home_short}</span>
                      <span className="font-mono text-lg font-bold text-bone">{match.home_goals}–{match.away_goals}</span>
                      <span className="text-[12px] text-sage">{match.away_short}</span>
                      {match.is_upset && <span className="ml-auto text-[10px] font-mono uppercase text-brass">Upset</span>}
                    </div>
                    {match.penalty_score && match.penalty_score.length === 2 && (
                      <p className="text-[10px] font-mono text-sage mt-1">Penalties {match.penalty_score[0]}–{match.penalty_score[1]}</p>
                    )}
                    <div className="mt-2 text-[11px] text-sage space-y-1">
                      {(match.scorers ?? []).length > 0 && (
                        <p>Scorers: {match.scorers.map((e) => `${e.name} ${e.minute}'`).join(' · ')}</p>
                      )}
                      {match.motm && <p>MOTM: <span className="text-bone">{match.motm}</span></p>}
                      {(match.red_cards ?? []).length > 0 && <p className="text-[#D89A84]">Red: {match.red_cards.join(' · ')}</p>}
                    </div>
                  </article>
                ))}
              </div>
            </section>

            <section>
              <h3 className="font-display font-semibold text-bone mb-2">Table movement</h3>
              <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-2">
                {(digest.table_movement ?? []).map((club) => (
                  <div key={club.club_id} className="border border-line bg-ink/35 px-3 py-2 flex items-center gap-2">
                    <span className="font-mono text-[12px] text-bone">{club.after}</span>
                    <span className="text-[12px] text-sage truncate">{club.short_name}</span>
                    <span className={`ml-auto font-mono text-[11px] ${club.delta > 0 ? 'text-pitchtone' : club.delta < 0 ? 'text-ember' : 'text-sage'}`}>
                      {club.delta > 0 ? `▲${club.delta}` : club.delta < 0 ? `▼${Math.abs(club.delta)}` : '—'}
                    </span>
                  </div>
                ))}
              </div>
            </section>

            <section>
              <h3 className="font-display font-semibold text-bone mb-2">Prodigy watch</h3>
              {(digest.wonderkid_highlights ?? []).length === 0 ? (
                <p className="text-[13px] text-sage">No franchise prodigy registered a notable event in this matchweek.</p>
              ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
                  {digest.wonderkid_highlights.map((wk) => (
                    <button key={wk.player_id} onClick={() => openPlayer(wk.player_id)} className="text-left border border-brass/25 bg-brass/[0.04] p-3 hover:border-brass/50">
                      <div className="flex items-center justify-between gap-2">
                        <span className="font-semibold text-[13px] text-bone">{wk.full_name}</span>
                        <span className="font-mono text-[11px] text-brass">{wk.goals}G · {wk.assists}A · {wk.ovr} OVR</span>
                      </div>
                      <p className="text-[11px] text-sage mt-0.5">{wk.club_short} · age {wk.age}</p>
                      {wk.note && <p className="text-[11px] text-sage mt-1">{wk.note}</p>}
                    </button>
                  ))}
                </div>
              )}
            </section>

            {(digest.manager_events ?? []).length > 0 && (
              <section>
                <h3 className="font-display font-semibold text-bone mb-2">Manager carousel</h3>
                <div className="space-y-2">
                  {digest.manager_events.map((event, index) => (
                    <div key={`${event.club_id}-${event.matchweek}-${index}`} className="border border-ember/25 bg-ember/[0.04] p-3">
                      <p className="text-[13px] text-bone font-semibold">{event.club_name}: {event.old_manager} out · {event.new_manager} in</p>
                      <p className="text-[11px] text-sage mt-1">{event.old_style} → {event.new_style} · {event.reason}</p>
                    </div>
                  ))}
                </div>
              </section>
            )}
          </>
        )}
      </div>
    </Modal>
  );
};
