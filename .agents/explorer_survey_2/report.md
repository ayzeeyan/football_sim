# Survey Report: R2 Calendar (44 Matchweeks) & R3 Tactics (Position-Aware Formations and Pitch Coordinates)

**Date**: 2026-09-10  
**Author**: Survey Explorer 2 (R2 Calendar & R3 Tactics Specialist)  
**Scope**: `backend_go` (`pkg/tournament`, `pkg/matchengine`, `pkg/models`, `pkg/server`) and `frontend` (`src/components/`, `src/services/`, `src/App.tsx`, `src/types/`)  

---

## 1. Executive Summary

This report delivers the architectural survey, code locations, root cause analyses, and detailed implementation blueprints for:
1. **R2: 44-Matchweek Calendar & Seasonal Game Volume (~55–60 Games)**:
   - Expanding the Super League calendar from 33 matchweeks (triple round-robin) to **44 matchweeks (quadruple round-robin)** consisting of 4 cycles of 11 rounds across the 12 clubs (264 total league fixtures, 6 fixtures per week slate).
   - Guaranteeing that each pair of clubs plays exactly 4 times with balanced venues (2 home, 2 away) and every league week contains all 6 fixtures with no dropped or truncated matches.
   - Synchronizing Champions Cup (10 games max) and Super Cup (3–4 games max) so deep-run elite clubs play **~57–58 total matches per season** (within the ~55–60 target).
   - Updating calendar strips, month bands (August–May), league phases, and standings calculations across backend and frontend.

2. **R3: Position-Driven Tactical Lineup & Pitch Coordinates**:
   - Eliminating the "generic left-wing clustering" bug where Central Attacking Midfielders (`CAM`), including wonderkids (Cantalejo, Bantol, Baguio), were assigned left-wing coordinates (`{0.58, 0.20}`).
   - Implementing natural position-aware coordinate mapping in `backend_go/pkg/models` and `backend_go/pkg/matchengine`:
     - `CAM`: centrally in attacking midfield (`{0.51, 0.50}`) between CMs (`{0.39, 0.33/0.67}`) and ST (`{0.63, 0.50}`).
     - `CDM`: deep central midfield (`{0.31, 0.50}`).
     - `CM`: central midfield channels (`{0.39, 0.33}` and `{0.39, 0.67}`).
     - Flank players (`LB`, `LWB`, `RB`, `RWB`, `LM`, `RM`, `LW`, `RW`): appropriately along wings.
     - Strikers / Center Forwards (`ST`, `CF`): centrally leading the attack (`{0.63, 0.50}`).
   - Ensuring Away coordinates mirror Home coordinates (`X_away = 1.0 - X_home`, `Y_away = 1.0 - Y_home`) with zero left-wing bias.
   - Updating `LiveMatchEngine` to maintain per-player tactical base coordinates rather than static slice index lookups.
   - Updating `PitchCanvas.tsx`, `MatchDetailModal.tsx`, and lineup HUDs to accurately reflect dynamic tactical positioning.

---

## 2. Requirement R2: 44-Matchweek Calendar & Seasonal Volume

### 2.1 Problem Analysis & Current State
- **Current Calendar Constant**: `backend_go/pkg/tournament/constants.go:4` defines `LeagueRounds = 33` and `UCLFinalWeek = 33`.
- **Current Fixture Generation**: `backend_go/pkg/tournament/fixtures.go:38-152` implements `GenerateLeagueFixtures(clubs, rng)` using a 3-cycle generator:
  - Cycle 1 (MW 1–11): Berger polygon rounds 0..10.
  - Cycle 2 (MW 12–22): Reversed venues.
  - Cycle 3 (MW 23–33): Reuses Cycle 1 pairings with `pickCycle3Venue()` attempting to balance 16 or 17 home games. Total fixtures = 198 (33 * 6).
