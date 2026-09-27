package tournament

import (
	"regexp"
	"testing"

	"football_sim/pkg/matchreport"
)

var clockOnlyRe = regexp.MustCompile(`^\d+(?:\+\d+)?'$`)

func clockOnly(display string) bool {
	return clockOnlyRe.MatchString(display)
}

func TestSimulateFixtureBuildsAuthoritativeReport(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	var fx *Fixture
	scan := func(list []Fixture) {
		for i := range list {
			f := &list[i]
			if fx == nil && f.Status == "scheduled" && f.Matchweek == tm.CurrentMatchweek && tm.isWorldDomesticLeague(f.Competition) {
				copy := *f
				fx = &copy
			}
		}
	}
	scan(tm.Fixtures)
	if tm.World != nil {
		scan(tm.World.Fixtures)
	}
	if fx == nil {
		t.Fatal("no scheduled domestic fixture")
	}
	id := fx.FixtureID
	res := tm.SimulateFixture(id)
	if res["status"] != "success" {
		t.Fatalf("simulate: %+v", res)
	}
	if res["report_ready"] != true {
		t.Fatalf("report_ready missing: %+v", res)
	}
	finished := tm.FindFixture(id)
	if finished == nil || finished.Status != "finished" || finished.Report == nil {
		t.Fatal("fixture was not finalized with a report")
	}
	report := finished.Report
	if report.HomeXI == nil || report.AwayXI == nil {
		t.Fatal("lineups missing")
	}
	if report.Stats.Home.Shots+report.Stats.Away.Shots == 0 && report.HomeGoals+report.AwayGoals > 0 {
		t.Fatal("scoring match produced empty shot stats")
	}
	rated := false
	rows := append([]matchreport.MatchPlayerRow{}, report.HomeXI...)
	rows = append(rows, report.AwayXI...)
	for _, row := range rows {
		if row.Played && row.Rating != nil {
			rated = true
			break
		}
	}
	if !rated {
		t.Fatal("ratings were not finalized")
	}
	if report.TableImpact == nil || !report.TableImpact.Applicable {
		t.Fatal("league table impact missing")
	}
	for _, e := range report.Events {
		if e.Display != "" && !clockOnly(e.Display) {
			t.Fatalf("display still has presentation text: %q", e.Display)
		}
		if (e.Type == "goal" || e.Type == "penalty" || e.Type == "own_goal") && e.PlayerName == "" && e.PlayerID == "" {
			t.Fatalf("scoring event missing player identity: %+v", e)
		}
		if e.Assister != nil && e.AssistPlayerID == "" {
			t.Fatalf("assister missing flat id: %+v", e)
		}
		if (e.PlayerID != "" || e.PlayerName != "") && e.ClubID == "" {
			t.Fatalf("attributed event missing club: %+v", e)
		}
	}

	again := tm.SimulateFixture(id)
	if again["status"] != "error" {
		t.Fatalf("second finalize should fail, got %+v", again)
	}
}

func TestMatchPreviewUsesRealState(t *testing.T) {
	tm, _, _ := loadEuropeanWorldForTest(t)
	var fx *Fixture
	for i := range tm.Fixtures {
		f := &tm.Fixtures[i]
		if f.Status == "scheduled" {
			fx = f
			break
		}
	}
	if fx == nil {
		t.Fatal("no fixture")
	}
	home := tm.Clubs[fx.HomeID]
	away := tm.Clubs[fx.AwayID]
	note := tm.KickoffNote(fx, home, away)
	if note == "" {
		t.Fatal("empty kickoff note")
	}
	imp := tm.EvaluateMatchImportance(fx)
	if imp == "" {
		t.Fatal("empty importance")
	}
}
