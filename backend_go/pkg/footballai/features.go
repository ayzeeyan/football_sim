package footballai

// Feature groups (spec section 16): reusable, strongly typed blocks that are
// assembled into task-specific requests. The game runtime never builds
// string-keyed maps; it fills these structs and encodes them into the fixed
// input slot vector.

// PlayerFeatures describes one player for player-level tasks.
type PlayerFeatures struct {
	Age               int
	OVR               float32
	Potential         float32
	Fitness           float32
	Sharpness         float32
	Fatigue           float32
	ConsecutiveStarts int
	MinutesLast14Days float32
	Morale            float32
	FormModifier      float32
	RecentInjury      bool
	MatchImportance   float32
	ReplacementOVR    float32
	AcademyQuality    float32
	TrainingQuality   float32
	Position          Position
	Personality       Personality
}

// ClubFeatures describes club-level economics and standing.
type ClubFeatures struct {
	Reputation          float32
	FinancialPower      float32
	SellingTendency     float32
	RecruitmentAmbition float32
	FinancialPressure   float32
}

// MatchContext describes an upcoming fixture for match-level tasks.
type MatchContext struct {
	HomeRating      float32
	AwayRating      float32
	HomeForm        float32
	AwayForm        float32
	HomeFitness     float32
	AwayFitness     float32
	HomeFatigue     float32
	AwayFatigue     float32
	TacticalEdge    float32
	MatchImportance float32
	Derby           bool
	Rain            bool
	EuropeanNight   bool
	HomeAbsences    int
	AwayAbsences    int
	HomeAttackBias  float32
	AwayAttackBias  float32
}

// FinancialContext describes contract and market economics for a player.
type FinancialContext struct {
	ContractYears int
	Loyalty       float32
	// Money values are raw euros; the stored normalization metadata applies
	// the log transform, identically at training and inference time.
	BaselineAnchorEUR float32
	BaselineWageEUR   float32
	BidPriceEUR       float32
	AskingPriceEUR    float32
	WageHeadroom      float32
	SquadRole         SquadRole
}

// BoardContext describes board/manager pressure for club tasks.
type BoardContext struct {
	PointsPerGame       float32
	ExpectedPPG         float32
	Last5PPG            float32
	BoardPatience       float32
	SpendRatio          float32
	ManagerTenureMonths int
	CupProgress         float32
}

// ManagerFeatures encodes manager characteristics. Managers are represented
// by traits, not identity, so newly generated managers behave sensibly
// before accumulating individual history.
type ManagerFeatures struct {
	Risk               float32
	YouthPreference    float32
	FinancialCaution   float32
	Patience           float32
	RotationPreference float32
	Attacking          float32
	Pressing           float32
	Reputation         float32
	Experience         float32
	Adaptability       float32
	TenureMonths       int
}

// WorldContext is the coarse routing context shared by all tasks.
type WorldContext struct {
	MatchImportance  float32
	PlayerFatigue    float32
	TransferPressure float32
	ClubPressure     float32
}

// encodeInto writes a feature group's raw slots into the input vector.
func (p PlayerFeatures) encodeInto(x []float32) {
	x[slotAge] = float32(p.Age)
	x[slotOVR] = p.OVR
	x[slotPotential] = p.Potential
	x[slotFitness] = p.Fitness
	x[slotSharpness] = p.Sharpness
	x[slotFatigue] = p.Fatigue
	x[slotConsecutiveStarts] = float32(p.ConsecutiveStarts)
	x[slotMinutes14d] = p.MinutesLast14Days
	x[slotMorale] = p.Morale
	x[slotFormModifier] = p.FormModifier
	if p.RecentInjury {
		x[slotRecentInjury] = 1
	}
	x[slotPlayerImportance] = p.MatchImportance
	x[slotReplacementOVR] = p.ReplacementOVR
	x[slotAcademyQuality] = p.AcademyQuality
	x[slotTrainingQuality] = p.TrainingQuality
	x[slotPositionOneHot+int(p.Position)] = 1
	x[slotPersonality+int(p.Personality)] = 1
}

