package datamanager

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"football_sim/pkg/growth"
	"football_sim/pkg/models"
)

// AcademyFirst contains 54 realistic international first names for youth academy graduates.
var AcademyFirst = []string{
	"Luka", "Mateo", "Noah", "Elio", "Jonas", "Hugo", "Theo", "Kai",
	"Nico", "Omar", "Ilya", "Rafa", "Enzo", "Malik", "Soren", "Pavel",
	"Gabriel", "Lucas", "Julian", "Liam", "Leo", "Milan", "Florian", "Felix",
	"Adrian", "Dante", "Bastien", "Thiago", "Valentin", "Arda", "Kenan", "Stefan",
	"Dominik", "Matias", "Alvaro", "Ruben", "Jesper", "Rasmus", "Hampus", "Elias",
	"Tariq", "Zayn", "Amir", "Kacper", "Jakub", "Tomas", "Filip", "Oliver",
	"Arthur", "Maxime", "Samir", "Danilo", "Dario", "Leon",
}

// ACADEMY_FIRST provides a Python-compatible alias.
var ACADEMY_FIRST = AcademyFirst

// AcademyLast contains 54 realistic international surnames for youth academy graduates.
var AcademyLast = []string{
	"Vermeer", "Bergström", "Kovács", "Moretti", "Diallo", "Petrov",
	"Okafor", "Hassan", "Duarte", "Nakamura", "Ibrahim", "Castelo",
	"Horvath", "Nielsen", "Kozlov", "Ferreira", "Lindqvist", "Camara",
	"Mendes", "Svensson", "Gomez", "Ricci", "Fontana", "Bauer",
	"Schneider", "Novak", "Varga", "Popov", "Jovanovic", "Demir",
	"Yilmaz", "Aydin", "Santos", "Ribeiro", "Morales", "Navarro",
	"Larsson", "Holm", "Bakker", "De Jong", "Kowalski", "Wisniewski",
	"Dimitrov", "Stankovic", "Toure", "Mensah", "Traore", "Keita",
	"Soler", "Vidal", "Costa", "Perez", "Romero", "Benali",
}

// ACADEMY_LAST provides a Python-compatible alias.
var ACADEMY_LAST = AcademyLast

// AcademyPositions lists the 11 possible pitch positions for academy regens.
var AcademyPositions = []string{"GK", "CB", "LB", "RB", "CDM", "CM", "CAM", "LW", "RW", "ST", "CF"}

// ACADEMY_POS provides a Python-compatible alias.
var ACADEMY_POS = AcademyPositions

// RegenPersonalities lists the 4 core archetypes assigned to youth regens.
var RegenPersonalities = []string{"dedicated_pro", "flamboyant_star", "academic_dual", "big_game_performer"}

// REGEN_PERSONALITIES provides a Python-compatible alias.
var REGEN_PERSONALITIES = RegenPersonalities

// AcademySignee pairs a newly graduated academy player with their club.
type AcademySignee struct {
	Player *models.Player
	Club   *models.Club
}

// RunYouthIntake generates 2-4 academy graduates for the specified club (or all clubs if empty/"all").
// Squad size is strictly capped at 34 players. Enrolls new players into the GrowthEngine.
func (dm *DataManager) RunYouthIntake(clubID string) ([]*models.Player, error) {
	return dm.RunYouthIntakeWithCount(clubID, 0)
}

// RunYouthIntakeWithCount generates academy graduates with an optional explicit count per club.
// If count <= 0, a random roll between 2 and 4 is used per club.
func (dm *DataManager) RunYouthIntakeWithCount(clubID string, count int) ([]*models.Player, error) {
	var targetClubs []*models.Club
	if clubID == "" || strings.EqualFold(clubID, "all") {
		targetClubs = dm.ClubsList
	} else {
		club, ok := dm.Clubs[clubID]
		if !ok {
			return nil, fmt.Errorf("club %q not found", clubID)
		}
		targetClubs = []*models.Club{club}
	}

	var countPtr *int
	if count > 0 {
		c := count
		countPtr = &c
	}

	allClubs := dm.ClubsList
	if len(allClubs) == 0 {
		allClubs = targetClubs
	}

	return runYouthIntakeInternal(targetClubs, allClubs, countPtr, dm.GrowthEngine, dm.rng)
}

