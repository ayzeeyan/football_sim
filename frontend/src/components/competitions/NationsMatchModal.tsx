import React, { useEffect, useState } from 'react';
import { Globe2, Zap } from 'lucide-react';
import type { Fixture, NationsCupFixture, NationsCupResponse } from '../../types';
import { fetchNationsCup, fetchNationsFixture, simulateNationsFixture } from '../../services/api';
import { useAsyncData } from '../../hooks/useAsyncData';
import { cx } from '../../lib/format';
import { soundManager } from '../../audio/webAudio';
import { ClubCrest, Modal } from '../ui/ui';
import { usePlayerSheet } from '../clubs/PlayerSheet';
import { PostMatchBroadcast } from '../postmatch/PostMatchBroadcast';

/**
 * One international fixture, end to end: pre-match briefing with squads and
 * records while scheduled, the full broadcast report once played. Reuses the
 * club report components — the backend emits the same wire shape.
 */
export const NationsMatchModal: React.FC<{
  fixture: NationsCupFixture | null;
  onClose: () => void;
  onPlayed?: () => void;
}> = ({ fixture, onClose, onPlayed }) => {
  const { openPlayer } = usePlayerSheet();
  const [full, setFull] = useState<Fixture | null>(null);
  const [loading, setLoading] = useState(false);
  const [busy, setBusy] = useState(false);
  const { data: nations } = useAsyncData(fetchNationsCup, []);

  useEffect(() => {
    if (!fixture) {
      setFull(null);
      return;
    }
    let cancelled = false;
    setLoading(true);
    fetchNationsFixture(fixture.fixture_id ?? fixture.id)
      .then((row) => { if (!cancelled) setFull(row); })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, [fixture?.fixture_id, fixture?.id]);

  const play = async () => {
    if (!fixture || busy) return;
    setBusy(true);
    soundManager.playWhistle();
    try {
      const result = await simulateNationsFixture(fixture.fixture_id ?? fixture.id);
      if (!result || result.status !== 'finished') {
        soundManager.playClick();
        return;
      }
      setFull(result);
      onPlayed?.();
    } finally {
      setBusy(false);
    }
  };

  const teamOf = (id: string) => nations?.participants.find((team) => team.id === id) ?? null;
  const home = full?.home ?? null;
  const away = full?.away ?? null;

  return (
    <Modal open={!!fixture} onClose={onClose} maxWidth="max-w-7xl" fillViewport>
      {fixture && (full?.status === 'finished' ? (
        <PostMatchBroadcast
          fixture={full}
          homeClub={home}
          awayClub={away}
          onOpenPlayer={openPlayer}
          onBackToMatches={onClose}
          bounded
        />
      ) : (
        <div className="flex min-h-0 flex-1 flex-col overflow-y-auto">
          <div className="relative shrink-0 overflow-hidden border-b border-brass/20 bg-gradient-to-br from-[#1c4937] via-[#0d271b] to-[#06110c] px-4 py-5 sm:px-7">
            <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_50%_120%,rgba(243,230,196,0.16),transparent_50%)]" />
            <button type="button" onClick={onClose} aria-label="Close international match" className="absolute right-3 top-3 z-10 grid h-8 w-8 place-items-center rounded-sm border border-white/10 bg-black/20 text-sage hover:text-bone">×</button>
            <div className="relative flex flex-wrap items-center justify-between gap-2 text-[10px] font-semibold uppercase tracking-[0.14em] text-sage">
              <span className="inline-flex items-center gap-1.5"><Globe2 size={12} /> European Nations Cup · {fixture.stage}</span>
              <span className="rounded-full border border-brass/30 bg-brass/10 px-3 py-1 text-brass">Matchweek {fixture.matchweek}</span>
            </div>
            <div className="relative mx-auto mt-4 grid max-w-3xl grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] items-center gap-3 sm:gap-8">
              {[{ team: teamOf(fixture.home_id), align: 'right' as const }, { team: teamOf(fixture.away_id), align: 'left' as const }].map(({ team, align }) => (
                <div key={align} className={cx('flex min-w-0 flex-col items-center text-center', align === 'right' ? 'order-1' : 'order-3')}>
                  <span className="grid h-[64px] w-[64px] place-items-center rounded-full border border-brass/30 bg-brass/10 font-display text-[24px] font-bold text-brass">
                    {team?.country?.slice(0, 3).toUpperCase() ?? '—'}
                  </span>
                  <strong className="mt-2 block max-w-full truncate font-display text-[19px] font-bold text-bone sm:text-[24px]">{team?.name ?? fixture.home_id}</strong>
                  <span className="mt-1 text-[11px] text-sage">FIFA-style rating {team?.rating ?? '—'}</span>
                </div>
              ))}
              <div className="order-2 min-w-[70px] text-center">
                <p className="rounded-full border border-white/10 bg-black/25 px-3 py-1 text-[9px] font-bold uppercase tracking-[0.16em] text-sage">Upcoming</p>
                <div className="mt-2 font-display text-[26px] font-bold text-brass">VS</div>
              </div>
            </div>
          </div>

          <div className="p-4 sm:p-6">
            {loading && <p className="text-center text-[13px] text-sage">Loading international fixture…</p>}
            {!loading && (
              <>
                <div className="mx-auto flex max-w-md flex-col items-center gap-3">
                  <button type="button" disabled={busy} onClick={() => void play()} className="gold-btn min-h-10">
                    <Zap size={15} aria-hidden="true" /> {busy ? 'Simulating…' : 'Simulate international'}
                  </button>
                  <p className="text-center text-[11px] leading-relaxed text-sage">
                    International results never affect club standings, fatigue, or player statistics.
                  </p>
                </div>

                {nations && (
                  <div className="mt-6 grid grid-cols-1 gap-4 lg:grid-cols-2">
                    {[fixture.home_id, fixture.away_id].map((teamID) => {
                      const team = teamOf(teamID);
                      if (!team) return null;
                      const record = nations.table.find((row) => row.team_id === teamID);
                      return (
                        <section key={teamID} className="console-card overflow-hidden">
                          <div className="flex items-center justify-between gap-3 border-b border-line bg-cardLight/50 px-4 py-3">
                            <div className="min-w-0">
                              <h4 className="truncate font-display text-[16px] font-bold text-bone">{team.name}</h4>
                              <p className="text-[10px] uppercase tracking-[0.12em] text-sage">{team.country} · {team.players.length} players</p>
                            </div>
                            {record && (
                              <span className="shrink-0 font-mono text-[11px] text-sage">
                                {record.won}W {record.drawn}D {record.lost}L · {record.points} pts
                              </span>
                            )}
                          </div>
                          <ul className="max-h-64 divide-y divide-line/60 overflow-y-auto">
                            {team.players.map((player) => (
                              <li key={player.player_id} className="flex items-center justify-between gap-2 px-4 py-1.5 text-[11px]">
                                <span className="min-w-0 truncate text-bone">{player.full_name}<span className="text-sage"> · {player.position}</span></span>
                                <span className="shrink-0 font-mono text-sage">{player.ovr} · {player.age}y</span>
                              </li>
                            ))}
                          </ul>
                        </section>
                      );
                    })}
                  </div>
                )}
              </>
            )}
          </div>
        </div>
      ))}
    </Modal>
  );
};
