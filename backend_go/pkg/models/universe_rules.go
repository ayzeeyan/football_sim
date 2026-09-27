package models

// MaxSeniorSquadSize is the shared hard roster ceiling used by academy,
// transfer, loan, validation, and long-run simulation systems.
const MaxSeniorSquadSize = 34

// MinSeniorSquadSize is the minimum playable roster retained by permanent
// outgoing moves and replenished by annual academy intake.
const MinSeniorSquadSize = 11

// AcademyIntakeSquadTarget is the soft senior-roster ceiling used by the
// automatic season transition. Clubs can still build larger squads through
// transfers, but academy intake does not silently saturate every club at the
// 34-player emergency hard cap over a long career.
const AcademyIntakeSquadTarget = 28

// IsCanonicalWonderkidID reports whether playerID is one of the 12 canonical
// wonderkid stable IDs. It intentionally does not depend on display order, OVR, age,
// current club, or frontend indexes.
var CanonicalWonderkidIDs = map[string]struct{}{
	"WK_Venjamin_Valerio":         {},
	"WK_Maverick_Cantalejo":       {},
	"WK_Yeshua_Emmanuel_Gocotano": {},
	"WK_Izyan_Levin_Bantol":       {},
	"WK_James_Bernard_Rizon":      {},
	"WK_Reid_Randell_Libatan":     {},
	"WK_Ashle_Zylle_Baguio":       {},
	"WK_Cliergy_Jave_Lanticse":    {},
	"WK_Ezail_Zamora":             {},
	"WK_Earl_Josh_Hernando":       {},
	"WK_Rich_Lorenz_Suico":        {},
	"WK_Jhed_Anthony_Guinita":     {},
}

// DesignatedWonderkidClubIDs is the 12-club ecosystem in which canonical
// wonderkids may permanently move. PSG's canonical dataset ID is FL1-PSG.
var DesignatedWonderkidClubIDs = map[string]struct{}{
	"LAL-BAR": {},
	"LAL-RMA": {},
	"LAL-ATM": {},
	"EPL-ARS": {},
	"EPL-LIV": {},
	"EPL-TOT": {},
	"BUN-BAY": {},
	"BUN-DOR": {},
	"SEA-INT": {},
	"SEA-NAP": {},
	"SEA-MIL": {},
	"FL1-PSG": {},
}

func IsCanonicalWonderkidID(playerID string) bool {
	_, ok := CanonicalWonderkidIDs[playerID]
	return ok
}

func IsDesignatedWonderkidClubID(clubID string) bool {
	_, ok := DesignatedWonderkidClubIDs[clubID]
	return ok
}
