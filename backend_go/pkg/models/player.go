package models

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Player represents a football player with season stats, dynamic tracking,
// career ledger, education progression, and match availability.
type Player struct {
	PlayerID          string `json:"player_id"`
	FullName          string `json:"full_name"`
	Position          string `json:"position"`
	Category          string `json:"category"`
	OVR               int    `json:"ovr"`
	Age               int    `json:"age"`
	MarketValueEUR    int64  `json:"market_value_eur"`
	WageEUR           int64  `json:"wage_eur"`
	ContractYears     int    `json:"contract_years"`
	Loyalty           int    `json:"loyalty"`
	UniverseWonderkid bool   `json:"universe_wonderkid"`
	PlayerSource      string `json:"player_source"`
	ClubID            string `json:"club_id"`
	OriginalClubID    string `json:"original_club_id"`

	// Current Season Stats
	Goals       int    `json:"goals"`
	Assists     int    `json:"assists"`
	Appearances int    `json:"appearances"`
	Season      string `json:"season"`

	// Career Records
	CareerGoals   int    `json:"career_goals"`
	CareerAssists int    `json:"career_assists"`
	CareerApps    int    `json:"career_apps"`
	BestGoals     int    `json:"best_goals"`
	BestAssists   int    `json:"best_assists"`
	BestSeason    string `json:"best_season"`
	OwnGoals      int    `json:"own_goals"`

	// Availability
	SuspendedMatches int    `json:"suspended_matches"`
	InjuredMatches   int    `json:"injured_matches"`
	Injury           string `json:"injury"`

	// Wonderkid & Academics
	Education         string `json:"education"`
	EducationPending  bool   `json:"education_pending"`
	SchoolWant        string `json:"school_want"`
	SchoolTrack       string `json:"school_track"`
	PositionPath      string `json:"position_path"`
	SecondaryPosition string `json:"secondary_position"`
	PositionXP        int    `json:"position_xp"`
	Personality       string `json:"personality"`
	MentorID          string `json:"mentor_id,omitempty"`
	MentorName        string `json:"mentor_name,omitempty"`
	MentorOVR         int    `json:"mentor_ovr,omitempty"`
	Composure         int    `json:"composure"`
	ConsecutiveStarts int    `json:"consecutive_starts"`
}

// playerAlias is used for JSON deserialization to avoid recursion.
type playerAlias Player

// rawSeasonStats captures nested season stats in dataset.json.
type rawSeasonStats struct {
	Goals       int    `json:"goals"`
	Assists     int    `json:"assists"`
	Appearances int    `json:"appearances"`
	Season      string `json:"season"`
}

// rawPlayerData handles alternative key names from raw dataset.json.
type rawPlayerData struct {
	playerAlias
	EstimatedMarketValueEUR int64           `json:"estimated_market_value_eur"`
	SeasonStats             *rawSeasonStats `json:"season_stats"`
}

// UnmarshalJSON implements custom JSON deserialization supporting dataset.json anomalies.
func (p *Player) UnmarshalJSON(data []byte) error {
	var raw rawPlayerData
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	*p = Player(raw.playerAlias)

	// Market Value fallback
	if p.MarketValueEUR == 0 && raw.EstimatedMarketValueEUR > 0 {
		p.MarketValueEUR = raw.EstimatedMarketValueEUR
	}

	// Nested Season Stats fallback
	if raw.SeasonStats != nil {
		if p.Goals == 0 && raw.SeasonStats.Goals > 0 {
			p.Goals = raw.SeasonStats.Goals
		}
		if p.Assists == 0 && raw.SeasonStats.Assists > 0 {
			p.Assists = raw.SeasonStats.Assists
		}
		if p.Appearances == 0 && raw.SeasonStats.Appearances > 0 {
			p.Appearances = raw.SeasonStats.Appearances
		}
		if p.Season == "" && raw.SeasonStats.Season != "" {
			p.Season = raw.SeasonStats.Season
		}
	}

	// Ensure Category is derived
	if p.Category == "" {
		p.Category = GetPositionCategory(p.Position)
	}

	// Default wage if missing
	if p.WageEUR == 0 && p.OVR > 0 {
		p.WageEUR = WageForOVR(p.OVR)
	}

	// Default contract years
	if p.ContractYears == 0 {
		p.ContractYears = 3
	}

	// Default loyalty
	if p.Loyalty == 0 {
		p.Loyalty = 65
	}
	if p.UniverseWonderkid && p.Loyalty < 70 {
		p.Loyalty = 70
	}

	// Default education
	if p.Education == "" {
		if p.UniverseWonderkid {
			p.Education = p.EducationForAge()
		} else {
			p.Education = "none"
		}
	}

	// Default school want
	if p.SchoolWant == "" {
		if p.UniverseWonderkid {
			p.SchoolWant = SchoolWantFor(p.FullName)
		} else {
			p.SchoolWant = "open"
		}
	}

	// Default school track: football-first kids play through exam weeks,
	// everyone else stays in school unless the club forces them out.
	if p.SchoolTrack == "" {
		if p.UniverseWonderkid {
			p.SchoolTrack = DefaultSchoolTrack(p.SchoolWant)
		}
	}

	// Default personality
	if p.Personality == "" {
		if p.UniverseWonderkid {
			p.Personality = PersonalityFor(p.FullName)
		} else {
			p.Personality = "dedicated_pro"
		}
	}

	// Default composure
	if p.Composure == 0 {
		p.Composure = 75
	}

	// Default original club ID
	if p.OriginalClubID == "" {
		p.OriginalClubID = p.ClubID
	}

	return nil
}

