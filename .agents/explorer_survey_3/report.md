# Comprehensive Survey Report: R4 Transfers, Warchests & Build Pipeline

**Author**: Survey Explorer 3 (R4 Transfers & Build Pipeline Specialist)  
**Date**: 2026-09-09T16:15:00Z  
**Task Scope**: R4 (12-Week Off-Season Transfer Window, Club Warchests, Single-Transfer Lock, Wonderkid Loan Returns) & Build/Test Pipelines  
**Working Directory**: `c:\Users\Izyan\General\football_sim\.agents\explorer_survey_3`  

---

## 1. Executive Summary

This report delivers an exhaustive investigation of the existing football simulation codebase against Requirement **R4** of the Overhaul Specification (`ORIGINAL_REQUEST.md`) and surveys the build and verification pipelines across `backend_go/` and `frontend/`.

### Key Findings Summary:
1. **Transfer Window Progression**:
   - Currently, the off-season transfer window advances on daily ticks via `CurrentDay int` (incrementing unboundedly) rather than 12 weekly stages.
   - Frontend displays `Window open · day ${data.window_day}` with button `Advance One Day`.
   - **Required**: Restructure the off-season window into 12 distinct weekly stages (Weeks 1 to 12) advancing week-by-week (`CurrentWeek` 1..12), with Week 12 acting as deadline week, while retaining `window_day` in the JSON payload for backward compatibility.
2. **Club Warchests / Budgets**:
   - Currently, manager budgets are computed in `managers.BuildManagers()` as `int64(60_000_000 + (rating-78)*12_000_000)`, which tops out around €180M for Real Madrid and €60M for 78-rated clubs.
   - **Required**: Explicit, persistent club warchests initialized between **€50M and €250M** based on club stature. The 12 Super League clubs range from rating 81 (Tottenham) to 90 (PSG).
   - Currently, `executeTransfer` deducts from buyer budget and adds to seller budget, but `aiInitiateBid` does **not** check the buyer's available budget, nor do counter-offers or hijacks verify solvency. Club budgets can be driven negative by runaway bids.
   - **Required**: Strict solvency validation preventing any AI bid, counter-offer, or hijack from exceeding available warchest, guaranteeing zero negative balances.
3. **Single-Transfer Lock**:
   - Currently, `aiInitiateBid()` and `TriggerSpecificBid()` only check if a player is currently in `ActiveNegotiations`. If a player completes a transfer in Week 2, they can be targeted and transferred again in Week 6 of the same window.
   - **Required**: Strict single-transfer rule: once a player completes a transfer in a window, they are locked (`TransferredThisWindow[playerID] = true`) and cannot be transferred again in that same window.
4. **Wonderkid Loan Return Rules**:
   - Currently, in `backend_go/pkg/tournament/season.go` (`ResetNewSeason`), homecoming blindly resets **every** player whose `OriginalClubID != ClubID` back to `OriginalClubID`. Because `executeTransfer` does not update `OriginalClubID` for regular players, permanent transfers of normal players were erroneously reverted on season reset!
   - **Required**:
     - Wonderkids (`WK_` prefix or `UniverseWonderkid == true`) are restricted to transferring strictly between the 12 Super League clubs.
     - At season reset (`ResetNewSeason`), **only wonderkids** return to their canonical parent clubs (`OriginalClubID` / `ProdigyHomes`), functioning as seasonal development loans.
     - Permanent transfers of non-wonderkids update `OriginalClubID = BuyerID` and remain with their new clubs across seasons.
5. **Build & Test Pipeline Status**:
   - Backend Go pipeline (`cd backend_go && go test -count=1 ./...`): **100% PASS** across all 10 packages (`pkg/datamanager`, `pkg/growth`, `pkg/managers`, `pkg/matchengine`, `pkg/matchreport`, `pkg/models`, `pkg/persistence`, `pkg/server`, `pkg/tournament`, `pkg/transfers`) in ~8.1s with 0 compiler warnings, 0 runtime errors, and 0 panics.
   - Frontend build pipeline (`cd frontend && bun run build`): **100% PASS** via `tsc && vite build` in 4.05s, 0 TypeScript errors.
   - Note: Frontend is Bun-only (`bun install`, `bun run build`, `bun run dev`) as documented in `scripts/run.ps1`.

---

## 2. Detailed Technical Investigation of R4

### 2.1 12-Week Off-Season Transfer Window Progression