- **Current Month Bands**: `backend_go/pkg/tournament/constants.go:8-20` bands 33 weeks across August to March (`{29, 33, "March"}`), omitting April and May.
- **Current League Phase**: `LeaguePhase(mw)` returns `"Opening series"` (<=11), `"Return series"` (<=22), or `"Home stretch"` (<=33).
- **Current Slate Resolution**: `backend_go/pkg/tournament/sim.go:52-108` (`slateUnlocked`) queries fixtures for the matchweek. Every league week produces fixtures matching `tm.Fixtures[i].Matchweek == mw`.
- **Current Rollover**: `backend_go/pkg/tournament/sim.go:567-602` (`maybeRolloverUnlocked`) advances `CurrentMatchweek++` once all league matches for `CurrentMatchweek` finish. If `CurrentMatchweek > tm.MaxMatchweeks`, the season finishes and transitions to `transfer_window`.

### 2.2 Mathematical Specifications for 44-Matchweek Quadruple Round-Robin
- **Clubs**: 12 elite clubs ($N = 12$).
- **Opponents per club**: 11 opponents ($N - 1 = 11$).
- **Cycles**: 4 full cycles of 11 rounds each ($4 \times 11 = 44$ matchweeks).
- **Matches per round**: $12 / 2 = 6$ fixtures per matchweek slate.
- **Total league matches in season**: $44 \times 6 = 264$ fixtures (exactly 44 per club).
- **Venue Symmetry**:
  - Cycle 1 (MW 1–11): Home vs Away ($H \to A$).
  - Cycle 2 (MW 12–22): Away vs Home ($A \to H$, reversed).
  - Cycle 3 (MW 23–33): Home vs Away ($H \to A$, matching Cycle 1).
  - Cycle 4 (MW 34–44): Away vs Home ($A \to H$, matching Cycle 2).
  - **Symmetry Proof**: For any pair $(C_i, C_j)$, $C_i$ hosts $C_j$ exactly in Cycle 1 and Cycle 3 (2 home matches); $C_j$ hosts $C_i$ exactly in Cycle 2 and Cycle 4 (2 away matches).
  - Every club plays **exactly 22 home matches and 22 away matches**. Zero venue disparity.

### 2.3 Month Bands Expansion (August to May, 10 Months)
To align with the European football season spanning August through May:
```go
var MonthBands = []struct {
	Lo, Hi int
	Name   string
}{
	{1, 4, "August"},      // 4 weeks
	{5, 8, "September"},   // 4 weeks
	{9, 12, "October"},    // 4 weeks
	{13, 16, "November"},  // 4 weeks
	{17, 20, "December"},  // 4 weeks
	{21, 24, "January"},   // 4 weeks
	{25, 28, "February"},  // 4 weeks
	{29, 33, "March"},     // 5 weeks
	{34, 38, "April"},     // 5 weeks
	{39, 44, "May"},       // 6 weeks
}
```
*Note*: Weeks 1–33 maintain exact backward compatibility with all existing tests and labels. Weeks 34–38 become April; Weeks 39–44 become May.

### 2.4 Cup Schedule & Seasonal Game Volume (~55–60 Matches)
1. **Super Cup Schedule**:
   - 12 clubs seeded 1–12 (Seeds 1–4 receive byes).
   - Play-In: MW 5 (4 matches; seeds 5–12).
   - Quarter-finals: MW 12 (4 matches; seeds 1–4 enter).
   - Semi-finals: MW 20 (2 matches).
   - Final: MW 26 (1 match; February mid-season silverware).
   - Games for finalist: 3 matches (with bye) or 4 matches (via play-in).
2. **Champions Cup (UCL) Schedule**:
   - Group Stage (2 groups of 6, single round-robin): 5 matchweeks across MW 3, 8, 13, 19, 24. (5 matches per club).
   - Knockouts (Top 4 from each group):
     - Quarter-finals (2 legs): MW 35 & MW 36 (2 matches).
     - Semi-finals (2 legs): MW 39 & MW 40 (2 matches).
     - Grand Final (1 match): MW 44 (1 match, final day of season alongside MW 44 league finale).
   - Games for finalist: $5 + 2 + 2 + 1 = 10$ matches.
3. **Total Season Game Volume for Elite Clubs**:
   - League: 44 matches.
   - Champions Cup: 10 matches (finalist) / 9 (semi-finalist) / 7 (quarter-finalist) / 5 (group).
   - Super Cup: 3 to 4 matches (finalist) / 2 to 3 (semi-finalist) / 1 to 2 (quarter-finalist).
   - **Deep-run Finalist Total**: $44 + 10 + 4 = 58$ matches (or $44 + 10 + 3 = 57$ matches with bye).
   - This lands **exactly within the target window of ~55–60 games/season**.

