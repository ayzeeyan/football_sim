package growth

import (
	"reflect"
	"testing"
)

func TestSeasonCapBlocksPositiveAttributeMutationAtPlateau(t *testing.T) {
	ge := NewGrowthEngine(7001)
	ge.RegisterProdigy("cap_plateau", "Cap Plateau", 14, 168, 58, "FWD", 75, 95, 19)

	ge.mu.Lock()
	ge.internalNudgeToOVR("cap_plateau", "FWD", 80)
	attrsBefore := *ge.Attributes["cap_plateau"]
	bioBefore := *ge.Biometrics["cap_plateau"]
	ge.mu.Unlock()
	if got := ge.CalculateOVR("cap_plateau", "FWD"); got != 80 {
		t.Fatalf("plateau setup OVR=%d, want 80", got)
	}

	mentorOVR, mentorName, personality := 92, "Senior Legend", "dedicated_pro"
	matchOpts := MatchXPOptions{MentorOVR: &mentorOVR, MentorName: &mentorName, Personality: &personality}
	mentorshipOpts := MentorshipOptions{MentorOVR: &mentorOVR, MentorName: &mentorName, Personality: &personality}
	var events int
	for i := 0; i < 40; i++ {
		events += len(ge.ApplyMatchXP("cap_plateau", "Cap Plateau", "FWD", 10, 5, 5, matchOpts))
		events += len(ge.ApplyMentorshipTick("cap_plateau", "Cap Plateau", mentorshipOpts))
		for _, focus := range []string{"hypertrophy", "technical", "tactical"} {
			res, err := ge.RunTrainingCycle("cap_plateau", focus, false)
			if err != nil {
				t.Fatalf("training focus %s failed: %v", focus, err)
			}
			if attrs, ok := res["gains"].(map[string]interface{})["attributes"].([]string); ok && len(attrs) > 0 {
				t.Fatalf("training at ceiling reported attribute gains: %v", attrs)
			}
		}
		ge.SimulatePubertyCycle("cap_plateau", i+1)
	}

	if got := ge.EnforceSeasonOVRCap("cap_plateau", "FWD"); got != 80 {
		t.Fatalf("capped OVR=%d, want 80", got)
	}
	ge.mu.RLock()
	attrsAfter := *ge.Attributes["cap_plateau"]
	bioAfter := *ge.Biometrics["cap_plateau"]
	ge.mu.RUnlock()
	if !reflect.DeepEqual(attrsAfter, attrsBefore) {
		t.Fatalf("attributes changed after plateau: before=%+v after=%+v", attrsBefore, attrsAfter)
	}
	if bioAfter.AccumulatedXP <= bioBefore.AccumulatedXP {
		t.Fatalf("match XP/mentorship did not accumulate at plateau: before=%.1f after=%.1f", bioBefore.AccumulatedXP, bioAfter.AccumulatedXP)
	}
	if events != 0 {
		t.Fatalf("expected 0 attribute increase events at ceiling plateau, got %d", events)
	}
}

func TestSeasonalGrowthKeepsBaselineUntilSeasonBoundary(t *testing.T) {
	ge := NewGrowthEngine(7002)
	ge.RegisterProdigy("season_boundary", "Season Boundary", 14, 168, 58, "FWD", 75, 95, 19)
	bio := ge.Biometrics["season_boundary"]
	initialBaseline := bio.SeasonStartOVR

	for i := 0; i < 12; i++ {
		current := ge.CalculateOVR("season_boundary", "FWD")
		ge.ApplySeasonalGrowth("season_boundary", 14, 50, 95, "FWD", current)
		if bio.SeasonStartOVR != initialBaseline {
			t.Fatalf("repeated same-season growth reset baseline to %d (initial %d)", bio.SeasonStartOVR, initialBaseline)
		}
		if got := ge.CalculateOVR("season_boundary", "FWD"); got > initialBaseline+MaxAnnualOVRGain {
			t.Fatalf("same-season growth reached OVR %d above ceiling %d", got, initialBaseline+MaxAnnualOVRGain)
		}
	}

	endOfSeason := ge.CalculateOVR("season_boundary", "FWD")
	got, ok := ge.AdvanceSeasonStartOVR("season_boundary", "FWD")
	if !ok || got != endOfSeason {
		t.Fatalf("advanced baseline returned %d, want current OVR %d", got, endOfSeason)
	}
	if bio.SeasonStartOVR != endOfSeason {
		t.Fatalf("boundary baseline=%d, want %d", bio.SeasonStartOVR, endOfSeason)
	}

	ge.ApplySeasonalGrowth("season_boundary", 15, 50, 95, "FWD", endOfSeason)
	if bio.SeasonStartOVR != endOfSeason {
		t.Fatalf("same next-season growth reset baseline to %d (start %d)", bio.SeasonStartOVR, endOfSeason)
	}
}

