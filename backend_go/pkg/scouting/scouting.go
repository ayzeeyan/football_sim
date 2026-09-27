// Package scouting generates deterministic recruitment intelligence from
// already-computed world state: per-player scouting reports (consistency,
// potential ceiling, form, value trend, risk) and per-club shortlists.
// Everything here is a pure read — no RNG, no clock, no mutation. Consumers
// must sort before any selection so candidate order never depends on map
// iteration.
package scouting

import (
	"sort"

	"football_sim/pkg/models"
)

// Report is one player's scouting assessment. All scores are 0-100 ints so
// the payload stays stable across saves; the labels are derived, never
// stored.
type Report struct {
	PlayerID       string `json:"player_id"`
	FullName       string `json:"full_name"`
	Position       string `json:"position"`
	Category       string `json:"category"`
	Age            int    `json:"age"`
	OVR            int    `json:"ovr"`
	ClubID         string `json:"club_id"`
	ClubName       string `json:"club_name"`
	ClubShort      string `json:"club_short"`
	League         string `json:"league"`
	Region         string `json:"region"`
	OnLoan         bool   `json:"on_loan"`
	MarketValueEUR int64  `json:"market_value_eur"`

	// PotentialCeiling is the tracked growth potential when the growth
	// engine has one (prodigies), else a deterministic age/OVR projection.
	PotentialCeiling int      `json:"potential_ceiling"`
	CeilingDelta     int      `json:"ceiling_delta"`
	Consistency      int      `json:"consistency"`
	Form             string   `json:"form"`
	ValueTrend       string   `json:"value_trend"`
	Risk             int      `json:"risk"`
	RiskFactors      []string `json:"risk_factors,omitempty"`
	Verdict          string   `json:"verdict"`
	ScoutScore       int      `json:"scout_score"`
}

// RegionForClub maps a club's country to a broad scouting region. The world
// is European-only, so this is a country-level assignment today and a seam
// for wider worlds later.
func RegionForClub(club *models.Club) string {
	if club == nil || club.Country == "" {
		return "Unknown"
	}
	return club.Country
}

// ProjectedCeiling estimates a player's OVR ceiling from age and current
// level when no tracked potential exists. Young players keep headroom;
// veterans are assumed to be at or near their peak.
func ProjectedCeiling(age, ovr int) int {
	switch {
	case age <= 19:
		return clampOVR(ovr + 12)
	case age <= 22:
		return clampOVR(ovr + 8)
	case age <= 25:
		return clampOVR(ovr + 4)
	case age <= 28:
		return clampOVR(ovr + 1)
	default:
		return clampOVR(ovr)
	}
}

func clampOVR(v int) int {
	if v < 1 {
		return 1
	}
	if v > 99 {
		return 99
	}
	return v
}

func clampScore(v float64) int {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return int(v)
}

// consistencyFromRatings converts the spread of recent match ratings into a
// 0-100 consistency score. Tighter rating bands mean a more predictable
// performer. Players without ratings get a neutral, appearance-scaled score.
func consistencyFromRatings(p *models.Player) int {
	if len(p.RecentRatings) == 0 {
		if p.Appearances >= 10 {
			return 65
		}
		return 50
	}
	mean := 0.0
	for _, r := range p.RecentRatings {
		mean += r
	}
	mean /= float64(len(p.RecentRatings))
	variance := 0.0
	for _, r := range p.RecentRatings {
		d := r - mean
		variance += d * d
	}
	variance /= float64(len(p.RecentRatings))
	// stddev of ~0.4 ratings maps to ~90 consistency; ~2.0 maps to ~20.
	return clampScore(100 - (variance * 20))
}

// formLabel summarises recent ratings on the match-report 1-10 scale.
func formLabel(p *models.Player) string {
	if len(p.RecentRatings) == 0 {
		return "unproven"
	}
	mean := 0.0
	for _, r := range p.RecentRatings {
		mean += r
	}
	mean /= float64(len(p.RecentRatings))
	switch {
	case mean >= 7.5:
		return "hot"
	case mean >= 6.5:
		return "warm"
	case mean >= 5.5:
		return "steady"
	default:
		return "cold"
	}
}

// valueTrendFor combines age and form into a market-direction read.
func valueTrendFor(p *models.Player, form string) string {
	switch {
	case p.Age <= 23 && (form == "hot" || form == "warm"):
		return "rising"
	case p.Age >= 30:
		return "falling"
	case p.Age >= 29 && form == "cold":
		return "falling"
	case form == "hot":
		return "rising"
	case form == "cold":
		return "falling"
	default:
		return "stable"
	}
}

