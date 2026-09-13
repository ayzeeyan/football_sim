package transfers

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"

	"football_sim/pkg/managers"
	"football_sim/pkg/models"
)

func createTransferTestUniverse() (*TransferEngine, *models.Club, *models.Club) {
	barca := &models.Club{
		ClubID: "LAL-BAR", ClubName: "FC Barcelona", ShortName: "BAR", OverallTeamRating: 85,
		Squad: []*models.Player{
			{PlayerID: "P1", FullName: "Barca Star", OVR: 84, Age: 24, ClubID: "LAL-BAR"},
			{PlayerID: "WK_Venjamin_Valerio", FullName: "Venjamin Valerio", OVR: 78, Age: 14, UniverseWonderkid: true, ClubID: "LAL-BAR"},
		},
	}
	madrid := &models.Club{
		ClubID: "LAL-RMA", ClubName: "Real Madrid", ShortName: "RMA", OverallTeamRating: 86,
		Squad: []*models.Player{{PlayerID: "P2", FullName: "Madrid Veteran", OVR: 86, Age: 29, ClubID: "LAL-RMA"}},
	}
	clubs := []*models.Club{barca, madrid}
	mgrs := managers.BuildManagers(clubs)
	te := NewTransferEngine(clubs, mgrs, 42)
	return te, barca, madrid
}

func TestTransferEngine_TriggerSpecificBidIsIdempotent(t *testing.T) {
	te, barca, madrid := createTransferTestUniverse()
	te.BeginOffSeasonWindow()
	p := barca.Squad[0]
	p.MarketValueEUR = 40_000_000

	first := te.TriggerSpecificBid(madrid.ClubID, barca.ClubID, p.PlayerID)
	if first == nil || first.Buyer.ClubID != madrid.ClubID || first.Seller.ClubID != barca.ClubID {
		t.Fatalf("expected an offseason Super League bid, got %+v", first)
	}
	if first.StageName != "INQUIRY" || first.CurrentBid <= 0 {
		t.Fatalf("bid not initialized: %+v", first)
	}
	second := te.TriggerSpecificBid(madrid.ClubID, barca.ClubID, p.PlayerID)
	if second != first {
		t.Fatal("second bid should return the live negotiation")
	}
	if len(te.ActiveNegotiations) != 1 {
		t.Fatalf("duplicate talks: %d", len(te.ActiveNegotiations))
	}
}

func TestTransferEngine_WindowStatus(t *testing.T) {
	te, _, _ := createTransferTestUniverse()
	if te.IsWindowOpen() {
		t.Fatal("transfer market must be closed during the league season")
	}
	te.BeginOffSeasonWindow()
	if !te.IsWindowOpen() || te.CurrentWeek != 1 {
		t.Fatalf("new offseason should open at week 1, got open=%v week=%d", te.IsWindowOpen(), te.CurrentWeek)
	}
	for i := 0; i < TransferWindowWeeks; i++ {
		te.AdvanceOpenWindow()
	}
	if te.IsWindowOpen() || te.CurrentWeek != TransferWindowWeeks {
		t.Fatalf("window should close after exactly %d processed weeks without exposing an extra week; week=%d", TransferWindowWeeks, te.CurrentWeek)
	}
}

func TestTransferEngine_SummerWindowNeverExposesWeekThirteen(t *testing.T) {
	te, _, _ := createTransferTestUniverse()
	te.BeginOffSeasonWindow()
	for want := 1; want <= TransferWindowWeeks; want++ {
		if !te.IsWindowOpen() || te.CurrentWeek != want {
			t.Fatalf("before processing: open=%v week=%d, want open Week %d", te.IsWindowOpen(), te.CurrentWeek, want)
		}
		te.AdvanceOpenWindow()
		if te.CurrentWeek > TransferWindowWeeks {
			t.Fatalf("processed Week %d exposed illegal Week %d", want, te.CurrentWeek)
		}
	}
	if te.IsWindowOpen() || te.CurrentWeek != TransferWindowWeeks || te.ProcessedWeeks != TransferWindowWeeks {
		t.Fatalf("closed summer state = open:%v week:%d processed:%d", te.IsWindowOpen(), te.CurrentWeek, te.ProcessedWeeks)
	}
	// Closing is terminal and idempotent; it cannot manufacture another week.
	te.AdvanceOpenWindow()
	if te.CurrentWeek != TransferWindowWeeks || te.ProcessedWeeks != TransferWindowWeeks {
		t.Fatalf("closed window mutated to week=%d processed=%d", te.CurrentWeek, te.ProcessedWeeks)
	}
}

