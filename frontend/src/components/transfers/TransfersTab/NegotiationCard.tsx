import React, { useState } from 'react';
import { Flame, Sparkle } from 'lucide-react';
import { soundManager } from '../../../audio/webAudio';
import type { TransferNegotiation } from '../../../types';
import { cx } from '../../../lib/format';
import { ClubDot, ProgressBar } from '../../ui/ui';
import { STAGE_LABELS, STAGE_TONES } from './helpers';

export type NegotiationRespondFn = (neg: TransferNegotiation, action: 'improve' | 'withdraw', amount?: number) => void;

export const NegotiationCard: React.FC<{
  neg: TransferNegotiation;
  onPlayerClick: (id: string) => void;
  /** Tier B (B3): when provided, the viewer can improve or withdraw. */
  onRespond?: NegotiationRespondFn;
}> = ({ neg, onPlayerClick, onRespond }) => {
  const [amount, setAmount] = useState('');
  const closed = neg.stage_name === 'COLLAPSED' || neg.stage_name === 'COMPLETED';

  return (
    <div
      className={cx(
        'p-4 rounded-xl border transition-colors',
        neg.is_hijacked ? 'border-ember/50 bg-ember/[0.06]' : neg.is_wonderkid ? 'border-brass/45 bg-brass/[0.06]' : 'border-line bg-ink/40',
      )}
    >
      <div className="flex items-center justify-between gap-2">
        <button
          type="button"
          onClick={() => {
            soundManager.playClick();
            onPlayerClick(neg.player.player_id);
          }}
          className="text-left font-semibold text-[14px] text-bone hover:text-brass transition-colors min-w-0 flex items-center gap-2"
        >
          <span className="truncate">{neg.player.full_name}</span>
          <span className="text-[12px] text-sage font-mono shrink-0">[{neg.player.ovr} · {neg.player.position}]</span>
          {neg.is_wonderkid && (
            <span className="px-1.5 py-0.2 rounded bg-brass/20 text-brass text-[10px] font-mono font-bold flex items-center gap-0.5 shrink-0">
              <Sparkle size={10} /> PRODIGY
            </span>
          )}
        </button>
        <div className="text-[#A9CDBB] font-bold font-mono text-[14px] shrink-0">
          Bid {neg.formatted_bid}
          {neg.formatted_asking_price && !closed && (
            <span className="ml-2 text-[11px] text-sage">asks {neg.formatted_asking_price}</span>
          )}
        </div>
      </div>
      <div className="text-[13px] text-sage mt-2 flex items-center gap-2 flex-wrap">
        <span className="inline-flex items-center gap-1.5"><ClubDot club={neg.seller} size={22} />{neg.seller.club_name}</span>
        <span aria-hidden className="text-sage/60">→</span>
        <span className="text-bone/85 font-semibold inline-flex items-center gap-1.5"><ClubDot club={neg.buyer} size={22} />{neg.buyer.club_name}</span>
        {neg.is_hijacked && (
          <span className="text-[#D89A84] font-semibold text-[11px] flex items-center gap-1 ml-auto uppercase tracking-[0.06em]">
            <Flame size={12} className="text-ember" /> Rival bid from {neg.original_buyer?.short_name}
          </span>
        )}
      </div>
      <div className="mt-3.5 space-y-1.5">
        <div className="flex justify-between font-mono text-[11px] font-semibold">
          <span className="text-sage uppercase tracking-[0.08em]">
            Stage {neg.stage_index}/5 · {STAGE_LABELS[neg.stage_index] || neg.stage_name}
          </span>
          <span className="text-bone/80">{neg.progress_pct}%</span>
        </div>
        <ProgressBar pct={neg.progress_pct} toneClass={STAGE_TONES[neg.stage_index] || 'bg-brass'} />
      </div>

      {onRespond && !closed && (
        <div className="mt-3 flex flex-wrap items-center gap-2 border-t border-line/60 pt-3">
          <input
            type="number"
            min={0}
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
            placeholder="Improved offer (EUR)"
            className="min-w-0 flex-1 border border-line bg-ink px-2 py-1.5 text-[12px] text-bone"
            aria-label={`Improved offer for ${neg.player.full_name}`}
          />
          <button
            type="button"
            onClick={() => {
              soundManager.playClick();
              onRespond(neg, 'improve', Number(amount) || 0);
            }}
            disabled={amount === ''}
            className="min-h-8 border border-brass/40 bg-brass/10 px-2.5 text-[11px] font-semibold text-brass hover:bg-brass/20 disabled:opacity-50"
          >
            Improve offer
          </button>
          <button
            type="button"
            onClick={() => {
              soundManager.playClick();
              onRespond(neg, 'withdraw');
            }}
            className="min-h-8 border border-line px-2.5 text-[11px] text-sage hover:text-bone"
          >
            Withdraw
          </button>
        </div>
      )}
    </div>
  );
};
