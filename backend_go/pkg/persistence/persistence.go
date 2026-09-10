package persistence

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"football_sim/pkg/growth"
	"football_sim/pkg/models"
	"football_sim/pkg/tournament"
	"football_sim/pkg/transfers"
)

const (
	SaveVersion     = 2
	DefaultSavePath = "saves/career.json"
	clubIndexKey    = "club_index"
)

// ProdigyMap handles legacy player ID to canonical wonderkid ID resolution.
var ProdigyMap = map[string]string{
	"P00016": "WK_Izyan_Levin_Bantol",
	"P00321": "WK_James_Bernard_Rizon",
	"P00541": "WK_Yeshua_Emmanuel_Gocotano",
	"P00576": "WK_Venjamin_Valerio",
	"P00857": "WK_Maverick_Cantalejo",
	"P01135": "WK_Cliergy_Jave_Lanticse",
	"P01241": "WK_Earl_Josh_Hernando",
	"P01296": "WK_Ezail_Zamora",
	"P01486": "WK_Reid_Randell_Libatan",
	"P01555": "WK_Ashle_Zylle_Baguio",
	"P02099": "WK_Jhed_Anthony_Guinita",
	"P02187": "WK_Rich_Lorenz_Suico",
}

// GrowthSnapshot captures player progression, attributes, biometrics, and milestones.
type GrowthSnapshot struct {
	TrainingEnergy    int                                    `json:"training_energy"`
	MaxTrainingEnergy int                                    `json:"max_training_energy"`
	Biometrics        map[string]*growth.BiometricProfile    `json:"biometrics"`
	Attributes        map[string]*growth.TechnicalAttributes `json:"attributes"`
	Milestones        []growth.GrowthMilestone               `json:"milestones"`
	Timeline          map[string][]growth.TimelineEntry      `json:"timeline"`
}

// TransfersSnapshot captures market activity, negotiations, completed deals,
// budgets, and the active-window guards required for exact restart continuity.
type TransfersSnapshot struct {
	CurrentDay            int                              `json:"current_day"`
	CurrentMatchweek      int                              `json:"current_matchweek"`
	CurrentWeek           int                              `json:"current_week,omitempty"`
	IsOffSeason           bool                             `json:"is_off_season,omitempty"`
	TransferredThisWindow map[string]bool                  `json:"transferred_this_window,omitempty"`
	Feed                  []transfers.TransferFeedItem     `json:"feed"`
	Completed             []transfers.CompletedTransfer    `json:"completed"`
	AllTime               []transfers.CompletedTransfer    `json:"all_time"`
	ActiveNegotiations    []*transfers.TransferNegotiation `json:"active_negotiations,omitempty"`
	ManagerBudgets        map[string]int64                 `json:"manager_budgets,omitempty"`
}

