# Handoff Report: Reviewer 2 — Milestone 1 (R1 Wonderkid Growth Curve Rebalance)

**Reviewer**: Reviewer 2 (Roles: reviewer, critic)  
**Parent Agent**: Orchestrator (`3e97d900-03a3-4902-b7ad-5a877f27dac3`)  
**Working Directory**: `c:\Users\Izyan\General\football_sim\.agents\reviewer_m1_2`  
**Date**: 2026-09-10T00:31:00+08:00  
**Handoff Type**: Hard Handoff (Review Complete)  

---

## 1. Observation

### 1.1 Code and Test Inspection
1. **Implementation Files**:
   - `backend_go/pkg/growth/progression.go`:
     - Lines 56–67: Age multipliers (`<= 16: 1.05`, `<= 18: 1.00`, `<= 21: 0.90`, `<= 24: 0.75`, `25+: 0.50`).
     - Lines 84–87: Match XP formulas (`baseXP = matchRating * 2.2`, `goalXP = goals * 5.0`, `assistXP = assists * 3.0`).
     - Line 96: `bio.LevelXPTarget = math.Round(bio.LevelXPTarget*1.04*10) / 10`.
     - Lines 248–250: Mentorship random XP injection (`6.0 + rng * 6.0`).
   - `backend_go/pkg/growth/engine.go`:
     - Line 121: `LevelXPTarget: 160.0`.
     - Lines 256–267: `internalCalculateOVR` strictly enforces `cap = bio.Potential`.
   - `backend_go/pkg/growth/aging.go`:
     - Lines 123–133: Appearance bump for registered prodigies (`attrs != nil`):
       - `appearances >= 32 && base < 88`: `bump = 2`
       - `appearances >= 25`: `bump = 1`
       - `appearances >= 12`: `bump = 1`
       - `appearances < 12`: `bump = 0`
     - Lines 133–141: Target and nudging:
       - `target := minInt(potential, maxInt(base, cur)+bump)`
       - `ge.internalNudgeToOVR(playerID, cat, target)`
       - `finalOVR := ge.internalCalculateOVR(playerID, cat)`
     - Lines 144–159: Generic fallback preserved verbatim (`bump = 3 for >=20, 2 for >=8, 1 otherwise`).
   - `backend_go/pkg/growth/growth_curve_test.go`:
     - Tested single season across 5 seeds (1001–1005).

### 1.2 Test Execution and Failure Output
Running `go test -v -count=1 ./pkg/growth/...` and `go test -count=1 ./...` produces the following verbatim failure output:

```
--- FAIL: TestEmpirical_NormalSeason_75OVR (0.14s)
    empirical_stress_test.go:36: [Seed 880 Debug] startOVR=75, endOVR=80, gain=5, apps=38
    empirical_stress_test.go:39: [Seed 880] Gain +5 outside expected normal [+2, +4] (apps: 38, endOVR: 80)
    empirical_stress_test.go:47: === TestEmpirical_NormalSeason_75OVR Results (1000 runs) ===
    empirical_stress_test.go:48: Min Gain: +3, Max Gain: +5
    empirical_stress_test.go:50:   Gain +3: 41 (4.1%)
    empirical_stress_test.go:50:   Gain +4: 958 (95.8%)
    empirical_stress_test.go:50:   Gain +5: 1 (0.1%)
--- FAIL: TestEmpirical_AdversarialCeiling_SingleSeason (0.10s)
    empirical_stress_test.go:115: [Seed 5389] VIOLATION: Adversarial extreme season gain was +6 (> +5 ceiling)! in-season OVR: 79, endOVR: 81
    empirical_stress_test.go:120: === TestEmpirical_AdversarialCeiling_SingleSeason Results (500 runs) ===
    empirical_stress_test.go:121: Min Gain: +4, Max Gain: +6
    empirical_stress_test.go:123:   Gain +4: 49 (9.8%)
    empirical_stress_test.go:123:   Gain +5: 450 (90.0%)
    empirical_stress_test.go:123:   Gain +6: 1 (0.2%)
FAIL
FAIL	football_sim/pkg/growth	2.109s
FAIL
```

