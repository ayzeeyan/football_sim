package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"football_sim/pkg/persistence"
)

// Save-slot archives (F9). The active career keeps living at the resolved
// save path; slots are self-contained named archives the viewer manages:
// each slot directory holds a monolithic career snapshot plus the universe
// seed, so export/import round-trips a whole universe in one file.

var slotIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

type slotMeta struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Season    string `json:"season"`
	Matchweek int    `json:"matchweek"`
	SizeBytes int64  `json:"size_bytes"`
}

type slotIndex struct {
	NextID int        `json:"next_id"`
	Slots  []slotMeta `json:"slots"`
}

func (s *Server) slotsDir() string {
	return filepath.Join(filepath.Dir(s.savePath), "slots")
}

func (s *Server) slotIndexPath() string {
	return filepath.Join(s.slotsDir(), "index.json")
}

func (s *Server) slotDir(slotID string) string {
	return filepath.Join(s.slotsDir(), slotID)
}

func (s *Server) slotSnapshotPath(slotID string) string {
	return filepath.Join(s.slotDir(slotID), "career.json")
}

func (s *Server) slotSeedPath(slotID string) string {
	return filepath.Join(s.slotDir(slotID), "seed.txt")
}

// loadSlotIndex reads the index, returning an empty one when absent.
func (s *Server) loadSlotIndex() (*slotIndex, error) {
	data, err := os.ReadFile(s.slotIndexPath())
	if err != nil {
		if os.IsNotExist(err) {
			return &slotIndex{NextID: 1, Slots: []slotMeta{}}, nil
		}
		return nil, err
	}
	var idx slotIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("slot index is corrupt: %w", err)
	}
	if idx.Slots == nil {
		idx.Slots = []slotMeta{}
	}
	if idx.NextID < 1 {
		idx.NextID = 1
	}
	return &idx, nil
}

func (s *Server) writeSlotIndex(idx *slotIndex) error {
	if err := os.MkdirAll(s.slotsDir(), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.slotIndexPath(), data, 0644)
}

func (s *Server) findSlot(idx *slotIndex, slotID string) (*slotMeta, int) {
	for i := range idx.Slots {
		if idx.Slots[i].ID == slotID {
			return &idx.Slots[i], i
		}
	}
	return nil, -1
}

// writeSlotSnapshot stores a monolithic snapshot plus the universe seed in
// the slot directory. Caller must hold s.saveMu.
func (s *Server) writeSlotSnapshot(slotID string, snapshot []byte, seed int64) error {
	dir := s.slotDir(slotID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(s.slotSnapshotPath(slotID), snapshot, 0644); err != nil {
		return err
	}
	return os.WriteFile(s.slotSeedPath(slotID), []byte(fmt.Sprintf("%d\n", seed)), 0644)
}

// currentUniverseSeed returns the active career's persisted seed, generating
// one only when the sidecar is missing.
func (s *Server) currentUniverseSeed() (int64, error) {
	return persistence.LoadOrCreateUniverseSeed(s.savePath)
}

func slotNameFromRequest(r *http.Request, fallback string) string {
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		name = strings.TrimSpace(r.URL.Query().Get("name"))
	}
	if name == "" {
		name = fallback
	}
	if len(name) > 64 {
		name = name[:64]
	}
	return name
}

func (s *Server) handleListSlots(w http.ResponseWriter, r *http.Request) {
	s.saveMu.Lock()
	idx, err := s.loadSlotIndex()
	s.saveMu.Unlock()
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	sort.SliceStable(idx.Slots, func(i, j int) bool {
		if idx.Slots[i].UpdatedAt != idx.Slots[j].UpdatedAt {
			return idx.Slots[i].UpdatedAt > idx.Slots[j].UpdatedAt
		}
		return idx.Slots[i].ID < idx.Slots[j].ID
	})
	writeJSON(w, map[string]interface{}{"slots": idx.Slots})
}

func (s *Server) handleCreateSlot(w http.ResponseWriter, r *http.Request) {
	name := slotNameFromRequest(r, "")
	if name == "" {
		writeErrorJSON(w, http.StatusBadRequest, "A slot name is required")
		return
	}

	// Snapshot the live world, then archive it outside the world lock.
	s.worldMu.Lock()
	snapshot, _ := s.takeCareerSnapshotLocked()
	season := s.TournamentManager.SeasonName
	matchweek := s.TournamentManager.CurrentMatchweek
	s.worldMu.Unlock()
	if len(snapshot) == 0 {
		writeErrorJSON(w, http.StatusInternalServerError, "Could not encode the current career")
		return
	}
	seed, err := s.currentUniverseSeed()
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	idx, err := s.loadSlotIndex()
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slotID := fmt.Sprintf("slot-%d", idx.NextID)
	idx.NextID++
	if err := s.writeSlotSnapshot(slotID, snapshot, seed); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	meta := slotMeta{
		ID: slotID, Name: name,
		CreatedAt: time.Now().UTC().Format(time.RFC3339), UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Season: season, Matchweek: matchweek,
		SizeBytes: int64(len(snapshot)),
	}
	idx.Slots = append(idx.Slots, meta)
	if err := s.writeSlotIndex(idx); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, map[string]interface{}{"status": "success", "slot": meta})
}

