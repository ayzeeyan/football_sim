package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"time"

	"football_sim/pkg/datamanager"
	"football_sim/pkg/models"
	"football_sim/pkg/persistence"
	"football_sim/pkg/tournament"
	"football_sim/pkg/transfers"
)

// Career lifecycle: seasons, transfer market and inbox.
func (s *Server) handleResetSeason(w http.ResponseWriter, r *http.Request) {
	s.worldMu.Lock()
	held := true
	defer func() {
		if held {
			s.worldMu.Unlock()
		}
	}()

	res := s.TournamentManager.FinalizeSeasonTransition()
	if status, ok := res["status"].(string); !ok || status != "success" {
		message := "Season transition could not be finalized."
		if detail, ok := res["message"].(string); ok && detail != "" {
			message = detail
		}
		s.worldMu.Unlock()
		held = false
		writeErrorJSON(w, http.StatusBadRequest, message)
		return
	}
	s.clearLiveFixtureSelection()
	s.lastCommittedLiveInstance = -1
	snap, gen := s.takeCareerSnapshotLocked()
	s.worldMu.Unlock()
	held = false
	s.commitCareerSnapshot(snap, gen)
	writeJSON(w, res)
}

func (s *Server) handleRestartSeason(w http.ResponseWriter, r *http.Request) {
	s.worldMu.Lock()
	held := true
	defer func() {
		if held {
			s.worldMu.Unlock()
		}
	}()

	s.clearLiveFixtureSelection()
	s.lastCommittedLiveInstance = -1
	res := s.TournamentManager.RestartCurrentSeason()
	snap, gen := s.takeCareerSnapshotLocked()
	s.worldMu.Unlock()
	held = false
	s.commitCareerSnapshot(snap, gen)
	writeJSON(w, res)
}

func (s *Server) handleGetDefaultHomes(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	defer s.worldMu.RUnlock()
	// New careers pre-fill the last draw so a second career starts where the
	// last one did. The client can still reroll via preview-shuffle.
	if len(s.TournamentManager.ProdigyHomes) > 0 {
		payload := s.prodigyDrawPayload(s.TournamentManager.ProdigyHomes)
		payload["shuffle"] = s.TournamentManager.LastCareerShuffle
		payload["from_last"] = true
		writeJSON(w, payload)
		return
	}
	writeJSON(w, s.prodigyDrawPayload(datamanager.DefaultProdigyHomes()))
}

func (s *Server) handlePreviewShuffle(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	defer s.worldMu.RUnlock()
	// Previews are explicitly non-career randomness: each reroll varies.
	// Career draws always use the universe stream (see handleNewCareer).
	writeJSON(w, s.prodigyDrawPayload(datamanager.ShuffleProdigyHomes(rand.New(rand.NewSource(time.Now().UnixNano())))))
}

