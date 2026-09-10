package matchengine

import "football_sim/pkg/models"

// ResumeHalfTime validates the whole dugout choice before changing the match.
// Empty stance keeps the current tactic; empty player IDs mean no substitution.
func (e *LiveMatchEngine) ResumeHalfTime(side, stance, outID, inID string) bool {
	if e.State != "HALF_TIME" || (side != "home" && side != "away") {
		return false
	}
	if stance != "" && stance != "OVERLOAD" && stance != "PARK_BUS" {
		return false
	}
	var out, in *models.Player
	if outID != "" || inID != "" {
		if outID == "" || inID == "" || e.SubstitutionsMade[side] >= 5 || e.Bookings[outID] >= 2 || e.Bookings[inID] >= 2 {
			return false
		}
		starters, bench := e.HomeStarters, e.HomeBench
		if side == "away" {
			starters, bench = e.AwayStarters, e.AwayBench
		}
		for _, p := range starters {
			if p.PlayerID == inID {
				return false
			}
			if p.PlayerID == outID {
				out = p
			}
		}
		for _, p := range bench {
			if p.PlayerID == inID {
				in = p
			}
		}
		for _, event := range e.Events {
			if event.Side == side && ((event.PlayerIn != nil && event.PlayerIn.PlayerID == inID) || (event.PlayerOut != nil && event.PlayerOut.PlayerID == inID)) {
				return false
			}
		}
		if out == nil || in == nil {
			return false
		}
	}
	if stance != "" {
		e.setStance(side, stance)
		e.AddCommentary(45, e.managerName(side)+" sets the second-half stance to "+stance+".", "NORMAL", false)
	}
	if out != nil {
		e.ExecuteDirectSub(side, out, in, 45, "Half-time dugout")
		for i := range e.PlannedSubs {
			s := &e.PlannedSubs[i]
			if s.Side == side && ((s.Out != nil && (s.Out.PlayerID == outID || s.Out.PlayerID == inID)) || (s.In != nil && (s.In.PlayerID == inID || s.In.PlayerID == outID))) {
				s.Done = true
			}
		}
	}
	e.halfTimeReached = true
	e.State = "PLAYING"
	e.AddCommentary(45, "The second half is underway!", "KICKOFF", false)
	return true
}
