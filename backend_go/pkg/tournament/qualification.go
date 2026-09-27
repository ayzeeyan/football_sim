package tournament

import (
	"sort"
)

const (
	compChampionsLeague  = "champions-league"
	compEuropaLeague     = "europa-league"
	compConferenceLeague = "conference-league"

	sourceLeaguePosition   = "League position"
	sourceDomesticCup      = "Domestic cup winners"
	sourceUCLHolder        = "Champions League holders"
	sourceEuropaHolder     = "Europa League holders"
	sourceConferenceHolder = "Conference League holders"
	sourceRebalance        = "Access list rebalance"
)

func europeanTargetSize(compID string) int {
	switch compID {
	case compChampionsLeague:
		return 36
	case compEuropaLeague, compConferenceLeague:
		return 20
	default:
		return 0
	}
}

var europeanCompOrder = []string{compChampionsLeague, compEuropaLeague, compConferenceLeague}

type qualBook struct {
	assigned map[string]string
	source   map[string]map[string]string
}

func newQualBook() *qualBook {
	return &qualBook{
		assigned: map[string]string{},
		source: map[string]map[string]string{
			compChampionsLeague:  {},
			compEuropaLeague:     {},
			compConferenceLeague: {},
		},
	}
}

func (b *qualBook) add(compID, clubID, source string) bool {
	if b == nil || clubID == "" || compID == "" {
		return false
	}
	if b.assigned[clubID] != "" {
		return false
	}
	b.assigned[clubID] = compID
	if b.source[compID] == nil {
		b.source[compID] = map[string]string{}
	}
	b.source[compID][clubID] = source
	return true
}

func (b *qualBook) promote(clubID, compID, source string) {
	if b == nil || clubID == "" || compID == "" {
		return
	}
	if old := b.assigned[clubID]; old != "" && b.source[old] != nil {
		delete(b.source[old], clubID)
	}
	b.assigned[clubID] = compID
	if b.source[compID] == nil {
		b.source[compID] = map[string]string{}
	}
	b.source[compID][clubID] = source
}

func (b *qualBook) drop(clubID string) {
	if b == nil || clubID == "" {
		return
	}
	if old := b.assigned[clubID]; old != "" && b.source[old] != nil {
		delete(b.source[old], clubID)
	}
	delete(b.assigned, clubID)
}