#### Current Implementation:
- In `backend_go/pkg/transfers/transfers.go`:
  - Lines 79-80: `CurrentMatchweek int`, `CurrentDay int`.
  - Lines 213-239: `AdvanceOpenWindow()` calls `updateDailyMarketUnlocked(true)` which executes:
    ```go
    for _, neg := range te.ActiveNegotiations {
        resolved := te.progressNegotiation(neg)
        ...
    }
    te.CurrentDay++
    ```
- In `backend_go/pkg/server/server.go`:
  - Line 2226: `careerWindowName(open bool)` returns `"Summer Window (Open)"` or `"Window Closed (Opens at season end)"`.
  - Line 2343: `transfersPayload()` returns `"window_day": s.TransferEngine.CurrentDay`.
  - Lines 2397-2425: `handleTransferAdvance()` calls `s.TransferEngine.AdvanceOpenWindow()`.
- In `frontend/src/components/TransfersTab.tsx`:
  - Line 98: `kicker={windowOpen ? 'Window open · day ${data.window_day}' : 'Window closed'}`
  - Line 109: `<PrimaryButton tone="cyan" onClick={handleAdvance} disabled={advancing}><Zap size={15} /> {advancing ? 'Advancing…' : 'Advance One Day'}</PrimaryButton>`
  - Line 65: `onShowToast('Window day ${res.window_day}. Negotiations and wire updated.')`
  - Line 173: `<Badge tone="gold">Day {data.window_day}</Badge>`

#### Required Changes for 12 Weekly Stages:
1. In `TransferEngine`:
   - Add `CurrentWeek int` (initialized to 1, max 12).
   - In `AdvanceOpenWindow()`:
     - Check `if te.CurrentWeek >= 12` -> window has reached its deadline conclusion.
     - Increment `te.CurrentWeek++`.
     - Keep `te.CurrentDay = te.CurrentWeek` to preserve compatibility with existing persistence and tests (`saved.Transfers.CurrentDay`).
     - Set TransferFeed timestamp to `fmt.Sprintf("Week %d", te.CurrentWeek)` during the off-season window.
2. In `server.go`:
   - Update `careerWindowName(open bool, week int) string`:
     - If open and `week < 12`: `fmt.Sprintf("Summer Window (Week %d of 12)", week)`
     - If open and `week >= 12`: `"Summer Window (Week 12 of 12 · Deadline Week)"`
     - If closed: `"Window Closed (Opens at season end)"` (matches existing test in `transfer_api_test.go:63`).
   - In `transfersPayload()`:
     - Return `"window_week": s.TransferEngine.CurrentWeek`
     - Return `"max_window_weeks": 12`
     - Keep `"window_day": s.TransferEngine.CurrentWeek`
   - In `handleTransferAdvance()`:
     - If `s.TransferEngine.CurrentWeek >= 12`, respond with 400 Bad Request: `"The 12-week transfer window has concluded. Please proceed to the new season."`
3. In `frontend/src/services/api.ts` & `frontend/src/types/index.ts`:
   - Add `window_week?: number;` and `max_window_weeks?: number;` to `TransfersResponse`.
4. In `frontend/src/components/TransfersTab.tsx`:
   - Update button label to `'Advance One Week'`.
   - Update kicker to: `windowOpen ? 'Window open · Week ${data.window_week || data.window_day} of 12' : 'Window closed'`.
   - Update toast to: `'Week ${res.window_week || res.window_day} of 12. Negotiations and wire updated.'`.
   - Update badge to: `'Week ${data.window_week || data.window_day} of 12'`.

---

### 2.2 Club Warchests (€50M to €250M) and Solvency Protection

#### Current Implementation:
- In `backend_go/pkg/managers/managers.go`:
  - Lines 273-276:
    ```go
    budget := int64(60_000_000)
    if diff := rating - 78; diff > 0 {
        budget += int64(diff * 12_000_000)
    }
    ```
  - In `transfers.ResetForNewSeason()`:
    ```go
    rating := club.OverallTeamRating - 78
    if rating < 0 { rating = 0 }
    manager.BudgetEur = int64(60_000_000 + rating*12_000_000)
    ```
