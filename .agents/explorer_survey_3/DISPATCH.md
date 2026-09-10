# Dispatch Task: Survey Explorer 3 — R4 Transfers/Warchests & Build/Test Pipelines

**Original Request**: c:\Users\Izyan\General\football_sim\.agents\ORIGINAL_REQUEST.md
**Working Directory**: c:\Users\Izyan\General\football_sim\.agents\explorer_survey_3
**Role**: teamwork_preview_explorer (Survey Specialist - R4 Transfers & Build Pipeline)

## Objective
Investigate R4 requirements and overall build/verification requirements in ORIGINAL_REQUEST.md:
### R4: 12-Week Off-Season Transfer Window, Club Warchests & Wonderkid Rules
- Restructure off-season transfer window into 12 weekly stages (Weeks 1 to 12) advancing week-by-week.
- Persistent transfer warchest/budget for each of the 12 clubs initialized between €50M and €250M based on club stature.
- Player purchases deduct from buyer's warchest and add to seller's warchest; prevent bids exceeding available balance.
- Strict single-transfer rule per window (cannot transfer again in same window).
- Restrict 12 canonical wonderkids (`WK_` IDs) so they only transfer between the 12 Super League clubs, and at season reset / start of new campaign, return them automatically to their canonical original parent clubs.
- Check persistence (`saves/career.json`, serialization/deserialization).

### Verification & Stability Pipeline
- Check backend test structure (`backend_go/`, running tests, existing test coverage).
- Check frontend build setup (`frontend/package.json`, Vite, TypeScript compilation, components consuming transfer data).

## Exploration Scope
1. Inspect `backend_go/pkg/transfers/`, `backend_go/pkg/persistence/`, `backend_go/pkg/models/`, `backend_go/pkg/datamanager/`.
2. Inspect frontend components related to transfers (`frontend/src/components/Transfers.tsx`, etc.).
3. Identify current build scripts and verify how frontend and backend are built/tested.
4. Identify exact code locations and recommend changes and test additions.

## Output
Write your comprehensive report and handoff to:
`c:\Users\Izyan\General\football_sim\.agents\explorer_survey_3\report.md` and `handoff.md`.

## 2026-09-09T16:09:52Z
You are Survey Explorer 3 (R4 Transfers & Build Pipeline).
Your working directory is: c:\Users\Izyan\General\football_sim\.agents\explorer_survey_3
Your task is defined in: c:\Users\Izyan\General\football_sim\.agents\explorer_survey_3\DISPATCH.md
Authoritative user request: c:\Users\Izyan\General\football_sim\.agents\ORIGINAL_REQUEST.md

Please read ORIGINAL_REQUEST.md and DISPATCH.md. Investigate R4 (12-week off-season transfer window, club warchests, single-transfer lock, wonderkid loan returns) and the build & test pipeline (backend_go tests, frontend build).
Write your comprehensive report and handoff to:
c:\Users\Izyan\General\football_sim\.agents\explorer_survey_3\report.md and handoff.md.
When done, send a completion message back to parent.
