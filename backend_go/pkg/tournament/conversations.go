package tournament

import (
	"fmt"
	"sort"
	"strings"

	"football_sim/pkg/models"
)

func (tm *TournamentManager) maybePlayerConversationsUnlocked(completedMW int) {
	if tm == nil || completedMW < 6 || completedMW%3 != 0 {
		return
	}
	type cand struct {
		player *models.Player
		club   *models.Club
		kind   string
	}
	cands := make([]cand, 0)
	for _, club := range tm.ClubsList {
		if club == nil {
			continue
		}
		for _, p := range club.Squad {
			if p == nil {
				continue
			}
			kind := conversationKindWithClub(p, club.Played, tm.clubInEuropeUnlocked(club.ClubID))
			if kind == "" {
				continue
			}
			cands = append(cands, cand{player: p, club: club, kind: kind})
		}
	}
	sort.SliceStable(cands, func(i, j int) bool {
		if cands[i].kind != cands[j].kind {
			return cands[i].kind < cands[j].kind
		}
		return cands[i].player.PlayerID < cands[j].player.PlayerID
	})
	for _, c := range cands {
		headline := conversationHeadline(c.player, c.kind)
		already := false
		for _, item := range tm.Inbox {
			if item.PlayerID == c.player.PlayerID && item.Headline == headline && item.SeasonName == tm.SeasonName {
				already = true
				break
			}
		}
		if already {
			continue
		}
		body, choices := conversationBody(c.player, c.club, c.kind)
		id := tm.PushInbox("dugout", headline, body, completedMW, []string{c.club.ClubID}, c.player.PlayerID, "")
		if id == "" {
			continue
		}
		for i := range tm.Inbox {
			if tm.Inbox[i].ID == id {
				tm.Inbox[i].Choices = choices
				break
			}
		}
		return
	}
}

func conversationKind(p *models.Player, clubPlayed int) string {
	return conversationKindWithClub(p, clubPlayed, false)
}

// clubInEuropeUnlocked reports whether a club holds a European place this
// season. Legacy (non-world) careers have no European fields.
func (tm *TournamentManager) clubInEuropeUnlocked(clubID string) bool {
	if tm == nil || tm.World == nil || tm.World.Competitions == nil {
		return false
	}
	for _, eid := range europeanDefinitions {
		if comp := tm.World.Competitions[eid.ID]; comp != nil {
			for _, pid := range comp.ParticipantIDs {
				if pid == clubID {
					return true
				}
			}
		}
	}
	return false
}

// conversationKindWithClub adds contract, loan, and competition-importance
// triggers while keeping the original priority order deterministic.
// Choices only ever nudge morale, loyalty, or transfer requests.
func conversationKindWithClub(p *models.Player, clubPlayed int, inEurope bool) string {
	if p == nil {
		return ""
	}
	if p.TransferRequested {
		return "leave"
	}
	if p.ContractYears <= 1 && p.Appearances >= 3 && (p.OVR >= 76 || p.SquadRole == models.RoleCrucial || p.SquadRole == models.RoleImportant) {
		return "contract"
	}
	// Both playing-time triggers read minutes-weighted time, consistent with
	// development and loans: cameo stacks do not count as involvement.
	played := effectiveAppearances(p)
	if p.Age <= 20 && !p.UniverseWonderkid && (p.SquadRole == models.RoleProspect || p.SquadRole == models.RoleSquad) && clubPlayed >= 8 && played <= clubPlayed/3 {
		return "loan"
	}
	if (p.SquadRole == models.RoleCrucial || p.SquadRole == models.RoleImportant) && clubPlayed >= 8 && played <= clubPlayed/4 {
		return "minutes"
	}
	if p.FormModifier() >= 3 {
		return "form"
	}
	// Bigger-stage talks only happen at clubs actually playing in Europe;
	// otherwise fringe veterans have no European nights to ask for.
	if inEurope && (p.SquadRole == models.RoleRotation || p.SquadRole == models.RoleSquad) && clubPlayed >= 10 && p.Age >= 24 && p.Morale >= 45 && p.Morale <= 70 {
		return "competition"
	}
	if p.Morale >= 85 && p.SquadRole != "" {
		return "happy"
	}
	return ""
}

func conversationHeadline(p *models.Player, kind string) string {
	switch kind {
	case "leave":
		return p.FullName + " wants to leave"
	case "minutes":
		return p.FullName + " asks for more playing time"
	case "form":
		return p.FullName + " is in strong form"
	case "contract":
		return p.FullName + " wants to talk contracts"
	case "loan":
		return p.FullName + " asks about a loan move"
	case "competition":
		return p.FullName + " wants a bigger stage"
	default:
		return p.FullName + " is happy with his role"
	}
}

