import React from 'react';
import type { Club } from '../types';
import { Landmark, ShieldCheck } from 'lucide-react';
import { Card } from './ui/ui';

interface ClubIdentityPanelProps {
  club: Club;
}

const ratingLabel = (value: number): string => {
  if (value >= 90) return 'Elite';
  if (value >= 80) return 'Very high';
  if (value >= 70) return 'High';
  if (value >= 55) return 'Balanced';
  if (value >= 40) return 'Moderate';
  return 'Low';
};

const IdentityMeter: React.FC<{ label: string; value: number; hint: string }> = ({ label, value, hint }) => (
  <div className="rounded-xl border border-line bg-ink/40 p-3" title={hint}>
    <div className="flex items-center justify-between gap-2">
      <span className="text-[11px] font-semibold uppercase tracking-[0.07em] text-sage">{label}</span>
      <span className="font-mono text-[12px] font-bold text-bone">{value}</span>
    </div>
    <div className="mt-2 h-1.5 overflow-hidden rounded-full bg-cardLight">
      <div className="h-full rounded-full bg-brass" style={{ width: `${Math.max(0, Math.min(100, value))}%` }} />
    </div>
    <p className="mt-1.5 text-[10.5px] text-sage">{ratingLabel(value)}</p>
  </div>
);

export const ClubIdentityPanel: React.FC<ClubIdentityPanelProps> = ({ club }) => {
  const identity = club.identity;
  if (!identity) return null;

  const warchest = club.formatted_transfer_warchest ?? club.manager?.formatted_budget ?? '—';
  const balance = club.finances?.balance;
  const formattedBalance = typeof balance === 'number'
    ? new Intl.NumberFormat('en', { style: 'currency', currency: 'EUR', notation: 'compact', maximumFractionDigits: 1 }).format(balance)
    : '—';

  const rows = [
    ['Historical prestige', identity.historical_prestige, 'Long-term stature; intentionally changes much more slowly than current reputation.'],
    ['Financial power', identity.financial_power, 'Structural financial strength used as one input to transfer budgets.'],
    ['Board patience', identity.board_patience, 'Higher patience gives managers more time through poor runs.'],
    ['Academy quality', identity.academy_quality, 'Existing club youth-development identity.'],
    ['Recruitment ambition', identity.recruitment_ambition, 'How ambitious the club is expected to be in recruitment.'],
    ['Youth preference', identity.youth_preference, 'How strongly the club identity leans toward younger players.'],
    ['Transfer aggression', identity.transfer_aggressiveness, 'How assertive the club tends to be in the market.'],
    ['Selling tendency', identity.selling_tendency, 'How willing the club is to sell when offers arrive.'],
  ] as const;

  return (
    <Card>
      <div className="flex flex-wrap items-start justify-between gap-4 border-b border-line pb-4">
        <div>
          <p className="eyebrow flex items-center gap-1.5"><ShieldCheck size={13} /> Club identity</p>
          <h3 className="mt-1 font-display text-[19px] font-semibold text-bone">Current stature and operating profile</h3>
          <p className="mt-1 max-w-2xl text-[12px] text-sage">Identity is persistent simulation state. Reputation evolves from results; prestige remains the slower historical anchor.</p>
        </div>
        <div className="flex gap-2">
          <div className="rounded-xl border border-brass/40 bg-brass/[0.08] px-4 py-2 text-right">
            <p className="text-[10px] font-semibold uppercase tracking-[0.08em] text-brass">Reputation</p>
            <p className="font-mono text-[23px] font-bold text-bone">{identity.reputation}</p>
          </div>
          <div className="rounded-xl border border-line bg-ink/40 px-4 py-2 text-right">
            <p className="flex items-center justify-end gap-1 text-[10px] font-semibold uppercase tracking-[0.08em] text-sage"><Landmark size={11} /> Warchest</p>
            <p className="font-mono text-[18px] font-bold text-brass">{warchest}</p>
            <p className="font-mono text-[10px] text-sage">Balance {formattedBalance}</p>
          </div>
        </div>
      </div>

      <div className="mt-4 grid grid-cols-2 gap-2 md:grid-cols-4">
        {rows.map(([label, value, hint]) => <IdentityMeter key={label} label={label} value={value} hint={hint} />)}
      </div>
    </Card>
  );
};