func TestVeteranDeclineAge30Transition(t *testing.T) {
	ge := NewGrowthEngine(7003)
	playerID := "vet_age_30"
	ge.RegisterProdigy(playerID, "Veteran Thirty", 29, 185, 78, "FWD", 85, 95, 19)

	ge.mu.Lock()
	ge.internalNudgeToOVR(playerID, "FWD", 85)
	ge.Biometrics[playerID].SeasonStartOVR = 85
	ge.mu.Unlock()

	if got := ge.CalculateOVR(playerID, "FWD"); got != 85 {
		t.Fatalf("setup OVR=%d, want 85", got)
	}

	// Transition to age 30
	ge.mu.Lock()
	bio := ge.Biometrics[playerID]
	bio.Age = 30
	ge.mu.Unlock()

	// 1. Verify SeasonalOVRDrop returns exact -1 drop
	declinedOVR := SeasonalOVRDrop(30, 85)
	if declinedOVR != 84 {
		t.Fatalf("SeasonalOVRDrop at age 30: want 84, got %d", declinedOVR)
	}

	// 2. Apply aging physical decline
	changed := ge.ApplyAgingDecline(playerID, 30)
	if len(changed) != 4 {
		t.Fatalf("expected 4 physical attributes to drop, got %v", changed)
	}

	// 3. Advance season start OVR with declined model OVR
	advanced, ok := ge.AdvanceSeasonStartOVR(playerID, "FWD", declinedOVR)
	if !ok || advanced != 84 {
		t.Fatalf("AdvanceSeasonStartOVR returned (%d, %v), want (84, true)", advanced, ok)
	}
	if bio.SeasonStartOVR != 84 {
		t.Fatalf("bio.SeasonStartOVR=%d, want 84", bio.SeasonStartOVR)
	}

	// 4. CalculateOVR must respect veteran decline and not bounce back to 85
	if got := ge.CalculateOVR(playerID, "FWD"); got != 84 {
		t.Fatalf("CalculateOVR after age-30 decline: got %d, want 84", got)
	}

	// 5. ApplySeasonalGrowth must not overwrite veteran decline
	seasonalOVR := ge.ApplySeasonalGrowth(playerID, 30, 38, 95, "FWD", declinedOVR)
	if seasonalOVR > 84 {
		t.Fatalf("ApplySeasonalGrowth overwrote veteran decline: got %d, want <= 84", seasonalOVR)
	}

	// 6. Manual actions cannot grow veteran above declined ceiling
	for _, focus := range []string{"hypertrophy", "technical", "tactical"} {
		res, err := ge.RunTrainingCycle(playerID, focus, false)
		if err != nil {
			t.Fatalf("training focus %s failed: %v", focus, err)
		}
		if attrs, ok := res["gains"].(map[string]interface{})["attributes"].([]string); ok && len(attrs) > 0 {
			t.Fatalf("veteran training reported attribute gains: %v", attrs)
		}
	}
	if got := ge.CalculateOVR(playerID, "FWD"); got > 84 {
		t.Fatalf("CalculateOVR after training exceeded veteran ceiling: got %d, want <= 84", got)
	}
}
