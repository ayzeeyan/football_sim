# Project: Football Sim Game Balance & Core Systems Overhaul

## Architecture Overview
A high-performance career football simulator featuring Go 1.22+ backend (`backend_go`) and React 18 / TypeScript / Vite frontend (`frontend`).
The core systems overhaul encompasses five modular subsystems:
1. **Growth Engine (`backend_go/pkg/growth`)**:
   - Rebalanced match XP formulas, level-up target scaling (from `1.18` to `1.04`), and calibrated appearance progression.
   - Guaranteed multi-year trajectory: +2 to +4 OVR/season under regular playing time, ~81 OVR at age 16, ~86 OVR at age 18, approaching 93–96 canonical ceiling in early 20s.
   - Strict potential cap invariance (`[93, 96]`).
2. **Tournament & Calendar Engine (`backend_go/pkg/tournament`)**:
   - Quadruple round-robin Super League schedule: 4 cycles of 11 Berger rounds = exactly 44 matchweeks.
   - 264 total league fixtures, guaranteed 6 fixtures per weekly slate (all 12 clubs play each week, zero dropped matches).
   - Season game volume: 44 league + 10 Champions Cup + 3–4 Super Cup = ~57–58 total matches for elite clubs.
   - MonthBands expanded to 10 months (August–May: Weeks 34–38 April, Weeks 39–44 May).
   - `adoptLongSeasonUnlocked` in `season.go` automatically expands legacy saves to 44 weeks.
3. **Tactical Coordinates & Pitch Rendering (`backend_go/pkg/matchengine`, `pkg/models`, `frontend/src`)**:
   - Position-aware pitch coordinate assignment based on natural player positions:
     - `CAM`: centrally in attacking midfield `{0.51, 0.50}` between CMs and ST.
     - `CDM`: deep central midfield `{0.31, 0.50}`.
     - `CM`: central midfield channels `{0.39, 0.33}` and `{0.39, 0.67}`.
     - Flank players (`LB`, `LWB`, `RB`, `RWB`, `LM`, `RM`, `LW`, `RW`): along wings ($Y \approx 0.16$ and $Y \approx 0.84$).
     - Strikers / Center Forwards (`ST`, `CF`): centrally leading attack `{0.63, 0.50}`.
   - Elimination of left-wing clustering on pitch canvas, live match radar, and lineup displays.
4. **Transfer Engine & Warchests (`backend_go/pkg/transfers`, `pkg/managers`, `pkg/persistence`, `frontend/src`)**:
   - 12-week off-season transfer window advancing week-by-week (`CurrentWeek` 1..12).
   - Club warchests initialized between €50M and €250M based on stature, with strict solvency checks preventing negative balances.
   - Strict single-transfer lock per window (`TransferredThisWindow`).
   - Canonical wonderkids (`WK_` IDs) restricted to the 12 Super League clubs, with automatic loan return to original parent club at season reset.
5. **Verification & Stability**:
   - 100% test pass across all backend packages (`go test -count=1 ./...`).
   - Clean production build of frontend (`bun run build` / `npm run build` with zero TypeScript errors).

---

## Feature Inventory
Every feature from the survey phase mapped to its assigned milestone:

| # | Feature | Description | Milestone | Source |
|---|---------|-------------|-----------|--------|
| 1 | Growth XP Scaling Rebalance | Match XP base `2.2`, goals `5.0`, assists `3.0`, level target `1.04`, target `160.0` | M1 | Survey Explorer 1 |
| 2 | Calibrated Seasonal Appearance Bump | Wonderkid appearance bump `wkBump` (+1 to +2 OVR for >=25 apps), legacy fallback for `attrs == nil` | M1 | Survey Explorer 1 |
| 3 | Multi-Year Progression Invariant | +2 to +4 OVR/season, ~81 at 16, ~86 at 18, [93, 96] ceiling | M1 | Survey Explorer 1 |
| 4 | 44-Matchweek Super League Schedule | 4 cycles of 11 Berger rounds, 264 total fixtures, 22 home / 22 away | M2 | Survey Explorer 2 |
| 5 | 6 Fixtures Per Weekly Slate | Guarantee all 12 clubs play each week with 0 dropped fixtures | M2 | Survey Explorer 2 |
| 6 | Seasonal Volume & Calendar Realignment | ~55-60 games/season for elite clubs, 10 MonthBands, UCL knockout alignment | M2 | Survey Explorer 2 |
| 7 | Legacy Save Expansion | `adoptLongSeasonUnlocked` expanding saves < 44 weeks / < 264 fixtures | M2 | Survey Explorer 2 |
| 8 | Frontend League Pace & Strips | Update standings pace calculation from `* 33` to `* 44` / `max_matchweeks` | M2 | Survey Explorer 2 |
| 9 | Position-Aware Pitch Coordinates | Map natural positions (CAM central, CDM deep, CM channels, wings, ST/CF lead) | M3 | Survey Explorer 2 |
| 10 | Dynamic Match Coordinates | `LiveMatchEngine` using player-assigned coordinates, preserving tactical shifts | M3 | Survey Explorer 2 |
| 11 | Pitch Radar & HUD Alignment | Render pitch canvas and radar without left-wing bias, fix `MatchDetailModal.tsx` | M3 | Survey Explorer 2 |
| 12 | 12-Week Off-Season Window | Restructure transfer window into 12 weekly stages advancing week-by-week | M4 | Survey Explorer 3 |
| 13 | Club Warchests (€50M–€250M) | Persistent club transfer budgets by club stature with solvency checks | M4 | Survey Explorer 3 |
| 14 | Single-Transfer Lock | Player can only transfer once per transfer window (`TransferredThisWindow`) | M4 | Survey Explorer 3 |
| 15 | Wonderkid Loan Return Rules | Wonderkids transfer only between 12 clubs; return to canonical club at season reset | M4 | Survey Explorer 3 |
| 16 | Frontend Transfer Weekly Display | Update Transfers tab labels, stages, and badges for 12 weekly phases | M4 | Survey Explorer 3 |
| 17 | Full System Integration & E2E Verification | `go test ./...` passes 100%, `bun run build` succeeds with 0 errors | M5 | Survey Explorer 3 |

