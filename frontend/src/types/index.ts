export interface Player {
  player_id: string;
  full_name: string;
  position: string;
  ovr: number;
  age: number;
  market_value_eur: number;
  universe_wonderkid: boolean;
  player_source: string;
  club_id: string;
  goals: number;
  assists: number;
  appearances: number;
  category: 'GK' | 'DEF' | 'MID' | 'FWD';
  formatted_value: string;
  wage_eur: number;
  formatted_wage: string;
  contract_years: number;
  loyalty: number;
  career_goals: number;
  career_assists: number;
  career_apps?: number;
  own_goals: number;
  suspended_matches?: number;
  injured_matches?: number;
  injury?: string;
  availability?: string;
  grew_note?: string | null;
  education?: string;
  education_label?: string;
  education_pending?: boolean;
  school_want?: string;
  school_want_label?: string;
  school_track?: string;
  school_track_label?: string;
  secondary_position?: string;
  position_path?: string;
  position_xp?: number;
  position_options?: string[];
  best_goals?: number;
  best_assists?: number;
  best_season?: string;
  personality?: string;
  personality_title?: string;
  personality_badge?: string;
  mentor_id?: string | null;
  mentor_name?: string | null;
  mentor_ovr?: number | null;
}

export interface ManagerInfo {
  name: string;
  tactic: string;
  archetype?: 'high_press' | 'possession' | 'low_block' | 'free_flowing';
  archetype_label?: string;
  style: string;
  line_height?: string;
  press_intensity?: string;
  tempo?: string;
  focus: string;
  budget_eur: number;
  formatted_budget: string;
  adaptability?: number;
}

export interface Club {
  club_id: string;
  club_name: string;
  short_name: string;
  league: string;
  country: string;
  home_stadium: string;
  stadium_capacity: number;
  overall_team_rating: number;
  squad_size: number;
  squad_avg_ovr: number;
  primary_color: [number, number, number];
  secondary_color: [number, number, number];
  p: number;
  w: number;
  d: number;
  l: number;
  gf: number;
  ga: number;
  gd: number;
  pts: number;
  form: string[];
  morale?: number;
  manager: ManagerInfo | null;
  cup_status?: 'qualified' | 'eliminated' | 'must_win' | 'live' | null;
}

export interface CommentaryItem {
  minute: number;
  text: string;
  category: 'NORMAL' | 'KICKOFF' | 'CHANCE' | 'SAVE' | 'GOAL' | 'FOUL' | 'WONDERKID' | 'FULLTIME';
  is_wonderkid: boolean;
  timestamp: string;
}

export interface MatchTickPayload {
  tick_type?: 'full' | 'delta';
  seq?: number;
  state: 'NOT_STARTED' | 'PLAYING' | 'PAUSED' | 'HALF_TIME' | 'GOAL_PAUSE' | 'FULL_TIME';
  minute: number;
  speed: number;
  phase: string;
  active_third: 'DEFENSIVE' | 'MIDFIELD' | 'ATTACKING';
  home_score: number;
  away_score: number;
  home_shots: number;
  away_shots: number;
  home_shots_on_target: number;
  away_shots_on_target: number;
  home_corners: number;
  away_corners: number;
  home_possession_pct: number;
  away_possession_pct: number;
  possession_momentum: number;
  ball: {
    x: number;
    y: number;
    height: number;
    is_shot: boolean;
  };
  pass_trail: {
    from: [number, number];
    to: [number, number];
    is_shot: boolean;
    color: [number, number, number];
  } | null;
  home_coords: Array<{ x: number; y: number; player: Player; sent_off?: boolean }>;
  away_coords: Array<{ x: number; y: number; player: Player; sent_off?: boolean }>;
  goal_banner: string | null;
  commentary: CommentaryItem[];
  match_events?: MatchEventItem[];
  league_fixture: { id: string; status: 'scheduled' | 'finished' } | null;
  home_bench?: LiveBenchRow[];
  away_bench?: LiveBenchRow[];
  home_tactical_stance?: 'NORMAL' | 'OVERLOAD' | 'PARK_BUS';
  away_tactical_stance?: 'NORMAL' | 'OVERLOAD' | 'PARK_BUS';
  latest_tactical_shift?: TacticalShiftItem | null;
  home_manager?: ManagerInfo | null;
  away_manager?: ManagerInfo | null;
  weather?: string;
}

