# Worker M2 Implementation Report: 44-Matchweek Calendar & Seasonal Game Volume

**Agent**: Worker M2 (R2 44-Matchweek Calendar & Seasonal Game Volume)  
**Parent**: `3e97d900-03a3-4902-b7ad-5a877f27dac3`  
**Date**: 2026-09-10  
**Status**: Complete  

---

## 1. Executive Summary

Worker M2 has implemented the full 44-matchweek quadruple round-robin calendar overhaul and seasonal match volume alignment across the Go backend and React frontend.

Key deliverables achieved:
1. **Quadruple Round-Robin Super League Engine (`pkg/tournament`)**:
   - Expanded calendar from 33 rounds (198 fixtures) to 44 rounds (264 fixtures).
   - Generates 4 full cycles of 11 Berger polygon rounds across the 12 elite clubs ($4 \times 11 = 44$ matchweeks, $44 \times 6 = 264$ league fixtures).
   - Perfect venue symmetry: Cycle 1 ($H \to A$), Cycle 2 ($A \to H$, reversed), Cycle 3 ($H \to A$), Cycle 4 ($A \to H$, reversed).
   - Every club plays exactly 44 league matches (22 home, 22 away). Every pair of clubs meets 4 times (2 home, 2 away).
   - Every single league matchweek slate contains all 6 fixtures with all 12 clubs participating (zero dropped or truncated matches).
2. **Season Month Bands & Cup Alignment**:
   - Expanded `MonthBands` to 10 months covering weeks 1 through 44 (added April: weeks 34–38, May: weeks 39–44).
   - Realiged Champions Cup (UCL) knockouts to the season home stretch: QF in MW 35–36, SF in MW 39–40, Grand Final in MW 44. Preserved group stage at weeks 3, 8, 13, 19, 24.
   - Four league phases: Opening series (MW 1–11), Return series (MW 12–22), Third series (MW 23–33), Final stretch (MW 34–44).
3. **Seasonal Game Volume (~55–60 Games)**:
   - Elite clubs reaching both Champions Cup and Super Cup finals play: 44 league + 10 UCL (5 group + 2 QF + 2 SF + 1 Final) + 3–4 Super Cup matches = **57–58 total matches**, strictly satisfying the ~55–60 game requirement.
4. **Legacy Save Adoption**:
   - Updated `adoptLongSeasonUnlocked` in `season.go` to target 264 fixtures across 44 matchweeks.
   - Automatically upgrades legacy 22-week and 33-week saves to 44 matchweeks while preserving all played match records (matching both direct and reversed match pairing IDs).
5. **Frontend Alignment**:
   - `StandingsTab.tsx`: Updated table narrative pace calculation from hardcoded `* 33` to `* (maxMatchweeks || 44)`, and updated header subtitle to "Quadruple round-robin, 44 games each."
   - `UclTournamentTab.tsx`: Updated subtitle to "Two groups of six across the 44-week calendar."
   - `App.tsx`: Updated footer banner to "12 clubs · 44-week Super League".
   - `api.ts`: Updated fallback defaults for `max_matchweeks` from 33 to 44 across all service endpoints.
6. **Verification & Quality**:
   - Added comprehensive test suite `pkg/tournament/calendar_44_test.go` verifying 44 rounds, 6 fixtures/round, 22H/22A per club, month bands, 57–58 deep-run game volume, and full season simulation.
   - Updated `packet9_cycle_test.go` and `tournament_test.go`.
   - `cd backend_go && go test -v -count=1 ./pkg/tournament/...` passes 100% (44/44 tests).
   - `cd frontend && npm run build` compiles cleanly with 0 TypeScript errors.

---

## 2. Code Changes by File

### 2.1 Backend (`backend_go/pkg/tournament/`)

1. **`constants.go`**:
   - `LeagueRounds = 44` (from 33).
   - `UCLFinalWeek = 44` (from 33).
   - Added April `{34, 38, "April"}` and May `{39, 44, "May"}` to `MonthBands`.
   - Updated `LeaguePhase(mw)`: MW 1–11 ("Opening series"), MW 12–22 ("Return series"), MW 23–33 ("Third series"), MW 34–44 ("Final stretch").
   - Updated `UCLQFWeeks = []int{35, 36}` and `UCLSFWeeks = []int{39, 40}`.
   - Updated `WeekChapter` to map MW 39+ to home stretch.

2. **`fixtures.go`**:
   - Updated `GenerateLeagueFixtures` to loop through 4 cycles of 11 Berger rounds ($4 \times 11 = 44$).
   - `reverseVenues := (cycle%2 == 0)` ensures Cycles 1 & 3 are $H \to A$ and Cycles 2 & 4 are $A \to H$.
   - Removed obsolete `pickCycle3Venue` helper since the 4-cycle structure guarantees perfect 22 home and 22 away games.

