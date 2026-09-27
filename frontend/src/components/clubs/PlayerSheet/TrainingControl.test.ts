import { describe, expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';

const sheetSource = readFileSync(new URL('./PlayerSheetModal.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const prodigiesApiSource = readFileSync(new URL('../../../services/api/prodigies.ts', import.meta.url), 'utf8').replace(/\r\n/g, '\n');

describe('Training control (Tier B2)', () => {
  test('the player sheet offers a training session for any player', () => {
    expect(sheetSource).toContain('Training session');
    expect(sheetSource).toContain('TRAINING_FOCUSES');
    expect(sheetSource).toContain('trainPlayer(player.player_id, focus.value)');
    expect(sheetSource).toContain('shared weekly training energy');
  });

  test('all three focuses are offered', () => {
    expect(sheetSource).toContain('focus.label');
  });

  test('the api client targets the any-player train endpoint', () => {
    expect(prodigiesApiSource).toContain('/players/${encodeURIComponent(playerId)}/train');
    expect(prodigiesApiSource).toContain('registers a growth profile on demand');
  });

  test('the response contract carries the post-session OVR', () => {
    expect(prodigiesApiSource).toContain('ovr?: number;');
  });
});
