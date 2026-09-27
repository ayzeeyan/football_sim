import React, { useEffect, useState } from 'react';
import { Landmark } from 'lucide-react';
import type { Club } from '../../../types';
import { fetchClubProfile, setClubBoard } from '../../../services/api';
import { Card, LoadingState } from '../../ui/ui';
import { formatEUR } from '../../../lib/format';

const BOARD_OBJECTIVES = [
  'Title challenge',
  'Champions League qualification',
  'European qualification',
  'Top-half finish',
  'Survival',
  'Competitive finish',
] as const;

/**
 * Board & finance control (Tier B, B4): the viewer sets one club's transfer
 * budget, wage cap, and board expectation. The domain invariants bind on the
 * server: budget never exceeds the balance, cap never sits below the wage
 * bill.
 */
export const BoardControlPanel: React.FC<{
  club: Club | null;
  onToast: (message: string) => void;
}> = ({ club, onToast }) => {
  const [balance, setBalance] = useState<number>(0);
  const [budget, setBudget] = useState('');
  const [wageCap, setWageCap] = useState('');
  const [objective, setObjective] = useState('');
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!club) return;
    let cancelled = false;
    setLoading(true);
    fetchClubProfile(club.club_id)
      .then((profile) => {
        if (cancelled || !profile) return;
        const c = profile.club;
        setBalance(c?.finances?.balance ?? 0);
        setBudget(String(c?.finances?.transfer_budget ?? 0));
        setWageCap(String(c?.finances?.wage_cap ?? 0));
        setObjective(c?.board_objective ?? 'Competitive finish');
      })
      .catch(() => undefined)
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [club?.club_id]);

  if (!club) return null;

  const handleSave = async () => {
    if (saving) return;
    setSaving(true);
    const res = await setClubBoard(club.club_id, {
      transfer_budget: budget === '' ? undefined : Number(budget),
      wage_cap: wageCap === '' ? undefined : Number(wageCap),
      board_objective: objective === '' ? undefined : objective,
    });
    setSaving(false);
    if (res.ok) {
      onToast('Board settings saved.');
    } else {
      onToast(res.message ?? 'The board rejected the request.');
    }
  };

  return (
    <Card className="p-4 sm:p-5">
      <div className="flex items-center justify-between gap-3">
        <p className="match-section-title flex items-center gap-2">
          <Landmark size={15} aria-hidden="true" /> Board &amp; finance control
        </p>
        <span className="text-[10px] uppercase tracking-[0.12em] text-sage">{club.short_name}</span>
      </div>
      <p className="mt-2 text-[11px] leading-relaxed text-sage">
        Set the transfer budget, the wage cap, and the board's expectation. The budget can never exceed the
        available balance ({formatEUR(balance)}) and the cap can never sit below the committed wage bill.
      </p>

      {loading && <LoadingState message="Reading the club's books…" />}

      {!loading && (
        <div className="mt-4 grid grid-cols-1 gap-3 md:grid-cols-3">
          <label className="rounded-md border border-line bg-black/20 p-3">
            <span className="block text-[10px] font-bold uppercase tracking-[0.12em] text-sage">Transfer budget</span>
            <input
              type="number"
              min={0}
              max={balance}
              value={budget}
              onChange={(e) => setBudget(e.target.value)}
              className="mt-2 w-full border border-line bg-ink px-2 py-1.5 text-right text-[12px] text-bone"
              aria-label="Transfer budget in EUR"
            />
            <span className="mt-1 block font-mono text-[10px] text-sage">{formatEUR(Number(budget) || 0)}</span>
          </label>
          <label className="rounded-md border border-line bg-black/20 p-3">
            <span className="block text-[10px] font-bold uppercase tracking-[0.12em] text-sage">Wage cap</span>
            <input
              type="number"
              min={0}
              value={wageCap}
              onChange={(e) => setWageCap(e.target.value)}
              className="mt-2 w-full border border-line bg-ink px-2 py-1.5 text-right text-[12px] text-bone"
              aria-label="Wage cap in EUR"
            />
            <span className="mt-1 block font-mono text-[10px] text-sage">{formatEUR(Number(wageCap) || 0)}</span>
          </label>
          <label className="rounded-md border border-line bg-black/20 p-3">
            <span className="block text-[10px] font-bold uppercase tracking-[0.12em] text-sage">Board objective</span>
            <select
              value={objective}
              onChange={(e) => setObjective(e.target.value)}
              className="mt-2 w-full border border-line bg-ink px-2 py-1.5 text-[12px] text-bone"
              aria-label="Board objective"
            >
              {BOARD_OBJECTIVES.map((o) => (
                <option key={o} value={o}>{o}</option>
              ))}
            </select>
          </label>
        </div>
      )}

      <button
        type="button"
        onClick={() => void handleSave()}
        disabled={saving || loading}
        className="mt-4 min-h-9 border border-brass/40 bg-brass/10 px-4 text-[12px] font-semibold text-brass hover:bg-brass/20 disabled:opacity-50"
      >
        {saving ? 'Saving…' : 'Save board settings'}
      </button>
    </Card>
  );
};
