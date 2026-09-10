package transfers

import "football_sim/pkg/models"

// IsCanonicalWonderkid reports whether p belongs to the fixed 12-player
// franchise-prodigy pool. The rule is stable-ID based and independent of
// current club, OVR, display position, or UniverseWonderkid flags on regens.
func IsCanonicalWonderkid(p *models.Player) bool {
	return isCanonicalWonderkid(p)
}

// IsDesignatedSuperLeagueClub reports whether a club is part of the fixed
// 12-club ecosystem in which canonical wonderkids are allowed to transfer.
func IsDesignatedSuperLeagueClub(clubID string) bool {
	return isSuperLeagueClub(clubID)
}
