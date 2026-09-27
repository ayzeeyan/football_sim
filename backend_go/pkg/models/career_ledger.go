package models

import "sort"

const (
	RegistrationRegistered = "REGISTERED"
	RegistrationFreeAgent  = "FREE_AGENT"

	ContractEventSigned   = "signed"
	ContractEventRenewed  = "renewed"
	ContractEventExpired  = "expired"
	ContractEventReleased = "released"

	MovePermanent = "Permanent"
	MoveFree      = "Free"
	MoveLoan      = "Loan"
	MoveReturn    = "Returned"
)

// PlayerSeasonRecord is one compact, authoritative season (or competition) line.
type PlayerSeasonRecord struct {
	Season        string  `json:"season"`
	ClubID        string  `json:"club_id,omitempty"`
	ClubName      string  `json:"club_name,omitempty"`
	CompetitionID string  `json:"competition_id,omitempty"`
	Appearances   int     `json:"appearances"`
	Starts        int     `json:"starts"`
	Minutes       int     `json:"minutes"`
	Goals         int     `json:"goals"`
	Assists       int     `json:"assists"`
	CleanSheets   int     `json:"clean_sheets,omitempty"`
	OVR           int     `json:"ovr"`
	AvgRating     float64 `json:"avg_rating,omitempty"`
}

// PlayerMoveRecord is a chronological club-to-club move. Empty fields stay empty
// when the historical save never recorded them.
type PlayerMoveRecord struct {
	Season     string `json:"season,omitempty"`
	FromClubID string `json:"from_club_id,omitempty"`
	ToClubID   string `json:"to_club_id,omitempty"`
	Type       string `json:"type"`
	FeeEUR     int64  `json:"fee_eur,omitempty"`
	Matchweek  int    `json:"matchweek,omitempty"`
}

// PlayerContractEvent records a real contract lifecycle event going forward.
type PlayerContractEvent struct {
	Season string `json:"season,omitempty"`
	ClubID string `json:"club_id,omitempty"`
	Kind   string `json:"kind"`
	Years  int    `json:"years,omitempty"`
}

// PlayerHonourRecord is a trophy or individual award actually won.
type PlayerHonourRecord struct {
	Season string `json:"season,omitempty"`
	Title  string `json:"title"`
	Kind   string `json:"kind,omitempty"`
}

// IsFreeAgent reports an unattached player. ClubID is empty and the player
// must not sit in any club squad.
func (p *Player) IsFreeAgent() bool {
	if p == nil {
		return false
	}
	if p.RegistrationStatus == RegistrationFreeAgent {
		return true
	}
	return p.ClubID == "" && p.RegistrationStatus != RegistrationRegistered
}

// MarkFreeAgent detaches the player from a club without deleting identity.
func (p *Player) MarkFreeAgent(previousClubID string) {
	if p == nil {
		return
	}
	if previousClubID != "" {
		p.PreviousClubID = previousClubID
	} else if p.ClubID != "" {
		p.PreviousClubID = p.ClubID
	}
	p.ClubID = ""
	p.RegistrationStatus = RegistrationFreeAgent
	p.ContractYears = 0
	p.OnLoan = false
	p.ParentClubID = ""
	p.LoanBuyClauseEUR = 0
	p.IsCaptain = false
	p.IsViceCaptain = false
	p.RegisteredEurope = false
}

// MarkRegistered attaches a free agent (or existing player) to a club.
func (p *Player) MarkRegistered(clubID string) {
	if p == nil {
		return
	}
	p.ClubID = clubID
	p.RegistrationStatus = RegistrationRegistered
}

// RecordContractEvent appends a compact contract-history row.
func (p *Player) RecordContractEvent(season, clubID, kind string, years int) {
	if p == nil || kind == "" {
		return
	}
	p.ContractHistory = append(p.ContractHistory, PlayerContractEvent{
		Season: season, ClubID: clubID, Kind: kind, Years: years,
	})
}

// RecordMove appends a compact transfer-history row.
func (p *Player) RecordMove(season, fromID, toID, moveType string, fee int64, matchweek int) {
	if p == nil || moveType == "" {
		return
	}
	p.TransferHistory = append(p.TransferHistory, PlayerMoveRecord{
		Season: season, FromClubID: fromID, ToClubID: toID, Type: moveType, FeeEUR: fee, Matchweek: matchweek,
	})
}

// RecordHonour appends a trophy or award that was actually won.
func (p *Player) RecordHonour(season, title, kind string) {
	if p == nil || title == "" {
		return
	}
	p.AwardsHistory = append(p.AwardsHistory, PlayerHonourRecord{Season: season, Title: title, Kind: kind})
}

// ArchiveSeason appends compact season lines from live stats. It never invents
// historical rows for seasons that were not simulated.
func (p *Player) ArchiveSeason(season, clubID, clubName string) {
	if p == nil || season == "" {
		return
	}
	avg := 0.0
	if n := len(p.RecentRatings); n > 0 {
		sum := 0.0
		for _, r := range p.RecentRatings {
			sum += r
		}
		avg = float64(int(sum/float64(n)*100+0.5)) / 100
	}
	starts, minutes := 0, 0
	if len(p.CompetitionStats) > 0 {
		ids := make([]string, 0, len(p.CompetitionStats))
		for id := range p.CompetitionStats {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			st := p.CompetitionStats[id]
			if st == nil {
				continue
			}
			starts += st.Starts
			minutes += st.Minutes
			p.SeasonHistory = append(p.SeasonHistory, PlayerSeasonRecord{
				Season:        season,
				ClubID:        clubID,
				ClubName:      clubName,
				CompetitionID: st.CompetitionID,
				Appearances:   st.Appearances,
				Starts:        st.Starts,
				Minutes:       st.Minutes,
				Goals:         st.Goals,
				Assists:       st.Assists,
				OVR:           p.OVR,
			})
		}
	}
	if p.Appearances > 0 || p.Goals > 0 || p.Assists > 0 || starts > 0 {
		p.SeasonHistory = append(p.SeasonHistory, PlayerSeasonRecord{
			Season:      season,
			ClubID:      clubID,
			ClubName:    clubName,
			Appearances: p.Appearances,
			Starts:      starts,
			Minutes:     minutes,
			Goals:       p.Goals,
			Assists:     p.Assists,
			CleanSheets: p.CleanSheets,
			OVR:         p.OVR,
			AvgRating:   avg,
		})
	}
}
