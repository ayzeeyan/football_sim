package persistence

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"football_sim/pkg/growth"
	"football_sim/pkg/managers"
	"football_sim/pkg/matchreport"
	"football_sim/pkg/medical"
	"football_sim/pkg/models"
	"football_sim/pkg/tournament"
	"football_sim/pkg/transfers"
)

const (
	// SaveVersion 9 adds Fixture.ReportSummary: finished matches aged past the
	// retention window persist an archival summary instead of a full report.
	// SaveVersion 10 adds the multi-entity watchlist (WatchlistState).
	// SaveVersion 11 adds the achievement ledger (Achievements,
	// AchievementsFired, ClubUnbeatenRuns, YoungestScorer).
	// SaveVersion 12 adds per-player injury history (Player.InjuryHistory).
	// SaveVersion 13 adds the viewer lineup override (Club.LineupOverride).
	// SaveVersion 14 adds the viewer manager career (ViewerManager).
	// SaveVersion 15 adds viewer-selected national squads
	// (NationalTeamsCompetition.ViewerSquads).
	// SaveVersion 16 adds the current season's promotion/relegation moves
	// (SeasonLeagueMoves).
	// SaveVersion 17 writes back swap-era saves realigned to the
	// country-pure pyramid (domestic registries re-seeded from the clubs'
	// national leagues; the season restarts from matchweek 1).
	// SaveVersion 18 removes the directed-control fields (viewer lineup
	// override, viewer manager career, viewer national squads): the world
	// is a neutral simulation again and every club, player, and transfer is
	// machine-selected. Older saves load; the dropped keys are ignored.
	SaveVersion     = 18
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
	WindowType            transfers.WindowType             `json:"window_type,omitempty"`
	WindowOpen            bool                             `json:"window_open,omitempty"`
	ProcessedWeeks        int                              `json:"processed_weeks,omitempty"`
	TransferredThisWindow map[string]bool                  `json:"transferred_this_window,omitempty"`
	Feed                  []transfers.TransferFeedItem     `json:"feed"`
	Completed             []transfers.CompletedTransfer    `json:"completed"`
	AllTime               []transfers.CompletedTransfer    `json:"all_time"`
	ActiveNegotiations    []*transfers.TransferNegotiation `json:"active_negotiations,omitempty"`
	ManagerBudgets        map[string]int64                 `json:"manager_budgets,omitempty"`
	FreeAgents            []*models.Player                 `json:"free_agents,omitempty"`
	ScriptedSwapDone      bool                             `json:"scripted_swap_done,omitempty"`
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
	ManagerLastChange     map[string]int                       `json:"manager_last_change,omitempty"`
	ManagerHistory        []tournament.ManagerHistoryEntry     `json:"manager_history,omitempty"`
	Managers              map[string]*managers.ManagerProfile  `json:"managers,omitempty"`
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
	World                 *tournament.EuropeanWorld            `json:"world,omitempty"`
	FavouriteClubID       string                               `json:"favourite_club_id,omitempty"`
	Watchlist             tournament.WatchlistState            `json:"watchlist,omitempty"`
	Achievements          []tournament.Achievement             `json:"achievements,omitempty"`
	AchievementsFired     map[string]bool                      `json:"achievements_fired,omitempty"`
	ClubUnbeatenRuns      map[string]int                       `json:"club_unbeaten_runs,omitempty"`
	YoungestScorer        *tournament.YoungestScorerRecord     `json:"youngest_scorer,omitempty"`
	SeasonLeagueMoves     []tournament.RelegationMove          `json:"season_league_moves,omitempty"`
	LastCareerShuffle     bool                                 `json:"last_career_shuffle,omitempty"`

	ReputationAppliedSeason string   `json:"reputation_applied_season,omitempty"`
	ContractsResolvedSeason string   `json:"contracts_resolved_season,omitempty"`
	RetiredPlayerIDs        []string `json:"retired_player_ids,omitempty"`
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
	if te != nil {
		te.SyncAllManagerBudgets()
	}
	var retiredIDs []string
	if tm != nil && tm.RetiredPlayerIDs != nil {
		for id := range tm.RetiredPlayerIDs {
			retiredIDs = append(retiredIDs, id)
		}
		sort.Strings(retiredIDs)
	}
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
		ManagerLastChange:     tm.ManagerLastChange,
		ManagerHistory:        tm.ManagerHistory,
		Managers:              tm.Managers,
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
		World:                 tm.World,
		FavouriteClubID:       tm.FavouriteClubID,
		Watchlist:             tm.Watch,
		Achievements:          tm.Achievements,
		AchievementsFired:     tm.AchievementsFired,
		ClubUnbeatenRuns:      tm.ClubUnbeatenRuns,
		YoungestScorer:        tm.YoungestScorer,
		SeasonLeagueMoves:     tm.SeasonLeagueMoves,
		LastCareerShuffle:     tm.LastCareerShuffle,

		ReputationAppliedSeason: tm.ReputationAppliedSeason,
		ContractsResolvedSeason: tm.ContractsResolvedSeason,
		RetiredPlayerIDs:        retiredIDs,
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
			WindowType:            te.WindowType,
			WindowOpen:            te.WindowOpen,
			ProcessedWeeks:        te.ProcessedWeeks,
			TransferredThisWindow: copyBoolMap(te.TransferredThisWindow),
			Feed:                  te.TransferFeed,
			Completed:             te.CompletedTransfers,
			AllTime:               te.AllTimeTransfers,
			ActiveNegotiations:    te.ActiveNegotiations,
			ManagerBudgets:        budgets,
			FreeAgents:            append([]*models.Player(nil), te.FreeAgents...),
			ScriptedSwapDone:      te.ScriptedSwapDone,
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
	data, err := json.Marshal(snap)
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
	f, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to write temp save file: %w", err)
	}
	tmpFile := f.Name()
	defer os.Remove(tmpFile)
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmpFile)
		return fmt.Errorf("failed to write temp save file: %w", err)
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmpFile)
		return fmt.Errorf("failed to flush temp save file: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to close temp save file: %w", err)
	}
	if err := os.Rename(tmpFile, path); err != nil {
		return fmt.Errorf("failed to commit save file: %w", err)
	}
	return nil
}

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

	type sidecarDelta struct {
		id   string
		name string
		raw  []byte
		sum  [sha256.Size]byte
	}
	var deltas []sidecarDelta
	index := make(map[string]map[string]string, len(clubs))
	for id, raw := range clubs {
		if len(raw) == 0 || string(raw) == "null" {
			continue
		}
		name := safeClubFileName(id) + ".json"
		sum := sha256.Sum256(raw)
		if existing, err := os.ReadFile(filepath.Join(dir, name)); err == nil && bytes.Equal(existing, raw) {
			index[id] = map[string]string{
				"file":   "clubs/" + name,
				"sha256": hex.EncodeToString(sum[:]),
			}
			continue
		}
		deltas = append(deltas, sidecarDelta{id: id, name: name, raw: raw, sum: sum})
	}

	staged := make([]string, 0, len(deltas))
	abort := func(failErr error) (string, error) {
		for _, tmp := range staged {
			_ = os.Remove(tmp)
		}
		return "", failErr
	}
	for _, delta := range deltas {
		tmpPath := filepath.Join(dir, delta.name+".tmp")
		if err := writeAtomicTemp(tmpPath, delta.raw); err != nil {
			return abort(fmt.Errorf("failed to stage club save file %s: %w", delta.name, err))
		}
		staged = append(staged, tmpPath)
	}
	for _, delta := range deltas {
		path := filepath.Join(dir, delta.name)
		if err := os.Rename(tmpPathFor(dir, delta.name), path); err != nil {
			if _err := os.Remove(path); _err == nil {
				if err = os.Rename(tmpPathFor(dir, delta.name), path); err != nil {
					return abort(fmt.Errorf("failed to commit club save file %s: %w", delta.name, err))
				}
			} else {
				return abort(fmt.Errorf("failed to commit club save file %s: %w", delta.name, err))
			}
		}
		staged = staged[1:]
		index[delta.id] = map[string]string{
			"file":   "clubs/" + delta.name,
			"sha256": hex.EncodeToString(delta.sum[:]),
		}
	}

	indexRaw, err := json.Marshal(index)
	if err != nil {
		return abort(fmt.Errorf("failed to encode club index: %w", err))
	}
	full[clubIndexKey] = indexRaw
	delete(full, "clubs")

	manifest, err := json.Marshal(full)
	if err != nil {
		return abort(fmt.Errorf("failed to encode career manifest: %w", err))
	}
	manifestTmp := destPath + ".tmp"
	if err := writeAtomicTemp(manifestTmp, manifest); err != nil {
		return abort(fmt.Errorf("failed to stage career manifest: %w", err))
	}
	staged = append(staged, manifestTmp)
	if err := commitStagedFile(manifestTmp, destPath); err != nil {
		return abort(fmt.Errorf("failed to commit career manifest: %w", err))
	}
	staged = staged[:len(staged)-1]
	// Sweep sidecars the current universe no longer indexes (e.g. clubs
	// absent from the map): without this, stale clubs/*.json accumulate and
	// a future reader could resurrect dead squads. Runs last so a crash can
	// never delete before the manifest is safely committed.
	sweepStaleClubSidecars(dir, index)
	return destPath, nil
}

