## 2026-09-09T16:32:39Z
You are Worker M1 Iteration 2 (R1 Wonderkid Growth Curve Rebalance Fix).
Your working directory is: c:\Users\Izyan\General\football_sim\.agents\worker_m1_r2
Authoritative user request: c:\Users\Izyan\General\football_sim\.agents\ORIGINAL_REQUEST.md
Project architecture & contracts: c:\Users\Izyan\General\football_sim\.agents\orchestrator_v2\PROJECT.md
Review feedback from Challengers & Reviewers:
- Challenger 1 handoff: c:\Users\Izyan\General\football_sim\.agents\challenger_m1_1\handoff.md
- Challenger 2 handoff: c:\Users\Izyan\General\football_sim\.agents\challenger_m1_2\handoff.md
- Reviewer 2 handoff: c:\Users\Izyan\General\football_sim\.agents\reviewer_m1_2\handoff.md

MANDATORY INTEGRITY WARNING:
DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. A teamwork_preview_auditor will independently verify your work. Integrity violations WILL be detected and your work WILL be rejected.

CRITICAL DEFECT TO FIX:
1. Hard ceiling violation: Under extreme adversarial performance (44 appearances, 10.0 rating, 40 goals, 25 assists; Seed 5389), single-season gain reached +6 OVR (start 75 -> in-season 79 -> end-of-season 81), breaching acceptance criterion: "never exceeding +5 OVR in a single season".
2. Normal season breach: In a 1,000-season normal simulation, Seed 880 gained +5 OVR, exceeding the expected normal [+2, +4] corridor.
3. Root cause: In `backend_go/pkg/growth/aging.go` (`ApplySeasonalGrowth`), `bump = 2` is added unconditionally on top of `cur` (the OVR after match XP level-ups), without tracking cumulative seasonal gain against season-start baseline.

REQUIRED IMPLEMENTATION:
1. In `backend_go/pkg/growth/aging.go` (and `engine.go` / `models/biometric.go` if appropriate):
   - Track or determine season-start OVR (e.g. `bio.SeasonStartOVR` or baseline OVR initialized in `RegisterProdigy` and updated at the end of `ApplySeasonalGrowth` / season reset).
   - In `ApplySeasonalGrowth`, for registered wonderkids (`attrs != nil`):
     - Calculate in-season gain already achieved: `inSeasonGain := cur - seasonStartOVR`.
     - Adapt `bump` accordingly: total seasonal gain `(cur - seasonStartOVR) + bump` must be bounded such that normal seasons with regular starts achieve [+2, +4] OVR gain.
     - Enforce a STRICT HARD CEILING: `target <= seasonStartOVR + 5`. Single-season growth MUST NEVER exceed +5 OVR under any circumstances, even with 44 matches, 10.0 ratings, and 40 goals.
     - At the end of `ApplySeasonalGrowth`, update `bio.SeasonStartOVR = finalOVR` for the next season.
   - For generic regens without attributes (`attrs == nil`), retain exact legacy fallback logic (+3, +2, +1) unchanged.
2. Update `growth_curve_test.go`:
   - Add empirical test covering 500 adversarial extreme seasons (verify max gain <= 5).
   - Add empirical test covering 1,000 normal seasons (verify every season gain is in [+2, +4], no +5 in normal play).
   - Ensure all existing and new tests pass cleanly.
3. Run `go test -v -count=1 ./pkg/growth/...` and `go test -count=1 ./...` in `backend_go` to confirm 100% pass across all 10 packages.
4. Write report and `handoff.md` to `c:\Users\Izyan\General\football_sim\.agents\worker_m1_r2\`, then send completion message to parent.
