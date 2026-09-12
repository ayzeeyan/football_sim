package matchengine

import (
	"fmt"
	"math"
	"math/rand"

	"football_sim/pkg/managers"
	"football_sim/pkg/matchreport"
	"football_sim/pkg/models"
)

// Headless live-match simulation core, ported from match_engine.py
// (MatchEngine phase machine, shot/penalty/booking resolution, planned and
// AI-driven substitutions, game-state tactics, and the live report payload).
//
// Sound and renderer calls are intentionally omitted: ball-target updates
// record live touches (renderer-present semantics) so downstream heatmaps and
// shot maps stay faithful, while explosions, trails, and whistles have no
// headless equivalent.

func (e *LiveMatchEngine) rng() *rand.Rand {
	if e.RNG == nil {
		e.RNG = rand.New(rand.NewSource(1))
	}
	return e.RNG
}

func liveMini(p *models.Player) matchreport.MiniPlayer {
	return matchreport.ToMiniPlayer(p)
}

func round2(x float64) float64 {
	return math.Round(x*100) / 100
}

// Update advances the live simulation clock and phase machine
// (Python: MatchEngine.update without renderer/sound).
func (e *LiveMatchEngine) Update(dt float64) {
	if e.State == "GOAL_PAUSE" {
		e.BannerTimer -= dt
		if e.BannerTimer <= 0 {
			e.Banner = ""
			e.State = "PLAYING"
			e.BallPos = Coordinate{X: 0.5, Y: 0.5}
			e.BallTarget = Coordinate{X: 0.5, Y: 0.5}
			e.BallHeight = 0
			e.BallIsShot = false
			e.Phase = "BUILDUP"
		}
		return
	}
	if e.State != "PLAYING" {
		return
	}
	if e.Speed == 999 {
		e.InstantSimulate()
		return
	}
	previousMinute := e.CurrentMinute
	e.CurrentMinute += 1.67 * float64(e.Speed) * dt
	if !e.halfTimeReached && previousMinute < 45 && e.CurrentMinute >= 45 {
		e.CurrentMinute = 45
		e.halfTimeReached = true
		e.State = "HALF_TIME"
		e.AddCommentary(45, "Half-time. The teams return to the dugout.", "NORMAL", false)
		return
	}
	e.MaybeSubstitute()
	e.EvaluateGameStateTactics()

	if e.PossessionTeam == "home" {
		e.HomePossessionTicks++
		e.HomePossessionWeighted += thirdWeight(e.BallTarget.X, true)
	} else {
		e.AwayPossessionTicks++
		e.AwayPossessionWeighted += thirdWeight(e.BallTarget.X, false)
	}
	// Field tilt: ease momentum toward the side in possession so the HUD tilt
	// is non-zero whenever the sim is running.
	momentumTarget := 0.55
	if e.PossessionTeam != "home" {
		momentumTarget = -0.55
	}
	e.PossessionMomentum += (momentumTarget - e.PossessionMomentum) * 0.06
	if e.PossessionMomentum > 1 {
		e.PossessionMomentum = 1
	} else if e.PossessionMomentum < -1 {
		e.PossessionMomentum = -1
	}
	// Shot elevation decays so the ball arc is visible for ~1s after a shot.
	if e.BallHeight > 0 {
		e.BallHeight -= dt * 8.0 * float64(e.Speed)
		if e.BallHeight <= 0.3 {
			e.BallHeight = 0
			e.BallIsShot = false
		}
	}

	if e.CurrentMinute >= 90.0 {
		e.CurrentMinute = 90.0
		e.State = "FULL_TIME"
		e.AddCommentary(90, fmt.Sprintf("FULL TIME! Final Score: %s %d - %d %s.", e.HomeClub.ShortName, e.HomeScore, e.AwayScore, e.AwayClub.ShortName), "FULLTIME", false)
		return
	}

	e.PhaseTimer += dt * float64(e.Speed)
	if e.PhaseTimer >= 1.6 {
		e.PhaseTimer = 0.0
		e.AdvancePhase()
	}
}

// triggerMovement records a headless ball movement and its live touch.
// It also publishes the pass_trail segment the pitch canvas draws.
func (e *LiveMatchEngine) triggerMovement(target [2]float64, team string) {
	from := [2]float64{round2(e.BallTarget.X), round2(e.BallTarget.Y)}
	e.BallTarget = Coordinate{X: target[0], Y: target[1]}
	if e.LiveTouches == nil {
		e.LiveTouches = map[string][][2]float64{"home": {}, "away": {}}
	}
	e.LiveTouches[team] = append(e.LiveTouches[team], [2]float64{round2(target[0]), round2(target[1])})
	isShot := target[0] >= 0.93 || target[0] <= 0.07
	e.PassTrail = &PassTrailItem{
		From: from, To: [2]float64{round2(target[0]), round2(target[1])},
		IsShot: isShot, Color: e.trailColor(team),
	}
	if team == "home" {
		e.PossessionMomentum = math.Min(1, e.PossessionMomentum+0.03)
	} else {
		e.PossessionMomentum = math.Max(-1, e.PossessionMomentum-0.03)
	}
}

// trailColor carries the possessing side's primary color for the pass line.
func (e *LiveMatchEngine) trailColor(team string) [3]uint8 {
	steel := [3]uint8{90, 138, 171}
	if team == "home" {
		if e.HomeClub != nil && e.HomeClub.PrimaryColor != [3]uint8{0, 0, 0} {
			return e.HomeClub.PrimaryColor
		}
		return steel
	}
	if e.AwayClub != nil && e.AwayClub.PrimaryColor != [3]uint8{0, 0, 0} {
		return e.AwayClub.PrimaryColor
	}
	return steel
}

// markShotBall elevates the ball and flags it as goal-bound for the canvas.
func (e *LiveMatchEngine) markShotBall(team string) {
	e.BallHeight = 6 + e.rng().Float64()*8
	e.BallIsShot = true
	if e.PassTrail != nil {
		e.PassTrail.IsShot = true
	}
	if team == "home" {
		e.PossessionMomentum = math.Min(1, e.PossessionMomentum+0.15)
	} else {
		e.PossessionMomentum = math.Max(-1, e.PossessionMomentum-0.15)
	}
}

