package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	urlpath "path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"football_sim/pkg/datamanager"
	"football_sim/pkg/growth"
	"football_sim/pkg/managers"
	"football_sim/pkg/matchengine"
	"football_sim/pkg/models"
	"football_sim/pkg/persistence"
	"football_sim/pkg/tournament"
	"football_sim/pkg/transfers"

	"github.com/gorilla/websocket"
)

// Server coordinates HTTP REST endpoints, WebSocket live simulation streaming,
// and static single-page application delivery.
//
// Locking: worldMu and wsMu are never held together. worldMu guards career
// state and the live engine. wsMu guards only the WebSocket client set.
// saveMu serializes installing a snapshot on disk and is never held with
// worldMu. Snapshot under worldMu, persist after release.
type Server struct {
	DataManager       *datamanager.DataManager
	GrowthEngine      *growth.GrowthEngine
	TournamentManager *tournament.TournamentManager
	TransferEngine    *transfers.TransferEngine
	LiveMatchEngine   *matchengine.LiveMatchEngine

	mux                       *http.ServeMux
	upgrader                  websocket.Upgrader
	wsClients                 map[*websocket.Conn]bool
	wsMu                      sync.Mutex
	worldMu                   sync.RWMutex
	saveMu                    sync.Mutex
	saveGen                   atomic.Uint64
	savePath                  string
	staticDir                 string
	ctx                       context.Context
	cancel                    context.CancelFunc
	activeTick                bool
	lastCommittedLiveInstance int
	liveFixtureID             string
	wsWrite                   func(*websocket.Conn, interface{}) error
	// Delta tick protocol (F3): every field below is owned by worldMu.
	// wsForceFull requests the next broadcast as a full snapshot (kickoff,
	// goal/event, client join, resubscribe). Otherwise ticks ship deltas.
	wsTickSeq       uint64
	wsForceFull     bool
	wsLastHomeScore int
	wsLastAwayScore int
	wsLastEvents    int
}

var errFreshCareer = errors.New("could not rebuild the Super League from dataset.json")

// NewServer initializes and configures the HTTP & WebSocket engine.
func NewServer(
	dm *datamanager.DataManager,
	ge *growth.GrowthEngine,
	tm *tournament.TournamentManager,
	te *transfers.TransferEngine,
	savePath string,
	staticDir string,
) *Server {
	if savePath == "" {
		savePath = persistence.SavePath()
	}
	if staticDir == "" {
		// Detect frontend/dist relative to executable or root
		candidates := []string{
			filepath.Clean("frontend/dist"),
			filepath.Clean("../frontend/dist"),
			filepath.Clean("../../frontend/dist"),
		}
		for _, c := range candidates {
			if info, err := os.Stat(c); err == nil && info.IsDir() {
				staticDir = c
				break
			}
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Default live match matchup
	var homeClub, awayClub *models.Club
	var homeMgr, awayMgr *managers.ManagerProfile
	if len(tm.ClubsList) >= 2 {
		homeClub = tm.ClubsList[0]
		awayClub = tm.ClubsList[1]
		homeMgr = tm.Managers[homeClub.ClubID]
		awayMgr = tm.Managers[awayClub.ClubID]
	}

	liveEngine := matchengine.NewLiveMatchEngine(homeClub, awayClub, homeMgr, awayMgr, time.Now().UnixNano())

	if tm != nil && te != nil {
		tm.TransferEngine = te
	}

	s := &Server{
		DataManager:       dm,
		GrowthEngine:      ge,
		TournamentManager: tm,
		TransferEngine:    te,
		LiveMatchEngine:   liveEngine,
		mux:               http.NewServeMux(),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow development clients on Vite (5173), etc.
			},
		},
		wsClients:                 make(map[*websocket.Conn]bool),
		savePath:                  savePath,
		staticDir:                 staticDir,
		ctx:                       ctx,
		cancel:                    cancel,
		lastCommittedLiveInstance: -1,
		wsForceFull:               true,
	}

	s.setupRoutes()
	s.StartLiveTicker()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// CORS Middleware
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	s.mux.ServeHTTP(w, r)
}

func (s *Server) setupRoutes() {
	// WebSocket Real-time Match Simulation
	s.mux.HandleFunc("/ws/match", s.handleWebSocketMatch)

	// Health & System
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/stats", s.handleStats)

	// Clubs & Rosters
	s.mux.HandleFunc("GET /api/clubs", s.handleGetClubs)
	s.mux.HandleFunc("GET /api/clubs/{club_id}/squad", s.handleGetClubSquad)
	s.mux.HandleFunc("GET /api/clubs/{club_id}/xi", s.handleGetClubXI)
	s.mux.HandleFunc("GET /api/clubs/{club_id}/history", s.handleGetClubHistory)
	s.mux.HandleFunc("GET /api/h2h/{club_a}/{club_b}", s.handleGetH2H)
	s.mux.HandleFunc("GET /api/players/{player_id}", s.handleGetPlayerProfile)

	// Wonderkids & Growth
	s.mux.HandleFunc("GET /api/prodigies", s.handleGetProdigies)
	s.mux.HandleFunc("GET /api/prodigies/watch", s.handleGetProdigyWatch)
	s.mux.HandleFunc("GET /api/wonderkids", s.handleGetWonderkids)
	s.mux.HandleFunc("POST /api/prodigies/{player_id}/train", s.handleTrainProdigy)
	s.mux.HandleFunc("POST /api/prodigies/{player_id}/position-path", s.handleSetPositionPath)
	s.mux.HandleFunc("POST /api/prodigies/{player_id}/school-track", s.handleSetSchoolTrack)
	s.mux.HandleFunc("GET /api/growth/milestones", s.handleGetGrowthMilestones)
	s.mux.HandleFunc("GET /api/prodigies/{player_id}/timeline", s.handleGetProdigyTimeline)
	s.mux.HandleFunc("GET /api/training/status", s.handleGetTrainingStatus)
	s.mux.HandleFunc("GET /api/nxgn50", s.handleGetNXGN50)

	// Competitions & Super League Calendar
	s.mux.HandleFunc("GET /api/super-league", s.handleGetSuperLeague)
	s.mux.HandleFunc("GET /api/ucl", s.handleGetUCL)
	s.mux.HandleFunc("GET /api/super-cup", s.handleGetSuperCup)
	s.mux.HandleFunc("GET /api/ucl/fixtures", s.handleGetUCLFixtures)
	s.mux.HandleFunc("GET /api/calendar", s.handleGetCalendar)
	s.mux.HandleFunc("GET /api/fixtures", s.handleGetFixtures)
	s.mux.HandleFunc("GET /api/fixtures/{fixture_id}", s.handleGetFixture)
	s.mux.HandleFunc("POST /api/fixtures/{fixture_id}/simulate", s.handleSimulateFixture)
	s.mux.HandleFunc("POST /api/fixtures/simulate-remaining", s.handleSimulateRemaining)
	s.mux.HandleFunc("POST /api/sim/week", s.handleSimWeek)
	s.mux.HandleFunc("POST /api/sim/month", s.handleSimMonth)
	s.mux.HandleFunc("POST /api/sim/season", s.handleSimSeason)
	s.mux.HandleFunc("GET /api/scoring-race", s.handleGetScoringRace)
	s.mux.HandleFunc("GET /api/trophies", s.handleGetTrophies)
	s.mux.HandleFunc("GET /api/records", s.handleGetRecords)
	s.mux.HandleFunc("GET /api/season/awards", s.handleGetSeasonAwards)
	s.mux.HandleFunc("GET /api/season/awards/ceremony", s.handleGetAwardsCeremony)
	s.mux.HandleFunc("GET /api/season/history", s.handleGetSeasonHistory)
	s.mux.HandleFunc("GET /api/season/stats", s.handleGetSeasonStats)
	s.mux.HandleFunc("POST /api/season/reset", s.handleResetSeason)
	s.mux.HandleFunc("POST /api/season/restart", s.handleRestartSeason)

	// Career Management
	s.mux.HandleFunc("GET /api/career/default-homes", s.handleGetDefaultHomes)
	s.mux.HandleFunc("GET /api/career/preview-shuffle", s.handlePreviewShuffle)
	s.mux.HandleFunc("POST /api/career/new", s.handleNewCareer)
	s.mux.HandleFunc("GET /api/favourite", s.handleGetFavourite)
	s.mux.HandleFunc("POST /api/favourite", s.handleSetFavourite)
	s.mux.HandleFunc("GET /api/week/watch", s.handleWeekWatch)

	// Transfer Market
	s.mux.HandleFunc("GET /api/transfers", s.handleGetTransfers)
	s.mux.HandleFunc("POST /api/transfers/bid", s.handleTransferBid)
	s.mux.HandleFunc("POST /api/transfers/advance", s.handleTransferAdvance)
	s.mux.HandleFunc("GET /api/transfers/records", s.handleGetTransferRecords)

	// News Wire & Inbox
	s.mux.HandleFunc("GET /api/inbox", s.handleGetInbox)
	s.mux.HandleFunc("POST /api/inbox/read", s.handleMarkInboxRead)

	// Static SPA Fallback
	s.mux.HandleFunc("/", s.handleStaticSPA)
}

