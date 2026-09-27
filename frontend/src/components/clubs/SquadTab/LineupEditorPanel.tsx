import React, { useEffect, useMemo, useState } from 'react';
import { ClipboardCheck, RotateCcw } from 'lucide-react';
import type { Club, Player } from '../../../types';
import { clearClubLineup, fetchClubXi, setClubLineup } from '../../../services/api';
import { Card, ClubCrest, PrimaryButton } from '../../ui/ui';
import { FormationPitch } from '../../matches/FormationPitch';
import { FORMATION_SLOTS, normalizeFormation, positionFit } from '../../../lib/tactics';
import { cx } from '../../../lib/format';

const FORMATIONS = ['4-3-3', '4-3-3 Attack', '4-2-3-1', '4-4-2'] as const;

/**
 * Lineup editor (Tier B, B1): the viewer sets one club's formation and
 * starting XI through the rigid tactical-slot contract. The AI fallback
 * applies whenever the saved lineup cannot be fielded.
 */
export const LineupEditorPanel: React.FC<{
  club: Club | null;
  squad: Player[];
  onToast: (message: string) => void;
  onOpenPlayer: (id: string) => void;
}> = ({ club, squad, onToast, onOpenPlayer }) => {
  const [formation, setFormation] = useState<string>('4-3-3');
  const [assignments, setAssignments] = useState<Record<string, string>>({});
  const [selectedSlot, setSelectedSlot] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [overrideActive, setOverrideActive] = useState(false);

  // Seed the editor from the club's current XI.
  useEffect(() => {
    if (!club) return;
    let cancelled = false;
    fetchClubXi(club.club_id)
      .then((xi) => {
        if (cancelled || xi.length === 0) return;
        const next: Record<string, string> = {};
        let form = '4-3-3';
        for (const row of xi) {
          const slot = (row as { starting_slot?: string }).starting_slot;
          if (slot) next[slot] = row.player_id;
        }
        // Derive the formation from the observed slot set.
        for (const candidate of FORMATIONS) {
          const slots = FORMATION_SLOTS[candidate] ?? [];
          if (slots.length === Object.keys(next).length && slots.every((s) => next[s])) {
            form = candidate;
            break;
          }
        }
        if (!cancelled) {
          setAssignments(next);
          setFormation(form);
        }
      })
      .catch(() => undefined);
    return () => {
      cancelled = true;
    };
  }, [club?.club_id]);

  const slots = FORMATION_SLOTS[normalizeFormation(formation)] ?? FORMATION_SLOTS['4-3-3'] ?? [];

  const assignedIDs = useMemo(() => new Set(Object.values(assignments)), [assignments]);
  const availableForSlot = useMemo(
    () => squad.filter((p) => !assignedIDs.has(p.player_id) || assignments[selectedSlot ?? ''] === p.player_id),
    [squad, assignedIDs, assignments, selectedSlot],
  );

  if (!club) return null;

  const pickPlayer = (playerID: string) => {
    if (!selectedSlot) return;
    setAssignments((prev) => ({ ...prev, [selectedSlot]: playerID }));
    setSelectedSlot(null);
  };

  const changeFormation = (next: string) => {
    setFormation(next);
    // Keep the players but re-seed slots for the new shape: unassigned slots
    // fall back to the first available squad players.
    const nextSlots = FORMATION_SLOTS[normalizeFormation(next)] ?? [];
    const used = new Set<string>();
    const seeded: Record<string, string> = {};
    for (const slot of nextSlots) {
      const existing = Object.values(assignments).find((id) => !used.has(id));
      if (existing) {
        used.add(existing);
        seeded[slot] = existing;
      }
    }
    for (const slot of nextSlots) {
      if (seeded[slot]) continue;
      const candidate = squad.find((p) => !used.has(p.player_id));
      if (candidate) {
        used.add(candidate.player_id);
        seeded[slot] = candidate.player_id;
      }
    }
    setAssignments(seeded);
    setSelectedSlot(null);
  };

  const handleSave = async () => {
    if (saving) return;
    const missing = slots.filter((slot) => !assignments[slot]);
    if (missing.length > 0) {
      onToast(`Assign every slot before saving (${missing.join(', ')}).`);
      return;
    }
    setSaving(true);
    const res = await setClubLineup(club.club_id, formation, assignments);
    setSaving(false);
    if (res.status === 'success') {
      setOverrideActive(true);
      onToast('Lineup saved. The XI is now under your control.');
    } else {
      onToast(res.message || 'The lineup was rejected.');
    }
  };

  const handleReset = async () => {
    if (saving) return;
    setSaving(true);
    const res = await clearClubLineup(club.club_id);
    setSaving(false);
    if (res.status === 'success') {
      setOverrideActive(false);
      onToast('Lineup cleared. The AI staff pick the XI again.');
    } else {
      onToast('Could not clear the lineup.');
    }
  };

  const pitchAssignments = slots
    .filter((slot) => assignments[slot])
    .map((slot) => {
      const player = squad.find((p) => p.player_id === assignments[slot]);
      return player ? { slot, player } : null;
    })
    .filter((row): row is { slot: string; player: Player } => row !== null);

  return (
    <Card className="p-4 sm:p-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p className="match-section-title flex items-center gap-2">
            <ClipboardCheck size={15} aria-hidden="true" /> Lineup control
          </p>
          <p className="mt-1 flex items-center gap-1.5 text-[12px] text-sage">
            <ClubCrest club={club} size={14} /> {club.club_name}
            {overrideActive ? ' · viewer lineup active' : ' · AI selection'}
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <select
            value={formation}
            onChange={(e) => changeFormation(e.target.value)}
            className="border border-line bg-ink px-2 py-1.5 text-[12px] text-bone"
            aria-label="Formation"
          >
            {FORMATION_SLOTS && FORMATIONS.map((f) => (
              <option key={f} value={f}>{f}</option>
            ))}
          </select>
          <PrimaryButton tone="green" onClick={() => void handleSave()} disabled={saving}>
            {saving ? 'Saving…' : 'Save lineup'}
          </PrimaryButton>
          <button
            type="button"
            onClick={() => void handleReset()}
            disabled={saving}
            className="flex min-h-9 items-center gap-1.5 border border-line px-3 text-[12px] font-semibold text-sage hover:text-bone disabled:opacity-50"
          >
            <RotateCcw size={13} aria-hidden="true" /> Back to AI
          </button>
        </div>
      </div>

      <p className="mt-2 text-[11px] leading-relaxed text-sage">
        Pick a slot, then choose a player. Every rigid tactical slot must be filled; an unavailable player
        invalidates the whole lineup and the club falls back to the AI XI.
      </p>

      <div className="mt-4 grid grid-cols-1 gap-4 lg:grid-cols-2">
        <div>
          <FormationPitch
            players={squad}
            formation={formation}
            assignments={pitchAssignments}
            onPlayerClick={(player) => {
              const slot = slots.find((s) => assignments[s] === player.player_id);
              if (slot) setSelectedSlot(slot);
              else onOpenPlayer(player.player_id);
            }}
          />
          {selectedSlot && (
            <p className="mt-2 text-center text-[12px] text-brass">
              Selecting for slot {selectedSlot} — pick a player from the list.
            </p>
          )}
        </div>

        <div>
          <p className="eyebrow">Slot assignments</p>
          <div className="mt-2 grid grid-cols-1 gap-1.5">
            {slots.map((slot) => {
              const player = squad.find((p) => p.player_id === assignments[slot]);
              const fit = player ? positionFit(player, slot) : undefined;
              return (
                <button
                  key={slot}
                  type="button"
                  onClick={() => setSelectedSlot(selectedSlot === slot ? null : slot)}
                  className={cx(
                    'flex items-center justify-between gap-3 rounded-md border px-3 py-2 text-left text-[12px]',
                    selectedSlot === slot ? 'border-brass bg-brass/10' : 'border-line bg-black/20 hover:border-brass/30',
                  )}
                >
                  <span className="font-mono font-bold text-sage">{slot}</span>
                  <span className="min-w-0 flex-1 truncate text-bone">
                    {player ? player.full_name : <span className="text-sage/70">Unassigned</span>}
                  </span>
                  <span className={cx('text-[10px] uppercase', fit === 'NATURAL' ? 'text-emerald-400' : fit ? 'text-brass' : 'text-sage/60')}>
                    {fit ?? '—'}
                  </span>
                </button>
              );
            })}
          </div>

          {selectedSlot && (
            <div className="mt-3">
              <p className="eyebrow">Available players</p>
              <div className="mt-2 max-h-56 space-y-1 overflow-y-auto pr-1">
                {availableForSlot.length === 0 && (
                  <p className="text-[12px] text-sage">No unassigned players left.</p>
                )}
                {availableForSlot.map((player) => (
                  <button
                    key={player.player_id}
                    type="button"
                    onClick={() => pickPlayer(player.player_id)}
                    className="flex w-full items-center justify-between gap-3 rounded-md border border-line bg-black/20 px-3 py-1.5 text-left text-[12px] hover:border-brass/40"
                  >
                    <span className="min-w-0 truncate text-bone">{player.full_name}</span>
                    <span className="font-mono text-[10px] text-sage">
                      {player.position} · {player.ovr}
                      {player.injured_matches ? ' · injured' : ''}
                    </span>
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </Card>
  );
};
