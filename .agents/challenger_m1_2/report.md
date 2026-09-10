# Empirical Challenge Report: Milestone 1 (R1 Wonderkid Growth Curve Rebalance)

**Challenger**: Challenger 2 (Empirical Challenger)  
**Parent Agent**: Orchestrator (`3e97d900-03a3-4902-b7ad-5a877f27dac3`)  
**Working Directory**: `c:\Users\Izyan\General\football_sim\.agents\challenger_m1_2`  
**Date**: 2026-09-10T00:31:00+08:00  
**Verdict**: **REQUEST_CHANGES**

---

## Challenge Summary

**Overall risk assessment**: **HIGH**  
While generic player growth logic, veteran aging decline, and potential ceiling bounds (<= potential, <= 96 OVR) are strictly preserved and robust, the full backend test suite (`go test -count=1 ./...`) **fails with exit code 1**. Specifically, the test suite reveals that under high match performance, a wonderkid's single-season gain can reach **+6 OVR**, directly violating the strict acceptance criterion: *"Simulating a full 44-week season with regular starts results in a wonderkid gaining +2 to +4 OVR (never exceeding +5 OVR in a single season)."*

---

## Challenges

### [Critical] Challenge 1: Single-Season Growth Exceeds +5 Hard Ceiling Under High Match Performance (+6 OVR Observed)

- **Assumption challenged**: The worker assumed that tuning `ageMult`, `baseXP`, `goalXP`, `assistXP`, and appearance bumps (+1/+2) would intrinsically keep single-season growth to <= +4 (and never > +5) without requiring an explicit season-gain ceiling clamp.
- **Attack scenario**: In `TestEmpirical_AdversarialCeiling_SingleSeason` (seed 5389), a 14-year-old wonderkid playing 44 matches with a 10.0 rating, 40 goals, 25 assists, and a 92-rated dedicated-pro mentor accumulates enough XP to gain +4 OVR in-season (rising from 75 to 79 OVR). At the end of the season, `ApplySeasonalGrowth` executes:
  ```go
  if appearances >= 32 && base < 88 {
      bump = 2
  }
  target := minInt(potential, maxInt(base, cur)+bump)
  ```
  Since `base = cur = 79`, `target` becomes `79 + 2 = 81`. The player finishes the season at 81 OVR, giving a single-season gain of `81 - 75 = +6 OVR`.
- **Blast radius**: Violates user acceptance criterion 73: *(never exceeding +5 OVR in a single season)*. Breaks multi-year progression pacing and accelerates wonderkids into senior-level ratings too quickly. Causes `go test -count=1 ./...` to fail.
- **Mitigation**:
  1. In `ApplySeasonalGrowth`, track or pass the season-opening baseline OVR (e.g. from `bio.BaselineOVR` or a parameter) and enforce:
     `maxAllowed := minInt(potential, seasonStartOVR + 5)`
     `target := minInt(maxAllowed, maxInt(base, cur)+bump)`
  2. Alternatively, calibrate in-season level-up XP or appearance bumps so that `(inSeasonGain + appearanceBump) <= 5` under all mathematical worst-case conditions.

### [Medium] Challenge 2: Normal Season Gain Exceeds Expected Range [+2, +4] (Seed 880 Yields +5 OVR)

- **Assumption challenged**: Regular starter wonderkids under realistic match performance will always gain between +2 and +4 OVR.
- **Attack scenario**: In `TestEmpirical_NormalSeason_75OVR` across 1,000 simulated seasons with realistic starter ratings (6.5 to 8.0) and 38 appearances:
  - 41 runs (4.1%) produced +3 OVR
  - 958 runs (95.8%) produced +4 OVR
  - 1 run (0.1%, Seed 880) produced **+5 OVR** (start 75 -> end 80).
- **Blast radius**: While +5 is within the hard ceiling (<= 5), it fails `TestEmpirical_NormalSeason_75OVR` which asserts that normal seasons fall strictly in [+2, +4].
- **Mitigation**: Slightly tighten match XP or ensure regular starter seasonal bump takes into account whether in-season growth already reached +3.

---

## Non-Regression & Edge Case Verification (PASSED)

### 1. Generic Players with No Biometric Profile (`attrs == nil`)
- **Status**: **PASS (100% Equivalence)**
- **Evidence**: Executed `TestChallenger2_GenericPlayer_ApplySeasonalGrowth_LegacyEquivalence` over **130,560 permutations** spanning:
  - Ages: 14 to 40
  - Appearances: 0 to 100
  - Potentials: 65 to 99
  - Categories: "FWD", "MID", "DEF", and ""
  - CurrentOVRs: omitted, 60, 70, 72, 75, 80, 85, 90, 94, 96
- **Result**: Zero discrepancies found between `GrowthEngine.ApplySeasonalGrowth`, `ApplySeasonalGrowthHelper(ge)`, `ApplySeasonalGrowthHelper(nil)`, and the exact legacy specification.