func (c ClubFeatures) encodeInto(x []float32) {
	x[slotClubReputation] = c.Reputation
	x[slotFinancialPower] = c.FinancialPower
	x[slotSellingTendency] = c.SellingTendency
	x[slotRecruitmentAmbition] = c.RecruitmentAmbition
	x[slotFinancialPressure] = c.FinancialPressure
}

func (m MatchContext) encodeInto(x []float32) {
	x[slotHomeRating] = m.HomeRating
	x[slotAwayRating] = m.AwayRating
	x[slotHomeForm] = m.HomeForm
	x[slotAwayForm] = m.AwayForm
	x[slotHomeFitness] = m.HomeFitness
	x[slotAwayFitness] = m.AwayFitness
	x[slotHomeFatigue] = m.HomeFatigue
	x[slotAwayFatigue] = m.AwayFatigue
	x[slotTacticalEdge] = m.TacticalEdge
	x[slotMatchImportance] = m.MatchImportance
	if m.Derby {
		x[slotDerby] = 1
	}
	if m.Rain {
		x[slotRain] = 1
	}
	if m.EuropeanNight {
		x[slotEuropeanNight] = 1
	}
	x[slotHomeAbsences] = float32(m.HomeAbsences)
	x[slotAwayAbsences] = float32(m.AwayAbsences)
	x[slotHomeAttackBias] = m.HomeAttackBias
	x[slotAwayAttackBias] = m.AwayAttackBias
}

func (f FinancialContext) encodeInto(x []float32) {
	x[slotContractYears] = float32(f.ContractYears)
	x[slotLoyalty] = f.Loyalty
	x[slotBaselineAnchor] = f.BaselineAnchorEUR
	x[slotBaselineWage] = f.BaselineWageEUR
	x[slotBidPrice] = f.BidPriceEUR
	x[slotAskingPrice] = f.AskingPriceEUR
	x[slotWageHeadroom] = f.WageHeadroom
	x[slotSquadRoleOneHot+int(f.SquadRole)] = 1
}

func (b BoardContext) encodeInto(x []float32) {
	x[slotPointsPerGame] = b.PointsPerGame
	x[slotExpectedPPG] = b.ExpectedPPG
	x[slotLast5PPG] = b.Last5PPG
	x[slotBoardPatience] = b.BoardPatience
	x[slotSpendRatio] = b.SpendRatio
	x[slotManagerTenureMonths] = float32(b.ManagerTenureMonths)
	x[slotCupProgress] = b.CupProgress
}

func (m ManagerFeatures) encodeInto(x []float32) {
	x[slotMgrRisk] = m.Risk
	x[slotMgrYouthPreference] = m.YouthPreference
	x[slotMgrFinancialCaution] = m.FinancialCaution
	x[slotMgrPatience] = m.Patience
	x[slotMgrRotationPreference] = m.RotationPreference
	x[slotMgrAttacking] = m.Attacking
	x[slotMgrPressing] = m.Pressing
	x[slotMgrReputation] = m.Reputation
	x[slotMgrExperience] = m.Experience
	x[slotMgrAdaptability] = m.Adaptability
	x[slotMgrTenure] = float32(m.TenureMonths)
}

func (w WorldContext) encodeInto(x []float32) {
	x[slotWorldImportance] = w.MatchImportance
	x[slotWorldFatigue] = w.PlayerFatigue
	x[slotTransferPressure] = w.TransferPressure
	x[slotClubPressure] = w.ClubPressure
}

// ---------------------------------------------------------------------------
// Strongly typed task requests.
// ---------------------------------------------------------------------------

// MatchRequest asks for expected goals before a fixture. The simulation RNG
// still resolves the actual scoreline.
type MatchRequest struct {
	Match   MatchContext
	World   WorldContext
	Manager ManagerFeatures
}

// InjuryRequest asks for a player's injury probability for the next match.
type InjuryRequest struct {
	Player PlayerFeatures
	World  WorldContext
}

