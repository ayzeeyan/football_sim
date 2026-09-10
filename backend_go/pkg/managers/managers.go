package managers

import (
	"math"
	"math/rand"
	"strings"

	"football_sim/pkg/models"
)

// TacticalArchetype defines an AI manager's overarching tactical philosophy.
type TacticalArchetype struct {
	Key            string `json:"key"`
	Title          string `json:"title"`
	ShortName      string `json:"short_name"`
	Badge          string `json:"badge"`
	Description    string `json:"description"`
	LineHeight     string `json:"line_height"`
	PressIntensity string `json:"press_intensity"`
	Tempo          string `json:"tempo"`
	Beats          string `json:"beats"`
}

// TacticalArchetypes defines the 4 core dogmas.
var TacticalArchetypes = map[string]TacticalArchetype{
	"high_press": {
		Key:            "high_press",
		Title:          "The High-Press Dogmatist",
		ShortName:      "High-Press",
		Badge:          "Gegenpress",
		Description:    "Ultra-high defensive line, relentless counter-pressing, early stamina burn, high risk/reward.",
		LineHeight:     "high",
		PressIntensity: "extreme",
		Tempo:          "fast",
		Beats:          "possession",
	},
	"possession": {
		Key:            "possession",
		Title:          "The Positional Mastermind",
		ShortName:      "Positional Play",
		Badge:          "Tiki-Taka / Positional",
		Description:    "Patient passing build-up, strict positional discipline, inverted fullbacks, overloading half-spaces.",
		LineHeight:     "medium_high",
		PressIntensity: "structured",
		Tempo:          "controlled",
		Beats:          "low_block",
	},
	"low_block": {
		Key:            "low_block",
		Title:          "The Low-Block Pragmatist",
		ShortName:      "Low-Block Counter",
		Badge:          "Low-Block & Transition",
		Description:    "Compact defense, deep defensive line, tactical fouls, rapid transition counters, game-killing clock control.",
		LineHeight:     "deep",
		PressIntensity: "low_block",
		Tempo:          "direct",
		Beats:          "free_flowing",
	},
	"free_flowing": {
		Key:            "free_flowing",
		Title:          "The Free-Flowing Attacker",
		ShortName:      "Fluid Attacking",
		Badge:          "Free-Flowing Flair",
		Description:    "Creative freedom, fluid attacking rotations, heavy reliance on wonderkid brilliance and individual magic.",
		LineHeight:     "medium",
		PressIntensity: "moderate",
		Tempo:          "dynamic",
		Beats:          "high_press",
	},
}

var CanonicalStyles = map[string]string{
	"press":        "high_press",
	"high_press":   "high_press",
	"possession":   "possession",
	"counter":      "low_block",
	"low_block":    "low_block",
	"free_flowing": "free_flowing",
}

var StyleBeats = map[string]string{
	"high_press":   "possession",
	"possession":   "low_block",
	"low_block":    "free_flowing",
	"free_flowing": "high_press",
	"press":        "possession",
	"counter":      "press",
}

var StyleLabels = map[string]string{
	"high_press":   "Gegenpress",
	"possession":   "Positional play",
	"low_block":    "Low-block counter",
	"free_flowing": "Free-flowing attack",
	"press":        "Gegenpress",
	"counter":      "Low-block counter",
}

var ManagerNames = []string{
	"Carlo Benedetti",
	"Jürgen Köhler",
	"Rafael Duarte",
	"Erik Lindqvist",
	"Didier Fontaine",
	"Marco Aldana",
	"Andrés Salazar",
	"Tomasz Krajewski",
	"Paulo Mendonça",
	"Gareth Whitmore",
	"Lucien Berger",
	"Santiago Vega",
}

var ManagerPool = []string{
	"Marco Silva", "Unai Emery", "Gian Piero Gasperini", "Roberto De Zerbi",
	"Vincent Kompany", "Xabi Alonso", "Thiago Motta", "Ruben Amorim",
	"Arne Slot", "Simone Inzaghi", "Luis Enrique", "Thomas Frank",
	"Oliver Glasner", "Nuno Espirito Santo", "Andoni Iraola", "Enzo Maresca",
}