- Elite Club ratings in `dataset.json` (ratings range 81 to 90):
  | Club ID | Club Name | Rating | Current Budget | Target Stature Warchest |
  |---|---|---|---|---|
  | `FL1-PSG` | Paris Saint-Germain | 90 | €204M | **€250M** |
  | `LAL-RMA` | Real Madrid | 89 | €192M | **€240M** |
  | `BUN-BAY` | Bayern Munich | 89 | €192M | **€220M** |
  | `LAL-BAR` | FC Barcelona | 88 | €180M | **€200M** |
  | `EPL-ARS` | Arsenal | 87 | €168M | **€180M** |
  | `SEA-INT` | Inter Milan | 86 | €156M | **€160M** |
  | `EPL-LIV` | Liverpool | 85 | €144M | **€150M** |
  | `LAL-ATM` | Atlético Madrid | 84 | €132M | **€120M** |
  | `BUN-DOR` | Borussia Dortmund | 84 | €132M | **€110M** |
  | `SEA-MIL` | AC Milan | 83 | €120M | **€90M** |
  | `SEA-NAP` | Napoli | 82 | €108M | **€70M** |
  | `EPL-TOT` | Tottenham Hotspur | 81 | €96M | **€50M** |

#### Stature Warchest Function:
A canonical function `CalculateClubWarchest(club *models.Club) int64` or lookup map guarantees all 12 clubs receive budgets strictly within `[€50M, €250M]`:
```go
var StatureWarchests = map[string]int64{
	"FL1-PSG": 250_000_000,
	"LAL-RMA": 240_000_000,
	"BUN-BAY": 220_000_000,
	"LAL-BAR": 200_000_000,
	"EPL-ARS": 180_000_000,
	"SEA-INT": 160_000_000,
	"EPL-LIV": 150_000_000,
	"LAL-ATM": 120_000_000,
	"BUN-DOR": 110_000_000,
	"SEA-MIL":  90_000_000,
	"SEA-NAP":  70_000_000,
	"EPL-TOT":  50_000_000,
}

func InitialWarchest(clubID string, rating int) int64 {
	if w, ok := StatureWarchests[clubID]; ok {
		return w
	}
	// Dynamic formula fallback clamped strictly in [50M, 250M]
	if rating < 81 { rating = 81 }
	if rating > 90 { rating = 90 }
	val := int64(50_000_000) + int64(rating-81)*int64(22_222_222)
	if val < 50_000_000 { val = 50_000_000 }
	if val > 250_000_000 { val = 250_000_000 }
	return val
}
```

#### Solvency Invariant & Negative Balance Prevention:
1. **In `aiInitiateBid()`**:
   - Retrieve `buyerMgr := te.Managers[buyer.ClubID]`.
   - Calculate available balance: `buyerMgr.BudgetEur`.
   - Filter `validTargets` to only those where `initialBid <= buyerMgr.BudgetEur`.
   - If `buyerMgr.BudgetEur < 5_000_000` or no valid targets can be afforded, skip bid initiation for this buyer.
2. **In `progressNegotiation()`**:
   - **Counter-Offer Stage (Stage 2)**:
     If `neg.AskingPrice > buyerMgr.BudgetEur`, the buyer cannot afford the asking price. Set `neg.StageName = "COLLAPSED"` and return `true` (terminate negotiation).
   - **Hijack Stage (Stage 3)**:
     Only select rivals where `rivalMgr.BudgetEur >= hijackedBid`. If no rival can afford it, no hijack occurs.
3. **In `executeTransfer()`**:
   - Assert `buyerMgr.BudgetEur >= neg.CurrentBid`. If insufficient, mark `COLLAPSED` without mutating squads.
   - Deduct `buyerMgr.BudgetEur -= neg.CurrentBid`.
   - Credit `sellerMgr.BudgetEur += neg.CurrentBid`.
   - Guarantee `buyerMgr.BudgetEur >= 0` at all times.

---

### 2.3 Single-Transfer Lock per Window

#### Problem in Current Code:
In `backend_go/pkg/transfers/transfers.go`, there is no tracking of whether a player was already transferred in the current window.
Line 404 only checks `alreadyNeg`:
```go
alreadyNeg := false
for _, n := range te.ActiveNegotiations {
    if n.Player.PlayerID == p.PlayerID {
        alreadyNeg = true; break
    }
}
```
If a player transfers in Week 2, their transfer completes and is removed from `ActiveNegotiations`. In Week 5, another club can bid on them and transfer them again within the same off-season window!

#### Required Implementation:
1. In `TransferEngine`:
   - Add field `TransferredThisWindow map[string]bool`.
   - Initialize in `NewTransferEngine()` and clear in `ResetForNewSeason()`.
2. In `executeTransfer(neg)`:
   - Record `te.TransferredThisWindow[neg.Player.PlayerID] = true`.
3. In `aiInitiateBid()`:
   - In the target selection loop:
     ```go
     if te.TransferredThisWindow[p.PlayerID] {
         continue
     }
     ```
