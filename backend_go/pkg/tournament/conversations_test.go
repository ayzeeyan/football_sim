package tournament

import (
	"testing"

	"football_sim/pkg/models"
)

func TestPlayerConversationReplyChangesMorale(t *testing.T) {
	club := &models.Club{ClubID: "C1", ClubName: "Test", ShortName: "TST", Played: 10}
	p := &models.Player{PlayerID: "P1", FullName: "Talker", ClubID: "C1", SquadRole: models.RoleCrucial, Appearances: 1, Morale: 60}
	club.Squad = []*models.Player{p}
	tm := &TournamentManager{Clubs: map[string]*models.Club{"C1": club}, ClubsList: []*models.Club{club}, SeasonName: "2026-27"}
	tm.maybePlayerConversationsUnlocked(6)
	if len(tm.Inbox) == 0 || len(tm.Inbox[0].Choices) == 0 {
		t.Fatalf("expected a conversation, inbox=%#v", tm.Inbox)
	}
	item := tm.Inbox[0]
	out := tm.ReplyInboxUnlocked(item.ID, "promise")
	if out["status"] != "success" {
		t.Fatalf("reply failed: %#v", out)
	}
	if p.Morale <= 60 {
		t.Fatalf("promise should lift morale, got %d", p.Morale)
	}
	if !tm.Inbox[0].Resolved {
		t.Fatal("conversation should close after a reply")
	}
}

func TestCompetitionTalksNeedEuropeanFootball(t *testing.T) {
	mkVet := func() *models.Player {
		return &models.Player{
			PlayerID: "VET1", FullName: "Fringe Veteran", ClubID: "C1",
			SquadRole: models.RoleRotation, Appearances: 12, Morale: 60, Age: 28, OVR: 76,
			ContractYears: 3,
		}
	}
	// Same player, same season shape: only the European flag differs.
	if got := conversationKindWithClub(mkVet(), 12, true); got != "competition" {
		t.Fatalf("european fringe veteran kind=%q want competition", got)
	}
	if got := conversationKindWithClub(mkVet(), 12, false); got != "" {
		t.Fatalf("non-european fringe veteran kind=%q want none", got)
	}
}

func TestContractAndLoanRepliesNudgeWithoutDialogueGame(t *testing.T) {
	contractor := &models.Player{PlayerID: "K1", FullName: "Expiring", ContractYears: 1, Appearances: 10, OVR: 80, Morale: 60, Loyalty: 60}
	if got := conversationKindWithClub(contractor, 12, false); got != "contract" {
		t.Fatalf("expiring regular kind=%q want contract", got)
	}
	msg := applyConversationChoice(contractor, "extend")
	if contractor.Morale != 66 || contractor.TransferRequested {
		t.Fatalf("extend should lift morale and clear requests: morale=%d requested=%v msg=%q", contractor.Morale, contractor.TransferRequested, msg)
	}
	loanee := &models.Player{PlayerID: "K2", FullName: "Kid", Age: 19, SquadRole: models.RoleProspect, Appearances: 1, Morale: 60}
	if got := conversationKindWithClub(loanee, 12, false); got != "loan" {
		t.Fatalf("unused prospect kind=%q want loan", got)
	}
	applyConversationChoice(loanee, "loan")
	if loanee.Morale != 65 || loanee.TransferRequested {
		t.Fatalf("loan promise should nudge morale only: morale=%d requested=%v", loanee.Morale, loanee.TransferRequested)
	}
}
