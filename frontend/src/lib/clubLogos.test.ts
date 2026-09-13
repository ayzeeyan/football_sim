import { describe, expect, test } from 'bun:test';
import { CLUB_CREST_URLS, getClubCrestUrl, getClubCrestUrlByShort } from './clubLogos';

describe('club crest catalogue', () => {
  test('covers every club in the 96-club career dataset', () => {
    expect(Object.keys(CLUB_CREST_URLS)).toHaveLength(96);
    expect(getClubCrestUrl({ club_id: 'EPL-ARS' })).toContain('Arsenal%20FC.png');
    expect(getClubCrestUrl({ club_id: 'LAL-RMA' })).toContain('Real%20Madrid.png');
    expect(getClubCrestUrl({ club_id: 'SEA-JUV' })).toContain('Juventus%20FC.png');
    expect(getClubCrestUrl({ club_id: 'BUN-BAY' })).toContain('Bayern%20Munich.png');
    expect(getClubCrestUrl({ club_id: 'FL1-PSG' })).toContain('Paris%20Saint-Germain.png');
  });

  test('resolves short-name payloads without an initials placeholder', () => {
    for (const short of ['ARS', 'BET', 'JUV', 'BVB', 'PSG']) {
      expect(getClubCrestUrlByShort(short)).toContain('.png');
    }
  });
});