func (s *Server) handleNewCareer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Shuffle bool              `json:"shuffle"`
		Homes   map[string]string `json:"homes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeErrorJSON(w, http.StatusBadRequest, "Invalid new-career request")
		return
	}

	s.worldMu.Lock()
	held := true
	defer func() {
		if held {
			s.worldMu.Unlock()
		}
	}()

	// A new career is a new universe, but the previous save is only retired
	// AFTER the fresh universe boots successfully: a failed boot must never
	// destroy the existing career. The fresh seed is generated in memory
	// (NewUniverseSeed) and persisted (WriteUniverseSeed) only on success, so
	// named slot archives and the active save can never be clobbered by a
	// half-finished regeneration.
	freshSeed, err := persistence.NewUniverseSeed()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	universe := tournament.NewSubsystemRNG(freshSeed)
	homes := resolveNewCareerHomes(req.Shuffle, req.Homes, universe.New("prodigy_draw"))
	if err := s.bootFreshCareer(homes, req.Shuffle, freshSeed); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := persistence.DeleteCareer(s.savePath); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "Could not remove the previous career save")
		return
	}
	if err := persistence.WriteUniverseSeed(s.savePath, freshSeed); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	payload := map[string]interface{}{
		"status":            "success",
		"message":           "New career. 2026-27 matchweek 1. All-time stats, growth and values reset.",
		"season_name":       s.TournamentManager.SeasonName,
		"current_matchweek": s.TournamentManager.CurrentMatchweek,
		"max_matchweeks":    s.TournamentManager.MaxMatchweeks,
		"homes":             copyStringMap(s.DataManager.ProdigyHomes),
		"draw":              s.DataManager.DescribeProdigyDraw(s.DataManager.ProdigyHomes),
	}
	snap, gen := s.takeCareerSnapshotLocked()
	s.worldMu.Unlock()
	held = false
	s.commitCareerSnapshot(snap, gen)
	writeJSON(w, payload)
}

func (s *Server) careerWindowOpen() bool {
	return s.TransferEngine != nil && s.TransferEngine.IsWindowOpen()
}

func careerWindowName(open bool, week int) string {
	if open {
		if week > 0 {
			return fmt.Sprintf("Summer Window (Week %d of 12)", week)
		}
		return "Summer Window (Open)"
	}
	return "Window Closed (Opens at season end)"
}

func (s *Server) serializeNegotiation(neg *transfers.TransferNegotiation) map[string]interface{} {
	if neg == nil {
		return nil
	}
	return map[string]interface{}{
		"negotiation_id":         neg.NegotiationID,
		"player":                 s.serializePlayer(neg.Player),
		"buyer":                  s.serializeClub(neg.Buyer),
		"seller":                 s.serializeClub(neg.Seller),
		"current_bid":            neg.CurrentBid,
		"formatted_bid":          models.FormatCurrency(neg.CurrentBid),
		"asking_price":           neg.AskingPrice,
		"formatted_asking_price": models.FormatCurrency(neg.AskingPrice),
		"history":                neg.History,
		"stage_index":            neg.StageIndex,
		"stage_name":             neg.StageName,
		"progress_pct":           neg.ProgressPct,
		"is_wonderkid":           neg.IsWonderkid,
		"is_hijacked":            neg.IsHijacked,
		"original_buyer":         s.serializeClub(neg.OriginalBuyer),
	}
}

func (s *Server) transfersPayload() map[string]interface{} {
	open := s.careerWindowOpen()
	negs := make([]map[string]interface{}, 0, len(s.TransferEngine.ActiveNegotiations))
	for _, neg := range s.TransferEngine.ActiveNegotiations {
		if serialized := s.serializeNegotiation(neg); serialized != nil {
			negs = append(negs, serialized)
		}
	}

	feed := s.TransferEngine.TransferFeed
	if feed == nil {
		feed = []transfers.TransferFeedItem{}
	}
	completed := s.TransferEngine.CompletedTransfers
	if completed == nil {
		completed = []transfers.CompletedTransfer{}
	}

	var expiring []map[string]interface{}
	type expRow struct {
		player *models.Player
		club   *models.Club
	}
	var rows []expRow
	for _, club := range s.TournamentManager.ClubsList {
		for _, p := range club.Squad {
			if p != nil && p.ContractYears <= 1 {
				rows = append(rows, expRow{player: p, club: club})
			}
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].player.OVR != rows[j].player.OVR {
			return rows[i].player.OVR > rows[j].player.OVR
		}
		return rows[i].player.Loyalty > rows[j].player.Loyalty
	})
	if len(rows) > 8 {
		rows = rows[:8]
	}
	for _, row := range rows {
		expiring = append(expiring, map[string]interface{}{
			"player_id":      row.player.PlayerID,
			"full_name":      row.player.FullName,
			"position":       row.player.Position,
			"ovr":            row.player.OVR,
			"club_name":      row.club.ClubName,
			"club_short":     row.club.ShortName,
			"formatted_wage": models.FormatWage(row.player.WageEUR),
			"loyalty":        row.player.Loyalty,
			"is_wonderkid":   row.player.UniverseWonderkid,
		})
	}
	if expiring == nil {
		expiring = []map[string]interface{}{}
	}

	var warchests []map[string]interface{}
	for _, club := range s.TournamentManager.ClubsList {
		if club == nil {
			continue
		}
		mgr := s.TournamentManager.Managers[club.ClubID]
		if mgr == nil && s.TransferEngine != nil {
			mgr = s.TransferEngine.Managers[club.ClubID]
		}
		if mgr == nil {
			continue
		}
		budget := club.Finances.TransferBudget
		warchests = append(warchests, map[string]interface{}{
			"club_name":        club.ClubName,
			"club_short":       club.ShortName,
			"manager_name":     mgr.Name,
			"tactic":           mgr.Tactic(),
			"focus":            mgr.FocusLabel(),
			"budget_eur":       budget,
			"formatted_budget": models.FormatCurrency(budget),
			"wage_bill_eur":    mgr.WageBill(club),
			"wage_cap_eur":     mgr.WageCap(club),
			"wage_bill":        club.WageBill(),
			"wage_cap":         club.WageCap(),
		})
	}
	sort.SliceStable(warchests, func(i, j int) bool {
		bi, _ := warchests[i]["budget_eur"].(int64)
		bj, _ := warchests[j]["budget_eur"].(int64)
		return bi > bj
	})
	if warchests == nil {
		warchests = []map[string]interface{}{}
	}

	maxWeeks := 0
	windowName := "Window Closed (Opens at season end)"
	windowType := transfers.WindowClosed
	if s.TransferEngine != nil {
		maxWeeks = s.TransferEngine.WindowWeeks()
		windowName = s.TransferEngine.GetWindowName()
		windowType = s.TransferEngine.WindowType
	}
	if maxWeeks == 0 {
		maxWeeks = transfers.TransferWindowWeeks
	}

	return map[string]interface{}{
		"window_name":         windowName,
		"is_window_open":      open,
		"window_type":         windowType,
		"season_phase":        s.TournamentManager.SeasonPhase,
		"window_day":          s.TransferEngine.CurrentDay,
		"window_week":         s.TransferEngine.CurrentWeek,
		"max_window_weeks":    maxWeeks,
		"active_negotiations": negs,
		"transfer_feed":       feed,
		"completed_transfers": completed,
		"expiring_contracts":  expiring,
		"warchests":           warchests,
		"deadline_day":        open && maxWeeks > 0 && s.TransferEngine != nil && s.TransferEngine.CurrentWeek >= maxWeeks-1,
	}
}

func (s *Server) handleGetTransfers(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	payload := s.transfersPayload()
	s.worldMu.RUnlock()
	writeJSON(w, payload)
}

func (s *Server) handleTransferBid(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlayerID  string `json:"player_id"`
		BuyerID   string `json:"buyer_id"`
		SellerID  string `json:"seller_id"`
		BidAmount int64  `json:"bid_amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	s.worldMu.Lock()
	held := true
	defer func() {
		if held {
			s.worldMu.Unlock()
		}
	}()

	if !s.careerWindowOpen() {
		writeErrorJSON(w, http.StatusBadRequest, "The window opens when the season ends.")
		return
	}

	neg := s.TransferEngine.TriggerSpecificBid(req.BuyerID, req.SellerID, req.PlayerID)
	if neg == nil {
		writeErrorJSON(w, http.StatusBadRequest, "Could not create negotiation")
		return
	}
	payload := s.serializeNegotiation(neg)
	snap, gen := s.takeCareerSnapshotLocked()
	s.worldMu.Unlock()
	held = false
	s.commitCareerSnapshot(snap, gen)
	writeJSON(w, payload)
}

