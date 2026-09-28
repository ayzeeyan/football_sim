package tournament

import (
	"fmt"

	"football_sim/pkg/managers"
	"football_sim/pkg/models"
)

// ViewerJobRecord is one employment stint in the viewer's managerial career.
type ViewerJobRecord struct {
	ClubID         string `json:"club_id"`
	ClubName       string `json:"club_name"`
	Season         string `json:"season"`
	HiredMatchweek int    `json:"hired_matchweek"`
	EndedMatchweek int    `json:"ended_matchweek"`
	Outcome        string `json:"outcome"` // active, sacked, resigned
}

// ViewerManager is the Tier B manager career: the club the viewer manages,
// the trophies won on the job, and the full employment ledger.
type ViewerManager struct {
	Name           string            `json:"name"`
	ClubID         string            `json:"club_id"`
	HiredSeason    string            `json:"hired_season"`
	HiredMatchweek int               `json:"hired_matchweek"`
	Sackings       int               `json:"sackings"`
	Trophies       []string          `json:"trophies"`
	History        []ViewerJobRecord `json:"history"`
}

// MaxViewerJobRecords bounds the persisted career ledger.
const MaxViewerJobRecords = 30

// ViewerManagerName is the default name for the viewer's manager profile.
const ViewerManagerName = "The Viewer"

// AcceptViewerJob installs the viewer as the manager of one club. The
// previous manager's record is preserved in the new profile's history the
// same way AI appointments preserve it. Caller must hold tm.mu.
func (tm *TournamentManager) AcceptViewerJob(clubID, name string) (*ViewerManager, string) {
	club := tm.Clubs[clubID]
	if club == nil {
		return nil, "Club not found."
	}
	if name == "" {
		name = ViewerManagerName
	}

	// Leave any current job first.
	if tm.ViewerManager != nil && tm.ViewerManager.ClubID != "" {
		tm.resignViewerJobUnlocked("resigned")
	}
	if tm.ViewerManager == nil {
		tm.ViewerManager = &ViewerManager{Name: name, Trophies: []string{}, History: []ViewerJobRecord{}}
	}
	tm.ViewerManager.Name = name

	old := tm.Managers[clubID]
	style, focus := "possession", ""
	if old != nil {
		style, focus = old.Style, old.Focus
	}
	next := &managers.ManagerProfile{
		ClubID:             clubID,
		Name:               name,
		Style:              style,
		Focus:              focus,
		BudgetEur:          club.Finances.TransferBudget,
		JobSecurity:        "Safe",
		AppointedSeason:    tm.SeasonName,
		AppointedMatchweek: tm.CurrentMatchweek + 1,
	}
	if old != nil {
		next.History = append([]managers.ManagerHistoryEntry(nil), old.History...)
		next.History = append(next.History, managers.ManagerHistoryEntry{
			ClubID: clubID, ClubName: club.ClubName, ManagerName: old.Name,
			Style:           old.CanonicalStyle(),
			AppointedSeason: old.AppointedSeason, AppointedMatchweek: old.AppointedMatchweek,
			DepartedSeason: tm.SeasonName, DepartedMatchweek: tm.CurrentMatchweek,
			Reason: "took a new role",
		})
	}
	tm.Managers[clubID] = next
	if tm.TransferEngine != nil && tm.TransferEngine.Managers != nil {
		tm.TransferEngine.Managers[clubID] = next
	}

	tm.ViewerManager.ClubID = clubID
	tm.ViewerManager.HiredSeason = tm.SeasonName
	tm.ViewerManager.HiredMatchweek = tm.CurrentMatchweek + 1
	tm.ViewerManager.History = append(tm.ViewerManager.History, ViewerJobRecord{
		ClubID: clubID, ClubName: club.ClubName, Season: tm.SeasonName,
		HiredMatchweek: tm.CurrentMatchweek + 1, Outcome: "active",
	})
	if len(tm.ViewerManager.History) > MaxViewerJobRecords {
		tm.ViewerManager.History = tm.ViewerManager.History[len(tm.ViewerManager.History)-MaxViewerJobRecords:]
	}

	tm.PushInbox(
		MsgCategoryDugout,
		fmt.Sprintf("%s appoint %s as manager", club.ClubName, name),
		fmt.Sprintf("%s take charge of %s from matchweek %d. The board expects the club's identity to be preserved while results improve.", name, club.ClubName, tm.CurrentMatchweek+1),
		tm.CurrentMatchweek,
		[]string{clubID}, "", "",
	)
	return tm.ViewerManager, ""
}

