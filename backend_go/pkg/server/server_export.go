package server

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"football_sim/pkg/tournament"
)

// User-initiated data export (Phase 3 F7). Exports are read-only views of
// resolved state served as CSV attachments; nothing is written to disk and
// runtime saves are never touched. Ordering is deterministic.

// csvEscape quotes a CSV field when it contains a separator, quote, or
// newline.
func csvEscape(value string) string {
	if strings.ContainsAny(value, ",\"\n\r") {
		return "\"" + strings.ReplaceAll(value, "\"", "\"\"") + "\""
	}
	return value
}

func writeCSV(w http.ResponseWriter, filename string, header []string, rows [][]string) {
	var b strings.Builder
	b.WriteString(strings.Join(header, ","))
	b.WriteString("\r\n")
	for _, row := range rows {
		cells := make([]string, len(row))
		for i, cell := range row {
			cells[i] = csvEscape(cell)
		}
		b.WriteString(strings.Join(cells, ","))
		b.WriteString("\r\n")
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

// handleExportStandings exports one domestic league table as CSV.
func (s *Server) handleExportStandings(w http.ResponseWriter, r *http.Request) {
	league := r.URL.Query().Get("league")
	s.worldMu.RLock()
	clubs := s.TournamentManager.GetStandingsForLeague(league)
	season := s.TournamentManager.SeasonName
	s.worldMu.RUnlock()

	header := []string{"position", "club_id", "club_name", "league", "played", "won", "drawn", "lost", "goals_for", "goals_against", "goal_difference", "points"}
	rows := make([][]string, 0, len(clubs))
	for i, club := range clubs {
		rows = append(rows, []string{
			fmt.Sprint(i + 1), club.ClubID, club.ClubName, club.League,
			fmt.Sprint(club.Played), fmt.Sprint(club.Won), fmt.Sprint(club.Drawn), fmt.Sprint(club.Lost),
			fmt.Sprint(club.GoalsFor), fmt.Sprint(club.GoalsAgainst), fmt.Sprint(club.GoalDifference), fmt.Sprint(club.Points),
		})
	}
	writeCSV(w, "standings-"+strings.ReplaceAll(season, "/", "-")+".csv", header, rows)
}

// handleExportSquad exports one club's squad ledger as CSV.
func (s *Server) handleExportSquad(w http.ResponseWriter, r *http.Request) {
	clubID := r.URL.Query().Get("club_id")
	s.worldMu.RLock()
	club := s.TournamentManager.Clubs[clubID]
	s.worldMu.RUnlock()
	if club == nil {
		http.Error(w, "Club not found", http.StatusNotFound)
		return
	}
	header := []string{"player_id", "full_name", "position", "category", "age", "ovr", "appearances", "goals", "assists", "morale", "fitness", "value_eur"}
	rows := make([][]string, 0, len(club.Squad))
	for _, p := range club.Squad {
		if p == nil {
			continue
		}
		rows = append(rows, []string{
			p.PlayerID, p.FullName, p.Position, p.Category, fmt.Sprint(p.Age), fmt.Sprint(p.OVR),
			fmt.Sprint(p.Appearances), fmt.Sprint(p.Goals), fmt.Sprint(p.Assists),
			fmt.Sprint(p.Morale), fmt.Sprint(p.Fitness), fmt.Sprint(p.MarketValueEUR),
		})
	}
	writeCSV(w, "squad-"+clubID+".csv", header, rows)
}

// handleExportFixtures exports every fixture with its status and scoreline,
// sorted by matchweek then fixture id for a stable file.
func (s *Server) handleExportFixtures(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	season := s.TournamentManager.SeasonName
	type row struct {
		mw, id, comp, stage, home, away, status, score string
	}
	rows := make([]row, 0, 512)
	scan := func(pool []tournament.Fixture) {
		for i := range pool {
			f := &pool[i]
			score := ""
			if f.HomeGoals != nil && f.AwayGoals != nil {
				score = fmt.Sprintf("%d-%d", *f.HomeGoals, *f.AwayGoals)
			}
			rows = append(rows, row{
				mw: fmt.Sprint(f.Matchweek), id: f.FixtureID, comp: f.Competition, stage: f.Stage,
				home: f.HomeID, away: f.AwayID, status: f.Status, score: score,
			})
		}
	}
	scan(s.TournamentManager.Fixtures)
	scan(s.TournamentManager.UCLFixtures)
	scan(s.TournamentManager.SuperCupFixtures)
	if s.TournamentManager.World != nil {
		scan(s.TournamentManager.World.Fixtures)
	}
	s.worldMu.RUnlock()

	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].mw != rows[j].mw {
			return rows[i].mw < rows[j].mw
		}
		return rows[i].id < rows[j].id
	})
	header := []string{"matchweek", "fixture_id", "competition", "stage", "home_club_id", "away_club_id", "status", "score"}
	out := make([][]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, []string{row.mw, row.id, row.comp, row.stage, row.home, row.away, row.status, row.score})
	}
	writeCSV(w, "fixtures-"+strings.ReplaceAll(season, "/", "-")+".csv", header, out)
}

// handleExportTransfers exports the full completed-transfer ledger as CSV.
func (s *Server) handleExportTransfers(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	all := s.TransferEngine.AllTransfers()
	s.worldMu.RUnlock()

	header := []string{"player_id", "player_name", "position", "seller_club", "buyer_club", "fee_eur", "matchweek"}
	rows := make([][]string, 0, len(all))
	for _, t := range all {
		rows = append(rows, []string{
			t.PlayerID, t.PlayerName, t.PlayerPos, t.SellerName, t.BuyerName,
			fmt.Sprint(t.FeeEUR), fmt.Sprint(t.Matchweek), "",
		})
	}
	writeCSV(w, "transfers.csv", header, rows)
}
