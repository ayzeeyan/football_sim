# Explorer Investigation & Fix Strategy Report: Pointer-Aliased Deduplication Defect

**Agent**: Explorer Round 2 Instance 3 (`explorer_r2_3`)  
**Target Milestone**: Chunk 1 — Milestone 3 Remediation (`backend_go/pkg/datamanager`)  
**Date**: 2026-09-07  
**Verdict**: **Remediation Strategy Ready for Worker**  

---

## 1. Observation

### 1.1 Verbatim Reproduction of Test Failures
Executing the authoritative project test command from `ORIGINAL_REQUEST.md`:
```powershell
cd c:\Users\Izyan\Downloads\General\football_sim\backend_go
go test -v ./pkg/datamanager -run "TestChallenger_DuplicateInjection_(Identical|CrossClub)"
```
Output:
```
=== RUN   TestChallenger_DuplicateInjection_IdenticalPointerAttack
    challenger_stress_test.go:365: Identical pointer duplicate test: removed=0, remaining count=2
    challenger_stress_test.go:367: BUG FOUND: expected exactly 1 copy remaining of Identical Pointer Player, got 2 (removed=0)
--- FAIL: TestChallenger_DuplicateInjection_IdenticalPointerAttack (0.02s)
=== RUN   TestChallenger_DuplicateInjection_CrossClubSharedPointerAttack
    challenger_stress_test.go:415: Cross-club shared pointer test: removed=0, arsHas=true, cheHas=true
    challenger_stress_test.go:417: BUG FOUND: Cross Club Shared Player still present in both Arsenal and Chelsea! Dedupe failed (removed=0)
--- FAIL: TestChallenger_DuplicateInjection_CrossClubSharedPointerAttack (0.02s)
FAIL
FAIL	football_sim/pkg/datamanager	1.048s
FAIL
```

### 1.2 Inspection of Defective Code in `backend_go/pkg/datamanager/datamanager.go`
Lines 320–345:
```go
320:		for _, cp := range copies {
321:			if cp.player == keepPlayer {
322:				continue
323:			}
324:
325:			// Merge stats into canonical player via max()
326:			keepPlayer.Appearances = maxInt(keepPlayer.Appearances, cp.player.Appearances)
327:			keepPlayer.Goals = maxInt(keepPlayer.Goals, cp.player.Goals)
328:			keepPlayer.Assists = maxInt(keepPlayer.Assists, cp.player.Assists)
329:			keepPlayer.CareerGoals = maxInt(keepPlayer.CareerGoals, cp.player.CareerGoals)
330:			keepPlayer.CareerAssists = maxInt(keepPlayer.CareerAssists, cp.player.CareerAssists)
331:			keepPlayer.CareerApps = maxInt(keepPlayer.CareerApps, cp.player.CareerApps)
332:
333:			// Remove duplicate player from club squad
334:			newSquad := make([]*models.Player, 0, len(cp.club.Squad))
335:			for _, p := range cp.club.Squad {
336:				if p != cp.player {
337:					newSquad = append(newSquad, p)
338:				}
339:			}
340:			cp.club.Squad = newSquad
341:			cp.club.SquadSize = len(cp.club.Squad)
342:			removed++
343:		}
```

### 1.3 Inspection of Secondary Vulnerability in `backend_go/pkg/datamanager/prodigies.go`
Lines 305–319 (as flagged in Reviewer 2 Finding 2):
```go
305:		// Clean up any lingering duplicate copies of this wonderkid across all squads
306:		for _, c := range dm.ClubsList {
307:			newSquad := make([]*models.Player, 0, len(c.Squad))
308:			for _, p := range c.Squad {
309:				if p != prodigy && strings.EqualFold(p.FullName, cfg.FullName) {
310:					prodigy.Appearances = maxInt(prodigy.Appearances, p.Appearances)
311:					prodigy.Goals = maxInt(prodigy.Goals, p.Goals)
312:					prodigy.Assists = maxInt(prodigy.Assists, p.Assists)
313:					// extra duplicate is eliminated
314:				} else {
315:					newSquad = append(newSquad, p)
316:				}
317:			}
318:			c.Squad = newSquad
319:			c.SquadSize = len(c.Squad)
320:		}
```