export interface LiveBenchRow {
  player: Player;
  status: 'bench' | 'on';
  on_minute: number | null;
}

// --- Career match centre ----------------------------------------------------

export interface MatchMiniPlayer {
  player_id: string;
  full_name: string;
  position: string;
  ovr: number;
  age: number;
  category: string;
  is_wk: boolean;
}

export interface MatchEventItem {
  minute: number;
  display: string;
  seq: number;
  type: 'goal' | 'penalty' | 'penalty_miss' | 'own_goal' | 'corner_goal' | 'free_kick_goal' | 'yellow' | 'red' | 'sub' | 'var_review';
  side: 'home' | 'away';
  beneficiary?: 'home' | 'away' | null;
  scorer?: MatchMiniPlayer | null;
  assister?: MatchMiniPlayer | null;
  player?: MatchMiniPlayer | null;
  player_in?: MatchMiniPlayer | null;
  player_out?: MatchMiniPlayer | null;
  sent_off?: boolean | null;
  detail?: string | null;
  disallowed?: boolean;
  outcome?: string | null;
  reason?: string | null;
  decision?: string | null;
  home_score?: number | null;
  away_score?: number | null;
}

export interface MatchPlayerRow extends MatchMiniPlayer {
  rating: number;
  match_goals: number;
  match_assists: number;
  match_og: number;
  match_pen_miss: number;
  card: 'yellow' | 'red' | null;
  minutes?: number;
  on_minute?: number | null;
  off_minute?: number | null;
  starter?: boolean;
  played?: boolean;
}

export interface TeamMatchStats {
  shots: number;
  on_target: number;
  possession: number;
  corners: number;
  passes: number;
  pass_accuracy: number;
  fouls: number;
  yellows: number;
  reds: number;
  xg?: number;
}

export interface ShotItem {
  minute: number;
  team: 'home' | 'away';
  shooter: MatchMiniPlayer | { full_name: string; position: string; ovr: number };
  x: number;
  y: number;
  xg: number;
  outcome: 'goal' | 'save' | 'miss';
  is_wonderkid?: boolean;
}

export interface XGFlowPoint {
  minute: number;
  home_xg: number;
  away_xg: number;
}

export interface ShotMapData {
  shots: ShotItem[];
  xg_flow: XGFlowPoint[];
  total_home_xg: number;
  total_away_xg: number;
}

export interface TouchHeatmapData {
  home_points: Array<[number, number, number]>;
  away_points: Array<[number, number, number]>;
  home_zones: {
    defensive: number;
    midfield: number;
    attacking: number;
    left: number;
    center: number;
    right: number;
  };
  away_zones: {
    defensive: number;
    midfield: number;
    attacking: number;
    left: number;
    center: number;
    right: number;
  };
}

export interface PressConferenceData {
  headline: string;
  home_quote: string;
  away_quote: string;
  home_manager?: string;
  away_manager?: string;
}

export interface TacticalShiftItem {
  minute: number;
  team: 'home' | 'away';
  club_short: string;
  manager: string;
  stance: 'NORMAL' | 'OVERLOAD' | 'PARK_BUS';
  label: string;
  text: string;
}

export interface HeadToHeadRow {
  id: string;
  matchweek: number;
  competition: string;
  home_id: string;
  away_id: string;
  home_goals: number | null;
  away_goals: number | null;
  stage: string;
}