// StartLiveTicker launches the 60 FPS live match broadcast loop in the background.
func (s *Server) StartLiveTicker() {
	if s.activeTick {
		return
	}
	s.activeTick = true

	go func() {
		ticker := time.NewTicker(16 * time.Millisecond) // ~60 FPS
		defer ticker.Stop()

		for {
			select {
			case <-s.ctx.Done():
				return
			case <-ticker.C:
				if s.wsClientCount() == 0 || s.LiveMatchEngine == nil {
					continue
				}

				s.worldMu.Lock()
				s.LiveMatchEngine.Tick(0.016)
				var snap []byte
				var gen uint64
				if s.LiveMatchEngine.State == "FULL_TIME" && s.LiveMatchEngine.InstanceID != s.lastCommittedLiveInstance {
					if s.liveFixtureID != "" && s.LiveMatchEngine.HomeClub != nil && s.LiveMatchEngine.AwayClub != nil {
						result := s.TournamentManager.CommitLiveFixtureByID(s.liveFixtureID, s.LiveMatchEngine)
						if recorded, _ := result["recorded"].(bool); recorded {
							s.lastCommittedLiveInstance = s.LiveMatchEngine.InstanceID
							snap, gen = s.takeCareerSnapshotLocked()
						} else if terminal, _ := result["terminal"].(bool); terminal {
							// Stale, finished, or otherwise terminal selections have
							// been handled for this engine instance. Retryable errors
							// leave the instance unhandled so a later tick can retry.
							s.lastCommittedLiveInstance = s.LiveMatchEngine.InstanceID
						}
					}
				}
				payload := s.buildTickPayloadLocked()
				s.worldMu.Unlock()

				s.commitCareerSnapshot(snap, gen)
				s.broadcastMatchTick(payload)
			}
		}
	}()
}

// Stop shuts down the server background tasks and WebSocket clients.
func (s *Server) Stop() {
	s.cancel()
	s.wsMu.Lock()
	defer s.wsMu.Unlock()
	for conn := range s.wsClients {
		_ = conn.Close()
	}
	s.wsClients = make(map[*websocket.Conn]bool)
}

// -----------------------------------------------------------------------------
// WEBSOCKET HANDLERS & BROADCAST
// -----------------------------------------------------------------------------

func (s *Server) handleWebSocketMatch(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WebSocket] upgrade error: %v", err)
		return
	}

	s.wsMu.Lock()
	s.wsClients[conn] = true
	s.wsMu.Unlock()

	// A joining client has no snapshot: the next broadcast must be full so
	// it never paints from a delta. Never hold worldMu and wsMu together.
	s.worldMu.Lock()
	s.wsForceFull = true
	s.worldMu.Unlock()

	defer func() {
		s.wsMu.Lock()
		delete(s.wsClients, conn)
		s.wsMu.Unlock()
		_ = conn.Close()
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var cmd struct {
			Action string `json:"action"`
			HomeID string `json:"home_id"`
			AwayID string `json:"away_id"`
			Speed  int    `json:"speed"`
			Side   string `json:"side"`
			Stance string `json:"stance"`
			OutID  string `json:"out_id"`
			InID   string `json:"in_id"`
		}
		if err := json.Unmarshal(msg, &cmd); err != nil {
			continue
		}

		s.worldMu.Lock()
		switch cmd.Action {
		case "resubscribe":
			// Late joiners (or clients that dropped their snapshot) wait for
			// the next broadcast, which is forced to a full snapshot.
			s.wsForceFull = true
		case "set_clubs":
			// A new selection starts a new live session. Clear the previous
			// fixture before validating the requested clubs so an exhibition or
			// invalid selection cannot inherit an old fixture identity.
			s.liveFixtureID = ""
			s.LiveMatchEngine.ClearFixtureContext()
			home := s.TournamentManager.Clubs[cmd.HomeID]
			away := s.TournamentManager.Clubs[cmd.AwayID]
			if home != nil && away != nil {
				homeMgr := s.TournamentManager.Managers[cmd.HomeID]
				awayMgr := s.TournamentManager.Managers[cmd.AwayID]
				s.LiveMatchEngine.SetClubs(home, away, homeMgr, awayMgr)
				// First opened live match pins the favourite watch club when
				// none is set yet (persisted to the save on next snapshot).
				if s.TournamentManager.FavouriteClubID == "" {
					s.TournamentManager.FavouriteClubID = cmd.HomeID
				}
				if f := s.findLiveFixture(cmd.HomeID, cmd.AwayID); f != nil {
					s.liveFixtureID = f.FixtureID
					if real := s.TournamentManager.FindFixture(f.FixtureID); real != nil {
						f = real
					}
					highHeat := f.IsHighHeatDerby || f.DerbyHeat > 70
					s.LiveMatchEngine.SetFixtureContext(f.Competition, f.Matchweek, f.DerbyName != "", highHeat)
					s.LiveMatchEngine.SetFixtureWeather(s.TournamentManager.EnsureFixtureWeather(f))
					s.LiveMatchEngine.SetFixtureStage(f.Stage)
				}
			}
			s.wsForceFull = true
		case "kickoff":
			// Kickoff after full time resets the engine for another session;
			// do not keep reporting or committing the prior finished fixture.
			if s.LiveMatchEngine.State == "FULL_TIME" {
				s.liveFixtureID = ""
				s.LiveMatchEngine.ClearFixtureContext()
			}
			s.LiveMatchEngine.Kickoff()
			s.wsForceFull = true
		case "pause":
			s.LiveMatchEngine.TogglePause()
		case "halftime_resume":
			if s.LiveMatchEngine.ResumeHalfTime(cmd.Side, cmd.Stance, cmd.OutID, cmd.InID) {
				s.wsForceFull = true
			}
		case "set_speed":
			s.LiveMatchEngine.SetSpeed(cmd.Speed)
		case "seek_70":
			if s.LiveMatchEngine.SeekTo70() {
				s.wsForceFull = true
			}
		case "seek_chance":
			if s.LiveMatchEngine.SeekNextChance() {
				s.wsForceFull = true
			}
		case "reset":
			s.liveFixtureID = ""
			s.LiveMatchEngine.ClearFixtureContext()
			s.LiveMatchEngine.Reset()
			s.wsForceFull = true
		}
		s.worldMu.Unlock()
	}
}

// buildTickPayloadLocked decides between a full snapshot and a delta tick.
// Caller must hold worldMu. Full snapshots go out on kickoff, on any new
// match event (goal/sub/card/penalty), and when a client joins or
// resubscribes; every other tick ships coords + score + last event plus the
// ball/momentum/trail fields.
func (s *Server) buildTickPayloadLocked() map[string]interface{} {
	e := s.LiveMatchEngine
	if e == nil {
		return nil
	}
	s.wsTickSeq++
	full := s.wsForceFull ||
		e.HomeScore != s.wsLastHomeScore ||
		e.AwayScore != s.wsLastAwayScore ||
		len(e.Events) != s.wsLastEvents
	var payload map[string]interface{}
	if full {
		payload = s.buildMatchTickPayload()
	} else {
		payload = s.buildDeltaTickPayload()
	}
	s.wsForceFull = false
	s.wsLastHomeScore = e.HomeScore
	s.wsLastAwayScore = e.AwayScore
	s.wsLastEvents = len(e.Events)
	return payload
}

func (s *Server) buildMatchTickPayload() map[string]interface{} {
	e := s.LiveMatchEngine
	if e == nil {
		return nil
	}
	if s.wsTickSeq == 0 {
		s.wsTickSeq = 1
	}

	homeCoords := make([]map[string]interface{}, len(e.HomePlayers))
	for i, p := range e.HomePlayers {
		homeCoords[i] = map[string]interface{}{
			"x":        p.X,
			"y":        p.Y,
			"player":   p,
			"sent_off": e.Bookings[p.PlayerID] >= 2,
		}
	}

	awayCoords := make([]map[string]interface{}, len(e.AwayPlayers))
	for i, p := range e.AwayPlayers {
		awayCoords[i] = map[string]interface{}{
			"x":        p.X,
			"y":        p.Y,
			"player":   p,
			"sent_off": e.Bookings[p.PlayerID] >= 2,
		}
	}

	var leagueFixture map[string]interface{}
	if s.liveFixtureID != "" && s.TournamentManager != nil {
		if f := s.TournamentManager.FindFixture(s.liveFixtureID); f != nil {
			leagueFixture = map[string]interface{}{"id": f.FixtureID, "status": f.Status}
		}
	}

	return map[string]interface{}{
		"tick_type":            "full",
		"seq":                  s.wsTickSeq,
		"state":                e.State,
		"minute":               math.Round(e.CurrentMinute*10) / 10,
		"speed":                e.Speed,
		"phase":                e.Phase,
		"active_third":         e.ActiveThird,
		"home_score":           e.HomeScore,
		"away_score":           e.AwayScore,
		"home_shots":           e.HomeShots,
		"away_shots":           e.AwayShots,
		"home_shots_on_target": e.HomeShotsOn,
		"away_shots_on_target": e.AwayShotsOn,
		"home_corners":         e.HomeCorners,
		"away_corners":         e.AwayCorners,
		"home_possession_pct":  e.HomePossessionPct(),
		"away_possession_pct":  e.AwayPossessionPct(),
		"possession_momentum":  math.Round(e.PossessionMomentum*1000) / 1000,
		"ball": map[string]interface{}{
			"x":       e.BallPos.X,
			"y":       e.BallPos.Y,
			"height":  math.Round(e.BallHeight*100) / 100,
			"is_shot": e.BallIsShot,
		},
		"goal_banner":           e.Banner,
		"league_fixture":        leagueFixture,
		"pass_trail":            serializePassTrail(e.PassTrail),
		"home_coords":           homeCoords,
		"away_coords":           awayCoords,
		"commentary":            e.Commentary,
		"match_events":          e.Events,
		"home_tactical_stance":  e.HomeStance,
		"away_tactical_stance":  e.AwayStance,
		"latest_tactical_shift": e.LatestTacticalShift,
		"home_manager":          s.serializeManager(e.HomeManager),
		"away_manager":          s.serializeManager(e.AwayManager),
		"weather":               e.Weather,
		"home_bench":            s.liveBenchPayload("home"),
		"away_bench":            s.liveBenchPayload("away"),
	}
}

