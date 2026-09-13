package tournament

import (
	"fmt"
	"hash/fnv"
	"math/rand"
	"sort"

	"football_sim/pkg/models"
)

// CompetitionKind keeps league, domestic knockout, and European competition
// state on one extensible model rather than growing a new bespoke tournament
// engine for every cup.
type CompetitionKind string

const (
	CompetitionLeague   CompetitionKind = "LEAGUE"
	CompetitionDomestic CompetitionKind = "DOMESTIC_CUP"
	CompetitionEuropean CompetitionKind = "EUROPEAN"
)

type CompetitionDefinition struct {
	ID       string
	Name     string
	Country  string
	League   string
	Kind     CompetitionKind
	Prestige int
}

var domesticLeagueDefinitions = []CompetitionDefinition{
	{ID: "premier-league", Name: "Premier League", Country: "England", League: "Premier League", Kind: CompetitionLeague, Prestige: 95},
	{ID: "la-liga", Name: "La Liga", Country: "Spain", League: "La Liga", Kind: CompetitionLeague, Prestige: 93},
	{ID: "bundesliga", Name: "Bundesliga", Country: "Germany", League: "Bundesliga", Kind: CompetitionLeague, Prestige: 91},
	{ID: "serie-a", Name: "Serie A", Country: "Italy", League: "Serie A", Kind: CompetitionLeague, Prestige: 91},
	{ID: "ligue-1", Name: "Ligue 1", Country: "France", League: "Ligue 1", Kind: CompetitionLeague, Prestige: 86},
}

var domesticCupDefinitions = []CompetitionDefinition{
	{ID: "fa-cup", Name: "FA Cup", Country: "England", League: "Premier League", Kind: CompetitionDomestic, Prestige: 78},
	{ID: "efl-cup", Name: "EFL Cup", Country: "England", League: "Premier League", Kind: CompetitionDomestic, Prestige: 70},
	{ID: "copa-del-rey", Name: "Copa del Rey", Country: "Spain", League: "La Liga", Kind: CompetitionDomestic, Prestige: 76},
	{ID: "dfb-pokal", Name: "DFB-Pokal", Country: "Germany", League: "Bundesliga", Kind: CompetitionDomestic, Prestige: 74},
	{ID: "coppa-italia", Name: "Coppa Italia", Country: "Italy", League: "Serie A", Kind: CompetitionDomestic, Prestige: 74},
	{ID: "coupe-de-france", Name: "Coupe de France", Country: "France", League: "Ligue 1", Kind: CompetitionDomestic, Prestige: 72},
}

var europeanDefinitions = []CompetitionDefinition{
	{ID: "champions-league", Name: "UEFA Champions League", Country: "Europe", Kind: CompetitionEuropean, Prestige: 100},
	{ID: "europa-league", Name: "UEFA Europa League", Country: "Europe", Kind: CompetitionEuropean, Prestige: 84},
	{ID: "conference-league", Name: "UEFA Conference League", Country: "Europe", Kind: CompetitionEuropean, Prestige: 68},
}

// KnockoutRound is persisted as a bracket round. Fixture IDs, rather than
// pointers, keep saves small and make fixture lookup unambiguous.
// TieIDs groups fixtures into home-and-away ties for European knockouts:
// one entry per tie, each covering one (final) or two (legs) fixture IDs.
type KnockoutRound struct {
	Stage      string   `json:"stage"`
	FixtureIDs []string `json:"fixture_ids"`
	EntrantIDs []string `json:"entrant_ids"`
	ByeIDs     []string `json:"bye_ids,omitempty"`
	WinnerIDs  []string `json:"winner_ids,omitempty"`
	TieIDs     []string `json:"tie_ids,omitempty"`
}

// Competition is the reusable state shape for every new world competition.
// Domestic league tables deliberately reuse a club's single domestic record;
// European league phases own independent CompetitionRecords.
type Competition struct {
	ID                   string            `json:"id"`
	Name                 string            `json:"name"`
	Country              string            `json:"country"`
	Kind                 CompetitionKind   `json:"kind"`
	Prestige             int               `json:"prestige"`
	ParticipantIDs       []string          `json:"participant_ids"`
	QualificationSources map[string]string `json:"qualification_sources,omitempty"`
	// Pots snapshots the seeded league-phase pot draw (pot index -> club IDs
	// in ClubID order) so draws stay verifiable after later rating drift.
	// Only set for pot-based phases (36-team Champions League).
	Pots                  [][]string                           `json:"pots,omitempty"`
	Records               map[string]*models.CompetitionRecord `json:"records,omitempty"`
	LeaguePhaseFixtureIDs []string                             `json:"league_phase_fixture_ids,omitempty"`
	Rounds                []KnockoutRound                      `json:"rounds,omitempty"`
	PendingByeIDs         []string                             `json:"pending_bye_ids,omitempty"`
	Stage                 string                               `json:"stage"`
	ChampionID            string                               `json:"champion_id,omitempty"`
}