export interface Fixture {
  id: string;
  matchweek: number;
  competition: string;
  stage: string;
  leg: number | null;
  tie_id: string | null;
  status: 'scheduled' | 'finished';
  method: 'instant' | 'live' | null;
  home: Club;
  away: Club;
  home_goals: number | null;
  away_goals: number | null;
  events: MatchEventItem[];
  home_xi: MatchPlayerRow[];
  away_xi: MatchPlayerRow[];
  home_bench?: MatchPlayerRow[];
  away_bench?: MatchPlayerRow[];
  stats: { home: TeamMatchStats; away: TeamMatchStats } | null;
  motm: (MatchPlayerRow & { side: 'home' | 'away' }) | null;
  decided_by?: string | null;
  penalties?: [number, number] | null;
  derby?: string | null;
  derby_name?: string | null;
  derby_heat?: number;
  is_derby?: boolean;
  is_high_heat_derby?: boolean;
  weather?: string;
  ht_home?: number | null;
  ht_away?: number | null;
  attendance?: number | null;
  referee?: string | null;
  head_to_head?: HeadToHeadRow[];
  preview?: FixturePreview | null;
  night?: EuropeanNight | null;
  shot_map?: ShotMapData | null;
  touch_heatmap?: TouchHeatmapData | null;
  press_conference?: PressConferenceData | null;
  home_manager?: ManagerInfo | null;
  away_manager?: ManagerInfo | null;
  tactical_shifts?: TacticalShiftItem[];
}

export interface EuropeanNight {
  kind: 'ucl' | 'super-cup';
  badge: string;
  story: string;
  group_status?: 'qualified' | 'eliminated' | 'must_win' | 'live' | null;
  aggregate?: string | null;
  leg1_label?: string | null;
  leg?: number | null;
}

export interface FixturePreview {
  kickoff_note: string;
  venue: string;
  capacity: number;
  home_form: string[];
  away_form: string[];
  home_pos: number | null;
  away_pos: number | null;
  home_pts: number;
  away_pts: number;
  home_xi: Player[];
  away_xi: Player[];
  home_bench: Player[];
  away_bench: Player[];
  home_missing: Player[];
  away_missing: Player[];
  home_xi_avg: number;
  away_xi_avg: number;
}

export interface FixturesResponse {
  current_matchweek: number;
  max_matchweeks: number;
  season_phase: 'season' | 'transfer_window';
  season_name: string;
  matchweek: number;
  fixtures: Fixture[];
  ucl_pending_ids: string[];
}

export interface ProdigyAttributes {
  pace: number;
  shooting: number;
  passing: number;
  dribbling: number;
  defending: number;
  physicality: number;
  aerial_reach: number;
  heading_power: number;
  strength: number;
  shielding: number;
  press_resistance: number;
  stamina: number;
  composure: number;
}

export interface ProdigyData {
  player_id: string;
  full_name: string;
  age: number;
  current_height_cm: number;
  baseline_height_cm: number;
  height_gain_cm: number;
  height_display?: string;
  adult_height_age?: number;
  still_growing?: boolean;
  current_weight_kg: number;
  baseline_weight_kg: number;
  weight_gain_kg: number;
  potential: number;
  puberty_stage: string;
  growth_velocity: number;
  accumulated_xp: number;
  level_xp_target: number;
  xp_pct: number;
  ovr: number;
  club_name: string;
  club_short: string;
  primary_color: [number, number, number];
  formatted_value: string;
  goals: number;
  assists: number;
  appearances: number;
  position: string;
  training_energy?: number;
  max_training_energy?: number;
  career_goals: number;
  career_assists: number;
  best_goals: number;
  best_assists: number;
  best_season: string;
  attributes: ProdigyAttributes;
  education?: string;
  education_label?: string;
  education_pending?: boolean;
  school_want?: string;
  school_want_label?: string;
  school_track?: string;
  school_track_label?: string;
  secondary_position?: string;
  position_path?: string;
  position_xp?: number;
  position_options?: string[];
  personality?: string;
  personality_title?: string;
  personality_badge?: string;
  personality_desc?: string;
  mentor_id?: string | null;
  mentor_name?: string | null;
  mentor_ovr?: number | null;
  progression_history?: ProgressionHistoryEntry[];
}

