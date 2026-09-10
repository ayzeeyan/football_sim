package datamanager

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"football_sim/pkg/growth"
	"football_sim/pkg/models"
)

// DatasetMetadata contains top-level dataset configuration and counts.
type DatasetMetadata struct {
	Dataset           string                   `json:"dataset"`
	GeneratedAt       string                   `json:"generated_at"`
	Season            string                   `json:"season"`
	Notes             []string                 `json:"notes"`
	LeagueCounts      map[string]int           `json:"league_counts"`
	Totals            map[string]int           `json:"totals"`
	WonderkidUniverse []map[string]interface{} `json:"wonderkid_universe,omitempty"`
}

// rawDatasetFile maps the schema of dataset.json.
type rawDatasetFile struct {
	Dataset           string                   `json:"dataset"`
	GeneratedAt       string                   `json:"generated_at"`
	Season            string                   `json:"season"`
	Notes             []string                 `json:"notes"`
	LeagueCounts      map[string]int           `json:"league_counts"`
	Totals            map[string]int           `json:"totals"`
	WonderkidUniverse []map[string]interface{} `json:"wonderkid_universe"`
	Clubs             []*models.Club           `json:"clubs"`
}

// DataManager coordinates dataset ingestion, squad deduplication,
// elite wonderkid initialization, and academy youth intakes.
type DataManager struct {
	JSONPath     string
	GrowthEngine *growth.GrowthEngine
	Clubs        map[string]*models.Club
	ClubsList    []*models.Club
	Leagues      map[string][]*models.Club
	Wonderkids   []*models.Player
	Metadata     DatasetMetadata
	ProdigyHomes map[string]string
	rng          *rand.Rand
}

// NewDataManager initializes a DataManager instance.
// If the dataset file exists at jsonPath, it loads the dataset,
// executes deduplication, and sets up elite wonderkids automatically.
func NewDataManager(jsonPath string, ge *growth.GrowthEngine) *DataManager {
	if ge == nil {
		ge = growth.NewGrowthEngine(0)
	}

	dm := &DataManager{
		JSONPath:     jsonPath,
		GrowthEngine: ge,
		Clubs:        make(map[string]*models.Club),
		ClubsList:    make([]*models.Club, 0),
		Leagues:      make(map[string][]*models.Club),
		Wonderkids:   make([]*models.Player, 0),
		ProdigyHomes: DefaultProdigyHomes(),
		rng:          rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	if jsonPath != "" {
		resolvedPath := resolveDatasetPath(jsonPath)
		if _, err := os.Stat(resolvedPath); err == nil {
			if err := dm.LoadDataset(); err == nil {
				dm.DedupePlayers()
				_ = dm.InitializeEliteProdigies()
			}
		}
	}

	return dm
}

// SetRNG sets a custom pseudo-random number generator for deterministic testing.
func (dm *DataManager) SetRNG(r *rand.Rand) {
	if r != nil {
		dm.rng = r
	}
}

// SetSeed re-seeds the internal random number generator.
func (dm *DataManager) SetSeed(seed int64) {
	dm.rng = rand.New(rand.NewSource(seed))
}

// resolveDatasetPath attempts to find the dataset file, checking relative paths
// and parent directory fallbacks when running tests from subpackages.
func resolveDatasetPath(path string) string {
	if _, err := os.Stat(path); err == nil {
		return path
	}
	candidates := []string{
		filepath.Join("..", path),
		filepath.Join("..", "..", path),
		filepath.Join("..", "..", "..", path),
	}
	for _, cand := range candidates {
		if _, err := os.Stat(cand); err == nil {
			return cand
		}
	}
	return path
}

// LoadDataset reads dataset.json into Go memory losslessly and resets player season ledgers.
func (dm *DataManager) LoadDataset() error {
	resolvedPath := resolveDatasetPath(dm.JSONPath)
	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return fmt.Errorf("missing '%s'. Please ensure dataset.json is in root directory: %w", dm.JSONPath, err)
	}

	var raw rawDatasetFile
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("failed to parse dataset json: %w", err)
	}

	season := raw.Season
	if season == "" {
		season = "2026-27"
	}

	dm.Metadata = DatasetMetadata{
		Dataset:           raw.Dataset,
		GeneratedAt:       raw.GeneratedAt,
		Season:            season,
		Notes:             raw.Notes,
		LeagueCounts:      raw.LeagueCounts,
		Totals:            raw.Totals,
		WonderkidUniverse: raw.WonderkidUniverse,
	}

	dm.Clubs = make(map[string]*models.Club, len(raw.Clubs))
	dm.ClubsList = make([]*models.Club, 0, len(raw.Clubs))
	dm.Leagues = make(map[string][]*models.Club)

	for _, club := range raw.Clubs {
		dm.Clubs[club.ClubID] = club
		dm.ClubsList = append(dm.ClubsList, club)

		leagueName := club.League
		dm.Leagues[leagueName] = append(dm.Leagues[leagueName], club)
	}

	// First kickoff of the career save: reset season match and career statistics to 0.
	for _, club := range dm.ClubsList {
		for _, p := range club.Squad {
			p.CareerGoals = 0
			p.CareerAssists = 0
			p.CareerApps = 0
			p.BestGoals = 0
			p.BestAssists = 0
			p.BestSeason = ""
			p.OwnGoals = 0
			p.SuspendedMatches = 0
			p.InjuredMatches = 0
			p.Injury = ""
			p.Goals = 0
			p.Assists = 0
			p.Appearances = 0
		}
	}

	return nil
}

