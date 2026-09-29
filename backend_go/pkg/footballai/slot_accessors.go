package footballai

// Exported slot constants for training-data converters and tooling. The
// game runtime uses the typed request structs instead of slot arithmetic.
const (
	SlotAge               = slotAge
	SlotOVR               = slotOVR
	SlotPotential         = slotPotential
	SlotFitness           = slotFitness
	SlotSharpness         = slotSharpness
	SlotFatigue           = slotFatigue
	SlotConsecutiveStarts = slotConsecutiveStarts
	SlotMinutes14d        = slotMinutes14d
	SlotMorale            = slotMorale
	SlotFormModifier      = slotFormModifier
	SlotRecentInjury      = slotRecentInjury
	SlotPlayerImportance  = slotPlayerImportance
	SlotReplacementOVR    = slotReplacementOVR
	SlotAcademyQuality    = slotAcademyQuality
	SlotTrainingQuality   = slotTrainingQuality
	SlotPositionOneHot    = slotPositionOneHot
	SlotPersonality       = slotPersonality

	SlotClubReputation      = slotClubReputation
	SlotFinancialPower      = slotFinancialPower
	SlotSellingTendency     = slotSellingTendency
	SlotRecruitmentAmbition = slotRecruitmentAmbition
	SlotFinancialPressure   = slotFinancialPressure

	SlotHomeRating      = slotHomeRating
	SlotAwayRating      = slotAwayRating
	SlotHomeForm        = slotHomeForm
	SlotAwayForm        = slotAwayForm
	SlotHomeFitness     = slotHomeFitness
	SlotAwayFitness     = slotAwayFitness
	SlotHomeFatigue     = slotHomeFatigue
	SlotAwayFatigue     = slotAwayFatigue
	SlotTacticalEdge    = slotTacticalEdge
	SlotMatchImportance = slotMatchImportance
	SlotDerby           = slotDerby
	SlotRain            = slotRain
	SlotEuropeanNight   = slotEuropeanNight
	SlotHomeAbsences    = slotHomeAbsences
	SlotAwayAbsences    = slotAwayAbsences
	SlotHomeAttackBias  = slotHomeAttackBias
	SlotAwayAttackBias  = slotAwayAttackBias

	SlotContractYears   = slotContractYears
	SlotLoyalty         = slotLoyalty
	SlotBaselineAnchor  = slotBaselineAnchor
	SlotBaselineWage    = slotBaselineWage
	SlotBidPrice        = slotBidPrice
	SlotAskingPrice     = slotAskingPrice
	SlotWageHeadroom    = slotWageHeadroom
	SlotSquadRoleOneHot = slotSquadRoleOneHot

	SlotPointsPerGame       = slotPointsPerGame
	SlotExpectedPPG         = slotExpectedPPG
	SlotLast5PPG            = slotLast5PPG
	SlotBoardPatience       = slotBoardPatience
	SlotSpendRatio          = slotSpendRatio
	SlotManagerTenureMonths = slotManagerTenureMonths
	SlotCupProgress         = slotCupProgress

	SlotWorldImportance  = slotWorldImportance
	SlotWorldFatigue     = slotWorldFatigue
	SlotTransferPressure = slotTransferPressure
	SlotClubPressure     = slotClubPressure

	SlotMgrRisk               = slotMgrRisk
	SlotMgrYouthPreference    = slotMgrYouthPreference
	SlotMgrFinancialCaution   = slotMgrFinancialCaution
	SlotMgrPatience           = slotMgrPatience
	SlotMgrRotationPreference = slotMgrRotationPreference
	SlotMgrAttacking          = slotMgrAttacking
	SlotMgrPressing           = slotMgrPressing
	SlotMgrReputation         = slotMgrReputation
	SlotMgrExperience         = slotMgrExperience
	SlotMgrAdaptability       = slotMgrAdaptability
	SlotMgrTenure             = slotMgrTenure
)