4. In `TriggerSpecificBid()` and `InitiateBid()`:
   - Check if `te.TransferredThisWindow[playerID]` is true. If true, return `nil` / reject bid.
5. In Persistence:
   - Add `TransferredThisWindow []string` to `TransfersSnapshot` so locks survive server restarts and browser reloads during the transfer window.

---

### 2.4 Wonderkid Super League Restriction & Loan Returns

#### Current Implementation & Gaps:
1. **Super League Club Restriction**:
   - The 12 canonical wonderkids (`WK_` IDs, `UniverseWonderkid == true`) must ONLY transfer between the 12 European Super League clubs (`LAL-BAR`, `LAL-RMA`, `LAL-ATM`, `EPL-ARS`, `EPL-LIV`, `BUN-BAY`, `BUN-DOR`, `SEA-INT`, `SEA-NAP`, `SEA-MIL`, `FL1-PSG`, `EPL-TOT`).
   - In `InitiateBid` and `TriggerSpecificBid`: If target is a wonderkid, enforce `isSuperLeagueClub(buyer.ClubID) && isSuperLeagueClub(seller.ClubID)`.
2. **Loan Return Rule**:
   - Currently, in `season.go:217-233`:
     ```go
     // Homecoming
     for _, club := range tm.ClubsList {
         for _, p := range append([]*models.Player(nil), club.Squad...) {
             homeID := p.OriginalClubID
             if homeID == "" {
                 homeID = p.ClubID
             }
             home := tm.Clubs[homeID]
             if home == nil || home == club {
                 continue
             }
             removeFromSquad(club, p)
             p.ClubID = home.ClubID
             if !inSquad(home, p) {
                 home.Squad = append(home.Squad, p)
             }
         }
     }
     ```
   - **Bug in Current Code**: This homecoming runs on **ALL** players. If an 88-rated veteran (e.g., David Raya or a star striker) transfers from Arsenal to Bayern for €80M, `p.OriginalClubID` remains `EPL-ARS` because `executeTransfer` never updated it. At season reset, David Raya was being warped back to Arsenal!
   - **Solution**:
     1. In `executeTransfer`:
        ```go
        if !neg.Player.UniverseWonderkid && !strings.HasPrefix(neg.Player.PlayerID, "WK_") {
            neg.Player.OriginalClubID = neg.Buyer.ClubID
        }
        ```
     2. In `ResetNewSeason()` (`season.go`):
        Only return players if they are wonderkids:
        ```go
        for _, club := range tm.ClubsList {
            for _, p := range append([]*models.Player(nil), club.Squad...) {
                if !p.UniverseWonderkid && !strings.HasPrefix(p.PlayerID, "WK_") {
                    continue
                }
                homeID := p.OriginalClubID
                if homeID == "" && tm.ProdigyHomes != nil {
                    homeID = tm.ProdigyHomes[p.FullName]
                }
                if homeID == "" {
                    homeID = datamanager.DefaultProdigyHomes()[p.FullName]
                }
                home := tm.Clubs[homeID]
                if home == nil || home == club {
                    continue
                }
                removeFromSquad(club, p)
                p.ClubID = home.ClubID
                if !inSquad(home, p) {
                    home.Squad = append(home.Squad, p)
                }
            }
        }
        ```
     This strictly guarantees:
     - All 12 canonical wonderkids return to their original home clubs upon season reset.
     - Permanent signings of non-wonderkids remain with their purchasing clubs across seasons.

---

### 2.5 Persistence Model Audit (`saves/career.json`)

#### Current Fields in `TransfersSnapshot` (`backend_go/pkg/persistence/persistence.go`):
```go
type TransfersSnapshot struct {
    CurrentDay         int                              `json:"current_day"`
    CurrentMatchweek   int                              `json:"current_matchweek"`
    Feed               []transfers.TransferFeedItem     `json:"feed"`
    Completed          []transfers.CompletedTransfer    `json:"completed"`
    AllTime            []transfers.CompletedTransfer    `json:"all_time"`
    ActiveNegotiations []*transfers.TransferNegotiation `json:"active_negotiations,omitempty"`
    ManagerBudgets     map[string]int64                 `json:"manager_budgets,omitempty"`
}
```

