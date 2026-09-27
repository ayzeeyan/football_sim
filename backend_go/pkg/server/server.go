package server

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	urlpath "path"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"football_sim/pkg/datamanager"
	"football_sim/pkg/growth"
	"football_sim/pkg/matchengine"
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

var errFreshCareer = errors.New("could not rebuild the European world from dataset.json")

// LiveMatchStreaming exists only for legacy engine contract tests. The product
// is simulation-only, so no WebSocket route, ticker or live engine is started.
var LiveMatchStreaming = false

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

	var liveEngine *matchengine.LiveMatchEngine
	if LiveMatchStreaming && tm != nil && len(tm.ClubsList) >= 2 {
		homeClub, awayClub := tm.ClubsList[0], tm.ClubsList[1]
		liveEngine = matchengine.NewLiveMatchEngine(homeClub, awayClub,
			tm.Managers[homeClub.ClubID], tm.Managers[awayClub.ClubID], time.Now().UnixNano())
	}

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
	if LiveMatchStreaming {
		s.StartLiveTicker()
	}
	return s
}

// gzipResponseWriter compresses handler bodies when the client accepts gzip.
type gzipResponseWriter struct {
	http.ResponseWriter
	gz *gzip.Writer
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.gz.Write(b)
}

func (w *gzipResponseWriter) WriteHeader(status int) {
	w.Header().Del("Content-Length")
	w.ResponseWriter.WriteHeader(status)
}

func acceptsGzip(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
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

	// WebSocket upgrades and range requests must pass through unencoded.
	if strings.HasPrefix(r.URL.Path, "/ws/") || r.Header.Get("Range") != "" || !acceptsGzip(r) {
		s.mux.ServeHTTP(w, r)
		return
	}

	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Add("Vary", "Accept-Encoding")
	gz := gzip.NewWriter(w)
	defer gz.Close()
	s.mux.ServeHTTP(&gzipResponseWriter{ResponseWriter: w, gz: gz}, r)
}

func (s *Server) setupRoutes() {
	// Retired live endpoint. Legacy tests opt into the handler explicitly.
	if LiveMatchStreaming {
		s.mux.HandleFunc("/ws/match", s.handleWebSocketMatch)
	} else {
		s.mux.HandleFunc("/ws/match", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "live matches have been removed; use fixture simulation", http.StatusGone)
		})
	}

	// Health & System
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/stats", s.handleStats)

	// Clubs & Rosters
	s.mux.HandleFunc("GET /api/clubs", s.handleGetClubs)
	s.mux.HandleFunc("GET /api/clubs/{club_id}/squad", s.handleGetClubSquad)
	s.mux.HandleFunc("GET /api/clubs/{club_id}/xi", s.handleGetClubXI)
	s.mux.HandleFunc("GET /api/clubs/{club_id}/history", s.handleGetClubHistory)
	s.mux.HandleFunc("GET /api/clubs/{club_id}/profile", s.handleGetClubProfile)
	s.mux.HandleFunc("GET /api/clubs/{club_id}/fixtures", s.handleGetClubFixtures)
	s.mux.HandleFunc("GET /api/clubs/{club_id}/transfers", s.handleGetClubTransfers)
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
	s.mux.HandleFunc("GET /api/competitions", s.handleGetCompetitions)
	s.mux.HandleFunc("GET /api/competitions/{competition_id}", s.handleGetCompetition)
	s.mux.HandleFunc("GET /api/competitions/nations-cup/fixtures/{fixture_id}", s.handleGetNationsFixture)
	s.mux.HandleFunc("POST /api/competitions/nations-cup/fixtures/{fixture_id}/simulate", s.handleSimulateNationsFixture)
	s.mux.HandleFunc("GET /api/ucl/fixtures", s.handleGetUCLFixtures)
	s.mux.HandleFunc("GET /api/calendar", s.handleGetCalendar)
	s.mux.HandleFunc("GET /api/fixtures", s.handleGetFixtures)
	s.mux.HandleFunc("GET /api/fixtures/{fixture_id}", s.handleGetFixture)
	s.mux.HandleFunc("POST /api/fixtures/{fixture_id}/simulate", s.handleSimulateFixture)
	s.mux.HandleFunc("POST /api/fixtures/simulate-remaining", s.handleSimulateRemaining)
	s.mux.HandleFunc("POST /api/sim/week", s.handleSimWeek)
	s.mux.HandleFunc("POST /api/sim/month", s.handleSimMonth)
	s.mux.HandleFunc("POST /api/sim/season", s.handleSimSeason)
	s.mux.HandleFunc("POST /api/sim/continue", s.handleSimContinue)
	s.mux.HandleFunc("GET /api/world/dashboard", s.handleGetWorldDashboard)
	s.mux.HandleFunc("GET /api/search", s.handleSearchWorld)
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
	s.mux.HandleFunc("POST /api/inbox/reply", s.handleInboxReply)

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
		idleTicks := 0

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
				// Idle states (pre-match / full time / paused) do not need a
				// 60Hz blast: clients only poll for status. Keep FULL_TIME
				// commit ticking but publish at ~2Hz to cut scroll-jank load.
				state := s.LiveMatchEngine.State
				idle := state == "NOT_STARTED" || state == "FULL_TIME" || state == "PAUSED"
				if idle {
					idleTicks++
					if idleTicks%30 != 0 {
						s.worldMu.Unlock()
						s.commitCareerSnapshot(snap, gen)
						continue
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
		w.Header().Set("Cache-Control", "no-cache")
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
		if strings.HasPrefix(rel, "assets/") {
			// Vite fingerprints asset filenames; safe to cache immutably.
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		http.ServeFile(w, r, target)
		return
	}

	indexPath := filepath.Join(root, "index.html")
	if _, err := os.Stat(indexPath); err == nil {
		w.Header().Set("Cache-Control", "no-cache")
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
