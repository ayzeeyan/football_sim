# Empirical Stress Test Report: Milestone 1 (R1 Wonderkid Growth Curve Rebalance)

**Challenger**: Challenger 1 (EMPIRICAL CHALLENGER — critic, specialist)  
**Date**: 2026-09-10T00:32:00+08:00  
**Target Package**: `backend_go/pkg/growth/`  
**Verdict**: **REQUEST_CHANGES**  

---

## 1. Executive Summary & Verdict

We conducted exhaustive, empirical adversarial stress testing of the rebalanced wonderkid growth curve implementation in `backend_go/pkg/growth/`. Across thousands of simulated seasons and multi-year careers, we uncovered a **reproducible critical defect** where the hard single-season growth ceiling of +5 OVR is violated, alongside boundary leaks in standard season distributions.

### Verdict: **REQUEST_CHANGES**
- **Critical Violation**: Under high-performance / adversarial conditions (44 appearances, 10.0 rating, 40 goals, 25 assists, elite mentor), wonderkids can gain **+6 OVR in a single season** (e.g. Seed `5389`: 75 OVR -> 81 OVR). This directly violates the hard acceptance criterion in `ORIGINAL_REQUEST.md`:
  > *"Simulating a full 44-week season with regular starts results in a wonderkid gaining +2 to +4 OVR (never exceeding +5 OVR in a single season)."*
- **Normal Season Range Breach**: Under realistic regular starter conditions (38 appearances, ratings 6.5–8.0, 0.3 goals/game), Seed `880` gained **+5 OVR** (75 OVR -> 80 OVR), exceeding the specified `[+2, +4]` normal single-season corridor. In 1,000 runs, 0.0% gained +2 OVR, 4.1% gained +3 OVR, 95.8% gained +4 OVR, and 0.1% gained +5 OVR (skewed at the ceiling with average +3.96 OVR).
- **Breakout Star Vulnerability**: In realistic breakout star seasons (44 appearances, 8.5 rating, 30 goals), **22.2%** of seasons resulted in +5 OVR gains.

---

## 2. Test Suites & Empirical Findings

### Test Suite 1: Single Season Normal Performance (1,000 Monte Carlo Runs)
- **Configuration**: Starting age 14, 75 OVR FWD, potential 95, 38 appearances (due to 6 middle school exam weeks), match ratings 6.5–8.0, 0.35 goal probability, 0.22 assist probability, 82 OVR mentor.
- **Results**:
  - Minimum Gain: **+3 OVR**
  - Maximum Gain: **+5 OVR**
  - Average Gain: **+3.96 OVR**
- **Distribution**:
  - `+2 OVR`: 0 / 1000 (0.0%)
  - `+3 OVR`: 41 / 1000 (4.1%)
  - `+4 OVR`: 958 / 1000 (95.8%)
  - `+5 OVR`: 1 / 1000 (0.1%) — **Breach of [+2, +4] corridor**
- **Specific Reproducible Failure**:
  - **Seed 880**: In-season match XP and mentorship nudged attributes from 75 OVR to 78 OVR (+3 in-season). At season end, `ApplySeasonalGrowth` unconditionally applied `bump = 2`, resulting in `endOVR = 80`. Total gain: **+5 OVR**.

### Test Suite 2: Adversarial Extreme Performance (500 Monte Carlo Runs)
- **Configuration**: Starting age 14, 75 OVR FWD, potential 95, 44 appearances, 10.0 rating every match, 40 goals, 25 assists, 92 OVR dedicated pro mentor.
- **Results**:
  - Minimum Gain: **+4 OVR**
  - Maximum Gain: **+6 OVR**
- **Distribution**:
  - `+4 OVR`: 49 / 500 (9.8%)
  - `+5 OVR`: 450 / 500 (90.0%)
  - `+6 OVR`: 1 / 500 (0.2%) — **CRITICAL HARD CEILING VIOLATION**
- **Specific Reproducible Failure**:
  - **Seed 5389**: In-season match XP upgraded attributes from 75 OVR to 79 OVR (+4 in-season). At season end, `ApplySeasonalGrowth` with 44 appearances assigned `bump = 2` on top of 79, nudging the player to `endOVR = 81`. Total gain: **+6 OVR** (81 - 75 = +6).