---

## 3. Requirement R3: Position-Driven Tactical Lineup & Pitch Coordinates

### 3.1 The Left-Wing Clustering Bug: Root Cause Analysis
Two interconnected issues caused generic left-wing clustering on the pitch canvas and radar:

1. **Category Flattening in `pkg/models/constants.go:43-58`**:
   `GetPositionCategory(pos)` maps positions to `"GK"`, `"DEF"`, `"MID"`, or `"FWD"`.
   CAM is explicitly categorized as `"FWD"`:
   ```go
   // GetPositionCategory maps a specific position to GK, DEF, MID, or FWD.
   // NOTE: CAM is explicitly categorized as FWD per the match engine design.
   ```
   In `Club.GetStartingEleven()` (`pkg/models/club.go:210-265`), available players are partitioned into `gks`, `defs`, `mids`, `fwds`.
   `sortPlayersForXI(fwds)` sorts forwards by wonderkid status and OVR.
   A wonderkid CAM (e.g., Cantalejo at Real Madrid, Bantol at Arsenal, Baguio at Dortmund) is prioritized and placed at index 0 of `fwds`.
   `startingXI` appends:
   - Index 0: GK
   - Indices 1–4: DEFs
   - Indices 5–7: MIDs
   - Indices 8–10: FWDs
   Therefore, the team's CAM is placed at **Index 8** of `startingXI`.

2. **Hardcoded Array Lookups in `pkg/matchengine/live.go:154-180` and `radarPlayers`**:
   `live.go` defined static coordinate arrays:
   ```go
   var baseHomeCoords = [][2]float64{
       {0.06, 0.50}, // 0: GK
       {0.20, 0.16}, // 1: LB
       {0.18, 0.38}, // 2: CB
       {0.18, 0.62}, // 3: CB
       {0.20, 0.84}, // 4: RB
       {0.32, 0.50}, // 5: CDM
       {0.40, 0.32}, // 6: CM
       {0.40, 0.68}, // 7: CM
       {0.58, 0.20}, // 8: LW  <--- Index 8 is hardcoded to LEFT WING!
       {0.62, 0.50}, // 9: ST
       {0.58, 0.80}, // 10: RW
   }
   ```
   `radarPlayers()` mapped `LivePlayerRadar.X = coords[i][0]` and `Y = coords[i][1]` strictly based on the slice index `i` (0..10), ignoring the player's natural position.
   Consequently, **any CAM starting at index 8 received coordinates `{X: 0.58, Y: 0.20}` (Left Wing)**.
   Furthermore, in `MatchDetailModal.tsx:13-18`, `FULL_PITCH_SLOTS[8]` was also hardcoded to `[22, 106]` (left wing).
   This caused every CAM, plus any actual Left Winger, to cluster on the left flank.

### 3.2 Position-Aware Formation Logic & Coordinates Design

Pitch dimensions use normalized coordinates $[0.0, 1.0] \times [0.0, 1.0]$:
- Length: $X \in [0.0, 1.0]$. Home attacks left-to-right ($0 \to 1$); Away attacks right-to-left ($1 \to 0$).
- Width: $Y \in [0.0, 1.0]$. $Y = 0.50$ is center-pitch. For Home: $Y < 0.50$ is left flank, $Y > 0.50$ is right flank. For Away: $Y > 0.50$ is left flank, $Y < 0.50$ is right flank.