// AdvancePhase runs one territory/progression evaluation (Python: _advance_phase).
func (e *LiveMatchEngine) AdvancePhase() {
	rng := e.rng()
	attackingClub := e.HomeClub
	defendingClub := e.AwayClub
	attackingStarters := e.HomeStarters
	defendingStarters := e.AwayStarters
	if e.PossessionTeam != "home" {
		attackingClub, defendingClub = defendingClub, attackingClub
		attackingStarters, defendingStarters = defendingStarters, attackingStarters
	}

	delta := e.liveStrength(attackingStarters)
	if e.PossessionTeam == "home" {
		delta += 2
	}
	delta -= e.liveStrength(defendingStarters)

	if e.HomeManager != nil && e.AwayManager != nil {
		tEdge := managers.TacticEdge(e.HomeManager.Style, e.AwayManager.Style)
		if e.PossessionTeam == "home" {
			delta += tEdge * 10.0
		} else {
			delta -= tEdge * 10.0
		}
	}

	attStance := e.HomeStance
	defStance := e.AwayStance
	if e.PossessionTeam != "home" {
		attStance, defStance = defStance, attStance
	}
	if attStance == "OVERLOAD" {
		delta += 3.5
	}
	if defStance == "OVERLOAD" {
		delta += 3.0
	}
	if defStance == "PARK_BUS" {
		delta -= 3.5
	}
	if attStance == "PARK_BUS" {
		delta -= 2.0
	}

	minute := int(e.CurrentMinute)
	switch e.Phase {
	case "BUILDUP":
		if e.PossessionTeam == "home" {
			e.ActiveThird = "DEFENSIVE"
		} else {
			e.ActiveThird = "ATTACKING"
		}
		if rng.Float64() < 0.82+delta*0.012 {
			e.Phase = "MIDFIELD"
			e.ActiveThird = "MIDFIELD"
			shift := 0.15
			if e.PossessionTeam != "home" {
				shift = -0.15
			}
			e.triggerMovement([2]float64{0.50 + shift, 0.3 + rng.Float64()*0.4}, e.PossessionTeam)
		} else {
			e.Turnover("pressed in buildup")
		}
	case "MIDFIELD":
		e.ActiveThird = "MIDFIELD"
		roll := rng.Float64()
		if roll < 0.60+delta*0.015 {
			e.Phase = "ATTACKING"
			if e.PossessionTeam == "home" {
				e.ActiveThird = "ATTACKING"
			} else {
				e.ActiveThird = "DEFENSIVE"
			}
			onPitch := e.OnPitch(attackingStarters)
			pool := make([]*models.Player, 0, len(onPitch))
			for _, p := range onPitch {
				if p.Category == "MID" || p.Category == "FWD" {
					pool = append(pool, p)
				}
			}
			if len(pool) == 0 {
				pool = onPitch
			}
			if len(pool) == 0 {
				e.Turnover("no available creator")
				return
			}
			creator := pool[rng.Intn(len(pool))]
			x := 0.75
			if e.PossessionTeam != "home" {
				x = 0.25
			}
			e.triggerMovement([2]float64{x, 0.2 + rng.Float64()*0.6}, e.PossessionTeam)
			if rng.Float64() < 0.35 {
				e.AddCommentary(minute, fmt.Sprintf("%s strings together a penetrative pass into the attacking third.", creator.FullName), "CHANCE", creator.UniverseWonderkid)
			}
		} else if roll < 0.88 {
			e.triggerMovement([2]float64{0.50 + (rng.Float64()*0.2 - 0.1), 0.25 + rng.Float64()*0.5}, e.PossessionTeam)
		} else {
			e.Turnover("tackled in midfield duel")
		}
	case "ATTACKING":
		if e.PossessionTeam == "home" {
			e.ActiveThird = "ATTACKING"
		} else {
			e.ActiveThird = "DEFENSIVE"
		}
		if rng.Float64() < 0.55+delta*0.018 {
			e.Phase = "SHOT"
			e.ResolveShot(attackingClub, defendingClub, attackingStarters, defendingStarters)
		} else {
			if rng.Float64() < 0.4 {
				if e.PossessionTeam == "home" {
					e.HomeCorners++
				} else {
					e.AwayCorners++
				}
				x := 0.88
				if e.PossessionTeam != "home" {
					x = 0.12
				}
				ys := []float64{0.15, 0.85}
				e.triggerMovement([2]float64{x, ys[rng.Intn(2)]}, e.PossessionTeam)
				e.AddCommentary(minute, fmt.Sprintf("Deflected behind for a %s corner kick.", attackingClub.ShortName), "NORMAL", false)
			} else {
				e.Turnover("defensive interception in the penalty area")
			}
		}
	case "SHOT":
		e.Phase = "BUILDUP"
	}
}