### 1.3 Backward Compatibility Verification
`TestChallenger2_GenericPlayer_ApplySeasonalGrowth_LegacyEquivalence` in `challenger_m1_2_test.go`:
```
=== RUN   TestChallenger2_GenericPlayer_ApplySeasonalGrowth_LegacyEquivalence
    challenger_m1_2_test.go:116: Successfully verified 130560 generic player permutations against legacy logic with 0 discrepancies.
--- PASS: TestChallenger2_GenericPlayer_ApplySeasonalGrowth_LegacyEquivalence (0.04s)
```

---

## 2. Logic Chain

1. **Acceptance Criteria Specification**:
   - `ORIGINAL_REQUEST.md` line 73 states: *"Simulating a full 44-week season with regular starts results in a wonderkid gaining +2 to +4 OVR (never exceeding +5 OVR in a single season)"*.
   - Line 92 states: *"cd backend_go && go test ./... passes 100% with 0 panics and 0 errors"*.
2. **Double-Dipping Mechanism**:
   - During the season, `ApplyMatchXP` triggers attribute upgrades (lines 94–150). Over 38 to 44 fixtures, accumulated XP raises player ratings by +1 to +4 OVR before the season concludes.
   - At season conclusion, `ApplySeasonalGrowth` receives the post-match rating as `cur` / `currentOVR`.
   - In `aging.go:133`, `target := minInt(potential, maxInt(base, cur) + bump)` unconditionally adds `bump` (+2 when `base < 88` and `apps >= 32`).
   - Because `base` is already elevated by in-season match gains, adding +2 on top results in total seasonal growth of $\text{InSeasonGain} + 2$.
3. **Breach Reproduction**:
   - In normal season simulation (`TestEmpirical_NormalSeason_75OVR`, 1000 runs), Seed 880 earned +3 in-season (75 -> 78) and +2 at season end, ending at 80 OVR (**+5 gain**).
   - In adversarial superstar performance (`TestEmpirical_AdversarialCeiling_SingleSeason`, 500 runs), Seed 5389 earned +4 in-season (75 -> 79) and +2 at season end, ending at 81 OVR (**+6 gain**).
   - This directly breaks the hard specification ceiling: *"never exceeding +5 OVR in a single season"*.
4. **Test Suite Failure**:
   - Because these tests are part of `backend_go/pkg/growth/`, `go test -count=1 ./...` exits with code 1.
5. **Conclusion Derivation**:
   - Steps 1–4 conclusively establish that the implementation fails acceptance criteria, requiring a verdict of **REQUEST_CHANGES**.

---

## 3. Caveats

1. **No Integrity Violation / Cheating**: Worker M1 did NOT hardcode player IDs, test checks, or mock results into production source code. The failure is an authentic calibration defect (uncapped double-dipping) that only surfaces across large sample sizes and superstar edge cases.
2. **Generic Regens Are Sound**: Generic regens (without registered biometric attributes) retain 100% backward compatibility across all 130,560 tested input permutations.
3. **Ceiling Clamping Is Sound**: The potential cap `[93, 96]` is universally respected across all functions; no player ever exceeds assigned potential or 96.

---

## 4. Conclusion

**Verdict**: **REQUEST_CHANGES**

Worker M1 must make a targeted update in `backend_go/pkg/growth/aging.go` (and optionally `progression.go`):
1. **Enforce Single-Season Growth Clamping**: In `ApplySeasonalGrowth`, ensure that total seasonal gain relative to the player's starting OVR cannot exceed +4 in normal circumstances and never exceeds +5 under any circumstance.
2. **Calibrate Bump Adaptively**: If a wonderkid has already gained $\ge +2$ OVR during the season from match XP, reduce the seasonal appearance bump to +1 or +0.
3. **Re-run Test Suite**: Ensure `go test -v -count=1 ./pkg/growth/...` and `go test -count=1 ./...` pass 100% with 0 errors and 0 failures.

---

## 5. Verification Method

To verify whether the requested changes have resolved the issue:

1. **Run Full Backend Regression Suite**:
   ```pwsh
   cd backend_go
   go test -count=1 ./...
   ```
   *Expectation*: 100% PASS across all 10 packages with 0 failures, 0 panics, exit code 0.

2. **Run Growth Package Tests with Empirical Monte Carlo Suites**:
   ```pwsh
   cd backend_go
   go test -v -count=1 ./pkg/growth/...
   ```
   *Expectation*: `TestEmpirical_NormalSeason_75OVR` (1,000 runs) passes with all gains in `[+2, +4]`; `TestEmpirical_AdversarialCeiling_SingleSeason` (500 runs) passes with all gains $\le +5$.
