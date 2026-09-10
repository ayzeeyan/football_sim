# BRIEFING — 2026-09-09T16:15:30Z

## Mission
Investigate R4 (12-week off-season transfer window, club warchests, single-transfer lock, wonderkid loan returns) and build/test pipelines.

## 🔒 My Identity
- Archetype: explorer
- Roles: Survey Specialist - R4 Transfers & Build Pipeline
- Working directory: c:\Users\Izyan\General\football_sim\.agents\explorer_survey_3
- Original parent: 3e97d900-03a3-4902-b7ad-5a877f27dac3
- Milestone: Exploration & Investigation Survey

## 🔒 Key Constraints
- Read-only investigation — do NOT implement source changes
- Focus on R4 (Transfers, Warchests, Single-Transfer Lock, Wonderkid Loan Returns) and Build & Test Pipeline
- Output report.md and handoff.md in working directory
- Communicate back to parent via send_message

## Current Parent
- Conversation ID: 3e97d900-03a3-4902-b7ad-5a877f27dac3
- Updated: not yet

## Investigation State
- **Explored paths**: `backend_go/pkg/transfers`, `backend_go/pkg/persistence`, `backend_go/pkg/managers`, `backend_go/pkg/tournament`, `backend_go/pkg/server`, `backend_go/cmd/server`, `frontend/src/components/TransfersTab.tsx`, `frontend/src/services/api.ts`, `frontend/src/types/index.ts`, `scripts/run.ps1`
- **Key findings**:
  1. Transfer window uses unbounded daily ticks (`CurrentDay`); needs 12 weekly stages (`CurrentWeek` 1..12).
  2. Budgets in `BuildManagers` do not reach €250M; `aiInitiateBid` does not check available budget, causing negative balances.
  3. No single-transfer lock exists in `transfers.go`; completed transfers can be traded again in the same window.
  4. `season.go` homecoming returns ALL players whose `OriginalClubID != ClubID`, resetting permanent non-wonderkid transfers; must be restricted to wonderkids.
  5. Backend tests pass 100% (8.1s); Frontend build passes 100% (4.05s, Bun-only).
- **Unexplored areas**: None for R4 survey scope. Investigation complete.

## Key Decisions Made
- Structured the 12-week progression to maintain `window_day` compatibility alongside `window_week`.
- Formulated stature-based warchests for all 12 Super League clubs within [€50M, €250M].
- Designed strict solvency guards and single-transfer lock per window.
- Isolated homecoming in `ResetNewSeason` strictly to wonderkids (`UniverseWonderkid` / `WK_`).

## Artifact Index
- c:\Users\Izyan\General\football_sim\.agents\explorer_survey_3\DISPATCH.md — Task definition
- c:\Users\Izyan\General\football_sim\.agents\explorer_survey_3\BRIEFING.md — Persistent context index
- c:\Users\Izyan\General\football_sim\.agents\explorer_survey_3\progress.md — Liveness & heartbeat
- c:\Users\Izyan\General\football_sim\.agents\explorer_survey_3\report.md — Comprehensive Survey Report
- c:\Users\Izyan\General\football_sim\.agents\explorer_survey_3\handoff.md — 5-Component Handoff Report
