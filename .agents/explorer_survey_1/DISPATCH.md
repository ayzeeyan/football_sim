# Dispatch Task: Survey Explorer 1 — R1 Wonderkid Growth Curve Rebalance

**Original Request**: c:\Users\Izyan\General\football_sim\.agents\ORIGINAL_REQUEST.md
**Working Directory**: c:\Users\Izyan\General\football_sim\.agents\explorer_survey_1
**Role**: teamwork_preview_explorer (Survey Specialist - R1 Growth)

## Objective
Investigate the R1 requirements in ORIGINAL_REQUEST.md:
- Rebalance match XP formulas, level-up XP scaling, and appearance progression in `backend_go/pkg/growth/`.
- Ensure U-14 wonderkids (starting age 14, OVR 72-75) gain an average of +2 to +4 OVR per full season under consistent playing time.
- Multi-year trajectory: ~79-82 OVR at age 16, ~85-88 OVR at age 18, approaching 93-96 ceiling in early 20s.
- Eliminate single-season leaps into world-class ratings.
- Verify potential caps stay strictly in [93, 96].

## Exploration Scope
1. Inspect `backend_go/pkg/growth/` files (growth engine, XP curves, training, match XP).
2. Inspect `backend_go/pkg/models/` for player attributes, OVR calculation, wonderkid initial setup.
3. Check existing tests in `backend_go/pkg/growth/` and see what tests currently verify growth.
4. Identify exact code locations, formulas, constants, and edge cases.
5. Provide a concrete implementation recommendation and test plan.

## Output
Write your comprehensive report and handoff to:
`c:\Users\Izyan\General\football_sim\.agents\explorer_survey_1\report.md` and `handoff.md`.

## 2026-09-09T16:09:52Z
You are Survey Explorer 1 (R1 Growth).
Your working directory is: c:\Users\Izyan\General\football_sim\.agents\explorer_survey_1
Your task is defined in: c:\Users\Izyan\General\football_sim\.agents\explorer_survey_1\DISPATCH.md
Authoritative user request: c:\Users\Izyan\General\football_sim\.agents\ORIGINAL_REQUEST.md

Please read ORIGINAL_REQUEST.md and DISPATCH.md. Investigate R1 requirements across backend_go (pkg/growth, pkg/models, and tests). Check current state of codebase and tests.
Write your comprehensive report and handoff to:
c:\Users\Izyan\General\football_sim\.agents\explorer_survey_1\report.md and handoff.md.
When done, send a completion message back to parent.