### Test Suite 3: Breakout Star Season (500 Monte Carlo Runs)
- **Configuration**: 44 appearances, 8.5 rating, 30 goals, 15 assists, 85 OVR mentor.
- **Results**:
  - `+4 OVR`: 389 / 500 (77.8%)
  - `+5 OVR`: 111 / 500 (22.2%)
- **Observation**: For an elite breakout season without perfect 10.0 ratings, almost a quarter (22.2%) of players gain +5 OVR, indicating that the curve lacks deceleration headroom when performance is high.

### Test Suite 4: Parameter Grid Sweep (300 Combinations, 10 Runs Each)
- **Sweep Dimensions**:
  - Ratings: `[5.0, 6.0, 7.0, 8.0, 9.0, 10.0]`
  - Goals: `[0, 5, 15, 30, 40]`
  - Appearances: `[0, 5, 11, 12, 24, 25, 31, 32, 38, 44]`
- **Key Empirical Observations**:
  - `Rating 5.0, 0 goals, 0 apps`: +0 OVR gain (end 75).
  - `Rating 5.0, 0 goals, 12 apps`: +1 OVR gain (end 76).
  - `Rating 5.0, 0 goals, 44 apps`: +3 OVR gain (end 78).
  - `Rating 7.0, 15 goals, 32 apps`: +4 OVR gain (end 79).
  - `Rating 10.0, 40 goals, 44 apps`: +4 to +6 OVR gain.

### Test Suite 5: Multi-Year Trajectory Across 8 Seasons (All 12 Canonical Wonderkids)
Simulated 30 multi-year careers (ages 14 to 22) per canonical prodigy (360 careers total):

| Wonderkid | Start OVR | Potential | Avg @ Age 16 (Target ~79–82) | Avg @ Age 18 (Target ~85–88) | Avg @ Age 22 (Target ~93–96) | Max Single Season Gain |
|---|:---:|:---:|:---:|:---:|:---:|:---:|
| **Venjamin Valerio** | 78 | 96 | 84.97 | 89.67 | 94.33 | +4 |
| **Maverick Cantalejo** | 77 | 95 | 83.93 | 89.00 | 93.50 | +4 |
| **Yeshua Emmanuel Gocotano** | 75 | 93 | 81.97 | 87.67 | 92.27 | +4 |
| **Izyan Levin Bantol** | 76 | 95 | 82.90 | 88.60 | 93.03 | +4 |
| **James Bernard Rizon** | 76 | 94 | 82.90 | 88.60 | 93.03 | +4 |
| **Reid Randell Libatan** | 76 | 95 | 82.90 | 88.60 | 93.03 | +4 |
| **Ashle Zylle Baguio** | 75 | 95 | 81.97 | 87.67 | 92.30 | +4 |
| **Cliergy Jave Lanticse** | 75 | 94 | 81.97 | 87.67 | 92.30 | +4 |
| **Ezail Zamora** | 77 | 96 | 83.93 | 89.00 | 93.50 | +4 |
| **Earl Josh Hernando** | 75 | 94 | 81.97 | 87.67 | 92.30 | +4 |
| **Rich Lorenz Suico** | 75 | 94 | 81.97 | 87.67 | 92.30 | +4 |
| **Jhed Anthony Guinita** | 75 | 94 | 81.97 | 87.67 | 92.30 | +4 |

- **75 OVR Starting Wonderkids**:
  - Average at age 16 is **81.97 OVR**, perfectly inside the `~79–82` target.
  - Average at age 18 is **87.67 OVR**, perfectly inside the `~85–88` target.
  - Average at age 22 is **92.30 OVR**, approaching the 93–96 potential ceiling.
- **77–78 OVR Starting Wonderkids**:
  - Track 2–3 OVR higher throughout their careers (Valerio reaches 89.67 by age 18), because starting ratings in `dataset.json` / `prodigies.go` are 77–78 rather than 75.