// RotationRequest asks for start/rest scores for one squad player.
type RotationRequest struct {
	Player  PlayerFeatures
	World   WorldContext
	Manager ManagerFeatures
}

// DevelopmentRequest asks for the expected season OVR delta of a player.
type DevelopmentRequest struct {
	Player PlayerFeatures
	World  WorldContext
}

// DeclineRequest asks for the expected season OVR decline of a player.
type DeclineRequest struct {
	Player PlayerFeatures
	World  WorldContext
}

// ValuationRequest asks for a market-value premium multiplier over the
// deterministic baseline valuation.
type ValuationRequest struct {
	Player PlayerFeatures
	Club   ClubFeatures
	Fin    FinancialContext
	World  WorldContext
}

// NegotiationRequest asks for accept/counter/walk-away probabilities for one
// incoming bid.
type NegotiationRequest struct {
	Player PlayerFeatures
	Club   ClubFeatures
	Fin    FinancialContext
	World  WorldContext
}

// ContractRequest asks for renewal probability and expected wage demand.
type ContractRequest struct {
	Player PlayerFeatures
	Club   ClubFeatures
	Fin    FinancialContext
	World  WorldContext
}

// BoardPatienceRequest asks for the manager sack probability.
type BoardPatienceRequest struct {
	Board   BoardContext
	Club    ClubFeatures
	Manager ManagerFeatures
	World   WorldContext
}

// Request is the encoded internal form of any task request.
type Request struct {
	Task  TaskType
	Slots []float32
}

func newRequest(task TaskType, groups ...func([]float32)) Request {
	x := make([]float32, InputWidth)
	for _, g := range groups {
		g(x)
	}
	return Request{Task: task, Slots: x}
}

// Task returns the task type served by the request.
func (r MatchRequest) Task() TaskType         { return TaskMatchPrediction }
func (r InjuryRequest) Task() TaskType        { return TaskInjuryRisk }
func (r RotationRequest) Task() TaskType      { return TaskRotation }
func (r DevelopmentRequest) Task() TaskType   { return TaskDevelopment }
func (r DeclineRequest) Task() TaskType       { return TaskDecline }
func (r ValuationRequest) Task() TaskType     { return TaskValuation }
func (r NegotiationRequest) Task() TaskType   { return TaskNegotiation }
func (r ContractRequest) Task() TaskType      { return TaskContract }
func (r BoardPatienceRequest) Task() TaskType { return TaskBoardPatience }

// Encode converts the request into the fixed-width slot vector.
func (r MatchRequest) Encode() Request {
	return newRequest(TaskMatchPrediction, r.Match.encodeInto, r.World.encodeInto, r.Manager.encodeInto)
}

func (r InjuryRequest) Encode() Request {
	return newRequest(TaskInjuryRisk, r.Player.encodeInto, r.World.encodeInto)
}

func (r RotationRequest) Encode() Request {
	return newRequest(TaskRotation, r.Player.encodeInto, r.World.encodeInto, r.Manager.encodeInto)
}

func (r DevelopmentRequest) Encode() Request {
	return newRequest(TaskDevelopment, r.Player.encodeInto, r.World.encodeInto)
}

func (r DeclineRequest) Encode() Request {
	return newRequest(TaskDecline, r.Player.encodeInto, r.World.encodeInto)
}

func (r ValuationRequest) Encode() Request {
	return newRequest(TaskValuation, r.Player.encodeInto, r.Club.encodeInto, r.Fin.encodeInto, r.World.encodeInto)
}

func (r NegotiationRequest) Encode() Request {
	return newRequest(TaskNegotiation, r.Player.encodeInto, r.Club.encodeInto, r.Fin.encodeInto, r.World.encodeInto)
}

func (r ContractRequest) Encode() Request {
	return newRequest(TaskContract, r.Player.encodeInto, r.Club.encodeInto, r.Fin.encodeInto, r.World.encodeInto)
}

func (r BoardPatienceRequest) Encode() Request {
	return newRequest(TaskBoardPatience, r.Board.encodeInto, r.Club.encodeInto, r.Manager.encodeInto, r.World.encodeInto)
}
