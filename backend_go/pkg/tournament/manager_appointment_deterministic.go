package tournament

import (
	"math/rand"
	"sort"
	"strings"

	"football_sim/pkg/managers"
	"football_sim/pkg/models"
)

// appointManagerDeterministic mirrors managers.AppointManager but never builds
// an RNG-indexed choice list from Go map iteration. Any choice list whose
// order can affect the seeded universe is sorted before an RNG draw.
func appointManagerDeterministic(managerMap map[string]*managers.ManagerProfile, club *models.Club, rng *rand.Rand) (*managers.ManagerProfile, *managers.ManagerProfile) {
	if club == nil || rng == nil {
		return nil, nil
	}
	old := managerMap[club.ClubID]
	if old == nil {
		return nil, nil
	}

	used := make(map[string]bool, len(managerMap))
	for _, m := range managerMap {
		if m != nil {
			used[m.Name] = true
		}
	}

	availableNames := make([]string, 0, len(managers.ManagerPool))
	for _, name := range managers.ManagerPool {
		if !used[name] {
			availableNames = append(availableNames, name)
		}
	}
	if len(availableNames) == 0 {
		for _, name := range append(append([]string{}, managers.ReplacementNames...), managers.ManagerNames...) {
			if !used[name] {
				availableNames = append(availableNames, name)
			}
		}
	}

	chosenName := "Interim Head Coach"
	if len(availableNames) > 0 {
		chosenName = availableNames[rng.Intn(len(availableNames))]
	} else {
		parts := strings.Fields(old.Name)
		if len(parts) > 0 {
			chosenName = "Interim " + parts[len(parts)-1]
		}
	}

	styles := make([]string, 0, len(managers.TacticalArchetypes))
	oldStyle := old.CanonicalStyle()
	for style := range managers.TacticalArchetypes {
		if style != oldStyle {
			styles = append(styles, style)
		}
	}
	sort.Strings(styles)
	if len(styles) == 0 {
		styles = []string{"free_flowing", "high_press", "low_block", "possession"}
	}

	focuses := make([]string, 0, len(managers.FocusLabels))
	for focus := range managers.FocusLabels {
		if focus != old.Focus {
			focuses = append(focuses, focus)
		}
	}
	sort.Strings(focuses)
	if len(focuses) == 0 {
		focuses = []string{"balance", "stars", "youth"}
	}

	next := &managers.ManagerProfile{
		ClubID:       club.ClubID,
		Name:         chosenName,
		Style:        styles[rng.Intn(len(styles))],
		Focus:        focuses[rng.Intn(len(focuses))],
		BudgetEur:    old.BudgetEur,
		Adaptability: 75 + rng.Intn(18),
		JobSecurity:  "Safe",
	}
	managerMap[club.ClubID] = next
	return old, next
}