### 1.4 Baseline Passing Components
All other Chunk 1 units are fully operational and passing:
- `football_sim/pkg/models`: 32/32 unit & stress tests pass (`TestBaselineValue`, `TestClampValue`, `TestPlayerEffectiveOVR`, `TestClubStartingEleven433`, `TestStandingsSortingTiebreakers`, etc.).
- `football_sim/pkg/growth`: 22/22 unit & stress tests pass (`TestGrowthEngine_RegisterProdigy`, `TestGrowthEngine_PubertySimulation`, `TestChallenger_AgingDecline_VeteransFloor35`, `TestChallenger_SeasonalOVRDrop_ExhaustiveGrid`, etc.).
- `football_sim/pkg/datamanager`: 20/22 unit & stress tests pass (Ingestion of 96 clubs & 2,294 players, Guinita relocation to EPL-TOT index 0, Youth Intake boundary caps 30–40, 10-season multi-round youth intake with 970 grads). Only the 2 pointer-aliasing duplicate injection tests fail.

---

## 2. Logic Chain

1. **Root Cause Mechanism**:
   - `DedupePlayers()` groups squad members into `byName` by normalized name (`strings.ToLower(strings.TrimSpace(p.FullName))`).
   - When duplicate players are injected by referencing the identical pointer (e.g. `ars.Squad = append(ars.Squad, p, p)` or `che.Squad = append(che.Squad, sharedP)`):
     - `copies` slice contains `[{club: ars, player: p}, {club: ars, player: p}]` or `[{club: ars, player: sharedP}, {club: che, player: sharedP}]`.
     - `canonicalCP` selects one copy (e.g. `copies[0]`), defining `keepPlayer = canonicalCP.player`.
     - In lines 321–323, the elimination loop executes:
       ```go
       for _, cp := range copies {
           if cp.player == keepPlayer {
               continue
           }
           ...
       }
       ```
     - For **every single element** in `copies`, `cp.player` holds the identical pointer address as `keepPlayer`.
     - Therefore, `cp.player == keepPlayer` evaluates to `true` on every iteration.
     - The entire body (stats merging, squad filtering, squad size recalculation, and `removed++`) is bypassed via `continue`.
     - `removed` remains 0, leaving all duplicate pointers in place.

2. **Inner Squad Purge Flaw**:
   - Even if the `continue` guard were removed, lines 335–339 perform squad filtering by pointer comparison:
     ```go
     for _, p := range cp.club.Squad {
         if p != cp.player {
             newSquad = append(newSquad, p)
         }
     }
     ```
   - If `cp.club == keepClub`, filtering by `p != cp.player` would wipe out **all** occurrences of `p`, deleting the canonical player entirely.
   - Conversely, across multiple clubs, filtering must differentiate between the canonical club (which retains exactly 1 occurrence) and non-canonical clubs (which retain 0 occurrences).

3. **Origin of Defect**:
   - In Python (`data_manager.py:287`), the loop checked `if player is keep_player: continue`.
   - In Go, pointer equality `cp.player == keepPlayer` mirrored Python's identity check `player is keep_player`.
   - While distinct heap structs worked seamlessly (passing 100-duplicate stress tests), pointer aliasing bypassed the logic completely.

4. **Remediation Architecture (Two-Step Algorithm)**:
   - **Step 1: Stats Merging via `max()` over distinct struct instances**:
     ```go
     for _, cp := range copies {
         if cp.player != keepPlayer {
             keepPlayer.Appearances = maxInt(keepPlayer.Appearances, cp.player.Appearances)
             keepPlayer.Goals = maxInt(keepPlayer.Goals, cp.player.Goals)
             keepPlayer.Assists = maxInt(keepPlayer.Assists, cp.player.Assists)
             keepPlayer.CareerGoals = maxInt(keepPlayer.CareerGoals, cp.player.CareerGoals)
             keepPlayer.CareerAssists = maxInt(keepPlayer.CareerAssists, cp.player.CareerAssists)
             keepPlayer.CareerApps = maxInt(keepPlayer.CareerApps, cp.player.CareerApps)
         }
     }
     ```
     - If `cp.player == keepPlayer`, merging with self is a redundant no-op and skipped.
     - If `cp.player != keepPlayer`, all season and career statistics are aggregated into `keepPlayer` before squad modification.
   - **Step 2: Universe-wide Squad Cleansing with Exact Canonical Slotting**:
     ```go
     canonicalRetained := false
     for _, c := range dm.ClubsList {
         newSquad := make([]*models.Player, 0, len(c.Squad))
         modified := false
         for _, p := range c.Squad {
             isMatch := strings.EqualFold(strings.TrimSpace(p.FullName), name) || p == keepPlayer
             if !isMatch {
                 newSquad = append(newSquad, p)
                 continue
             }

             if c.ClubID == keepClub.ClubID && !canonicalRetained {
                 canonicalRetained = true
                 newSquad = append(newSquad, keepPlayer)
             } else {
                 removed++
                 modified = true
             }
         }
         if modified {
             c.Squad = newSquad
             c.SquadSize = len(c.Squad)
         }
     }
     ```
     - Evaluates every squad player by normalized name AND pointer identity (`strings.EqualFold(...) || p == keepPlayer`).
     - In `keepClub`, the **first** matching player is replaced by `keepPlayer`, and `canonicalRetained` is marked `true`.
     - Any subsequent matching player in `keepClub` (e.g. intra-squad aliased duplicate) is omitted, triggering `removed++` and `modified = true`.
     - In any other club (`c.ClubID != keepClub.ClubID`), every matching player is omitted, triggering `removed++` and `modified = true`.
     - When `modified` is true, `c.Squad` is reassigned to `newSquad` and `c.SquadSize` is recalculated as `len(c.Squad)`.

