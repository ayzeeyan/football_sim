package datamanager

import (
	"math/rand"
	"strings"
	"time"

	"football_sim/pkg/models"
)

// EliteProdigyConfig defines the exact biometrics and potential for a canonical wonderkid.
type EliteProdigyConfig struct {
	ClubID         string  `json:"club_id"`
	FullName       string  `json:"full_name"`
	Position       string  `json:"position"`
	Age            int     `json:"age"`
	HeightCM       float64 `json:"height_cm"`
	WeightKG       float64 `json:"weight_kg"`
	BaselineOVR    int     `json:"baseline_ovr"`
	Potential      int     `json:"potential"`
	AdultHeightAge int     `json:"adult_height_age"`
}

// EliteProdigyConfigs lists the 12 canonical Under-14 Outfield Franchise Prodigies.
// Notice potentials are strictly clamped in [93, 96] and are never 99.
var EliteProdigyConfigs = []EliteProdigyConfig{
	{
		ClubID:         "LAL-BAR",
		FullName:       "Venjamin Valerio",
		Position:       "ST",
		Age:            14,
		HeightCM:       173.0,
		WeightKG:       61.0,
		BaselineOVR:    78,
		Potential:      96,
		AdultHeightAge: 19,
	},
	{
		ClubID:         "LAL-RMA",
		FullName:       "Maverick Cantalejo",
		Position:       "CAM",
		Age:            14,
		HeightCM:       169.0,
		WeightKG:       57.0,
		BaselineOVR:    77,
		Potential:      95,
		AdultHeightAge: 18,
	},
	{
		ClubID:         "LAL-ATM",
		FullName:       "Yeshua Emmanuel Gocotano",
		Position:       "CF",
		Age:            14,
		HeightCM:       171.0,
		WeightKG:       60.0,
		BaselineOVR:    75,
		Potential:      93,
		AdultHeightAge: 19,
	},
	{
		ClubID:         "EPL-ARS",
		FullName:       "Izyan Levin Bantol",
		Position:       "CAM",
		Age:            14,
		HeightCM:       170.0,
		WeightKG:       58.0,
		BaselineOVR:    76,
		Potential:      95,
		AdultHeightAge: 18,
	},
	{
		ClubID:         "EPL-LIV",
		FullName:       "James Bernard Rizon",
		Position:       "RW",
		Age:            14,
		HeightCM:       178.0,
		WeightKG:       66.0,
		BaselineOVR:    76,
		Potential:      94,
		AdultHeightAge: 20,
	},
	{
		ClubID:         "BUN-BAY",
		FullName:       "Reid Randell Libatan",
		Position:       "LW",
		Age:            14,
		HeightCM:       176.0,
		WeightKG:       64.0,
		BaselineOVR:    76,
		Potential:      95,
		AdultHeightAge: 19,
	},
	{
		ClubID:         "BUN-DOR",
		FullName:       "Ashle Zylle Baguio",
		Position:       "CAM",
		Age:            14,
		HeightCM:       168.0,
		WeightKG:       57.0,
		BaselineOVR:    75,
		Potential:      95,
		AdultHeightAge: 19,
	},
	{
		ClubID:         "SEA-INT",
		FullName:       "Cliergy Jave Lanticse",
		Position:       "RW",
		Age:            14,
		HeightCM:       169.0,
		WeightKG:       58.0,
		BaselineOVR:    75,
		Potential:      94,
		AdultHeightAge: 20,
	},
	{
		ClubID:         "SEA-NAP",
		FullName:       "Ezail Zamora",
		Position:       "ST",
		Age:            14,
		HeightCM:       172.0,
		WeightKG:       61.0,
		BaselineOVR:    77,
		Potential:      96,
		AdultHeightAge: 19,
	},
	{
		ClubID:         "SEA-MIL",
		FullName:       "Earl Josh Hernando",
		Position:       "LW",
		Age:            14,
		HeightCM:       170.0,
		WeightKG:       59.0,
		BaselineOVR:    75,
		Potential:      94,
		AdultHeightAge: 19,
	},
	{
		ClubID:         "FL1-PSG",
		FullName:       "Rich Lorenz Suico",
		Position:       "LW",
		Age:            14,
		HeightCM:       170.0,
		WeightKG:       59.0,
		BaselineOVR:    75,
		Potential:      94,
		AdultHeightAge: 18,
	},
	{
		ClubID:         "EPL-TOT",
		FullName:       "Jhed Anthony Guinita",
		Position:       "CF",
		Age:            14,
		HeightCM:       174.0,
		WeightKG:       62.0,
		BaselineOVR:    75,
		Potential:      94,
		AdultHeightAge: 19,
	},
}