func (s *Server) handleTransferAdvance(w http.ResponseWriter, r *http.Request) {
	s.worldMu.Lock()
	held := true
	defer func() {
		if held {
			s.worldMu.Unlock()
		}
	}()

	openedNow := false
	if s.TournamentManager != nil && s.TournamentManager.SeasonPhase == "transfer_window" && s.TransferEngine != nil {
		if !s.TransferEngine.IsOffSeason {
			if !s.TournamentManager.ApplyCompletedSeasonReputation() && s.TournamentManager.ReputationAppliedSeason != s.TournamentManager.SeasonName {
				writeErrorJSON(w, http.StatusConflict, "The completed season is not ready for the transfer window; finish all league and cup results first.")
				return
			}
			s.TransferEngine.BeginOffSeasonWindow()
			openedNow = true
		}
	}
	if !s.careerWindowOpen() {
		writeErrorJSON(w, http.StatusBadRequest, "The window opens when the season ends.")
		return
	}

	if !openedNow {
		s.TransferEngine.AdvanceOpenWindow()
	}
	// Mentor-leaves drama before re-pairing clears the old MentorIDs.
	// Idempotent per season: re-paired kids no longer match, and flags guard
	// the rest, so replaying completed history is safe.
	for _, done := range s.TransferEngine.CompletedTransfers {
		s.TournamentManager.NoteMentorDeparture(done.PlayerID, done.PlayerName, done.SellerID, done.BuyerName, s.TournamentManager.CurrentMatchweek)
	}
	tournament.PairSeniorMentors(s.TournamentManager.ClubsList, s.GrowthEngine)
	payload := s.transfersPayload()
	snap, gen := s.takeCareerSnapshotLocked()
	s.worldMu.Unlock()
	held = false
	s.commitCareerSnapshot(snap, gen)
	writeJSON(w, payload)
}

