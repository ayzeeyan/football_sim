# Dispatch: Worker M3 (Data Ingestion, Deduplication, Canonical Wonderkids & Academy Intake)

Authoritative Request: c:\Users\Izyan\Downloads\General\football_sim\.agents\ORIGINAL_REQUEST.md
Working Directory: c:\Users\Izyan\Downloads\General\football_sim\.agents\worker_m3
Spec Miner 3 Report: c:\Users\Izyan\Downloads\General\football_sim\.agents\spec_miner_survey_3\handoff.md
Project Spec: c:\Users\Izyan\Downloads\General\football_sim\.agents\orchestrator\PROJECT.md

## Exclusive File Ownership
You exclusively own and may create/edit files in:
`backend_go/pkg/datamanager/`
DO NOT touch any files outside this directory. You may import and read from `backend_go/pkg/models` and `backend_go/pkg/growth`.

## Objective
Implement `backend_go/pkg/datamanager/` with full fidelity to `ORIGINAL_REQUEST.md` and `spec_miner_survey_3/handoff.md`:
1. `datamanager.go`:
   - `DataManager` struct, `NewDataManager(jsonPath string, ge *growth.GrowthEngine) *DataManager`.
   - `LoadDataset() error`: reads `dataset.json` (96 clubs, 2,294 players) into Go memory losslessly.
   - Resets all player match and career stats to 0 for fresh save kickoff.
   - `DedupePlayers() int`: guarantees invariant of exactly 0 duplicate players across and within clubs. Uses `PREFERRED_HOMES`, elite club priority, and highest `(OVR, appearances)` fallback. Merges stats via `max()`. Updates `SquadSize = len(Squad)`.
   - `ResetMarketToBaseline()`: snaps and clamps market values using `models.BaselineValue` and `models.ClampValue`.
2. `prodigies.go`:
   - `ELITE_PRODIGY_CONFIGS`: 12 canonical wonderkids with exact biometrics, age 14, potentials [93, 96] (never 99).
   - `PREFERRED_HOMES`: 12 transfer home mappings.
   - `InitializeEliteProdigies()`: Relocates Jhed Anthony Guinita from `FL1-OM` to `EPL-TOT`. Sets `Age = 14`, `Education = "middle_school"`, `EducationPending = false`, `Category = "FWD"`, `UniverseWonderkid = true`, IDs prefixed with `WK_`. Registers in `GrowthEngine`. Recomputes OVR and composure. Snaps and clamps market values.
   - `AdoptU14Prodigies()`, `GetEliteClubs()`.
3. `youth_intake.go`:
   - Name pools (54 first, 54 last), positions, personalities.
   - `RunYouthIntake(clubID string) ([]*models.Player, error)`: 2–4 graduates, squad cap 34, 20% golden generation (OVR 72–78, potential 90–95), standard (OVR 58–72, potential 75–92), registered in GrowthEngine.
4. `datamanager_test.go`:
   - Verifies 96 clubs, 2,294 players loaded losslessly.
   - Verifies exactly 0 duplicate players.
   - Verifies all 12 wonderkids start at age 14, in middle school, with canonical `WK_` IDs.
   - Verifies wonderkid potentials match biometrics exactly in range [93, 96] (never 99).
   - Verifies Jhed Anthony Guinita is at Tottenham Hotspur (`EPL-TOT`).
   - Verifies youth intake respects squad cap 34 and registers with GrowthEngine.

Run tests: `cd backend_go && go test -v ./pkg/datamanager/...`
Ensure 100% tests pass with zero compiler warnings and zero runtime panics.

MANDATORY INTEGRITY WARNING:
DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. A auditor will independently verify your work. Integrity violations WILL be detected and your work WILL be rejected.

## 2026-09-07T06:57:15Z
You are Worker M3 for the Football Sim Go backend rewrite Chunk 1.

MANDATORY: Read the authoritative user request first:
c:\Users\Izyan\Downloads\General\football_sim\.agents\ORIGINAL_REQUEST.md

Working Directory: c:\Users\Izyan\Downloads\General\football_sim\.agents\worker_m3
Read instructions from: c:\Users\Izyan\Downloads\General\football_sim\.agents\worker_m3\DISPATCH.md
Read Spec Miner 3 findings from: c:\Users\Izyan\Downloads\General\football_sim\.agents\spec_miner_survey_3\handoff.md
Read Project Spec from: c:\Users\Izyan\Downloads\General\football_sim\.agents\orchestrator\PROJECT.md

Exclusive File Ownership:
You exclusively own and may create/edit files in:
c:\Users\Izyan\Downloads\General\football_sim\backend_go\pkg\datamanager\
Do NOT touch any files outside this directory. You may import and use `football_sim/pkg/models` and `football_sim/pkg/growth`.

Your task:
Implement data ingestion, squad deduplication, 12 canonical U-14 wonderkids setup, and academy youth intake in `backend_go/pkg/datamanager/`:
- datamanager.go
- prodigies.go
- youth_intake.go
- datamanager_test.go

MANDATORY INTEGRITY WARNING:
DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. A auditor will independently verify your work. Integrity violations WILL be detected and your work WILL be rejected.

Run `go test -v ./pkg/datamanager/...` in `backend_go` to verify your implementation.
Also run `go test -v ./...` in `backend_go` to ensure everything compiles and passes across the entire project.
Write your completion report in `c:\Users\Izyan\Downloads\General\football_sim\.agents\worker_m3\handoff.md` with test outputs.
Update your `progress.md` frequently with a timestamp header for liveness.
When finished, send a message to orchestrator with your results summary.

