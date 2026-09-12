package tournament

import (
	"reflect"
	"testing"

	"football_sim/pkg/transfers"
)

func TestRestartCurrentSeasonResetsTransferStateAndPreservesCareerData(t *testing.T) {
	tm, ge := loadTestUniverse(t)
	te := transfers.NewTransferEngine(tm.ClubsList, tm.Managers, 88)
	tm.TransferEngine = te
	seedPlayer := tm.ClubsList[0].Squad[0]
	seedPlayer.CareerGoals = 7
	seedPlayer.CareerAssists = 5
	seedPlayer.CareerApps = 18
	seedPlayer.BestGoals = 7
	seedPlayer.BestAssists = 5
	seedPlayer.BestSeason = "2025-26"

	beforeSquads := make(map[string][]string, len(tm.ClubsList))
	beforeFinances := make(map[string]int64, len(tm.ClubsList))
	type playerState struct {
		Age           int
		CareerGoals   int
		CareerAssists int
		CareerApps    int
		BestGoals     int
		BestAssists   int
		BestSeason    string
	}
	beforePlayers := make(map[string]playerState)
	var growthID string
	var growthHeight float64
	var growthXP float64
	for _, club := range tm.ClubsList {
		ids := make([]string, 0, len(club.Squad))
		beforeFinances[club.ClubID] = club.Finances.TransferBudget
		for _, player := range club.Squad {
			ids = append(ids, player.PlayerID)
			beforePlayers[player.PlayerID] = playerState{
				Age:           player.Age,
				CareerGoals:   player.CareerGoals,
				CareerAssists: player.CareerAssists,
				CareerApps:    player.CareerApps,
				BestGoals:     player.BestGoals,
				BestAssists:   player.BestAssists,
				BestSeason:    player.BestSeason,
			}
			if growthID == "" && ge.Biometrics[player.PlayerID] != nil {
				growthID = player.PlayerID
				growthHeight = ge.Biometrics[player.PlayerID].CurrentHeightCM
			}
		}
		beforeSquads[club.ClubID] = ids
	}
	if growthID == "" {
		t.Fatal("expected a player with growth data")
	}
	ge.Biometrics[growthID].AccumulatedXP = 12.5
	growthXP = ge.Biometrics[growthID].AccumulatedXP
	tm.ReputationAppliedSeason = tm.SeasonName

	te.ActiveNegotiations = []*transfers.TransferNegotiation{{NegotiationID: "active"}}
	te.CompletedTransfers = []transfers.CompletedTransfer{{PlayerID: "completed"}}
	te.AllTimeTransfers = []transfers.CompletedTransfer{{PlayerID: "all-time"}}
	te.CurrentDay = 17
	te.CurrentMatchweek = 22
	tm.CurrentMatchweek = 22
	for _, manager := range te.Managers {
		manager.BudgetEur = 1
	}

	res := tm.RestartCurrentSeason()
	if res["status"] != "success" {
		t.Fatalf("restart response: %v", res)
	}
	if tm.ReputationAppliedSeason != "" {
		t.Fatalf("restart retained completed-season reputation marker %q", tm.ReputationAppliedSeason)
	}
	if te.CurrentDay != 1 || te.CurrentMatchweek != 1 {
		t.Fatalf("transfer calendar after restart = day %d, week %d", te.CurrentDay, te.CurrentMatchweek)
	}
	if len(te.ActiveNegotiations) != 0 || len(te.CompletedTransfers) != 0 {
		t.Fatalf("season transfer state was not cleared: active=%d completed=%d", len(te.ActiveNegotiations), len(te.CompletedTransfers))
	}
	if !reflect.DeepEqual(te.AllTimeTransfers, []transfers.CompletedTransfer{{PlayerID: "all-time"}}) {
		t.Fatalf("all-time transfer history changed: %+v", te.AllTimeTransfers)
	}
	for _, club := range tm.ClubsList {
		if got, want := club.Finances.TransferBudget, beforeFinances[club.ClubID]; got != want {
			t.Fatalf("%s club-owned transfer budget changed on season restart: got=%d want=%d", club.ClubID, got, want)
		}
		if got, want := te.Managers[club.ClubID].BudgetEur, club.Finances.TransferBudget; got != want {
			t.Fatalf("%s manager compatibility budget=%d, want club budget %d", club.ClubID, got, want)
		}
		ids := make([]string, 0, len(club.Squad))
		for _, player := range club.Squad {
			ids = append(ids, player.PlayerID)
			before := beforePlayers[player.PlayerID]
			got := playerState{
				Age:           player.Age,
				CareerGoals:   player.CareerGoals,
				CareerAssists: player.CareerAssists,
				CareerApps:    player.CareerApps,
				BestGoals:     player.BestGoals,
				BestAssists:   player.BestAssists,
				BestSeason:    player.BestSeason,
			}
			if got != before {
				t.Fatalf("player %s career state changed: before=%+v after=%+v", player.PlayerID, before, got)
			}
		}
		if !reflect.DeepEqual(ids, beforeSquads[club.ClubID]) {
			t.Fatalf("squad %s changed: before=%v after=%v", club.ClubID, beforeSquads[club.ClubID], ids)
		}
	}
	if got := ge.Biometrics[growthID].CurrentHeightCM; got != growthHeight {
		t.Fatalf("growth height changed from %.2f to %.2f", growthHeight, got)
	}
	if got := ge.Biometrics[growthID].AccumulatedXP; got != growthXP {
		t.Fatalf("growth XP changed from %.2f to %.2f", growthXP, got)
	}
}
