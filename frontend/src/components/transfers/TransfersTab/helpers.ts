import type { TransfersResponse } from '../../../services/api/transfers';


export interface TransfersTabProps {
  onShowToast: (msg: string) => void;
}

export function canStartNextSeason(data: Pick<TransfersResponse, 'is_window_open' | 'window_type' | 'season_phase' | 'window_week' | 'max_window_weeks'>): boolean {
  const week = data.window_week ?? 0;
  const maxWeeks = data.max_window_weeks ?? 12;
  return !data.is_window_open && data.window_type === 'SUMMER' && data.season_phase === 'transfer_window' && week === maxWeeks;
}

export const STAGE_TONES = ['', 'bg-[#5B7FA6]', 'bg-[#C98A4B]', 'bg-[#8E86C8]', 'bg-[#8AB4C8]', 'bg-pitchtone'];

export type TransfersSubTab = 'negotiations' | 'feed' | 'warchests' | 'history' | 'records';

export const STAGE_LABELS: Record<number, string> = {
  1: 'Initial Enquiry & Scouting',
  2: 'Opening Valuation Bids',
  3: 'Transfer Fee Agreed',
  4: 'Personal Terms & Medical',
  5: 'Final Contract Signed',
};