---

## Milestones

| # | Name | Scope | Dependencies | Status |
|---|------|-------|-------------|--------|
| M1 | Wonderkid Growth Curve Rebalance | `pkg/growth/progression.go`, `engine.go`, `aging.go`, growth tests | None | DONE |
| M2 | 44-Matchweek Calendar & Quadruple Round-Robin | `pkg/tournament/season.go`, `schedule.go`, `calendar.go`, `frontend/.../LeagueTab.tsx` | None | PLANNED |
| M3 | Position-Driven Lineup & Pitch Coordinates | `pkg/matchengine/live.go`, `tactics.go`, `pkg/models/constants.go`, `frontend/.../MatchDetailModal.tsx` | None | PLANNED |
| M4 | 12-Week Transfer Window, Warchests & Loan Rules | `pkg/transfers/transfers.go`, `pkg/managers/managers.go`, `pkg/tournament/season.go`, `pkg/persistence/persistence.go`, `pkg/server/server.go`, `frontend/.../TransfersTab.tsx` | None | PLANNED |
| M5 | Full System Integration & Verification | Backend full test suite (`go test ./...`), frontend build (`bun run build`), regression audit | M1, M2, M3, M4 | PLANNED |

---

## Interface Contracts

### 1. Growth Engine Interface (`pkg/growth`)
- `ApplyMatchXP(bio *models.BiometricProfile, matchRating float64, minutes int, goals int, assists int, cleanSheet bool, posCat string, mentorOVR int) (bool, []string)`
- `ApplySeasonalGrowth(playerID string, age int, appearances int, potential int, cat string, currentOVR ...int) int`
- Invariant: U-14 starting at 75 OVR gains +2 to +4 OVR per full 44-week season. Never exceeds +5 in single season. Potential cap strictly `[93, 96]`.

### 2. Tournament & Calendar Interface (`pkg/tournament`)
- `GenerateSuperLeagueSchedule(clubs []*models.Club) [][]*models.Fixture`
  - Returns 44 rounds, 6 fixtures per round = 264 fixtures.
  - Symmetrical venue distribution: 22 home, 22 away per club.
- `adoptLongSeasonUnlocked(tm *TournamentManager)`: Expands saved calendars to 44 weeks / 264 fixtures.
- `GetMonthBands()`: Returns 10 month bands (August to May, covering weeks 1..44).

### 3. Tactical Coordinates Interface (`pkg/matchengine`, `pkg/models`)
- `AssignPitchCoordinates(startingEleven []*models.Player, formation string) [][2]float64`
  - Returns normalized `(X, Y)` coordinates in `[0.0, 1.0]`:
    - `CAM`: `{0.51, 0.50}`
    - `CDM`: `{0.31, 0.50}`
    - `CM`: `{0.39, 0.33}` and `{0.39, 0.67}`
    - `LB/LWB`: `{0.20, 0.16}`, `RB/RWB`: `{0.20, 0.84}`
    - `LM/LW`: `{0.50, 0.16}`, `RM/RW`: `{0.50, 0.84}`
    - `ST/CF`: `{0.63, 0.50}`
- `LiveMatchEngine.HomeBaseCoords` and `AwayBaseCoords`: Dynamic tactical coordinates assigned per match.

### 4. Transfer Engine Interface (`pkg/transfers`)
- `AdvanceOpenWindow()`:
  - Increments `CurrentWeek` (1..12).
  - Processes weekly market negotiation progress.
- `TransferredThisWindow`: `map[string]bool` tracking players moved in the active window.
- `ClubWarchests`: `map[string]int64` initialized in `[€50M, €250M]`, decremented on purchase, incremented on sale, strictly nonnegative.
- Canonical Wonderkid Homecoming in `season.go`:
  - Only players with `p.UniverseWonderkid == true` or `strings.HasPrefix(p.PlayerID, "WK_")` are returned to `OriginalClubID`. Permanent transfers for other players remain with buyer.

---

## Code Layout & File Ownership

| Subsystem | Primary Source Files | Primary Test Files |
|-----------|----------------------|--------------------|
| M1 Growth | `backend_go/pkg/growth/progression.go`, `aging.go`, `engine.go` | `backend_go/pkg/growth/*_test.go` |
| M2 Calendar | `backend_go/pkg/tournament/season.go`, `schedule.go`, `calendar.go`, `frontend/src/components/LeagueTab.tsx` | `backend_go/pkg/tournament/*_test.go` |
| M3 Tactics | `backend_go/pkg/matchengine/live.go`, `tactics.go`, `backend_go/pkg/models/constants.go`, `frontend/src/components/MatchDetailModal.tsx` | `backend_go/pkg/matchengine/*_test.go` |
| M4 Transfers | `backend_go/pkg/transfers/transfers.go`, `backend_go/pkg/managers/managers.go`, `backend_go/pkg/persistence/persistence.go`, `backend_go/pkg/server/server.go`, `frontend/src/components/TransfersTab.tsx`, `frontend/src/services/api.ts` | `backend_go/pkg/transfers/*_test.go`, `backend_go/pkg/server/transfer_api_test.go` |
| M5 Verification | Full project tree | `cd backend_go && go test ./...`, `cd frontend && bun run build` |
