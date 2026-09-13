package tournament

import (
	"fmt"
	"math"

	"football_sim/pkg/models"
	"football_sim/pkg/transfers"
)

// ValidateWorldState checks durable universe invariants without mutating state.
// It is intended for save loading, tests, soak runs, and developer diagnostics.
func (tm *TournamentManager) ValidateWorldState() error {
	if tm == nil {
		return fmt.Errorf("world validation: tournament manager is nil")
	}
	if tm.SeasonName == "" {
		return fmt.Errorf("world validation: season name is empty")
	}
	if tm.SeasonPhase != "season" && tm.SeasonPhase != "transfer_window" {
		return fmt.Errorf("world validation: unknown season phase %q", tm.SeasonPhase)
	}
	if tm.World == nil && tm.MaxMatchweeks != LeagueRounds {
		return fmt.Errorf("world validation: max matchweeks=%d want %d", tm.MaxMatchweeks, LeagueRounds)
	}
	if tm.World != nil && tm.MaxMatchweeks != 38 {
		return fmt.Errorf("world validation: European calendar has %d matchweeks, want 38", tm.MaxMatchweeks)
	}
	if tm.CurrentMatchweek < 1 || tm.CurrentMatchweek > tm.MaxMatchweeks+1 {
		return fmt.Errorf("world validation: current matchweek %d outside legal range 1..%d", tm.CurrentMatchweek, tm.MaxMatchweeks+1)
	}

	clubIDs := make(map[string]struct{}, len(tm.ClubsList))
	playerClub := make(map[string]string)
	for i, club := range tm.ClubsList {
		if club == nil {
			return fmt.Errorf("world validation: clubs list contains nil club at index %d", i)
		}
		if club.ClubID == "" {
			return fmt.Errorf("world validation: club at index %d has empty id", i)
		}
		if _, exists := clubIDs[club.ClubID]; exists {
			return fmt.Errorf("world validation: duplicate club id %q", club.ClubID)
		}
		clubIDs[club.ClubID] = struct{}{}
		if mapped := tm.Clubs[club.ClubID]; mapped != club {
			return fmt.Errorf("world validation: club %q map/list references disagree", club.ClubID)
		}
		if err := validateClubState(club, playerClub); err != nil {
			return err
		}
	}
	if len(tm.Clubs) != len(clubIDs) {
		return fmt.Errorf("world validation: club map/list size mismatch map=%d list=%d", len(tm.Clubs), len(clubIDs))
	}

	if tm.World != nil {
		if err := validateEuropeanWorldSchedule(tm, clubIDs); err != nil {
			return err
		}
	} else {
		if err := validateLeagueSchedule(tm, clubIDs); err != nil {
			return err
		}
	}
	worldFixtureCount := 0
	if tm.World != nil {
		worldFixtureCount = len(tm.World.Fixtures)
	}
	fixtureIDs := make(map[string]struct{}, len(tm.Fixtures)+len(tm.UCLFixtures)+len(tm.SuperCupFixtures)+worldFixtureCount)
	for scope, fixtures := range map[string][]Fixture{"league": tm.Fixtures, "ucl": tm.UCLFixtures, "super_cup": tm.SuperCupFixtures} {
		for i := range fixtures {
			if err := validateFixtureState(scope, i, &fixtures[i], clubIDs, fixtureIDs); err != nil {
				return err
			}
		}
	}
	if tm.World != nil {
		for i := range tm.World.Fixtures {
			if err := validateFixtureState("world", i, &tm.World.Fixtures[i], clubIDs, fixtureIDs); err != nil {
				return err
			}
		}
	}

	for clubID, manager := range tm.Managers {
		if _, ok := clubIDs[clubID]; !ok {
			return fmt.Errorf("world validation: manager references unknown club %q", clubID)
		}
		if manager == nil {
			return fmt.Errorf("world validation: active manager for club %q is nil", clubID)
		}
		if manager.ClubID != "" && manager.ClubID != clubID {
			return fmt.Errorf("world validation: manager %q club id %q does not match map key %q", manager.Name, manager.ClubID, clubID)
		}
		if manager.BudgetEur < 0 {
			return fmt.Errorf("world validation: manager %q has negative budget %d", manager.Name, manager.BudgetEur)
		}
	}
	for _, entry := range tm.ManagerHistory {
		if entry.ClubID != "" {
			if _, ok := clubIDs[entry.ClubID]; !ok {
				return fmt.Errorf("world validation: manager history references unknown club %q", entry.ClubID)
			}
		}
		if entry.Matchweek < 0 {
			return fmt.Errorf("world validation: manager history has negative matchweek for club %q", entry.ClubID)
		}
	}

	if tm.TransferEngine != nil {
		te := tm.TransferEngine
		if te.WindowType != transfers.WindowClosed && te.WindowType != transfers.WindowSummer && te.WindowType != transfers.WindowWinter {
			return fmt.Errorf("world validation: unknown transfer window type %q", te.WindowType)
		}
		if te.IsWindowOpen() {
			if te.CurrentWeek < 1 || te.CurrentWeek > te.WindowWeeks() {
				return fmt.Errorf("world validation: open %s window week %d outside legal range 1..%d", te.WindowType, te.CurrentWeek, te.WindowWeeks())
			}
		} else if te.CurrentWeek < 0 || te.CurrentWeek > transfers.TransferWindowWeeks {
			return fmt.Errorf("world validation: closed transfer week %d outside legal range 0..%d", te.CurrentWeek, transfers.TransferWindowWeeks)
		}
		if te.CurrentDay < 0 || te.CurrentMatchweek < 0 {
			return fmt.Errorf("world validation: transfer counters cannot be negative (day=%d matchweek=%d)", te.CurrentDay, te.CurrentMatchweek)
		}
		// While the window is active, every transfer lock must identify a player
		// that is still part of the active universe. After the window closes we
		// intentionally retain those locks until the next BeginOffSeasonWindow;
		// a player may retire during the season transition in the meantime.
		if te.IsOffSeason {
			for playerID, moved := range te.TransferredThisWindow {
				if !moved {
					continue
				}
				if _, ok := playerClub[playerID]; !ok {
					return fmt.Errorf("world validation: active-window transfer marker player %q is not in an active squad", playerID)
				}
			}
		}
		seenCompleted := map[string]bool{}
		for _, transfer := range te.CompletedTransfers {
			if transfer.FeeEUR < 0 {
				return fmt.Errorf("world validation: completed transfer for player %q has negative fee %d", transfer.PlayerID, transfer.FeeEUR)
			}
			// Historical/foreign counterparties outside the active map are
			// explicitly permitted: snapshot validation allows them for
			// multi-season saves (pinned by TestValidateCareerSnapshotAllows-
			// ExpiredNegotiationsAndHistoricalBuyers), so rejecting them here
			// would fatal saves that passed snapshot validation at boot.
			// Marker checks are likewise scoped to fully-local deals, but the
			// duplicate-player check stays universal: CompletedTransfers is
			// window-local by construction (cleared every ResetForNewSeason).
			_, sellerKnown := clubIDs[transfer.SellerID]
			_, buyerKnown := clubIDs[transfer.BuyerID]
			historical := (transfer.SellerID != "" && !sellerKnown) || (transfer.BuyerID != "" && !buyerKnown)
			if seenCompleted[transfer.PlayerID] {
				return fmt.Errorf("world validation: player %q completed more than one transfer in the active window", transfer.PlayerID)
			}
			seenCompleted[transfer.PlayerID] = true
			if !historical && !te.TransferredThisWindow[transfer.PlayerID] {
				return fmt.Errorf("world validation: completed transfer player %q lacks transferred-this-window marker", transfer.PlayerID)
			}
		}
		for _, negotiation := range te.ActiveNegotiations {
			if negotiation == nil {
				return fmt.Errorf("world validation: active transfer negotiations contain nil entry")
			}
			if negotiation.CurrentBid < 0 || negotiation.AskingPrice < 0 {
				return fmt.Errorf("world validation: negotiation %q has negative financial value", negotiation.NegotiationID)
			}
			if negotiation.Buyer == nil || negotiation.Seller == nil || negotiation.Player == nil {
				return fmt.Errorf("world validation: negotiation %q has incomplete participants", negotiation.NegotiationID)
			}
			if _, ok := clubIDs[negotiation.Buyer.ClubID]; !ok {
				return fmt.Errorf("world validation: negotiation %q references unknown buyer %q", negotiation.NegotiationID, negotiation.Buyer.ClubID)
			}
			if _, ok := clubIDs[negotiation.Seller.ClubID]; !ok {
				return fmt.Errorf("world validation: negotiation %q references unknown seller %q", negotiation.NegotiationID, negotiation.Seller.ClubID)
			}
		}
	}

	if tm.GrowthEngine != nil {
		for playerID, bio := range tm.GrowthEngine.Biometrics {
			if bio == nil {
				return fmt.Errorf("world validation: biometric profile for player %q is nil", playerID)
			}
			if bio.PlayerID != "" && bio.PlayerID != playerID {
				return fmt.Errorf("world validation: biometric map key %q does not match profile player id %q", playerID, bio.PlayerID)
			}
			for name, value := range map[string]float64{
				"current height": bio.CurrentHeightCM, "baseline height": bio.BaselineHeightCM,
				"current weight": bio.CurrentWeightKG, "baseline weight": bio.BaselineWeightKG,
				"growth velocity": bio.GrowthVelocity, "accumulated XP": bio.AccumulatedXP,
				"level XP target": bio.LevelXPTarget, "yearly height taken": bio.YearlyHeightTaken,
			} {
				if math.IsNaN(value) || math.IsInf(value, 0) {
					return fmt.Errorf("world validation: player %q biometric %s is not finite", playerID, name)
				}
			}
			if bio.Age < 0 || bio.Potential < 0 || bio.Potential > 100 || bio.CurrentHeightCM <= 0 || bio.CurrentWeightKG <= 0 || bio.LevelXPTarget < 0 || bio.AccumulatedXP < 0 {
				return fmt.Errorf("world validation: player %q has invalid biometric bounds", playerID)
			}
			if models.IsCanonicalWonderkidID(playerID) && (bio.Potential < 93 || bio.Potential > 96) {
				return fmt.Errorf("world validation: canonical wonderkid %q potential %d outside [93, 96]", playerID, bio.Potential)
			}
		}
	}
	return nil
}