func conversationBody(p *models.Player, club *models.Club, kind string) (string, []InboxChoice) {
	clubName := ""
	if club != nil {
		clubName = club.ClubName
	}
	switch kind {
	case "leave":
		return fmt.Sprintf("%s has told the %s staff he wants a move. How do you answer?", p.FullName, clubName), []InboxChoice{
			{ID: "promise", Label: "Promise a bigger role"},
			{ID: "sell", Label: "Agree to listen to offers"},
			{ID: "reject", Label: "Tell him to focus"},
		}
	case "minutes":
		return fmt.Sprintf("%s feels his minutes at %s do not match a %s role.", p.FullName, clubName, p.SquadRole), []InboxChoice{
			{ID: "promise", Label: "Promise more starts"},
			{ID: "reject", Label: "The XI is earned"},
		}
	case "form":
		return fmt.Sprintf("%s has been in excellent form for %s and wants recognition.", p.FullName, clubName), []InboxChoice{
			{ID: "praise", Label: "Praise him publicly"},
			{ID: "promise", Label: "Keep him central"},
		}
	case "contract":
		return fmt.Sprintf("%s has a year left and wants clarity on his %s future.", p.FullName, clubName), []InboxChoice{
			{ID: "extend", Label: "Promise a new deal"},
			{ID: "sell", Label: "Listen to offers"},
			{ID: "reject", Label: "Focus on football"},
		}
	case "loan":
		return fmt.Sprintf("%s feels a loan from %s would help his development. How do you answer?", p.FullName, clubName), []InboxChoice{
			{ID: "loan", Label: "Promise a loan"},
			{ID: "reject", Label: "Stay and fight"},
		}
	case "competition":
		return fmt.Sprintf("%s wants more decisive nights for %s — cups and European games.", p.FullName, clubName), []InboxChoice{
			{ID: "promise", Label: "Promise bigger games"},
			{ID: "praise", Label: "Praise professionalism"},
		}
	default:
		return fmt.Sprintf("%s is content at %s and just wanted the manager to know.", p.FullName, clubName), []InboxChoice{
			{ID: "praise", Label: "Thank him"},
		}
	}
}

func (tm *TournamentManager) ReplyInboxUnlocked(itemID, choiceID string) map[string]interface{} {
	if tm == nil || itemID == "" {
		return map[string]interface{}{"status": "error", "message": "Conversation not found."}
	}
	for i := range tm.Inbox {
		item := &tm.Inbox[i]
		if item.ID != itemID {
			continue
		}
		if item.Resolved || len(item.Choices) == 0 {
			return map[string]interface{}{"status": "error", "message": "That conversation is already closed."}
		}
		valid := false
		for _, ch := range item.Choices {
			if ch.ID == choiceID {
				valid = true
				break
			}
		}
		if !valid {
			return map[string]interface{}{"status": "error", "message": "Unknown reply."}
		}
		player, _ := tm.findPlayerUnlocked(item.PlayerID)
		item.Resolved = true
		item.Unread = false
		item.Choices = nil
		kind := promiseKindFromHeadline(item.Headline, choiceID)
		msg := applyConversationChoice(player, choiceID, tm.SeasonName, completedMatchweekOr(item.Matchweek, tm.CurrentMatchweek), kind)
		item.Body = item.Body + " " + msg
		return map[string]interface{}{"status": "success", "message": msg, "item": item}
	}
	return map[string]interface{}{"status": "error", "message": "Conversation not found."}
}

func completedMatchweekOr(itemMW, current int) int {
	if itemMW > 0 {
		return itemMW
	}
	return current
}

func promiseKindFromHeadline(headline, choice string) string {
	h := strings.ToLower(headline)
	switch choice {
	case "loan":
		return "loan"
	case "extend":
		return "contract"
	case "promise":
		if strings.Contains(h, "leave") {
			return "role"
		}
		if strings.Contains(h, "playing time") || strings.Contains(h, "starts") {
			return "minutes"
		}
		if strings.Contains(h, "stage") || strings.Contains(h, "bigger games") {
			return "europe"
		}
		return "minutes"
	default:
		return ""
	}
}

func applyConversationChoice(p *models.Player, choice, season string, matchweek int, kind string) string {
	if p == nil {
		return "The staff noted it."
	}
	stamp := func(k string) {
		if k == "" {
			return
		}
		p.PromiseKind = k
		p.PromiseSeason = season
		p.PromiseMatchweek = matchweek
	}
	switch choice {
	case "promise":
		p.AdjustMorale(6)
		p.TransferRequested = false
		if kind == "" {
			kind = "minutes"
		}
		stamp(kind)
		return p.FullName + " accepts the promise — for now."
	case "extend":
		p.AdjustMorale(6)
		p.TransferRequested = false
		p.Loyalty += 5
		if p.Loyalty > 95 {
			p.Loyalty = 95
		}
		stamp("contract")
		return p.FullName + " welcomes the contract signal."
	case "loan":
		p.AdjustMorale(5)
		p.TransferRequested = false
		stamp("loan")
		return p.FullName + " will wait for a loan opening."
	case "sell":
		p.TransferRequested = true
		p.AdjustMorale(2)
		return p.FullName + " will wait for a move."
	case "reject":
		p.AdjustMorale(-5)
		return p.FullName + " is unhappy with the answer."
	case "praise":
		p.AdjustMorale(4)
		return p.FullName + " leaves the office smiling."
	default:
		return "The staff noted it."
	}
}

func (tm *TournamentManager) findPlayerUnlocked(playerID string) (*models.Player, *models.Club) {
	if playerID == "" {
		return nil, nil
	}
	for _, club := range tm.ClubsList {
		if club == nil {
			continue
		}
		for _, p := range club.Squad {
			if p != nil && p.PlayerID == playerID {
				return p, club
			}
		}
	}
	return nil, nil
}
