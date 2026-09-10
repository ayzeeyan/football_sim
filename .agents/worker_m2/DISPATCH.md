# Dispatch: Worker M2 (Growth Engine & Biometrics)

Authoritative Request: c:\Users\Izyan\Downloads\General\football_sim\.agents\ORIGINAL_REQUEST.md
Working Directory: c:\Users\Izyan\Downloads\General\football_sim\.agents\worker_m2
Explorer 2 Report: c:\Users\Izyan\Downloads\General\football_sim\.agents\explorer_survey_2\handoff.md
Project Spec: c:\Users\Izyan\Downloads\General\football_sim\.agents\orchestrator\PROJECT.md

## Exclusive File Ownership
You exclusively own and may create/edit files in:
`backend_go/pkg/growth/`
DO NOT touch any files outside this directory.

## Objective
Implement the biometric growth systems, aging decline curves, youth development, and comprehensive unit tests in `backend_go/pkg/growth/`.

Key components to implement:
1. `biometrics.go`: `BiometricProfile` (height, weight, potential 93-96, growth velocity, adult height age, level XP target 145 * 1.18^lvl, etc.), `TechnicalAttributes` (hexagonal matrix + 7 sub-attributes), `GrowthMilestone`, `TimelineEntry`.
2. `engine.go`: `GrowthEngine` with thread safety (`sync.RWMutex`), seeding, deterministic RNG option for tests.
3. `puberty.go`: `SimulatePubertyCycle()`, annual growth caps (age <=14: 2.6cm, <=16: 2.4cm, >16: 1.8cm), weight gain caps (5kg total), transition to adult frame.
4. `progression.go`: `CalculateOVR()` (FWD, MID, DEF weighting), `ApplyMatchXP()` (age mult, mentor mult, composure transfer), `ApplyMentorshipTick()` (personality synergy, composure boost), `RunTrainingCycle()`, `ReplenishTrainingEnergy()`.
5. `aging.go`:
   - `ApplyAgingDecline()`: For age 30+ veterans, physical attributes (pace, stamina, strength, physicality) drop by -1 (30-33), -2 (34-35), -3 (36+) down to floor 35. OVR seasonal drop down to 55.
   - `ApplySeasonalGrowth()`: For young players (<25), appearance-scaled growth (+1, +2, +3) bounded strictly by potential ceiling (93-96 for wonderkids, 75-95 for regens, never 99).
6. Unit tests: `growth_test.go` covering all curves, biometrics, thread safety (`-race`), aging decline floor, youth potential bounds.

Run tests: `cd backend_go && go test -v ./pkg/growth/...`
Ensure 100% tests pass with zero compiler warnings and zero runtime panics.

MANDATORY INTEGRITY WARNING:
DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. A auditor will independently verify your work. Integrity violations WILL be detected and your work WILL be rejected.

## 2026-09-07T06:50:54Z
You are Worker M2 for the Football Sim Go backend rewrite Chunk 1.
Working Directory: c:\Users\Izyan\Downloads\General\football_sim\.agents\worker_m2
Exclusive File Ownership: backend_go/pkg/growth/
Tasks: Implement biometric growth systems, aging decline curves, youth development, and unit tests.

## 2026-09-09T16:43:18Z
You are Worker M2 (R2 44-Matchweek Calendar & Seasonal Game Volume).
Your working directory is: c:\Users\Izyan\General\football_sim\.agents\worker_m2
Authoritative user request: c:\Users\Izyan\General\football_sim\.agents\ORIGINAL_REQUEST.md
Project architecture & contracts: c:\Users\Izyan\General\football_sim\.agents\orchestrator_v2\PROJECT.md
Explorer 2 handoff: c:\Users\Izyan\General\football_sim\.agents\explorer_survey_2\handoff.md and report.md

MANDATORY INTEGRITY WARNING:
DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. A teamwork_preview_auditor will independently verify your work. Integrity violations WILL be detected and your work WILL be rejected.

File Ownership:
You exclusively own and may edit:
- `backend_go/pkg/tournament/constants.go`
- `backend_go/pkg/tournament/fixtures.go`
- `backend_go/pkg/tournament/season.go` (calendar/schedule/adoption portions)
- `backend_go/pkg/tournament/calendar.go`
- `backend_go/pkg/tournament/` test files
- `frontend/src/components/StandingsTab.tsx`
- `frontend/src/components/UclTournamentTab.tsx`
- `frontend/src/App.tsx`
- `frontend/src/services/api.ts`
Do not edit files in `backend_go/pkg/growth/` or `backend_go/pkg/matchengine/`.

Implementation Tasks:
1. In `backend_go/pkg/tournament/constants.go`:
   - Set `LeagueRounds = 44`.
   - Update `MonthBands` to 10 months covering weeks 1 to 44 (add April: weeks 34–38, May: weeks 39–44).
   - Align UCL knockouts to the season home stretch: QF in MW 35–36, SF in MW 39–40, Final in MW 44 (preserving group stage in weeks 3, 8, 13, 19, 24).
2. In `backend_go/pkg/tournament/fixtures.go`:
   - Update `GenerateLeagueFixtures` to generate 4 cycles of 11 Berger rounds (4 * 11 = 44 rounds, 264 fixtures).
   - Cycle 1: H -> A, Cycle 2: A -> H, Cycle 3: H -> A, Cycle 4: A -> H.
   - Guarantee perfectly balanced home/away matches (each club gets exactly 22 home and 22 away games).
   - Guarantee every league matchweek slate contains all 6 fixtures (12 clubs playing every week) with zero dropped or truncated fixtures.
3. In `backend_go/pkg/tournament/season.go`:
   - Update `adoptLongSeasonUnlocked` and target fixture calculations so saved careers automatically adopt 44 rounds and 264 fixtures without dropping played matches.
4. Total season match volume:
   - Ensure elite clubs reaching UCL & Super Cup finals play ~57–58 total games (44 league + 10 UCL + 3–4 Super Cup), strictly satisfying the ~55–60 game requirement.
5. In `frontend/`:
   - `frontend/src/components/StandingsTab.tsx`: Update pace calculation from hardcoded `* 33` to `* (standings.max_matchweeks || 44)`. Update header subtitle from "33 games" to "44 games".
   - `frontend/src/components/UclTournamentTab.tsx`: Update subtitle from 33-week to 44-week calendar.
   - `frontend/src/App.tsx`: Update "33-week Super League" text to "44-week Super League".
   - `frontend/src/services/api.ts`: Update fallback `max_matchweeks: 44`.
6. Testing:
   - Add/update tests in `backend_go/pkg/tournament/` verifying:
     - Schedule has exactly 44 rounds, 6 fixtures per round, 264 total fixtures.
     - Each club plays exactly 44 games (22 home, 22 away).
     - Every week slate has 6 matches.
     - Deep-run clubs play ~55–60 games.
   - Run `go test -v -count=1 ./pkg/tournament/...` and `go test -count=1 ./...` in `backend_go`.
   - Run `npm run build` in `frontend` to verify 0 TypeScript errors.
7. Write your report to `c:\Users\Izyan\General\football_sim\.agents\worker_m2\report.md` and `handoff.md`, then send completion message to parent.
