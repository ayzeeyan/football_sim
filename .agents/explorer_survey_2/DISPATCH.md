# Dispatch Task: Survey Explorer 2 — R2 Calendar & R3 Tactical Lineups/Coordinates

**Original Request**: c:\Users\Izyan\General\football_sim\.agents\ORIGINAL_REQUEST.md
**Working Directory**: c:\Users\Izyan\General\football_sim\.agents\explorer_survey_2
**Role**: teamwork_preview_explorer (Survey Specialist - R2 Calendar & R3 Tactics)

## Objective
Investigate R2 and R3 requirements in ORIGINAL_REQUEST.md:
### R2: 44-Matchweek Calendar & Seasonal Game Volume (~60 Games)
- Expand Super League schedule to 44 matchweeks (quadruple round-robin: 4 full cycles of 11 rounds across 12 clubs).
- Guarantee every league matchweek slate contains all 6 league fixtures (12 clubs playing each week), with no dropped or truncated matches.
- Elite clubs play ~55-60 total matches per season (league + Champions Cup + Super Cup).
- Update calendar strips and standings calculations.

### R3: Position-Driven Tactical Lineup & Pitch Coordinates
- Implement position-aware formation logic and pitch coordinate assignment in `backend_go/pkg/models` and `backend_go/pkg/matchengine` reflecting natural positions:
  - `CAM`: centrally in attacking midfield between CMs and ST.
  - `CDM`: deep central midfield.
  - `CM`: central midfield channels.
  - Flank players (`LB`, `LWB`, `RB`, `RWB`, `LM`, `RM`, `LW`, `RW`): appropriately along wings.
  - Strikers / Center Forwards (`ST`, `CF`): centrally leading attack.
- Ensure live pitch canvas, lineup HUDs, and radar accurately render players according to assigned coordinates, eliminating left-wing clustering. Check frontend files (`frontend/src/components/`).

## Exploration Scope
1. Inspect `backend_go/pkg/tournament/` (calendar generation, rounds, fixtures, cups).
2. Inspect `backend_go/pkg/matchengine/` and `backend_go/pkg/models/` (formations, tactics, coordinates, pitch mapping).
3. Inspect `frontend/src/` for pitch canvas, radar, lineup display, and calendar strips.
4. Check tests for tournament and matchengine.
5. Identify exact code locations and recommend changes and test additions.

## Output
Write your comprehensive report and handoff to:
`c:\Users\Izyan\General\football_sim\.agents\explorer_survey_2\report.md` and `handoff.md`.

## 2026-09-10T00:09:52Z
You are Survey Explorer 2 (R2 Calendar & R3 Tactics).
Your working directory is: c:\Users\Izyan\General\football_sim\.agents\explorer_survey_2
Your task is defined in: c:\Users\Izyan\General\football_sim\.agents\explorer_survey_2\DISPATCH.md
Authoritative user request: c:\Users\Izyan\General\football_sim\.agents\ORIGINAL_REQUEST.md

Please read ORIGINAL_REQUEST.md and DISPATCH.md. Investigate R2 (44 matchweeks, quadruple round-robin, slates, cups) and R3 (position-aware formations, coordinates, pitch rendering, radar) across backend_go and frontend.
Write your comprehensive report and handoff to:
c:\Users\Izyan\General\football_sim\.agents\explorer_survey_2\report.md and handoff.md.
When done, send a completion message back to parent.