// riskFor scores transfer risk 0-100 (higher is riskier) and lists the
// concrete factors behind the number.
func riskFor(p *models.Player, consistency int) (int, []string) {
	risk := 20.0
	var factors []string
	if p.Age <= 18 {
		risk += 15
		factors = append(factors, "very young — development path uncertain")
	}
	if p.InjuredMatches > 0 || p.Injury != "" {
		risk += 15
		factors = append(factors, "current or recent injury")
	}
	if p.ContractYears <= 1 {
		risk += 10
		factors = append(factors, "contract expiring within a year")
	}
	if p.TransferRequested {
		risk += 10
		factors = append(factors, "has requested a transfer")
	}
	if p.Loyalty < 40 {
		risk += 10
		factors = append(factors, "low loyalty profile")
	}
	if consistency < 40 {
		risk += 10
		factors = append(factors, "volatile match ratings")
	}
	if p.Appearances < 5 {
		risk += 5
		factors = append(factors, "minimal senior exposure")
	}
	return clampScore(risk), factors
}

// verdictFor turns the numbers into a scout's one-line recommendation.
func verdictFor(r Report) string {
	switch {
	case r.PotentialCeiling >= 90 && r.Age <= 21:
		return "generational ceiling — track weekly"
	case r.CeilingDelta >= 8 && r.Age <= 23:
		return "high-growth target — bid early"
	case r.OVR >= 85:
		return "first-team ready at the top level"
	case r.Risk >= 70:
		return "talented but high-risk profile"
	case r.ValueTrend == "falling" && r.Age >= 30:
		return "experienced depth — short contract only"
	default:
		return "useful squad option"
	}
}

// GenerateReport assesses one player. potential is the tracked growth
// potential when the growth engine has one (ok=true), otherwise the caller
// passes ok=false and the ceiling is projected from age and level.
func GenerateReport(p *models.Player, club *models.Club, potential int, tracked bool) Report {
	ceiling := potential
	if !tracked || ceiling <= 0 {
		ceiling = ProjectedCeiling(p.Age, p.OVR)
	}
	consistency := consistencyFromRatings(p)
	form := formLabel(p)
	risk, factors := riskFor(p, consistency)
	r := Report{
		PlayerID: p.PlayerID, FullName: p.FullName, Position: p.Position, Category: p.Category,
		Age: p.Age, OVR: p.OVR,
		ClubID: p.ClubID, ClubName: club.ClubName, ClubShort: club.ShortName,
		League: club.League, Region: RegionForClub(club), OnLoan: p.OnLoan,
		MarketValueEUR:   p.MarketValueEUR,
		PotentialCeiling: ceiling, CeilingDelta: ceiling - p.OVR,
		Consistency: consistency, Form: form,
		ValueTrend: valueTrendFor(p, form),
		Risk:       risk, RiskFactors: factors,
	}
	r.Verdict = verdictFor(r)
	r.ScoutScore = scoutScore(r)
	return r
}

// scoutScore is the composite used to order shortlists: ceiling headroom and
// current level first, then consistency, minus risk and a value penalty so
// a shortlist is not just the twelve most expensive players.
func scoutScore(r Report) int {
	score := float64(r.OVR) + float64(r.CeilingDelta)*1.5 + float64(r.Consistency)*0.2 - float64(r.Risk)*0.3
	if r.Form == "hot" {
		score += 3
	} else if r.Form == "cold" {
		score -= 2
	}
	if r.ValueTrend == "rising" {
		score += 2
	} else if r.ValueTrend == "falling" {
		score -= 2
	}
	if r.MarketValueEUR > 150_000_000 {
		score -= 6
	}
	return int(score)
}

// Candidate is a player plus the club it belongs to, gathered by the caller
// from world state in deterministic order.
type Candidate struct {
	Player    *models.Player
	Club      *models.Club
	Potential int
	Tracked   bool
}

// Shortlist ranks candidates for one club and returns at most limit reports.
// Candidates are sorted by scout score descending, then player ID, so the
// output never depends on input order. Players already at the buying club
// are excluded.
func Shortlist(clubID string, candidates []Candidate, limit int) []Report {
	if limit <= 0 {
		limit = 12
	}
	reports := make([]Report, 0, len(candidates))
	for _, c := range candidates {
		if c.Player == nil || c.Club == nil || c.Player.ClubID == clubID {
			continue
		}
		reports = append(reports, GenerateReport(c.Player, c.Club, c.Potential, c.Tracked))
	}
	sort.SliceStable(reports, func(i, j int) bool {
		if reports[i].ScoutScore != reports[j].ScoutScore {
			return reports[i].ScoutScore > reports[j].ScoutScore
		}
		return reports[i].PlayerID < reports[j].PlayerID
	})
	if len(reports) > limit {
		reports = reports[:limit]
	}
	return reports
}