// CareerSnapshot contains the full serialized state of the football universe across seasons.
type CareerSnapshot struct {
	Version               int                                  `json:"version"`
	SeasonName            string                               `json:"season_name"`
	CurrentMatchweek      int                                  `json:"current_matchweek"`
	MaxMatchweeks         int                                  `json:"max_matchweeks"`
	SeasonPhase           string                               `json:"season_phase"`
	RecentResults         []map[string]interface{}             `json:"recent_results,omitempty"`
	GrowthNotifications   []map[string]interface{}             `json:"growth_notifications,omitempty"`
	PlayerOfTheWeek       interface{}                          `json:"player_of_the_week,omitempty"`
	MonthlyAwards         []map[string]interface{}             `json:"monthly_awards,omitempty"`
	SeasonHistory         []map[string]interface{}             `json:"season_history,omitempty"`
	Inbox                 []tournament.InboxItem               `json:"inbox"`
	DerbyHeat             map[string]int                       `json:"derby_heat"`
	WonderkidMilestones   map[string]map[string]bool           `json:"wonderkid_milestones"`
	ManagerConsecutiveHot map[string]int                       `json:"manager_consecutive_hot,omitempty"`
	Clubs                 map[string]*models.Club              `json:"clubs"`
	Fixtures              []tournament.Fixture                 `json:"fixtures"`
	UCLFixtures           []tournament.Fixture                 `json:"ucl_fixtures,omitempty"`
	SuperCupFixtures      []tournament.Fixture                 `json:"super_cup_fixtures,omitempty"`
	UCLStage              string                               `json:"ucl_stage,omitempty"`
	UCLChampionID         string                               `json:"ucl_champion_id,omitempty"`
	SuperCupStage         string                               `json:"super_cup_stage,omitempty"`
	SuperCupChampionID    string                               `json:"super_cup_champion_id,omitempty"`
	RecentTicker          []string                             `json:"recent_results_ticker,omitempty"`
	Growth                GrowthSnapshot                       `json:"growth"`
	Transfers             TransfersSnapshot                    `json:"transfers"`
	ProdigyHomes          map[string]string                    `json:"prodigy_homes,omitempty"`
	ClubSeasonHistory     map[string][]map[string]interface{}  `json:"club_season_history,omitempty"`
	MatchweekWeather      map[int]string                       `json:"matchweek_weather,omitempty"`
	GrowthNotes           []string                             `json:"growth_notes,omitempty"`
	InboxSeq              int                                  `json:"inbox_seq,omitempty"`
	UCLGroupA             []string                             `json:"ucl_group_a,omitempty"`
	UCLGroupB             []string                             `json:"ucl_group_b,omitempty"`
	UCLRecords            map[string]*models.CompetitionRecord `json:"ucl_records,omitempty"`
	UCLQuarterFinals      map[string]tournament.CupTie         `json:"ucl_quarter_finals,omitempty"`
	UCLSemiFinals         map[string]tournament.CupTie         `json:"ucl_semi_finals,omitempty"`
	UCLFinal              tournament.CupTie                    `json:"ucl_final,omitempty"`
	SuperCupByes          []string                             `json:"super_cup_byes,omitempty"`
	SuperCupPlayIn        map[string]tournament.CupTie         `json:"super_cup_play_in,omitempty"`
	SuperCupQuarterFinals map[string]tournament.CupTie         `json:"super_cup_quarter_finals,omitempty"`
	SuperCupSemiFinals    map[string]tournament.CupTie         `json:"super_cup_semi_finals,omitempty"`
	SuperCupFinal         tournament.CupTie                    `json:"super_cup_final,omitempty"`
	FavouriteClubID       string                               `json:"favourite_club_id,omitempty"`
	LastCareerShuffle     bool                                 `json:"last_career_shuffle,omitempty"`
}

// SavePath returns the resolved file path for saving career snapshots,
// checking FOOTBALL_SIM_SAVE environment variable with fallback detection.
func SavePath() string {
	if env := os.Getenv("FOOTBALL_SIM_SAVE"); env != "" {
		return env
	}
	// Check relative to current working directory or parent directory
	if _, err := os.Stat("saves"); err == nil {
		return filepath.Clean(filepath.Join("saves", "career.json"))
	}
	if _, err := os.Stat(filepath.Join("..", "saves")); err == nil {
		return filepath.Clean(filepath.Join("..", "saves", "career.json"))
	}
	return filepath.Clean(DefaultSavePath)
}

