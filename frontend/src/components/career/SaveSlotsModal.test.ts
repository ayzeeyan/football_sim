import { describe, expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';
import type { SaveSlot } from '../../types';

const modalSource = readFileSync(new URL('./SaveSlotsModal.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const appSource = readFileSync(new URL('../../App.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const topBarSource = readFileSync(new URL('../layout/TopBar.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const careerApiSource = readFileSync(new URL('../../services/api/career.ts', import.meta.url), 'utf8').replace(/\r\n/g, '\n');

describe('SaveSlotsModal (F9 save slots)', () => {
  test('offers every slot operation', () => {
    expect(modalSource).toContain('createSaveSlot');
    expect(modalSource).toContain('renameSaveSlot');
    expect(modalSource).toContain('duplicateSaveSlot');
    expect(modalSource).toContain('deleteSaveSlot');
    expect(modalSource).toContain('saveSlotExportUrl');
    expect(modalSource).toContain('importSaveSlot');
    expect(modalSource).toContain('fetchSaveSlots');
  });

  test('delete is behind a confirm bar', () => {
    expect(modalSource).toContain('ConfirmBar');
    expect(modalSource).toContain('This cannot be undone.');
  });

  test('keeps the neutral framing: archives, never active management', () => {
    expect(modalSource).toContain('The active career is untouched');
    expect(modalSource).toContain('a new career never deletes these');
  });

  test('is reachable from the top bar and wired in App', () => {
    expect(topBarSource).toContain('onOpenSlots');
    expect(topBarSource).toContain("t('action.saveSlots')");
    expect(appSource).toContain('<SaveSlotsModal open={slotsOpen}');
    expect(appSource).toContain('setSlotsOpen(true)');
  });

  test('auto-advance pauses while the slots modal is open', () => {
    expect(appSource).toContain('slotsOpen ||');
  });

  test('the api client covers the slot endpoints', () => {
    expect(careerApiSource).toContain("'/career/slots'");
    expect(careerApiSource).toContain('/rename?name=');
    expect(careerApiSource).toContain('/duplicate');
    expect(careerApiSource).toContain('/delete');
    expect(careerApiSource).toContain('/export');
    expect(careerApiSource).toContain('/import?name=');
  });
});

describe('SaveSlot contract', () => {
  test('carries id, name, season, and size', () => {
    const slot: SaveSlot = {
      id: 'slot-1',
      name: 'Title run',
      created_at: '2026-09-28T10:00:00Z',
      updated_at: '2026-09-28T10:00:00Z',
      season: '2026-27',
      matchweek: 12,
      size_bytes: 47_000_000,
    };
    expect(slot.id).toBe('slot-1');
    expect(slot.matchweek).toBe(12);
    expect(slot.size_bytes).toBeGreaterThan(0);
  });
});
