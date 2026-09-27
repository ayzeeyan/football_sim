package server

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"time"

	"football_sim/pkg/matchengine"
	"football_sim/pkg/tournament"

	"github.com/gorilla/websocket"
)

// Live match delivery: /ws/match streaming, tick payloads and broadcasts.
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
			Action    string `json:"action"`
			HomeID    string `json:"home_id"`
			AwayID    string `json:"away_id"`
			FixtureID string `json:"fixture_id"`
			Speed     int    `json:"speed"`
			Side      string `json:"side"`
			Stance    string `json:"stance"`
			OutID     string `json:"out_id"`
			InID      string `json:"in_id"`
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
			// A refresh reconnects the browser and repeats its current selection.
			// Treat that as a resubscribe while the same fixture is live; calling
			// SetClubs here would reset the clock, score and events back to 0'.
			sameLiveFixture := cmd.FixtureID != "" && cmd.FixtureID == s.liveFixtureID
			sameLiveClubs := s.LiveMatchEngine.HomeClub != nil && s.LiveMatchEngine.AwayClub != nil &&
				s.LiveMatchEngine.HomeClub.ClubID == cmd.HomeID && s.LiveMatchEngine.AwayClub.ClubID == cmd.AwayID
			liveState := s.LiveMatchEngine.State == "PLAYING" || s.LiveMatchEngine.State == "PAUSED" ||
				s.LiveMatchEngine.State == "HALF_TIME" || s.LiveMatchEngine.State == "GOAL_PAUSE"
			if sameLiveFixture && sameLiveClubs && liveState {
				s.wsForceFull = true
				break
			}
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
				var f *tournament.Fixture
				if cmd.FixtureID != "" {
					if exact := s.TournamentManager.FindFixture(cmd.FixtureID); exact != nil &&
						exact.Status == "scheduled" && exact.Matchweek == s.TournamentManager.CurrentMatchweek &&
						exact.HomeID == cmd.HomeID && exact.AwayID == cmd.AwayID {
						f = exact
					}
				} else {
					f = s.findLiveFixture(cmd.HomeID, cmd.AwayID)
				}
				if f != nil {
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
		case "halftime_resume_ai":
			// Spectator mode has no user-controlled dugout. Both AI managers
			// make their interval decisions and play resumes automatically.
			if s.LiveMatchEngine.ResumeHalfTimeAI() {
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
		"home_formation":        e.HomeFormation,
		"away_formation":        e.AwayFormation,
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
	// A disconnected or suspended browser must never be able to block the
	// single live ticker forever. The next failed broadcast unregisters it;
	// the client reconnects and requests a fresh full snapshot.
	if err := conn.SetWriteDeadline(time.Now().Add(2 * time.Second)); err != nil {
		return err
	}
	return conn.WriteJSON(payload)
}

func (s *Server) writeTickBytes(conn *websocket.Conn, data []byte) error {
	if err := conn.SetWriteDeadline(time.Now().Add(2 * time.Second)); err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, data)
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
	if len(clients) == 0 {
		return
	}

	var failed []*websocket.Conn
	// Marshal once so every client shares a single JSON encoding.
	// Test hook path still receives the raw payload object.
	if s.wsWrite != nil {
		for _, conn := range clients {
			if err := s.writeMatchTick(conn, payload); err != nil {
				failed = append(failed, conn)
			}
		}
	} else {
		data, err := json.Marshal(payload)
		if err != nil {
			return
		}
		for _, conn := range clients {
			if err := s.writeTickBytes(conn, data); err != nil {
				failed = append(failed, conn)
			}
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