5. **Secondary Hardening in `prodigies.go`**:
   - Apply the same pattern to `InitializeEliteProdigies()` at lines 305–320: ensure `isWonderkidMatch := p == prodigy || strings.EqualFold(strings.TrimSpace(p.FullName), cfg.FullName)`, retaining exactly one copy in `cid` and purging all other instances.

---

## 3. Caveats

1. **Fresh Ingestion vs In-Memory Duplication**:
   - In raw `dataset.json`, Go's `json.Unmarshal` allocates distinct heap structs for all 2,294 players. Raw ingestion naturally starts with 0 duplicates.
   - The pointer-aliasing defect manifests when players are shared programmatically (e.g. roster transfers, adversarial stress tests, mock fixture setups).
2. **Deterministic Retention Order**:
   - `canonicalRetained` guarantees that exactly 1 instance of the player exists in `keepClub.Squad` across the entire database of 96 clubs.
   - In `keepClub`, the player occupies the slot of the first duplicate occurrence.
3. **No Other Packages Impacted**:
   - The fix requires edits **only** to `backend_go/pkg/datamanager/datamanager.go` (and optionally `prodigies.go`).
   - No models, growth engine APIs, or interface signatures are changed.

---

## 4. Conclusion & Actionable Fix Strategy for Worker

### Exact Code Replacement 1: `backend_go/pkg/datamanager/datamanager.go`

**Target File**: `backend_go/pkg/datamanager/datamanager.go`  
**Target Lines**: 321–344  

#### Replace This:
```go
		for _, cp := range copies {
			if cp.player == keepPlayer {
				continue
			}

			// Merge stats into canonical player via max()
			keepPlayer.Appearances = maxInt(keepPlayer.Appearances, cp.player.Appearances)
			keepPlayer.Goals = maxInt(keepPlayer.Goals, cp.player.Goals)
			keepPlayer.Assists = maxInt(keepPlayer.Assists, cp.player.Assists)
			keepPlayer.CareerGoals = maxInt(keepPlayer.CareerGoals, cp.player.CareerGoals)
			keepPlayer.CareerAssists = maxInt(keepPlayer.CareerAssists, cp.player.CareerAssists)
			keepPlayer.CareerApps = maxInt(keepPlayer.CareerApps, cp.player.CareerApps)

			// Remove duplicate player from club squad
			newSquad := make([]*models.Player, 0, len(cp.club.Squad))
			for _, p := range cp.club.Squad {
				if p != cp.player {
					newSquad = append(newSquad, p)
				}
			}
			cp.club.Squad = newSquad
			cp.club.SquadSize = len(cp.club.Squad)
			removed++
		}
```

#### With This:
```go
		// 1. Merge stats from distinct duplicate instances into canonical player via max()
		for _, cp := range copies {
			if cp.player != keepPlayer {
				keepPlayer.Appearances = maxInt(keepPlayer.Appearances, cp.player.Appearances)
				keepPlayer.Goals = maxInt(keepPlayer.Goals, cp.player.Goals)
				keepPlayer.Assists = maxInt(keepPlayer.Assists, cp.player.Assists)
				keepPlayer.CareerGoals = maxInt(keepPlayer.CareerGoals, cp.player.CareerGoals)
				keepPlayer.CareerAssists = maxInt(keepPlayer.CareerAssists, cp.player.CareerAssists)
				keepPlayer.CareerApps = maxInt(keepPlayer.CareerApps, cp.player.CareerApps)
			}
		}

		// 2. Cleanse all squads: keep exactly 1 occurrence in keepClub, 0 in all other clubs
		canonicalRetained := false
		for _, c := range dm.ClubsList {
			newSquad := make([]*models.Player, 0, len(c.Squad))
			modified := false
			for _, p := range c.Squad {
				isMatch := strings.EqualFold(strings.TrimSpace(p.FullName), name) || p == keepPlayer
				if !isMatch {
					newSquad = append(newSquad, p)
					continue
				}

				if c.ClubID == keepClub.ClubID && !canonicalRetained {
					// Retain the single canonical player instance in keepClub
					canonicalRetained = true
					newSquad = append(newSquad, keepPlayer)
				} else {
					// Duplicate instance (either intra-club or cross-club) removed
					removed++
					modified = true
				}
			}
			if modified {
				c.Squad = newSquad
				c.SquadSize = len(c.Squad)
			}
		}
```