// ResolveShot evaluates one shot at goal (Python: _resolve_shot).
func (e *LiveMatchEngine) ResolveShot(attackingClub, defendingClub *models.Club, attackingStarters, defendingStarters []*models.Player) {
	rng := e.rng()
	defendingSide := "away"
	if e.PossessionTeam != "home" {
		defendingSide = "home"
	}
	if e.PossessionTeam == "home" {
		e.HomeShots++
	} else {
		e.AwayShots++
	}

	if rng.Float64() < 0.06 {
		e.ResolvePenalty(attackingClub, attackingStarters)
		return
	}

	onPitch := e.OnPitch(attackingStarters)
	cands := make([]*models.Player, 0, len(onPitch))
	for _, p := range onPitch {
		if p.Category == "FWD" || p.Category == "MID" {
			cands = append(cands, p)
		}
	}
	if len(cands) == 0 {
		cands = onPitch
	}
	if len(cands) == 0 {
		return
	}
	weights := make([]float64, len(cands))
	for i, p := range cands {
		w := math.Pow(float64(p.OVR)/75.0, 2)
		if p.Category == "FWD" {
			w *= 3.0
		} else {
			w *= 1.2
		}
		weights[i] = w
	}
	shooter := cands[matchreport.WeightedChoice(rng, weights)]

	defendingOnPitch := e.OnPitch(defendingStarters)
	if len(defendingOnPitch) == 0 {
		return
	}
	var gk *models.Player
	for _, p := range defendingOnPitch {
		if p.Category == "GK" {
			gk = p
			break
		}
	}
	if gk == nil {
		// A short or improvised XI may have no keeper; use an active outfield
		// player as the safe fallback, never a dismissed keeper.
		gk = defendingOnPitch[0]
	}

	targetX := 0.98
	if e.PossessionTeam != "home" {
		targetX = 0.02
	}
	targetY := 0.42 + rng.Float64()*0.16
	e.triggerMovement([2]float64{targetX, targetY}, e.PossessionTeam)
	e.markShotBall(e.PossessionTeam)

	conv := 0.28 + float64(shooter.OVR-gk.OVR)*0.012 + (e.liveStrength(attackingStarters)-e.liveStrength(defendingStarters))*0.008
	attMorale := attackingClub.Morale
	if attMorale > 80 {
		conv += 0.02
		if math.Abs(float64(e.HomeScore-e.AwayScore)) <= 1 {
			conv += 0.01
		}
	} else if attMorale < 40 {
		conv -= 0.02
	}
	if shooter.UniverseWonderkid {
		if shooter.Personality == "big_game_performer" && bigGameContext(e.Competition, e.IsRecognizedDerby) {
			conv += 0.03
		} else if shooter.Personality == "flamboyant_star" && rng.Float64() < 0.25 {
			conv += 0.04
		}
		if shooter.MentorName != "" {
			conv += 0.02
		}
		if shooter.Composure > 75 {
			conv += math.Min(0.04, float64(shooter.Composure-75)*0.002)
		}
	}
	attStance, defStance := e.attDefStances()
	if defStance == "PARK_BUS" {
		conv -= 0.06
	}
	if attStance == "OVERLOAD" {
		conv += 0.03
	}
	conv = math.Max(0.06, math.Min(0.65, conv))

	shotTeam := e.PossessionTeam
	shotMinute := int(e.CurrentMinute)
	shotX := round2(0.76 + rng.Float64()*0.18)
	if shotTeam != "home" {
		shotX = round2(0.06 + rng.Float64()*0.18)
	}
	shotY := round2(0.28 + rng.Float64()*0.44)
	shooterMini := liveMini(shooter)

	if rng.Float64() < 0.78*shootingAccuracyWeatherFactor(e.Weather) {
		if e.PossessionTeam == "home" {
			e.HomeShotsOn++
		} else {
			e.AwayShotsOn++
		}
		if rng.Float64() < conv {
			e.LiveShots = append(e.LiveShots, matchreport.ShotMapItem{
				Minute: shotMinute, Team: shotTeam, Shooter: miniShooter(shooter),
				X: shotX, Y: shotY, XG: round2(conv), Outcome: "goal", IsWonderkid: shooter.UniverseWonderkid,
			})
			if e.PossessionTeam == "home" {
				e.HomeScore++
			} else {
				e.AwayScore++
			}

			var ogScorer *models.Player
			if rng.Float64() < 0.04 {
				var ogCands []*models.Player
				for _, p := range defendingOnPitch {
					if p.Category == "DEF" || p.Category == "GK" {
						ogCands = append(ogCands, p)
					}
				}
				if len(ogCands) == 0 {
					ogCands = defendingOnPitch
				}
				if len(ogCands) > 0 {
					ogScorer = ogCands[rng.Intn(len(ogCands))]
				}
			}

			var assister *models.Player
			if ogScorer == nil && rng.Float64() < 0.72 && len(onPitch) > 1 {
				var aCands []*models.Player
				for _, p := range onPitch {
					if p != shooter {
						aCands = append(aCands, p)
					}
				}
				aWeights := make([]float64, len(aCands))
				for i, p := range aCands {
					mult := 1.5
					if p.Category == "MID" {
						mult = 2.5
					}
					aWeights[i] = float64(p.OVR) / 75.0 * mult
				}
				assister = aCands[matchreport.WeightedChoice(rng, aWeights)]
			}

			e.State = "GOAL_PAUSE"
			e.BannerTimer = 2.0
			scoreStr := fmt.Sprintf("%s %d - %d %s", e.HomeClub.ShortName, e.HomeScore, e.AwayScore, e.AwayClub.ShortName)
			var banner, commText string
			isWK := false
			assistTxt := ""
			if assister != nil {
				assistTxt = fmt.Sprintf(" (Assist: %s)", assister.FullName)
			}
			switch {
			case ogScorer != nil:
				banner = fmt.Sprintf("OWN GOAL %s | %s", ogScorer.FullName, scoreStr)
				commText = fmt.Sprintf("OWN GOAL! %s turns it into his own net! (%s)", ogScorer.FullName, scoreStr)
				isWK = ogScorer.UniverseWonderkid
			case shooter.UniverseWonderkid:
				banner = fmt.Sprintf("%s [%d OVR]%s | %s", shooter.FullName, shooter.OVR, assistTxt, scoreStr)
				mentorTag := ""
				if shooter.MentorName != "" && rng.Float64() < 0.35 {
					mentorTag = fmt.Sprintf(" (mentored by %s)", shooter.MentorName)
				}
				commText = fmt.Sprintf("WONDERKID GOAL! %s%s shows breathtaking class and composure to smash home! (%s)", shooter.FullName, mentorTag, scoreStr)
				isWK = true
			case assister != nil && assister.UniverseWonderkid:
				banner = fmt.Sprintf("%s [%d OVR]%s | %s", shooter.FullName, shooter.OVR, assistTxt, scoreStr)
				commText = fmt.Sprintf("WONDERKID VISION! %s delivers a sublime assist for %s! (%s)", assister.FullName, shooter.FullName, scoreStr)
				isWK = true
			default:
				banner = fmt.Sprintf("%s [%d OVR]%s | %s", shooter.FullName, shooter.OVR, assistTxt, scoreStr)
				commText = fmt.Sprintf("GOAAAL! %s fires %s into the net!%s (%s)", shooter.FullName, attackingClub.ClubName, assistTxt, scoreStr)
			}
			e.Banner = banner
			e.AddCommentary(shotMinute, commText, "GOAL", isWK)

			if ogScorer != nil {
				defSide := "away"
				if e.PossessionTeam != "home" {
					defSide = "home"
				}
				ogMini := liveMini(ogScorer)
				event := e.eventForPlayer(matchreport.MatchEventItem{
					Minute: shotMinute, Display: fmt.Sprintf("%d'", shotMinute),
					Type: "own_goal", Side: defSide, Beneficiary: e.PossessionTeam,
					Scorer: &ogMini, HomeScore: e.HomeScore, AwayScore: e.AwayScore,
				}, ogScorer)
				e.Events = append(e.Events, event)
			} else {
				var aMini *matchreport.MiniPlayer
				if assister != nil {
					m := liveMini(assister)
					aMini = &m
				}
				event := e.eventForPlayer(matchreport.MatchEventItem{
					Minute: shotMinute, Display: fmt.Sprintf("%d'", shotMinute),
					Type: "goal", Side: e.PossessionTeam,
					Scorer: &shooterMini, Assister: aMini,
					HomeScore: e.HomeScore, AwayScore: e.AwayScore,
				}, shooter)
				e.Events = append(e.Events, event)
			}
			if e.PossessionTeam == "home" {
				e.PossessionTeam = "away"
			} else {
				e.PossessionTeam = "home"
			}
		} else {
			e.LiveShots = append(e.LiveShots, matchreport.ShotMapItem{
				Minute: shotMinute, Team: shotTeam, Shooter: miniShooter(shooter),
				X: shotX, Y: shotY, XG: round2(conv), Outcome: "save", IsWonderkid: shooter.UniverseWonderkid,
			})
			e.AddCommentary(shotMinute, fmt.Sprintf("BIG SAVE! %s [%d OVR] makes a magnificent reflex stop to deny %s!", gk.FullName, gk.OVR, shooter.FullName), "SAVE", gk.UniverseWonderkid)
			e.Turnover("goalkeeper parry")
		}
	} else {
		e.LiveShots = append(e.LiveShots, matchreport.ShotMapItem{
			Minute: shotMinute, Team: shotTeam, Shooter: miniShooter(shooter),
			X: shotX, Y: shotY, XG: round2(conv), Outcome: "miss", IsWonderkid: shooter.UniverseWonderkid,
		})
		missText := fmt.Sprintf("%s drags the effort just wide of the post.", shooter.FullName)
		if line := e.personalityMissLine(shooter, shotTeam, shotMinute); line != "" {
			missText = line
		}
		e.AddCommentary(shotMinute, missText, "NORMAL", shooter.UniverseWonderkid)
		e.Turnover("goal kick")
	}

	e.MaybeBookPlayer(defendingStarters, defendingSide)
}

