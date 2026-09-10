# Handoff Report: R2 44-Matchweek Calendar & Seasonal Game Volume (Worker M2)

**Agent**: Worker M2 (teamwork implementer, qa, specialist)  
**Recipient**: Parent (`3e97d900-03a3-4902-b7ad-5a877f27dac3`)  
**Date**: 2026-09-10  
**Status**: Hard Handoff (Task Complete)  
**Working Directory**: `c:\Users\Izyan\General\football_sim\.agents\worker_m2`  
**Detailed Report**: `c:\Users\Izyan\General\football_sim\.agents\worker_m2\report.md`  

---

## 1. Observation

Direct observations and evidence from the implementation and verification:

1. **Backend Tournament Constants & Calendar**:
   - `backend_go/pkg/tournament/constants.go`:
     - Line 4: `LeagueRounds = 44` (expanded from 33).
     - Line 5: `UCLFinalWeek = 44` (realigned to finale).
     - Lines 8–20: `MonthBands` expanded from 8 to 10 months to cover matchweeks 1 through 44 (April: MW 34–38, May: MW 39–44).
     - Lines 22–31: `LeaguePhase(matchweek)` returns `"Opening series"` (<=11), `"Return series"` (<=22), `"Third series"` (<=33), and `"Final stretch"` (<=44).
     - Lines 100–104: `UCLGroupWeeks = []int{3, 8, 13, 19, 24}`, `UCLQFWeeks = []int{35, 36}`, `UCLSFWeeks = []int{39, 40}`, and `SuperCupWeeks = map[string]int{"play_in": 5, "qf": 12, "sf": 20, "final": 26}`.

2. **Fixture Generation & Venue Balance**:
   - `backend_go/pkg/tournament/fixtures.go`:
     - `GenerateLeagueFixtures(clubs, rng)` generates 4 cycles of 11 Berger rounds ($4 \times 11 = 44$ rounds, 264 total fixtures).
     - Cycle 1 (MW 1–11): $H \to A$, Cycle 2 (MW 12–22): $A \to H$, Cycle 3 (MW 23–33): $H \to A$, Cycle 4 (MW 34–44): $A \to H$.
     - Obsolete `pickCycle3Venue` helper removed; venue balance is guaranteed analytically by the 4-cycle alternation.
     - Every club plays exactly 44 league games (22 home, 22 away). Every head-to-head pair meets 4 times (2 home, 2 away).
     - Every matchweek slate contains exactly 6 matches involving all 12 clubs without exception.

3. **Season Career Adoption**:
   - `backend_go/pkg/tournament/season.go`:
     - `target := (len(tm.ClubsList) * LeagueRounds) / 2` calculates 264 target fixtures.
     - `adoptLongSeasonUnlocked` preserves existing fixtures matching direct `f.FixtureID` or reversed `altID := fmt.Sprintf("MW%d-%s-%s", f.Matchweek, f.AwayID, f.HomeID)`.
     - Populates `MatchweekWeather` for all 44 matchweeks.

4. **Frontend Integration**:
   - `frontend/src/components/StandingsTab.tsx`:
     - Line 36 & 48: `tableNarrative(clubs: Club[], maxMatchweeks: number = 44)` computes `pace = ((first?.pts ?? 0) / Math.max(1, played)) * (maxMatchweeks || 44)`.
     - Line 238: Header subtitle updated to `"Quadruple round-robin, 44 games each. Champions Cup and Super Cup sit on the same slate. Decided games lock."`
     - Line 259: Calls `tableNarrative(league.clubs, league.max_matchweeks)`.
   - `frontend/src/components/UclTournamentTab.tsx`:
     - Line 126: Subtitle updated to `"Two groups of six across the 44-week calendar..."`
   - `frontend/src/App.tsx`:
     - Line 287: Footer text updated to `"12 clubs · 44-week Super League"`.
   - `frontend/src/services/api.ts`:
     - Lines 189, 225, 353, 559, 658: Default fallback `max_matchweeks` updated to `44`.

5. **Test Results**:
   - `cd backend_go && go test -v -count=1 ./pkg/tournament/...`:
     - 44/44 tests passed in 2.0s with 0 warnings and 0 panics.
   - `cd backend_go && go vet ./pkg/tournament/...`:
     - 0 lint/vet warnings.
   - `cd frontend && npm run build`:
     - Clean compilation with 0 TypeScript errors (built in 5.55s).

---

## 2. Logic Chain

