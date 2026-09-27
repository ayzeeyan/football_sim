package server

import (
	"net/http"
	"regexp"
	"sort"
	"strings"
)

// Route describes one registered API endpoint. The table is the single
// source of truth for routing, documentation, and the OpenAPI spec: every
// entry is registered on the mux, must appear in README.md's API list, and
// is emitted into /api/openapi.json.
type Route struct {
	Method  string
	Path    string
	Handler func(http.ResponseWriter, *http.Request)
	Summary string
}

// apiRoutes returns the full API route table in stable, documented order.
// Handlers are method values on the server, so the table is per-instance.
func (s *Server) apiRoutes() []Route {
	return []Route{
		// Health & System
		{Method: "GET", Path: "/api/health", Handler: s.handleHealth, Summary: "Liveness probe"},
		{Method: "GET", Path: "/api/stats", Handler: s.handleStats, Summary: "Diagnostics snapshot"},

		// Clubs & Rosters
		{Method: "GET", Path: "/api/clubs", Handler: s.handleGetClubs, Summary: "All clubs"},
		{Method: "GET", Path: "/api/clubs/{club_id}/squad", Handler: s.handleGetClubSquad, Summary: "Club squad"},
		{Method: "GET", Path: "/api/clubs/{club_id}/xi", Handler: s.handleGetClubXI, Summary: "Club probable starting XI"},
		{Method: "GET", Path: "/api/clubs/{club_id}/history", Handler: s.handleGetClubHistory, Summary: "Club season history"},
		{Method: "GET", Path: "/api/clubs/{club_id}/profile", Handler: s.handleGetClubProfile, Summary: "Club identity and finances"},
		{Method: "GET", Path: "/api/clubs/{club_id}/fixtures", Handler: s.handleGetClubFixtures, Summary: "Club fixture list"},
		{Method: "GET", Path: "/api/clubs/{club_id}/transfers", Handler: s.handleGetClubTransfers, Summary: "Club transfer activity"},
		{Method: "GET", Path: "/api/clubs/{club_id}/scouting", Handler: s.handleGetClubScouting, Summary: "AI recruitment shortlist for one club (?limit=)"},
		{Method: "GET", Path: "/api/h2h/{club_a}/{club_b}", Handler: s.handleGetH2H, Summary: "Head-to-head record"},
		{Method: "GET", Path: "/api/players/{player_id}", Handler: s.handleGetPlayerProfile, Summary: "Player profile"},

		// Wonderkids & Growth
		{Method: "GET", Path: "/api/prodigies", Handler: s.handleGetProdigies, Summary: "Franchise prodigies"},
		{Method: "GET", Path: "/api/prodigies/watch", Handler: s.handleGetProdigyWatch, Summary: "Prodigy watch list"},
		{Method: "GET", Path: "/api/wonderkids", Handler: s.handleGetWonderkids, Summary: "Wonderkids (legacy alias of /api/prodigies)"},
		{Method: "POST", Path: "/api/prodigies/{player_id}/train", Handler: s.handleTrainProdigy, Summary: "Run one training regimen"},
		{Method: "POST", Path: "/api/prodigies/{player_id}/position-path", Handler: s.handleSetPositionPath, Summary: "Set a prodigy's position path"},
		{Method: "POST", Path: "/api/prodigies/{player_id}/school-track", Handler: s.handleSetSchoolTrack, Summary: "Set a prodigy's education track"},
		{Method: "GET", Path: "/api/growth/milestones", Handler: s.handleGetGrowthMilestones, Summary: "Growth milestone ledger"},
		{Method: "GET", Path: "/api/prodigies/{player_id}/timeline", Handler: s.handleGetProdigyTimeline, Summary: "Prodigy development timeline"},
		{Method: "GET", Path: "/api/training/status", Handler: s.handleGetTrainingStatus, Summary: "Weekly training energy"},
		{Method: "GET", Path: "/api/training/projection/{player_id}", Handler: s.handleGetTrainingProjection, Summary: "Read-only staff training projection for any player"},
		{Method: "GET", Path: "/api/nxgn50", Handler: s.handleGetNXGN50, Summary: "NXGN 50 wonderkid rankings"},

		// Competitions & Calendar
		{Method: "GET", Path: "/api/super-league", Handler: s.handleGetSuperLeague, Summary: "Domestic-league standings (optional ?league= selector)"},
		{Method: "GET", Path: "/api/ucl", Handler: s.handleGetUCL, Summary: "Champions League state"},
		{Method: "GET", Path: "/api/super-cup", Handler: s.handleGetSuperCup, Summary: "Super Cup state"},
		{Method: "GET", Path: "/api/competitions", Handler: s.handleGetCompetitions, Summary: "Competition hub"},
		{Method: "GET", Path: "/api/competitions/{competition_id}", Handler: s.handleGetCompetition, Summary: "One competition"},
		{Method: "GET", Path: "/api/competitions/nations-cup", Handler: s.handleGetNationsCup, Summary: "European Nations Cup (explicit alias of the competition endpoint)"},
		{Method: "GET", Path: "/api/competitions/nations-cup/fixtures/{fixture_id}", Handler: s.handleGetNationsFixture, Summary: "One national-team fixture"},
		{Method: "POST", Path: "/api/competitions/nations-cup/fixtures/{fixture_id}/simulate", Handler: s.handleSimulateNationsFixture, Summary: "Simulate one national-team fixture"},
		{Method: "GET", Path: "/api/ucl/fixtures", Handler: s.handleGetUCLFixtures, Summary: "Champions League fixtures"},
		{Method: "GET", Path: "/api/calendar", Handler: s.handleGetCalendar, Summary: "Season calendar"},
		{Method: "GET", Path: "/api/fixtures", Handler: s.handleGetFixtures, Summary: "Matchweek fixture summaries"},
		{Method: "GET", Path: "/api/fixtures/{fixture_id}", Handler: s.handleGetFixture, Summary: "Full fixture with report"},
		{Method: "POST", Path: "/api/fixtures/{fixture_id}/simulate", Handler: s.handleSimulateFixture, Summary: "Simulate one fixture"},
		{Method: "GET", Path: "/api/fixtures/{fixture_id}/whatif", Handler: s.handleGetWhatIf, Summary: "Read-only what-if replay of one fixture under a scratch seed (?seed=)"},
		{Method: "POST", Path: "/api/fixtures/simulate-remaining", Handler: s.handleSimulateRemaining, Summary: "Simulate the whole slate"},
		{Method: "POST", Path: "/api/sim/week", Handler: s.handleSimWeek, Summary: "Advance one matchweek"},
		{Method: "POST", Path: "/api/sim/month", Handler: s.handleSimMonth, Summary: "Advance one month"},
		{Method: "POST", Path: "/api/sim/season", Handler: s.handleSimSeason, Summary: "Advance to season end"},
		{Method: "POST", Path: "/api/sim/continue", Handler: s.handleSimContinue, Summary: "Continue the world"},
		{Method: "GET", Path: "/api/world/dashboard", Handler: s.handleGetWorldDashboard, Summary: "World dashboard digest"},
		{Method: "GET", Path: "/api/search", Handler: s.handleSearchWorld, Summary: "Global search"},
		{Method: "GET", Path: "/api/scoring-race", Handler: s.handleGetScoringRace, Summary: "Scoring race"},
		{Method: "GET", Path: "/api/trophies", Handler: s.handleGetTrophies, Summary: "Trophy cabinet"},
		{Method: "GET", Path: "/api/records", Handler: s.handleGetRecords, Summary: "All-time records"},
		{Method: "GET", Path: "/api/export/standings", Handler: s.handleExportStandings, Summary: "CSV export of one domestic league table (?league=)"},
		{Method: "GET", Path: "/api/export/squad", Handler: s.handleExportSquad, Summary: "CSV export of one club squad (?club_id=)"},
		{Method: "GET", Path: "/api/export/fixtures", Handler: s.handleExportFixtures, Summary: "CSV export of every fixture with its scoreline"},
		{Method: "GET", Path: "/api/export/transfers", Handler: s.handleExportTransfers, Summary: "CSV export of the completed-transfer ledger"},
		{Method: "GET", Path: "/api/season/awards", Handler: s.handleGetSeasonAwards, Summary: "Season awards"},
		{Method: "GET", Path: "/api/season/awards/ceremony", Handler: s.handleGetAwardsCeremony, Summary: "Awards ceremony"},
		{Method: "GET", Path: "/api/season/history", Handler: s.handleGetSeasonHistory, Summary: "Season history"},
		{Method: "GET", Path: "/api/season/stats", Handler: s.handleGetSeasonStats, Summary: "Season statistics"},
		{Method: "GET", Path: "/api/season/stats/advanced", Handler: s.handleGetAdvancedSeasonStats, Summary: "Advanced statistics centre: player percentiles, shot zones, club trends"},
		{Method: "POST", Path: "/api/season/reset", Handler: s.handleResetSeason, Summary: "Archive and start the next season"},
		{Method: "POST", Path: "/api/season/restart", Handler: s.handleRestartSeason, Summary: "Restart the current season from matchweek 1"},

		// Career Management
		{Method: "GET", Path: "/api/career/default-homes", Handler: s.handleGetDefaultHomes, Summary: "Default prodigy homes"},
		{Method: "GET", Path: "/api/career/preview-shuffle", Handler: s.handlePreviewShuffle, Summary: "Preview a prodigy home shuffle"},
		{Method: "POST", Path: "/api/career/new", Handler: s.handleNewCareer, Summary: "Start a new career"},
		{Method: "GET", Path: "/api/favourite", Handler: s.handleGetFavourite, Summary: "Observational favourite club"},
		{Method: "POST", Path: "/api/favourite", Handler: s.handleSetFavourite, Summary: "Set the observational favourite club"},
		{Method: "GET", Path: "/api/week/watch", Handler: s.handleWeekWatch, Summary: "Matchweek watch digest"},
		{Method: "GET", Path: "/api/watchlist", Handler: s.handleGetWatchlist, Summary: "Multi-entity watchlist (observational)"},
		{Method: "POST", Path: "/api/watchlist", Handler: s.handleToggleWatchlist, Summary: "Add or remove one watchlist entity"},

		// Transfer Market
		{Method: "GET", Path: "/api/transfers", Handler: s.handleGetTransfers, Summary: "Transfer market state"},
		{Method: "POST", Path: "/api/transfers/bid", Handler: s.handleTransferBid, Summary: "Submit a transfer bid"},
		{Method: "POST", Path: "/api/transfers/advance", Handler: s.handleTransferAdvance, Summary: "Advance the window"},
		{Method: "GET", Path: "/api/transfers/records", Handler: s.handleGetTransferRecords, Summary: "Transfer records"},

		// News Wire & Inbox
		{Method: "GET", Path: "/api/inbox", Handler: s.handleGetInbox, Summary: "Inbox feed"},
		{Method: "POST", Path: "/api/inbox/read", Handler: s.handleMarkInboxRead, Summary: "Mark inbox items read"},
		{Method: "POST", Path: "/api/inbox/reply", Handler: s.handleInboxReply, Summary: "Reply to a conversation"},

		// API self-description
		{Method: "GET", Path: "/api/openapi.json", Handler: s.handleGetOpenAPISpec, Summary: "OpenAPI 3.1 spec generated from the route table"},
	}
}