#### Natural Coordinates Mapping Matrix:
| Position | Role | Home Base $(X, Y)$ | Away Base $(X, Y)$ | Tactical Channel |
|---|---|---|---|---|
| **GK** | Goalkeeper | `(0.06, 0.50)` | `(0.94, 0.50)` | Goalmouth |
| **LB** / **LWB** | Left Back / Wingback | `(0.20, 0.16)` | `(0.80, 0.84)` | Left defensive flank |
| **CB** (Left) | Left Center Back | `(0.18, 0.38)` | `(0.82, 0.62)` | Left central defense |
| **CB** (Right) | Right Center Back | `(0.18, 0.62)` | `(0.82, 0.38)` | Right central defense |
| **RB** / **RWB** | Right Back / Wingback | `(0.20, 0.84)` | `(0.80, 0.16)` | Right defensive flank |
| **CDM** | Defensive Midfielder | `(0.31, 0.50)` | `(0.69, 0.50)` | Deep central midfield |
| **CM** (Left) | Left Central Midfielder | `(0.39, 0.33)` | `(0.61, 0.67)` | Left midfield channel |
| **CM** (Right) | Right Central Midfielder | `(0.39, 0.67)` | `(0.61, 0.33)` | Right midfield channel |
| **LM** | Left Midfielder | `(0.42, 0.16)` | `(0.58, 0.84)` | Left wing midfield |
| **RM** | Right Midfielder | `(0.42, 0.84)` | `(0.58, 0.16)` | Right wing midfield |
| **CAM** | Attacking Midfielder | `(0.51, 0.50)` | `(0.49, 0.50)` | **Centrally in attacking midfield** |
| **LW** | Left Winger | `(0.58, 0.18)` | `(0.42, 0.82)` | Left attacking wing |
| **RW** | Right Winger | `(0.58, 0.82)` | `(0.42, 0.18)` | Right attacking wing |
| **ST** / **CF** | Striker / Center Forward | `(0.63, 0.50)` | `(0.37, 0.50)` | **Centrally leading attack** |

#### Dual Striker / Dual CAM / 3-CB Adjustments:
- If 2 strikers start (`ST` + `CF` or two `ST`s): split coordinates to `(0.63, 0.42)` and `(0.63, 0.58)`.
- If 2 CAMs start: split coordinates to `(0.51, 0.38)` and `(0.51, 0.62)`.
- If 2 CDMs start (double pivot): split coordinates to `(0.31, 0.40)` and `(0.31, 0.60)`.
- If 3 CBs start: `(0.18, 0.32)`, `(0.17, 0.50)`, `(0.18, 0.68)`.

---

## 4. Code Locations & Detailed Change Inventory

### 4.1 Backend (`backend_go`)

#### 1. `backend_go/pkg/tournament/constants.go`
- **Line 4**: Change `LeagueRounds = 33` to `LeagueRounds = 44`.
- **Line 5**: Change `UCLFinalWeek = 33` to `UCLFinalWeek = 44`.
- **Lines 8–20**: Expand `MonthBands` to 44 matchweeks covering August through May (Weeks 34–38: April, Weeks 39–44: May).
- **Lines 22–30**: Update `LeaguePhase(matchweek int)`:
  - MW 1–11: `"Opening series"`
  - MW 12–22: `"Return series"`
  - MW 23–33: `"Third series"`
  - MW 34–44: `"Final stretch"`
- **Lines 101–104**: Alight cup knockout weeks:
  - `UCLQFWeeks = []int{35, 36}`
  - `UCLSFWeeks = []int{39, 40}`
  - `UCLFinalWeek = 44`

#### 2. `backend_go/pkg/tournament/fixtures.go`
- **Lines 38–152**: Update `GenerateLeagueFixtures(clubs, rng)`:
  - Replace 3-cycle generation with a 4-cycle loop over `singleCycle`:
    - Cycle 1 (MW 1–11): Berger round fixtures as generated ($H \to A$).
    - Cycle 2 (MW 12–22): Reversed fixtures ($A \to H$).
    - Cycle 3 (MW 23–33): Same pairings as Cycle 1 ($H \to A$).
    - Cycle 4 (MW 34–44): Same pairings as Cycle 2 ($A \to H$).
  - Every matchweek gets exactly 6 fixtures. Total = 264 fixtures.
  - Every pair of clubs plays exactly 4 times (2 home, 2 away).

#### 3. `backend_go/pkg/tournament/season.go`
- **Lines 396–439**: Update `adoptLongSeasonUnlocked`:
  - `target := (len(tm.ClubsList) * LeagueRounds) / 2` calculates 264 fixtures.
  - Saves with < 44 weeks or < 264 fixtures are automatically upgraded, generating the 44-matchweek calendar while preserving any played fixtures.