// buildDeltaTickPayload ships an ordinary-tick update: slim coords (no player
// blobs), scores, the newest event/commentary items with their totals, and the
// ball/momentum/trail fields. The client applies it onto the last full
// snapshot. Caller must hold worldMu.
func (s *Server) buildDeltaTickPayload() map[string]interface{} {
	e := s.LiveMatchEngine
	if e == nil {
		return nil
	}

	slim := func(actors []matchengine.LivePlayerRadar) []map[string]interface{} {
		out := make([]map[string]interface{}, len(actors))
		for i, p := range actors {
			out[i] = map[string]interface{}{
				"x":         p.X,
				"y":         p.Y,
				"player_id": p.PlayerID,
				"sent_off":  e.Bookings[p.PlayerID] >= 2,
			}
		}
		return out
	}

	var lastEvent interface{}
	if len(e.Events) > 0 {
		lastEvent = e.Events[len(e.Events)-1]
	}
	var lastCommentary interface{}
	if len(e.Commentary) > 0 {
		lastCommentary = e.Commentary[len(e.Commentary)-1]
	}

	var leagueFixture map[string]interface{}
	if s.liveFixtureID != "" && s.TournamentManager != nil {
		if f := s.TournamentManager.FindFixture(s.liveFixtureID); f != nil {
			leagueFixture = map[string]interface{}{"id": f.FixtureID, "status": f.Status}
		}
	}

	return map[string]interface{}{
		"tick_type":            "delta",
		"seq":                  s.wsTickSeq,
		"state":                e.State,
		"minute":               math.Round(e.CurrentMinute*10) / 10,
		"speed":                e.Speed,
		"phase":                e.Phase,
		"active_third":         e.ActiveThird,
		"home_score":           e.HomeScore,
		"away_score":           e.AwayScore,
		"home_shots":           e.HomeShots,
		"away_shots":           e.AwayShots,
		"home_shots_on_target": e.HomeShotsOn,
		"away_shots_on_target": e.AwayShotsOn,
		"home_corners":         e.HomeCorners,
		"away_corners":         e.AwayCorners,
		"home_possession_pct":  e.HomePossessionPct(),
		"away_possession_pct":  e.AwayPossessionPct(),
		"possession_momentum":  math.Round(e.PossessionMomentum*1000) / 1000,
		"ball": map[string]interface{}{
			"x":       e.BallPos.X,
			"y":       e.BallPos.Y,
			"height":  math.Round(e.BallHeight*100) / 100,
			"is_shot": e.BallIsShot,
		},
		"goal_banner":           e.Banner,
		"league_fixture":        leagueFixture,
		"pass_trail":            serializePassTrail(e.PassTrail),
		"home_coords":           slim(e.HomePlayers),
		"away_coords":           slim(e.AwayPlayers),
		"last_event":            lastEvent,
		"event_count":           len(e.Events),
		"last_commentary":       lastCommentary,
		"commentary_count":      len(e.Commentary),
		"home_tactical_stance":  e.HomeStance,
		"away_tactical_stance":  e.AwayStance,
		"latest_tactical_shift": e.LatestTacticalShift,
		"weather":               e.Weather,
	}
}

// serializePassTrail maps the engine's latest ball segment to the shape the
// pitch canvas already consumes. Nil stays nil so the canvas draws no line.
func serializePassTrail(trail *matchengine.PassTrailItem) map[string]interface{} {
	if trail == nil {
		return nil
	}
	return map[string]interface{}{
		"from":    [2]float64{trail.From[0], trail.From[1]},
		"to":      [2]float64{trail.To[0], trail.To[1]},
		"is_shot": trail.IsShot,
		"color":   [3]uint8{trail.Color[0], trail.Color[1], trail.Color[2]},
	}
}

func (s *Server) liveBenchPayload(side string) []map[string]interface{} {
	e := s.LiveMatchEngine
	if e == nil {
		return []map[string]interface{}{}
	}
	bench := e.HomeBench
	if side != "home" {
		bench = e.AwayBench
	}
	onByID := map[string]int{}
	for _, sub := range e.PlannedSubs {
		if sub.Done && sub.Side == side && sub.In != nil {
			onByID[sub.In.PlayerID] = sub.Minute
		}
	}
	for _, ev := range e.Events {
		if ev.Type == "sub" && ev.Side == side && ev.PlayerIn != nil && ev.PlayerIn.PlayerID != "" {
			onByID[ev.PlayerIn.PlayerID] = ev.Minute
		}
	}
	rows := make([]map[string]interface{}, 0, len(bench))
	for _, p := range bench {
		if p == nil {
			continue
		}
		status := "bench"
		var onMin interface{}
		if minute, ok := onByID[p.PlayerID]; ok {
			status = "on"
			onMin = minute
		}
		rows = append(rows, map[string]interface{}{
			"player":    s.serializePlayer(p),
			"status":    status,
			"on_minute": onMin,
		})
	}
	return rows
}

func (s *Server) wsClientCount() int {
	s.wsMu.Lock()
	defer s.wsMu.Unlock()
	return len(s.wsClients)
}

func (s *Server) writeMatchTick(conn *websocket.Conn, payload interface{}) error {
	if s.wsWrite != nil {
		return s.wsWrite(conn, payload)
	}
	return conn.WriteJSON(payload)
}

func (s *Server) broadcastMatchTick(payload map[string]interface{}) {
	if payload == nil {
		return
	}

	s.wsMu.Lock()
	clients := make([]*websocket.Conn, 0, len(s.wsClients))
	for conn := range s.wsClients {
		clients = append(clients, conn)
	}
	s.wsMu.Unlock()

	var failed []*websocket.Conn
	for _, conn := range clients {
		if err := s.writeMatchTick(conn, payload); err != nil {
			failed = append(failed, conn)
		}
	}
	if len(failed) == 0 {
		return
	}

	s.wsMu.Lock()
	for _, conn := range failed {
		if _, ok := s.wsClients[conn]; ok {
			delete(s.wsClients, conn)
			_ = conn.Close()
		}
	}
	s.wsMu.Unlock()
}

// takeCareerSnapshotLocked encodes career state for a later disk write.
// Caller must hold worldMu so maps are not mutated during marshal.
func (s *Server) takeCareerSnapshotLocked() ([]byte, uint64) {
	if s.TransferEngine != nil && s.TournamentManager != nil {
		s.TransferEngine.CurrentMatchweek = s.TournamentManager.CurrentMatchweek
	}
	data, err := json.MarshalIndent(persistence.BuildSnapshot(s.TournamentManager, s.GrowthEngine, s.TransferEngine), "", "  ")
	if err != nil {
		return nil, 0
	}
	return data, s.saveGen.Add(1)
}

// commitCareerSnapshot writes encoded JSON only if it is still the latest
// generation. Must not be called while holding worldMu or wsMu.
func (s *Server) commitCareerSnapshot(data []byte, gen uint64) {
	if len(data) == 0 || gen == 0 {
		return
	}
	if s.saveGen.Load() != gen {
		return
	}
	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	if s.saveGen.Load() != gen {
		return
	}
	_, _ = persistence.WriteSnapshotBytes(data, s.savePath)
}

// -----------------------------------------------------------------------------
// SERIALIZATION HELPERS
// -----------------------------------------------------------------------------

func (s *Server) serializePlayer(p *models.Player) map[string]interface{} {
	if p == nil {
		return nil
	}

	careerGoals := p.Goals + p.CareerGoals
	careerAssists := p.Assists + p.CareerAssists
	careerApps := p.Appearances + p.CareerApps

	return map[string]interface{}{
		"player_id":          p.PlayerID,
		"full_name":          p.FullName,
		"position":           p.Position,
		"ovr":                p.OVR,
		"age":                p.Age,
		"market_value_eur":   p.MarketValueEUR,
		"universe_wonderkid": p.UniverseWonderkid,
		"player_source":      p.PlayerSource,
		"club_id":            p.ClubID,
		"goals":              p.Goals,
		"assists":            p.Assists,
		"appearances":        p.Appearances,
		"category":           p.Category,
		"formatted_value":    models.FormatCurrency(p.MarketValueEUR),
		"wage_eur":           p.WageEUR,
		"formatted_wage":     models.FormatWage(p.WageEUR),
		"contract_years":     p.ContractYears,
		"loyalty":            p.Loyalty,
		"career_goals":       careerGoals,
		"career_assists":     careerAssists,
		"career_apps":        careerApps,
		"own_goals":          p.OwnGoals,
		"suspended_matches":  p.SuspendedMatches,
		"injured_matches":    p.InjuredMatches,
		"injury":             p.Injury,
		"availability":       p.AvailabilityNote("super-league", s.TournamentManager.CurrentMatchweek),
		"effective_ovr":      p.EffectiveOVR(),
		"consecutive_starts": p.ConsecutiveStarts,
		"education":          p.Education,
		"education_label":    p.EducationLabel(),
		"education_pending":  p.EducationPending,
		"school_want":        p.SchoolWant,
		"school_want_label":  p.SchoolWantLine(),
		"school_track":       p.SchoolTrack,
		"school_track_label": p.SchoolTrackLabel(),
		"secondary_position": p.SecondaryPosition,
		"position_path":      p.PositionPath,
		"position_xp":        p.PositionXP,
		"position_options":   p.PositionOptions(),
		"best_goals":         p.BestGoals,
		"best_assists":       p.BestAssists,
		"best_season":        p.BestSeason,
		"personality":        p.Personality,
		"personality_title":  p.PersonalityTitle(),
		"personality_badge":  p.PersonalityBadge(),
		"personality_desc":   p.PersonalityInfo().Description,
		"mentor_id":          p.MentorID,
		"mentor_name":        p.MentorName,
		"mentor_ovr":         p.MentorOVR,
	}
}

func (s *Server) serializeClub(c *models.Club) map[string]interface{} {
	return s.serializeClubRecord(c, nil)
}

