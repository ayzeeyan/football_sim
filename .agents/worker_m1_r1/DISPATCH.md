## 2026-09-09T16:20:00Z
You are Worker M1 (R1 Wonderkid Growth Curve Rebalance).
Your working directory is: c:\Users\Izyan\General\football_sim\.agents\worker_m1_r1
Authoritative user request: c:\Users\Izyan\General\football_sim\.agents\ORIGINAL_REQUEST.md
Project architecture & contracts: c:\Users\Izyan\General\football_sim\.agents\orchestrator_v2\PROJECT.md
Survey Explorer 1 handoff & report: c:\Users\Izyan\General\football_sim\.agents\explorer_survey_1\handoff.md and report.md

MANDATORY INTEGRITY WARNING:
DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. A teamwork_preview_auditor will independently verify your work. Integrity violations WILL be detected and your work WILL be rejected.

File Ownership:
You exclusively own and may edit:
- backend_go/pkg/growth/progression.go
- backend_go/pkg/growth/engine.go
- backend_go/pkg/growth/aging.go
- backend_go/pkg/growth/ test files (e.g. backend_go/pkg/growth/growth_curve_test.go)
Do not edit files outside backend_go/pkg/growth/.

Implementation Tasks:
1. Rebalance match XP formulas, level-up target scaling, and mentorship in backend_go/pkg/growth/progression.go:
   - baseXP = matchRating * 2.2
   - goalXP = float64(goals) * 5.0
   - assistXP = float64(assists) * 3.0
   - ageMult: <= 16: 1.05, <= 18: 1.00, <= 21: 0.90, <= 24: 0.75, 25+: 0.50
   - Level-up scaling: bio.LevelXPTarget = math.Round(bio.LevelXPTarget * 1.04 * 10) / 10
   - Mentorship tick XP: randXP := 6.0 + ge.rng.Float64() * (12.0 - 6.0)
2. In backend_go/pkg/growth/engine.go:
   - Initial LevelXPTarget: 160.0 in RegisterProdigy
3. In backend_go/pkg/growth/aging.go:
   - In ApplySeasonalGrowth:
     - For registered wonderkids with attributes (attrs != nil):
       - Calibrate appearance bump: if appearances >= 25: bump = 1 (or +2 if appearances >= 32 and current OVR < 88); if appearances >= 12: bump = 1; else bump = 0.
       - Clamp target to potential strictly.
     - For generic regens without attributes (attrs == nil):
       - Retain exact legacy fallback (bump = 3 for >=20, 2 for >=8, 1 otherwise) so existing non-wonderkid tests pass 100%.
4. Add comprehensive tests in backend_go/pkg/growth/growth_curve_test.go verifying:
   - Simulating a full 44-week season with regular starts results in a wonderkid gaining +2 to +4 OVR (never exceeding +5 OVR).
   - Wonderkids develop along multi-year trajectory reaching ~79–82 OVR by age 16, ~85–88 OVR by age 18, and approaching canonical 93–96 ceiling in early 20s.
   - Wonderkid potentials remain strictly within [93, 96].
   - All tests pass in backend_go/pkg/growth/.
5. Run go test -v -count=1 ./pkg/growth/... in backend_go and verify all tests pass.
6. Write your report to c:\Users\Izyan\General\football_sim\.agents\worker_m1_r1\report.md and handoff.md, then send a completion message to parent.
