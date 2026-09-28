import React, { useCallback, useEffect, useState } from 'react';
import { Briefcase, Trophy, Users } from 'lucide-react';
import { acceptViewerJob, fetchViewerCareer, resignViewerJob, type ViewerCareer, type ViewerCareerClub } from '../../services/api';
import { cx } from '../../lib/format';
import { soundManager } from '../../audio/webAudio';

const EMPTY: ViewerCareer = { manager: null, job: null, clubs: [] };

export interface ManagerCareerPanelProps {
  onOpenClub?: (clubId: string) => void;
  onShowToast: (msg: string) => void;
}

/**
 * Tier B manager career (B5): the viewer can accept a club's dugout, be
 * sacked by the patience system, win trophies, and resign. Every other
 * club keeps running AI-first.
 */
export const ManagerCareerPanel: React.FC<ManagerCareerPanelProps> = ({ onOpenClub, onShowToast }) => {
  const [career, setCareer] = useState<ViewerCareer>(EMPTY);
  const [busy, setBusy] = useState(false);
  const [confirmResign, setConfirmResign] = useState(false);
  const [applying, setApplying] = useState(false);

  const reload = useCallback(() => {
    fetchViewerCareer().then(setCareer).catch(() => undefined);
  }, []);

  useEffect(() => { reload(); }, [reload]);

  const accept = async (club: ViewerCareerClub) => {
    if (busy) return;
    soundManager.playClick();
    setBusy(true);
    try {
      const res = await acceptViewerJob(club.club_id);
      if (res.status === 'error') {
        onShowToast(res.message ?? 'Could not accept the job.');
      } else {
        onShowToast(`You take charge of ${club.club_name}.`);
        setApplying(false);
        reload();
      }
    } finally {
      setBusy(false);
    }
  };

  const resign = async () => {
    if (busy) return;
    soundManager.playClick();
    setBusy(true);
    try {
      const res = await resignViewerJob();
      if (res.status === 'error') {
        onShowToast(res.message ?? 'Could not resign.');
      } else {
        onShowToast('You step down. The club appoint a successor.');
        setConfirmResign(false);
        reload();
      }
    } finally {
      setBusy(false);
    }
  };

  const manager = career.manager;
  const activeJob = manager != null && manager.club_id !== '';
  const jobSecurity = (career.job as { job_security?: string } | null)?.job_security;

  return (
    <section className="console-card p-5" data-manager-career-panel="true">
      <div className="flex items-center justify-between gap-2">
        <h3 className="font-display text-[16px] font-bold text-bone flex items-center gap-1.5">
          <Briefcase size={15} className="text-brass" aria-hidden="true" /> Manager career
        </h3>
        {activeJob && jobSecurity != null && (
          <span className={cx('text-[10px] font-semibold uppercase tracking-[0.12em]',
            jobSecurity === 'Safe' ? 'text-sage' : jobSecurity === 'Hot Seat' ? 'text-brass' : 'text-red-400')}>
            {jobSecurity}
          </span>
        )}
      </div>

      {activeJob && manager ? (
        <>
          <p className="mt-1 text-[11px] text-sage">
            In charge since matchweek {String(manager.hired_matchweek).padStart(2, '0')} of {manager.hired_season}.
          </p>
          <button
            type="button"
            onClick={onOpenClub ? () => onOpenClub(manager.club_id) : undefined}
            disabled={!onOpenClub}
            className="mt-3 block w-full text-left"
          >
            <span className="font-display text-[18px] font-bold text-bone hover:text-brass">
              {career.clubs.find((c) => c.club_id === manager.club_id)?.club_name ?? manager.club_id}
            </span>
          </button>
          <dl className="mt-3 grid grid-cols-3 gap-2 text-center">
            <div className="rounded-md border border-line bg-cardLight p-2">
              <dt className="text-[10px] uppercase tracking-[0.1em] text-sage">Jobs</dt>
              <dd className="font-display text-[16px] font-bold text-bone">{(manager.history ?? []).length}</dd>
            </div>
            <div className="rounded-md border border-line bg-cardLight p-2">
              <dt className="text-[10px] uppercase tracking-[0.1em] text-sage">Sackings</dt>
              <dd className="font-display text-[16px] font-bold text-bone">{manager.sackings}</dd>
            </div>
            <div className="rounded-md border border-line bg-cardLight p-2">
              <dt className="text-[10px] uppercase tracking-[0.1em] text-sage">Trophies</dt>
              <dd className="font-display text-[16px] font-bold text-bone">{(manager.trophies ?? []).length}</dd>
            </div>
          </dl>
          {(manager.trophies ?? []).length > 0 && (
            <ul className="mt-3 space-y-1">
              {(manager.trophies ?? []).map((trophy) => (
                <li key={trophy} className="flex items-center gap-1.5 text-[12px] text-bone">
                  <Trophy size={12} className="text-brass" aria-hidden="true" /> {trophy}
                </li>
              ))}
            </ul>
          )}
          {confirmResign ? (
            <div className="mt-4 flex items-center gap-2">
              <span className="text-[12px] text-sage">Step down?</span>
              <button
                type="button"
                onClick={() => void resign()}
                disabled={busy}
                className="rounded-md border border-red-400/50 bg-red-400/10 px-2.5 py-1.5 text-[12px] font-semibold text-red-300 hover:bg-red-400/20"
              >
                Confirm
              </button>
              <button
                type="button"
                onClick={() => setConfirmResign(false)}
                className="rounded-md border border-line bg-cardLight px-2.5 py-1.5 text-[12px] font-semibold text-sage hover:text-bone"
              >
                Stay
              </button>
            </div>
          ) : (
            <div className="mt-4 flex items-center gap-2">
              <button
                type="button"
                onClick={() => setConfirmResign(true)}
                disabled={busy}
                className="rounded-md border border-line bg-cardLight px-2.5 py-1.5 text-[12px] font-semibold text-sage hover:border-brass/40 hover:text-bone"
              >
                Resign
              </button>
              <button
                type="button"
                onClick={() => setApplying((v) => !v)}
                className="rounded-md border border-line bg-cardLight px-2.5 py-1.5 text-[12px] font-semibold text-sage hover:border-brass/40 hover:text-bone"
              >
                {applying ? 'Close list' : 'Move clubs'}
              </button>
            </div>
          )}
        </>
      ) : (
        <p className="mt-1 text-[11px] text-sage">
          {manager
            ? `Between jobs. ${manager.sackings} sacking${manager.sackings === 1 ? '' : 's'} and ${(manager.trophies ?? []).length} trophy${(manager.trophies ?? []).length === 1 ? '' : 'ies'} so far.`
            : 'Unemployed. Pick a dugout below to start a managerial career.'}
        </p>
      )}

      {(applying || !activeJob) && (
        <div className="mt-4">
          <p className="mb-1.5 flex items-center gap-1 text-[10px] font-bold uppercase tracking-[0.12em] text-sage">
            <Users size={11} aria-hidden="true" /> Open dugouts
          </p>
          <ul className="max-h-56 space-y-0.5 overflow-y-auto">
            {career.clubs.map((club) => (
              <li key={club.club_id} className="flex items-center justify-between gap-2 rounded-md px-2 py-1.5 hover:bg-white/[0.04]">
                <button
                  type="button"
                  onClick={onOpenClub ? () => onOpenClub(club.club_id) : undefined}
                  disabled={!onOpenClub}
                  className="min-w-0 flex-1 text-left"
                >
                  <span className="block truncate text-[12px] font-semibold text-bone">{club.club_name}</span>
                  <span className="block truncate text-[10px] text-sage">
                    {club.league} · {club.manager_name ?? 'Vacant'}
                    {club.job_security ? ` · ${club.job_security}` : ''}
                  </span>
                </button>
                <button
                  type="button"
                  onClick={() => void accept(club)}
                  disabled={busy || (manager != null && manager.club_id === club.club_id)}
                  aria-label={`Take charge of ${club.club_name}`}
                  className="shrink-0 rounded-md border border-line bg-cardLight px-2 py-1 text-[11px] font-semibold text-sage hover:border-brass/40 hover:text-bone disabled:opacity-40"
                >
                  Take charge
                </button>
              </li>
            ))}
          </ul>
        </div>
      )}

      {manager && (manager.history ?? []).length > 0 && (
        <div className="mt-4">
          <p className="mb-1 text-[10px] font-bold uppercase tracking-[0.12em] text-sage">Career ledger</p>
          <ul className="space-y-0.5">
            {(manager.history ?? []).map((rec, i) => (
              <li key={`${rec.club_id}-${i}`} className="flex items-center justify-between gap-2 text-[11px]">
                <span className="truncate text-bone">{rec.club_name}</span>
                <span className="shrink-0 text-sage">
                  {rec.season} · {rec.outcome === 'active' ? 'in post' : rec.outcome}
                </span>
              </li>
            ))}
          </ul>
        </div>
      )}
    </section>
  );
};
