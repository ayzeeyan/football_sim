import { describe, expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';

const cardSource = readFileSync(new URL('./NegotiationCard.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const tabSource = readFileSync(new URL('./TransfersTab.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const sheetSource = readFileSync(new URL('../../clubs/PlayerSheet/PlayerSheetModal.tsx', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
const transfersApiSource = readFileSync(new URL('../../../services/api/transfers.ts', import.meta.url), 'utf8').replace(/\r\n/g, '\n');

describe('Transfer bid UX (Tier B3)', () => {
  test('active negotiations offer improve and withdraw controls', () => {
    expect(cardSource).toContain("onRespond?: NegotiationRespondFn");
    expect(cardSource).toContain("'improve'");
    expect(cardSource).toContain("'withdraw'");
    expect(cardSource).toContain('Improve offer');
    expect(cardSource).toContain('asks {neg.formatted_asking_price}');
  });

  test('the transfers tab wires the respond handler with feedback and reload', () => {
    expect(tabSource).toContain('respondToNegotiation(n.negotiation_id, action, amount)');
    expect(tabSource).toContain('reload();');
    expect(tabSource).toContain('Offer submitted.');
  });

  test('the player sheet makes an offer for the viewed player', () => {
    expect(sheetSource).toContain('Make an offer');
    expect(sheetSource).toContain('submitTransferOffer(player.player_id, offerBuyer');
    expect(sheetSource).toContain('meeting the asking price completes the deal');
  });

  test('the api client covers offer and respond', () => {
    expect(transfersApiSource).toContain("'/api/transfers/offer'");
    expect(transfersApiSource).toContain('/api/transfers/negotiations/${encodeURIComponent(negotiationId)}/respond');
  });

  test('the negotiation type carries the asking price', () => {
    const typesSource = readFileSync(new URL('../../../types/index.ts', import.meta.url), 'utf8').replace(/\r\n/g, '\n');
    expect(typesSource).toContain('asking_price?: number;');
    expect(typesSource).toContain('formatted_asking_price?: string;');
  });
});
