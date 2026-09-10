package models

const (
	ClubRatingMin = 0
	ClubRatingMax = 100
	EuroMillion   = int64(1_000_000)
)

// ClubIdentity contains the small set of persistent traits that define how a
// club behaves in the living universe. All values use the same bounded 0-100
// scale; none of them are match-engine bonuses.
type ClubIdentity struct {
	Reputation             int `json:"reputation"`
	HistoricalPrestige     int `json:"historical_prestige"`
	FinancialPower         int `json:"financial_power"`
	BoardPatience          int `json:"board_patience"`
	AcademyQuality         int `json:"academy_quality"`
	RecruitmentAmbition    int `json:"recruitment_ambition"`
	YouthPreference        int `json:"youth_preference"`
	TransferAggressiveness int `json:"transfer_aggressiveness"`
	SellingTendency        int `json:"selling_tendency"`

	// present distinguishes an explicitly persisted all-zero identity from a
	// missing identity block in legacy/static-data payloads. It is intentionally
	// excluded from JSON and carried through value copies.
	present bool
}

// ClubFinances separates structural financial strength from spendable cash.
type ClubFinances struct {
	TransferBudget int64 `json:"transfer_budget"`
	Balance        int64 `json:"balance"`
}

type clubIdentityPreset struct {
	rep, prestige, financial, patience, academy, ambition, youth, aggression, selling int
}

// Identity values are simulation-balancing presets, not representations of
// audited real-world accounts. They are deliberately readable and distinct.
var clubIdentityPresets = map[string]clubIdentityPreset{
	"LAL-RMA": {96, 100, 98, 38, 88, 100, 72, 92, 20},
	"LAL-BAR": {94, 99, 88, 48, 96, 94, 88, 82, 28},
	"BUN-BAY": {93, 98, 94, 42, 91, 93, 80, 84, 22},
	"EPL-LIV": {92, 97, 92, 58, 86, 91, 75, 82, 24},
	"SEA-MIL": {86, 96, 76, 52, 82, 83, 76, 72, 36},
	"SEA-INT": {88, 93, 82, 46, 78, 88, 67, 81, 31},
	"EPL-ARS": {89, 91, 91, 62, 92, 93, 91, 80, 25},
	"LAL-ATM": {85, 88, 77, 70, 76, 84, 65, 77, 34},
	"BUN-DOR": {84, 86, 75, 66, 95, 88, 98, 83, 58},
	"FL1-PSG": {91, 79, 100, 34, 82, 100, 62, 96, 18},
	"FRA-PSG": {91, 79, 100, 34, 82, 100, 62, 96, 18},
	"SEA-NAP": {83, 80, 73, 54, 75, 83, 70, 78, 42},
	"EPL-TOT": {82, 78, 88, 45, 88, 91, 84, 87, 35},
}

func ClampClubRating(v int) int {
	if v < ClubRatingMin {
		return ClubRatingMin
	}
	if v > ClubRatingMax {
		return ClubRatingMax
	}
	return v
}

func (i ClubIdentity) IsZero() bool {
	return !i.present && i == (ClubIdentity{})
}

// Clamp returns a copy with every identity field constrained to 0-100.
func (i ClubIdentity) Clamp() ClubIdentity {
	i.Reputation = ClampClubRating(i.Reputation)
	i.HistoricalPrestige = ClampClubRating(i.HistoricalPrestige)
	i.FinancialPower = ClampClubRating(i.FinancialPower)
	i.BoardPatience = ClampClubRating(i.BoardPatience)
	i.AcademyQuality = ClampClubRating(i.AcademyQuality)
	i.RecruitmentAmbition = ClampClubRating(i.RecruitmentAmbition)
	i.YouthPreference = ClampClubRating(i.YouthPreference)
	i.TransferAggressiveness = ClampClubRating(i.TransferAggressiveness)
	i.SellingTendency = ClampClubRating(i.SellingTendency)
	return i
}

func DefaultClubIdentity(clubID string, teamRating int) ClubIdentity {
	if p, ok := clubIdentityPresets[clubID]; ok {
		return ClubIdentity{
			Reputation: p.rep, HistoricalPrestige: p.prestige,
			FinancialPower: p.financial, BoardPatience: p.patience,
			AcademyQuality: p.academy, RecruitmentAmbition: p.ambition,
			YouthPreference: p.youth, TransferAggressiveness: p.aggression,
			SellingTendency: p.selling,
		}
	}
	base := ClampClubRating(teamRating)
	if base == 0 {
		base = 65
	}
	return ClubIdentity{
		Reputation: base, HistoricalPrestige: base, FinancialPower: base,
		BoardPatience: 60, AcademyQuality: base, RecruitmentAmbition: base,
		YouthPreference: 60, TransferAggressiveness: 60, SellingTendency: 45,
	}.Clamp()
}

// InitialClubFinances derives a simple opening warchest from structural
// financial power and reputation. It is only used when a universe/old save
// has no finance block; it is never called merely because a club spent to €0.
func InitialClubFinances(identity ClubIdentity) ClubFinances {
	identity = identity.Clamp()
	budgetMillions := int64(20 + (identity.FinancialPower*13)/10 + (identity.Reputation*3)/10)
	if budgetMillions < 25 {
		budgetMillions = 25
	}
	if budgetMillions > 180 {
		budgetMillions = 180
	}
	budget := budgetMillions * EuroMillion
	return ClubFinances{TransferBudget: budget, Balance: budget * 2}
}

func (f ClubFinances) Valid() bool {
	return f.TransferBudget >= 0 && f.Balance >= 0
}
