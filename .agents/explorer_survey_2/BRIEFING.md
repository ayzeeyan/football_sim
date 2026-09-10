# BRIEFING — 2026-09-10T00:10:00Z

## Mission
Investigate R2 (44 matchweeks, quadruple round-robin, slates, cups) and R3 (position-aware formations, coordinates, pitch rendering, radar) across backend_go and frontend.

## 🔒 My Identity
- Archetype: explorer
- Roles: investigation, synthesis
- Working directory: c:\Users\Izyan\Downloads\General\football_sim\.agents\explorer_survey_2
- Original parent: 7396780a-6b1b-46cf-b464-6d10f96706f4
- Milestone: Chunk 1 Survey - Growth Engine & Biometrics
- Archetype: explorer
- Roles: investigation, synthesis
- Working directory: c:\Users\Izyan\General\football_sim\.agents\explorer_survey_2
- Original parent: 3e97d900-03a3-4902-b7ad-5a877f27dac3
- Milestone: Survey Explorer 2 — R2 Calendar & R3 Tactics

## 🔒 Key Constraints
- Read-only investigation — do NOT implement
- Analyze Python growth systems (growth.py) and existing Go growth files in backend_go/pkg/growth
- Detail biometric growth, aging decline curves (30+), youth development (<25), and Go test/implementation status
- Produce structured 5-component handoff.md and update progress.md
- Investigate R2 (44 matchweeks, quadruple round-robin, slates, cups) and R3 (position-aware formations, coordinates, pitch rendering, radar) across backend_go and frontend
- Write comprehensive report.md and handoff.md to c:\Users\Izyan\General\football_sim\.agents\explorer_survey_2
- Communicate via send_message to parent (id: 3e97d900-03a3-4902-b7ad-5a877f27dac3)

## Current Parent
- Conversation ID: 3e97d900-03a3-4902-b7ad-5a877f27dac3
- Updated: 2026-09-10T00:10:00Z

## Investigation State
- **Explored paths**:
  - `backend_go/pkg/tournament/constants.go` (LeagueRounds=33, UCLFinalWeek=33, MonthBands, LeaguePhase, WeekChapter, DerbyNames)
  - `backend_go/pkg/tournament/fixtures.go` (GenerateLeagueFixtures triple round-robin 33-round Berger polygon, venue balancing)
  - `backend_go/pkg/tournament/cup.go` & `knockout.go` (Champions Cup 2 groups of 6, QF/SF/Final; Super Cup 12 seeds, play-in, QF, SF, Final)
  - `backend_go/pkg/tournament/season.go` (AdoptLongSeason, RestartCurrentSeason, StartNewSeason, rollover, awards)
  - `backend_go/pkg/tournament/sim.go` & `slate_pool.go` (slateUnlocked, simulateRemaining, parallel worker pool, serial application)
  - `backend_go/pkg/tournament/weekly.go` & `school.go` (weekly ticks, exam weeks, player of week, month crowns)
  - `backend_go/pkg/models/club.go` (GetStartingEleven, AvailableSquad, fatigue rotation)
  - `backend_go/pkg/models/constants.go` (ExamWeeks, GetPositionCategory, KitColors)
  - `backend_go/pkg/models/standings.go` (StandingsRow, SortStandings, SortClubs)
  - `backend_go/pkg/matchengine/live.go` (Coordinate, LivePlayerRadar, baseHomeCoords, baseAwayCoords, radarPlayers, stance shifts)
  - `backend_go/pkg/server/server.go` (WebSocket tick building, home_coords serialization, GetCalendar endpoint, GetFixtures)
  - `frontend/src/components/PitchCanvas.tsx` (Canvas 2D render loop, player dots, category rings, position labels)
  - `frontend/src/components/StandingsTab.tsx` (tableNarrative pace*33, calendar strip, week progress, simulate remaining)
  - `frontend/src/components/MatchDetailModal.tsx` (FULL_PITCH_SLOTS static 4-3-3 slot mapping, tactical display)
  - `frontend/src/components/CalendarStrip.tsx`, `HalfTimeDugout.tsx`, `PreMatchModal.tsx`, `MatchdayTab.tsx`
  - Tests: `tournament_test.go`, `packet9_cycle_test.go`, `chunk3_backfill_test.go`, `chunk3_live_test.go`, `server_test.go`
- **Key findings**:
  - **R2**: Currently hardcoded to 33 rounds (triple round-robin: 198 league matches). Quadruple round-robin expands this to 4 full cycles of 11 rounds (44 matchweeks, 264 matches total, 6 matches per week slate). Every pair of clubs plays 4 times (exactly 2 home, 2 away for perfect symmetry). Together with UCL (up to 10 matches) and Super Cup (up to 4 matches), elite clubs play 57–58 matches per season (~55–60 target).
  - **R3**: Generic left-wing clustering is caused by a double bug: (1) `GetPositionCategory("CAM")` categorizes CAM as `FWD`, sorting CAM into `fwds` list in `GetStartingEleven()`; (2) `radarPlayers` and `PitchCanvas`/`MatchDetailModal` statically assign coordinates by slice index 0..10 (`baseHomeCoords[8]` = `{0.58, 0.20}` which is Left Wing). So CAMs (including wonderkids Cantalejo, Bantol, Baguio) were placed directly on the left wing! Solved by position-aware formation coordinate assignment: CAM centrally at `{0.51, 0.50}`, CDM deep central at `{0.31, 0.50}`, CMs in channels at `{0.39, 0.33/0.67}`, flank players on wings, ST/CF centrally leading attack at `{0.63, 0.50}`.
- **Unexplored areas**: None. Both R2 and R3 code paths fully mapped across backend and frontend.

## Key Decisions Made
- Mapped 4-cycle quadruple round-robin with balanced venue distribution (Cycle 1: H/A, Cycle 2: A/H, Cycle 3: H/A, Cycle 4: A/H).
- Expanded MonthBands to 44 matchweeks across 10 months (August to May).
- Aligned UCL knockouts into weeks 35-44 with UCL Final on week 44.
- Designed `AssignPitchCoordinates` mapping natural positions to coordinates without left-wing bias.
- Identified all frontend touchpoints in `StandingsTab.tsx`, `MatchDetailModal.tsx`, `App.tsx`, `api.ts`.

## Artifact Index
- DISPATCH.md — Survey task instructions
- BRIEFING.md — Situational awareness
- progress.md — Liveness heartbeat and progress log
- report.md — Comprehensive survey report
- handoff.md — 5-component handoff report