// BuildSnapshot extracts in-memory state into a CareerSnapshot structure.
func BuildSnapshot(
	tm *tournament.TournamentManager,
	ge *growth.GrowthEngine,
	te *transfers.TransferEngine,
) *CareerSnapshot {
	snap := &CareerSnapshot{
		Version:               SaveVersion,
		SeasonName:            tm.SeasonName,
		CurrentMatchweek:      tm.CurrentMatchweek,
		MaxMatchweeks:         tm.MaxMatchweeks,
		SeasonPhase:           tm.SeasonPhase,
		SeasonHistory:         tm.SeasonHistory,
		Inbox:                 tm.Inbox,
		DerbyHeat:             tm.DerbyHeat,
		WonderkidMilestones:   tm.MilestonesFired,
		ManagerConsecutiveHot: tm.ManagerConsecutiveHot,
		Clubs:                 make(map[string]*models.Club),
		Fixtures:              tm.Fixtures,
		UCLFixtures:           tm.UCLFixtures,
		SuperCupFixtures:      tm.SuperCupFixtures,
		UCLStage:              tm.UCLStage,
		UCLChampionID:         tm.UCLChampionID,
		SuperCupStage:         tm.SuperCupStage,
		SuperCupChampionID:    tm.SuperCupChampionID,
		RecentTicker:          tm.RecentResults,
		ProdigyHomes:          copyStringMap(tm.ProdigyHomes),
		ClubSeasonHistory:     tm.ClubSeasonHistory,
		MatchweekWeather:      tm.MatchweekWeather,
		GrowthNotes:           append([]string(nil), tm.GrowthNotifications...),
		InboxSeq:              tm.InboxSeq,
		PlayerOfTheWeek:       tm.PlayerOfTheWeek,
		MonthlyAwards:         tm.MonthlyAwards,
		UCLGroupA:             clubIDs(tm.UCLGroupA),
		UCLGroupB:             clubIDs(tm.UCLGroupB),
		UCLRecords:            tm.UCLRecords,
		UCLQuarterFinals:      tm.UCLQuarterFinals,
		UCLSemiFinals:         tm.UCLSemiFinals,
		UCLFinal:              tm.UCLFinal,
		SuperCupByes:          clubIDs(tm.SuperCupByes),
		SuperCupPlayIn:        tm.SuperCupPlayIn,
		SuperCupQuarterFinals: tm.SuperCupQuarterFinals,
		SuperCupSemiFinals:    tm.SuperCupSemiFinals,
		SuperCupFinal:         tm.SuperCupFinal,
		FavouriteClubID:       tm.FavouriteClubID,
		LastCareerShuffle:     tm.LastCareerShuffle,
	}

	// Snapshot all clubs and rosters
	for cid, club := range tm.Clubs {
		snap.Clubs[cid] = club
	}

	// Growth engine snapshot
	if ge != nil {
		snap.Growth = GrowthSnapshot{
			TrainingEnergy:    ge.TrainingEnergy,
			MaxTrainingEnergy: ge.MaxTrainingEnergy,
			Biometrics:        ge.Biometrics,
			Attributes:        ge.Attributes,
			Milestones:        ge.Milestones,
			Timeline:          ge.Timeline,
		}
	}

	// Transfer engine snapshot
	if te != nil {
		budgets := make(map[string]int64)
		for cid, m := range te.Managers {
			budgets[cid] = m.BudgetEur
		}
		snap.Transfers = TransfersSnapshot{
			CurrentDay:            te.CurrentDay,
			CurrentMatchweek:      te.CurrentMatchweek,
			CurrentWeek:           te.CurrentWeek,
			IsOffSeason:           te.IsOffSeason,
			TransferredThisWindow: copyBoolMap(te.TransferredThisWindow),
			Feed:                  te.TransferFeed,
			Completed:             te.CompletedTransfers,
			AllTime:               te.AllTimeTransfers,
			ActiveNegotiations:    te.ActiveNegotiations,
			ManagerBudgets:        budgets,
		}
	}

	return snap
}

// SaveCareer snapshots the live world and writes it atomically to disk.
func SaveCareer(
	tm *tournament.TournamentManager,
	ge *growth.GrowthEngine,
	te *transfers.TransferEngine,
	destPath string,
) (string, error) {
	return WriteSnapshot(BuildSnapshot(tm, ge, te), destPath)
}

// WriteSnapshot marshals an already-taken career snapshot and stores it as a
// sharded save: one club file per squad plus a manifest at destPath.
// Callers that hold a world lock should encode under that lock and write the
// bytes afterwards so disk I/O cannot pin the live ticker.
func WriteSnapshot(snap *CareerSnapshot, destPath string) (string, error) {
	if snap == nil {
		return "", fmt.Errorf("career snapshot is nil")
	}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to encode career snapshot: %w", err)
	}
	return WriteSnapshotBytes(data, destPath)
}

// WriteSnapshotBytes stores already-encoded snapshot JSON as a sharded save.
// Only club files whose bytes changed are rewritten, so full time of one
// match touches the manifest plus the two participating squads instead of
// every club file. Legacy single-file saves remain readable via LoadCareer.
func WriteSnapshotBytes(data []byte, destPath string) (string, error) {
	return writeShardedSnapshot(data, destPath)
}

// clubsDirFor returns the sidecar directory holding per-club squad files.
func clubsDirFor(destPath string) string {
	return filepath.Join(filepath.Dir(destPath), "clubs")
}