func (s *Server) serializeClubRecord(c *models.Club, rec *models.CompetitionRecord) map[string]interface{} {
	if c == nil {
		return nil
	}

	mgr := s.TournamentManager.Managers[c.ClubID]
	p, w, d, l := c.Played, c.Won, c.Drawn, c.Lost
	gf, ga, gd, pts := c.GoalsFor, c.GoalsAgainst, c.GoalDifference, c.Points
	form := c.Form
	if rec != nil {
		p, w, d, l = rec.Played, rec.Won, rec.Drawn, rec.Lost
		gf, ga, gd, pts = rec.GoalsFor, rec.GoalsAgainst, rec.GoalDifference, rec.Points
		form = rec.Form
	}
	return map[string]interface{}{
		"club_id":             c.ClubID,
		"club_name":           c.ClubName,
		"short_name":          c.ShortName,
		"league":              c.League,
		"country":             c.Country,
		"home_stadium":        c.HomeStadium,
		"stadium_capacity":    c.StadiumCapacity,
		"overall_team_rating": c.OverallTeamRating,
		"squad_size":          c.SquadSize,
		"squad_avg_ovr":       c.SquadAvgOVR,
		"primary_color":       c.PrimaryColor,
		"secondary_color":     c.SecondaryColor,
		"p":                   p,
		"w":                   w,
		"d":                   d,
		"l":                   l,
		"gf":                  gf,
		"ga":                  ga,
		"gd":                  gd,
		"pts":                 pts,
		"form":                form,
		"morale":              c.Morale,
		"manager":             s.serializeManager(mgr),
	}
}

func (s *Server) serializeManager(m *managers.ManagerProfile) map[string]interface{} {
	if m == nil {
		return nil
	}

	arch := m.ArchetypeInfo()
	return map[string]interface{}{
		"name":                m.Name,
		"tactic":              m.Tactic(),
		"style":               m.Style,
		"canonical_style":     m.CanonicalStyle(),
		"dogma_title":         m.DogmaTitle(),
		"description":         arch.Description,
		"line_height":         arch.LineHeight,
		"press_intensity":     arch.PressIntensity,
		"tempo":               arch.Tempo,
		"focus":               m.FocusLabel(),
		"budget_eur":          m.BudgetEur,
		"formatted_budget":    models.FormatCurrency(m.BudgetEur),
		"adaptability":        m.Adaptability,
		"archetype":           m.CanonicalStyle(),
		"archetype_label":     m.DogmaTitle(),
		"job_security":        m.JobSecurity,
		"appointed_season":    m.AppointedSeason,
		"appointed_matchweek": m.AppointedMatchweek,
		"history":             m.History,
	}
}

func (s *Server) serializeFixture(f *tournament.Fixture) map[string]interface{} {
	if f == nil {
		return nil
	}

	home := s.TournamentManager.Clubs[f.HomeID]
	away := s.TournamentManager.Clubs[f.AwayID]

	derby := f.DerbyName
	if derby == "" {
		derby = tournament.GetDerbyName(f.HomeID, f.AwayID)
	}
	heat := f.DerbyHeat
	if derby != "" && heat == 0 {
		heat = 50
		if v, ok := s.TournamentManager.DerbyHeat[derby]; ok {
			heat = v
		}
	}
	out := map[string]interface{}{
		"id":                 f.FixtureID,
		"fixture_id":         f.FixtureID,
		"matchweek":          f.Matchweek,
		"competition":        f.Competition,
		"stage":              f.Stage,
		"leg":                f.Leg,
		"tie_id":             f.TieID,
		"home_id":            f.HomeID,
		"away_id":            f.AwayID,
		"home":               s.serializeClub(home),
		"away":               s.serializeClub(away),
		"home_club":          s.serializeClub(home),
		"away_club":          s.serializeClub(away),
		"status":             f.Status,
		"method":             f.Method,
		"home_goals":         f.HomeGoals,
		"away_goals":         f.AwayGoals,
		"weather":            s.TournamentManager.ResolveWeather(f),
		"derby":              derby,
		"derby_name":         derby,
		"is_derby":           derby != "",
		"derby_heat":         heat,
		"is_high_heat_derby": f.IsHighHeatDerby || heat > 70,
		"referee":            f.Referee,
		"decided_by":         f.DecidedBy,
		"penalties":          f.Penalties,
		"home_manager":       s.serializeManager(s.TournamentManager.Managers[f.HomeID]),
		"away_manager":       s.serializeManager(s.TournamentManager.Managers[f.AwayID]),
		"preview":            s.fixturePreview(f, home, away),
		"head_to_head":       s.TournamentManager.FixtureHeadToHead(f),
		"night":              s.TournamentManager.EuropeanNight(f),
		"events":             []interface{}{},
		"home_xi":            []interface{}{},
		"away_xi":            []interface{}{},
		"home_bench":         []interface{}{},
		"away_bench":         []interface{}{},
	}
	if f.Report != nil {
		out["report"] = f.Report
		out["events"] = f.Report.Events
		out["home_xi"] = f.Report.HomeXI
		out["away_xi"] = f.Report.AwayXI
		out["home_bench"] = f.Report.HomeBench
		out["away_bench"] = f.Report.AwayBench
		out["stats"] = f.Report.Stats
		out["motm"] = f.Report.MOTM
		out["ht_home"] = f.Report.HTHome
		out["ht_away"] = f.Report.HTAway
		out["attendance"] = f.Report.Attendance
		out["shot_map"] = f.Report.ShotMap
		out["touch_heatmap"] = f.Report.Heatmap
		out["press_conference"] = f.Report.PressConference
		if f.Report.Referee != "" {
			out["referee"] = f.Report.Referee
		}
		if f.Report.DecidedBy != nil {
			out["decided_by"] = *f.Report.DecidedBy
		}
		if f.Report.Penalties != nil {
			out["penalties"] = f.Report.Penalties
		}
	}
	return out
}

func (s *Server) clubFixtureContext(clubID string) string {
	mw := 1
	if s.TournamentManager != nil && s.TournamentManager.CurrentMatchweek > 0 {
		mw = s.TournamentManager.CurrentMatchweek
	}
	if s.TournamentManager != nil {
		for _, f := range s.TournamentManager.GetSlate(mw) {
			if f.HomeID == clubID || f.AwayID == clubID {
				return models.FixtureContext(f.Competition, f.Matchweek)
			}
		}
	}
	return models.FixtureContext("super-league", mw)
}

func (s *Server) fixturePreview(f *tournament.Fixture, home, away *models.Club) map[string]interface{} {
	if f == nil || home == nil || away == nil || s.TournamentManager == nil {
		return nil
	}
	fx := models.FixtureContext(f.Competition, f.Matchweek)
	table := s.TournamentManager.GetStandings()
	pos := map[string]int{}
	for i, c := range table {
		pos[c.ClubID] = i + 1
	}
	serializeXI := func(players []*models.Player) []map[string]interface{} {
		out := make([]map[string]interface{}, 0, len(players))
		for _, p := range players {
			if p == nil {
				continue
			}
			row := s.serializePlayer(p)
			row["availability"] = p.AvailabilityNote(f.Competition, f.Matchweek)
			if note := s.grewNoteFor(p); note != "" {
				row["grew_note"] = note
			}
			out = append(out, row)
		}
		return out
	}
	missing := func(club *models.Club) []map[string]interface{} {
		out := make([]map[string]interface{}, 0)
		for _, p := range club.Squad {
			if p == nil || !p.IsUnavailable(f.Competition, f.Matchweek) {
				continue
			}
			row := s.serializePlayer(p)
			row["availability"] = p.AvailabilityNote(f.Competition, f.Matchweek)
			out = append(out, row)
		}
		return out
	}
	xiAvg := func(xi []*models.Player) int {
		if len(xi) == 0 {
			return 0
		}
		sum := 0
		n := 0
		for _, p := range xi {
			if p == nil {
				continue
			}
			sum += p.OVR
			n++
		}
		if n == 0 {
			return 0
		}
		return int(math.Round(float64(sum) / float64(n)))
	}
	homeXI := home.GetStartingEleven(fx)
	awayXI := away.GetStartingEleven(fx)
	homeForm := home.Form
	if len(homeForm) > 5 {
		homeForm = homeForm[len(homeForm)-5:]
	}
	awayForm := away.Form
	if len(awayForm) > 5 {
		awayForm = awayForm[len(awayForm)-5:]
	}
	homePos, awayPos := interface{}(nil), interface{}(nil)
	if p := pos[home.ClubID]; p > 0 {
		homePos = p
	}
	if p := pos[away.ClubID]; p > 0 {
		awayPos = p
	}
	return map[string]interface{}{
		"kickoff_note": s.TournamentManager.KickoffNote(f, home, away),
		"venue":        home.HomeStadium,
		"capacity":     home.StadiumCapacity,
		"home_form":    homeForm,
		"away_form":    awayForm,
		"home_pos":     homePos,
		"away_pos":     awayPos,
		"home_pts":     home.Points,
		"away_pts":     away.Points,
		"home_xi":      serializeXI(homeXI),
		"away_xi":      serializeXI(awayXI),
		"home_bench":   serializeXI(home.GetBench(homeXI, 7, fx)),
		"away_bench":   serializeXI(away.GetBench(awayXI, 7, fx)),
		"home_missing": missing(home),
		"away_missing": missing(away),
		"home_xi_avg":  xiAvg(homeXI),
		"away_xi_avg":  xiAvg(awayXI),
	}
}

// grewNoteFor returns the pre-match XI one-liner when a prodigy has grown
// since the August baseline, or "" otherwise. Read-only: puberty curves and
// composure are untouched.
func (s *Server) grewNoteFor(p *models.Player) string {
	if p == nil || !p.UniverseWonderkid || s.GrowthEngine == nil {
		return ""
	}
	data, ok := s.GrowthEngine.GetProdigyData(p.PlayerID, p.Category)
	if !ok {
		return ""
	}
	gain, ok := data["height_gain_cm"].(float64)
	if !ok || gain < 0.1 {
		return ""
	}
	return fmt.Sprintf("+%.1f cm since August — growing into the shirt.", gain)
}