// ELITE_PRODIGY_CONFIGS provides a Python-compatible alias.
var ELITE_PRODIGY_CONFIGS = EliteProdigyConfigs

// PreferredHomes maps names of players with known transfer destinations in 2026-27.
var PreferredHomes = map[string]string{
	"khvicha kvaratskhelia": "FL1-PSG",
	"martín zubimendi":      "EPL-ARS",
	"martin zubimendi":      "EPL-ARS",
	"alexander isak":        "EPL-LIV",
	"florian wirtz":         "BUN-BAY",
	"benjamin šeško":        "EPL-MUN",
	"benjamin sesko":        "EPL-MUN",
	"omar marmoush":         "EPL-MCI",
	"cole palmer":           "EPL-CHE",
	"luke shaw":             "EPL-MUN",
	"robin le normand":      "LAL-ATM",
	"waldemar anton":        "BUN-DOR",
}

// PREFERRED_HOMES provides a Python-compatible alias.
var PREFERRED_HOMES = PreferredHomes

// DefaultProdigyHomes creates the canonical prodigy-to-club mapping.
func DefaultProdigyHomes() map[string]string {
	homes := make(map[string]string, len(EliteProdigyConfigs))
	for _, cfg := range EliteProdigyConfigs {
		homes[cfg.FullName] = cfg.ClubID
	}
	return homes
}