3. **`season.go`**:
   - Updated `adoptLongSeasonUnlocked` to target `(len(tm.ClubsList) * LeagueRounds) / 2 = 264` fixtures.
   - When merging existing fixtures into generated fixtures, matches both `f.FixtureID` and `altID := fmt.Sprintf("MW%d-%s-%s", f.Matchweek, f.AwayID, f.HomeID)` to guarantee all completed fixtures are preserved.
   - Ensures `MatchweekWeather` is populated for all 44 matchweeks.

4. **`packet9_cycle_test.go`**:
   - Renamed `TestPacket9TripleRoundRobinShape` to `TestPacket9QuadrupleRoundRobinShape`, asserting 264 fixtures, 44 matchweeks with 6 fixtures each, 4 meetings per pair, and exactly 22 home games per club.
   - Added `TestPacket9CycleVenuesAndSymmetry` asserting Cycle 1 == Cycle 3, Cycle 2 == Cycle 4, and each directed pair occurs exactly twice.
   - Updated `TestPacket9AdoptLongSeasonKeepsOpeningResultsWhenAddingHomeStretch` to verify stretching to 44 matchweeks and 264 fixtures.

5. **`tournament_test.go`**:
   - Updated `expectedFixtures := 44 * 6` (264 fixtures) in `TestTournamentManager_Initialization`.

6. **`calendar_44_test.go` (New Test File)**:
   - `TestCalendar44_ScheduleStructure`: Verifies 264 total fixtures, 44 matchweeks with 6 matches each, 12 unique clubs playing each week, 22 home and 22 away games per club, and symmetric head-to-head hosting.
   - `TestCalendar44_MonthBandsAndChapters`: Verifies 10 month bands (August–May), UCL knockout weeks (35, 36, 39, 40, 44), and chapter descriptions.
   - `TestCalendar44_GetCalendarAPI`: Verifies API calendar strip contains 44 weeks with 6 league matches each.
   - `TestCalendar44_SeasonalMatchVolume_DeepRun`: Verifies mathematical volume for deep-run finalist clubs lands on 57–58 games.
   - `TestCalendar44_AdoptLongSeasonFromLegacy33Weeks`: Verifies a 33-week save with 198 fixtures seamlessly expands to 44 weeks and 264 fixtures with completed results preserved.
   - `TestCalendar44_FullSeasonSimulationAndTotalVolume`: Simulates all 44 weeks of a full season, confirming all 264 fixtures finish, table records 44 games for each club, season rolls over to `transfer_window`, and deep-run clubs record 57–58 matches in total volume.

---

### 2.2 Frontend (`frontend/src/`)

1. **`components/StandingsTab.tsx`**:
   - `tableNarrative(clubs: Club[], maxMatchweeks: number = 44)`: updated title pace calculation to `((first?.pts ?? 0) / Math.max(1, played)) * (maxMatchweeks || 44)`.
   - Updated `PanelHeader` subtitle from "Triple round-robin, 33 games each" to "Quadruple round-robin, 44 games each. Champions Cup and Super Cup sit on the same slate. Decided games lock."
   - Passed `league.max_matchweeks` into `tableNarrative(league.clubs, league.max_matchweeks)`.

2. **`components/UclTournamentTab.tsx`**:
   - Updated `PanelHeader` subtitle to "Two groups of six across the 44-week calendar. Top four reach two-legged quarter-finals, then semis and a final. Ties sit on the same fixture list."

3. **`App.tsx`**:
   - Updated footer text to `<p>12 clubs · 44-week Super League</p>`.

4. **`services/api.ts`**:
   - Updated default fallback values from `max_matchweeks: 33` to `max_matchweeks: 44` in:
     - `fetchSuperLeague` (line 189)
     - `fetchFixtures` (line 225)
     - `resetSeason` (line 353)
     - `fetchCareerHistory` (line 559)
     - `fetchCalendar` (line 658)

---

## 3. Verification Results

1. **Backend Tournament Test Suite**:
   ```
   cd backend_go
   go test -v -count=1 ./pkg/tournament/...
   ```
   Result: **PASS** (44/44 tests passed in 2.0s, 0 compiler warnings, 0 panics).

2. **Backend Vet**:
   ```
   cd backend_go
   go vet ./pkg/tournament/...
   ```
   Result: **PASS** (0 warnings/errors).

3. **Frontend Production Build**:
   ```
   cd frontend
   npm run build
   ```
   Result: **PASS** (0 TypeScript errors, Vite bundle created in 5.55s).

4. **Seasonal Game Volume Invariant**:
   - Super League: 44 matches
   - Champions Cup: 10 matches (5 group, 2 QF, 2 SF, 1 Final)
   - Super Cup: 3 matches (seeds 1–4 bye) or 4 matches (seeds 5–12 play-in)
   - Total matches for finalists: 57 or 58 matches, strictly satisfying the ~55–60 game requirement.