// attDefStances returns the attacking and defending stances for the side
// currently in possession.
func (e *LiveMatchEngine) attDefStances() (att, def string) {
	if e.PossessionTeam == "home" {
		return e.HomeStance, e.AwayStance
	}
	return e.AwayStance, e.HomeStance
}

func miniShooter(p *models.Player) matchreport.ShotShooter {
	return matchreport.ShotShooter{FullName: p.FullName, Position: p.Position, OVR: p.OVR, PlayerID: p.PlayerID}
}

// eventForPlayer makes event identity independent of the consumer. The nested
// MiniPlayer fields remain useful for event-specific data (such as assists and
// substitutions), while the flat identity fields give REST and WebSocket
// clients one consistent player/club contract for every player-led event.
func (e *LiveMatchEngine) eventForPlayer(event matchreport.MatchEventItem, player *models.Player) matchreport.MatchEventItem {
	if player == nil {
		return event
	}
	event.PlayerID = player.PlayerID
	event.PlayerName = player.FullName

	var club *models.Club
	if event.Side == "home" {
		club = e.HomeClub
	} else if event.Side == "away" {
		club = e.AwayClub
	}
	if club != nil {
		event.ClubID = club.ClubID
		event.ClubName = club.ClubName
	} else {
		event.ClubID = player.ClubID
	}
	return event
}

// OnPitch returns starters still involved (sent-off players are excluded).
func (e *LiveMatchEngine) OnPitch(starters []*models.Player) []*models.Player {
	on := make([]*models.Player, 0, len(starters))
	for _, p := range starters {
		if p != nil && e.Bookings[p.PlayerID] < 2 {
			on = append(on, p)
		}
	}
	return on
}

// liveStrength sums active effective ratings against the fixed normal XI
// denominator so a dismissal always reduces manpower strength.
func (e *LiveMatchEngine) liveStrength(starters []*models.Player) float64 {
	var total float64
	for _, p := range e.OnPitch(starters) {
		total += float64(p.EffectiveOVR())
	}
	return total / 11.0
}