---

### Exact Code Replacement 2 (Recommended): `backend_go/pkg/datamanager/prodigies.go`

**Target File**: `backend_go/pkg/datamanager/prodigies.go`  
**Target Lines**: 305–319  

#### Replace This:
```go
		// Clean up any lingering duplicate copies of this wonderkid across all squads
		for _, c := range dm.ClubsList {
			newSquad := make([]*models.Player, 0, len(c.Squad))
			for _, p := range c.Squad {
				if p != prodigy && strings.EqualFold(p.FullName, cfg.FullName) {
					prodigy.Appearances = maxInt(prodigy.Appearances, p.Appearances)
					prodigy.Goals = maxInt(prodigy.Goals, p.Goals)
					prodigy.Assists = maxInt(prodigy.Assists, p.Assists)
					// extra duplicate is eliminated
				} else {
					newSquad = append(newSquad, p)
				}
			}
			c.Squad = newSquad
			c.SquadSize = len(c.Squad)
		}
```

#### With This:
```go
		// Clean up any lingering duplicate copies of this wonderkid across all squads.
		// Ensures exactly one canonical instance exists in the home club even under pointer aliasing.
		for _, c := range dm.ClubsList {
			newSquad := make([]*models.Player, 0, len(c.Squad))
			keptProdigyInClub := false
			for _, p := range c.Squad {
				isWonderkidMatch := p == prodigy || strings.EqualFold(strings.TrimSpace(p.FullName), cfg.FullName)
				if !isWonderkidMatch {
					newSquad = append(newSquad, p)
					continue
				}

				if c.ClubID == cid && !keptProdigyInClub {
					newSquad = append(newSquad, prodigy)
					keptProdigyInClub = true
				} else {
					if p != prodigy {
						prodigy.Appearances = maxInt(prodigy.Appearances, p.Appearances)
						prodigy.Goals = maxInt(prodigy.Goals, p.Goals)
						prodigy.Assists = maxInt(prodigy.Assists, p.Assists)
					}
					// duplicate is eliminated
				}
			}
			c.Squad = newSquad
			c.SquadSize = len(c.Squad)
		}
```

---

## 5. Verification Method

### Step 1: Reproduction Verification (Current Failing State)
```powershell
cd c:\Users\Izyan\Downloads\General\football_sim\backend_go
go test -v ./pkg/datamanager -run "TestChallenger_DuplicateInjection_(Identical|CrossClub)"
```
Observed result: `FAIL` (2 failing tests).

### Step 2: Post-Fix Target Verification
After Worker applies the changes above, run:
```powershell
cd c:\Users\Izyan\Downloads\General\football_sim\backend_go
go test -v ./pkg/datamanager -run "TestChallenger_DuplicateInjection_(Identical|CrossClub)"
```
Expected result:
```
=== RUN   TestChallenger_DuplicateInjection_IdenticalPointerAttack
    challenger_stress_test.go:365: Identical pointer duplicate test: removed=1, remaining count=1
--- PASS: TestChallenger_DuplicateInjection_IdenticalPointerAttack (0.02s)
=== RUN   TestChallenger_DuplicateInjection_CrossClubSharedPointerAttack
    challenger_stress_test.go:415: Cross-club shared pointer test: removed=1, arsHas=true, cheHas=false
--- PASS: TestChallenger_DuplicateInjection_CrossClubSharedPointerAttack (0.02s)
PASS
ok  	football_sim/pkg/datamanager	0.985s
```

### Step 3: Full Workspace Regression Test
```powershell
cd c:\Users\Izyan\Downloads\General\football_sim\backend_go
go test -v ./...
```
Expected result:
- `football_sim/pkg/models`: PASS (32/32 tests pass)
- `football_sim/pkg/growth`: PASS (22/22 tests pass)
- `football_sim/pkg/datamanager`: PASS (22/22 tests pass)
- Total: 76/76 tests pass with 0 compiler warnings, 0 runtime panics, and 100% exit code 0.