var ReplacementNames = []string{
	"Niko Varela",
	"Henrik Solberg",
	"Massimo Ricci",
	"Owen Cartwright",
	"Youssef El-Amin",
	"Piotr Lewandowski",
	"Jean-Luc Moreau",
	"Iker Valdés",
	"Stefan Obrenović",
	"Callum Reilly",
	"Fernando Quintero",
	"Mats Eisenberg",
	"Rui Castelo",
	"Lukas Horváth",
	"Sébastien Roux",
	"Diego Farías",
}

var FocusLabels = map[string]string{
	"youth":   "Wonderkid project",
	"stars":   "Galáctico hunting",
	"balance": "Balanced rebuild",
}

// ClubArchetypeMap presets tactical styles for elite clubs.
var ClubArchetypeMap = map[string][2]string{
	"LAL-RMA": {"free_flowing", "stars"},
	"LAL-BAR": {"possession", "youth"},
	"BUN-BAY": {"high_press", "balance"},
	"EPL-ARS": {"possession", "youth"},
	"EPL-MCI": {"possession", "stars"},
	"EPL-LIV": {"high_press", "balance"},
	"LAL-ATM": {"low_block", "balance"},
	"SEA-INT": {"low_block", "balance"},
	"FRA-PSG": {"free_flowing", "stars"},
	"FL1-PSG": {"free_flowing", "stars"},
	"BUN-DOR": {"high_press", "youth"},
	"SEA-MIL": {"free_flowing", "youth"},
	"SEA-NAP": {"low_block", "balance"},
	"EPL-TOT": {"high_press", "youth"},
}

// ManagerProfile represents a club's AI head coach.
type ManagerProfile struct {
	ClubID       string `json:"club_id"`
	Name         string `json:"name"`
	Style        string `json:"style"`
	Focus        string `json:"focus"`
	BudgetEur    int64  `json:"budget_eur"`
	Adaptability int    `json:"adaptability"`
}

func (m *ManagerProfile) CanonicalStyle() string {
	if s, ok := CanonicalStyles[m.Style]; ok {
		return s
	}
	return "possession"
}

func (m *ManagerProfile) ArchetypeInfo() TacticalArchetype {
	if a, ok := TacticalArchetypes[m.CanonicalStyle()]; ok {
		return a
	}
	return TacticalArchetypes["possession"]
}

func (m *ManagerProfile) DogmaTitle() string {
	return m.ArchetypeInfo().Title
}

func (m *ManagerProfile) Tactic() string {
	return m.ArchetypeInfo().Badge
}

func (m *ManagerProfile) FocusLabel() string {
	if f, ok := FocusLabels[m.Focus]; ok {
		return f
	}
	return m.Focus
}

func (m *ManagerProfile) WageBill(club *models.Club) int64 {
	var total int64
	for _, p := range club.Squad {
		total += int64(p.WageEUR)
	}
	return total * 52
}

func (m *ManagerProfile) WageCap(club *models.Club) int64 {
	return int64(150_000_000 + (club.OverallTeamRating-78)*25_000_000)
}

func (m *ManagerProfile) CanAfford(club *models.Club, fee int64, wageEUR int) bool {
	if fee > m.BudgetEur {
		return false
	}
	return m.WageBill(club)+int64(wageEUR)*52 <= m.WageCap(club)
}

func (m *ManagerProfile) WeakestLine(club *models.Club) string {
	bestLine := "MID"
	bestAvg := math.MaxFloat64

	for _, line := range []string{"DEF", "MID", "FWD"} {
		var members []*models.Player
		for _, p := range club.Squad {
			if p.Category == line {
				members = append(members, p)
			}
		}
		if len(members) < 2 {
			continue
		}
		var sum int
		for _, p := range members {
			sum += p.OVR
		}
		avg := float64(sum) / float64(len(members))
		if avg < bestAvg {
			bestAvg = avg
			bestLine = line
		}
	}
	return bestLine
}