// MaybeBookPlayer books a defender/midfielder after a shot (Python: _maybe_book_player).
func (e *LiveMatchEngine) MaybeBookPlayer(defendingStarters []*models.Player, defendingSide ...string) {
	rng := e.rng()
	roll := rng.Float64()
	if roll >= 0.085*weatherCardClimateFactor(e.Weather) {
		return
	}
	live := e.OnPitch(defendingStarters)
	cands := make([]*models.Player, 0, len(live))
	for _, p := range live {
		if p.Category == "DEF" || p.Category == "MID" {
			cands = append(cands, p)
		}
	}
	if len(cands) == 0 {
		cands = live
	}
	if len(cands) == 0 {
		return
	}
	booked := cands[rng.Intn(len(cands))]
	side := "away"
	if len(defendingSide) > 0 && (defendingSide[0] == "home" || defendingSide[0] == "away") {
		side = defendingSide[0]
	} else if len(defendingStarters) > 0 && defendingStarters[0] != nil && e.AwayClub != nil && defendingStarters[0].ClubID == e.AwayClub.ClubID {
		side = "away"
	} else if len(defendingStarters) > 0 && defendingStarters[0] != nil && e.HomeClub != nil && defendingStarters[0].ClubID == e.HomeClub.ClubID {
		side = "home"
	} else if e.PossessionTeam != "home" {
		side = "home"
	}

	priors := e.Bookings[booked.PlayerID]
	secondYellow := priors >= 1
	redThreshold := 0.012
	if e.IsHighHeatDerby {
		redThreshold += 0.03
	}
	color := "yellow"
	sentOff := false
	if secondYellow || roll < redThreshold {
		color = "red"
		sentOff = true
		e.Bookings[booked.PlayerID] = 2
	} else {
		e.Bookings[booked.PlayerID] = 1
	}
	bMini := liveMini(booked)
	detail := ""
	if color == "red" {
		if secondYellow {
			detail = "second_yellow"
		} else {
			detail = "straight_red"
		}
	}
	event := e.eventForPlayer(matchreport.MatchEventItem{
		Minute: int(e.CurrentMinute), Display: fmt.Sprintf("%d'", int(e.CurrentMinute)),
		Type: color, Side: side, Player: &bMini, SentOff: sentOff,
		HomeScore: e.HomeScore, AwayScore: e.AwayScore, Detail: detail,
		Seq: len(e.Events) + 1,
	}, booked)
	e.Events = append(e.Events, event)
	var text string
	if secondYellow {
		text = fmt.Sprintf("SECOND YELLOW! %s is sent off!", booked.FullName)
	} else {
		word := "Yellow card"
		if color == "red" {
			word = "Red card"
		}
		text = fmt.Sprintf("%s for %s after a tactical foul.", word, booked.FullName)
	}
	minute := int(e.CurrentMinute)
	e.AddCommentary(minute, text, "CARD", booked.UniverseWonderkid)
	if sentOff {
		if line := e.dedicatedTenManLine(side, minute); line != "" {
			e.AddCommentary(minute, line, "WONDERKID", true)
		}
	}
}

// ResolvePenalty settles a live spot-kick (Python: _resolve_penalty).
func (e *LiveMatchEngine) ResolvePenalty(attackingClub *models.Club, attackingStarters []*models.Player) {
	rng := e.rng()
	onPitch := e.OnPitch(attackingStarters)
	cands := make([]*models.Player, 0, len(onPitch))
	for _, p := range onPitch {
		if p.Category == "FWD" || p.Category == "MID" {
			cands = append(cands, p)
		}
	}
	if len(cands) == 0 {
		cands = onPitch
	}
	if len(cands) == 0 {
		return
	}
	weights := make([]float64, len(cands))
	for i, p := range cands {
		w := math.Pow(float64(p.OVR)/75.0, 2)
		if p.Category == "FWD" {
			w *= 3.0
		} else {
			w *= 1.2
		}
		weights[i] = w
	}
	taker := cands[matchreport.WeightedChoice(rng, weights)]

	if e.PossessionTeam == "home" {
		e.HomeShots++
	} else {
		e.AwayShots++
	}
	e.AddCommentary(int(e.CurrentMinute), fmt.Sprintf("PENALTY! %s steps up for %s!", taker.FullName, attackingClub.ClubName), "CHANCE", taker.UniverseWonderkid)

	conv := math.Max(0.60, math.Min(0.88, 0.76+float64(taker.OVR-75)*0.005))
	if taker.UniverseWonderkid {
		if taker.MentorName != "" {
			conv = math.Min(0.92, conv+0.02)
		}
		if taker.Composure > 75 {
			conv = math.Min(0.95, conv+math.Min(0.04, float64(taker.Composure-75)*0.002))
		}
	}
	minute := int(e.CurrentMinute)
	shotX := 0.88
	if e.PossessionTeam != "home" {
		shotX = 0.12
	}
	// Spot-kicks are goal-bound ball flights for the canvas.
	goalX := 0.98
	if e.PossessionTeam != "home" {
		goalX = 0.02
	}
	e.triggerMovement([2]float64{goalX, 0.50}, e.PossessionTeam)
	e.markShotBall(e.PossessionTeam)
	tMini := liveMini(taker)
	if rng.Float64() < conv {
		e.LiveShots = append(e.LiveShots, matchreport.ShotMapItem{
			Minute: minute, Team: e.PossessionTeam, Shooter: miniShooter(taker),
			X: shotX, Y: 0.50, XG: 0.76, Outcome: "goal", IsWonderkid: taker.UniverseWonderkid,
		})
		if e.PossessionTeam == "home" {
			e.HomeScore++
			e.HomeShotsOn++
		} else {
			e.AwayScore++
			e.AwayShotsOn++
		}
		e.State = "GOAL_PAUSE"
		e.BannerTimer = 2.0
		scoreStr := fmt.Sprintf("%s %d - %d %s", e.HomeClub.ShortName, e.HomeScore, e.AwayScore, e.AwayClub.ShortName)
		e.Banner = fmt.Sprintf("PENALTY %s | %s", taker.FullName, scoreStr)
		e.AddCommentary(minute, fmt.Sprintf("GOAAAL! %s buries the penalty! (%s)", taker.FullName, scoreStr), "GOAL", taker.UniverseWonderkid)
		event := e.eventForPlayer(matchreport.MatchEventItem{
			Minute: minute, Display: fmt.Sprintf("%d'", minute),
			Type: "penalty", Side: e.PossessionTeam, Scorer: &tMini,
			HomeScore: e.HomeScore, AwayScore: e.AwayScore,
		}, taker)
		e.Events = append(e.Events, event)
	} else {
		e.LiveShots = append(e.LiveShots, matchreport.ShotMapItem{
			Minute: minute, Team: e.PossessionTeam, Shooter: miniShooter(taker),
			X: shotX, Y: 0.50, XG: 0.76, Outcome: "save", IsWonderkid: taker.UniverseWonderkid,
		})
		e.AddCommentary(minute, fmt.Sprintf("MISSED PENALTY! %s drags it wide!", taker.FullName), "SAVE", taker.UniverseWonderkid)
		event := e.eventForPlayer(matchreport.MatchEventItem{
			Minute: minute, Display: fmt.Sprintf("%d'", minute),
			Type: "penalty_miss", Side: e.PossessionTeam, Scorer: &tMini,
			HomeScore: e.HomeScore, AwayScore: e.AwayScore,
		}, taker)
		e.Events = append(e.Events, event)
	}
	e.Turnover("penalty aftermath")
}