### Test Suite 6: Boundary, Edge Cases, and Invariants
- **Potential Ceilings**: Tested under 500 extreme matches with 10.0 ratings and max mentorship. Wonderkids **NEVER exceeded potential ceiling** [93, 96]. 100% compliant.
- **0 Appearances (Benched Entire Season)**: End OVR is 75 (gain is strictly +0 OVR). 100% compliant.
- **Veteran Aging Decline Isolation**: Tested across ages 14–29 (0 drop) and ages 30+ (exact -1, -2, -3 drops down to 35 attribute floor and 55 OVR floor). 100% compliant.
- **Generic Player Fallback (`attrs == nil`)**: Verified 100% legacy equivalence across 32,640 permutations. 100% compliant.

---

## 3. Root Cause Analysis

The root cause of the +6 single-season overgrowth and the +5 normal season breach lies in the architectural decoupling between in-season match progression and off-season seasonal growth in `backend_go/pkg/growth/aging.go`:

```go
// backend_go/pkg/growth/aging.go:115-134
if attrs != nil {
    cur := ge.internalCalculateOVR(playerID, cat)
    base := cur
    if len(currentOVR) > 0 && currentOVR[0] > base {
        base = currentOVR[0]
    }

    var bump int
    if appearances >= 32 && base < 88 {
        bump = 2
    } else if appearances >= 25 {
        bump = 1
    } else if appearances >= 12 {
        bump = 1
    } else {
        bump = 0
    }

    target := minInt(potential, maxInt(base, cur)+bump)
    ge.internalNudgeToOVR(playerID, cat, target)
...
```

1. **Unconditional Stacking**: `ApplySeasonalGrowth` computes `target := minInt(potential, maxInt(base, cur) + bump)`. It evaluates `cur` (the OVR *after* all 44 weeks of match XP) and unconditionally adds `bump = 2` on top of it.
2. **Missing Season-Start Differential Clamp**: There is no check comparing `target` against the player's true rating at the start of the season (`seasonStartOVR`). If in-season XP already pushed the player up by +3 or +4 OVR:
   $$\text{Total Gain} = (\text{In-Season Gain}) + (\text{Seasonal Bump}) = 4 + 2 = +6 \text{ OVR}$$
3. **Flawed `base` Logic**:
   `if len(currentOVR) > 0 && currentOVR[0] > base`
   If a caller passes the season-start OVR in `currentOVR[0]` (e.g. 75), `currentOVR[0] > base` evaluates to `75 > 79` which is `false`. The season-start rating is discarded.

---

## 4. Actionable Remediation for Worker M1

To resolve this defect and achieve a bulletproof growth curve:

1. **Enforce Hard Single-Season Ceiling in `ApplySeasonalGrowth`**:
   In `backend_go/pkg/growth/aging.go`:
   When `currentOVR` is passed representing the season-start OVR (or via `baseOVR` in `BiometricProfile`):
   ```go
   // Calculate maximum allowed target OVR for this single season
   // Hard ceiling: never exceed seasonStartOVR + 4 (normal) or seasonStartOVR + 5 (absolute hard cap)
   if len(currentOVR) > 0 {
       seasonStart := currentOVR[0]
       // Dynamic bump compensation: subtract in-season gain from bump
       inSeasonGain := maxInt(0, cur - seasonStart)
       allowedBump := maxInt(0, bump - inSeasonGain)
       target = minInt(potential, cur + allowedBump)
       if target > seasonStart + 4 {
           target = seasonStart + 4
       }
   }
   ```
2. **Dynamic In-Season Offset**:
   Alternatively, calculate `bump` as the total desired season gain minus in-season gain:
   - Desired total season gain for starter (>= 32 apps) = +3 to +4 OVR.
   - If player already gained +2 in-season, seasonal bump should be `+1` or `+2`.
   - If player already gained +3 or +4 in-season, seasonal bump should be `+0` (or at most clamped to `startOVR + 4`).
3. **Calibrate Normal Distribution**:
   By factoring in-season gains into seasonal bumps, normal starter gains will be distributed realistically across `[+2, +3, +4]` rather than being 95.8% concentrated at `+4` with occasional leaks into `+5`.

---

## 5. Summary of Verification Commands

All empirical data reported above was generated by writing and running test harnesses in Go directly against `backend_go/pkg/growth/`.

- To verify baseline repository integrity (all 10 packages pass 100%):
  ```pwsh
  cd backend_go
  go test -count=1 ./...
  ```
- All temporary test scripts created for this empirical challenge were completely cleaned up; no source or test files remain in `.agents/` or outside git status.