// BuildManagers initializes AI manager profiles for the clubs.
func BuildManagers(clubs []*models.Club) map[string]*ManagerProfile {
	rng := rand.New(rand.NewSource(20260704))
	names := make([]string, len(ManagerNames))
	copy(names, ManagerNames)
	rng.Shuffle(len(names), func(i, j int) { names[i], names[j] = names[j], names[i] })

	styles := []string{"high_press", "possession", "low_block", "free_flowing"}
	focuses := []string{"youth", "stars", "balance"}
	managers := make(map[string]*ManagerProfile)

	for i, club := range clubs {
		rating := club.OverallTeamRating
		if rating <= 0 {
			rating = 80
		}
		defaultStyle := styles[i%len(styles)]
		defaultFocus := focuses[i%len(focuses)]

		if preset, ok := ClubArchetypeMap[club.ClubID]; ok {
			defaultStyle = preset[0]
			defaultFocus = preset[1]
		}

		budget := int64(60_000_000)
		if diff := rating - 78; diff > 0 {
			budget += int64(diff * 12_000_000)
		}

		adaptability := 80 + (rating-75)*2 + rng.Intn(8) - 3
		if adaptability > 99 {
			adaptability = 99
		} else if adaptability < 60 {
			adaptability = 60
		}

		managers[club.ClubID] = &ManagerProfile{
			ClubID:       club.ClubID,
			Name:         names[i%len(names)],
			Style:        defaultStyle,
			Focus:        defaultFocus,
			BudgetEur:    budget,
			Adaptability: adaptability,
		}
	}
	return managers
}

// AppointManager appoints a new manager to a club, replacing the previous one.
func AppointManager(managers map[string]*ManagerProfile, club *models.Club, rng *rand.Rand) (*ManagerProfile, *ManagerProfile) {
	if rng == nil {
		rng = rand.New(rand.NewSource(rand.Int63()))
	}
	old, ok := managers[club.ClubID]
	if !ok || old == nil {
		return nil, nil
	}

	used := make(map[string]bool)
	for _, m := range managers {
		used[m.Name] = true
	}

	var availablePool []string
	for _, n := range ManagerPool {
		if !used[n] {
			availablePool = append(availablePool, n)
		}
	}
	if len(availablePool) == 0 {
		for _, n := range append(ReplacementNames, ManagerNames...) {
			if !used[n] {
				availablePool = append(availablePool, n)
			}
		}
	}

	chosenName := "Interim Head Coach"
	if len(availablePool) > 0 {
		chosenName = availablePool[rng.Intn(len(availablePool))]
	} else {
		parts := strings.Split(old.Name, " ")
		chosenName = "Interim " + parts[len(parts)-1]
	}

	var styles []string
	for s := range TacticalArchetypes {
		if s != old.Style {
			styles = append(styles, s)
		}
	}
	if len(styles) == 0 {
		styles = []string{"possession", "high_press", "low_block", "free_flowing"}
	}

	var focuses []string
	for f := range FocusLabels {
		if f != old.Focus {
			focuses = append(focuses, f)
		}
	}
	if len(focuses) == 0 {
		focuses = []string{"youth", "stars", "balance"}
	}

	newMgr := &ManagerProfile{
		ClubID:       club.ClubID,
		Name:         chosenName,
		Style:        styles[rng.Intn(len(styles))],
		Focus:        focuses[rng.Intn(len(focuses))],
		BudgetEur:    old.BudgetEur,
		Adaptability: 75 + rng.Intn(18),
	}
	managers[club.ClubID] = newMgr
	return old, newMgr
}

// TacticEdge calculates the tactical edge nudge for the home side from the matchup.
// Returns +0.16 if home beats away, -0.12 if away beats home, or 0.0 otherwise.
func TacticEdge(homeStyle, awayStyle string) float64 {
	h := homeStyle
	if c, ok := CanonicalStyles[homeStyle]; ok {
		h = c
	}
	a := awayStyle
	if c, ok := CanonicalStyles[awayStyle]; ok {
		a = c
	}
	if h == "" || a == "" || h == a {
		return 0.0
	}
	if StyleBeats[h] == a {
		return 0.16
	}
	if StyleBeats[a] == h {
		return -0.12
	}
	return 0.0
}
