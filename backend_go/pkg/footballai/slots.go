package footballai

// Feature slot layout (feature_schema_version = 1).
//
// Every FootballMoE request is encoded into one fixed-width vector of raw
// slots. Each task fills the groups relevant to it and leaves the rest zero,
// so the shared encoder always sees the same layout. Slots are documented in
// SlotNames and normalized per-slot using metadata stored in the .fmoe file.

const (
	playerGroupWidth  = 24 // 15 numerics + 4 position one-hot + 5 personality one-hot
	clubGroupWidth    = 5
	matchGroupWidth   = 17
	financialWidth    = 8
	squadRoleWidth    = 5
	boardWidth        = 7
	worldContextWidth = 4
	managerWidthFeats = 11

	// InputWidth is the total number of input slots.
	InputWidth = playerGroupWidth + clubGroupWidth + matchGroupWidth + financialWidth +
		squadRoleWidth + boardWidth + worldContextWidth + managerWidthFeats
)

// Contiguous slot offsets for each feature group.
const (
	slotPlayerStart = 0
	slotClubStart   = slotPlayerStart + playerGroupWidth
	slotMatchStart  = slotClubStart + clubGroupWidth
	slotFinStart    = slotMatchStart + matchGroupWidth
	slotRoleStart   = slotFinStart + financialWidth
	slotBoardStart  = slotRoleStart + squadRoleWidth
	slotWorldStart  = slotBoardStart + boardWidth
	slotMgrStart    = slotWorldStart + worldContextWidth
)

// Individual slot offsets inside groups.
const (
	slotAge = slotPlayerStart + iota
	slotOVR
	slotPotential
	slotFitness
	slotSharpness
	slotFatigue
	slotConsecutiveStarts
	slotMinutes14d
	slotMorale
	slotFormModifier
	slotRecentInjury
	slotPlayerImportance
	slotReplacementOVR
	slotAcademyQuality
	slotTrainingQuality
	slotPositionOneHot // 4 slots
	slotPersonality    // 5 slots, ends at slotPlayerStart+24
)

const (
	slotClubReputation = slotClubStart + iota
	slotFinancialPower
	slotSellingTendency
	slotRecruitmentAmbition
	slotFinancialPressure
)

const (
	slotHomeRating = slotMatchStart + iota
	slotAwayRating
	slotHomeForm
	slotAwayForm
	slotHomeFitness
	slotAwayFitness
	slotHomeFatigue
	slotAwayFatigue
	slotTacticalEdge
	slotMatchImportance
	slotDerby
	slotRain
	slotEuropeanNight
	slotHomeAbsences
	slotAwayAbsences
	slotHomeAttackBias
	slotAwayAttackBias
)

const (
	slotContractYears = slotFinStart + iota
	slotLoyalty
	slotBaselineAnchor
	slotBaselineWage
	slotBidPrice
	slotAskingPrice
	slotWageHeadroom
	slotSquadRoleOneHot // 5 slots
)

const (
	slotPointsPerGame = slotBoardStart + iota
	slotExpectedPPG
	slotLast5PPG
	slotBoardPatience
	slotSpendRatio
	slotManagerTenureMonths
	slotCupProgress
)

const (
	slotWorldImportance = slotWorldStart + iota
	slotWorldFatigue
	slotTransferPressure
	slotClubPressure
)

const (
	slotMgrRisk = slotMgrStart + iota
	slotMgrYouthPreference
	slotMgrFinancialCaution
	slotMgrPatience
	slotMgrRotationPreference
	slotMgrAttacking
	slotMgrPressing
	slotMgrReputation
	slotMgrExperience
	slotMgrAdaptability
	slotMgrTenure
)