// EuropeanWorld is a persisted, indexed competition universe. Domestic
// league fixtures remain on TournamentManager.Fixtures for compatibility
// with the existing match center; all shared-calendar cup fixtures live here.
type EuropeanWorld struct {
	Version          int                     `json:"version"`
	Seed             int64                   `json:"seed"`
	Competitions     map[string]*Competition `json:"competitions"`
	CompetitionOrder []string                `json:"competition_order"`
	Fixtures         []Fixture               `json:"fixtures"`
}

func isTopFiveLeague(name string) bool {
	return leagueIDForName(name) != ""
}

func leagueIDForName(name string) string {
	for _, def := range domesticLeagueDefinitions {
		if def.League == name {
			return def.ID
		}
	}
	return ""
}

func worldSeedFor(seed int64, key string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(fmt.Sprintf("%d:%s", seed, key)))
	v := int64(h.Sum64() & 0x7fffffffffffffff)
	if v == 0 {
		return 1
	}
	return v
}

func sortedClubIDs(clubs []*models.Club) []string {
	ids := make([]string, 0, len(clubs))
	for _, club := range clubs {
		if club != nil && club.ClubID != "" {
			ids = append(ids, club.ClubID)
		}
	}
	sort.Strings(ids)
	return ids
}

func clubsForIDs(all map[string]*models.Club, ids []string) []*models.Club {
	out := make([]*models.Club, 0, len(ids))
	for _, id := range ids {
		if club := all[id]; club != nil {
			out = append(out, club)
		}
	}
	return out
}

func stageForKnockoutSize(n int) string {
	switch n {
	case 2:
		return "Final"
	case 4:
		return "Semi-final"
	case 8:
		return "Quarter-final"
	case 16:
		return "Round of 16"
	default:
		return fmt.Sprintf("Round of %d", n)
	}
}

// GenerateDoubleRoundRobinFixtures is the top-flight scheduler: every pair
// meets once in each venue. The circle method supports odd leagues through a
// virtual bye and never consumes a caller's match RNG stream.
func GenerateDoubleRoundRobinFixtures(clubs []*models.Club, competition string) ([]Fixture, int) {
	if len(clubs) < 2 {
		return nil, 0
	}
	rotation := append([]*models.Club(nil), clubs...)
	sort.SliceStable(rotation, func(i, j int) bool { return rotation[i].ClubID < rotation[j].ClubID })
	if len(rotation)%2 != 0 {
		rotation = append(rotation, nil)
	}
	slots := len(rotation)
	roundsPerCycle := slots - 1
	base := make([][]Fixture, 0, roundsPerCycle)
	for round := 0; round < roundsPerCycle; round++ {
		row := make([]Fixture, 0, slots/2)
		for i := 0; i < slots/2; i++ {
			a, b := rotation[i], rotation[slots-1-i]
			if a == nil || b == nil {
				continue
			}
			home, away := a, b
			if (round+i)%2 != 0 {
				home, away = away, home
			}
			row = append(row, Fixture{Competition: competition, Stage: "League", HomeID: home.ClubID, AwayID: away.ClubID, Home: home, Away: away, Status: "scheduled", DerbyName: GetDerbyName(home.ClubID, away.ClubID), DerbyHeat: 50})
		}
		base = append(base, row)
		last := rotation[slots-1]
		copy(rotation[2:], rotation[1:slots-1])
		rotation[1] = last
	}
	out := make([]Fixture, 0, len(clubs)*(len(clubs)-1))
	mw := 1
	for cycle := 0; cycle < 2; cycle++ {
		for _, row := range base {
			for _, f := range row {
				home, away := f.Home, f.Away
				if cycle == 1 {
					home, away = away, home
				}
				out = append(out, Fixture{FixtureID: fmt.Sprintf("%s-MW%02d-%s-%s", competition, mw, home.ClubID, away.ClubID), Matchweek: mw, Competition: competition, Stage: "League", HomeID: home.ClubID, AwayID: away.ClubID, Home: home, Away: away, Status: "scheduled", DerbyName: GetDerbyName(home.ClubID, away.ClubID), DerbyHeat: 50})
			}
			mw++
		}
	}
	return out, mw - 1
}

func rankClubsForOpening(clubs []*models.Club) []*models.Club {
	out := append([]*models.Club(nil), clubs...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].OverallTeamRating != out[j].OverallTeamRating {
			return out[i].OverallTeamRating > out[j].OverallTeamRating
		}
		if out[i].Identity.Reputation != out[j].Identity.Reputation {
			return out[i].Identity.Reputation > out[j].Identity.Reputation
		}
		return out[i].ClubID < out[j].ClubID
	})
	return out
}

func shuffledIDs(seed int64, key string, ids []string) []string {
	out := append([]string(nil), ids...)
	r := rand.New(rand.NewSource(worldSeedFor(seed, key)))
	r.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}