// tmpPathFor builds the staging path for a club sidecar during a sharded save.
func tmpPathFor(dir, name string) string {
	return filepath.Join(dir, name+".tmp")
}

// writeAtomicTemp stages bytes at tmpPath with a pre-rename fsync. The rename
// into its final name is left to the caller so multi-file publications can
// gate every commit behind all stages succeeding.
func writeAtomicTemp(tmpPath string, data []byte) error {
	f, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return err
	}
	return f.Close()
}

// commitStagedFile renames a staged temp file over its final destination,
// falling back to remove-then-rename on Windows semantics.
func commitStagedFile(tmpPath, path string) error {
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(path)
		if err := os.Rename(tmpPath, path); err != nil {
			return err
		}
	}
	return nil
}

// sweepStaleClubSidecars removes clubs/*.json files no current index entry
// references. Failures are ignored: leftovers are harmless (the manifest
// index is authoritative on load) and saves must not fail on cleanup.
func sweepStaleClubSidecars(dir string, index map[string]map[string]string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	live := make(map[string]bool, len(index))
	for _, meta := range index {
		live[filepath.Base(meta["file"])] = true
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if len(name) < 6 || name[len(name)-5:] != ".json" {
			continue
		}
		if !live[name] {
			_ = os.Remove(filepath.Join(dir, name))
		}
	}
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
		wireSnapshotFixtures(&snap)
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
	wireSnapshotFixtures(&snap)
	return &snap, nil
}

