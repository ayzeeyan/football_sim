# BRIEFING — 2026-09-10T00:43:40Z

## Mission
Implement R2 44-Matchweek Calendar & Seasonal Game Volume: Berger 4-cycle league schedule (44 rounds, 264 fixtures, perfectly balanced 22H/22A), 10 month bands, UCL calendar alignment (Final MW 44), career save adoption, frontend standings/UCL/banner 44-game updates, and unit tests.

## 🔒 My Identity
- Archetype: worker
- Roles: implementer, qa, specialist
- Working directory: c:\Users\Izyan\Downloads\General\football_sim\.agents\worker_m2
- Original parent: 7396780a-6b1b-46cf-b464-6d10f96706f4
- Milestone: M2 (Growth Engine & Biometrics)
- Milestone R2: Worker M2 (R2 44-Matchweek Calendar & Seasonal Game Volume)
- Working directory R2: c:\Users\Izyan\General\football_sim\.agents\worker_m2
- Parent R2: 3e97d900-03a3-4902-b7ad-5a877f27dac3

## 🔒 Key Constraints
- Exclusive file ownership: backend_go/pkg/growth/
- Zero internal package dependencies for pkg/growth
- Genuine implementation with thread safety (sync.RWMutex)
- 100% test pass rate with zero compiler warnings and zero runtime panics
- Never allow youth growth or wonderkid potential to exceed bounds (never 99 for wonderkids, 93-96 canonical)
- Aging decline floor strictly at 35 for attributes, 55 for OVR
- R2 Ownership:
  - backend_go/pkg/tournament/constants.go
  - backend_go/pkg/tournament/fixtures.go
  - backend_go/pkg/tournament/season.go (calendar/schedule/adoption portions)
  - backend_go/pkg/tournament/calendar.go
  - backend_go/pkg/tournament/ test files
  - frontend/src/components/StandingsTab.tsx
  - frontend/src/components/UclTournamentTab.tsx
  - frontend/src/App.tsx
  - frontend/src/services/api.ts
- Do not edit files in backend_go/pkg/growth/ or backend_go/pkg/matchengine/
- Genuine implementations only: no hardcoding test outputs or dummy fixtures
- Schedule must have exactly 44 rounds, 6 fixtures per round, 264 total fixtures
- Exactly 22 home and 22 away games per club
- Every league matchweek slate contains all 6 fixtures (12 clubs playing every week)
- Deep-run clubs play ~55-60 games (44 league + 10 UCL + 3-4 Super Cup = 57-58 games)

## Current Parent
- Conversation ID: 3e97d900-03a3-4902-b7ad-5a877f27dac3
- Updated: 2026-09-10T00:43:40Z

## Task Summary
- **What to build**: 44-matchweek calendar across backend and frontend, 4-cycle Berger schedule, 10 month bands, UCL knockout realignment, career adoption, frontend displays, verification tests.
- **Success criteria**: All go tests pass (`cd backend_go && go test -v ./pkg/tournament/...` and `./...`), frontend compiles (`npm run build`), deep-run teams play 57-58 games.
- **Interface contracts**: PROJECT.md & AGENTS.md
- **Code layout**: PROJECT.md

## Key Decisions Made
- Generating 4 rounds of Berger round-robin for 12 clubs: 4 cycles of 11 rounds = 44 matchweeks. Cycles alternate H/A (Cycle 1 H->A, Cycle 2 A->H, Cycle 3 H->A, Cycle 4 A->H).
- Expanded MonthBands to 10 bands (Aug-May: Apr 34-38, May 39-44).
- Aligned UCL knockouts: QF 35-36, SF 39-40, Final 44.
- Updated AdoptLongSeason to preserve legacy finished matches (matching both direct and reversed pairing IDs).
- Updated StandingsTab, UclTournamentTab, App footer, and api fallbacks to 44 matchweeks.

## Artifact Index
- backend_go/pkg/tournament/constants.go
- backend_go/pkg/tournament/fixtures.go
- backend_go/pkg/tournament/season.go
- backend_go/pkg/tournament/packet9_cycle_test.go
- backend_go/pkg/tournament/tournament_test.go
- backend_go/pkg/tournament/calendar_44_test.go
- frontend/src/components/StandingsTab.tsx
- frontend/src/components/UclTournamentTab.tsx
- frontend/src/App.tsx
- frontend/src/services/api.ts

## Change Tracker
- **Files modified**:
  - `backend_go/pkg/tournament/constants.go`: Set LeagueRounds = 44, UCLFinalWeek = 44, 10 MonthBands, UCL knockouts (35-36, 39-40, 44), 4 LeaguePhase series.
  - `backend_go/pkg/tournament/fixtures.go`: Implemented 4-cycle Berger schedule generating 264 fixtures, perfectly balanced 22H/22A, 6 fixtures per week.
  - `backend_go/pkg/tournament/season.go`: Updated adoptLongSeasonUnlocked to target 264 fixtures and 44 rounds while preserving finished results.
  - `backend_go/pkg/tournament/packet9_cycle_test.go`: Updated tests to verify 44 rounds, 264 fixtures, 22H/22A, and 4-cycle symmetry.
  - `backend_go/pkg/tournament/tournament_test.go`: Updated expectedFixtures from 198 to 264.
  - `backend_go/pkg/tournament/calendar_44_test.go`: Added test suite for schedule structure, venue symmetry, month bands, calendar API, 55-60 match volume, and full season simulation.
  - `frontend/src/components/StandingsTab.tsx`: Dynamic pace calculation `max_matchweeks || 44`, subtitle "44 games each".
  - `frontend/src/components/UclTournamentTab.tsx`: Updated subtitle to "44-week calendar".
  - `frontend/src/App.tsx`: Updated footer to "44-week Super League".
  - `frontend/src/services/api.ts`: Updated default fallbacks to `max_matchweeks: 44`.
- **Build status**: PASS (`go test -v -count=1 ./pkg/tournament/...` 44/44 pass; `npm run build` clean 0 errors).
- **Pending issues**: None.

## Quality Status
- **Build/test result**: PASS (44/44 tests in pkg/tournament pass in 2.0s).
- **Lint status**: Clean (`go vet ./pkg/tournament/...` 0 issues).
- **Tests added/modified**: `calendar_44_test.go` (5 new tests covering schedule, venue symmetry, month bands, 55-60 game volume, and full season simulation), updated `packet9_cycle_test.go` and `tournament_test.go`.

## Loaded Skills
- None specified

