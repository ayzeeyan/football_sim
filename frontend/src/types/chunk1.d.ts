import './index';

declare module './index' {
  interface ClubIdentity {
    reputation: number;
    historical_prestige: number;
    financial_power: number;
    board_patience: number;
    academy_quality: number;
    recruitment_ambition: number;
    youth_preference: number;
    transfer_aggressiveness: number;
    selling_tendency: number;
  }

  interface ClubFinances {
    transfer_budget: number;
    balance: number;
  }

  interface Club {
    identity?: ClubIdentity;
    finances?: ClubFinances;
    reputation?: number;
    transfer_warchest_eur?: number;
    formatted_transfer_warchest?: string;
  }

  interface AwardsCategory {
    winner_id?: string;
  }
}

export {};