func (b *qualBook) clubsIn(compID string) []string {
	ids := make([]string, 0, len(b.source[compID]))
	for id := range b.source[compID] {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (b *qualBook) result() map[string]map[string]string {
	out := map[string]map[string]string{
		compChampionsLeague:  {},
		compEuropaLeague:     {},
		compConferenceLeague: {},
	}
	for _, compID := range europeanCompOrder {
		for _, clubID := range b.clubsIn(compID) {
			out[compID][clubID] = b.source[compID][clubID]
		}
	}
	return out
}

func isTitleholderSource(source string) bool {
	return source == sourceUCLHolder || source == sourceEuropaHolder || source == sourceConferenceHolder
}

func (tm *TournamentManager) nextEuropeanQualificationUnlocked() map[string]map[string]string {
	return tm.resolveEuropeanQualificationUnlocked()
}

func (tm *TournamentManager) resolveEuropeanQualificationUnlocked() map[string]map[string]string {
	book := newQualBook()
	var reserves []string
	seenReserve := map[string]bool{}

	for _, league := range domesticLeagueDefinitions {
		table := tm.worldLeagueStandingsUnlocked(league.ID)
		if len(table) == 0 {
			continue
		}
		uclWant := championsLeagueOpeningSlots(len(table))
		taken := 0
		for _, club := range table {
			if club == nil || taken >= uclWant {
				break
			}
			if book.add(compChampionsLeague, club.ClubID, sourceLeaguePosition) {
				taken++
			}
		}
		cupWinner := ""
		for _, cup := range domesticCupDefinitions {
			if cup.League != league.League {
				continue
			}
			if comp := tm.worldCompetitionUnlocked(cup.ID); comp != nil {
				cupWinner = comp.ChampionID
			}
			break
		}
		elTaken := 0
		if book.add(compEuropaLeague, cupWinner, sourceDomesticCup) {
			elTaken++
		}
		for _, club := range table {
			if club == nil || elTaken >= 4 {
				break
			}
			if book.add(compEuropaLeague, club.ClubID, sourceLeaguePosition) {
				elTaken++
			}
		}
		uelTaken := 0
		for _, club := range table {
			if club == nil || uelTaken >= 4 {
				break
			}
			if book.add(compConferenceLeague, club.ClubID, sourceLeaguePosition) {
				uelTaken++
			}
		}
		for _, club := range table {
			if club == nil || book.assigned[club.ClubID] != "" || seenReserve[club.ClubID] {
				continue
			}
			seenReserve[club.ClubID] = true
			reserves = append(reserves, club.ClubID)
		}
	}

	tm.applyTitleholderEntitlement(book, tm.europeanChampionIDUnlocked(compChampionsLeague), compChampionsLeague, sourceUCLHolder)
	tm.applyTitleholderEntitlement(book, tm.europeanChampionIDUnlocked(compEuropaLeague), compChampionsLeague, sourceEuropaHolder)
	tm.applyTitleholderEntitlement(book, tm.europeanChampionIDUnlocked(compConferenceLeague), compEuropaLeague, sourceConferenceHolder)
	tm.rebalanceEuropeanAccess(book, reserves)
	return book.result()
}

func (tm *TournamentManager) europeanChampionIDUnlocked(compID string) string {
	if comp := tm.worldCompetitionUnlocked(compID); comp != nil {
		return comp.ChampionID
	}
	return ""
}

func (tm *TournamentManager) applyTitleholderEntitlement(book *qualBook, clubID, target, source string) {
	if book == nil || clubID == "" || tm.Clubs[clubID] == nil {
		return
	}
	current := book.assigned[clubID]
	if current == target {
		return
	}
	if current == "" {
		book.add(target, clubID, source)
		return
	}
	if europeanRank(current) > europeanRank(target) {
		book.promote(clubID, target, source)
	}
}

func europeanRank(compID string) int {
	switch compID {
	case compChampionsLeague:
		return 0
	case compEuropaLeague:
		return 1
	case compConferenceLeague:
		return 2
	default:
		return 9
	}
}

func (tm *TournamentManager) rebalanceEuropeanAccess(book *qualBook, reserves []string) {
	if book == nil {
		return
	}
	nextLower := map[string]string{
		compChampionsLeague:  compEuropaLeague,
		compEuropaLeague:     compConferenceLeague,
		compConferenceLeague: "",
	}
	for _, compID := range europeanCompOrder {
		want := europeanTargetSize(compID)
		for len(book.clubsIn(compID)) > want {
			dropID := tm.worstRebalanceCandidate(book, compID)
			if dropID == "" {
				break
			}
			lower := nextLower[compID]
			book.drop(dropID)
			if lower != "" {
				book.add(lower, dropID, sourceRebalance)
			}
		}
	}
	for _, compID := range europeanCompOrder {
		want := europeanTargetSize(compID)
		for len(book.clubsIn(compID)) < want {
			var promoted string
			switch compID {
			case compChampionsLeague:
				promoted = tm.bestRebalanceCandidate(book, compEuropaLeague)
				if promoted != "" {
					book.promote(promoted, compID, sourceRebalance)
					continue
				}
			case compEuropaLeague:
				promoted = tm.bestRebalanceCandidate(book, compConferenceLeague)
				if promoted != "" {
					book.promote(promoted, compID, sourceRebalance)
					continue
				}
			}
			promoted = ""
			for _, id := range reserves {
				if book.assigned[id] == "" && tm.Clubs[id] != nil {
					promoted = id
					break
				}
			}
			if promoted == "" {
				break
			}
			book.add(compID, promoted, sourceRebalance)
		}
	}
}

func (tm *TournamentManager) worstRebalanceCandidate(book *qualBook, compID string) string {
	var worst string
	var worstCoeff, worstRating int
	for _, id := range book.clubsIn(compID) {
		if isTitleholderSource(book.source[compID][id]) {
			continue
		}
		club := tm.Clubs[id]
		coeff, rating := 0, 0
		if club != nil {
			coeff, rating = club.Coefficient, club.OverallTeamRating
		}
		if worst == "" || coeff < worstCoeff || (coeff == worstCoeff && rating < worstRating) || (coeff == worstCoeff && rating == worstRating && id > worst) {
			worst = id
			worstCoeff, worstRating = coeff, rating
		}
	}
	return worst
}

func (tm *TournamentManager) bestRebalanceCandidate(book *qualBook, compID string) string {
	var best string
	var bestCoeff, bestRating int
	for _, id := range book.clubsIn(compID) {
		if isTitleholderSource(book.source[compID][id]) {
			continue
		}
		club := tm.Clubs[id]
		coeff, rating := 0, 0
		if club != nil {
			coeff, rating = club.Coefficient, club.OverallTeamRating
		}
		if best == "" || coeff > bestCoeff || (coeff == bestCoeff && rating > bestRating) || (coeff == bestCoeff && rating == bestRating && id < best) {
			best = id
			bestCoeff, bestRating = coeff, rating
		}
	}
	return best
}