func (s *Server) findLiveFixture(homeID, awayID string) *tournament.Fixture {
	mw := s.TournamentManager.CurrentMatchweek
	for _, f := range s.TournamentManager.GetSlate(mw) {
		if f.HomeID == homeID && f.AwayID == awayID && f.Status == "scheduled" {
			cp := f
			return &cp
		}
	}
	return nil
}

func (s *Server) clearLiveFixtureSelection() {
	s.liveFixtureID = ""
	if s.LiveMatchEngine != nil {
		s.LiveMatchEngine.ClearFixtureContext()
	}
}

func copyStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func (s *Server) prodigyDrawPayload(homes map[string]string) map[string]interface{} {
	if len(homes) == 0 {
		homes = datamanager.DefaultProdigyHomes()
	}
	return map[string]interface{}{
		"homes": copyStringMap(homes),
		"draw":  s.DataManager.DescribeProdigyDraw(homes),
	}
}

func resolveNewCareerHomes(shuffle bool, homes map[string]string) map[string]string {
	if len(homes) > 0 {
		return copyStringMap(homes)
	}
	if shuffle {
		return datamanager.ShuffleProdigyHomes(nil)
	}
	return datamanager.DefaultProdigyHomes()
}

func (s *Server) resetLiveMatchDefault() {
	s.lastCommittedLiveInstance = -1
	s.liveFixtureID = ""
	s.wsForceFull = true
	s.wsLastHomeScore = 0
	s.wsLastAwayScore = 0
	s.wsLastEvents = 0
	if s.LiveMatchEngine == nil || s.TournamentManager == nil {
		return
	}
	home := s.TournamentManager.Clubs["LAL-RMA"]
	away := s.TournamentManager.Clubs["EPL-ARS"]
	if home == nil || away == nil {
		if len(s.TournamentManager.ClubsList) >= 2 {
			home = s.TournamentManager.ClubsList[0]
			away = s.TournamentManager.ClubsList[1]
		}
	}
	if home == nil || away == nil {
		s.LiveMatchEngine.ClearFixtureContext()
		return
	}
	s.LiveMatchEngine.SetClubs(home, away, s.TournamentManager.Managers[home.ClubID], s.TournamentManager.Managers[away.ClubID])
}

func (s *Server) bootFreshCareer(homes map[string]string, shuffle bool) error {
	datasetPath := ""
	if s.DataManager != nil {
		datasetPath = s.DataManager.JSONPath
	}
	previousDefault := growth.GetDefaultGrowthEngine()
	ge := growth.NewGrowthEngine(time.Now().UnixNano())
	dm := datamanager.NewDataManager(datasetPath, ge)
	if dm == nil || len(dm.GetEliteClubs()) != 12 {
		if previousDefault != nil {
			growth.SetDefaultGrowthEngine(previousDefault)
		}
		return errFreshCareer
	}
	dm.ApplyProdigyHomes(homes)
	elite := dm.GetEliteClubs()
	tm := tournament.NewTournamentManager(elite, ge, time.Now().UnixNano())
	te := transfers.NewTransferEngine(elite, tm.Managers, time.Now().UnixNano())
	tm.TransferEngine = te
	tm.ProdigyHomes = copyStringMap(dm.ProdigyHomes)
	tm.LastCareerShuffle = shuffle

	s.GrowthEngine = ge
	s.DataManager = dm
	s.TournamentManager = tm
	s.TransferEngine = te
	s.resetLiveMatchDefault()
	return nil
}

func (s *Server) findCurrentLeagueFixture(homeID, awayID string) *tournament.Fixture {
	if s.TournamentManager == nil || s.TournamentManager.CurrentMatchweek > s.TournamentManager.MaxMatchweeks {
		return nil
	}
	for _, f := range s.TournamentManager.GetMatchweekFixtures(s.TournamentManager.CurrentMatchweek) {
		if f.HomeID == homeID && f.AwayID == awayID {
			cp := f
			return &cp
		}
	}
	return nil
}

func (s *Server) findPlayer(playerID string) (*models.Player, *models.Club) {
	for _, club := range s.TournamentManager.ClubsList {
		for _, p := range club.Squad {
			if p.PlayerID == playerID {
				return p, club
			}
		}
	}
	return nil, nil
}

// -----------------------------------------------------------------------------
// REST ENDPOINT HANDLERS
// -----------------------------------------------------------------------------

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{
		"status":  "ok",
		"backend": "go",
		"version": "1.0.0",
		"time":    time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	wsCount := s.wsClientCount()

	s.worldMu.RLock()
	defer s.worldMu.RUnlock()

	writeJSON(w, map[string]interface{}{
		"status":           "ok",
		"matchweek":        s.TournamentManager.CurrentMatchweek,
		"season_name":      s.TournamentManager.SeasonName,
		"clubs_count":      len(s.TournamentManager.ClubsList),
		"wonderkids_count": len(s.DataManager.Wonderkids),
		"ws_clients":       wsCount,
		"total_transfers":  len(s.TransferEngine.AllTimeTransfers),
		"total_news_items": len(s.TournamentManager.Inbox),
	})
}

// Fat GETs snapshot detached payloads under worldMu and encode after release
// so fat JSON cannot stall the 60 FPS ticker holding worldMu.
func (s *Server) handleGetClubs(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	var result []map[string]interface{}
	for _, c := range s.TournamentManager.ClubsList {
		result = append(result, s.serializeClub(c))
	}
	s.worldMu.RUnlock()
	writeJSON(w, result)
}

func (s *Server) handleGetClubSquad(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	cid := r.PathValue("club_id")
	club := s.TournamentManager.Clubs[cid]
	var squad []map[string]interface{}
	if club != nil {
		for _, p := range club.Squad {
			squad = append(squad, s.serializePlayer(p))
		}
	}
	s.worldMu.RUnlock()

	if club == nil {
		http.Error(w, "Club not found", http.StatusNotFound)
		return
	}
	writeJSON(w, squad)
}

func (s *Server) handleGetClubXI(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	cid := r.PathValue("club_id")
	club := s.TournamentManager.Clubs[cid]
	startersJSON := []map[string]interface{}{}
	if club != nil {
		starters := club.GetStartingEleven(s.clubFixtureContext(cid))
		for _, p := range starters {
			if p == nil {
				continue
			}
			startersJSON = append(startersJSON, s.serializePlayer(p))
		}
	}
	s.worldMu.RUnlock()

	if club == nil {
		http.Error(w, "Club not found", http.StatusNotFound)
		return
	}
	writeJSON(w, startersJSON)
}

func (s *Server) handleGetClubHistory(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	cid := r.PathValue("club_id")
	club := s.TournamentManager.Clubs[cid]
	var payload map[string]interface{}
	if club != nil {
		hist := append([]map[string]interface{}{}, s.TournamentManager.ClubSeasonHistory[cid]...)
		if hist == nil {
			hist = []map[string]interface{}{}
		}
		payload = map[string]interface{}{
			"club_id":          club.ClubID,
			"club_name":        club.ClubName,
			"short_name":       club.ShortName,
			"primary_color":    club.PrimaryColor,
			"history":          hist,
			"trophies_summary": clubHistoryTrophySummary(hist),
			"historical":       tournament.HistoricalClubTrophies[cid],
		}
	}
	s.worldMu.RUnlock()

	if club == nil {
		http.Error(w, "Club not found", http.StatusNotFound)
		return
	}
	writeJSON(w, payload)
}

func clubHistoryTrophySummary(hist []map[string]interface{}) map[string]int {
	out := map[string]int{"super_league": 0, "ucl": 0, "super_cup": 0}
	for _, row := range hist {
		var titles []string
		switch v := row["trophies"].(type) {
		case []string:
			titles = v
		case []interface{}:
			for _, item := range v {
				if s, ok := item.(string); ok {
					titles = append(titles, s)
				}
			}
		}
		for _, title := range titles {
			switch title {
			case "Super League Champion":
				out["super_league"]++
			case "Champions Cup":
				out["ucl"]++
			case "Super Cup":
				out["super_cup"]++
			}
		}
	}
	return out
}

func (s *Server) handleGetH2H(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	clubA := r.PathValue("club_a")
	clubB := r.PathValue("club_b")
	res := s.TournamentManager.GetHeadToHead(clubA, clubB)
	s.worldMu.RUnlock()

	if res == nil {
		http.Error(w, "Club not found", http.StatusNotFound)
		return
	}
	writeJSON(w, res)
}

func (s *Server) handleGetPlayerProfile(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	pid := r.PathValue("player_id")
	p, club := s.findPlayer(pid)
	var payload map[string]interface{}
	if p != nil {
		var bio *growth.BiometricProfile
		var attrs *growth.TechnicalAttributes
		var timeline []growth.TimelineEntry
		if s.GrowthEngine != nil {
			// Snapshot structs by value so the encode below cannot race an
			// in-place training update.
			if live := s.GrowthEngine.Biometrics[p.PlayerID]; live != nil {
				cp := *live
				bio = &cp
			}
			if live := s.GrowthEngine.Attributes[p.PlayerID]; live != nil {
				cp := *live
				attrs = &cp
			}
			timeline = append([]growth.TimelineEntry{}, s.GrowthEngine.Timeline[p.PlayerID]...)
		}

		log := s.TournamentManager.PlayerMatchLog(pid, 8)
		var avg interface{}
		rated := 0
		sum := 0.0
		for _, row := range log {
			if rating, ok := row["rating"].(float64); ok {
				sum += rating
				rated++
			}
		}
		if rated > 0 {
			avg = math.Round(sum/float64(rated)*100) / 100
		}
		payload = map[string]interface{}{
			"player":       s.serializePlayer(p),
			"club":         s.serializeClub(club),
			"biometrics":   bio,
			"attributes":   attrs,
			"timeline":     timeline,
			"last_matches": log,
			"avg_rating":   avg,
			"apps_rated":   rated,
		}
	}
	s.worldMu.RUnlock()

	if p == nil {
		http.Error(w, "Player not found", http.StatusNotFound)
		return
	}
	writeJSON(w, payload)
}