var pathParamPattern = regexp.MustCompile(`\{([^}]+)\}`)

// openAPIPathParams extracts {name} path parameters in order.
func openAPIPathParams(path string) []string {
	matches := pathParamPattern.FindAllStringSubmatch(path, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, m[1])
	}
	return out
}

// BuildOpenAPISpec emits an OpenAPI 3.1 document from a route table.
// Paths and parameters are derived; operation summaries come from the table.
func BuildOpenAPISpec(routes []Route) map[string]interface{} {
	paths := make(map[string]interface{}, len(routes))
	for _, route := range routes {
		params := make([]map[string]interface{}, 0)
		for _, name := range openAPIPathParams(route.Path) {
			params = append(params, map[string]interface{}{
				"name":     name,
				"in":       "path",
				"required": true,
				"schema":   map[string]interface{}{"type": "string"},
			})
		}
		operation := map[string]interface{}{
			"summary": route.Summary,
		}
		if len(params) > 0 {
			operation["parameters"] = params
		}
		openAPIPath := pathParamPattern.ReplaceAllString(route.Path, "{$1}")
		existing, _ := paths[openAPIPath].(map[string]interface{})
		if existing == nil {
			existing = map[string]interface{}{}
			paths[openAPIPath] = existing
		}
		existing[strings.ToLower(route.Method)] = operation
	}
	return map[string]interface{}{
		"openapi": "3.1.0",
		"info": map[string]interface{}{
			"title":   "Top Five European Football Sim API",
			"version": "1.0",
		},
		"paths": paths,
	}
}

// handleGetNationsCup serves the documented explicit alias for the European
// Nations Cup. The literal pattern has no {competition_id} wildcard, so the
// ID is pinned here instead of read from the path.
func (s *Server) handleGetNationsCup(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	payload := s.TournamentManager.GetCompetition("nations-cup")
	s.worldMu.RUnlock()
	if payload == nil {
		http.Error(w, "Competition not found", http.StatusNotFound)
		return
	}
	writeJSON(w, payload)
}

// handleGetOpenAPISpec serves the spec generated from the live route table.
func (s *Server) handleGetOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, BuildOpenAPISpec(s.apiRoutes()))
}

// sortedRouteKeys returns "METHOD path" keys for stable iteration in tests.
func sortedRouteKeys(routes []Route) []string {
	keys := make([]string, 0, len(routes))
	for _, route := range routes {
		keys = append(keys, route.Method+" "+route.Path)
	}
	sort.Strings(keys)
	return keys
}