// SlotNames lists a stable name for every input slot. The names are stored in
// exported models and used by modeltool for inspection.
func SlotNames() [InputWidth]string {
	var names [InputWidth]string
	set := func(slot int, name string) { names[slot] = name }
	set(slotAge, "player.age")
	set(slotOVR, "player.ovr")
	set(slotPotential, "player.potential")
	set(slotFitness, "player.fitness")
	set(slotSharpness, "player.sharpness")
	set(slotFatigue, "player.fatigue")
	set(slotConsecutiveStarts, "player.consecutive_starts")
	set(slotMinutes14d, "player.minutes_last_14_days")
	set(slotMorale, "player.morale")
	set(slotFormModifier, "player.form_modifier")
	set(slotRecentInjury, "player.recent_injury")
	set(slotPlayerImportance, "player.match_importance")
	set(slotReplacementOVR, "player.replacement_ovr")
	set(slotAcademyQuality, "player.academy_quality")
	set(slotTrainingQuality, "player.training_quality")
	for i, p := range []string{"GK", "DEF", "MID", "FWD"} {
		set(slotPositionOneHot+i, "player.position."+p)
	}
	for i, p := range []string{"academic_dual", "big_game_performer", "dedicated_pro", "flamboyant_star", "snake"} {
		set(slotPersonality+i, "player.personality."+p)
	}
	set(slotClubReputation, "club.reputation")
	set(slotFinancialPower, "club.financial_power")
	set(slotSellingTendency, "club.selling_tendency")
	set(slotRecruitmentAmbition, "club.recruitment_ambition")
	set(slotFinancialPressure, "club.financial_pressure")
	set(slotHomeRating, "match.home_rating")
	set(slotAwayRating, "match.away_rating")
	set(slotHomeForm, "match.home_form")
	set(slotAwayForm, "match.away_form")
	set(slotHomeFitness, "match.home_fitness")
	set(slotAwayFitness, "match.away_fitness")
	set(slotHomeFatigue, "match.home_fatigue")
	set(slotAwayFatigue, "match.away_fatigue")
	set(slotTacticalEdge, "match.tactical_edge")
	set(slotMatchImportance, "match.importance")
	set(slotDerby, "match.derby")
	set(slotRain, "match.rain")
	set(slotEuropeanNight, "match.european_night")
	set(slotHomeAbsences, "match.home_absences")
	set(slotAwayAbsences, "match.away_absences")
	set(slotHomeAttackBias, "match.home_attack_bias")
	set(slotAwayAttackBias, "match.away_attack_bias")
	set(slotContractYears, "fin.contract_years")
	set(slotLoyalty, "fin.loyalty")
	set(slotBaselineAnchor, "fin.baseline_anchor_eur")
	set(slotBaselineWage, "fin.baseline_wage_eur")
	set(slotBidPrice, "fin.bid_price_eur")
	set(slotAskingPrice, "fin.asking_price_eur")
	set(slotWageHeadroom, "fin.wage_headroom")
	for i, r := range []string{"star", "important", "starter", "rotation", "prospect"} {
		set(slotSquadRoleOneHot+i, "fin.squad_role."+r)
	}
	set(slotPointsPerGame, "board.points_per_game")
	set(slotExpectedPPG, "board.expected_points_per_game")
	set(slotLast5PPG, "board.last5_points_per_game")
	set(slotBoardPatience, "board.patience")
	set(slotSpendRatio, "board.spend_ratio_vs_expectation")
	set(slotManagerTenureMonths, "board.manager_tenure_months")
	set(slotCupProgress, "board.cup_progress_score")
	set(slotWorldImportance, "world.match_importance")
	set(slotWorldFatigue, "world.player_fatigue")
	set(slotTransferPressure, "world.transfer_pressure")
	set(slotClubPressure, "world.club_pressure")
	set(slotMgrRisk, "manager.risk")
	set(slotMgrYouthPreference, "manager.youth_preference")
	set(slotMgrFinancialCaution, "manager.financial_caution")
	set(slotMgrPatience, "manager.patience")
	set(slotMgrRotationPreference, "manager.rotation_preference")
	set(slotMgrAttacking, "manager.attacking")
	set(slotMgrPressing, "manager.pressing")
	set(slotMgrReputation, "manager.reputation")
	set(slotMgrExperience, "manager.experience")
	set(slotMgrAdaptability, "manager.adaptability")
	set(slotMgrTenure, "manager.tenure")
	return names
}

// SlotLayout returns the ordered slot-group definitions.
func SlotLayout() []SlotGroup {
	return []SlotGroup{
		{Name: "PlayerFeatures", Start: slotPlayerStart, Width: playerGroupWidth},
		{Name: "ClubFeatures", Start: slotClubStart, Width: clubGroupWidth},
		{Name: "MatchContext", Start: slotMatchStart, Width: matchGroupWidth},
		{Name: "FinancialContext", Start: slotFinStart, Width: financialWidth},
		{Name: "SquadRole", Start: slotRoleStart, Width: squadRoleWidth},
		{Name: "BoardContext", Start: slotBoardStart, Width: boardWidth},
		{Name: "WorldContext", Start: slotWorldStart, Width: worldContextWidth},
		{Name: "ManagerFeatures", Start: slotMgrStart, Width: managerWidthFeats},
	}
}