// Turnover swaps possession and restarts buildup (Python: _turnover, headless).
func (e *LiveMatchEngine) Turnover(reason string) {
	_ = reason
	if e.PossessionTeam == "home" {
		e.PossessionTeam = "away"
	} else {
		e.PossessionTeam = "home"
	}
	e.Phase = "BUILDUP"
	target := [2]float64{0.75, 0.3 + e.rng().Float64()*0.4}
	if e.PossessionTeam == "home" {
		target[0] = 0.25
	}
	e.BallTarget = Coordinate{X: target[0], Y: target[1]}
	e.triggerMovement(target, e.PossessionTeam)
}

// MaybeSubstitute executes planned changes due on the clock.
func (e *LiveMatchEngine) MaybeSubstitute() {
	minute := int(e.CurrentMinute)
	for i := range e.PlannedSubs {
		if e.PlannedSubs[i].Done {
			continue
		}
		if minute < e.PlannedSubs[i].Minute {
			continue
		}
		e.ExecuteSub(i)
	}
}

// ExecuteSub applies one planned substitution by index.
func (e *LiveMatchEngine) ExecuteSub(i int) {
	if i < 0 || i >= len(e.PlannedSubs) {
		return
	}
	sub := &e.PlannedSubs[i]
	if e.SubstitutionsMade[sub.Side] >= 5 {
		sub.Done = true
		return
	}
	if sub.Out == nil || sub.In == nil || e.Bookings[sub.In.PlayerID] >= 2 {
		sub.Done = true
		return
	}
	var starters []*models.Player
	var club *models.Club
	if sub.Side == "home" {
		starters = e.HomeStarters
		club = e.HomeClub
	} else {
		starters = e.AwayStarters
		club = e.AwayClub
	}
	if e.Bookings[sub.Out.PlayerID] >= 2 {
		sub.Done = true
		return
	}
	idx := -1
	for k, p := range starters {
		if p.PlayerID == sub.Out.PlayerID {
			idx = k
			break
		}
	}
	if idx < 0 {
		sub.Done = true
		return
	}
	starters[idx] = sub.In
	e.syncRadarActor(sub.Side, sub.Out.PlayerID, sub.In)
	if e.SubstitutionsMade == nil {
		e.SubstitutionsMade = map[string]int{"home": 0, "away": 0}
	}
	e.SubstitutionsMade[sub.Side]++
	minute := int(e.CurrentMinute)
	if minute < 1 {
		minute = 1
	}
	outMini := liveMini(sub.Out)
	inMini := liveMini(sub.In)
	event := e.eventForPlayer(matchreport.MatchEventItem{
		Minute: minute, Display: fmt.Sprintf("%d'", minute),
		Type: "sub", Side: sub.Side, PlayerOut: &outMini, PlayerIn: &inMini,
		HomeScore: e.HomeScore, AwayScore: e.AwayScore,
	}, sub.In)
	e.Events = append(e.Events, event)
	short := ""
	if club != nil {
		short = club.ShortName
	}
	e.AddCommentary(minute, fmt.Sprintf("Substitution, %s: %s on for %s.", short, sub.In.FullName, sub.Out.FullName),
		"NORMAL", sub.In.UniverseWonderkid || sub.Out.UniverseWonderkid)
	sub.Done = true
}

// EvaluateGameStateTactics triggers AI dugout adaptations from 58'.
func (e *LiveMatchEngine) EvaluateGameStateTactics() {
	if int(e.CurrentMinute) < 58 || e.State != "PLAYING" {
		return
	}
	diff := e.HomeScore - e.AwayScore
	e.ApplyAIManagerDecision("home", diff, int(e.CurrentMinute))
	e.ApplyAIManagerDecision("away", -diff, int(e.CurrentMinute))
}

func (e *LiveMatchEngine) managerName(side string) string {
	var mgr *managers.ManagerProfile
	var club *models.Club
	if side == "home" {
		mgr, club = e.HomeManager, e.HomeClub
	} else {
		mgr, club = e.AwayManager, e.AwayClub
	}
	if mgr != nil && mgr.Name != "" {
		return mgr.Name
	}
	if club != nil {
		return club.ShortName + " Manager"
	}
	return "Manager"
}

func (e *LiveMatchEngine) stance(side string) string {
	if side == "home" {
		return e.HomeStance
	}
	return e.AwayStance
}

func (e *LiveMatchEngine) setStance(side, stance string) {
	if side == "home" {
		e.HomeStance = stance
	} else {
		e.AwayStance = stance
	}
}

func (e *LiveMatchEngine) starters(side string) []*models.Player {
	if side == "home" {
		return e.HomeStarters
	}
	return e.AwayStarters
}

func (e *LiveMatchEngine) bench(side string) []*models.Player {
	if side == "home" {
		return e.HomeBench
	}
	return e.AwayBench
}

// plannedInUse reports whether a bench player already entered via a done sub.
func (e *LiveMatchEngine) plannedInUse(pid string) bool {
	for _, event := range e.Events {
		if event.Type == "sub" && event.PlayerIn != nil && event.PlayerIn.PlayerID == pid {
			return true
		}
	}
	for _, s := range e.PlannedSubs {
		if s.Done && s.In != nil && s.In.PlayerID == pid {
			return true
		}
	}
	return false
}