- **Line 360 & Line 491**: Ensure `MatchweekWeather` loop runs `for mw := 1; mw <= LeagueRounds; mw++` (or `<= 44`).

#### 4. `backend_go/pkg/tournament/tournament.go`
- **Line 143**: Update opening inbox wire string to `"Twelve clubs, 44 matchweeks. Champions Cup and Super Cup share the slate. Watch the kids grow."`

#### 5. `backend_go/pkg/models/club.go`
- **Lines 210–278**: Update `GetStartingEleven(fixture ...string) []*Player`:
  - Implement position-aware starter selection:
    - Pick 1 GK.
    - Pick 4 defenders with preference for flank defenders (`LB`/`LWB`, `RB`/`RWB`) and central defenders (`CB`).
    - Pick midfielders and attackers ensuring natural roles: if a CAM is available (e.g., wonderkid), select CAM for the attacking midfield role; pick CDM/CMs for midfield; pick wingers and central striker.
    - Sort the returned slice in tactical order: `[GK, LB, LCB, RCB, RB, CDM, LCM, RCM, CAM, LW, ST, RW]`.

#### 6. `backend_go/pkg/models/formation.go` (New or in `club.go`/`constants.go`)
- Add `AssignPitchCoordinates(xi []*Player, isAway bool) [][2]float64`:
  - Iterates over the selected 11 starters.
  - Matches each player's natural `Position` (`CAM`, `CDM`, `CM`, `LB`, `RB`, `CB`, `LW`, `RW`, `ST`, `CF`, `GK`).
  - Assigns normalized $(X, Y)$ according to the matrix defined in Section 3.2.
  - Applies symmetrical reflection for Away: `(1.0 - X, 1.0 - Y)`.

#### 7. `backend_go/pkg/matchengine/live.go`
- **Add Fields to `LiveMatchEngine`**:
  - `HomeBaseCoords [][2]float64`
  - `AwayBaseCoords [][2]float64`
- **Lines 220–227 (`initPlayersForXI`)**:
  - Call `AssignPitchCoordinates` to set `HomeBaseCoords` and `AwayBaseCoords`.
  - Pass `HomeBaseCoords` and `AwayBaseCoords` into `radarPlayers`.
- **Lines 350–382 (`Update`)**:
  - Replace `base := baseHomeCoords[i]` with `base := e.HomeBaseCoords[i]`.
  - Replace `base := baseAwayCoords[i]` with `base := e.AwayBaseCoords[i]`.
  - Stance offsets: `OVERLOAD` pushes $X$ toward opponent goal ($+0.08$ for Home, $-0.08$ for Away). `PARK_BUS` drops $X$ toward own goal ($-0.06$ for Home, $+0.06$ for Away).

---

### 4.2 Frontend (`frontend`)

#### 1. `frontend/src/App.tsx`
- **Line 287**: Change `"12 clubs · 33-week Super League"` to `"12 clubs · 44-week Super League"`.

#### 2. `frontend/src/services/api.ts`
- **Lines 189, 225, 353, 559, 658**: Update fallback default `max_matchweeks: 33` to `max_matchweeks: 44`.

#### 3. `frontend/src/components/StandingsTab.tsx`
- **Line 48**: Change pace calculation from `* 33` to `* (league?.max_matchweeks ?? 44)`.
- **Line 238**: Update subtitle to `"Quadruple round-robin, 44 games each. Champions Cup and Super Cup sit on the same slate. Decided games lock."`

#### 4. `frontend/src/components/UclTournamentTab.tsx`
- **Line 126**: Update subtitle to `"Two groups of six across the 44-week calendar..."`

#### 5. `frontend/src/components/MatchDetailModal.tsx`
- **Lines 13–18 & Line 384**:
  - Replace static `FULL_PITCH_SLOTS[i]` with a dynamic tactical slot mapper based on `player.position`:
    - `CAM`: `[50, 92]` (centrally in attacking midfield).
    - `CDM`: `[50, 62]` (deep central midfield).
    - `CM`: `[28, 75]` and `[72, 75]`.
    - `LB`: `[13, 44]`, `RB`: `[87, 44]`, `CB`: `[37.5, 44]`, `[62.5, 44]`.
    - `LW`: `[18, 106]`, `RW`: `[82, 106]`.
    - `ST` / `CF`: `[50, 114]` (centrally leading attack).