#### Proposed Extensions to `TransfersSnapshot`:
```go
type TransfersSnapshot struct {
    CurrentDay            int                              `json:"current_day"`
    CurrentWeek           int                              `json:"current_week,omitempty"`
    CurrentMatchweek      int                              `json:"current_matchweek"`
    Feed                  []transfers.TransferFeedItem     `json:"feed"`
    Completed             []transfers.CompletedTransfer    `json:"completed"`
    AllTime               []transfers.CompletedTransfer    `json:"all_time"`
    ActiveNegotiations    []*transfers.TransferNegotiation `json:"active_negotiations,omitempty"`
    ManagerBudgets        map[string]int64                 `json:"manager_budgets,omitempty"`
    TransferredThisWindow []string                         `json:"transferred_this_window,omitempty"`
}
```
- In `BuildSnapshot`:
  - `snap.Transfers.CurrentWeek = te.CurrentWeek`
  - `snap.Transfers.TransferredThisWindow = te.GetTransferredPlayerIDs()`
- In `RestoreCareer`:
  - `if snap.Transfers.CurrentWeek > 0 { te.CurrentWeek = snap.Transfers.CurrentWeek } else if snap.Transfers.CurrentDay > 0 { te.CurrentWeek = snap.Transfers.CurrentDay }`
  - Populate `te.TransferredThisWindow` from `snap.Transfers.TransferredThisWindow`.
- Backward Compatibility: Fully preserved. Existing saves with `current_day` populate `CurrentWeek` seamlessly.

---

## 3. Build & Test Pipeline Audit

### 3.1 Backend Go Test Pipeline
- Root: `backend_go/`
- Tool: Go 1.22+ (`go test ./...`)
- Verified execution:
  ```powershell
  cd backend_go
  go test -count=1 ./...
  ```
  Result:
  - `football_sim/cmd/server`: [no test files]
  - `football_sim/pkg/datamanager`: PASS (4.70s)
  - `football_sim/pkg/growth`: PASS (0.96s)
  - `football_sim/pkg/managers`: PASS (0.90s)
  - `football_sim/pkg/matchengine`: PASS (1.34s)
  - `football_sim/pkg/matchreport`: PASS (1.89s)
  - `football_sim/pkg/models`: PASS (0.97s)
  - `football_sim/pkg/persistence`: PASS (3.55s)
  - `football_sim/pkg/server`: PASS (8.14s)
  - `football_sim/pkg/tournament`: PASS (1.66s)
  - `football_sim/pkg/transfers`: PASS (0.84s)
  Total execution time: ~8.14s. 100% test pass rate with 0 panics, 0 compiler warnings, 0 runtime failures.

### 3.2 Frontend Build Pipeline
- Root: `frontend/`
- Tool: Bun runtime (`bun run build` executing `tsc && vite build`)
- Package Configuration:
  - Vite 5.4.3
  - TypeScript 5.5.4
  - React 18.3.1
  - Tailwind CSS 3.4.10
- Verified execution:
  ```powershell
  cd frontend
  bun run build
  ```
  Result:
  - `tsc` completed with 0 errors.
  - `vite build` completed in 4.05s:
    - `dist/index.html`: 1.47 kB (gzip: 0.76 kB)
    - `dist/assets/index-529Y6hnC.css`: 42.89 kB (gzip: 8.44 kB)
    - `dist/assets/index-pIaKqd9n.js`: 449.68 kB (gzip: 118.25 kB)
  - 100% clean production bundle generated in `frontend/dist/`.

---

## 4. Proposed Code Modifications (File-by-File)

### 4.1 `backend_go/pkg/managers/managers.go`
Update `BuildManagers` to initialize club stature warchests in `[€50M, €250M]`:
```go
// StatureWarchests defines canonical transfer warchests for Super League clubs.
var StatureWarchests = map[string]int64{
	"FL1-PSG": 250_000_000,
	"LAL-RMA": 240_000_000,
	"BUN-BAY": 220_000_000,
	"LAL-BAR": 200_000_000,
	"EPL-ARS": 180_000_000,
	"SEA-INT": 160_000_000,
	"EPL-LIV": 150_000_000,
	"LAL-ATM": 120_000_000,
	"BUN-DOR": 110_000_000,
	"SEA-MIL":  90_000_000,
	"SEA-NAP":  70_000_000,
	"EPL-TOT":  50_000_000,
}

func CalculateClubWarchest(clubID string, rating int) int64 {
	if w, ok := StatureWarchests[clubID]; ok {
		return w
	}
	if rating < 81 { rating = 81 }
	if rating > 90 { rating = 90 }
	w := int64(50_000_000) + int64(rating-81)*int64(22_222_222)
	if w < 50_000_000 { w = 50_000_000 }
	if w > 250_000_000 { w = 250_000_000 }
	return w
}
```