export interface ProgressionHistoryEntry {
  season: string;
  age: number;
  ovr: number;
  height_cm: number;
  weight_kg: number;
  goals: number;
  assists: number;
  appearances: number;
  club_short: string;
  mentor_name: string;
}

export interface NarrativeMilestone {
  id: string;
  title: string;
  description: string;
  unlocked: boolean;
  badge?: string;
  age?: number;
}

export interface ProdigyTimelineResponse {
  player_id: string;
  full_name: string;
  age: number;
  current_height_cm: number;
  baseline_height_cm: number;
  current_weight_kg: number;
  baseline_weight_kg: number;
  height_gain_cm: number;
  weight_gain_kg: number;
  ovr: number;
  potential: number;
  progression_history: ProgressionHistoryEntry[];
  milestones: NarrativeMilestone[];
}

export interface HeadToHeadData {
  club_a: {
    club_id: string;
    club_name: string;
    short_name: string;
    primary_color: [number, number, number];
  };
  club_b: {
    club_id: string;
    club_name: string;
    short_name: string;
    primary_color: [number, number, number];
  };
  matches_played: number;
  wins_a: number;
  wins_b: number;
  draws: number;
  goals_a: number;
  goals_b: number;
  derby_name: string | null;
  derby_heat: number;
  recent_matches: Array<{
    id: string;
    matchweek: number;
    competition: string;
    stage: string;
    home_id: string;
    away_id: string;
    home_goals: number;
    away_goals: number;
    winner: string;
  }>;
}

export interface ClubSeasonHistoryEntry {
  season_name: string;
  position: number;
  pts: number;
  w: number;
  d: number;
  l: number;
  gf: number;
  ga: number;
  gd: number;
  trophies: string[];
}

export interface ClubHistoryResponse {
  club_id: string;
  club_name: string;
  short_name: string;
  primary_color: [number, number, number];
  history: ClubSeasonHistoryEntry[];
  trophies_summary: {
    super_league: number;
    ucl: number;
    super_cup: number;
  };
  historical?: {
    super_league: number;
    ucl: number;
    super_cup: number;
  };
}

export interface BallonDorRankItem {
  rank: number;
  player_id: string;
  full_name: string;
  club_id: string;
  club_name: string;
  club_short: string;
  position: string;
  ovr: number;
  goals: number;
  assists: number;
  all_time_goals: number;
  all_time_assists: number;
  trophies_won: number;
  score: number;
  is_wonderkid: boolean;
}

export interface TOTSCard {
  player_id: string;
  full_name: string;
  position: string;
  role: string;
  ovr: number;
  age: number;
  club_id: string;
  club_name: string;
  club_short: string;
  primary_color: [number, number, number];
  goals: number;
  assists: number;
  appearances: number;
  is_wonderkid: boolean;
}

export interface TeamOfTheSeason {
  formation: string;
  gk: TOTSCard;
  lb: TOTSCard;
  cb1: TOTSCard;
  cb2: TOTSCard;
  rb: TOTSCard;
  mid1: TOTSCard;
  mid2: TOTSCard;
  mid3: TOTSCard;
  fwd1: TOTSCard;
  fwd2: TOTSCard;
  fwd3: TOTSCard;
  xi: TOTSCard[];
}

export interface ManagerOfTheYear {
  club_id: string;
  club_name: string;
  short_name: string;
  primary_color: [number, number, number];
  name: string;
  manager_name: string;
  tactic: string;
  style: string;
  actual_finish: number;
  expected_finish: number;
  outperformed_places: number;
  pts: number;
  trophies_won: number;
  score: number;
  accolade: string;
}

export interface TopSigningItem {
  player_id: string;
  player_name: string;
  player_pos: string;
  player_ovr: number;
  is_wonderkid: boolean;
  seller_id: string;
  seller_name: string;
  seller_short: string;
  buyer_id: string;
  buyer_name: string;
  buyer_short: string;
  fee_eur: number;
  formatted_fee: string;
  matchweek: number;
}

