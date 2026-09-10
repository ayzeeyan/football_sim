# Handoff Report: Survey Explorer 3 (R4 Transfers & Build Pipeline)

**Author**: Survey Explorer 3  
**Date**: 2026-09-09T16:15:00Z  
**Type**: Hard Handoff  
**Working Directory**: `c:\Users\Izyan\General\football_sim\.agents\explorer_survey_3`  
**Reference Report**: `c:\Users\Izyan\General\football_sim\.agents\explorer_survey_3\report.md`  

---

## 1. Observation

### 1.1 Codebase Structure & File Paths
- Transfer engine core: `c:\Users\Izyan\General\football_sim\backend_go\pkg\transfers\transfers.go` (650 lines)
- Transfer tests: `c:\Users\Izyan\General\football_sim\backend_go\pkg\transfers\transfers_test.go` (145 lines)
- Manager profiles & warchests: `c:\Users\Izyan\General\football_sim\backend_go\pkg\managers\managers.go` (388 lines)
- Tournament & season reset: `c:\Users\Izyan\General\football_sim\backend_go\pkg\tournament\season.go` (529 lines)
- Career persistence & snapshots: `c:\Users\Izyan\General\football_sim\backend_go\pkg\persistence\persistence.go` (854 lines)
- HTTP REST transfer handlers: `c:\Users\Izyan\General\football_sim\backend_go\pkg\server\server.go` (lines 2220–2435)
- Server transfer API tests: `c:\Users\Izyan\General\football_sim\backend_go\pkg\server\transfer_api_test.go` (181 lines)
- Frontend transfers tab: `c:\Users\Izyan\General\football_sim\frontend\src\components\TransfersTab.tsx` (290 lines)
- Frontend API contracts: `c:\Users\Izyan\General\football_sim\frontend\src\services\api.ts` (lines 415–480)

### 1.2 Verbatim Code & Current Deficiencies
1. **Daily Ticks vs 12 Weekly Stages**:
   - In `backend_go/pkg/transfers/transfers.go:213-239`:
     ```go
     func (te *TransferEngine) AdvanceOpenWindow() {
         te.mu.Lock()
         defer te.mu.Unlock()
         te.updateDailyMarketUnlocked(true)
     }
     ...
     te.CurrentDay++
     ```
   - In `frontend/src/components/TransfersTab.tsx:109`:
     ```tsx
     <PrimaryButton tone="cyan" onClick={handleAdvance} disabled={advancing}>
       <Zap size={15} aria-hidden="true" /> {advancing ? 'Advancing…' : 'Advance One Day'}
     </PrimaryButton>
     ```
   - Observation: Window advances unbounded daily ticks without a 12-week stage lifecycle.

2. **Warchest Limits & Solvency Gap**:
   - In `backend_go/pkg/managers/managers.go:273-276`:
     ```go
     budget := int64(60_000_000)
     if diff := rating - 78; diff > 0 {
         budget += int64(diff * 12_000_000)
     }
     ```
   - In `dataset.json`: Elite clubs have overall ratings between 81 (Tottenham) and 90 (PSG). With the current formula, maximum budget is €204M, below the requested €250M ceiling, and Tottenham starts with €96M rather than €50M.
   - In `backend_go/pkg/transfers/transfers.go:381-448` (`aiInitiateBid`):
     Target selection does **not** check `buyerMgr.BudgetEur >= initialBid`. AI clubs can make bids far exceeding their balances, driving warchests into negative balances on transfer execution.

3. **Absence of Single-Transfer Lock**:
   - In `backend_go/pkg/transfers/transfers.go:400-413`:
     ```go
     for _, n := range te.ActiveNegotiations {
         if n.Player.PlayerID == p.PlayerID {
             alreadyNeg = true
             break
         }
     }
     ```
   - Observation: Only active talks are checked. Once completed, a player can immediately be targeted again in the same window.

4. **Homecoming Reset Flaw (Wonderkid vs Veteran)**:
   - In `backend_go/pkg/tournament/season.go:216-233`:
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
   - In `backend_go/pkg/transfers/transfers.go:338-340`:
     ```go
     // 2. Add player to buyer squad
     neg.Player.ClubID = neg.Buyer.ClubID
     neg.Buyer.Squad = append(neg.Buyer.Squad, neg.Player)
     ```
   - Observation: `executeTransfer` never changes `neg.Player.OriginalClubID` for non-wonderkid players. On `ResetNewSeason`, non-wonderkids who were permanently bought are wrongly stripped from the buyer and returned to their old club.

### 1.3 Verbatim Tool Command Results
1. Backend test suite:
   - Command: `go test -count=1 ./...` in `c:\Users\Izyan\General\football_sim\backend_go`
   - Result:
     ```
     ok  	football_sim/pkg/datamanager	4.695s
     ok  	football_sim/pkg/growth	0.964s
     ok  	football_sim/pkg/managers	0.897s
     ok  	football_sim/pkg/matchengine	1.339s
     ok  	football_sim/pkg/matchreport	1.893s
     ok  	football_sim/pkg/models	0.970s
     ok  	football_sim/pkg/persistence	3.545s
     ok  	football_sim/pkg/server	8.139s
     ok  	football_sim/pkg/tournament	1.657s
     ok  	football_sim/pkg/transfers	0.839s
     ```
   - Status: Exit code 0, 100% pass across all 10 packages.
