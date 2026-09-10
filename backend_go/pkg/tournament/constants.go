package tournament

const (
	LeagueRounds = 44
	UCLFinalWeek = 44
)

var MonthBands = []struct {
	Lo, Hi int
	Name   string
}{
	{1, 4, "August"},
	{5, 8, "September"},
	{9, 12, "October"},
	{13, 16, "November"},
	{17, 20, "December"},
	{21, 24, "January"},
	{25, 28, "February"},
	{29, 33, "March"},
	{34, 38, "April"},
	{39, 44, "May"},
}

func LeaguePhase(matchweek int) string {
	if matchweek <= 11 {
		return "Opening series"
	}
	if matchweek <= 22 {
		return "Return series"
	}
	if matchweek <= 33 {
		return "Third series"
	}
	return "Final stretch"
}

func MonthLabel(matchweek int) string {
	for _, b := range MonthBands {
		if matchweek >= b.Lo && matchweek <= b.Hi {
			return b.Name
		}
	}
	return "Season"
}

func WeekChapter(matchweek int) string {
	month := MonthLabel(matchweek)
	uclGroup := containsInt(UCLGroupWeeks, matchweek)
	uclQF := containsInt(UCLQFWeeks, matchweek)
	uclSF := containsInt(UCLSFWeeks, matchweek)
	uclFinal := matchweek == UCLFinalWeek
	sc := false
	for _, w := range SuperCupWeeks {
		if w == matchweek {
			sc = true
			break
		}
	}
	switch {
	case uclFinal:
		return month + ": Champions Cup final night"
	case matchweek == SuperCupWeeks["final"]:
		return month + ": Super Cup final"
	case (uclGroup || uclQF || uclSF) && sc:
		return month + ": title pace vs cup congestion"
	case uclQF:
		return month + ": Champions Cup quarters"
	case uclSF:
		return month + ": Champions Cup semis"
	case uclGroup:
		return month + ": European group night"
	case sc:
		return month + ": Super Cup night"
	case matchweek <= 4:
		return month + ": opening series"
	case matchweek >= 21 && matchweek <= 24:
		return month + ": winter window"
	case matchweek >= 39:
		return month + ": home stretch"
	default:
		return month + ": " + toLowerFirst(LeaguePhase(matchweek))
	}
}

func containsInt(xs []int, v int) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

func toLowerFirst(s string) string {
	if s == "" {
		return s
	}
	b := []byte(s)
	if b[0] >= 'A' && b[0] <= 'Z' {
		b[0] = b[0] + ('a' - 'A')
	}
	return string(b)
}

var (
	UCLGroupWeeks   = []int{3, 8, 13, 19, 24}
	UCLQFWeeks      = []int{35, 36}
	UCLSFWeeks      = []int{39, 40}
	SuperCupWeeks   = map[string]int{"play_in": 5, "qf": 12, "sf": 20, "final": 26}
	WeatherOptions  = []string{"clear", "clear", "clear", "overcast", "rain", "rain", "wind", "snow"}
)

var DerbyNames = map[string]string{
	"LAL-BAR_LAL-RMA": "El Clasico",
	"LAL-RMA_LAL-BAR": "El Clasico",
	"LAL-ATM_LAL-RMA": "Madrid derby",
	"LAL-RMA_LAL-ATM": "Madrid derby",
	"LAL-ATM_LAL-BAR": "Spanish derby",
	"LAL-BAR_LAL-ATM": "Spanish derby",
	"EPL-ARS_EPL-TOT": "North London derby",
	"EPL-TOT_EPL-ARS": "North London derby",
	"SEA-INT_SEA-MIL": "Derby della Madonnina",
	"SEA-MIL_SEA-INT": "Derby della Madonnina",
	"SEA-MIL_SEA-NAP": "Derby del Sole",
	"SEA-NAP_SEA-MIL": "Derby del Sole",
	"SEA-INT_SEA-NAP": "Derby d'Italia",
	"SEA-NAP_SEA-INT": "Derby d'Italia",
	"BUN-BAY_BUN-DOR": "Der Klassiker",
	"BUN-DOR_BUN-BAY": "Der Klassiker",
	"EPL-ARS_EPL-LIV": "English heavyweight",
	"EPL-LIV_EPL-ARS": "English heavyweight",
	"EPL-LIV_EPL-TOT": "English heavyweight",
	"EPL-TOT_EPL-LIV": "English heavyweight",
}

type TrophyCounts struct {
	SuperLeague int `json:"super_league"`
	UCL         int `json:"ucl"`
	SuperCup    int `json:"super_cup"`
}

var HistoricalClubTrophies = map[string]TrophyCounts{
	"LAL-RMA": {SuperLeague: 36, UCL: 15, SuperCup: 13},
	"LAL-BAR": {SuperLeague: 27, UCL: 5, SuperCup: 14},
	"BUN-BAY": {SuperLeague: 33, UCL: 6, SuperCup: 10},
	"SEA-MIL": {SuperLeague: 19, UCL: 7, SuperCup: 7},
	"EPL-LIV": {SuperLeague: 19, UCL: 6, SuperCup: 16},
	"SEA-INT": {SuperLeague: 20, UCL: 3, SuperCup: 8},
	"EPL-ARS": {SuperLeague: 13, UCL: 0, SuperCup: 17},
	"LAL-ATM": {SuperLeague: 11, UCL: 0, SuperCup: 5},
	"BUN-DOR": {SuperLeague: 8, UCL: 1, SuperCup: 6},
	"FL1-PSG": {SuperLeague: 12, UCL: 0, SuperCup: 12},
	"FRA-PSG": {SuperLeague: 12, UCL: 0, SuperCup: 12},
	"SEA-NAP": {SuperLeague: 3, UCL: 0, SuperCup: 2},
	"EPL-TOT": {SuperLeague: 2, UCL: 0, SuperCup: 8},
}

// GetDerbyName returns the historic derby name for two clubs, or empty if none.
func GetDerbyName(clubA, clubB string) string {
	key := clubA + "_" + clubB
	if name, ok := DerbyNames[key]; ok {
		return name
	}
	return ""
}