type playerCopy struct {
	club   *models.Club
	player *models.Player
}

// DedupePlayers enforces the invariant of exactly 0 duplicate players across and within clubs.
// Uses PREFERRED_HOMES, elite club priority, and highest (OVR, appearances) fallback.
// Merges match/career stats via max() and synchronizes SquadSize = len(Squad).
// Returns the number of removed duplicates.
func (dm *DataManager) DedupePlayers() int {
	prodigyConfigs := make(map[string]EliteProdigyConfig, len(EliteProdigyConfigs))
	eliteClubs := make(map[string]bool, len(EliteProdigyConfigs))
	for _, cfg := range EliteProdigyConfigs {
		prodigyConfigs[strings.ToLower(strings.TrimSpace(cfg.FullName))] = cfg
		eliteClubs[cfg.ClubID] = true
	}

	byName := make(map[string][]playerCopy)
	for _, club := range dm.ClubsList {
		for _, p := range club.Squad {
			normName := strings.ToLower(strings.TrimSpace(p.FullName))
			byName[normName] = append(byName[normName], playerCopy{club: club, player: p})
		}
	}

	removed := 0

	for name, copies := range byName {
		if len(copies) < 2 {
			continue
		}

		var canonicalCP playerCopy
		wkCfg, isWonderkid := prodigyConfigs[name]

		if isWonderkid {
			var wkCopies []playerCopy
			for _, cp := range copies {
				if strings.HasPrefix(cp.player.PlayerID, "WK_") {
					wkCopies = append(wkCopies, cp)
				}
			}
			if len(wkCopies) == 0 {
				for _, cp := range copies {
					if cp.player.UniverseWonderkid {
						wkCopies = append(wkCopies, cp)
					}
				}
			}

			keepCID := wkCfg.ClubID
			if dm.ProdigyHomes != nil {
				if homeCID, ok := dm.ProdigyHomes[wkCfg.FullName]; ok && homeCID != "" {
					keepCID = homeCID
				}
			}

			searchPool := wkCopies
			if len(searchPool) == 0 {
				searchPool = copies
			}

			var matchingHome *playerCopy
			for _, cp := range searchPool {
				if cp.club.ClubID == keepCID {
					matchingHome = &cp
					break
				}
			}

			if matchingHome != nil {
				canonicalCP = *matchingHome
			} else if len(wkCopies) > 0 {
				canonicalCP = wkCopies[0]
			} else {
				canonicalCP = copies[0]
			}
		} else {
			var keepClubID string
			wantClubID, hasWant := PreferredHomes[name]

			wantInCopies := false
			if hasWant {
				for _, cp := range copies {
					if cp.club.ClubID == wantClubID {
						wantInCopies = true
						break
					}
				}
			}

			if hasWant && wantInCopies {
				keepClubID = wantClubID
			} else {
				eliteHitsMap := make(map[string]bool)
				for _, cp := range copies {
					if eliteClubs[cp.club.ClubID] {
						eliteHitsMap[cp.club.ClubID] = true
					}
				}

				if len(eliteHitsMap) == 1 {
					for cid := range eliteHitsMap {
						keepClubID = cid
					}
				} else {
					bestCopy := copies[0]
					for _, cp := range copies[1:] {
						if cp.player.OVR > bestCopy.player.OVR ||
							(cp.player.OVR == bestCopy.player.OVR && cp.player.Appearances > bestCopy.player.Appearances) {
							bestCopy = cp
						}
					}
					keepClubID = bestCopy.club.ClubID
				}
			}

			var bestInKeepClub *playerCopy
			for i := range copies {
				cp := &copies[i]
				if cp.club.ClubID == keepClubID {
					if bestInKeepClub == nil || cp.player.OVR > bestInKeepClub.player.OVR ||
						(cp.player.OVR == bestInKeepClub.player.OVR && cp.player.Appearances > bestInKeepClub.player.Appearances) {
						bestInKeepClub = cp
					}
				}
			}

			// keepClubID always originates from a copy's own club (preferred-home
			// hit, elite hit, or best-copy fallback), so a canonical instance is
			// always found here.
			if bestInKeepClub != nil {
				canonicalCP = *bestInKeepClub
			}
		}

		keepClub := canonicalCP.club
		keepPlayer := canonicalCP.player
		keepPlayer.ClubID = keepClub.ClubID

		if isWonderkid {
			keepPlayer.Age = 14
			keepPlayer.Education = "middle_school"
			keepPlayer.EducationPending = false
			keepPlayer.UniverseWonderkid = true
		}

		// 1. Merge stats from distinct duplicate instances into keepPlayer via max()
		for _, cp := range copies {
			if cp.player != keepPlayer {
				keepPlayer.Appearances = maxInt(keepPlayer.Appearances, cp.player.Appearances)
				keepPlayer.Goals = maxInt(keepPlayer.Goals, cp.player.Goals)
				keepPlayer.Assists = maxInt(keepPlayer.Assists, cp.player.Assists)
				keepPlayer.CareerGoals = maxInt(keepPlayer.CareerGoals, cp.player.CareerGoals)
				keepPlayer.CareerAssists = maxInt(keepPlayer.CareerAssists, cp.player.CareerAssists)
				keepPlayer.CareerApps = maxInt(keepPlayer.CareerApps, cp.player.CareerApps)
				keepPlayer.BestGoals = maxInt(keepPlayer.BestGoals, cp.player.BestGoals)
				keepPlayer.BestAssists = maxInt(keepPlayer.BestAssists, cp.player.BestAssists)
			}
		}

		// 2. Cleanse all squads: retain strictly the first occurrence in keepClub, purge all other occurrences
		canonicalRetained := false
		for _, club := range dm.ClubsList {
			newSquad := make([]*models.Player, 0, len(club.Squad))
			modified := false
			for _, p := range club.Squad {
				isMatch := (p == keepPlayer) || strings.EqualFold(strings.TrimSpace(p.FullName), name)
				if isMatch {
					if club.ClubID == keepClub.ClubID && !canonicalRetained {
						// Keep exactly one canonical instance in keepClub
						newSquad = append(newSquad, keepPlayer)
						canonicalRetained = true
					} else {
						// Purge duplicate instance: merge stats and count removal
						if p != keepPlayer {
							keepPlayer.Appearances = maxInt(keepPlayer.Appearances, p.Appearances)
							keepPlayer.Goals = maxInt(keepPlayer.Goals, p.Goals)
							keepPlayer.Assists = maxInt(keepPlayer.Assists, p.Assists)
							keepPlayer.CareerGoals = maxInt(keepPlayer.CareerGoals, p.CareerGoals)
							keepPlayer.CareerAssists = maxInt(keepPlayer.CareerAssists, p.CareerAssists)
							keepPlayer.CareerApps = maxInt(keepPlayer.CareerApps, p.CareerApps)
							keepPlayer.BestGoals = maxInt(keepPlayer.BestGoals, p.BestGoals)
							keepPlayer.BestAssists = maxInt(keepPlayer.BestAssists, p.BestAssists)
						}
						removed++
						modified = true
					}
				} else {
					newSquad = append(newSquad, p)
				}
			}
			if modified {
				club.Squad = newSquad
				club.SquadSize = len(club.Squad)
			}
		}
	}

	return removed
}

// ResetMarketToBaseline recalculates and clamps valuations for all players in all clubs.
func (dm *DataManager) ResetMarketToBaseline() {
	for _, club := range dm.ClubsList {
		for _, p := range club.Squad {
			p.MarketValueEUR = models.BaselineValue(p.OVR, p.Age, p.UniverseWonderkid)
			models.ClampPlayer(p)
		}
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