func maxOVRPick(pool []*models.Player, wkBoost bool) *models.Player {
	var best *models.Player
	bestScore := -1.0
	for _, p := range pool {
		score := float64(p.OVR)
		if wkBoost && p.UniverseWonderkid {
			score *= 2.0
		}
		if best == nil || score > bestScore {
			best, bestScore = p, score
		}
	}
	return best
}

// ApplyAIManagerDecision applies one dugout decision (Python: _apply_ai_manager_decision).
func (e *LiveMatchEngine) ApplyAIManagerDecision(side string, goalDiff, minute int) {
	rng := e.rng()
	mgrName := e.managerName(side)
	club := e.HomeClub
	if side != "home" {
		club = e.AwayClub
	}
	short := ""
	if club != nil {
		short = club.ShortName
	}
	current := e.stance(side)
	starters := e.starters(side)
	bench := e.bench(side)
	subsMade := e.SubstitutionsMade[side]

	if goalDiff < 0 && minute >= 70 && current != "OVERLOAD" {
		e.setStance(side, "OVERLOAD")
		msg := fmt.Sprintf("TACTICAL OVERLOAD: %s orders %s into an aggressive all-out attacking shape chasing an equalizer!", mgrName, short)
		e.AddCommentary(minute, msg, "TACTICAL", false)
		e.LatestTacticalShift = &TacticalShift{Minute: minute, Team: side, ClubShort: short, Manager: mgrName, Stance: "OVERLOAD", Label: "All-Out Overload", Text: msg}
		if subsMade < 5 {
			var avail []*models.Player
			for _, p := range bench {
				if p != nil && e.Bookings[p.PlayerID] < 2 && (p.Category == "FWD" || p.Category == "MID") && !e.plannedInUse(p.PlayerID) {
					avail = append(avail, p)
				}
			}
			var liveDef []*models.Player
			for _, p := range e.OnPitch(starters) {
				if p.Category == "DEF" {
					liveDef = append(liveDef, p)
				}
			}
			if len(avail) > 0 && len(liveDef) > 0 {
				e.ExecuteDirectSub(side, liveDef[len(liveDef)-1], maxOVRPick(avail, true), minute, "tactical overload")
			}
		}
	} else if goalDiff > 0 && minute >= 78 && current != "PARK_BUS" && goalDiff <= 2 {
		e.setStance(side, "PARK_BUS")
		msg := fmt.Sprintf("DEFENSIVE LOCKDOWN: %s shifts %s to a compact low block to protect the advantage!", mgrName, short)
		e.AddCommentary(minute, msg, "TACTICAL", false)
		e.LatestTacticalShift = &TacticalShift{Minute: minute, Team: side, ClubShort: short, Manager: mgrName, Stance: "PARK_BUS", Label: "Low-Block Lockdown", Text: msg}
		if subsMade < 5 {
			var avail []*models.Player
			for _, p := range bench {
				if p != nil && e.Bookings[p.PlayerID] < 2 && (p.Category == "DEF" || p.Category == "MID") && !e.plannedInUse(p.PlayerID) {
					avail = append(avail, p)
				}
			}
			var liveAtt []*models.Player
			for _, p := range e.OnPitch(starters) {
				if p.Category == "FWD" {
					liveAtt = append(liveAtt, p)
				}
			}
			if len(avail) > 0 && len(liveAtt) > 0 {
				e.ExecuteDirectSub(side, liveAtt[len(liveAtt)-1], maxOVRPick(avail, false), minute, "defensive lockdown")
			}
		}
	} else if minute >= 60 && subsMade < 5 && rng.Float64() < 0.18 {
		var booked []*models.Player
		for _, p := range e.OnPitch(starters) {
			if e.Bookings[p.PlayerID] == 1 && (p.Category == "DEF" || p.Category == "MID") {
				booked = append(booked, p)
			}
		}
		if len(booked) > 0 {
			target := booked[0]
			var avail []*models.Player
			for _, p := range bench {
				if p != nil && e.Bookings[p.PlayerID] < 2 && p.Category == target.Category && !e.plannedInUse(p.PlayerID) {
					avail = append(avail, p)
				}
			}
			if len(avail) > 0 {
				e.ExecuteDirectSub(side, target, avail[0], minute, "card risk hook")
				e.AddCommentary(minute, fmt.Sprintf("TACTICAL HOOK: %s removes booked %s to eliminate red card risk.", mgrName, target.FullName), "TACTICAL", false)
			}
		}
	}
}

// ExecuteDirectSub applies an AI-driven substitution immediately.
func (e *LiveMatchEngine) ExecuteDirectSub(side string, outP, inP *models.Player, minute int, reason string) {
	if outP == nil || inP == nil || e.Bookings[outP.PlayerID] >= 2 || e.Bookings[inP.PlayerID] >= 2 {
		return
	}
	var starters []*models.Player
	var club *models.Club
	if side == "home" {
		starters = e.HomeStarters
		club = e.HomeClub
	} else {
		starters = e.AwayStarters
		club = e.AwayClub
	}
	idx := -1
	for k, p := range starters {
		if p.PlayerID == outP.PlayerID {
			idx = k
			break
		}
	}
	if idx < 0 {
		return
	}
	starters[idx] = inP
	e.syncRadarActor(side, outP.PlayerID, inP)
	if e.SubstitutionsMade == nil {
		e.SubstitutionsMade = map[string]int{"home": 0, "away": 0}
	}
	e.SubstitutionsMade[side]++
	outMini := liveMini(outP)
	inMini := liveMini(inP)
	event := e.eventForPlayer(matchreport.MatchEventItem{
		Minute: minute, Display: fmt.Sprintf("%d'", minute),
		Type: "sub", Side: side, PlayerOut: &outMini, PlayerIn: &inMini,
		HomeScore: e.HomeScore, AwayScore: e.AwayScore, Reason: reason,
	}, inP)
	e.Events = append(e.Events, event)
	isWK := inP.UniverseWonderkid || outP.UniverseWonderkid
	reasonTxt := ""
	if reason != "" {
		reasonTxt = fmt.Sprintf(" (%s)", reason)
	}
	short := ""
	if club != nil {
		short = club.ShortName
	}
	cat := "TACTICAL"
	if isWK {
		cat = "WONDERKID"
	}
	e.AddCommentary(minute, fmt.Sprintf("Tactical Change, %s: %s on for %s%s.", short, inP.FullName, outP.FullName, reasonTxt), cat, isWK)
}

