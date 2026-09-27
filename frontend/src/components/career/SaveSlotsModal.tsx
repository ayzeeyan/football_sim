import React, { useEffect, useRef, useState } from 'react';
import { Archive, Download, Import } from 'lucide-react';
import type { SaveSlot } from '../../types';
import {
  createSaveSlot,
  deleteSaveSlot,
  duplicateSaveSlot,
  fetchSaveSlots,
  importSaveSlot,
  renameSaveSlot,
  saveSlotExportUrl,
} from '../../services/api';
import { Card, ConfirmBar, EmptyState, LoadingState, Modal, ModalHeader } from '../ui/ui';

/**
 * Save-slot manager (F9): named archives of the current career. Slots are
 * self-contained snapshot files the viewer can duplicate, export, and
 * re-import; the active career keeps running at the main save and a new
 * career never touches these archives.
 */
export const SaveSlotsModal: React.FC<{
  open: boolean;
  onClose: () => void;
  onToast: (message: string) => void;
}> = ({ open, onClose, onToast }) => {
  const [slots, setSlots] = useState<SaveSlot[]>([]);
  const [loading, setLoading] = useState(false);
  const [newName, setNewName] = useState('');
  const [busy, setBusy] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState<SaveSlot | null>(null);
  const [renaming, setRenaming] = useState<SaveSlot | null>(null);
  const [renameValue, setRenameValue] = useState('');
  const fileInputRef = useRef<HTMLInputElement | null>(null);

  const reload = () => {
    setLoading(true);
    fetchSaveSlots()
      .then((res) => setSlots(res.slots ?? []))
      .catch(() => setSlots([]))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    if (open) reload();
  }, [open]);

  const handleCreate = async () => {
    const name = newName.trim();
    if (!name || busy) return;
    setBusy(true);
    const res = await createSaveSlot(name);
    setBusy(false);
    if (res.status === 'success') {
      setNewName('');
      onToast(`Archived "${name}".`);
      reload();
    } else {
      onToast('Could not archive the current career.');
    }
  };

  const handleRename = async () => {
    if (!renaming || busy) return;
    const name = renameValue.trim();
    if (!name) return;
    setBusy(true);
    const res = await renameSaveSlot(renaming.id, name);
    setBusy(false);
    if (res.status === 'success') {
      setRenaming(null);
      reload();
    } else {
      onToast('Could not rename the slot.');
    }
  };

  const handleDuplicate = async (slot: SaveSlot) => {
    if (busy) return;
    setBusy(true);
    const res = await duplicateSaveSlot(slot.id);
    setBusy(false);
    if (res.status === 'success') {
      onToast(`Duplicated "${slot.name}".`);
      reload();
    } else {
      onToast('Could not duplicate the slot.');
    }
  };

  const handleDelete = async () => {
    if (!confirmDelete || busy) return;
    setBusy(true);
    const res = await deleteSaveSlot(confirmDelete.id);
    setBusy(false);
    setConfirmDelete(null);
    if (res.status === 'success') {
      onToast('Slot deleted.');
      reload();
    } else {
      onToast('Could not delete the slot.');
    }
  };

  const handleImportFile = async (file: File) => {
    if (busy) return;
    setBusy(true);
    try {
      const text = await file.text();
      const name = file.name.replace(/\.json$/i, '') || 'Imported slot';
      const res = await importSaveSlot(name, text);
      if (res.status === 'success') {
        onToast(`Imported "${name}".`);
        reload();
      } else {
        onToast('That file is not a valid career snapshot.');
      }
    } catch {
      onToast('Could not read the file.');
    } finally {
      setBusy(false);
      if (fileInputRef.current) fileInputRef.current.value = '';
    }
  };

  return (
    <Modal open={open} onClose={onClose} maxWidth="max-w-3xl">
      <ModalHeader
        title={
          <span className="flex items-center gap-2">
            <Archive size={18} aria-hidden="true" /> Save slots
          </span>
        }
        subtitle="Named archives of the current career. The active career is untouched; a new career never deletes these."
        onClose={onClose}
      />
      <div className="space-y-4 p-4 sm:p-5">
        <Card className="p-3">
          <div className="flex flex-wrap items-center gap-2">
            <input
              type="text"
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              placeholder="Archive the current career as…"
              maxLength={64}
              className="min-w-0 flex-1 border border-line bg-ink px-3 py-2 text-[12px] text-bone"
              aria-label="New slot name"
            />
            <button
              type="button"
              onClick={() => void handleCreate()}
              disabled={busy || newName.trim() === ''}
              className="min-h-9 border border-brass/40 bg-brass/10 px-3 text-[12px] font-semibold text-brass hover:bg-brass/20 disabled:opacity-50"
            >
              Archive now
            </button>
            <button
              type="button"
              onClick={() => fileInputRef.current?.click()}
              disabled={busy}
              className="min-h-9 flex items-center gap-1.5 border border-line bg-cardLight px-3 text-[12px] font-semibold text-sage hover:border-brass/30 hover:text-bone disabled:opacity-50"
            >
              <Import size={13} aria-hidden="true" /> Import
            </button>
            <input
              ref={fileInputRef}
              type="file"
              accept="application/json"
              className="hidden"
              onChange={(e) => {
                const file = e.target.files?.[0];
                if (file) void handleImportFile(file);
              }}
            />
          </div>
        </Card>

        {loading && <LoadingState message="Loading slots…" />}
        {!loading && slots.length === 0 && (
          <EmptyState message="No archived slots yet. Archive the current career to create one." />
        )}

        {slots.map((slot) => (
          <Card key={slot.id} className="p-3">
            {renaming?.id === slot.id ? (
              <div className="flex flex-wrap items-center gap-2">
                <input
                  type="text"
                  value={renameValue}
                  onChange={(e) => setRenameValue(e.target.value)}
                  maxLength={64}
                  className="min-w-0 flex-1 border border-line bg-ink px-3 py-2 text-[12px] text-bone"
                  aria-label="Slot name"
                />
                <button type="button" onClick={() => void handleRename()} disabled={busy} className="min-h-9 border border-brass/40 bg-brass/10 px-3 text-[12px] font-semibold text-brass">
                  Save
                </button>
                <button type="button" onClick={() => setRenaming(null)} className="min-h-9 border border-line px-3 text-[12px] text-sage">
                  Cancel
                </button>
              </div>
            ) : (
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div className="min-w-0">
                  <p className="truncate text-[13px] font-semibold text-bone">{slot.name}</p>
                  <p className="text-[11px] text-sage">
                    {slot.season} · week {slot.matchweek} · {(slot.size_bytes / 1_000_000).toFixed(1)} MB · {slot.id}
                  </p>
                </div>
                <div className="flex flex-wrap items-center gap-1.5">
                  <button
                    type="button"
                    onClick={() => {
                      setRenaming(slot);
                      setRenameValue(slot.name);
                    }}
                    className="min-h-8 border border-line px-2.5 text-[11px] text-sage hover:text-bone"
                  >
                    Rename
                  </button>
                  <button type="button" onClick={() => void handleDuplicate(slot)} disabled={busy} className="min-h-8 border border-line px-2.5 text-[11px] text-sage hover:text-bone">
                    Duplicate
                  </button>
                  <a
                    href={saveSlotExportUrl(slot.id)}
                    className="flex min-h-8 items-center gap-1 border border-line px-2.5 text-[11px] text-sage hover:text-bone"
                  >
                    <Download size={12} aria-hidden="true" /> Export
                  </a>
                  <button type="button" onClick={() => setConfirmDelete(slot)} className="min-h-8 border border-brass/40 px-2.5 text-[11px] text-brass hover:bg-brass/10">
                    Delete
                  </button>
                </div>
              </div>
            )}
          </Card>
        ))}

        {confirmDelete && (
          <ConfirmBar
            message={`Delete the archived slot "${confirmDelete.name}"? This cannot be undone.`}
            confirmLabel="Delete slot"
            busyLabel="Deleting…"
            busy={busy}
            onConfirm={() => void handleDelete()}
            onCancel={() => setConfirmDelete(null)}
          />
        )}
      </div>
    </Modal>
  );
};