func (s *Server) handleGetTransferRecords(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	records := s.TransferEngine.GetTransferRecords()
	s.worldMu.RUnlock()
	writeJSON(w, records)
}

func (s *Server) handleGetInbox(w http.ResponseWriter, r *http.Request) {
	limit := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	s.worldMu.RLock()
	unread := 0
	for _, item := range s.TournamentManager.Inbox {
		if item.Unread {
			unread++
		}
	}
	// Copy the backing array: MarkInboxRead mutates items in place.
	// Newest items are prepended, so a limit keeps the head of the list.
	items := append([]tournament.InboxItem{}, s.TournamentManager.Inbox...)
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	payload := map[string]interface{}{
		"unread":            unread,
		"items":             items,
		"season_name":       s.TournamentManager.SeasonName,
		"current_matchweek": s.TournamentManager.CurrentMatchweek,
	}
	s.worldMu.RUnlock()
	writeJSON(w, payload)
}

func (s *Server) handleMarkInboxRead(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID     string `json:"id"`
		ItemID string `json:"item_id"`
		All    bool   `json:"all"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "Invalid inbox read request")
		return
	}
	s.worldMu.Lock()
	target := req.ID
	if target == "" {
		target = req.ItemID
	}
	marked := 0
	for i := range s.TournamentManager.Inbox {
		if req.All || (target != "" && s.TournamentManager.Inbox[i].ID == target) {
			if s.TournamentManager.Inbox[i].Unread {
				s.TournamentManager.Inbox[i].Unread = false
				marked++
			}
		}
	}
	unread := 0
	for _, item := range s.TournamentManager.Inbox {
		if item.Unread {
			unread++
		}
	}
	var snap []byte
	var gen uint64
	if marked > 0 {
		snap, gen = s.takeCareerSnapshotLocked()
	}
	s.worldMu.Unlock()
	s.commitCareerSnapshot(snap, gen)
	writeJSON(w, map[string]interface{}{
		"status": "success",
		"unread": unread,
		"marked": marked,
	})
}

func (s *Server) handleInboxReply(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ItemID   string `json:"item_id"`
		ChoiceID string `json:"choice_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "Invalid inbox reply")
		return
	}
	s.worldMu.Lock()
	payload := s.TournamentManager.ReplyInboxUnlocked(req.ItemID, req.ChoiceID)
	var snap []byte
	var gen uint64
	if payload["status"] == "success" {
		snap, gen = s.takeCareerSnapshotLocked()
	}
	s.worldMu.Unlock()
	if payload["status"] != "success" {
		msg, _ := payload["message"].(string)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	s.commitCareerSnapshot(snap, gen)
	writeJSON(w, payload)
}
