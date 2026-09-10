import React, { useEffect, useState } from 'react';
import { RotateCcw, Shuffle } from 'lucide-react';
import { fetchDefaultHomes, previewCareerShuffle, startNewCareer, type ProdigyDraw } from '../services/api';
import { Modal, ModalHeader, PrimaryButton } from './ui/ui';
import { cx } from '../lib/format';
import { soundManager } from '../audio/webAudio';
import { getClubCrestUrlByShort } from '../lib/clubLogos';

interface NewCareerModalProps {
  open: boolean;
  onClose: () => void;
  onStarted: (message: string) => void;
}

export const NewCareerModal: React.FC<NewCareerModalProps> = ({ open, onClose, onStarted }) => {
  const [shuffle, setShuffle] = useState(false);
  const [draw, setDraw] = useState<ProdigyDraw | null>(null);
  const [busy, setBusy] = useState(false);
  const [loadingDraw, setLoadingDraw] = useState(false);

  const loadDraw = (nextShuffle: boolean) => {
    setLoadingDraw(true);
    const req = nextShuffle ? previewCareerShuffle() : fetchDefaultHomes();
    req
      .then((res) => {
        setDraw(res);
        // Second (and later) careers pre-fill the last draw; reroll stays free.
        if (!nextShuffle && res.from_last) setShuffle(!!res.shuffle);
      })
      .finally(() => setLoadingDraw(false));
  };

  useEffect(() => {
    if (!open) return;
    loadDraw(false);
  }, [open]);

  const pickShuffle = (on: boolean) => {
    soundManager.playClick();
    setShuffle(on);
    loadDraw(on);
  };

  const handleStart = async () => {
    soundManager.playWhistle();
    setBusy(true);
    try {
      const res = await startNewCareer(shuffle, draw?.homes);
      if (res.status === 'error') {
        onStarted(res.message);
        return;
      }
      onStarted(res.message);
      onClose();
    } finally {
      setBusy(false);
    }
  };

  return (
    <Modal open={open} onClose={onClose} maxWidth="max-w-xl">
      <ModalHeader
        title="New career"
        subtitle="This wipes all-time stats, growth, values, the table, cups and history. 2026–27 matchweek 1."
        onClose={onClose}
      />
      <div className="p-5 space-y-4 overflow-y-auto">
        <div className="grid grid-cols-2 gap-2">
          <button
            type="button"
            onClick={() => pickShuffle(false)}
            aria-pressed={!shuffle}
            className={cx(
              'px-3 py-3 rounded-xl border text-left transition-colors',
              !shuffle ? 'border-brass bg-brass/[0.08]' : 'border-line bg-ink/40 hover:border-sage/50',
            )}
          >
            <p className="font-semibold text-[14px] text-bone">Keep homes</p>
            <p className="text-[12px] text-sage mt-1">Each prodigy stays at his usual Super League club.</p>
          </button>
          <button
            type="button"
            onClick={() => pickShuffle(true)}
            aria-pressed={shuffle}
            className={cx(
              'px-3 py-3 rounded-xl border text-left transition-colors',
              shuffle ? 'border-brass bg-brass/[0.08]' : 'border-line bg-ink/40 hover:border-sage/50',
            )}
          >
            <p className="font-semibold text-[14px] text-bone flex items-center gap-1.5">
              <Shuffle size={14} /> Shuffle teams
            </p>
            <p className="text-[12px] text-sage mt-1">The twelve kids are dealt across the twelve clubs. One each.</p>
          </button>
        </div>

        <div className="rounded-xl border border-line bg-ink/40 overflow-hidden">
          <div className="px-3 py-2 border-b border-line flex items-center justify-between">
            <p className="eyebrow">{shuffle ? 'This draw' : draw?.from_last ? 'Last draw' : 'Default homes'}</p>
            {shuffle && (
              <button
                type="button"
                onClick={() => pickShuffle(true)}
                disabled={loadingDraw || busy}
                className="text-[12px] font-semibold text-brass hover:text-bone inline-flex items-center gap-1 disabled:opacity-50"
              >
                <RotateCcw size={12} /> Reroll
              </button>
            )}
          </div>
          <div className="max-h-[40vh] overflow-y-auto divide-y divide-line/70">
            {loadingDraw && <p className="px-3 py-6 text-center text-[13px] text-sage">Dealing the twelve…</p>}
            {!loadingDraw &&
              draw?.draw.map((row) => {
                const crest = getClubCrestUrlByShort(row.short_name);
                return (
                  <div key={row.full_name} className="px-3 py-2 flex items-center gap-2.5">
                    {crest ? (
                      <img src={crest} alt="" className="w-7 h-7 rounded-md bg-bone object-contain p-0.5 border border-bone/20" />
                    ) : (
                      <span className="w-7 h-7 rounded-md border border-line bg-cardLight" />
                    )}
                    <div className="min-w-0 flex-1">
                      <p className="text-[13px] font-semibold text-bone truncate">{row.full_name}</p>
                      <p className="text-[11px] font-mono text-sage">{row.position} · age {row.age}</p>
                    </div>
                    <p className="text-[13px] font-semibold text-bone truncate text-right">{row.club_name}</p>
                  </div>
                );
              })}
          </div>
        </div>

        <div className="flex items-center justify-end gap-2 pt-1">
          <button
            type="button"
            onClick={onClose}
            disabled={busy}
            className="px-4 py-2 text-[13px] font-semibold text-sage hover:text-bone"
          >
            Cancel
          </button>
          <PrimaryButton onClick={() => void handleStart()} disabled={busy || loadingDraw || !draw}>
            {busy ? 'Starting…' : 'Start new career'}
          </PrimaryButton>
        </div>
      </div>
    </Modal>
  );
};