func wireSnapshotFixtures(snap *CareerSnapshot) {
	if snap == nil {
		return
	}
	// Normalize legacy idle transfer window counters (pre-v5 saves used Week 1 as sentinel)
	if snap.Transfers.WindowType == transfers.WindowClosed && !snap.Transfers.IsOffSeason {
		snap.Transfers.CurrentWeek = 0
		snap.Transfers.CurrentDay = 0
		snap.Transfers.ProcessedWeeks = 0
		snap.Transfers.WindowOpen = false
	}
	if len(snap.Clubs) == 0 {
		return
	}
	wire := func(fixtures []tournament.Fixture) {
		for i := range fixtures {
			if fixtures[i].Home == nil && fixtures[i].HomeID != "" {
				fixtures[i].Home = snap.Clubs[fixtures[i].HomeID]
			}
			if fixtures[i].Away == nil && fixtures[i].AwayID != "" {
				fixtures[i].Away = snap.Clubs[fixtures[i].AwayID]
			}
		}
	}
	wire(snap.Fixtures)
	wire(snap.UCLFixtures)
	wire(snap.SuperCupFixtures)
	if snap.World != nil {
		wire(snap.World.Fixtures)
	}
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
	tm.ReputationAppliedSeason = snap.ReputationAppliedSeason
	tm.ContractsResolvedSeason = snap.ContractsResolvedSeason
	tm.RetiredPlayerIDs = make(map[string]bool)
	for _, id := range snap.RetiredPlayerIDs {
		tm.RetiredPlayerIDs[id] = true
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
	if snap.ManagerLastChange != nil {
		tm.ManagerLastChange = snap.ManagerLastChange
	}
	if snap.ManagerHistory != nil {
		tm.ManagerHistory = snap.ManagerHistory
	}
	if snap.Managers != nil {
		tm.Managers = snap.Managers
		for cid, manager := range tm.Managers {
			if manager == nil {
				continue
			}
			manager.ClubID = cid
			if manager.JobSecurity == "" {
				manager.JobSecurity = "Safe"
			}
		}
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
	tm.Watch = snap.Watchlist
	if len(snap.Achievements) > 0 {
		tm.Achievements = snap.Achievements
	}
	if snap.AchievementsFired != nil {
		tm.AchievementsFired = snap.AchievementsFired
	}
	if snap.ClubUnbeatenRuns != nil {
		tm.ClubUnbeatenRuns = snap.ClubUnbeatenRuns
	}
	if snap.YoungestScorer != nil {
		tm.YoungestScorer = snap.YoungestScorer
	}
	tm.SeasonLeagueMoves = snap.SeasonLeagueMoves
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

		// Identity is part of the persisted club state. A presence-aware zero
		// means an older/in-memory snapshot omitted identity, so retain the live
		// club's preset; explicit persisted zero values still restore as zero.
		if !savedClub.Identity.IsZero() {
			club.Identity = savedClub.Identity.Clamp()
		}
		// Saved finances always win on restore, including genuine €0
		// balances: the snapshot is authoritative for the saved moment.
		// (WageCap/EuropeanRevenue round-trip as struct fields; UnmarshalJSON
		// already migrated legacy caps below the committed bill.)
		club.Finances = savedClub.Finances
		if tm.Managers != nil {
			if mgr := tm.Managers[clubID]; mgr != nil {
				mgr.BudgetEur = club.Finances.TransferBudget
			}
		}

		// Standings & Form
		club.Coefficient = savedClub.Coefficient
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
		if savedClub.BoardObjective != "" {
			club.BoardObjective = savedClub.BoardObjective
		}
		if savedClub.ExpectedFinish > 0 {
			club.ExpectedFinish = savedClub.ExpectedFinish
		}
		club.CaptainID = savedClub.CaptainID
		club.ViceCaptainID = savedClub.ViceCaptainID
		club.FanExpectation = savedClub.FanExpectation
		club.MediaPressure = savedClub.MediaPressure
		club.Chemistry = savedClub.Chemistry
		club.SeasonAttendance = savedClub.SeasonAttendance
		club.AttendanceMatches = savedClub.AttendanceMatches
		club.PowerRank = savedClub.PowerRank

		// Squad synchronization
		if len(savedClub.Squad) > 0 {
			for _, savedPlayer := range savedClub.Squad {
				if savedPlayer == nil {
					continue
				}

				// Resolve canonical ID for wonderkids
				pid := savedPlayer.PlayerID
				// P00xxx aliases belong only to legacy wonderkid records. The
				// Top Five dataset legitimately reuses some historical numeric
				// IDs for ordinary players, so never remap them by ID alone.
				if canonical, ok := ProdigyMap[pid]; ok && savedPlayer.UniverseWonderkid {
					pid = canonical
					savedPlayer.PlayerID = canonical
				}

				nameKey := strings.ToLower(strings.TrimSpace(savedPlayer.FullName))
				livePlayer := existingPlayers[pid]
				// Player IDs are the ownership key. A name fallback is reserved for
				// genuinely ID-less legacy rows; using it for a missing modern ID
				// can steal a canonical wonderkid when two dataset rows share a
				// display name.
				if livePlayer == nil && pid == "" {
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
					if savedPlayer.OriginalClubID == "" {
						savedPlayer.OriginalClubID = clubID
					}
					club.Squad = append(club.Squad, savedPlayer)
					existingPlayers[savedPlayer.PlayerID] = savedPlayer
					existingByName[nameKey] = savedPlayer
				}
			}
		}
	}

	// 4. Saved squad membership is authoritative. Overlaying academy rows
	// onto a fresh dataset squad without dropping retired/departed players
	// is what pushes clubs such as Inter past the 34-player ceiling.
	for clubID, club := range tm.Clubs {
		savedClub, ok := snap.Clubs[clubID]
		if !ok || club == nil {
			continue
		}
		rebuilt := make([]*models.Player, 0, len(savedClub.Squad))
		seen := make(map[string]bool, len(savedClub.Squad))
		for _, savedPlayer := range savedClub.Squad {
			if savedPlayer == nil {
				continue
			}
			pid := savedPlayer.PlayerID
			if canonical, mapped := ProdigyMap[pid]; mapped && savedPlayer.UniverseWonderkid {
				pid = canonical
			}
			if pid == "" || seen[pid] || tm.IsPlayerRetired(pid) {
				continue
			}
			live := existingPlayers[pid]
			if live == nil {
				continue
			}
			seen[pid] = true
			live.ClubID = clubID
			rebuilt = append(rebuilt, live)
		}
		club.Squad = rebuilt
		club.SquadSize = len(club.Squad)
		club.RecalculateRatings()
	}
	tm.EnforceRosterCapsUnlocked()
	tm.SyncCaptainFlagsUnlocked()

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
	if snap.World != nil {
		tm.World = snap.World
		wireWorldFixtureClubs(tm)
		// Swap-era saves can carry domestic registries that disagree with
		// the dataset-derived club leagues. The country-pure invariant is
		// enforced here: a misaligned world is re-seeded and the season
		// restarts from matchweek 1 (all-time history kept).
		if tm.RealignCountryPureWorld() {
			wireWorldFixtureClubs(tm)
		}
	}
	// Tactical-slot fields are additive to the current save format. Finished
	// reports from before the field existed are reconstructed in memory from
	// the manager's deterministic formation and the saved natural positions.
	// Older SaveVersion values are written back immediately after a successful
	// restore via MaybeWriteMigratedCareer.
	formationForClub := func(clubID string) string {
		if manager := tm.Managers[clubID]; manager != nil {
			return models.FormationForStyle(manager.Style)
		}
		return models.Formation433
	}
	backfillReports := func(fixtures []tournament.Fixture) {
		for i := range fixtures {
			fixture := &fixtures[i]
			matchreport.BackfillTacticalSlots(fixture.Report, formationForClub(fixture.HomeID), formationForClub(fixture.AwayID))
		}
	}
	backfillReports(tm.Fixtures)
	backfillReports(tm.UCLFixtures)
	backfillReports(tm.SuperCupFixtures)
	if tm.World != nil {
		backfillReports(tm.World.Fixtures)
	}
	tm.CompactAgedReports()
	for _, club := range tm.ClubsList {
		if club == nil {
			continue
		}
		for _, p := range club.Squad {
			if p != nil {
				p.BackfillElapsedWonderkidContract()
			}
		}
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
		// Legacy P00xxx growth keys follow the same guarded remap as squads:
		// only aliases of restored wonderkids move (Top Five numeric IDs can
		// collide with legacy aliases for ordinary players). Milestones carry
		// no player ID and restore wholesale.
		wkIDs := make(map[string]bool)
		for _, club := range tm.ClubsList {
			for _, p := range club.Squad {
				if p != nil && p.UniverseWonderkid {
					wkIDs[p.PlayerID] = true
				}
			}
		}
		remapGrowthKey := func(pid string) string {
			if c, ok := ProdigyMap[pid]; ok && wkIDs[c] {
				return c
			}
			return pid
		}
		if snap.Growth.Biometrics != nil {
			for pid, bio := range snap.Growth.Biometrics {
				canonical := remapGrowthKey(pid)
				bio.PlayerID = canonical
				ge.Biometrics[canonical] = bio
			}
		}
		if snap.Growth.Attributes != nil {
			for pid, attrs := range snap.Growth.Attributes {
				ge.Attributes[remapGrowthKey(pid)] = attrs
			}
		}
		if snap.Growth.Milestones != nil {
			ge.Milestones = snap.Growth.Milestones
		}
		if snap.Growth.Timeline != nil {
			remapped := make(map[string][]growth.TimelineEntry, len(snap.Growth.Timeline))
			for pid, entries := range snap.Growth.Timeline {
				remapped[remapGrowthKey(pid)] = entries
			}
			ge.Timeline = remapped
		}
		// Version 8 raised all twelve canonical ceilings to 99. Preserve the
		// saved OVR and development history while upgrading old careers.
		ge.UpgradeCanonicalWonderkidPotentials()
	}

	// 7. Restore Transfer Engine
	if te != nil {
		if tm.Clubs != nil {
			te.Clubs = tm.Clubs
		}
		if tm.Managers != nil {
			te.Managers = tm.Managers
		}
		te.SyncAllManagerBudgets()
		if snap.Version >= 5 || snap.Transfers.CurrentDay > 0 {
			te.CurrentDay = snap.Transfers.CurrentDay
		}
		if snap.Transfers.CurrentMatchweek > 0 {
			te.CurrentMatchweek = snap.Transfers.CurrentMatchweek
		}
		if snap.Version >= 5 || snap.Transfers.CurrentWeek > 0 {
			te.CurrentWeek = snap.Transfers.CurrentWeek
		}
		te.IsOffSeason = snap.Transfers.IsOffSeason
		if snap.Transfers.WindowType != "" {
			te.WindowType = snap.Transfers.WindowType
			te.WindowOpen = snap.Transfers.WindowOpen
			te.ProcessedWeeks = snap.Transfers.ProcessedWeeks
		} else {
			// v3 and older encoded a closed summer market as Week 13. Migrate
			// that sentinel to a closed Week 12 rather than re-exposing it.
			if te.IsOffSeason {
				te.WindowType = transfers.WindowSummer
				te.WindowOpen = te.CurrentWeek >= 1 && te.CurrentWeek <= transfers.TransferWindowWeeks
				if te.CurrentWeek > transfers.TransferWindowWeeks {
					te.CurrentWeek = transfers.TransferWindowWeeks
					te.WindowOpen = false
				}
				te.ProcessedWeeks = te.CurrentWeek - 1
				if !te.WindowOpen {
					te.ProcessedWeeks = transfers.TransferWindowWeeks
				}
			} else {
				te.WindowType = transfers.WindowClosed
				te.WindowOpen = false
				te.CurrentWeek = 0
				te.CurrentDay = 0
				te.ProcessedWeeks = 0
			}
		}
		// v4 and older used Week 1 as the idle in-season sentinel. Normalize it
		// after decoding so Week 1 always means that a real window has begun.
		if snap.Version < 5 && te.WindowType == transfers.WindowClosed && !te.IsOffSeason {
			te.CurrentWeek = 0
			te.CurrentDay = 0
			te.ProcessedWeeks = 0
			te.WindowOpen = false
		}
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
		// Authoritative club finances win over any stale snapshot mirror:
		// re-sync after applying so hand-edited saves cannot desync the AI.
		te.SyncAllManagerBudgets()
		te.ScriptedSwapDone = snap.Transfers.ScriptedSwapDone
		restoreFreeAgents(tm, te, snap, existingPlayers)
	}

	// 8. Stretch short legacy calendars, then re-pair mentors on the restored squads.
	// Older careers may have stored Hernando in the removed snake state.
	for _, club := range tm.ClubsList {
		for _, player := range club.Squad {
			if player != nil && player.PlayerID == "WK_Earl_Josh_Hernando" && player.Personality == "snake" {
				player.Personality = "dedicated_pro"
				player.TransferRequested = false
				if player.Loyalty < 60 {
					player.Loyalty = 60
				}
			}
		}
	}
	_ = tm.AdoptLongSeason()
	tournament.PairSeniorMentors(tm.ClubsList, ge)
	// Pre-Nations-Cup careers have no nested competition state. Build it from
	// restored squads so continuing a save gains the new competition.
	tm.EnsureNationalTeams()
	return nil
}

// MaybeWriteMigratedCareer atomically replaces destPath with the current
// in-memory career when the loaded snapshot was an older supported version.
// Validation failures leave the original save untouched.
func MaybeWriteMigratedCareer(
	tm *tournament.TournamentManager,
	ge *growth.GrowthEngine,
	te *transfers.TransferEngine,
	destPath string,
	loadedVersion int,
) (bool, error) {
	if loadedVersion >= SaveVersion {
		return false, nil
	}
	if tm != nil {
		if err := tm.ValidateWorldState(); err != nil {
			return false, fmt.Errorf("migrated career failed world validation: %w", err)
		}
	}
	snap := BuildSnapshot(tm, ge, te)
	if err := ValidateCareerSnapshot(snap); err != nil {
		return false, fmt.Errorf("migrated career failed snapshot validation: %w", err)
	}
	if _, err := WriteSnapshot(snap, destPath); err != nil {
		return false, err
	}
	return true, nil
}

func restoreFreeAgents(
	tm *tournament.TournamentManager,
	te *transfers.TransferEngine,
	snap *CareerSnapshot,
	existingPlayers map[string]*models.Player,
) {
	if te == nil {
		return
	}
	te.FreeAgents = nil
	for _, saved := range snap.Transfers.FreeAgents {
		if saved == nil || saved.PlayerID == "" {
			continue
		}
		live := existingPlayers[saved.PlayerID]
		if live == nil {
			live = saved
			existingPlayers[saved.PlayerID] = live
		} else {
			updatePlayerFromSaved(live, saved)
			if live.ClubID != "" {
				removePlayerFromClub(tm.Clubs[live.ClubID], live)
			}
		}
		live.MarkFreeAgent(saved.PreviousClubID)
		if saved.PreviousClubID != "" {
			live.PreviousClubID = saved.PreviousClubID
		}
		te.FreeAgents = append(te.FreeAgents, live)
	}
}

// DeleteCareer deletes the snapshot manifest, its club sidecars, and the
// universe seed sidecar, if any. Removing the seed with the career keeps a
// deleted universe from pinning future fresh boots to a stale seed.
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
	if err := os.Remove(UniverseSeedPath(path)); err != nil && !os.IsNotExist(err) {
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

func wireWorldFixtureClubs(tm *tournament.TournamentManager) {
	if tm == nil || tm.World == nil {
		return
	}
	for i := range tm.World.Fixtures {
		fixture := &tm.World.Fixtures[i]
		fixture.Home = tm.Clubs[fixture.HomeID]
		fixture.Away = tm.Clubs[fixture.AwayID]
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
	// Injury history (SaveVersion 12) must round-trip; deep-copy so the live
	// squad never aliases the snapshot.
	if src.InjuryHistory != nil {
		dest.InjuryHistory = append([]medical.Record(nil), src.InjuryHistory...)
	} else {
		dest.InjuryHistory = nil
	}
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
	// Loan state must round-trip: without it, post-load ReturnLoans and buy
	// clauses silently no-op (the flags live on the player, not the clubs).
	dest.OnLoan = src.OnLoan
	dest.ParentClubID = src.ParentClubID
	dest.LoanBuyClauseEUR = src.LoanBuyClauseEUR
	// Per-competition minutes feed the playing-time development curve and the
	// player sheet; deep-copy so the live squad never aliases the snapshot.
	if src.CompetitionStats != nil {
		restored := make(map[string]*models.CompetitionSeasonStats, len(src.CompetitionStats))
		for k, row := range src.CompetitionStats {
			if row == nil {
				continue
			}
			cp := *row
			restored[k] = &cp
		}
		dest.CompetitionStats = restored
	} else {
		dest.CompetitionStats = nil
	}
	if src.RecentRatings != nil {
		dest.RecentRatings = append([]float64(nil), src.RecentRatings...)
	} else {
		dest.RecentRatings = nil
	}
	if src.OriginalClubID != "" {
		dest.OriginalClubID = src.OriginalClubID
	}
	if dest.OriginalClubID == "" && dest.ClubID != "" {
		dest.OriginalClubID = dest.ClubID
	}
	if src.PlayerSource != "" {
		dest.PlayerSource = src.PlayerSource
	}
	// Static identity round-trips when present (old saves may omit it).
	if src.FullName != "" {
		dest.FullName = src.FullName
	}
	if src.Position != "" {
		dest.Position = src.Position
	}
	if src.SquadRole != "" {
		dest.SquadRole = src.SquadRole
	}
	// Dynamics drift mid-season; restore them when present, mirroring the
	// club-level `> 0` guards (a zero here means "unset" in hand-built and
	// ancient saves, and live defaults already apply). Squad roles are
	// static within a season, so that copy is a no-op for honest saves.
	if src.Morale > 0 {
		dest.Morale = src.Morale
	}
	if src.Fitness > 0 {
		dest.Fitness = src.Fitness
	}
	if src.Sharpness > 0 {
		dest.Sharpness = src.Sharpness
	}
	dest.TransferRequested = src.TransferRequested
	dest.IsCaptain = src.IsCaptain
	dest.IsViceCaptain = src.IsViceCaptain
	if src.Leadership > 0 {
		dest.Leadership = src.Leadership
	}
	dest.Homegrown = src.Homegrown
	dest.AssociationTrained = src.AssociationTrained
	dest.RegisteredEurope = src.RegisteredEurope
	dest.CleanSheets = src.CleanSheets
	dest.CareerCleanSheets = src.CareerCleanSheets
	if src.Versatility > 0 {
		dest.Versatility = src.Versatility
	}
	dest.PromiseKind = src.PromiseKind
	dest.PromiseSeason = src.PromiseSeason
	dest.PromiseMatchweek = src.PromiseMatchweek
	dest.UniverseWonderkid = src.UniverseWonderkid
	if src.Season != "" {
		dest.Season = src.Season
	}
	if src.Category != "" {
		dest.Category = src.Category
	}
	if src.RegistrationStatus != "" {
		dest.RegistrationStatus = src.RegistrationStatus
	}
	if src.PreviousClubID != "" {
		dest.PreviousClubID = src.PreviousClubID
	}
	if src.SeasonHistory != nil {
		dest.SeasonHistory = append([]models.PlayerSeasonRecord(nil), src.SeasonHistory...)
	}
	if src.TransferHistory != nil {
		dest.TransferHistory = append([]models.PlayerMoveRecord(nil), src.TransferHistory...)
	}
	if src.ContractHistory != nil {
		dest.ContractHistory = append([]models.PlayerContractEvent(nil), src.ContractHistory...)
	}
	if src.AwardsHistory != nil {
		dest.AwardsHistory = append([]models.PlayerHonourRecord(nil), src.AwardsHistory...)
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
