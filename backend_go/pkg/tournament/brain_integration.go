package tournament

import (
	"football_sim/pkg/brain"
	"football_sim/pkg/managers"
)

// The world brain: one tiny online-learned model per career.
//
// Pre-training: the brain starts from the committed base artifact
// (brain.Base, fitted offline by cmd/pretrain).
// Post-training: every finished matchweek the brain takes one SGD step per
// match, in stable fixture order, from the actual goal differences. It
// never stops: the longer a career runs, the more the brain has learned
// about this particular world.
// Thinking: the learned style-matchup weights replace the constant
// tactical edge once enough matches have been observed.

// brainEdgeBlendSamples is the sample count at which the learned edge fully
// replaces the fixed style table. Below it the two are blended linearly, so
// a fresh career starts on the shipped priors and gradually thinks for
// itself.
const brainEdgeBlendSamples = 400

// brainEdgeClamp bounds the learned edge in goal-lambda units, matching
// the historical ±0.16 table scale with headroom for learned conviction.
const brainEdgeClamp = 0.25

// InitBrain installs a fresh pre-trained brain if none exists.
// Caller must hold tm.mu.
func (tm *TournamentManager) InitBrainUnlocked() {
	if tm.Brain == nil {
		tm.Brain = brain.New()
	}
}

// BrainTacticEdge returns the tactical edge for one philosophical pairing:
// the fixed table blended with the brain's learned conviction, weighted by
// how much match experience the brain has. Pure read.
func (tm *TournamentManager) BrainTacticEdge(homeStyle, awayStyle string) float64 {
	if tm == nil || tm.Brain == nil {
		return managers.TacticEdge(homeStyle, awayStyle)
	}
	fixed := managers.TacticEdge(homeStyle, awayStyle)
	learned := tm.Brain.PredictStyleEdge(brain.StyleEdgeFeatures(homeStyle, awayStyle))
	if learned > brainEdgeClamp {
		learned = brainEdgeClamp
	}
	if learned < -brainEdgeClamp {
		learned = -brainEdgeClamp
	}
	w := float64(tm.Brain.Samples) / float64(brainEdgeBlendSamples)
	if w > 1 {
		w = 1
	}
	return fixed*(1-w) + learned*w
}

// trainBrainUnlocked post-trains the brain on one completed matchweek's
// actual results: one deterministic SGD step per finished fixture, in
// stable fixture order. Caller must hold tm.mu.
func (tm *TournamentManager) trainBrainUnlocked(completedMW int) {
	if tm.Brain == nil {
		return
	}
	trainList := func(list []Fixture) {
		for i := range list {
			f := &list[i]
			if f.Matchweek != completedMW || f.Status != "finished" || f.HomeGoals == nil || f.AwayGoals == nil {
				continue
			}
			home, away := tm.Clubs[f.HomeID], tm.Clubs[f.AwayID]
			if home == nil || away == nil {
				continue
			}
			features := brain.MatchFeatures(home, away, tm.Managers[f.HomeID], tm.Managers[f.AwayID])
			target := float64(*f.HomeGoals - *f.AwayGoals)
			tm.Brain.Train(features, target)
		}
	}
	trainList(tm.Fixtures)
	if tm.World != nil {
		trainList(tm.World.Fixtures)
	}
}

// BrainState returns an independent copy of the brain for the wire surface.
func (tm *TournamentManager) BrainState() *brain.Model {
	if tm == nil {
		return nil
	}
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	if tm.Brain == nil {
		return nil
	}
	return tm.Brain.Clone()
}