// InstantSimulate fast-forwards the remainder at maximum speed.
func (e *LiveMatchEngine) InstantSimulate() {
	for e.CurrentMinute < 90.0 && e.State != "FULL_TIME" {
		if e.State == "GOAL_PAUSE" {
			e.Banner = ""
			e.State = "PLAYING"
			e.Phase = "BUILDUP"
		}
		e.CurrentMinute += 2.5
		e.MaybeSubstitute()
		e.EvaluateGameStateTactics()
		e.AdvancePhase()
	}
	e.CurrentMinute = 90.0
	e.State = "FULL_TIME"
	e.BallHeight = 0
	e.BallIsShot = false
	e.AddCommentary(90, fmt.Sprintf("FULL TIME (Instant Sim)! Final: %s %d - %d %s.", e.HomeClub.ShortName, e.HomeScore, e.AwayScore, e.AwayClub.ShortName), "FULLTIME", false)
}

// initialPossessionWeight is the reset seed (1.0 per side). Weighted share
// engages only above it, i.e. once the sim has recorded territory.
const initialPossessionWeight = 2.0

// thirdWeight scores one possession tick by territory using the sim's
// home-perspective thirds (x<0.35 defensive, <=0.65 midfield, else
// attacking, mirroring the touch-heatmap zones): sterile possession in your
// own box counts half, front-foot possession in the attacking third counts
// half as much again.
func thirdWeight(ballX float64, home bool) float64 {
	switch {
	case ballX < 0.35:
		if home {
			return 0.5
		}
		return 1.5
	case ballX <= 0.65:
		return 1.0
	default:
		if home {
			return 1.5
		}
		return 0.5
	}
}

// HomePossessionPct derives the home share from territory-weighted
// possession once the sim is running; before any territory is recorded it
// falls back to raw tick counts (50/50 on fresh engines).
func (e *LiveMatchEngine) HomePossessionPct() int {
	if e.HomePossessionWeighted+e.AwayPossessionWeighted > initialPossessionWeight {
		total := e.HomePossessionWeighted + e.AwayPossessionWeighted
		if total <= 0 {
			return 50
		}
		return int(e.HomePossessionWeighted / total * 100)
	}
	total := e.HomePossessionTicks + e.AwayPossessionTicks
	if total == 0 {
		return 50
	}
	return int(float64(e.HomePossessionTicks) / float64(total) * 100)
}

// AwayPossessionPct mirrors the home share to 100.
func (e *LiveMatchEngine) AwayPossessionPct() int {
	return 100 - e.HomePossessionPct()
}

// BuildLivePayload extracts score, events, XIs, and counters into a report
// payload (Python: live_match_payload; live HT counts goal/penalty only).
func (e *LiveMatchEngine) BuildLivePayload() matchreport.InstantPayload {
	rng := e.rng()
	weather := e.Weather
	if weather == "" {
		weather = "clear"
		e.Weather = weather
	}
	events := make([]matchreport.MatchEventItem, 0, len(e.Events))
	for i, ev := range e.Events {
		if ev.Display == "" {
			ev.Display = fmt.Sprintf("%d'", ev.Minute)
		}
		if ev.Seq == 0 {
			ev.Seq = i
		}
		events = append(events, ev)
	}
	stats := matchreport.BuildStats(
		e.HomeClub, e.AwayClub,
		e.HomeShots, e.AwayShots, e.HomeShotsOn, e.AwayShotsOn,
		e.HomePossessionPct(), e.HomeCorners, e.AwayCorners,
		events, weather, rng,
	)
	htH, htA := 0, 0
	for _, ev := range events {
		if ev.Minute > 45 {
			continue
		}
		switch ev.Type {
		case "goal", "penalty":
			if ev.Side == "home" {
				htH++
			} else {
				htA++
			}
		case "own_goal":
			if ev.Beneficiary == "home" {
				htH++
			} else {
				htA++
			}
		}
	}
	cap := 50000
	if e.HomeClub != nil && e.HomeClub.StadiumCapacity > 0 {
		cap = e.HomeClub.StadiumCapacity
	}
	attendance := int(math.Max(12000, math.Min(float64(cap), float64(cap)*(0.76+rng.Float64()*0.21))))

	homeXI := e.HomeKickoffXI
	if len(homeXI) == 0 {
		homeXI = e.HomeStarters
	}
	awayXI := e.AwayKickoffXI
	if len(awayXI) == 0 {
		awayXI = e.AwayStarters
	}
	shotMap := matchreport.GenerateShotMap(e.HomeClub, e.AwayClub, events,
		e.HomeShots, e.AwayShots, e.HomeShotsOn, e.AwayShotsOn, rng, e.LiveShots)
	heatmap := matchreport.GenerateTouchHeatmap(e.HomeClub, e.AwayClub, e.HomePossessionPct(), rng, e.LiveTouches)
	press := matchreport.GeneratePressConference(e.HomeClub, e.AwayClub, e.HomeScore, e.AwayScore, events,
		e.managerName("home"), e.managerName("away"))

	return matchreport.InstantPayload{HomeGoals: e.HomeScore, AwayGoals: e.AwayScore,
		HomeClubName: e.HomeClub.ClubName, AwayClubName: e.AwayClub.ClubName,
		Events: events,
		HomeXI: homeXI, AwayXI: awayXI,
		HomeBench: e.HomeBench, AwayBench: e.AwayBench,
		Stats:  stats,
		HTHome: htH, HTAway: htA,
		Attendance: attendance,
		Referee:    matchreport.PickRefereeName(rng, ""),
		Weather:    weather,
		ShotMap:    shotMap, Heatmap: heatmap, Press: press,
	}
}