// resignViewerJobUnlocked closes the active stint without appointing a
// replacement (the caller decides what happens to the chair).
// Caller must hold tm.mu.
func (tm *TournamentManager) resignViewerJobUnlocked(outcome string) {
	vm := tm.ViewerManager
	if vm == nil || vm.ClubID == "" {
		return
	}
	for i := len(vm.History) - 1; i >= 0; i-- {
		if vm.History[i].ClubID == vm.ClubID && vm.History[i].Outcome == "active" {
			vm.History[i].EndedMatchweek = tm.CurrentMatchweek
			vm.History[i].Outcome = outcome
			break
		}
	}
	vm.ClubID = ""
}

// ResignViewerJob steps down from the current job; the club appoints a
// deterministic replacement so the world keeps running.
func (tm *TournamentManager) ResignViewerJob() (*ViewerManager, string) {
	if tm.ViewerManager == nil || tm.ViewerManager.ClubID == "" {
		return tm.ViewerManager, "You do not have a job to leave."
	}
	clubID := tm.ViewerManager.ClubID
	club := tm.Clubs[clubID]
	tm.resignViewerJobUnlocked("resigned")
	if club != nil {
		if old := tm.Managers[clubID]; old != nil {
			if _, next := appointManagerDeterministic(tm.Managers, club, tm.RNG); next != nil {
				next.History = append([]managers.ManagerHistoryEntry(nil), old.History...)
				next.History = append(next.History, managers.ManagerHistoryEntry{
					ClubID: clubID, ClubName: club.ClubName, ManagerName: old.Name,
					Style:           old.CanonicalStyle(),
					AppointedSeason: old.AppointedSeason, AppointedMatchweek: old.AppointedMatchweek,
					DepartedSeason: tm.SeasonName, DepartedMatchweek: tm.CurrentMatchweek,
					Reason: "resigned",
				})
				next.AppointedSeason = tm.SeasonName
				next.AppointedMatchweek = tm.CurrentMatchweek + 1
				tm.Managers[clubID] = next
				if tm.TransferEngine != nil && tm.TransferEngine.Managers != nil {
					tm.TransferEngine.Managers[clubID] = next
				}
			}
		}
		tm.PushInbox(
			MsgCategoryDugout,
			fmt.Sprintf("%s announce managerial departure", club.ClubName),
			fmt.Sprintf("%s step down from %s. The club move quickly to appoint a successor.", tm.ViewerManager.Name, club.ClubName),
			tm.CurrentMatchweek,
			[]string{clubID}, "", "",
		)
	}
	return tm.ViewerManager, ""
}

// RecordViewerSacking closes the viewer's stint when the patience system
// dismisses the club's manager. Caller must hold tm.mu.
func (tm *TournamentManager) RecordViewerSacking(clubID string) {
	if tm.ViewerManager == nil || tm.ViewerManager.ClubID != clubID {
		return
	}
	tm.ViewerManager.Sackings++
	tm.resignViewerJobUnlocked("sacked")
}

// RecordViewerTrophies credits the trophies a club won this season to the
// viewer's career when the viewer was in charge. Caller must hold tm.mu.
func (tm *TournamentManager) RecordViewerTrophies(club *models.Club, trophies []string) {
	if tm.ViewerManager == nil || tm.ViewerManager.ClubID == "" || club == nil || len(trophies) == 0 {
		return
	}
	if club.ClubID != tm.ViewerManager.ClubID {
		return
	}
	for _, trophy := range trophies {
		tm.ViewerManager.Trophies = append(tm.ViewerManager.Trophies, fmt.Sprintf("%s (%s)", trophy, tm.SeasonName))
	}
	if len(tm.ViewerManager.Trophies) > MaxViewerJobRecords {
		tm.ViewerManager.Trophies = tm.ViewerManager.Trophies[len(tm.ViewerManager.Trophies)-MaxViewerJobRecords:]
	}
}

// GetViewerManager returns a copy of the career state.
func (tm *TournamentManager) GetViewerManager() *ViewerManager {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	if tm.ViewerManager == nil {
		return nil
	}
	cp := *tm.ViewerManager
	// Never hand out nil slices: the wire contract types them as arrays
	// and a nil slice would marshal as null.
	cp.Trophies = append([]string{}, tm.ViewerManager.Trophies...)
	cp.History = append([]ViewerJobRecord{}, tm.ViewerManager.History...)
	return &cp
}
