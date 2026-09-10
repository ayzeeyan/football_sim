# Dispatch: Worker M1 (Core Domain Models & Valuation Math)

Authoritative Request: c:\Users\Izyan\Downloads\General\football_sim\.agents\ORIGINAL_REQUEST.md
Working Directory: c:\Users\Izyan\Downloads\General\football_sim\.agents\worker_m1
Explorer 1 Report: c:\Users\Izyan\Downloads\General\football_sim\.agents\explorer_survey_1\handoff.md
Project Spec: c:\Users\Izyan\Downloads\General\football_sim\.agents\orchestrator\PROJECT.md

## Exclusive File Ownership
You exclusively own and may create/edit files in:
`backend_go/pkg/models/`
DO NOT touch any files outside this directory.

## Objective
Implement high-performance Go domain models, valuation curves, personality archetypes, and comprehensive unit tests in `backend_go/pkg/models/`.

Key components to implement:
1. `constants.go`: Position categories (CAM is FWD), Exam weeks {12, 13, 24, 25, 32, 33}, Position paths, Kit colors with HSV fallback.
2. `player.go`: `Player` struct with all fields from Explorer 1 report. Methods: `EffectiveOVR()`, `IsUnavailable()`, `DecideEducation()`, `AdvanceEducation()`, `FormattedValue()`, `FormattedWage()`.
3. `club.go`: `Club` struct with standings & form, `UpdateResult()`, `UpdateMorale()`, `GetStartingEleven()` (4-3-3: 1 GK, 4 DEF, 3 MID, 3 FWD with wonderkid prioritization and fatigue rotation), `GetBench()`.
4. `standings.go`: `CompetitionRecord`, `StandingsRow`, `StandingsTable` sorting (Points > GD > GF > Rating > Name).
5. `valuation.go`: `BaselineValue()`, `ClampValue()`, `ClampPlayer()`, `WageForOVR()`, `FormatCurrency()`, `FormatWage()`. Absolute bounds [€300k, €500M], dynamic corridor [0.35 * anchor, 3.0 * anchor].
6. `personality.go`: `PersonalityArchetype`, `PersonalityFor(name)`, `SchoolWantFor(name)`.
7. Unit tests: `player_test.go`, `club_test.go`, `standings_test.go`, `valuation_test.go`, `personality_test.go`.

Run tests: `cd backend_go && go test -v ./pkg/models/...`
Ensure 100% tests pass with zero compiler warnings and zero runtime panics.

MANDATORY INTEGRITY WARNING:

## 2026-09-07T06:50:54Z
You are Worker M1 for the Football Sim Go backend rewrite Chunk 1.

MANDATORY: Read the authoritative user request first:
c:\Users\Izyan\Downloads\General\football_sim\.agents\ORIGINAL_REQUEST.md

Working Directory: c:\Users\Izyan\Downloads\General\football_sim\.agents\worker_m1
Read instructions from: c:\Users\Izyan\Downloads\General\football_sim\.agents\worker_m1\DISPATCH.md
Read Explorer 1 findings from: c:\Users\Izyan\Downloads\General\football_sim\.agents\explorer_survey_1\handoff.md
Read Project Spec from: c:\Users\Izyan\Downloads\General\football_sim\.agents\orchestrator\PROJECT.md

Exclusive File Ownership:
You exclusively own and may create/edit files in:
c:\Users\Izyan\Downloads\General\football_sim\backend_go\pkg\models\
Do NOT touch any files outside this directory.

Your task:
Implement high-performance Go domain models, valuation curves, personality archetypes, and comprehensive unit tests in `backend_go/pkg/models/`:
- constants.go
- player.go
- club.go
- standings.go
- valuation.go
- personality.go
- player_test.go
- club_test.go
- standings_test.go
- valuation_test.go
- personality_test.go

MANDATORY INTEGRITY WARNING:
DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. A auditor will independently verify your work. Integrity violations WILL be detected and your work WILL be rejected.

Run `go test -v ./pkg/models/...` in `backend_go` to verify your implementation.
Write your completion report in `c:\Users\Izyan\Downloads\General\football_sim\.agents\worker_m1\handoff.md` with test outputs.
Update your `progress.md` frequently with a timestamp header for liveness.
When finished, send a message to orchestrator with your results summary.