// RunYouthIntakeClubs generates youth academy graduates for a slice of clubs.
func RunYouthIntakeClubs(clubs []*models.Club, count *int, ge *growth.GrowthEngine, rng *rand.Rand) ([]*models.Player, error) {
	return runYouthIntakeInternal(clubs, clubs, count, ge, rng)
}

func runYouthIntakeInternal(
	clubs []*models.Club,
	globalClubs []*models.Club,
	count *int,
	ge *growth.GrowthEngine,
	rng *rand.Rand,
) ([]*models.Player, error) {
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	taken := make(map[string]bool)
	for _, c := range globalClubs {
		if c == nil {
			continue
		}
		for _, p := range c.Squad {
			if p != nil && p.FullName != "" {
				taken[strings.ToLower(p.FullName)] = true
			}
		}
	}

	signed := make([]*models.Player, 0)

	for _, club := range clubs {
		if len(club.Squad) >= 34 {
			continue
		}

		hasGoldenGen := rng.Float64() < 0.20
		gradCount := 2 + rng.Intn(3) // 2, 3, or 4
		if count != nil && *count > 0 {
			gradCount = *count
		}

		for i := 0; i < gradCount; i++ {
			if len(club.Squad) >= 34 {
				break
			}

			var name string
			for attempt := 0; attempt < 40; attempt++ {
				first := AcademyFirst[rng.Intn(len(AcademyFirst))]
				last := AcademyLast[rng.Intn(len(AcademyLast))]
				cand := first + " " + last
				if !taken[strings.ToLower(cand)] {
					name = cand
					break
				}
			}
			if name == "" {
				first := AcademyFirst[rng.Intn(len(AcademyFirst))]
				last := AcademyLast[rng.Intn(len(AcademyLast))]
				suffix := 10 + rng.Intn(90)
				name = fmt.Sprintf("%s %s %d", first, last, suffix)
			}
			taken[strings.ToLower(name)] = true

			pos := AcademyPositions[rng.Intn(len(AcademyPositions))]
			age := 16 + rng.Intn(3) // 16, 17, or 18
			pers := RegenPersonalities[rng.Intn(len(RegenPersonalities))]

			var ovr, potential int
			if hasGoldenGen && i == 0 {
				ovr = 72 + rng.Intn(7)       // 72 to 78
				potential = 90 + rng.Intn(6) // 90 to 95
			} else {
				ovr = 58 + rng.Intn(15)      // 58 to 72
				potential = 75 + rng.Intn(18) // 75 to 92
			}

			randSuffix := 1000 + rng.Intn(9000)
			safeName := strings.ReplaceAll(name, " ", "_")
			playerID := fmt.Sprintf("AC_%s_%s_%d", club.ClubID, safeName, randSuffix)

			player := &models.Player{
				PlayerID:          playerID,
				FullName:          name,
				Position:          pos,
				Category:          models.GetPositionCategory(pos),
				OVR:               ovr,
				Age:               age,
				MarketValueEUR:    models.BaselineValue(ovr, age, false),
				UniverseWonderkid: false,
				PlayerSource:      "academy",
				ContractYears:     2 + rng.Intn(3),
				Loyalty:           70 + rng.Intn(21),
				Personality:       pers,
				ClubID:            club.ClubID,
				OriginalClubID:    club.ClubID,
				Education:         "none",
				SchoolWant:        models.SchoolWantFor(name),
			}
			player.WageEUR = models.WageForOVR(player.OVR)
			models.ClampPlayer(player)

			club.Squad = append(club.Squad, player)
			club.SquadSize = len(club.Squad)

			if ge != nil {
				h := float64(172 + rng.Intn(19)) // 172 to 190 cm
				w := float64(65 + rng.Intn(18))  // 65 to 82 kg
				ge.RegisterProdigy(
					player.PlayerID,
					player.FullName,
					player.Age,
					h,
					w,
					player.Category,
					player.OVR,
					potential,
					19,
				)
			}

			signed = append(signed, player)
		}
	}

	return signed, nil
}