1. **Schedule Expansion ($4 \times 11 = 44$ Matchweeks)**:
   - Observation 1 & 2 establish that a 12-club league requires 11 rounds per single round-robin cycle ($N - 1 = 11$).
   - Generating 4 cycles yields $4 \times 11 = 44$ matchweeks.
   - With 12 clubs playing every round, each round contains $12 / 2 = 6$ fixtures.
   - $44 \times 6 = 264$ total league fixtures.
   - Because Cycle 1 and 3 use $(C_i \to C_j)$ and Cycle 2 and 4 use $(C_j \to C_i)$, for every pair $(C_i, C_j)$, $C_i$ hosts $C_j$ twice and $C_j$ hosts $C_i$ twice.
   - Total home games for every club = $11 + 11 = 22$; total away games = $11 + 11 = 22$.

2. **Seasonal Match Volume Target (~55–60 Games)**:
   - Observation 1 & 4 detail tournament structures:
     - Super League: 44 matches.
     - Champions Cup (UCL): 5 group matches + 2 quarter-final legs + 2 semi-final legs + 1 one-off final = 10 matches for finalist.
     - Super Cup: 3 matches (finalist with top-4 bye) or 4 matches (finalist entering play-in).
   - Total seasonal matches for an elite club reaching both finals:
     - With bye: $44 + 10 + 3 = 57$ matches.
     - Via play-in: $44 + 10 + 4 = 58$ matches.
   - Verified via `TestCalendar44_SeasonalMatchVolume_DeepRun` and `TestCalendar44_FullSeasonSimulationAndTotalVolume`. Both counts fall strictly within the ~55–60 match target.

3. **Career Continuity & Save Adoption**:
   - Observation 3 shows `adoptLongSeasonUnlocked` checks `len(tm.Fixtures) < target || tm.MaxMatchweeks < LeagueRounds`.
   - Any save from a 22-week (132 fixtures) or 33-week (198 fixtures) campaign adopts 264 fixtures and `MaxMatchweeks = 44`.
   - Completed match results from earlier matchweeks are retained by checking both direct fixture IDs and swapped IDs.

4. **Frontend Table Narrative & Pace Mathematics**:
   - Observation 4 replaces the hardcoded `* 33` multiplier in `StandingsTab.tsx` with dynamic `* (maxMatchweeks || 44)`.
   - Mid-season title pace projections now accurately scale across the full 44-game campaign (e.g. 30 points after 10 games correctly projects to 132 points rather than 99 points).

---

## 3. Caveats

- **Tactical Pitch Coordinates (M3)**: Handled by Worker M3 in `pkg/matchengine`, `pkg/models`, and frontend match detail modals.
- **Transfer Window Mechanics & Warchests (M4)**: Handled by Worker M4 in `pkg/transfers`, `pkg/managers`, and `pkg/persistence`.
- **Downstream Unit Test Expectations in Other Packages**:
  - `pkg/persistence/persistence_test.go:79` and `pkg/server/` tests had hardcoded assertions expecting `198` fixtures and `33` matchweeks. Those packages fall under M4 and M5 ownership and will be updated during those respective milestones.
- **Exam Weeks**: Wonderkid school exams remain at matchweeks 12, 13, 24, 25, 32, 33 as specified.

---

## 4. Conclusion

Worker M2 has delivered a complete, robust, and verified implementation of the 44-matchweek quadruple round-robin calendar and seasonal game volume overhaul.
- All 12 elite clubs have a guaranteed 44-match league slate (22 home, 22 away).
- Every single league matchweek has all 6 fixtures.
- Elite clubs play 57–58 games per season.
- Frontend standings, UCL headers, footers, and API fallbacks are fully updated.
- 44/44 unit tests in `pkg/tournament` pass and the frontend compiles cleanly.

---

## 5. Verification Method

1. **Run Backend Tournament Test Suite**:
   ```pwsh
   cd backend_go
   go test -v -count=1 ./pkg/tournament/...
   ```
   *Expected*: 44 passing tests, 0 failures, 0 panics.

2. **Run Backend Vet**:
   ```pwsh
   cd backend_go
   go vet ./pkg/tournament/...
   ```
   *Expected*: 0 issues detected.

3. **Verify Frontend Build**:
   ```pwsh
   cd frontend
   npm run build
   ```
   *Expected*: 0 TypeScript errors, Vite distribution build created cleanly.

4. **Inspect Files for Invalidation Conditions**:
   - `backend_go/pkg/tournament/constants.go`: Verify `LeagueRounds == 44`, `len(MonthBands) == 10`, `UCLFinalWeek == 44`.
   - `backend_go/pkg/tournament/fixtures.go`: Verify `GenerateLeagueFixtures` loops 4 cycles and generates 264 fixtures.
   - `backend_go/pkg/tournament/calendar_44_test.go`: Verify tests assert 44 rounds, 22H/22A, 6 matches per week, and 57–58 game volume.
