# Handoff Report: R2 Calendar & R3 Tactics Survey

**Agent**: Survey Explorer 2 (teamwork_preview_explorer)  
**Parent Agent**: parent (`3e97d900-03a3-4902-b7ad-5a877f27dac3`)  
**Date**: 2026-09-10  
**Status**: Hard Handoff (Investigation Complete)  
**Working Directory**: `c:\Users\Izyan\General\football_sim\.agents\explorer_survey_2\`  
**Detailed Report**: `c:\Users\Izyan\General\football_sim\.agents\explorer_survey_2\report.md`  

---

## 1. Observation

Direct observations and evidence across the codebase:

1. **Hardcoded 33-Round League Calendar**:
   - `backend_go/pkg/tournament/constants.go:4`:
     ```go
     LeagueRounds = 33
     UCLFinalWeek = 33
     ```
   - `backend_go/pkg/tournament/fixtures.go:38,88-150`:
     `GenerateLeagueFixtures(clubs, rng)` uses Berger polygon algorithm with 11 rounds per cycle, generating Cycle 1 (MW 1–11), Cycle 2 (MW 12–22), and Cycle 3 (MW 23–33) using `pickCycle3Venue` to distribute 16 or 17 home games. Total fixtures = 198 (33 matchweeks $\times$ 6 games/week).
   - `backend_go/pkg/tournament/season.go:397-398`:
     ```go
     target := (len(tm.ClubsList) * LeagueRounds) / 2
     already := tm.MaxMatchweeks >= LeagueRounds && len(tm.Fixtures) >= target
     ```
   - `backend_go/pkg/tournament/constants.go:8-20`:
     `MonthBands` currently ends at matchweek 33:
     ```go
     {25, 28, "February"},
     {29, 33, "March"},
     ```
     Omits April and May entirely.

2. **Cups Scheduling and Game Counts**:
   - `backend_go/pkg/tournament/constants.go:101-104`:
     `UCLGroupWeeks = []int{3, 8, 13, 19, 24}` (5 group games).
     `UCLQFWeeks = []int{27, 28}` (2 QF legs).
     `UCLSFWeeks = []int{30, 31}` (2 SF legs).
     `UCLFinalWeek = 33` (1 final game). Total UCL games for finalist = 10.
     `SuperCupWeeks = map[string]int{"play_in": 5, "qf": 12, "sf": 20, "final": 26}`. Total Super Cup games for finalist = 3 to 4.
   - For an elite club in a 44-matchweek season: 44 league games + 10 UCL games + 3 to 4 Super Cup games = **57 to 58 total games**, matching the ~55–60 target.

3. **Left-Wing Clustering Root Cause**:
   - `backend_go/pkg/models/constants.go:44-58`:
     ```go
     // NOTE: CAM is explicitly categorized as FWD per the match engine design.
     func GetPositionCategory(pos string) string {
     ...
     case "CDM", "CM":
         return "MID"
     default:
         return "FWD"
     }
     ```
   - `backend_go/pkg/models/club.go:210-265` (`GetStartingEleven`):
     Partitions squad into `gks`, `defs`, `mids`, `fwds`.
     Because CAM is in `fwds`, `sortPlayersForXI(fwds)` places high-OVR/wonderkid CAMs at the top of `fwds`.
     `startingXI` appends: GK (idx 0), 4 DEFs (idx 1–4), 3 MIDs (idx 5–7), 3 FWDs (idx 8–10).
     Thus, CAM is placed at **Index 8**.
   - `backend_go/pkg/matchengine/live.go:154-180`:
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
         {0.58, 0.20}, // 8: LW  <-- Index 8 is hardcoded to Left Wing!
         {0.62, 0.50}, // 9: ST
         {0.58, 0.80}, // 10: RW
     }
     ```
   - `backend_go/pkg/matchengine/live.go:265-290` (`radarPlayers`):
     Directly maps player `i` to `coords[i]`.
     Result: **Every CAM starting at index 8 receives coordinates `{X: 0.58, Y: 0.20}` (Left Wing)**.
   - `backend_go/pkg/matchengine/live.go:354, 371`:
     In-match wobble and stance updates read `base := baseHomeCoords[i]` based on slice index, keeping CAM on the left flank throughout the match.
   - `frontend/src/components/MatchDetailModal.tsx:13-18`:
     `FULL_PITCH_SLOTS[8]` is hardcoded to `[22, 106]` (left wing slot), replicating the left-wing bias in post-match lineups.