func (s *Server) handleGetProdigies(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	list := []map[string]interface{}{}
	for _, p := range s.DataManager.Wonderkids {
		data, ok := s.GrowthEngine.GetProdigyData(p.PlayerID, p.Category)
		if !ok {
			continue
		}
		club := s.TournamentManager.Clubs[p.ClubID]
		if club != nil {
			data["club_name"] = club.ClubName
			data["club_short"] = club.ShortName
			data["primary_color"] = club.PrimaryColor
		} else {
			data["club_name"] = p.ClubID
			data["club_short"] = "UNK"
			data["primary_color"] = [3]uint8{200, 200, 200}
		}
		data["formatted_value"] = p.FormattedValue()
		data["goals"] = p.Goals
		data["assists"] = p.Assists
		data["appearances"] = p.Appearances
		data["position"] = p.Position
		data["education"] = p.Education
		data["education_label"] = p.EducationLabel()
		data["education_pending"] = p.EducationPending
		data["school_want"] = p.SchoolWant
		data["school_want_label"] = p.SchoolWantLine()
		data["school_track"] = p.SchoolTrack
		data["school_track_label"] = p.SchoolTrackLabel()
		data["secondary_position"] = p.SecondaryPosition
		data["position_path"] = p.PositionPath
		data["position_xp"] = p.PositionXP
		data["position_options"] = p.PositionOptions()
		data["career_goals"] = p.Goals + p.CareerGoals
		data["career_assists"] = p.Assists + p.CareerAssists
		data["career_apps"] = p.Appearances + p.CareerApps
		data["best_goals"] = p.BestGoals
		data["best_assists"] = p.BestAssists
		data["best_season"] = p.BestSeason
		data["personality"] = p.Personality
		data["personality_title"] = p.PersonalityTitle()
		data["personality_badge"] = p.PersonalityBadge()
		data["personality_desc"] = p.PersonalityInfo().Description
		data["mentor_id"] = p.MentorID
		data["mentor_name"] = p.MentorName
		data["mentor_ovr"] = p.MentorOVR
		data["composure"] = p.Composure
		list = append(list, data)
	}
	s.worldMu.RUnlock()
	writeJSON(w, list)
}

func (s *Server) handleGetProdigyWatch(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	payload := map[string]interface{}{
		"season_name": s.TournamentManager.SeasonName,
		"matchweek":   s.TournamentManager.CurrentMatchweek,
		"rankings":    s.TournamentManager.GetProdigyWatch(),
	}
	s.worldMu.RUnlock()
	writeJSON(w, payload)
}

func (s *Server) handleGetWonderkids(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	wks := s.DataManager.Wonderkids
	var topScorer, topAssister, topOVR *models.Player
	total := int64(0)
	serialized := []map[string]interface{}{}
	for _, p := range wks {
		serialized = append(serialized, s.serializePlayer(p))
		total += p.MarketValueEUR
		if topScorer == nil || p.Goals > topScorer.Goals {
			topScorer = p
		}
		if topAssister == nil || p.Assists > topAssister.Assists {
			topAssister = p
		}
		if topOVR == nil || p.OVR > topOVR.OVR {
			topOVR = p
		}
	}
	payload := map[string]interface{}{
		"top_scorer":      s.serializePlayer(topScorer),
		"top_assister":    s.serializePlayer(topAssister),
		"top_ovr":         s.serializePlayer(topOVR),
		"total_valuation": total,
		"wonderkids":      serialized,
	}
	s.worldMu.RUnlock()
	writeJSON(w, payload)
}