func (s *Server) handleRenameSlot(w http.ResponseWriter, r *http.Request) {
	slotID := r.PathValue("slot_id")
	if !slotIDPattern.MatchString(slotID) {
		writeErrorJSON(w, http.StatusBadRequest, "Invalid slot id")
		return
	}
	name := slotNameFromRequest(r, "")
	if name == "" {
		writeErrorJSON(w, http.StatusBadRequest, "A slot name is required")
		return
	}
	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	idx, err := s.loadSlotIndex()
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	meta, i := s.findSlot(idx, slotID)
	if meta == nil {
		writeErrorJSON(w, http.StatusNotFound, "Slot not found")
		return
	}
	idx.Slots[i].Name = name
	idx.Slots[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := s.writeSlotIndex(idx); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, map[string]interface{}{"status": "success", "slot": idx.Slots[i]})
}

func (s *Server) handleDuplicateSlot(w http.ResponseWriter, r *http.Request) {
	slotID := r.PathValue("slot_id")
	if !slotIDPattern.MatchString(slotID) {
		writeErrorJSON(w, http.StatusBadRequest, "Invalid slot id")
		return
	}
	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	idx, err := s.loadSlotIndex()
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	meta, _ := s.findSlot(idx, slotID)
	if meta == nil {
		writeErrorJSON(w, http.StatusNotFound, "Slot not found")
		return
	}
	snapshot, err := os.ReadFile(s.slotSnapshotPath(slotID))
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "Slot archive is missing its snapshot")
		return
	}
	seed := int64(0)
	if raw, err := os.ReadFile(s.slotSeedPath(slotID)); err == nil {
		fmt.Sscanf(strings.TrimSpace(string(raw)), "%d", &seed)
	}
	newID := fmt.Sprintf("slot-%d", idx.NextID)
	idx.NextID++
	if err := s.writeSlotSnapshot(newID, snapshot, seed); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	name := slotNameFromRequest(r, meta.Name+" copy")
	dup := slotMeta{
		ID: newID, Name: name,
		CreatedAt: time.Now().UTC().Format(time.RFC3339), UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Season: meta.Season, Matchweek: meta.Matchweek,
		SizeBytes: meta.SizeBytes,
	}
	idx.Slots = append(idx.Slots, dup)
	if err := s.writeSlotIndex(idx); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, map[string]interface{}{"status": "success", "slot": dup})
}

func (s *Server) handleDeleteSlot(w http.ResponseWriter, r *http.Request) {
	slotID := r.PathValue("slot_id")
	if !slotIDPattern.MatchString(slotID) {
		writeErrorJSON(w, http.StatusBadRequest, "Invalid slot id")
		return
	}
	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	idx, err := s.loadSlotIndex()
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	_, i := s.findSlot(idx, slotID)
	if i < 0 {
		writeErrorJSON(w, http.StatusNotFound, "Slot not found")
		return
	}
	if err := os.RemoveAll(s.slotDir(slotID)); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	idx.Slots = append(idx.Slots[:i], idx.Slots[i+1:]...)
	if err := s.writeSlotIndex(idx); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, map[string]interface{}{"status": "success", "deleted": slotID})
}

func (s *Server) handleExportSlot(w http.ResponseWriter, r *http.Request) {
	slotID := r.PathValue("slot_id")
	if !slotIDPattern.MatchString(slotID) {
		writeErrorJSON(w, http.StatusBadRequest, "Invalid slot id")
		return
	}
	s.saveMu.Lock()
	idx, err := s.loadSlotIndex()
	s.saveMu.Unlock()
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	meta, _ := s.findSlot(idx, slotID)
	if meta == nil {
		writeErrorJSON(w, http.StatusNotFound, "Slot not found")
		return
	}
	file, err := os.Open(s.slotSnapshotPath(slotID))
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "Slot archive is missing its snapshot")
		return
	}
	defer file.Close()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", slotID+"-career.json"))
	if _, err := io.Copy(w, file); err != nil {
		// Headers are already sent; nothing else to do.
		return
	}
}

func (s *Server) handleImportSlot(w http.ResponseWriter, r *http.Request) {
	// Monolithic snapshots are large; cap the upload well above the
	// biggest realistic save.
	body, err := io.ReadAll(io.LimitReader(r.Body, 256<<20))
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "Could not read the uploaded snapshot")
		return
	}
	var snap persistence.CareerSnapshot
	if err := json.Unmarshal(body, &snap); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "The uploaded file is not a career snapshot")
		return
	}
	if err := persistence.ValidateCareerSnapshot(&snap); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	name := slotNameFromRequest(r, "")
	if name == "" {
		name = snap.SeasonName
	}

	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	idx, err := s.loadSlotIndex()
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slotID := fmt.Sprintf("slot-%d", idx.NextID)
	idx.NextID++
	seed, err := persistence.NewUniverseSeed()
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := s.writeSlotSnapshot(slotID, body, seed); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	meta := slotMeta{
		ID: slotID, Name: name,
		CreatedAt: time.Now().UTC().Format(time.RFC3339), UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Season: snap.SeasonName, Matchweek: snap.CurrentMatchweek,
		SizeBytes: int64(len(body)),
	}
	idx.Slots = append(idx.Slots, meta)
	if err := s.writeSlotIndex(idx); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, map[string]interface{}{"status": "success", "slot": meta})
}