4. **Frontend Hardcoded 33-Matchweek References**:
   - `frontend/src/App.tsx:287`: `<p>12 clubs · 33-week Super League</p>`.
   - `frontend/src/components/StandingsTab.tsx:48`: `const pace = ((first?.pts ?? 0) / Math.max(1, played)) * 33;`.
   - `frontend/src/components/StandingsTab.tsx:238`: `subtitle="Triple round-robin, 33 games each..."`.
   - `frontend/src/components/UclTournamentTab.tsx:126`: `subtitle="Two groups of six across the 33-week calendar..."`.
   - `frontend/src/services/api.ts:189, 225, 353, 559, 658`: Fallback `max_matchweeks: 33`.

5. **Test Command Baselines**:
   - `cd backend_go && go test ./...` -> Exited code 0 (all packages pass).
   - `cd frontend && npm run build` -> Exited code 0 (Vite build successful, 0 errors).

---

## 2. Logic Chain

1. **Calendar Expansion (Observation 1 $\to$ Conclusion)**:
   - Observation 1 demonstrates `LeagueRounds = 33` with 3 cycles of 11 rounds.
   - Expanding to 4 cycles of 11 rounds yields $4 \times 11 = 44$ matchweeks.
   - With 12 clubs, each round has $12 / 2 = 6$ matches; $44 \times 6 = 264$ league fixtures.
   - Constructing Cycle 1 as $H \to A$, Cycle 2 as $A \to H$, Cycle 3 as $H \to A$, and Cycle 4 as $A \to H$ provides a mathematically balanced quadruple round-robin: every pair meets 4 times, exactly 2 at Club A's venue and 2 at Club B's venue (22 home, 22 away per club).
   - Expanding `MonthBands` to 10 months (Weeks 1–33 August–March, Weeks 34–38 April, Weeks 39–44 May) keeps existing labels backward-compatible while naturally supporting the full 44-week season.

2. **Seasonal Match Volume (Observation 2 $\to$ Conclusion)**:
   - 44 league fixtures + 10 UCL fixtures (5 group + 2 QF + 2 SF + 1 Final) + 3–4 Super Cup fixtures (Play-in + QF + SF + Final) = 57–58 matches.
   - This satisfies the requirement that deep-run elite clubs play ~55–60 games per season.
   - UCL knockouts naturally place in the home stretch: QF in MW 35–36, SF in MW 39–40, Grand Final in MW 44.

3. **Tactical Positioning & Left-Wing Bias Elimination (Observation 3 $\to$ Conclusion)**:
   - Observation 3 proves that CAM was categorized as FWD, placed at index 8 of the starting XI, and mapped to `baseHomeCoords[8] = {0.58, 0.20}` (Left Wing).
   - Assigning coordinates based on the player's natural position (`p.Position`) rather than array index `i` eliminates this bug:
     - `CAM`: centrally at $\{X: 0.51, Y: 0.50\}$.
     - `CDM`: deep central at $\{X: 0.31, Y: 0.50\}$.
     - `CM`: channels at $\{X: 0.39, Y: 0.33\}$ and $\{X: 0.39, Y: 0.67\}$.
     - Flank players (`LB`, `LWB`, `RB`, `RWB`, `LM`, `RM`, `LW`, `RW`): along the wings ($Y \approx 0.16$ and $Y \approx 0.84$).
     - `ST` / `CF`: centrally leading the attack at $\{X: 0.63, Y: 0.50\}$.
   - Away coordinates mirrored via $(1 - X, 1 - Y)$ preserve perfect positional symmetry without bias.
   - Storing `HomeBaseCoords` and `AwayBaseCoords` on `LiveMatchEngine` allows dynamic per-player tactical positioning while preserving stance shifts and wobble dynamics.