func (s *Server) handleTrainProdigy(w http.ResponseWriter, r *http.Request) {
	s.worldMu.Lock()
	held := true
	defer func() {
		if held {
			s.worldMu.Unlock()
		}
	}()

	pid := r.PathValue("player_id")
	var req struct {
		Focus string `json:"focus"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Focus == "" {
		req.Focus = "technical"
	}

	res, err := s.GrowthEngine.RunTrainingCycle(pid, req.Focus)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	p, _ := s.findPlayer(pid)
	if p != nil {
		p.OVR = s.GrowthEngine.CalculateOVR(pid, p.Category)
		res["ovr"] = p.OVR
	}
	snap, gen := s.takeCareerSnapshotLocked()
	s.worldMu.Unlock()
	held = false
	s.commitCareerSnapshot(snap, gen)
	writeJSON(w, res)
}

func (s *Server) handleSetPositionPath(w http.ResponseWriter, r *http.Request) {
	s.worldMu.Lock()
	held := true
	defer func() {
		if held {
			s.worldMu.Unlock()
		}
	}()

	pid := r.PathValue("player_id")
	var req struct {
		Position string `json:"position"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	p, _ := s.findPlayer(pid)
	if p == nil {
		http.Error(w, "Player not found", http.StatusNotFound)
		return
	}

	p.PositionPath = req.Position
	p.SecondaryPosition = req.Position
	payload := map[string]interface{}{
		"status": "success",
		"player": s.serializePlayer(p),
	}
	snap, gen := s.takeCareerSnapshotLocked()
	s.worldMu.Unlock()
	held = false
	s.commitCareerSnapshot(snap, gen)
	writeJSON(w, payload)
}

func (s *Server) handleSetSchoolTrack(w http.ResponseWriter, r *http.Request) {
	s.worldMu.Lock()
	held := true
	defer func() {
		if held {
			s.worldMu.Unlock()
		}
	}()

	pid := r.PathValue("player_id")
	var req struct {
		Track string `json:"track"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	p, _ := s.findPlayer(pid)
	if p == nil {
		http.Error(w, "Player not found", http.StatusNotFound)
		return
	}
	if !p.UniverseWonderkid {
		http.Error(w, "School track is only for prodigies", http.StatusBadRequest)
		return
	}
	if !p.SetSchoolTrack(req.Track) {
		http.Error(w, "Unknown track (stay, football_first, club_forced)", http.StatusBadRequest)
		return
	}
	payload := map[string]interface{}{
		"status": "success",
		"player": s.serializePlayer(p),
	}
	snap, gen := s.takeCareerSnapshotLocked()
	s.worldMu.Unlock()
	held = false
	s.commitCareerSnapshot(snap, gen)
	writeJSON(w, payload)
}

func (s *Server) handleGetGrowthMilestones(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	milestones := append([]growth.GrowthMilestone{}, s.GrowthEngine.Milestones...)
	s.worldMu.RUnlock()
	writeJSON(w, milestones)
}

// narrativeMilestonesFor builds the eight career achievements the timeline
// tab renders (id/title/description/unlocked/badge). Thresholds mirror the
// client fallback so both agree on what "unlocked" means.
func narrativeMilestonesFor(bio *growth.BiometricProfile, p *models.Player) []map[string]interface{} {
	goals, assists, apps, ovr, gain := 0, 0, 0, 0, 0.0
	if bio != nil {
		ovr = bio.BaselineOVR
		gain = bio.HeightGainCM()
	}
	if p != nil {
		ovr = p.OVR
		goals = p.Goals + p.CareerGoals
		assists = p.Assists + p.CareerAssists
		apps = p.Appearances + p.CareerApps
	}
	mk := func(id, title, description, badge string, unlocked bool) map[string]interface{} {
		return map[string]interface{}{
			"id": id, "title": title, "description": description,
			"unlocked": unlocked, "badge": badge,
		}
	}
	return []map[string]interface{}{
		mk("first_contract", "First Professional Contract", "Signed official youth forms with senior team at age 14.", "gold", true),
		mk("growth_spurt", "Puberty Growth Spurt", "Grew frame since the August baseline.", "cyan", gain >= 1.0),
		mk("senior_debut", "Senior Debut", "Earned first team match minutes in the European Super League.", "green", apps > 0),
		mk("first_goal", "First European Goal", "Slotted the ball home against elite continental competition.", "gold", goals >= 1),
		mk("youth_cap", "Youth International Cap", "Earned national youth honors after surpassing 78 OVR threshold.", "purple", ovr >= 78),
		mk("elite_playmaker", "Elite Playmaker", "Created 5+ career assists orchestrating offensive attacks.", "cyan", assists >= 5),
		mk("century_mark", "Trophy & Goal Contender", "Reached 15+ combined career goals and assists.", "gold", goals+assists >= 15),
		mk("ballon_dor_contender", "Ballon d'Or Candidate", "Surpassed 85 OVR ascending to world-class elite status.", "gold", ovr >= 85),
	}
}

func (s *Server) handleGetProdigyTimeline(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	pid := r.PathValue("player_id")
	bio := s.GrowthEngine.Biometrics[pid]
	var payload map[string]interface{}
	if bio != nil {
		p, club := s.findPlayer(pid)
		history := s.GrowthEngine.GetProgressionHistory(pid)
		if len(history) == 0 {
			ovr, clubShort := bio.BaselineOVR, "UNK"
			if p != nil {
				ovr = p.OVR
			}
			if club != nil {
				clubShort = club.ShortName
			}
			history = []growth.TimelineEntry{{
				Season: s.TournamentManager.SeasonName, Age: bio.Age, OVR: ovr,
				HeightCM: bio.CurrentHeightCM, WeightKG: bio.CurrentWeightKG,
				ClubShort: clubShort, MentorName: bio.MentorName,
			}}
		}
		ovr := bio.BaselineOVR
		if p != nil {
			ovr = p.OVR
		}
		payload = map[string]interface{}{
			"player_id":           pid,
			"full_name":           bio.FullName,
			"age":                 bio.Age,
			"current_height_cm":   bio.CurrentHeightCM,
			"baseline_height_cm":  bio.BaselineHeightCM,
			"current_weight_kg":   bio.CurrentWeightKG,
			"baseline_weight_kg":  bio.BaselineWeightKG,
			"height_gain_cm":      bio.HeightGainCM(),
			"weight_gain_kg":      bio.WeightGainKG(),
			"ovr":                 ovr,
			"potential":           bio.Potential,
			"progression_history": history,
			"milestones":          narrativeMilestonesFor(bio, p),
		}
	}
	s.worldMu.RUnlock()

	if bio == nil {
		http.Error(w, "Prodigy not found", http.StatusNotFound)
		return
	}
	writeJSON(w, payload)
}

func (s *Server) handleGetTrainingStatus(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	defer s.worldMu.RUnlock()

	writeJSON(w, map[string]interface{}{
		"training_energy":     s.GrowthEngine.TrainingEnergy,
		"max_training_energy": s.GrowthEngine.MaxTrainingEnergy,
	})
}

func (s *Server) handleGetNXGN50(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	rankings := tournament.GenerateNXGN50(s.TournamentManager.ClubsList, s.GrowthEngine)
	payload := map[string]interface{}{
		"status":   "success",
		"rankings": rankings,
	}
	s.worldMu.RUnlock()
	writeJSON(w, payload)
}

func (s *Server) handleGetSuperLeague(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	var standings []map[string]interface{}
	for _, c := range s.TournamentManager.GetStandings() {
		standings = append(standings, s.serializeClub(c))
	}
	payload := map[string]interface{}{
		"clubs":             standings,
		"standings":         standings,
		"season_name":       s.TournamentManager.SeasonName,
		"current_matchweek": s.TournamentManager.CurrentMatchweek,
		"max_matchweeks":    s.TournamentManager.MaxMatchweeks,
		"season_phase":      s.TournamentManager.SeasonPhase,
		"recent_results":    append([]string{}, s.TournamentManager.RecentResults...),
	}
	s.worldMu.RUnlock()
	writeJSON(w, payload)
}

func (s *Server) handleGetCalendar(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	calendar := s.TournamentManager.GetCalendar()
	s.worldMu.RUnlock()
	writeJSON(w, calendar)
}

func (s *Server) handleGetFixtures(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	mw := s.TournamentManager.CurrentMatchweek
	if mwStr := r.URL.Query().Get("matchweek"); mwStr != "" {
		if parsed, err := strconv.Atoi(mwStr); err == nil {
			mw = parsed
		}
	}
	slate := s.TournamentManager.GetSlate(mw)
	serialized := make([]map[string]interface{}, 0, len(slate))
	for i := range slate {
		serialized = append(serialized, s.serializeFixture(&slate[i]))
	}
	payload := map[string]interface{}{
		"current_matchweek": s.TournamentManager.CurrentMatchweek,
		"max_matchweeks":    s.TournamentManager.MaxMatchweeks,
		"season_phase":      s.TournamentManager.SeasonPhase,
		"season_name":       s.TournamentManager.SeasonName,
		"matchweek":         mw,
		"fixtures":          serialized,
		"ucl_pending_ids":   s.TournamentManager.PendingUCL(),
	}
	s.worldMu.RUnlock()
	writeJSON(w, payload)
}

func (s *Server) handleGetFixture(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	fid := r.PathValue("fixture_id")
	var payload map[string]interface{}
	if f := s.TournamentManager.FindFixture(fid); f != nil {
		payload = s.serializeFixture(f)
	}
	s.worldMu.RUnlock()

	if payload == nil {
		http.Error(w, "Fixture not found", http.StatusNotFound)
		return
	}
	writeJSON(w, payload)
}

func (s *Server) handleSimulateFixture(w http.ResponseWriter, r *http.Request) {
	s.worldMu.Lock()
	held := true
	defer func() {
		if held {
			s.worldMu.Unlock()
		}
	}()

	fid := r.PathValue("fixture_id")
	res := s.TournamentManager.SimulateFixture(fid)
	if res["status"] == "error" {
		msg, _ := res["message"].(string)
		if msg == "" {
			msg = "Could not simulate."
		}
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	snap, gen := s.takeCareerSnapshotLocked()
	s.worldMu.Unlock()
	held = false
	s.commitCareerSnapshot(snap, gen)
	writeJSON(w, res)
}

func (s *Server) handleSimulateRemaining(w http.ResponseWriter, r *http.Request) {
	s.worldMu.Lock()
	held := true
	defer func() {
		if held {
			s.worldMu.Unlock()
		}
	}()

	// Optional watch-one exclusion: the live-committed fixture must never be
	// replayed. Empty bodies (legacy callers) simulate the whole slate.
	var req struct {
		ExcludeFixtureID string `json:"exclude_fixture_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	exclude := req.ExcludeFixtureID
	if exclude == "" {
		exclude = r.URL.Query().Get("exclude_fixture_id")
	}
	// Safety net: never resim the currently-selected live fixture, even when
	// the client forgets to pass it.
	if exclude == "" {
		exclude = s.liveFixtureID
	}
	var res map[string]interface{}
	if exclude != "" {
		res = s.TournamentManager.SimulateRemainingExcluding(exclude)
	} else {
		res = s.TournamentManager.SimulateRemaining()
	}
	snap, gen := s.takeCareerSnapshotLocked()
	s.worldMu.Unlock()
	held = false
	s.commitCareerSnapshot(snap, gen)
	writeJSON(w, res)
}

// handleGetFavourite returns the persisted watched club id (may be empty).
func (s *Server) handleGetFavourite(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	defer s.worldMu.RUnlock()
	writeJSON(w, map[string]interface{}{"favourite_club_id": s.TournamentManager.FavouriteClubID})
}

// handleSetFavourite pins the watched club and persists it to the save.
func (s *Server) handleSetFavourite(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ClubID string `json:"club_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.ClubID == "" {
		req.ClubID = r.URL.Query().Get("club_id")
	}
	s.worldMu.Lock()
	ok := false
	if req.ClubID != "" {
		if _, exists := s.TournamentManager.Clubs[req.ClubID]; exists {
			s.TournamentManager.FavouriteClubID = req.ClubID
			ok = true
		}
	}
	snap, gen := s.takeCareerSnapshotLocked()
	fav := s.TournamentManager.FavouriteClubID
	s.worldMu.Unlock()
	if !ok {
		http.Error(w, "Unknown club", http.StatusBadRequest)
		return
	}
	s.commitCareerSnapshot(snap, gen)
	writeJSON(w, map[string]interface{}{"status": "success", "favourite_club_id": fav})
}

// handleWeekWatch resolves the favourite's fixture plus same-week cup jump
// targets. No simulation occurs here.
func (s *Server) handleWeekWatch(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	defer s.worldMu.RUnlock()
	favID, fav, cups, mw := s.TournamentManager.WeekWatch()
	var favPayload interface{}
	if fav != nil {
		favPayload = s.serializeFixture(fav)
	}
	cupPayload := make([]map[string]interface{}, 0, len(cups))
	for i := range cups {
		cupPayload = append(cupPayload, s.serializeFixture(&cups[i]))
	}
	writeJSON(w, map[string]interface{}{
		"favourite_club_id": favID,
		"matchweek":         mw,
		"fixture":           favPayload,
		"same_week_cups":    cupPayload,
	})
}

func (s *Server) handleGetUCL(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	state := s.TournamentManager.GetUCLState()
	groupA, _ := state["group_a"].([]*models.Club)
	groupB, _ := state["group_b"].([]*models.Club)
	records, _ := state["records"].(map[string]*models.CompetitionRecord)
	serGroup := func(clubs []*models.Club) []map[string]interface{} {
		var out []map[string]interface{}
		for _, c := range clubs {
			rec := records[c.ClubID]
			row := s.serializeClubRecord(c, rec)
			row["cup_status"] = s.TournamentManager.UCLGroupStatus(c.ClubID)
			out = append(out, row)
		}
		return out
	}
	payload := map[string]interface{}{
		"stage":          state["stage"],
		"group_a":        serGroup(groupA),
		"group_b":        serGroup(groupB),
		"quarter_finals": s.serializeTieMap(state["quarter_finals"]),
		"semi_finals":    s.serializeTieMap(state["semi_finals"]),
		"final":          s.serializeFinal(state["final"]),
		"champion":       s.serializeClubPtr(state["champion"]),
	}
	s.worldMu.RUnlock()
	writeJSON(w, payload)
}

func (s *Server) handleGetSuperCup(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	state := s.TournamentManager.GetSuperCupState()
	var byes []map[string]interface{}
	if raw, ok := state["byes"].([]*models.Club); ok {
		for _, c := range raw {
			byes = append(byes, s.serializeClub(c))
		}
	}
	payload := map[string]interface{}{
		"stage":          state["stage"],
		"byes":           byes,
		"play_in":        s.serializeTieMap(state["play_in"]),
		"quarter_finals": s.serializeTieMap(state["quarter_finals"]),
		"semi_finals":    s.serializeTieMap(state["semi_finals"]),
		"final":          s.serializeFinal(state["final"]),
		"champion":       s.serializeClubPtr(state["champion"]),
	}
	s.worldMu.RUnlock()
	writeJSON(w, payload)
}

func (s *Server) handleGetUCLFixtures(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	fx := s.TournamentManager.UCLFixturesCopy()
	out := make([]map[string]interface{}, 0, len(fx))
	for i := range fx {
		out = append(out, s.serializeFixture(&fx[i]))
	}
	s.worldMu.RUnlock()
	writeJSON(w, out)
}

func (s *Server) serializeClubPtr(v interface{}) map[string]interface{} {
	c, _ := v.(*models.Club)
	return s.serializeClub(c)
}

func (s *Server) serializeTieMap(v interface{}) map[string]interface{} {
	raw, _ := v.(map[string]interface{})
	out := map[string]interface{}{}
	for k, item := range raw {
		m, _ := item.(map[string]interface{})
		out[k] = map[string]interface{}{
			"home":       s.serializeClubPtr(m["home"]),
			"away":       s.serializeClubPtr(m["away"]),
			"leg1":       m["leg1"],
			"leg2":       m["leg2"],
			"winner":     s.serializeClubPtr(m["winner"]),
			"decided_by": m["decided_by"],
			"penalties":  m["penalties"],
		}
	}
	return out
}

func (s *Server) serializeFinal(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}
	return map[string]interface{}{
		"team1":      s.serializeClubPtr(m["team1"]),
		"team2":      s.serializeClubPtr(m["team2"]),
		"score":      m["score"],
		"winner":     s.serializeClubPtr(m["winner"]),
		"decided_by": m["decided_by"],
		"penalties":  m["penalties"],
	}
}


func (s *Server) handleGetScoringRace(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	var all []*models.Player
	for _, c := range s.TournamentManager.ClubsList {
		all = append(all, c.Squad...)
	}

	sort.Slice(all, func(i, j int) bool {
		if all[i].Goals != all[j].Goals {
			return all[i].Goals > all[j].Goals
		}
		if all[i].Assists != all[j].Assists {
			return all[i].Assists > all[j].Assists
		}
		return all[i].OVR > all[j].OVR
	})

	var race []map[string]interface{}
	for i := 0; i < 5 && i < len(all); i++ {
		p := all[i]
		club := s.TournamentManager.Clubs[p.ClubID]
		cName := p.ClubID
		cShort := ""
		if club != nil {
			cName = club.ClubName
			cShort = club.ShortName
		}
		race = append(race, map[string]interface{}{
			"player_id":    p.PlayerID,
			"full_name":    p.FullName,
			"position":     p.Position,
			"ovr":          p.OVR,
			"goals":        p.Goals,
			"assists":      p.Assists,
			"club_name":    cName,
			"club_short":   cShort,
			"is_wonderkid": p.UniverseWonderkid,
		})
	}
	s.worldMu.RUnlock()
	writeJSON(w, race)
}

func (s *Server) handleGetTrophies(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	cabinet := s.TournamentManager.GetTrophyCabinet()
	s.worldMu.RUnlock()
	writeJSON(w, map[string]interface{}{"status": "success", "cabinet": cabinet})
}

func (s *Server) handleGetRecords(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	payload := map[string]interface{}{
		"status":  "success",
		"records": s.TournamentManager.GetAllTimeRecords(),
	}
	s.worldMu.RUnlock()
	writeJSON(w, payload)
}

func (s *Server) handleGetSeasonAwards(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	awards := s.TournamentManager.GetSeasonAwards()
	s.worldMu.RUnlock()
	writeJSON(w, awards)
}

// handleGetAwardsCeremony serves the gala contract: ranked categories with
// nominees and winners, plus the Ballon d'Or shortlist. Team of the season
// and manager of the year are null until computed; the client guards them.
func (s *Server) handleGetAwardsCeremony(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	ceremony := s.TournamentManager.GetAwardsCeremony()
	awards := s.TournamentManager.GetSeasonAwards()
	s.worldMu.RUnlock()
	ceremony["ballon_dor"] = awards["ballon_dor"]
	ceremony["team_of_the_season"] = awards["team_of_the_season"]
	ceremony["manager_of_the_year"] = awards["manager_of_the_year"]
	writeJSON(w, ceremony)
}

func (s *Server) handleGetSeasonHistory(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	defer s.worldMu.RUnlock()
	writeJSON(w, s.careerHistoryPayload())
}

// careerHistoryPayload builds the History-tab contract: current season
// awards, live table, archived past seasons, trophy cabinet, and records.
// Caller must hold worldMu (TournamentManager methods take their own locks).
func (s *Server) careerHistoryPayload() map[string]interface{} {
	tm := s.TournamentManager
	table := []map[string]interface{}{}
	for _, c := range tm.GetStandings() {
		table = append(table, map[string]interface{}{
			"club_name":  c.ClubName,
			"short_name": c.ShortName,
			"pts":        c.Points,
			"gd":         c.GoalDifference,
			"p":          c.Played,
		})
	}
	past := []map[string]interface{}{}
	past = append(past, tm.SeasonHistory...)
	return map[string]interface{}{
		"season_name":       tm.SeasonName,
		"current_matchweek": tm.CurrentMatchweek,
		"max_matchweeks":    tm.MaxMatchweeks,
		"season_phase":      tm.SeasonPhase,
		"ucl_stage":         tm.UCLStage,
		"super_cup_stage":   tm.SuperCupStage,
		"current":           tm.GetSeasonAwards(),
		"table":             table,
		"past":              past,
		"trophy_cabinet":    tm.GetTrophyCabinet(),
		"all_time_records":  tm.GetAllTimeRecords(),
	}
}

func (s *Server) handleGetSeasonStats(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	stats := s.TournamentManager.GetSeasonStats()
	s.worldMu.RUnlock()
	writeJSON(w, stats)
}

func (s *Server) handleResetSeason(w http.ResponseWriter, r *http.Request) {
	s.worldMu.Lock()
	held := true
	defer func() {
		if held {
			s.worldMu.Unlock()
		}
	}()

	s.clearLiveFixtureSelection()
	s.lastCommittedLiveInstance = -1
	res := s.TournamentManager.ResetNewSeason()
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
	writeJSON(w, s.prodigyDrawPayload(datamanager.ShuffleProdigyHomes(nil)))
}

func (s *Server) handleNewCareer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Shuffle bool              `json:"shuffle"`
		Homes   map[string]string `json:"homes"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	s.worldMu.Lock()
	held := true
	defer func() {
		if held {
			s.worldMu.Unlock()
		}
	}()

	homes := resolveNewCareerHomes(req.Shuffle, req.Homes)
	if err := s.bootFreshCareer(homes, req.Shuffle); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = persistence.DeleteCareer(s.savePath)

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
	return s.TournamentManager != nil && s.TournamentManager.SeasonPhase == "transfer_window"
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
		"negotiation_id": neg.NegotiationID,
		"player":         s.serializePlayer(neg.Player),
		"buyer":          s.serializeClub(neg.Buyer),
		"seller":         s.serializeClub(neg.Seller),
		"current_bid":    neg.CurrentBid,
		"formatted_bid":  models.FormatCurrency(neg.CurrentBid),
		"stage_index":    neg.StageIndex,
		"stage_name":     neg.StageName,
		"progress_pct":   neg.ProgressPct,
		"is_wonderkid":   neg.IsWonderkid,
		"is_hijacked":    neg.IsHijacked,
		"original_buyer": s.serializeClub(neg.OriginalBuyer),
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
		mgr := s.TournamentManager.Managers[club.ClubID]
		if mgr == nil && s.TransferEngine != nil {
			mgr = s.TransferEngine.Managers[club.ClubID]
		}
		if mgr == nil {
			continue
		}
		warchests = append(warchests, map[string]interface{}{
			"club_name":        club.ClubName,
			"club_short":       club.ShortName,
			"manager_name":     mgr.Name,
			"tactic":           mgr.Tactic(),
			"focus":            mgr.FocusLabel(),
			"budget_eur":       mgr.BudgetEur,
			"formatted_budget": models.FormatCurrency(mgr.BudgetEur),
			"wage_bill_eur":    mgr.WageBill(club),
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

	week := 1
	if s.TransferEngine != nil {
		week = s.TransferEngine.CurrentWeek
	}

	return map[string]interface{}{
		"window_name":         careerWindowName(open, week),
		"is_window_open":      open,
		"season_phase":        s.TournamentManager.SeasonPhase,
		"window_day":          s.TransferEngine.CurrentDay,
		"window_week":         s.TransferEngine.CurrentWeek,
		"max_window_weeks":    12,
		"active_negotiations": negs,
		"transfer_feed":       feed,
		"completed_transfers": completed,
		"expiring_contracts":  expiring,
		"warchests":           warchests,
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

	if !s.careerWindowOpen() {
		writeErrorJSON(w, http.StatusBadRequest, "The window opens when the season ends.")
		return
	}

	s.TransferEngine.AdvanceOpenWindow()
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
	s.worldMu.RLock()
	unread := 0
	for _, item := range s.TournamentManager.Inbox {
		if item.Unread {
			unread++
		}
	}
	// Copy the backing array: MarkInboxRead mutates items in place.
	items := append([]tournament.InboxItem{}, s.TournamentManager.Inbox...)
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
	s.worldMu.Lock()
	defer s.worldMu.Unlock()

	var req struct {
		ID     string `json:"id"`
		ItemID string `json:"item_id"`
		All    bool   `json:"all"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
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
	writeJSON(w, map[string]interface{}{
		"status": "success",
		"unread": unread,
		"marked": marked,
	})
}

func (s *Server) handleStaticSPA(w http.ResponseWriter, r *http.Request) {
	if s.staticDir == "" {
		http.NotFound(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/ws/") {
		http.NotFound(w, r)
		return
	}

	rel := strings.TrimPrefix(urlpath.Clean("/"+r.URL.Path), "/")
	if rel == "." {
		rel = ""
	}
	if rel == "" {
		http.ServeFile(w, r, filepath.Join(s.staticDir, "index.html"))
		return
	}
	if strings.HasPrefix(rel, "../") || strings.Contains(rel, "/../") {
		http.NotFound(w, r)
		return
	}

	root, err := filepath.Abs(s.staticDir)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	target, err := filepath.Abs(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	relToRoot, err := filepath.Rel(root, target)
	if err != nil || strings.HasPrefix(relToRoot, "..") {
		http.NotFound(w, r)
		return
	}
	if info, err := os.Stat(target); err == nil && !info.IsDir() {
		http.ServeFile(w, r, target)
		return
	}

	indexPath := filepath.Join(root, "index.html")
	if _, err := os.Stat(indexPath); err == nil {
		http.ServeFile(w, r, indexPath)
		return
	}
	http.NotFound(w, r)
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}

func writeErrorJSON(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"detail": detail})
}
