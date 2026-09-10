package tournament

import (
	"math/rand"
	"testing"

	"football_sim/pkg/managers"
	"football_sim/pkg/matchengine"
	"football_sim/pkg/matchreport"
	"football_sim/pkg/models"
)

func packet3LiveTestClub(id string, ovr int) *models.Club {
	positions := []string{"GK", "LB", "CB", "CB", "RB", "CDM", "CM", "CM", "LW", "ST", "RW"}
	categories := []string{"GK", "DEF", "DEF", "DEF", "DEF", "MID", "MID", "MID", "FWD", "FWD", "FWD"}
	squad := make([]*models.Player, 0, len(positions)+1)
	for i := range positions {
		squad = append(squad, &models.Player{
			PlayerID: id + "-" + string(rune('A'+i)), FullName: id + " Player " + string(rune('A'+i)),
			Position: positions[i], Category: categories[i], OVR: ovr,
			Age: 25, ClubID: id, MarketValueEUR: 10000000,
		})
	}
	squad = append(squad, &models.Player{
		PlayerID: id + "-SUB", FullName: id + " Substitute", Position: "CM", Category: "MID", OVR: ovr - 2,
		Age: 22, ClubID: id, MarketValueEUR: 5000000,
	})
	return &models.Club{
		ClubID: id, ClubName: id + " FC", ShortName: id, HomeStadium: id + " Park",
		OverallTeamRating: ovr, Morale: 70, StadiumCapacity: 40000, Squad: squad,
	}
}

func packet3SeedWithRoll(t *testing.T, low, high float64) int64 {
	t.Helper()
	for seed := int64(0); seed < 100000; seed++ {
		roll := rand.New(rand.NewSource(seed)).Float64()
		if roll >= low && roll < high {
			return seed
		}
	}
	t.Fatalf("could not find deterministic random seed for roll [%v,%v)", low, high)
	return 0
}

func packet3FindRow(rows []matchreport.MatchPlayerRow, playerID string) *matchreport.MatchPlayerRow {
	for i := range rows {
		if rows[i].PlayerID == playerID {
			return &rows[i]
		}
	}
	return nil
}

func TestPacket3LiveRedCardsApplyOnceThroughReport(t *testing.T) {
	home := packet3LiveTestClub("HOME", 80)
	away := packet3LiveTestClub("AWAY", 78)
	engine := matchengine.NewLiveMatchEngine(
		home, away,
		&managers.ManagerProfile{ClubID: home.ClubID, Name: "Home Manager", Style: "possession"},
		&managers.ManagerProfile{ClubID: away.ClubID, Name: "Away Manager", Style: "possession"},
		42,
	)
	engine.ResetMatch()
	engine.StartKickoff()
	engine.PossessionTeam = "home" // away players are the defending card pool

	if len(engine.AwayStarters) < 5 {
		t.Fatalf("expected five away starters, got %d", len(engine.AwayStarters))
	}
	normalYellow := engine.AwayStarters[2]
	secondYellow := engine.AwayStarters[3]
	straightRed := engine.AwayStarters[4]

	// Generate an ordinary yellow, then a second yellow for a different player.
	engine.CurrentMinute = 10
	engine.RNG = rand.New(rand.NewSource(packet3SeedWithRoll(t, 0.012, 0.085)))
	engine.MaybeBookPlayer([]*models.Player{normalYellow})
	if len(engine.Events) == 0 || engine.Events[len(engine.Events)-1].Type != "yellow" {
		t.Fatalf("ordinary yellow was not generated: %+v", engine.Events)
	}

	engine.CurrentMinute = 20
	engine.RNG = rand.New(rand.NewSource(packet3SeedWithRoll(t, 0, 0.085)))
	engine.Bookings[secondYellow.PlayerID] = 1
	engine.MaybeBookPlayer([]*models.Player{secondYellow})
	secondEvent := engine.Events[len(engine.Events)-1]
	if secondEvent.Type != "red" || !secondEvent.SentOff || secondEvent.Detail != "second_yellow" {
		t.Fatalf("second-yellow red event wrong: %+v", secondEvent)
	}

	engine.CurrentMinute = 30
	engine.RNG = rand.New(rand.NewSource(packet3SeedWithRoll(t, 0, 0.012)))
	engine.MaybeBookPlayer([]*models.Player{straightRed})
	straightEvent := engine.Events[len(engine.Events)-1]
	if straightEvent.Type != "red" || !straightEvent.SentOff || straightEvent.Detail != "straight_red" {
		t.Fatalf("straight-red event wrong: %+v", straightEvent)
	}
	if len(engine.OnPitch(engine.AwayStarters)) != 9 {
		t.Fatalf("dismissed players remained active: %d", len(engine.OnPitch(engine.AwayStarters)))
	}

	payload := engine.BuildLivePayload()
	report := matchreport.AssembleReport(payload, "live", engine.RNG)

	secondRow := packet3FindRow(report.AwayXI, secondYellow.PlayerID)
	straightRow := packet3FindRow(report.AwayXI, straightRed.PlayerID)
	normalRow := packet3FindRow(report.AwayXI, normalYellow.PlayerID)
	for name, row := range map[string]*matchreport.MatchPlayerRow{
		"second yellow": secondRow, "straight red": straightRow, "normal yellow": normalRow,
	} {
		if row == nil {
			t.Fatalf("%s missing from assembled report", name)
		}
	}
	for name, row := range map[string]*matchreport.MatchPlayerRow{"second yellow": secondRow, "straight red": straightRow} {
		if row.Card == nil || *row.Card != "red" || row.OffMinute == nil || *row.OffMinute <= 0 || row.Minutes >= 90 {
			t.Fatalf("%s row lacks red/off-minute/reduced appearance: %+v", name, *row)
		}
	}
	if normalRow.Card == nil || *normalRow.Card != "yellow" || normalRow.OffMinute != nil {
		t.Fatalf("ordinary yellow row wrong: %+v", *normalRow)
	}
	if report.Stats.Away.RedCards != 2 || report.Stats.Away.YellowCards != 1 {
		t.Fatalf("card totals wrong: %+v", report.Stats.Away)
	}

	ApplyPlayerMatchStats(home, away, &report)
	if secondYellow.SuspendedMatches != 1 || straightRed.SuspendedMatches != 1 {
		t.Fatalf("dismissals should each bank one suspension: second=%d straight=%d", secondYellow.SuspendedMatches, straightRed.SuspendedMatches)
	}
	if normalYellow.SuspendedMatches != 0 {
		t.Fatalf("ordinary yellow should not suspend: %d", normalYellow.SuspendedMatches)
	}
	if secondYellow.Appearances != 1 || straightRed.Appearances != 1 {
		t.Fatalf("dismissed starters should each record one appearance: second=%d straight=%d", secondYellow.Appearances, straightRed.Appearances)
	}
}