export interface ClubNetSpendItem {
  club_id: string;
  club_name: string;
  short_name: string;
  primary_color: [number, number, number];
  spent: number;
  received: number;
  net: number;
}

export interface TransferRecordsData {
  top_signings: TopSigningItem[];
  net_spend: Record<string, ClubNetSpendItem>;
  total_transfers_count: number;
}

export interface SeasonAwards {
  season_name?: string;
  super_league_champion: {
    club_name: string;
    short_name: string;
    pts: number;
  } | null;
  super_league_runner_up: {
    club_name: string;
    short_name: string;
    pts: number;
  } | null;
  ucl_champion: {
    club_name: string;
    short_name: string;
  } | null;
  top_scorer: {
    full_name: string;
    goals: number;
    club_id: string;
  } | null;
  top_assister: {
    full_name: string;
    assists: number;
    club_id: string;
  } | null;
  golden_boy: {
    full_name: string;
    ovr: number;
    goals: number;
    assists: number;
    club_id: string;
  } | null;
  super_cup_champion?: {
    club_name: string;
    short_name: string;
  } | null;
  player_of_the_season: {
    full_name: string;
    club_name: string;
    short_name: string;
    goals: number;
    assists: number;
    ovr: number;
    is_wonderkid: boolean;
    club_id: string;
  } | null;
  ballon_dor?: BallonDorRankItem[];
  team_of_the_season?: TeamOfTheSeason | null;
  manager_of_the_year?: ManagerOfTheYear | null;
}

export interface AwardsNominee {
  full_name: string;
  position: string;
  ovr: number;
  age: number;
  goals: number;
  assists: number;
  club_name: string;
  short_name: string;
  is_wonderkid: boolean;
  stats_line: string;
}

export interface AwardsCategory {
  key: string;
  title: string;
  blurb: string;
  nominees: AwardsNominee[];
  winner: AwardsNominee | null;
}

export interface AwardsCeremony {
  season_name: string;
  categories: AwardsCategory[];
  ballon_dor?: BallonDorRankItem[];
  team_of_the_season?: TeamOfTheSeason | null;
  manager_of_the_year?: ManagerOfTheYear | null;
}

export interface GrowthMilestoneItem {
  timestamp: string;
  player_name: string;
  event_type: string;
  description: string;
  badge_color: string;
}

export interface UCLSemiFinalMatch {
  home: Club | null;
  away: Club | null;
  leg1: [number, number] | null;
  leg2: [number, number] | null;
  winner: Club | null;
  decided_by?: string | null;
  penalties?: [number, number] | null;
}

export interface UCLFinalMatch {
  team1: Club | null;
  team2: Club | null;
  score: [number, number] | null;
  winner: Club | null;
  decided_by?: string | null;
  penalties?: [number, number] | null;
}

export interface UCLTournamentState {
  stage: string;
  group_a: Club[];
  group_b: Club[];
  semi_finals: {
    semi_1?: UCLSemiFinalMatch;
    semi_2?: UCLSemiFinalMatch;
  };
  quarter_finals: {
    qf_1?: UCLSemiFinalMatch;
    qf_2?: UCLSemiFinalMatch;
    qf_3?: UCLSemiFinalMatch;
    qf_4?: UCLSemiFinalMatch;
  };
  final: UCLFinalMatch | null;
  champion: Club | null;
}

export interface SuperLeagueState {
  current_matchweek: number;
  max_matchweeks: number;
  season_phase: 'season' | 'transfer_window';
  season_name: string;
  clubs: Club[];
  recent_results: string[];
}

export interface TransferFeedItem {
  headline: string;
  category: 'EXCLUSIVE' | 'TWIST' | 'HIJACK' | 'HERE_WE_GO' | 'RUMOR' | 'REJECTED';
  is_wonderkid: boolean;
  matchweek: number;
  timestamp: string;
}