// ShuffleProdigyHomes creates a random one-to-one assignment of prodigies to the 12 elite clubs.
func ShuffleProdigyHomes(r *rand.Rand) map[string]string {
	names := make([]string, len(EliteProdigyConfigs))
	clubs := make([]string, len(EliteProdigyConfigs))
	for i, cfg := range EliteProdigyConfigs {
		names[i] = cfg.FullName
		clubs[i] = cfg.ClubID
	}

	if r == nil {
		r = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	r.Shuffle(len(clubs), func(i, j int) {
		clubs[i], clubs[j] = clubs[j], clubs[i]
	})

	result := make(map[string]string, len(names))
	for i := range names {
		result[names[i]] = clubs[i]
	}
	return result
}

// ProdigyStableID generates the deterministic identifier for a wonderkid.
func ProdigyStableID(fullName string) string {
	return "WK_" + strings.ReplaceAll(fullName, " ", "_")
}

// ProdigyDrawRow represents a display row for the wonderkid club distribution.
type ProdigyDrawRow struct {
	FullName  string `json:"full_name"`
	Position  string `json:"position"`
	Age       int    `json:"age"`
	ClubID    string `json:"club_id"`
	ClubName  string `json:"club_name"`
	ShortName string `json:"short_name"`
}

// InitializeEliteProdigies configures the 12 canonical U-14 outfield franchise wonderkids.
// Relocates Jhed Anthony Guinita from Marseille (FL1-OM) to Tottenham Hotspur (EPL-TOT).
// Enforces age 14, middle school status, category FWD, WK_ ID, exact potential [93, 96],
// registers with GrowthEngine, and recalculates clamped baseline valuations.
func (dm *DataManager) InitializeEliteProdigies() error {
	dm.Wonderkids = make([]*models.Player, 0, len(EliteProdigyConfigs))
	homes := dm.ProdigyHomes
	if len(homes) == 0 {
		homes = DefaultProdigyHomes()
	}

	for _, cfg := range EliteProdigyConfigs {
		cid, ok := homes[cfg.FullName]
		if !ok || cid == "" {
			cid = cfg.ClubID
		}
		club, ok := dm.Clubs[cid]
		if !ok {
			continue
		}

		// Look for player in club squad
		var prodigy *models.Player
		for _, p := range club.Squad {
			if strings.EqualFold(p.FullName, cfg.FullName) {
				prodigy = p
				break
			}
		}

		// If not in target club, check if they exist elsewhere in dataset and relocate them
		if prodigy == nil {
			for _, otherClub := range dm.ClubsList {
				for i, p := range otherClub.Squad {
					if strings.EqualFold(p.FullName, cfg.FullName) {
						// Remove from other club
						otherClub.Squad = append(otherClub.Squad[:i], otherClub.Squad[i+1:]...)
						otherClub.SquadSize = len(otherClub.Squad)

						prodigy = p
						prodigy.ClubID = cid
						club.Squad = append([]*models.Player{prodigy}, club.Squad...)
						club.SquadSize = len(club.Squad)
						break
					}
				}
				if prodigy != nil {
					break
				}
			}
		}

		// If still not found, instantiate directly
		if prodigy == nil {
			prodigy = &models.Player{
				PlayerID:          ProdigyStableID(cfg.FullName),
				FullName:          cfg.FullName,
				Position:          cfg.Position,
				OVR:               cfg.BaselineOVR,
				Age:               cfg.Age,
				MarketValueEUR:    35_000_000,
				UniverseWonderkid: true,
				PlayerSource:      "universe_wonderkid",
				ClubID:            cid,
				ContractYears:     3,
				Loyalty:           70,
				Education:         "middle_school",
			}
			club.Squad = append([]*models.Player{prodigy}, club.Squad...)
			club.SquadSize = len(club.Squad)
		}

		prodigy.PlayerID = ProdigyStableID(cfg.FullName)

		// Clean up any lingering duplicate copies of this wonderkid across all squads
		keptProdigy := false
		for _, c := range dm.ClubsList {
			newSquad := make([]*models.Player, 0, len(c.Squad))
			modified := false
			for _, p := range c.Squad {
				isMatch := (p == prodigy) || strings.EqualFold(strings.TrimSpace(p.FullName), cfg.FullName)
				if isMatch {
					if c.ClubID == cid && !keptProdigy {
						newSquad = append(newSquad, prodigy)
						keptProdigy = true
					} else {
						// Extra duplicate is eliminated; merge stats into prodigy
						if p != prodigy {
							prodigy.Appearances = maxInt(prodigy.Appearances, p.Appearances)
							prodigy.Goals = maxInt(prodigy.Goals, p.Goals)
							prodigy.Assists = maxInt(prodigy.Assists, p.Assists)
							prodigy.CareerGoals = maxInt(prodigy.CareerGoals, p.CareerGoals)
							prodigy.CareerAssists = maxInt(prodigy.CareerAssists, p.CareerAssists)
							prodigy.CareerApps = maxInt(prodigy.CareerApps, p.CareerApps)
							prodigy.BestGoals = maxInt(prodigy.BestGoals, p.BestGoals)
							prodigy.BestAssists = maxInt(prodigy.BestAssists, p.BestAssists)
						}
						modified = true
					}
				} else {
					newSquad = append(newSquad, p)
				}
			}
			if modified {
				c.Squad = newSquad
				c.SquadSize = len(c.Squad)
			}
		}

		// Set exact under-17 biometrics & attributes
		prodigy.Age = cfg.Age
		prodigy.Position = cfg.Position
		prodigy.OVR = cfg.BaselineOVR
		prodigy.UniverseWonderkid = true
		// Every franchise prodigy is an attacker, CAMs included
		prodigy.Category = "FWD"
		prodigy.Education = "middle_school"
		prodigy.EducationPending = false
		prodigy.SchoolWant = models.SchoolWantFor(prodigy.FullName)

		// Register in Growth Engine
		if dm.GrowthEngine != nil {
			_, attrs := dm.GrowthEngine.RegisterProdigy(
				prodigy.PlayerID,
				prodigy.FullName,
				prodigy.Age,
				cfg.HeightCM,
				cfg.WeightKG,
				prodigy.Category,
				cfg.BaselineOVR,
				cfg.Potential,
				cfg.AdultHeightAge,
			)
			prodigy.OVR = dm.GrowthEngine.CalculateOVR(prodigy.PlayerID, prodigy.Category)
			if attrs != nil {
				prodigy.Composure = attrs.Composure
			}

			dm.GrowthEngine.RecordTimelineEntry(
				prodigy.PlayerID,
				"2026-27",
				prodigy.Age,
				prodigy.OVR,
				cfg.HeightCM,
				cfg.WeightKG,
				prodigy.Goals,
				prodigy.Assists,
				prodigy.Appearances,
				club.ShortName,
				prodigy.MentorName,
			)
		}

		dm.Wonderkids = append(dm.Wonderkids, prodigy)
	}

	// Stamp every player's original club home now that squads have settled
	for _, club := range dm.ClubsList {
		for _, p := range club.Squad {
			p.OriginalClubID = club.ClubID
		}
		club.SquadSize = len(club.Squad)
	}

	dm.ResetMarketToBaseline()
	return nil
}

// AdoptU14Prodigies resets all franchise wonderkids back to age 14, middle school status,
// and regenerates their biometric profiles in the GrowthEngine.
// Returns true if any changes were made.
func (dm *DataManager) AdoptU14Prodigies() bool {
	byName := make(map[string]EliteProdigyConfig, len(EliteProdigyConfigs))
	for _, cfg := range EliteProdigyConfigs {
		byName[strings.ToLower(cfg.FullName)] = cfg
	}

	changed := false
	for _, p := range dm.Wonderkids {
		cfg, ok := byName[strings.ToLower(p.FullName)]
		if !ok {
			continue
		}
		expectedID := ProdigyStableID(cfg.FullName)
		if p.Age != 14 || p.Education != "middle_school" || p.PlayerID != expectedID {
			p.Age = 14
			p.Education = "middle_school"
			p.EducationPending = false
			p.PlayerID = expectedID
			if dm.GrowthEngine != nil {
				_, attrs := dm.GrowthEngine.RegisterProdigy(
					p.PlayerID,
					p.FullName,
					14,
					cfg.HeightCM,
					cfg.WeightKG,
					p.Category,
					cfg.BaselineOVR,
					cfg.Potential,
					cfg.AdultHeightAge,
				)
				p.OVR = dm.GrowthEngine.CalculateOVR(p.PlayerID, p.Category)
				if attrs != nil {
					p.Composure = attrs.Composure
				}
			}
			changed = true
		}
	}
	return changed
}

// ApplyProdigyHomes redistributes each franchise wonderkid to an elite club (one per club).
func (dm *DataManager) ApplyProdigyHomes(homes map[string]string) {
	elite := make(map[string]bool, len(EliteProdigyConfigs))
	for _, cfg := range EliteProdigyConfigs {
		elite[cfg.ClubID] = true
	}

	valid := make(map[string]string)
	for k, v := range homes {
		if elite[v] && dm.Clubs[v] != nil {
			valid[k] = v
		}
	}

	uniqueClubs := make(map[string]bool)
	for _, v := range valid {
		uniqueClubs[v] = true
	}
	if len(uniqueClubs) != len(EliteProdigyConfigs) || len(valid) != len(EliteProdigyConfigs) {
		valid = DefaultProdigyHomes()
	}

	byName := make(map[string]*models.Player)
	cfgNames := make(map[string]bool, len(EliteProdigyConfigs))
	for _, cfg := range EliteProdigyConfigs {
		cfgNames[cfg.FullName] = true
	}

	for _, club := range dm.ClubsList {
		newSquad := make([]*models.Player, 0, len(club.Squad))
		for _, p := range club.Squad {
			if cfgNames[p.FullName] || p.UniverseWonderkid {
				byName[p.FullName] = p
			} else {
				newSquad = append(newSquad, p)
			}
		}
		club.Squad = newSquad
	}

	dm.Wonderkids = make([]*models.Player, 0, len(EliteProdigyConfigs))
	for _, cfg := range EliteProdigyConfigs {
		name := cfg.FullName
		cid := valid[name]
		if cid == "" {
			cid = cfg.ClubID
		}
		dest := dm.Clubs[cid]
		if dest == nil {
			continue
		}
		p := byName[name]
		if p == nil {
			continue
		}
		p.ClubID = cid
		p.OriginalClubID = cid
		dest.Squad = append([]*models.Player{p}, dest.Squad...)
		dm.Wonderkids = append(dm.Wonderkids, p)
	}

	for _, club := range dm.ClubsList {
		club.SquadSize = len(club.Squad)
	}

	dm.ProdigyHomes = make(map[string]string, len(EliteProdigyConfigs))
	for _, cfg := range EliteProdigyConfigs {
		if val, ok := valid[cfg.FullName]; ok {
			dm.ProdigyHomes[cfg.FullName] = val
		} else {
			dm.ProdigyHomes[cfg.FullName] = cfg.ClubID
		}
	}
}

// DescribeProdigyDraw returns descriptive details of the prodigy club distribution.
func (dm *DataManager) DescribeProdigyDraw(homes map[string]string) []ProdigyDrawRow {
	mapping := homes
	if len(mapping) == 0 {
		mapping = dm.ProdigyHomes
	}
	if len(mapping) == 0 {
		mapping = DefaultProdigyHomes()
	}

	rows := make([]ProdigyDrawRow, 0, len(EliteProdigyConfigs))
	for _, cfg := range EliteProdigyConfigs {
		cid := mapping[cfg.FullName]
		if cid == "" {
			cid = cfg.ClubID
		}
		club := dm.Clubs[cid]
		clubName := cid
		shortName := cid
		if club != nil {
			clubName = club.ClubName
			shortName = club.ShortName
		}
		rows = append(rows, ProdigyDrawRow{
			FullName:  cfg.FullName,
			Position:  cfg.Position,
			Age:       cfg.Age,
			ClubID:    cid,
			ClubName:  clubName,
			ShortName: shortName,
		})
	}
	return rows
}

// GetEliteClubs returns the 12 European Super League elite clubs.
func (dm *DataManager) GetEliteClubs() []*models.Club {
	result := make([]*models.Club, 0, len(EliteProdigyConfigs))
	for _, cfg := range EliteProdigyConfigs {
		if club, ok := dm.Clubs[cfg.ClubID]; ok {
			result = append(result, club)
		}
	}
	return result
}