func validateClubState(club *models.Club, playerIDs map[string]string) error {
	identity := club.Identity
	traits := map[string]int{
		"reputation": identity.Reputation, "historical_prestige": identity.HistoricalPrestige,
		"financial_power": identity.FinancialPower, "board_patience": identity.BoardPatience,
		"academy_quality": identity.AcademyQuality, "recruitment_ambition": identity.RecruitmentAmbition,
		"youth_preference": identity.YouthPreference, "transfer_aggressiveness": identity.TransferAggressiveness,
		"selling_tendency": identity.SellingTendency,
	}
	for name, value := range traits {
		if value < models.ClubRatingMin || value > models.ClubRatingMax {
			return fmt.Errorf("world validation: club %q identity %s=%d outside 0..100", club.ClubID, name, value)
		}
	}
	if !club.Finances.Valid() {
		return fmt.Errorf("world validation: club %q has negative finances budget=%d balance=%d", club.ClubID, club.Finances.TransferBudget, club.Finances.Balance)
	}
	if club.Finances.TransferBudget > club.Finances.Balance {
		return fmt.Errorf("world validation: club %q transfer budget %d exceeds balance %d", club.ClubID, club.Finances.TransferBudget, club.Finances.Balance)
	}
	if club.Played < 0 || club.Won < 0 || club.Drawn < 0 || club.Lost < 0 || club.GoalsFor < 0 || club.GoalsAgainst < 0 || club.Points < 0 {
		return fmt.Errorf("world validation: club %q has negative standings values", club.ClubID)
	}
	if club.Played != club.Won+club.Drawn+club.Lost {
		return fmt.Errorf("world validation: club %q played=%d but W+D+L=%d", club.ClubID, club.Played, club.Won+club.Drawn+club.Lost)
	}
	if club.GoalDifference != club.GoalsFor-club.GoalsAgainst {
		return fmt.Errorf("world validation: club %q goal difference=%d but GF-GA=%d", club.ClubID, club.GoalDifference, club.GoalsFor-club.GoalsAgainst)
	}
	if club.OverallTeamRating < 0 || club.OverallTeamRating > 100 {
		return fmt.Errorf("world validation: club %q team rating %d outside 0..100", club.ClubID, club.OverallTeamRating)
	}
	if club.Coefficient < 0 {
		return fmt.Errorf("world validation: club %q has negative coefficient %d", club.ClubID, club.Coefficient)
	}
	for i, player := range club.Squad {
		if player == nil || player.PlayerID == "" {
			return fmt.Errorf("world validation: club %q has nil/empty-id player at index %d", club.ClubID, i)
		}
		if previousClub, exists := playerIDs[player.PlayerID]; exists {
			return fmt.Errorf("world validation: duplicate player id %q in clubs %q and %q", player.PlayerID, previousClub, club.ClubID)
		}
		playerIDs[player.PlayerID] = club.ClubID
		if player.ClubID != club.ClubID {
			return fmt.Errorf("world validation: player %q claims club %q but is in squad %q", player.PlayerID, player.ClubID, club.ClubID)
		}
		if transfers.IsCanonicalWonderkid(player) && !transfers.IsDesignatedSuperLeagueClub(player.ClubID) {
			return fmt.Errorf("world validation: canonical wonderkid %q is outside designated 12-club ecosystem at %q", player.PlayerID, player.ClubID)
		}
		if player.Goals < 0 || player.Assists < 0 || player.Appearances < 0 || player.CareerGoals < 0 || player.CareerAssists < 0 || player.CareerApps < 0 {
			return fmt.Errorf("world validation: player %q has negative statistics", player.PlayerID)
		}
		if player.OVR < 0 || player.OVR > 100 || player.Age < 0 || player.MarketValueEUR < 0 || player.WageEUR < 0 {
			return fmt.Errorf("world validation: player %q has invalid OVR/age/value/wage", player.PlayerID)
		}
		if player.LoanBuyClauseEUR < 0 || player.LoanBuyClauseEUR > 500_000_000 {
			return fmt.Errorf("world validation: player %q has out-of-bounds loan buy clause %d", player.PlayerID, player.LoanBuyClauseEUR)
		}
		if player.LoanBuyClauseEUR > 0 && player.LoanBuyClauseEUR < 300_000 {
			return fmt.Errorf("world validation: player %q loan buy clause %d below €300k floor", player.PlayerID, player.LoanBuyClauseEUR)
		}
		if player.UniverseWonderkid && player.LoanBuyClauseEUR > 0 {
			return fmt.Errorf("world validation: canonical wonderkid %q must not carry a loan buy clause", player.PlayerID)
		}
		if player.UniverseWonderkid && player.OnLoan {
			return fmt.Errorf("world validation: canonical wonderkid %q must not be on loan", player.PlayerID)
		}
		if player.Leadership < 0 || player.Leadership > 100 {
			return fmt.Errorf("world validation: player %q leadership %d outside 0..100", player.PlayerID, player.Leadership)
		}
		if player.Versatility < 0 || player.Versatility > 100 {
			return fmt.Errorf("world validation: player %q versatility %d outside 0..100", player.PlayerID, player.Versatility)
		}
		if player.CleanSheets < 0 || player.CareerCleanSheets < 0 {
			return fmt.Errorf("world validation: player %q has negative clean sheets", player.PlayerID)
		}
	}
	if club.Chemistry < 0 || club.Chemistry > 100 {
		return fmt.Errorf("world validation: club %q chemistry %d outside 0..100", club.ClubID, club.Chemistry)
	}
	if club.MediaPressure < 0 || club.MediaPressure > 100 {
		return fmt.Errorf("world validation: club %q media pressure %d outside 0..100", club.ClubID, club.MediaPressure)
	}
	if club.FanExpectation < 0 || club.FanExpectation > 100 {
		return fmt.Errorf("world validation: club %q fan expectation %d outside 0..100", club.ClubID, club.FanExpectation)
	}
	captains := 0
	for _, player := range club.Squad {
		if player != nil && player.IsCaptain {
			captains++
			if club.CaptainID != "" && player.PlayerID != club.CaptainID {
				return fmt.Errorf("world validation: club %q captain flag on %q but captain_id=%q", club.ClubID, player.PlayerID, club.CaptainID)
			}
		}
	}
	if captains > 1 {
		return fmt.Errorf("world validation: club %q has %d captains", club.ClubID, captains)
	}
	if club.CaptainID != "" {
		found := false
		for _, player := range club.Squad {
			if player != nil && player.PlayerID == club.CaptainID {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("world validation: club %q captain %q is not in the squad", club.ClubID, club.CaptainID)
		}
	}
	return nil
}

func validateLeagueSchedule(tm *TournamentManager, clubIDs map[string]struct{}) error {
	clubCount := len(clubIDs)
	if clubCount < 2 {
		return fmt.Errorf("world validation: league needs at least two clubs")
	}
	wantFixtures := 2 * clubCount * (clubCount - 1)
	if len(tm.Fixtures) != wantFixtures {
		return fmt.Errorf("world validation: league fixtures=%d want %d for %d-club four-cycle schedule", len(tm.Fixtures), wantFixtures, clubCount)
	}
	perWeek := map[int]map[string]bool{}
	directed := map[string]int{}
	clubGames := map[string]int{}
	for _, f := range tm.Fixtures {
		if f.Matchweek < 1 || f.Matchweek > LeagueRounds {
			return fmt.Errorf("world validation: league fixture %q has matchweek %d outside 1..%d", f.FixtureID, f.Matchweek, LeagueRounds)
		}
		if perWeek[f.Matchweek] == nil {
			perWeek[f.Matchweek] = map[string]bool{}
		}
		if perWeek[f.Matchweek][f.HomeID] || perWeek[f.Matchweek][f.AwayID] {
			return fmt.Errorf("world validation: club appears more than once in league matchweek %d", f.Matchweek)
		}
		perWeek[f.Matchweek][f.HomeID], perWeek[f.Matchweek][f.AwayID] = true, true
		directed[f.HomeID+">"+f.AwayID]++
		clubGames[f.HomeID]++
		clubGames[f.AwayID]++
	}
	for id := range clubIDs {
		if clubGames[id] != 4*(clubCount-1) {
			return fmt.Errorf("world validation: club %q has %d league fixtures want %d", id, clubGames[id], 4*(clubCount-1))
		}
	}
	for a := range clubIDs {
		for b := range clubIDs {
			if a == b {
				continue
			}
			if directed[a+">"+b] != 2 {
				return fmt.Errorf("world validation: directed league pairing %s>%s occurs %d times want 2", a, b, directed[a+">"+b])
			}
		}
	}
	return nil
}

func validateEuropeanWorldSchedule(tm *TournamentManager, clubIDs map[string]struct{}) error {
	if tm.World == nil || len(tm.World.Competitions) == 0 {
		return fmt.Errorf("world validation: European career has no competition registry")
	}
	leagueMembership := map[string]string{}
	for _, def := range domesticLeagueDefinitions {
		comp := tm.World.Competitions[def.ID]
		if comp == nil || comp.Kind != CompetitionLeague {
			return fmt.Errorf("world validation: missing domestic league %q", def.ID)
		}
		n := len(comp.ParticipantIDs)
		if n < 2 {
			return fmt.Errorf("world validation: league %q needs at least two clubs", def.ID)
		}
		seenParticipants := map[string]bool{}
		for _, id := range comp.ParticipantIDs {
			club := tm.Clubs[id]
			if club == nil {
				return fmt.Errorf("world validation: league %q references unknown club %q", def.ID, id)
			}
			if club.League != def.League {
				return fmt.Errorf("world validation: club %q is in %q but registered to %q", id, club.League, def.Name)
			}
			if seenParticipants[id] || leagueMembership[id] != "" {
				return fmt.Errorf("world validation: club %q has duplicate domestic membership", id)
			}
			seenParticipants[id] = true
			leagueMembership[id] = def.ID
		}

		fixtures := tm.worldCompetitionFixturesUnlocked(def.ID)
		wantFixtures := n * (n - 1)
		if len(fixtures) != wantFixtures {
			return fmt.Errorf("world validation: %s fixtures=%d want %d", def.Name, len(fixtures), wantFixtures)
		}
		games, homes, aways := map[string]int{}, map[string]int{}, map[string]int{}
		directed := map[string]int{}
		perWeek := map[int]map[string]bool{}
		for _, f := range fixtures {
			if f.Matchweek < 1 || f.Matchweek > 2*(n-1) {
				return fmt.Errorf("world validation: %s fixture %q has invalid matchweek %d", def.Name, f.FixtureID, f.Matchweek)
			}
			if !seenParticipants[f.HomeID] || !seenParticipants[f.AwayID] {
				return fmt.Errorf("world validation: %s fixture %q crosses league membership", def.Name, f.FixtureID)
			}
			if perWeek[f.Matchweek] == nil {
				perWeek[f.Matchweek] = map[string]bool{}
			}
			if perWeek[f.Matchweek][f.HomeID] || perWeek[f.Matchweek][f.AwayID] {
				return fmt.Errorf("world validation: %s schedules a club twice in matchweek %d", def.Name, f.Matchweek)
			}
			perWeek[f.Matchweek][f.HomeID], perWeek[f.Matchweek][f.AwayID] = true, true
			games[f.HomeID]++
			games[f.AwayID]++
			homes[f.HomeID]++
			aways[f.AwayID]++
			directed[f.HomeID+">"+f.AwayID]++
		}
		for id := range seenParticipants {
			if games[id] != 2*(n-1) || homes[id] != n-1 || aways[id] != n-1 {
				return fmt.Errorf("world validation: %s club %q has games/home/away %d/%d/%d want %d/%d/%d", def.Name, id, games[id], homes[id], aways[id], 2*(n-1), n-1, n-1)
			}
			for other := range seenParticipants {
				if id != other && directed[id+">"+other] != 1 {
					return fmt.Errorf("world validation: %s directed pairing %s>%s occurs %d times", def.Name, id, other, directed[id+">"+other])
				}
			}
		}
	}
	if len(leagueMembership) != len(clubIDs) {
		return fmt.Errorf("world validation: domestic membership covers %d clubs, universe has %d", len(leagueMembership), len(clubIDs))
	}

	europeanMembership := map[string]string{}
	for _, def := range europeanDefinitions {
		comp := tm.World.Competitions[def.ID]
		if comp == nil || comp.Kind != CompetitionEuropean {
			return fmt.Errorf("world validation: missing European competition %q", def.ID)
		}
		if len(comp.ParticipantIDs) == 0 {
			continue
		}
		for _, id := range comp.ParticipantIDs {
			if _, ok := clubIDs[id]; !ok {
				return fmt.Errorf("world validation: %s references unknown participant %q", def.Name, id)
			}
			if previous := europeanMembership[id]; previous != "" {
				return fmt.Errorf("world validation: club %q appears in both %s and %s", id, previous, def.ID)
			}
			europeanMembership[id] = def.ID
			if comp.Records[id] == nil {
				return fmt.Errorf("world validation: %s has no league-phase record for %q", def.Name, id)
			}
		}
	}
	return nil
}

func validateFixtureState(scope string, index int, fixture *Fixture, clubIDs map[string]struct{}, fixtureIDs map[string]struct{}) error {
	if fixture == nil || fixture.FixtureID == "" {
		return fmt.Errorf("world validation: %s fixture %d has nil/empty id", scope, index)
	}
	if _, exists := fixtureIDs[fixture.FixtureID]; exists {
		return fmt.Errorf("world validation: duplicate fixture id %q", fixture.FixtureID)
	}
	fixtureIDs[fixture.FixtureID] = struct{}{}
	if fixture.HomeID == "" || fixture.AwayID == "" || fixture.HomeID == fixture.AwayID {
		return fmt.Errorf("world validation: fixture %q has invalid home/away ids", fixture.FixtureID)
	}
	if _, ok := clubIDs[fixture.HomeID]; !ok {
		return fmt.Errorf("world validation: fixture %q references unknown home club %q", fixture.FixtureID, fixture.HomeID)
	}
	if _, ok := clubIDs[fixture.AwayID]; !ok {
		return fmt.Errorf("world validation: fixture %q references unknown away club %q", fixture.FixtureID, fixture.AwayID)
	}
	if fixture.Home != nil && fixture.Home.ClubID != fixture.HomeID {
		return fmt.Errorf("world validation: fixture %q home pointer does not match home id", fixture.FixtureID)
	}
	if fixture.Away != nil && fixture.Away.ClubID != fixture.AwayID {
		return fmt.Errorf("world validation: fixture %q away pointer does not match away id", fixture.FixtureID)
	}
	if fixture.Status != "" && fixture.Status != "scheduled" && fixture.Status != "playing" && fixture.Status != "finished" {
		return fmt.Errorf("world validation: fixture %q has unknown status %q", fixture.FixtureID, fixture.Status)
	}
	if (fixture.HomeGoals == nil) != (fixture.AwayGoals == nil) {
		return fmt.Errorf("world validation: fixture %q has only one goal value set", fixture.FixtureID)
	}
	if fixture.HomeGoals != nil && (*fixture.HomeGoals < 0 || *fixture.AwayGoals < 0) {
		return fmt.Errorf("world validation: fixture %q has negative score", fixture.FixtureID)
	}
	if fixture.Status == "finished" && fixture.HomeGoals == nil {
		return fmt.Errorf("world validation: finished fixture %q has no result", fixture.FixtureID)
	}
	return nil
}
