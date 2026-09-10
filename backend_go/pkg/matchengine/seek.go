package matchengine

import (
	"fmt"
)

// Seek controls (Feature 13): jump to ~70' and jump to the next chance.
//
// Both fast-forward through the exact live tick path, so bookings, subs,
// tactics, commentary and the half-time dugout behave as if watched. Seeks
// stop at HALF_TIME (the dugout still owns the interval) and at FULL_TIME
// (which still commits exactly once via the usual instance guard: seeks
// never touch InstanceID). Speed 999 keeps its instant-to-FT behavior.

const (
	seekStep     = 0.05
	seekMaxTicks = 200000
)

// fastForward runs live ticks until stop reports done or the match leaves
// playable states. A paused match resumes paused afterwards.
func (e *LiveMatchEngine) fastForward(stop func() bool) {
	if e == nil {
		return
	}
	if e.State != "PLAYING" && e.State != "PAUSED" && e.State != "GOAL_PAUSE" {
		return
	}
	wasPaused := e.State == "PAUSED"
	if wasPaused {
		e.State = "PLAYING"
	}
	for i := 0; i < seekMaxTicks; i++ {
		if e.State != "PLAYING" && e.State != "GOAL_PAUSE" {
			break
		}
		if stop != nil && stop() {
			break
		}
		e.Tick(seekStep)
	}
	if wasPaused && e.State == "PLAYING" {
		e.State = "PAUSED"
	}
}

// SeekTo70 jumps toward the 70th minute. Returns true when the clock moved.
// From the first half the run ends at the half-time dugout; resume and seek
// again for the second half.
func (e *LiveMatchEngine) SeekTo70() bool {
	if e == nil {
		return false
	}
	start := e.CurrentMinute
	state := e.State
	if state != "PLAYING" && state != "PAUSED" && state != "GOAL_PAUSE" {
		return false
	}
	if e.CurrentMinute >= 70 {
		return false
	}
	e.fastForward(func() bool { return e.CurrentMinute >= 70 })
	if e.CurrentMinute <= start {
		return false
	}
	if e.State == "PLAYING" || e.State == "PAUSED" {
		e.AddCommentary(int(e.CurrentMinute), fmt.Sprintf("Fast-forward — play resumes at %d'.", int(e.CurrentMinute)), "NORMAL", false)
	}
	return true
}

// SeekNextChance jumps to the next shot (or penalty). Returns true when a
// new chance occurred before a terminal state.
func (e *LiveMatchEngine) SeekNextChance() bool {
	if e == nil {
		return false
	}
	state := e.State
	if state != "PLAYING" && state != "PAUSED" && state != "GOAL_PAUSE" {
		return false
	}
	startShots := len(e.LiveShots)
	e.fastForward(func() bool { return len(e.LiveShots) > startShots })
	return len(e.LiveShots) > startShots
}