2. Frontend build:
   - Command: `bun run build` in `c:\Users\Izyan\General\football_sim\frontend`
   - Result:
     ```
     $ tsc && vite build
     vite v5.4.21 building for production...
     ✓ 1867 modules transformed.
     dist/index.html                   1.47 kB │ gzip:   0.76 kB
     dist/assets/index-529Y6hnC.css   42.89 kB │ gzip:   8.44 kB
     dist/assets/index-pIaKqd9n.js   449.68 kB │ gzip: 118.25 kB
     ✓ built in 4.05s
     ```
   - Status: Exit code 0, 0 TypeScript compilation errors.

---

## 2. Logic Chain

1. **Window Restructuring**:
   - Because the user requirement specifies "restructure the off-season transfer window into 12 weekly stages (Weeks 1 to 12) advancing week-by-week rather than daily ticks", `TransferEngine` must maintain `CurrentWeek int` (bounded in `[1, 12]`).
   - Each call to `AdvanceOpenWindow()` must increment `CurrentWeek` and progress market negotiations for that weekly stage.
   - To prevent regression with existing tests (`transfer_api_test.go:157` asserting `board["window_day"]`), the JSON payload must include `window_week`, `max_window_weeks: 12`, and keep `window_day: te.CurrentWeek`.

2. **Warchest Stature & Solvency Enforcement**:
   - The user requires warchests "initialized between €50M and €250M based on club stature" and that purchases "prevent AI clubs from making bids that exceed their available balance".
   - The 12 clubs span ratings 81 (Tottenham) to 90 (PSG). A canonical stature map or rating-scaling formula clamps Tottenham at €50M, PSG at €250M, Real Madrid at €240M, Bayern at €220M, Barcelona at €200M, etc.
   - By adding an available budget check in `aiInitiateBid` (`if buyerMgr.BudgetEur < initialBid { continue }`), checking solvency in counter-offers and hijacks, and asserting solvency before deducting in `executeTransfer`, warchests are strictly protected against negative balances.

3. **Single-Transfer Lock**:
   - The requirement requires "strict single-transfer rule per window: once a player completes a transfer in a window, they cannot be transferred again in that same window".
   - Storing completed transfers in `TransferredThisWindow map[string]bool` in `TransferEngine` and checking it before initiating AI or manual bids ensures 0 players transfer more than once per window.
   - The lock is cleared when `ResetForNewSeason()` is triggered.

4. **Wonderkid Loan Return Isolation**:
   - The requirement mandates that the 12 canonical wonderkids (`WK_` IDs) "only transfer between the 12 Super League clubs, and at the end of the season / start of the new campaign, automatically return them to their canonical original parent clubs".
   - In `season.go:217`, homecoming must be gated by `if !p.UniverseWonderkid && !strings.HasPrefix(p.PlayerID, "WK_") { continue }`.
   - In `transfers.go:338`, when a non-wonderkid transfers, set `neg.Player.OriginalClubID = neg.Buyer.ClubID`. This ensures normal permanent transfers remain permanent, while wonderkids return home on season reset.

---

## 3. Caveats

1. **Mid-Season Window vs Off-Season Window**:
   - In `transfers.go:135`, `IsWindowOpen()` contains legacy matchweek logic (`MW 1-4` and `MW 21-24`). In career mode, the off-season window is active when `tm.SeasonPhase == "transfer_window"`. Care should be taken so that off-season week ticks do not conflict with during-season matchweek simulation.
2. **Frontend UI Display**:
   - While `TransfersTab.tsx` currently displays days, changing it to weeks requires updating label strings ("Advance One Week") and badge text. The API payload changes are fully backwards compatible.
3. **No Caveats in Build Pipeline**:
   - Both `backend_go` tests and `frontend` build pass cleanly without any existing errors or warnings.

---

## 4. Conclusion

Requirement R4 is well-understood, clearly scoped, and ready for implementation by the builder/orchestrator team:
1. `backend_go/pkg/transfers/transfers.go`: Add `CurrentWeek` (1..12), `TransferredThisWindow` lock, solvency checks on bids/counter-offers/hijacks, and Super League wonderkid restrictions.
2. `backend_go/pkg/managers/managers.go`: Update warchest initialization to [€50M, €250M] based on club stature.
3. `backend_go/pkg/tournament/season.go`: Scope `Homecoming` strictly to canonical wonderkids (`UniverseWonderkid` / `WK_`).
4. `backend_go/pkg/persistence/persistence.go`: Persist `CurrentWeek` and `TransferredThisWindow`.
5. `backend_go/pkg/server/server.go`: Expose `window_week` (1..12) and `max_window_weeks: 12`, cap advance at Week 12.
6. `frontend/src/components/TransfersTab.tsx` & `api.ts`: Update display labels to weekly stages.

---

## 5. Verification Method

1. **Backend Tests**:
   ```powershell
   cd c:\Users\Izyan\General\football_sim\backend_go
   go test -v -count=1 ./pkg/transfers/... ./pkg/server/...
   go test -count=1 ./...
   ```
   *Expected*: All packages pass with 0 failures, 0 panics.

2. **Frontend Build**:
   ```powershell
   cd c:\Users\Izyan\General\football_sim\frontend
   bun run build
   ```
   *Expected*: `tsc && vite build` succeeds with 0 TypeScript compilation errors.

3. **Specific R4 Invariants to Verify**:
   - Super League warchests: `min(warchests) >= 50_000_000 && max(warchests) <= 250_000_000`.
   - Budget integrity: no club warchest `< 0` after 12 weeks of active market simulation.
   - Single-transfer lock: no player ID appears in more than 1 completed transfer in the same window.
   - Wonderkid loan return: any transferred wonderkid is back at `OriginalClubID` after `ResetNewSeason()`, while non-wonderkids remain with buyer.
   - Week progression: window advances exactly 1..12 and halts or reports conclusion at week 12.