### 4.2 `backend_go/pkg/transfers/transfers.go`
1. Add `CurrentWeek int` and `TransferredThisWindow map[string]bool`.
2. Add Super League club check:
```go
var SuperLeagueClubs = map[string]bool{
	"LAL-BAR": true, "LAL-RMA": true, "LAL-ATM": true,
	"EPL-ARS": true, "EPL-LIV": true, "EPL-TOT": true,
	"BUN-BAY": true, "BUN-DOR": true,
	"SEA-INT": true, "SEA-NAP": true, "SEA-MIL": true,
	"FL1-PSG": true,
}
```
3. Update `AdvanceOpenWindow()`:
   - Advance `CurrentWeek++` (up to 12).
   - Advance negotiations.
   - Restrict AI bids to solvent buyers and unlocked players.
4. Update `executeTransfer()`:
   - Lock player: `te.TransferredThisWindow[neg.Player.PlayerID] = true`.
   - Update `neg.Player.OriginalClubID = neg.Buyer.ClubID` for non-wonderkids.
   - Deduct/credit warchests with `< 0` check.
5. In `ResetForNewSeason()`:
   - Reset `CurrentWeek = 1`, `CurrentDay = 1`.
   - Clear `TransferredThisWindow`.
   - Re-initialize manager warchests via `CalculateClubWarchest`.

### 4.3 `backend_go/pkg/tournament/season.go`
In `ResetNewSeason()`:
- Only return wonderkids (`p.UniverseWonderkid || strings.HasPrefix(p.PlayerID, "WK_")`) to their canonical parent clubs. Non-wonderkids remain with their purchased clubs.

### 4.4 `backend_go/pkg/persistence/persistence.go`
- Persist and restore `CurrentWeek` and `TransferredThisWindow` in `TransfersSnapshot`.

### 4.5 `backend_go/pkg/server/server.go`
- Update `careerWindowName(open bool, week int)`: Return `"Summer Window (Week %d of 12)"`.
- In `transfersPayload()`: Return `"window_week"`, `"max_window_weeks": 12`, and `"window_day"`.
- In `handleTransferAdvance()`: Return 400 when `CurrentWeek >= 12`.

### 4.6 `frontend/src/components/TransfersTab.tsx`
- Replace "Advance One Day" with "Advance One Week".
- Display `Week ${data.window_week || data.window_day} of 12`.

---

## 5. Proposed Test Plan

Add comprehensive unit and integration tests in `backend_go/pkg/transfers/transfers_test.go` and `backend_go/pkg/server/transfer_api_test.go`:

1. `TestTransferEngine_12WeekWindowProgression`:
   - Initialize transfer engine with 12 clubs.
   - Advance window 11 times; verify `CurrentWeek` increments 1 through 12.
   - Verify advancing past week 12 is bounded / reported as closed.
2. `TestTransferEngine_WarchestInitialClamping`:
   - Verify all 12 Super League clubs start with budgets strictly between €50,000,000 and €250,000,000.
3. `TestTransferEngine_SolvencyAndNoNegativeBalances`:
   - Artificially reduce a club's warchest to €1M.
   - Verify AI bids cannot be initiated by this club for any player > €1M.
   - Force a negotiation to counter-offer above budget; verify negotiation collapses.
   - Force execution; verify warchest cannot become negative.
4. `TestTransferEngine_SingleTransferLockPerWindow`:
   - Complete a transfer for player P1 from Club A to Club B.
   - Attempt a new bid on P1 within the same window; verify `TriggerSpecificBid` returns nil and AI skips P1.
   - Call `ResetForNewSeason()`; verify lock clears for the next season's window.
5. `TestTransferEngine_WonderkidTransferRestrictions`:
   - Attempt to bid for a wonderkid using a non-Super League club ID; verify rejection.
   - Complete a wonderkid transfer between Super League clubs.
   - Trigger `ResetNewSeason()`; verify the wonderkid returns to their original canonical parent club.
   - Complete a non-wonderkid transfer between Super League clubs.
   - Trigger `ResetNewSeason()`; verify the non-wonderkid remains at the new club.
6. `TestTransfers_SaveLoadContinuity`:
   - Advance transfer window to Week 4 with completed transfers and modified warchests.
   - Save to disk via `SaveCareer`.
   - Restore via `RestoreCareer`; assert `CurrentWeek == 4`, warchests match exactly, and locks persist.