- **Line 365**: Compute formation label dynamically (e.g., `4-2-3-1`, `4-3-3`, `4-4-2`) instead of hardcoding `"4-3-3"`.

#### 6. `frontend/src/components/PitchCanvas.tsx`
- **Line 280–285**: Ensure `CAM` category ring uses midfield color (`#9AA79D` or sage) instead of attacker bone (`#EAE4D6`).

---

## 5. Affected Tests & Recommended New Test Suites

### 5.1 Existing Tests Needing Updates
1. `backend_go/pkg/tournament/tournament_test.go`:
   - `TestTournamentManager_Initialization`: Update `expectedFixtures := 44 * 6` (264 fixtures).
2. `backend_go/pkg/tournament/packet9_cycle_test.go`:
   - `TestPacket9TripleRoundRobinShape`: Rename/update to `TestPacket9QuadrupleRoundRobinShape`:
     - Assert `len(tm.Fixtures) == 44 * 6` (264 fixtures).
     - Assert `len(perWeek) == 44`, and each week has exactly 6 fixtures.
     - Assert each pair plays 4 times ($n = 4$).
     - Assert each club has exactly 22 home games and 22 away games.
   - `TestPacket9AdoptLongSeasonKeepsOpeningResultsWhenAddingHomeStretch`:
     - Test stretching a 22-week or 33-week campaign to 44 weeks.
3. `backend_go/pkg/tournament/chunk3_backfill_test.go`:
   - `TestChunk3BackfillWeekTriggers`: Update coronation test to MW 44.

### 5.2 Recommended New Test Suites
1. `backend_go/pkg/tournament/calendar_44_test.go`:
   - `TestCalendar44_FullSeasonFixtureCountAndSlates`: Simulates a full 44-matchweek slate, asserting all 44 slates have 6 fixtures and all 12 clubs appear once per week.
   - `TestCalendar44_VenueSymmetry`: Asserts all 12 clubs have exactly 22 home and 22 away matches.
   - `TestCalendar44_SeasonalMatchVolume`: Runs all league and cup fixtures, asserting final match counts for finalists reach 57–58 games.
2. `backend_go/pkg/matchengine/tactical_coordinates_test.go`:
   - `TestTacticalCoordinates_NaturalPositionMapping`:
     - Test CAM is centrally positioned at $Y = 0.50$, $X \in [0.49, 0.52]$.
     - Test CDM is deep central at $Y = 0.50$, $X \in [0.29, 0.33]$.
     - Test CMs are in left/right channels ($Y \in [0.30, 0.36]$ and $Y \in [0.64, 0.70]$).
     - Test ST is central at $Y = 0.50$, $X \in [0.60, 0.65]$.
     - Test flank players are wide ($Y < 0.22$ or $Y > 0.78$).
   - `TestTacticalCoordinates_NoLeftWingClustering`:
     - Run engine with Real Madrid (Cantalejo, CAM), Arsenal (Bantol, CAM), and Tottenham (Guinita, CF/CAM).
     - Assert CAM distance to left-wing $(0.58, 0.20)$ is $> 0.25$, and CAM $Y$-coordinate is within $[0.48, 0.52]$.
   - `TestTacticalCoordinates_AwaySymmetry`:
     - Assert Away team coordinates are mirrored ($X_{\text{away}} = 1 - X_{\text{home}}$, $Y_{\text{away}} = 1 - Y_{\text{home}}$).

---

## 6. Verification Plan & Commands
1. **Backend Tests**:
   - `cd backend_go && go test -v ./pkg/tournament/...`
   - `cd backend_go && go test -v ./pkg/matchengine/...`
   - `cd backend_go && go test -v ./pkg/models/...`
   - `cd backend_go && go test -v ./...`
   - Must pass with 0 warnings, 0 errors, and 0 panics.
2. **Frontend Compilation & Build**:
   - `cd frontend && npm run build` (or `bun run build`)
   - Must compile with 0 TypeScript errors and produce dist bundle.