func TestTransferEngine_RepeatedSummerOpenDoesNotResetProgress(t *testing.T) {
	te, _, _ := createTransferTestUniverse()
	te.BeginOffSeasonWindow()
	te.AdvanceOpenWindow()
	te.AdvanceOpenWindow()
	if te.CurrentWeek != 3 {
		t.Fatalf("before repeated open, week=%d want 3", te.CurrentWeek)
	}
	te.BeginOffSeasonWindow()
	if te.CurrentWeek != 3 || te.ProcessedWeeks != 2 || !te.IsWindowOpen() {
		t.Fatalf("repeated summer open reset state: week=%d processed=%d open=%v", te.CurrentWeek, te.ProcessedWeeks, te.IsWindowOpen())
	}
}

func TestTransferEngine_WinterWindowHasIndependentFiniteState(t *testing.T) {
	te, _, _ := createTransferTestUniverse()
	te.BeginWinterWindow()
	if te.IsOffSeason || te.WindowType != WindowWinter || te.WindowWeeks() != WinterTransferWindowWeeks {
		t.Fatalf("winter setup invalid: off=%v type=%s weeks=%d", te.IsOffSeason, te.WindowType, te.WindowWeeks())
	}
	for i := 0; i < WinterTransferWindowWeeks; i++ {
		te.AdvanceOpenWindow()
	}
	if te.IsWindowOpen() || te.CurrentWeek != WinterTransferWindowWeeks || te.CurrentWeek >= TransferWindowWeeks {
		t.Fatalf("winter close invalid: open=%v week=%d", te.IsWindowOpen(), te.CurrentWeek)
	}
}

func TestTransferEngine_WinterCalendarAdvanceIsFiniteAndIdempotent(t *testing.T) {
	te, _, _ := createTransferTestUniverse()
	for mw := 1; mw <= WinterWindowStartMatchweek+WinterTransferWindowWeeks; mw++ {
		te.AdvanceWinterForMatchweek(mw)
		if te.CurrentWeek > WinterTransferWindowWeeks {
			t.Fatalf("matchweek %d exposed winter week %d", mw, te.CurrentWeek)
		}
	}
	if te.WindowType != WindowWinter || te.IsWindowOpen() || te.CurrentWeek != WinterTransferWindowWeeks {
		t.Fatalf("winter calendar did not close cleanly: type=%s open=%v week=%d", te.WindowType, te.IsWindowOpen(), te.CurrentWeek)
	}
	for i := 0; i < 4; i++ {
		te.AdvanceWinterForMatchweek(WinterWindowStartMatchweek + WinterTransferWindowWeeks + i)
	}
	if te.CurrentWeek != WinterTransferWindowWeeks || te.ProcessedWeeks != WinterTransferWindowWeeks {
		t.Fatalf("closed winter market advanced or reopened: week=%d processed=%d", te.CurrentWeek, te.ProcessedWeeks)
	}
}