// safeClubFileName keeps club IDs filesystem-safe for sidecar files.
func safeClubFileName(clubID string) string {
	var b strings.Builder
	for _, r := range clubID {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	if b.Len() == 0 {
		return "club"
	}
	return b.String()
}

// writeAtomic replaces path atomically via temp file plus rename.
func writeAtomic(path string, data []byte) error {
	tmpFile := path + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp save file: %w", err)
	}
	if err := os.Rename(tmpFile, path); err != nil {
		// On Windows, if destination exists, remove first then rename
		_ = os.Remove(path)
		if err := os.Rename(tmpFile, path); err != nil {
			return fmt.Errorf("failed to commit save file: %w", err)
		}
	}
	return nil
}

// writeShardedSnapshot splits full snapshot JSON into per-club sidecars plus
// a manifest. Sections are carried as verbatim RawMessages so untouched data
// round-trips byte-identically; club files are rewritten only when their
// bytes differ, preserving mtimes for clean squads. Clubs are stored before
// the manifest so a crash can never leave a manifest pointing at missing
// squad files.
func writeShardedSnapshot(data []byte, destPath string) (string, error) {
	if destPath == "" {
		destPath = SavePath()
	}
	var full map[string]json.RawMessage
	if err := json.Unmarshal(data, &full); err != nil {
		return "", fmt.Errorf("failed to decode career snapshot JSON: %w", err)
	}
	clubsRaw, ok := full["clubs"]
	if !ok || len(clubsRaw) == 0 || string(clubsRaw) == "null" {
		return "", fmt.Errorf("career snapshot has no clubs map")
	}
	var clubs map[string]json.RawMessage
	if err := json.Unmarshal(clubsRaw, &clubs); err != nil {
		return "", fmt.Errorf("failed to decode clubs map: %w", err)
	}

	dir := clubsDirFor(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create save directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create save directory: %w", err)
	}

	index := make(map[string]map[string]string, len(clubs))
	for id, raw := range clubs {
		if len(raw) == 0 || string(raw) == "null" {
			continue
		}
		name := safeClubFileName(id) + ".json"
		path := filepath.Join(dir, name)
		sum := sha256.Sum256(raw)
		if existing, err := os.ReadFile(path); err != nil || !bytes.Equal(existing, raw) {
			if err := writeAtomic(path, raw); err != nil {
				return "", err
			}
		}
		index[id] = map[string]string{
			"file":   "clubs/" + name,
			"sha256": hex.EncodeToString(sum[:]),
		}
	}
	indexRaw, err := json.Marshal(index)
	if err != nil {
		return "", fmt.Errorf("failed to encode club index: %w", err)
	}
	full[clubIndexKey] = indexRaw
	delete(full, "clubs")

	manifest, err := json.MarshalIndent(full, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to encode career manifest: %w", err)
	}
	if err := writeAtomic(destPath, manifest); err != nil {
		return "", err
	}
	return destPath, nil
}

// LoadCareer reads and decodes a saved CareerSnapshot from disk. It accepts
// both the sharded layout (manifest plus clubs/*.json sidecars) and legacy
// single-file saves with an inline clubs map.
func LoadCareer(srcPath string) (*CareerSnapshot, error) {
	if srcPath == "" {
		srcPath = SavePath()
	}

	data, err := os.ReadFile(srcPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read career save file: %w", err)
	}

	var probe map[string]json.RawMessage
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, fmt.Errorf("failed to parse career snapshot JSON: %w", err)
	}
	if _, sharded := probe[clubIndexKey]; !sharded {
		var snap CareerSnapshot
		if err := json.Unmarshal(data, &snap); err != nil {
			return nil, fmt.Errorf("failed to parse career snapshot JSON: %w", err)
		}
		return &snap, nil
	}
	return loadShardedCareer(srcPath, probe)
}

