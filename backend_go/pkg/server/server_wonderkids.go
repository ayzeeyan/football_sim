package server

import (
	"net/http"

	"football_sim/pkg/growth"
	"football_sim/pkg/models"
	"football_sim/pkg/tournament"
)

// Wonderkid lab, school tracks, mentorship and growth endpoints.
func (s *Server) handleGetProdigies(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	list := []map[string]interface{}{}
	for _, p := range s.DataManager.Wonderkids {
		data, ok := s.GrowthEngine.GetProdigyData(p.PlayerID, p.Category)
		if !ok {
			continue
		}
		club := s.TournamentManager.Clubs[p.ClubID]
		if club != nil {
			data["club_name"] = club.ClubName
			data["club_short"] = club.ShortName
			data["primary_color"] = club.PrimaryColor
		} else {
			data["club_name"] = p.ClubID
			data["club_short"] = "UNK"
			data["primary_color"] = [3]uint8{200, 200, 200}
		}
		data["formatted_value"] = p.FormattedValue()
		data["goals"] = p.Goals
		data["assists"] = p.Assists
		data["appearances"] = p.Appearances
		data["position"] = p.Position
		data["education"] = p.Education
		data["education_label"] = p.EducationLabel()
		data["education_pending"] = p.EducationPending
		data["school_want"] = p.SchoolWant
		data["school_want_label"] = p.SchoolWantLine()
		data["school_track"] = p.SchoolTrack
		data["school_track_label"] = p.SchoolTrackLabel()
		data["secondary_position"] = p.SecondaryPosition
		data["position_path"] = p.PositionPath
		data["position_xp"] = p.PositionXP
		data["position_options"] = p.PositionOptions()
		data["career_goals"] = p.Goals + p.CareerGoals
		data["career_assists"] = p.Assists + p.CareerAssists
		data["career_apps"] = p.Appearances + p.CareerApps
		data["best_goals"] = p.BestGoals
		data["best_assists"] = p.BestAssists
		data["best_season"] = p.BestSeason
		data["personality"] = p.Personality
		data["personality_title"] = p.PersonalityTitle()
		data["personality_badge"] = p.PersonalityBadge()
		data["personality_desc"] = p.PersonalityInfo().Description
		data["mentor_id"] = p.MentorID
		data["mentor_name"] = p.MentorName
		data["mentor_ovr"] = p.MentorOVR
		data["composure"] = p.Composure
		list = append(list, data)
	}
	s.worldMu.RUnlock()
	writeJSON(w, list)
}

func (s *Server) handleGetProdigyWatch(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	payload := map[string]interface{}{
		"season_name": s.TournamentManager.SeasonName,
		"matchweek":   s.TournamentManager.CurrentMatchweek,
		"rankings":    s.TournamentManager.GetProdigyWatch(),
	}
	s.worldMu.RUnlock()
	writeJSON(w, payload)
}

func (s *Server) handleGetWonderkids(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	wks := s.DataManager.Wonderkids
	var topScorer, topAssister, topOVR *models.Player
	total := int64(0)
	serialized := []map[string]interface{}{}
	for _, p := range wks {
		serialized = append(serialized, s.serializePlayer(p))
		total += p.MarketValueEUR
		if topScorer == nil || p.Goals > topScorer.Goals {
			topScorer = p
		}
		if topAssister == nil || p.Assists > topAssister.Assists {
			topAssister = p
		}
		if topOVR == nil || p.OVR > topOVR.OVR {
			topOVR = p
		}
	}
	payload := map[string]interface{}{
		"top_scorer":      s.serializePlayer(topScorer),
		"top_assister":    s.serializePlayer(topAssister),
		"top_ovr":         s.serializePlayer(topOVR),
		"total_valuation": total,
		"wonderkids":      serialized,
	}
	s.worldMu.RUnlock()
	writeJSON(w, payload)
}

func (s *Server) handleGetGrowthMilestones(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	milestones := append([]growth.GrowthMilestone{}, s.GrowthEngine.Milestones...)
	s.worldMu.RUnlock()
	writeJSON(w, milestones)
}