func TestTransferEngine_NegotiationAndExecution(t *testing.T) {
	te, barca, madrid := createTransferTestUniverse()
	te.BeginOffSeasonWindow()
	te.RNG = rand.New(rand.NewSource(12345))
	p := barca.Squad[0]
	neg := &TransferNegotiation{
		NegotiationID: "TEST_NEG_1", Player: p, Buyer: madrid, Seller: barca,
		CurrentBid: 50_000_000, AskingPrice: 50_000_000, CreatedMatchweek: 45,
		StageIndex: 1, StageName: "INQUIRY", ProgressPct: 20,
	}
	te.ActiveNegotiations = append(te.ActiveNegotiations, neg)
	for step := 2; step <= 5; step++ {
		te.UpdateDailyMarket()
	}
	if len(te.CompletedTransfers) == 0 {
		t.Fatalf("expected completed transfer after 4 updates")
	}
	done := te.CompletedTransfers[0]
	if done.PlayerID != p.PlayerID {
		t.Errorf("expected player %s to be transferred, got %s", p.PlayerID, done.PlayerID)
	}
	foundInBuyer := false
	for _, pl := range madrid.Squad {
		if pl.PlayerID == p.PlayerID {
			foundInBuyer = true
			break
		}
	}
	if !foundInBuyer {
		t.Errorf("expected player to be present in buyer squad")
	}
	for _, pl := range barca.Squad {
		if pl.PlayerID == p.PlayerID {
			t.Errorf("expected player to be removed from seller squad")
		}
	}
	records := te.GetTransferRecords()
	if len(records.TopSignings) != 1 {
		t.Errorf("expected 1 top signing, got %d", len(records.TopSignings))
	}
	if records.NetSpend["LAL-RMA"].Spent != done.FeeEUR {
		t.Errorf("expected Madrid net spend %d, got %d", done.FeeEUR, records.NetSpend["LAL-RMA"].Spent)
	}
}

func createDeterministicTestUniverse(seed int64) *TransferEngine {
	clubIDs := []string{"EPL-ARS", "EPL-CHE", "LAL-BAR"}
	clubs := make([]*models.Club, 0, len(clubIDs))
	for _, cid := range clubIDs {
		squad := make([]*models.Player, 0, 20)
		for j := 1; j <= 20; j++ {
			squad = append(squad, &models.Player{
				PlayerID:       fmt.Sprintf("P_%s_%02d", cid, j),
				FullName:       fmt.Sprintf("Player %s %02d", cid, j),
				OVR:            78 + (j % 5),
				Age:            22 + (j % 8),
				ClubID:         cid,
				MarketValueEUR: int64(20_000_000 + j*1_000_000),
			})
		}
		club := &models.Club{
			ClubID:            cid,
			ClubName:          cid + " FC",
			ShortName:         cid[:3],
			OverallTeamRating: 82,
			Squad:             squad,
			Identity: models.ClubIdentity{
				FinancialPower: 80,
				Reputation:     80,
			},
			Finances: models.ClubFinances{
				TransferBudget: 150_000_000,
				Balance:        200_000_000,
			},
		}
		clubs = append(clubs, club)
	}
	mgrs := managers.BuildManagers(clubs)
	return NewTransferEngine(clubs, mgrs, seed)
}

func TestTransferEngine_DeterministicSelectionAcrossRuns(t *testing.T) {
	const fixedSeed = int64(987654321)
	const numRuns = 25

	type negRecord struct {
		NegotiationID string
		BuyerID       string
		SellerID      string
		PlayerID      string
		CurrentBid    int64
	}

	var baseline []negRecord

	for run := 0; run < numRuns; run++ {
		te := createDeterministicTestUniverse(fixedSeed)
		te.BeginOffSeasonWindow()
		for w := 0; w < 3; w++ {
			te.AdvanceOpenWindow()
		}

		records := make([]negRecord, len(te.ActiveNegotiations))
		for i, n := range te.ActiveNegotiations {
			records[i] = negRecord{
				NegotiationID: n.NegotiationID,
				BuyerID:       n.Buyer.ClubID,
				SellerID:      n.Seller.ClubID,
				PlayerID:      n.Player.PlayerID,
				CurrentBid:    n.CurrentBid,
			}
		}

		if run == 0 {
			if len(records) == 0 {
				t.Fatalf("expected at least one active negotiation in baseline run")
			}
			baseline = records
		} else {
			if !reflect.DeepEqual(baseline, records) {
				t.Fatalf("run %d diverged from baseline!\nExpected: %+v\nGot: %+v", run, baseline, records)
			}
		}
	}
}