// loadShardedCareer assembles a CareerSnapshot from a manifest plus club
// sidecars. Sidecar hashes are verified so corruption surfaces as an error
// instead of a silently wrong career.
func loadShardedCareer(srcPath string, manifest map[string]json.RawMessage) (*CareerSnapshot, error) {
	var index map[string]struct {
		File   string `json:"file"`
		Sha256 string `json:"sha256"`
	}
	if err := json.Unmarshal(manifest[clubIndexKey], &index); err != nil {
		return nil, fmt.Errorf("failed to parse club index: %w", err)
	}
	clubs := make(map[string]*models.Club, len(index))
	for id, ref := range index {
		clubPath := filepath.Join(filepath.Dir(srcPath), filepath.FromSlash(ref.File))
		raw, err := os.ReadFile(clubPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read club save file %s: %w", ref.File, err)
		}
		sum := sha256.Sum256(raw)
		if ref.Sha256 != "" && hex.EncodeToString(sum[:]) != ref.Sha256 {
			return nil, fmt.Errorf("club save file %s failed checksum", ref.File)
		}
		var club models.Club
		if err := json.Unmarshal(raw, &club); err != nil {
			return nil, fmt.Errorf("failed to parse club save file %s: %w", ref.File, err)
		}
		clubs[id] = &club
	}

	delete(manifest, clubIndexKey)
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to decode career manifest: %w", err)
	}
	var snap CareerSnapshot
	if err := json.Unmarshal(manifestBytes, &snap); err != nil {
		return nil, fmt.Errorf("failed to parse career manifest: %w", err)
	}
	snap.Clubs = clubs
	return &snap, nil
}