// narrativeMilestonesFor builds the eight career achievements the timeline
// tab renders (id/title/description/unlocked/badge). Thresholds mirror the
// client fallback so both agree on what "unlocked" means.
func narrativeMilestonesFor(bio *growth.BiometricProfile, p *models.Player) []map[string]interface{} {
	goals, assists, apps, ovr, gain := 0, 0, 0, 0, 0.0
	if bio != nil {
		ovr = bio.BaselineOVR
		gain = bio.HeightGainCM()
	}
	if p != nil {
		ovr = p.OVR
		goals = p.Goals + p.CareerGoals
		assists = p.Assists + p.CareerAssists
		apps = p.Appearances + p.CareerApps
	}
	mk := func(id, title, description, badge string, unlocked bool) map[string]interface{} {
		return map[string]interface{}{
			"id": id, "title": title, "description": description,
			"unlocked": unlocked, "badge": badge,
		}
	}
	return []map[string]interface{}{
		mk("first_contract", "First Professional Contract", "Signed official youth forms with senior team at age 14.", "gold", true),
		mk("growth_spurt", "Puberty Growth Spurt", "Grew frame since the August baseline.", "cyan", gain >= 1.0),
		mk("senior_debut", "Senior Debut", "Earned first team match minutes in the European Super League.", "green", apps > 0),
		mk("first_goal", "First European Goal", "Slotted the ball home against elite continental competition.", "gold", goals >= 1),
		mk("youth_cap", "Youth International Cap", "Earned national youth honors after surpassing 78 OVR threshold.", "purple", ovr >= 78),
		mk("elite_playmaker", "Elite Playmaker", "Created 5+ career assists orchestrating offensive attacks.", "cyan", assists >= 5),
		mk("century_mark", "Trophy & Goal Contender", "Reached 15+ combined career goals and assists.", "gold", goals+assists >= 15),
		mk("ballon_dor_contender", "Ballon d'Or Candidate", "Surpassed 85 OVR ascending to world-class elite status.", "gold", ovr >= 85),
	}
}

func (s *Server) handleGetProdigyTimeline(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	pid := r.PathValue("player_id")
	bio := s.GrowthEngine.Biometrics[pid]
	var payload map[string]interface{}
	if bio != nil {
		p, club := s.findPlayer(pid)
		history := s.GrowthEngine.GetProgressionHistory(pid)
		if len(history) == 0 {
			ovr, clubShort := bio.BaselineOVR, "UNK"
			if p != nil {
				ovr = p.OVR
			}
			if club != nil {
				clubShort = club.ShortName
			}
			history = []growth.TimelineEntry{{
				Season: s.TournamentManager.SeasonName, Age: bio.Age, OVR: ovr,
				HeightCM: bio.CurrentHeightCM, WeightKG: bio.CurrentWeightKG,
				ClubShort: clubShort, MentorName: bio.MentorName,
			}}
		}
		ovr := bio.BaselineOVR
		if p != nil {
			ovr = p.OVR
		}
		payload = map[string]interface{}{
			"player_id":           pid,
			"full_name":           bio.FullName,
			"age":                 bio.Age,
			"current_height_cm":   bio.CurrentHeightCM,
			"baseline_height_cm":  bio.BaselineHeightCM,
			"current_weight_kg":   bio.CurrentWeightKG,
			"baseline_weight_kg":  bio.BaselineWeightKG,
			"height_gain_cm":      bio.HeightGainCM(),
			"weight_gain_kg":      bio.WeightGainKG(),
			"ovr":                 ovr,
			"potential":           bio.Potential,
			"progression_history": history,
			"milestones":          narrativeMilestonesFor(bio, p),
		}
	}
	s.worldMu.RUnlock()

	if bio == nil {
		http.Error(w, "Prodigy not found", http.StatusNotFound)
		return
	}
	writeJSON(w, payload)
}

func (s *Server) handleGetTrainingStatus(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	defer s.worldMu.RUnlock()

	writeJSON(w, map[string]interface{}{
		"training_energy":     s.GrowthEngine.TrainingEnergy,
		"max_training_energy": s.GrowthEngine.MaxTrainingEnergy,
	})
}

// handleGetTrainingProjection returns the read-only "what the staff do"
// plan for any player. Prodigies are additionally trainable through the
// interactive planner (POST /api/prodigies/{player_id}/train).
func (s *Server) handleGetTrainingProjection(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	defer s.worldMu.RUnlock()

	pid := r.PathValue("player_id")
	player, club := s.findPlayer(pid)
	if player == nil {
		http.Error(w, "Player not found", http.StatusNotFound)
		return
	}
	trainable := s.GrowthEngine.Biometrics[pid] != nil
	proj := growth.ProjectTrainingWeek(player, trainable)
	payload := map[string]interface{}{
		"player_id":           pid,
		"player_name":         player.FullName,
		"club_id":             club.ClubID,
		"focus":               proj.Focus,
		"rationale":           proj.Rationale,
		"projected_gains":     proj.ProjectedGains,
		"trainable":           proj.Trainable,
		"training_energy":     s.GrowthEngine.TrainingEnergy,
		"max_training_energy": s.GrowthEngine.MaxTrainingEnergy,
	}
	writeJSON(w, payload)
}

func (s *Server) handleGetNXGN50(w http.ResponseWriter, r *http.Request) {
	s.worldMu.RLock()
	rankings := tournament.GenerateNXGN50(s.TournamentManager.ClubsList, s.GrowthEngine)
	payload := map[string]interface{}{
		"status":   "success",
		"rankings": rankings,
	}
	s.worldMu.RUnlock()
	writeJSON(w, payload)
}