### 2. Veteran Aging Decline (30+)
- **Status**: **PASS (Zero Regression)**
- **Evidence**: Executed `TestChallenger2_VeteranAgingDecline_IntegrityAndIsolation`:
  - Non-veterans (< 30) suffered 0 physical attribute drop and 0 OVR drop.
  - Ages 30–33: exact drop of -1 to pace, stamina, strength, physicality; non-physical attributes 100% untouched.
  - Ages 34–35: exact drop of -2 to physical attributes; non-physical untouched.
  - Ages 36+: exact drop of -3 to physical attributes; non-physical untouched.
  - Physical attributes floor strictly at 35 even after 50 repeated decline calls.
  - `SeasonalOVRDrop` floors strictly at 55 even after 50 repeated decline calls.
  - 30+ veterans registered in the growth engine receive 0 youth seasonal growth bump.

### 3. Potential Clamping and 96 OVR Ceiling
- **Status**: **PASS (Strict Invariant Preserved)**
- **Evidence**: Executed `TestChallenger2_PotentialClamping_UniversalStrictness`, `TestChallenger2_TrainingAndPuberty_CeilingIntegrity`, and `TestChallenger2_AdversarialInputs_SeasonalGrowth`:
  - Canonical wonderkids have initial potentials strictly in `[93, 96]` (never 99).
  - Subjecting wonderkids to 200 consecutive 10.0-rated matches with 5 goals/5 assists, 30 consecutive seasons of 50 appearances, 100 puberty cycles, and 150 intensive training cycles NEVER caused `CalculateOVR` or `ApplySeasonalGrowth` to exceed potential or 96 OVR.
  - Direct memory manipulation (forcing all 13 technical attributes to 99) was tested: `CalculateOVR` and `ApplySeasonalGrowth` strictly clamped the rating to `bio.Potential` (<= 96).

---

## Stress Test Results

| Test Scenario | Expected Behavior | Actual Behavior | Result |
|---|---|---|---|
| `TestChallenger2_GenericPlayer_ApplySeasonalGrowth_LegacyEquivalence` (130,560 cases) | Match legacy math 100% | 130,560 / 130,560 exact matches | **PASS** |
| `TestChallenger2_VeteranAgingDecline_IntegrityAndIsolation` (ages 14-50) | Decline rates -1/-2/-3; floors 35/55; youth zero decline | Exact rates matched; floors 35 and 55 held; youth untouched | **PASS** |
| `TestChallenger2_PotentialClamping_UniversalStrictness` (all 12 wonderkids) | OVR <= potential and <= 96 under extreme matches, seasons, tampering | Max OVR observed <= potential and <= 96 across all tests | **PASS** |
| `TestChallenger2_SingleSeasonGain_BoundedNeverExceeds5` (120 seasons) | Gain +2 to +4 under standard regular starts; gain <= 5 | Gains strictly +2 to +4; no gain > 4 observed in standard suite | **PASS** |
| `TestChallenger2_AdversarialInputs_SeasonalGrowth` (negative apps, base > pot, etc.) | Clamped to potential, safe defaults | Handled gracefully without crash; potential respected | **PASS** |
| `TestChallenger2_TrainingAndPuberty_CeilingIntegrity` (100 puberty, 150 training) | OVR <= potential and <= 96 | OVR <= potential and <= 96 | **PASS** |
| `TestChallenger2_CanonicalWonderkids_10SeasonCareer_EndToEnd` | Milestones at 16, 18, 24; ceiling respected | Age 16: ~79-83 (75 starters); Age 18: ~85-89; Age 24: 91-96 | **PASS** |
| `TestEmpirical_NormalSeason_75OVR` (1,000 normal seasons) | Gain strictly [+2, +4] | Seed 880 gained +5 OVR | **FAIL** |
| `TestEmpirical_AdversarialCeiling_SingleSeason` (500 extreme seasons) | Gain <= 5 OVR | Seed 5389 gained +6 OVR (start 75 -> end 81) | **FAIL** |
| Full backend test suite (`go test -count=1 ./...`) | Exit code 0 across all packages | Exit code 1 due to failures in `pkg/growth` | **FAIL** |

---

## Final Verdict

**REQUEST_CHANGES**

**Action items for Worker M1**:
1. Implement a hard ceiling guard in `ApplySeasonalGrowth` ensuring that total single-season growth (in-season match XP gains + end-of-season appearance bump) **never exceeds +5 OVR** under any performance extremes.
2. Calibrate in-season XP progression and appearance bump so that normal season simulations stay within [+2, +4] OVR across all random seeds, resolving `TestEmpirical_NormalSeason_75OVR` and `TestEmpirical_AdversarialCeiling_SingleSeason`.
3. Ensure `go test -count=1 ./...` passes cleanly across all packages with 0 failures.