// EffectiveOVR returns the effective overall rating after fatigue drops.
// When consecutive starts >= 3, rating drops by min(4, 2 + (consecutive_starts - 3)), floored at 40.
func (p *Player) EffectiveOVR() int {
	drop := 0
	if p.ConsecutiveStarts >= 3 {
		drop = 2 + (p.ConsecutiveStarts - 3)
		if drop > 4 {
			drop = 4
		}
	}
	eff := p.OVR - drop
	if eff < 40 {
		return 40
	}
	return eff
}

// PersonalityInfo returns the PersonalityArchetype details for this player.
func (p *Player) PersonalityInfo() PersonalityArchetype {
	return ArchetypeForKey(p.Personality)
}

// PersonalityTitle returns the archetype title.
func (p *Player) PersonalityTitle() string {
	return p.PersonalityInfo().Title
}

// PersonalityBadge returns the archetype badge label.
func (p *Player) PersonalityBadge() string {
	return p.PersonalityInfo().Badge
}

// HasMentor returns true if the player has an active mentor.
func (p *Player) HasMentor() bool {
	return p.MentorID != "" && p.MentorName != ""
}

// EducationForAge determines standard education status based on age.
func (p *Player) EducationForAge() string {
	if !p.UniverseWonderkid {
		return "none"
	}
	if p.Age <= 15 {
		return "middle_school"
	}
	if p.Age <= 17 {
		return "high_school"
	}
	return "graduated"
}

// SchoolConflict reports whether school obligations conflict with a fixture.
func (p *Player) SchoolConflict(competition string, matchweek int) bool {
	if !p.UniverseWonderkid {
		return false
	}
	edu := p.Education
	if edu == "" {
		edu = p.EducationForAge()
	}
	if edu == "dropout" || edu == "graduated" || edu == "none" {
		return false
	}
	if IsExamWeek(matchweek) {
		// Football-first and club-forced tracks play through exam weeks.
		if p.SchoolTrack == "football_first" || p.SchoolTrack == "club_forced" {
			return false
		}
		return true
	}
	compLower := strings.ToLower(strings.TrimSpace(competition))
	if edu == "middle_school" && (compLower == "ucl" || compLower == "super-cup") {
		return true
	}
	return false
}

// IsUnavailable reports whether this player cannot play due to suspension, injury, or school.
func (p *Player) IsUnavailable(competition string, matchweek int) bool {
	if p.SuspendedMatches > 0 || p.InjuredMatches > 0 {
		return true
	}
	return p.SchoolConflict(competition, matchweek)
}

// AvailabilityNote returns a descriptive badge/status note regarding match availability.
func (p *Player) AvailabilityNote(competition string, matchweek int) string {
	if p.InjuredMatches > 0 {
		kind := strings.TrimSpace(p.Injury)
		if kind == "" {
			kind = "Injured"
		} else {
			kind = strings.ToUpper(kind[:1]) + kind[1:]
		}
		unit := "matches"
		if p.InjuredMatches == 1 {
			unit = "match"
		}
		return fmt.Sprintf("%s · out %d %s", kind, p.InjuredMatches, unit)
	}
	if p.SuspendedMatches > 0 {
		unit := "matches"
		if p.SuspendedMatches == 1 {
			unit = "match"
		}
		return fmt.Sprintf("Suspended · out %d %s", p.SuspendedMatches, unit)
	}
	if p.SchoolConflict(competition, matchweek) {
		if IsExamWeek(matchweek) {
			return "Exams"
		}
		return "School"
	}
	return "Available"
}

// SchoolWantLine returns the human-readable string for this player's school preference.
func (p *Player) SchoolWantLine() string {
	return SchoolWantLabel(p.SchoolWant)
}

// School track values stored on the prodigy.
const (
	SchoolTrackStay          = "stay"
	SchoolTrackFootballFirst = "football_first"
	SchoolTrackClubForced    = "club_forced"
)

// DefaultSchoolTrack maps a school want onto the starting track.
func DefaultSchoolTrack(want string) string {
	if strings.ToLower(strings.TrimSpace(want)) == "football" {
		return SchoolTrackFootballFirst
	}
	return SchoolTrackStay
}

// ValidSchoolTrack reports whether a track value can be stored.
func ValidSchoolTrack(track string) bool {
	switch strings.ToLower(strings.TrimSpace(track)) {
	case SchoolTrackStay, SchoolTrackFootballFirst, SchoolTrackClubForced:
		return true
	}
	return false
}