4. **Frontend & UI Alignment (Observation 4 $\to$ Conclusion)**:
   - Standings tab pace formula must use dynamic `max_matchweeks` ($44$).
   - SVG pitch slots in `MatchDetailModal.tsx` must map players based on their natural positions rather than static index slots.

---

## 3. Caveats

- **Off-season Transfer Window (R4)**: Investigated by Survey Explorer 3. Our scope covers the calendar rollover trigger into the transfer window (`tm.CurrentMatchweek > 44`), which operates seamlessly via `maybeRolloverUnlocked`.
- **Wonderkid XP Progression Scaling (R1)**: Investigated by Survey Explorer 1.
- **Middle School Exam Weeks**: Wonderkids have exam unavailability during matchweeks 12, 13, 24, 25, 32, and 33. This logic in `models.IsExamWeek` and `school.go` remains intact and valid in the 44-week schedule.

---

## 4. Conclusion

1. **Calendar (R2)**:
   - Set `LeagueRounds = 44` and `UCLFinalWeek = 44`.
   - Update `GenerateLeagueFixtures` to 4 cycles of 11 rounds (264 fixtures total, 6 fixtures per week, 22 home/22 away per club).
   - Expand `MonthBands` to 44 matchweeks across August–May.
   - Alight UCL QF (MW 35–36), SF (MW 39–40), and Final (MW 44).
   - Update `adoptLongSeasonUnlocked` to stretch saves to 44 matchweeks and 264 fixtures.
   - Update frontend pace calculation, subtitles, and API fallbacks to 44 matchweeks.

2. **Tactics (R3)**:
   - Implement `AssignPitchCoordinates(xi, isAway)` in `backend_go/pkg/models` or `matchengine`.
   - Store `HomeBaseCoords` and `AwayBaseCoords` in `LiveMatchEngine` and update `radarPlayers` and `Update()`.
   - Position CAM centrally at $\{0.51, 0.50\}$, CDM at $\{0.31, 0.50\}$, CMs at $\{0.39, 0.33/0.67\}$, wingers on flanks, and ST/CF centrally at $\{0.63, 0.50\}$.
   - Update `MatchDetailModal.tsx` SVG pitch renderer to position players by natural position rather than static index slots.

---

## 5. Verification Method

1. **Verify Backend Suite**:
   ```pwsh
   cd backend_go
   go test -v ./pkg/tournament/...
   go test -v ./pkg/matchengine/...
   go test -v ./pkg/models/...
   go test -v ./...
   ```
   *Expected*: All tests pass with 0 warnings, 0 runtime errors, and 0 panics.

2. **Verify Calendar Properties**:
   - `len(tm.Fixtures) == 264` (44 matchweeks $\times$ 6 fixtures).
   - Every matchweek $1 \le \text{mw} \le 44$ contains exactly 6 fixtures with all 12 clubs playing.
   - Each directed pairing $(C_A, C_B)$ occurs exactly 2 times (2 home, 2 away per club).
   - Deep-run finalist club finishes season with 57–58 games in records.

3. **Verify Pitch Coordinates**:
   - Check CAM players (Maverick Cantalejo at Real Madrid, Izyan Levin Bantol at Arsenal, Ashle Zylle Baguio at Dortmund) have $Y \in [0.48, 0.52]$ and $X \in [0.49, 0.53]$ on live radar.
   - Distance from CAM to left-wing $(0.58, 0.20)$ exceeds $0.25$.
   - Live pitch canvas renders CAM in the central attacking channel between midfielders and striker.

4. **Verify Frontend Build**:
   ```pwsh
   cd frontend
   npm run build
   ```
   *Expected*: Clean compilation with 0 TypeScript compilation errors.