// RestoreCareer applies a deserialized snapshot onto an existing live world.
// Preserves referential integrity, reconnects pointers, and ensures 0 squad duplicates.
func RestoreCareer(
	tm *tournament.TournamentManager,
	ge *growth.GrowthEngine,
	te *transfers.TransferEngine,
	snap *CareerSnapshot,
) error {
	if snap == nil {
		return fmt.Errorf("cannot restore nil snapshot")
	}

	// 1. Restore Tournament Metadata & Narratives
	if snap.SeasonName != "" {
		tm.SeasonName = snap.SeasonName
	}
	if snap.CurrentMatchweek > 0 {
		tm.CurrentMatchweek = snap.CurrentMatchweek
	}
	if snap.MaxMatchweeks > 0 {
		tm.MaxMatchweeks = snap.MaxMatchweeks
	}
	if snap.SeasonPhase != "" {
		tm.SeasonPhase = snap.SeasonPhase
	}
	if snap.SeasonHistory != nil {
		tm.SeasonHistory = snap.SeasonHistory
	}
	if snap.Inbox != nil {
		tm.Inbox = snap.Inbox
	}
	if snap.DerbyHeat != nil {
		tm.DerbyHeat = snap.DerbyHeat
	}
	if snap.WonderkidMilestones != nil {
		tm.MilestonesFired = snap.WonderkidMilestones
	}
	if snap.ManagerConsecutiveHot != nil {
		tm.ManagerConsecutiveHot = snap.ManagerConsecutiveHot
	}
	if snap.UCLFixtures != nil {
		tm.UCLFixtures = snap.UCLFixtures
	}
	if snap.SuperCupFixtures != nil {
		tm.SuperCupFixtures = snap.SuperCupFixtures
	}
	if snap.UCLStage != "" {
		tm.UCLStage = snap.UCLStage
	}
	tm.UCLChampionID = snap.UCLChampionID
	if snap.SuperCupStage != "" {
		tm.SuperCupStage = snap.SuperCupStage
	}
	tm.SuperCupChampionID = snap.SuperCupChampionID
	if snap.RecentTicker != nil {
		tm.RecentResults = snap.RecentTicker
	}
	if len(snap.ProdigyHomes) > 0 {
		tm.ProdigyHomes = copyStringMap(snap.ProdigyHomes)
	}
	if snap.ClubSeasonHistory != nil {
		tm.ClubSeasonHistory = snap.ClubSeasonHistory
	}
	if snap.MatchweekWeather != nil {
		tm.MatchweekWeather = snap.MatchweekWeather
	}
	if snap.GrowthNotes != nil {
		tm.GrowthNotifications = snap.GrowthNotes
	}
	if snap.PlayerOfTheWeek != nil {
		if m, ok := snap.PlayerOfTheWeek.(map[string]interface{}); ok {
			tm.PlayerOfTheWeek = m
		}
	}
	if snap.MonthlyAwards != nil {
		tm.MonthlyAwards = snap.MonthlyAwards
	}
	if snap.InboxSeq > tm.InboxSeq {
		tm.InboxSeq = snap.InboxSeq
	}
	if snap.FavouriteClubID != "" {
		tm.FavouriteClubID = snap.FavouriteClubID
	}
	tm.LastCareerShuffle = snap.LastCareerShuffle

	// 2. Build index of existing players for fast lookup and deduplication
	existingPlayers := make(map[string]*models.Player)
	existingByName := make(map[string]*models.Player)
	for _, club := range tm.ClubsList {
		for _, p := range club.Squad {
			existingPlayers[p.PlayerID] = p
			nameKey := strings.ToLower(strings.TrimSpace(p.FullName))
			existingByName[nameKey] = p
		}
	}

	// 3. Restore Clubs, Standings & Rosters
	for clubID, club := range tm.Clubs {
		savedClub, ok := snap.Clubs[clubID]
		if !ok {
			continue
		}

		// Standings & Form
		club.Played = savedClub.Played
		club.Won = savedClub.Won
		club.Drawn = savedClub.Drawn
		club.Lost = savedClub.Lost
		club.GoalsFor = savedClub.GoalsFor
		club.GoalsAgainst = savedClub.GoalsAgainst
		club.GoalDifference = savedClub.GoalDifference
		club.Points = savedClub.Points
		if savedClub.Form != nil {
			club.Form = savedClub.Form
		}
		if savedClub.Morale > 0 {
			club.Morale = savedClub.Morale
		}

		// Squad synchronization
		if len(savedClub.Squad) > 0 {
			for _, savedPlayer := range savedClub.Squad {
				if savedPlayer == nil {
					continue
				}

				// Resolve canonical ID for wonderkids
				pid := savedPlayer.PlayerID
				if canonical, ok := ProdigyMap[pid]; ok {
					pid = canonical
					savedPlayer.PlayerID = canonical
				}

				nameKey := strings.ToLower(strings.TrimSpace(savedPlayer.FullName))
				livePlayer := existingPlayers[pid]
				if livePlayer == nil {
					livePlayer = existingByName[nameKey]
				}

				if livePlayer != nil {
					// Update player stats and progress
					updatePlayerFromSaved(livePlayer, savedPlayer)

					// Move player to target club if transferred
					targetClubID := savedPlayer.ClubID
					if targetClubID == "" {
						targetClubID = clubID
					}
					if livePlayer.ClubID != targetClubID {
						removePlayerFromClub(tm.Clubs[livePlayer.ClubID], livePlayer)
						livePlayer.ClubID = targetClubID
						if destClub, ok := tm.Clubs[targetClubID]; ok {
							destClub.Squad = append(destClub.Squad, livePlayer)
						}
					}
				} else {
					// New regen or academy player
					savedPlayer.ClubID = clubID
					club.Squad = append(club.Squad, savedPlayer)
					existingPlayers[savedPlayer.PlayerID] = savedPlayer
					existingByName[nameKey] = savedPlayer
				}
			}
		}
	}

	// 4. Enforce strict squad deduplication across all clubs
	for _, club := range tm.ClubsList {
		seen := make(map[string]bool)
		cleanSquad := make([]*models.Player, 0, len(club.Squad))
		for _, p := range club.Squad {
			key := strings.ToLower(strings.TrimSpace(p.FullName))
			if seen[key] || seen[p.PlayerID] {
				continue
			}
			seen[key] = true
			seen[p.PlayerID] = true
			cleanSquad = append(cleanSquad, p)
		}
		club.Squad = cleanSquad
		club.SquadSize = len(club.Squad)
		club.RecalculateRatings()
	}

	// 5. Restore calendars wholesale so a freshly generated Berger table
	// cannot drop finished results when fixture IDs or cycle-3 pairings differ.
	if len(snap.Fixtures) > 0 {
		tm.Fixtures = append([]tournament.Fixture(nil), snap.Fixtures...)
		wireFixtureClubs(tm, tm.Fixtures)
	}
	if len(snap.UCLFixtures) > 0 {
		tm.UCLFixtures = append([]tournament.Fixture(nil), snap.UCLFixtures...)
		wireFixtureClubs(tm, tm.UCLFixtures)
	}
	if len(snap.SuperCupFixtures) > 0 {
		tm.SuperCupFixtures = append([]tournament.Fixture(nil), snap.SuperCupFixtures...)
		wireFixtureClubs(tm, tm.SuperCupFixtures)
	}
	if g := clubsFromIDs(tm, snap.UCLGroupA); len(g) > 0 {
		tm.UCLGroupA = g
	}
	if g := clubsFromIDs(tm, snap.UCLGroupB); len(g) > 0 {
		tm.UCLGroupB = g
	}
	if snap.UCLRecords != nil {
		tm.UCLRecords = snap.UCLRecords
	}
	if snap.UCLQuarterFinals != nil {
		tm.UCLQuarterFinals = snap.UCLQuarterFinals
	}
	if snap.UCLSemiFinals != nil {
		tm.UCLSemiFinals = snap.UCLSemiFinals
	}
	if snap.UCLFinal.HomeID != "" || snap.UCLFinal.AwayID != "" || snap.UCLFinal.WinnerID != "" {
		tm.UCLFinal = snap.UCLFinal
	}
	if byes := clubsFromIDs(tm, snap.SuperCupByes); len(byes) > 0 {
		tm.SuperCupByes = byes
	}
	if snap.SuperCupPlayIn != nil {
		tm.SuperCupPlayIn = snap.SuperCupPlayIn
	}
	if snap.SuperCupQuarterFinals != nil {
		tm.SuperCupQuarterFinals = snap.SuperCupQuarterFinals
	}
	if snap.SuperCupSemiFinals != nil {
		tm.SuperCupSemiFinals = snap.SuperCupSemiFinals
	}
	if snap.SuperCupFinal.HomeID != "" || snap.SuperCupFinal.AwayID != "" || snap.SuperCupFinal.WinnerID != "" {
		tm.SuperCupFinal = snap.SuperCupFinal
	}

	// 6. Restore Growth Engine
	if ge != nil {
		if snap.Growth.TrainingEnergy > 0 {
			ge.TrainingEnergy = snap.Growth.TrainingEnergy
		}
		if snap.Growth.MaxTrainingEnergy > 0 {
			ge.MaxTrainingEnergy = snap.Growth.MaxTrainingEnergy
		}
		if snap.Growth.Biometrics != nil {
			for pid, bio := range snap.Growth.Biometrics {
				canonical := pid
				if c, ok := ProdigyMap[pid]; ok {
					canonical = c
				}
				bio.PlayerID = canonical
				ge.Biometrics[canonical] = bio
			}
		}
		if snap.Growth.Attributes != nil {
			for pid, attrs := range snap.Growth.Attributes {
				canonical := pid
				if c, ok := ProdigyMap[pid]; ok {
					canonical = c
				}
				ge.Attributes[canonical] = attrs
			}
		}
		if snap.Growth.Milestones != nil {
			ge.Milestones = snap.Growth.Milestones
		}
		if snap.Growth.Timeline != nil {
			ge.Timeline = snap.Growth.Timeline
		}
	}

	// 7. Restore Transfer Engine
	if te != nil {
		if snap.Transfers.CurrentDay > 0 {
			te.CurrentDay = snap.Transfers.CurrentDay
		}
		if snap.Transfers.CurrentMatchweek > 0 {
			te.CurrentMatchweek = snap.Transfers.CurrentMatchweek
		}
		if snap.Transfers.CurrentWeek > 0 {
			te.CurrentWeek = snap.Transfers.CurrentWeek
		}
		te.IsOffSeason = snap.Transfers.IsOffSeason
		if snap.Transfers.TransferredThisWindow != nil {
			te.TransferredThisWindow = copyBoolMap(snap.Transfers.TransferredThisWindow)
		}
		if snap.Transfers.Feed != nil {
			te.TransferFeed = snap.Transfers.Feed
		}
		if snap.Transfers.Completed != nil {
			te.CompletedTransfers = snap.Transfers.Completed
		}
		if snap.Transfers.AllTime != nil {
			te.AllTimeTransfers = snap.Transfers.AllTime
		}
		if snap.Transfers.ActiveNegotiations != nil {
			// Reconnect pointers to live clubs and players
			for _, neg := range snap.Transfers.ActiveNegotiations {
				if neg.Player != nil {
					if p := existingPlayers[neg.Player.PlayerID]; p != nil {
						neg.Player = p
					}
				}
				if neg.Buyer != nil {
					if b := tm.Clubs[neg.Buyer.ClubID]; b != nil {
						neg.Buyer = b
					}
				}
				if neg.Seller != nil {
					if s := tm.Clubs[neg.Seller.ClubID]; s != nil {
						neg.Seller = s
					}
				}
				if neg.OriginalBuyer != nil {
					if ob := tm.Clubs[neg.OriginalBuyer.ClubID]; ob != nil {
						neg.OriginalBuyer = ob
					}
				}
			}
			te.ActiveNegotiations = snap.Transfers.ActiveNegotiations
		}
		if snap.Transfers.ManagerBudgets != nil {
			for cid, budget := range snap.Transfers.ManagerBudgets {
				if m, ok := te.Managers[cid]; ok {
					m.BudgetEur = budget
				}
			}
		}
	}

	// 8. Stretch short legacy calendars, then re-pair mentors on the restored squads.
	_ = tm.AdoptLongSeason()
	tournament.PairSeniorMentors(tm.ClubsList, ge)
	return nil
}

