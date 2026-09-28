package brain

import (
	"football_sim/pkg/managers"
	"football_sim/pkg/models"
)

// MatchFeatures builds the model's input row for one fixture, home
// perspective. Pure: reads club and manager state, mutates nothing.
// The same builder is used for training (against the actual result) and
// for inference (the learned tactical edge), so the brain always thinks
// about the same world it learned from.
func MatchFeatures(home, away *models.Club, homeMgr, awayMgr *managers.ManagerProfile) [NumFeatures]float64 {
	var f [NumFeatures]float64
	f[FeatureHomeAdvantage] = 1.0

	if home != nil && away != nil {
		f[FeatureRatingDiff] = float64(home.OverallTeamRating-away.OverallTeamRating) / 10.0
		f[FeatureFormDiff] = float64(formPoints(home)-formPoints(away)) / 15.0
		f[FeatureMoraleDiff] = float64(home.Morale-away.Morale) / 100.0
	}

	homeStyle, awayStyle := "", ""
	if homeMgr != nil {
		homeStyle = homeMgr.Style
	}
	if awayMgr != nil {
		awayStyle = awayMgr.Style
	}
	if homeStyle != "" && awayStyle != "" && homeStyle != awayStyle {
		if managers.StyleBeats[homeStyle] == awayStyle {
			f[FeatureHomeBeats] = 1.0
		}
		if managers.StyleBeats[awayStyle] == homeStyle {
			f[FeatureAwayBeats] = 1.0
		}
	}

	if homeMgr != nil && awayMgr != nil {
		hp := managers.PersonalityForManager(homeMgr.Style, homeMgr.Name)
		ap := managers.PersonalityForManager(awayMgr.Style, awayMgr.Name)
		f[FeaturePressDiff] = float64(hp.PressIntensity-ap.PressIntensity) / 10.0
		f[FeatureRiskDiff] = hp.RiskAppetite - ap.RiskAppetite
	}
	return f
}

// StyleEdgeFeatures isolates the style-matchup signal: the row the brain
// is asked about when the engine needs a tactical edge. Everything except
// the two style indicators is zero, so the prediction is purely what the
// brain has learned about this philosophical pairing.
func StyleEdgeFeatures(homeStyle, awayStyle string) [NumFeatures]float64 {
	var f [NumFeatures]float64
	if homeStyle != "" && awayStyle != "" && homeStyle != awayStyle {
		if managers.StyleBeats[homeStyle] == awayStyle {
			f[FeatureHomeBeats] = 1.0
		}
		if managers.StyleBeats[awayStyle] == homeStyle {
			f[FeatureAwayBeats] = 1.0
		}
	}
	return f
}

// formPoints sums the club's last five results (W=3, D=1).
func formPoints(club *models.Club) int {
	points := 0
	counted := 0
	for _, result := range club.Form {
		if counted >= 5 {
			break
		}
		switch result {
		case "W", "w":
			points += 3
		case "D", "d":
			points++
		}
		counted++
	}
	return points
}