// SetSchoolTrack stores a validated school track. Returns false when invalid.
func (p *Player) SetSchoolTrack(track string) bool {
	t := strings.ToLower(strings.TrimSpace(track))
	if !ValidSchoolTrack(t) {
		return false
	}
	p.SchoolTrack = t
	return true
}

// SchoolTrackLabel returns the human-readable string for the stored track.
func (p *Player) SchoolTrackLabel() string {
	switch p.SchoolTrack {
	case SchoolTrackFootballFirst:
		return "Football-first · plays exam weeks"
	case SchoolTrackClubForced:
		return "Club-forced · football after breakout"
	case SchoolTrackStay:
		return "Stay-in-school · sits exam weeks"
	default:
		return "Stay-in-school · sits exam weeks"
	}
}

// EducationLabel returns a descriptive education label with want details.
func (p *Player) EducationLabel() string {
	edu := p.Education
	if edu == "" {
		edu = "none"
	}
	want := ""
	if p.UniverseWonderkid {
		want = SchoolWantLabel(p.SchoolWant)
	}

	switch edu {
	case "middle_school":
		base := "Middle school · sits cup nights and exam weeks"
		if p.SchoolTrack == SchoolTrackFootballFirst || p.SchoolTrack == SchoolTrackClubForced {
			base = "Middle school · sits cup nights, plays exam weeks"
		}
		if want != "" {
			return fmt.Sprintf("%s. %s.", base, want)
		}
		return base
	case "high_school":
		if p.SchoolTrack == SchoolTrackFootballFirst || p.SchoolTrack == SchoolTrackClubForced {
			return "High school · plays exam weeks"
		}
		return "High school · sits exam weeks"
	case "dropout":
		return "Left school · full-time football"
	case "graduated":
		return "Finished school"
	default:
		return "—"
	}
}

// DecideEducation computes whether the prodigy continues to high school or drops out.
func (p *Player) DecideEducation(clubPlace int, appearances int) string {
	want := p.SchoolWant
	if want == "" {
		want = SchoolWantFor(p.FullName)
	}

	loyalty := p.Loyalty
	if loyalty == 0 {
		loyalty = 50
	}
	place := clubPlace
	if place == 0 {
		place = 6
	}

	if want == "football" {
		if loyalty >= 92 && place <= 4 {
			return "high_school"
		}
		return "dropout"
	}
	if want == "school" {
		if loyalty < 40 && place >= 10 {
			return "dropout"
		}
		return "high_school"
	}

	score := 0
	if loyalty >= 70 {
		score += 2
	}
	if place <= 6 {
		score += 1
	}
	if appearances < 8 {
		score += 1
	}
	if place >= 10 {
		score -= 2
	}
	if appearances >= 18 {
		score -= 1
	}

	if score >= 1 {
		return "high_school"
	}
	return "dropout"
}

// AdvanceEducation processes the prodigy's academic transition upon birthday / season tick.
// Returns an action key ("stayed", "left", "graduated", or "").
func (p *Player) AdvanceEducation(clubPlace int) string {
	if !p.UniverseWonderkid {
		return ""
	}

	if p.Age >= 16 && (p.EducationPending || p.Education == "middle_school" || p.Education == "") {
		track := p.DecideEducation(clubPlace, p.Appearances)
		p.Education = track
		p.EducationPending = false
		if track == "high_school" {
			return "stayed"
		}
		return "left"
	}

	if p.Age >= 18 && (p.Education == "high_school" || p.Education == "middle_school") {
		p.Education = "graduated"
		p.EducationPending = false
		return "graduated"
	}

	return ""
}

// PositionOptions returns learnable secondary positions.
func (p *Player) PositionOptions() []string {
	return PositionOptions(p.Position)
}

// AllTimeGoals calculates total career goals including the current campaign.
func (p *Player) AllTimeGoals() int {
	return p.CareerGoals + p.Goals
}

// AllTimeAssists calculates total career assists including the current campaign.
func (p *Player) AllTimeAssists() int {
	return p.CareerAssists + p.Assists
}

// AllTimeApps calculates total career appearances including the current campaign.
func (p *Player) AllTimeApps() int {
	return p.CareerApps + p.Appearances
}

// LiveBest evaluates best season performance including current campaign.
func (p *Player) LiveBest(seasonName string) (int, int, string) {
	if p.Goals > p.BestGoals || (p.BestGoals <= 0 && p.Goals > 0) {
		s := seasonName
		if s == "" {
			s = p.Season
		}
		return p.Goals, p.Assists, s
	}
	return p.BestGoals, p.BestAssists, p.BestSeason
}

// FormattedWage returns weekly wage as a formatted string.
func (p *Player) FormattedWage() string {
	return FormatWage(p.WageEUR)
}

// FormattedValue returns market value as a formatted currency string.
func (p *Player) FormattedValue() string {
	return FormatCurrency(p.MarketValueEUR)
}

// RecordAppearance increments current appearances.
func (p *Player) RecordAppearance() {
	p.Appearances++
}

// RecordGoal increments current goals.
func (p *Player) RecordGoal() {
	p.Goals++
}

// RecordAssist increments current assists.
func (p *Player) RecordAssist() {
	p.Assists++
}