// DeleteCareer deletes the snapshot manifest and its club sidecars, if any.
func DeleteCareer(path string) error {
	if path == "" {
		path = SavePath()
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.RemoveAll(clubsDirFor(path)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func clubIDs(clubs []*models.Club) []string {
	if len(clubs) == 0 {
		return nil
	}
	ids := make([]string, 0, len(clubs))
	for _, c := range clubs {
		if c != nil {
			ids = append(ids, c.ClubID)
		}
	}
	return ids
}

func clubsFromIDs(tm *tournament.TournamentManager, ids []string) []*models.Club {
	if tm == nil || len(ids) == 0 {
		return nil
	}
	out := make([]*models.Club, 0, len(ids))
	for _, id := range ids {
		if c := tm.Clubs[id]; c != nil {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func wireFixtureClubs(tm *tournament.TournamentManager, fixtures []tournament.Fixture) {
	for i := range fixtures {
		fixtures[i].Home = tm.Clubs[fixtures[i].HomeID]
		fixtures[i].Away = tm.Clubs[fixtures[i].AwayID]
	}
}

func copyStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func copyBoolMap(in map[string]bool) map[string]bool {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]bool, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func updatePlayerFromSaved(dest, src *models.Player) {
	dest.OVR = src.OVR
	dest.Age = src.Age
	dest.MarketValueEUR = src.MarketValueEUR
	dest.WageEUR = src.WageEUR
	dest.ContractYears = src.ContractYears
	dest.Loyalty = src.Loyalty
	dest.Goals = src.Goals
	dest.Assists = src.Assists
	dest.Appearances = src.Appearances
	dest.CareerGoals = src.CareerGoals
	dest.CareerAssists = src.CareerAssists
	dest.CareerApps = src.CareerApps
	dest.BestGoals = src.BestGoals
	dest.BestAssists = src.BestAssists
	dest.BestSeason = src.BestSeason
	dest.OwnGoals = src.OwnGoals
	dest.SuspendedMatches = src.SuspendedMatches
	dest.InjuredMatches = src.InjuredMatches
	dest.Injury = src.Injury
	dest.Education = src.Education
	dest.EducationPending = src.EducationPending
	dest.SchoolWant = src.SchoolWant
	dest.SchoolTrack = src.SchoolTrack
	dest.PositionPath = src.PositionPath
	dest.SecondaryPosition = src.SecondaryPosition
	dest.PositionXP = src.PositionXP
	dest.Personality = src.Personality
	dest.MentorID = src.MentorID
	dest.MentorName = src.MentorName
	dest.MentorOVR = src.MentorOVR
	dest.Composure = src.Composure
	dest.ConsecutiveStarts = src.ConsecutiveStarts
	if src.Category != "" {
		dest.Category = src.Category
	}
}

func removePlayerFromClub(club *models.Club, target *models.Player) {
	if club == nil {
		return
	}
	idx := -1
	for i, p := range club.Squad {
		if p.PlayerID == target.PlayerID {
			idx = i
			break
		}
	}
	if idx >= 0 {
		club.Squad = append(club.Squad[:idx], club.Squad[idx+1:]...)
	}
}
