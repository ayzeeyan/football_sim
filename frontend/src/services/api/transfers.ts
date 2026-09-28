import { apiFetch } from './core';
import type { CompletedTransfer, TransferFeedItem, TransferNegotiation, TransferRecordsData } from '../../types';


// --- Transfers --------------------------------------------------------------

export interface ExpiringContract {
  player_id: string;
  full_name: string;
  position: string;
  ovr: number;
  club_name: string;
  club_short: string;
  formatted_wage: string;
  loyalty: number;
  is_wonderkid: boolean;
}

export interface WarchestRow {
  club_name: string;
  club_short: string;
  manager_name: string;
  tactic: string;
  focus: string;
  budget_eur: number;
  formatted_budget: string;
  wage_bill_eur: number;
}

export interface TransfersResponse {
  window_name: string;
  is_window_open: boolean;
  window_type: 'CLOSED' | 'SUMMER' | 'WINTER';
  season_phase: 'season' | 'transfer_window';
  window_day: number;
  window_week?: number;
  max_window_weeks?: number;
  active_negotiations: TransferNegotiation[];
  transfer_feed: TransferFeedItem[];
  completed_transfers: CompletedTransfer[];
  expiring_contracts: ExpiringContract[];
  warchests: WarchestRow[];
  deadline_day?: boolean;
}

const EMPTY_TRANSFERS: TransfersResponse = {
  window_name: 'Window Closed (Opens at season end)',
  is_window_open: false,
  window_type: 'CLOSED',
  season_phase: 'season',
  window_day: 0,
  window_week: 0,
  max_window_weeks: 12,
  active_negotiations: [],
  transfer_feed: [],
  completed_transfers: [],
  expiring_contracts: [],
  warchests: [],
};

export function fetchTransfers(): Promise<TransfersResponse> {
  return apiFetch<TransfersResponse>('/transfers', undefined, EMPTY_TRANSFERS);
}

// Tier B only (B3 transfer bid UX): hands a human control of an AI transfer
// negotiation, which the neutral-viewer contract in AGENTS.md forbids today.


export function fetchTransferRecords(): Promise<TransferRecordsData> {
  return apiFetch<TransferRecordsData>('/transfers/records', undefined, {
    top_signings: [],
    net_spend: {},
    total_transfers_count: 0,
  });
}

// Tier B (B3): open a negotiation with a viewer offer. The backend validates
// the amount against the valuation corridor, the buyer's budget, and the

// Tier B (B3): improve an active negotiation (meeting the asking price