export interface TransferNegotiation {
  negotiation_id: string;
  player: Player;
  buyer: Club;
  seller: Club;
  current_bid: number;
  formatted_bid: string;
  stage_index: number;
  stage_name: string;
  progress_pct: number;
  is_wonderkid: boolean;
  is_hijacked: boolean;
  original_buyer?: Club;
}

export interface CompletedTransfer {
  player_name: string;
  player_pos: string;
  player_ovr: number;
  is_wonderkid: boolean;
  seller_name: string;
  seller_short: string;
  buyer_name: string;
  buyer_short: string;
  fee_eur: number;
  formatted_fee: string;
  matchweek: number;
}

export interface InboxItem {
  id: string;
  timestamp: string;
  matchweek: number;
  season_name: string;
  category: 'match' | 'transfer' | 'wonderkid' | 'honour' | 'race' | 'cup' | 'system' | 'injury' | 'dugout' | 'youth';
  headline: string;
  body: string;
  club_ids: string[];
  player_id: string | null;
  fixture_id: string | null;
  unread: boolean;
}

export interface InboxFeed {
  unread: number;
  items: InboxItem[];
  season_name: string;
  current_matchweek: number;
}

export interface PlayerMatchLog {
  fixture_id: string;
  matchweek: number;
  competition: string;
  stage?: string;
  opponent: string;
  opponent_id: string;
  home: boolean;
  rating: number | null;
  goals: number;
  assists: number;
  minutes: number;
  motm: boolean;
  result: 'W' | 'D' | 'L';
  score: string;
}

export interface PlayerProfile {
  player: Player;
  club: Club | null;
  last_matches: PlayerMatchLog[];
  avg_rating: number | null;
  apps_rated: number;
}

export interface TrophyItem {
  type: 'ucl' | 'super_league' | 'super_cup';
  title: string;
  count: number;
  recent_years: string[];
}

export interface TrophyCabinetClub {
  club_id: string;
  club_name: string;
  short_name: string;
  primary_color: [number, number, number];
  total_trophies: number;
  super_league_count: number;
  ucl_count: number;
  super_cup_count: number;
  trophies: TrophyItem[];
  hist_super_league?: number;
  hist_ucl?: number;
  hist_super_cup?: number;
  hist_total?: number;
  career_super_league?: number;
  career_ucl?: number;
  career_super_cup?: number;
  career_total?: number;
}

export interface AllTimeRecordsData {
  top_goalscorers: Array<{
    player_id: string;
    full_name: string;
    club_name: string;
    short_name: string;
    goals: number;
    appearances: number;
    ovr: number;
    is_wonderkid: boolean;
  }>;
  top_assisters: Array<{
    player_id: string;
    full_name: string;
    club_name: string;
    short_name: string;
    assists: number;
    appearances: number;
    ovr: number;
    is_wonderkid: boolean;
  }>;
  highest_scoring_match: {
    fixture_id?: string;
    score: string;
    total_goals: number;
    competition?: string;
  };
  biggest_margin_victory: {
    fixture_id?: string;
    winner?: string;
    score: string;
    margin: number;
    competition?: string;
  };
  single_match_goals_record: {
    player_name: string;
    club_name: string;
    goals: number;
    fixture: string;
  };
  highest_season_points: {
    club_name: string;
    season_name: string;
    points: number;
  };
  wonderkid_milestones: {
    highest_ovr: {
      name: string;
      ovr: number;
      potential: number;
    };
    top_prodigy_goals: {
      name: string;
      goals: number;
    };
  };
}

export interface NXGNPlayer {
  rank: number;
  player_id: string;
  full_name: string;
  club_id: string;
  club_name: string;
  short_name: string;
  age: number;
  position: string;
  category: 'GK' | 'DEF' | 'MID' | 'FWD';
  ovr: number;
  potential: number;
  goals: number;
  assists: number;
  appearances: number;
  personality: string;
  personality_title: string;
  mentor_name: string | null;
  mentor_ovr: number | null;
  is_wonderkid: boolean;
  scout_verdict: string;
}
