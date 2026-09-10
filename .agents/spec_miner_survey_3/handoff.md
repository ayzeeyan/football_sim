# Handoff Report: Data Ingestion, Deduplication, Canonical Wonderkids & Academy Intake

**Agent**: Spec Miner 3  
**Working Directory**: `c:\Users\Izyan\Downloads\General\football_sim\.agents\spec_miner_survey_3`  
**Milestone**: Chunk 1 Specification Mining & Survey  
**Target Package**: `backend_go/pkg/datamanager`  

---

## 1. Observation

### 1.1 Codebase Inspection Findings
- **Authoritative Requirements (`.agents/ORIGINAL_REQUEST.md:15-28, 34`)**:
  - Ingest `dataset.json` (96 clubs, 2,294 players) into native Go memory without data loss.
  - Exactly 0 duplicate players in any squad across clubs and within clubs.
  - All 12 franchise wonderkids start at age 14, in middle school, with canonical `WK_` IDs.
  - Wonderkid potentials match biometrics exactly (93–96 range, never 99).
  - Verification requires `cd backend_go && go test -v ./...` passing 100% with zero compiler warnings or runtime panics.
- **Dataset Schema (`dataset.json`)**:
  - Top-level keys: `["clubs", "dataset", "generated_at", "league_counts", "notes", "season", "totals", "wonderkid_universe"]`.
  - `season`: `"2026-27"`.
  - `totals`: `{"clubs": 96, "players": 2294, "wonderkids": 12}`.
  - `league_counts`: `{"Premier League": 20, "La Liga": 20, "Serie A": 20, "Bundesliga": 18, "Ligue 1": 18}` (Sum = 96 clubs).
  - Clubs array: 96 items. Every club contains 11 fields: `club_id` (str), `club_name` (str), `country` (str), `home_stadium` (str), `league` (str), `overall_team_rating` (int), `short_name` (str), `squad` (list), `squad_avg_ovr` (float), `squad_size` (int), `stadium_capacity` (int). Zero `null` or missing values across all 96 clubs.
  - Squad players: Exactly 2,294 players. Every player contains 9 fields: `player_id` (str), `full_name` (str), `position` (str), `ovr` (int), `age` (int), `estimated_market_value_eur` (int), `season_stats` (dict), `universe_wonderkid` (bool), `player_source` (str). Zero `null` or missing values across all 2,294 players.
  - `season_stats`: 5 fields: `appearances` (int), `as_of` (str), `assists` (int), `goals` (int), `season` (str).
  - `wonderkid_universe`: 12 items.
- **Python Implementation (`data_manager.py:15-616`)**:
  - `PREFERRED_HOMES` (lines 16–29): 12 transfer home mappings for names that may be duplicated.
  - `ELITE_PRODIGY_CONFIGS` (lines 33–166): 12 explicit wonderkid definitions, all with `age: 14`, potentials in range [93, 96], baseline OVRs in range [75, 78], and physical dimensions.
  - Deduplication (`data_manager.py:241–308`): Groups players globally by `p.full_name.strip().lower()`. Selects canonical copy using PREFERRED_HOMES, elite club heuristic, and `(ovr, appearances)` max fallback. Merges stats (`appearances`, `goals`, `assists`, `career_goals`, `career_assists`, `career_apps`) using `max()`. Removes non-canonical copies from squads.
  - Wonderkid Setup (`data_manager.py:309–415`): Assigns stable ID `"WK_" + name.replace(" ", "_")`. Relocates Jhed Anthony Guinita from `FL1-OM` to `EPL-TOT`. Forces `age = 14`, `education = "middle_school"`, `education_pending = False`, `category = "FWD"`. Registers in GrowthEngine with exact potentials (93–96). Recomputes OVR and composure. Snaps market valuations to `baseline_value(ovr, 14, True)` and clamps them.
  - Youth Intake (`data_manager.py:530–616`): Generates 2–4 academy graduates per club (age 16–18, 58–72 OVR or 72–78 golden generation with 20% probability), caps club squad at 34, registers with GrowthEngine (potential 75–92 standard, 90–95 golden gen), and assigns player ID `AC_{club_id}_{name}_{random4}`.
- **Current State of Go Backend (`backend_go`)**:
  - `backend_go/go.mod` declares `module football_sim` with `go 1.27.1`.
  - `backend_go/pkg/datamanager` exists but is an **empty directory** (0 files).
  - `backend_go/pkg/models` and `backend_go/pkg/growth` are also empty directories.
  - Running `cd backend_go && go test -v ./...` produced:
    ```
    go: warning: "./..." matched no packages
    no packages to test
    exit status 1
    ```

---

## 2. Logic Chain

1. **Data Ingestion Fidelity**:
   - Observation: `dataset.json` contains 96 clubs and 2,294 squad entries with zero nulls or missing fields.
   - Deduction: Direct deserialization into Go structs (`RawDataset`, `RawClub`, `RawPlayer`, `RawSeasonStats`) using `encoding/json` provides a lossless representation.
   - Transition to domain model: Ingested domain `Player` and `Club` structs in `pkg/models` must preserve all JSON fields, while initializing runtime fields (e.g. career ledger to 0, wages computed from OVR, initial domestic league table records, kit colors).

2. **Deduplication Invariant (0 Duplicates)**:
   - Observation: In raw `dataset.json`, there are 0 duplicate names or IDs currently. However, `data_manager.py` contains `dedupe_players` which handles modified datasets, transfer conflicts, legacy data, and youth intake collisions.
   - Deduction: Go's `DedupePlayers` must execute immediately after dataset load to guarantee the acceptance criterion: *"Exactly 0 duplicate players in any squad across clubs and within clubs"*.
   - Logic of canonical selection:
     1. Wonderkids: prefer copy matching assigned home club and `WK_` prefix or `universe_wonderkid = true`.
     2. Regular players: prefer club in `PREFERRED_HOMES` if present; else prefer elite club if exactly one copy is in an elite club; else pick copy with highest `(OVR, Appearances)`.
     3. Non-canonical copies must merge their appearances, goals, assists, and career totals into the canonical player via `max()`, and then be excised from their club squads.

3. **12 Canonical U-14 Wonderkids Configuration**:
   - Observation: In raw `dataset.json`, wonderkids are listed with ages 17–19, IDs `P00016`–`P02187`, and Jhed Anthony Guinita is listed under `FL1-OM`.
   - Deduction: Ingestion MUST run `_initialize_elite_prodigies` logic:
     - Relocate Jhed Anthony Guinita from `FL1-OM` to `EPL-TOT`.
     - Rewrite `PlayerID` to `WK_` + full name with underscores.
     - Reset `Age = 14`, `Education = "middle_school"`, `EducationPending = false`, `UniverseWonderkid = true`, `Category = "FWD"`, `SchoolWant = SchoolWantFor(name)`.
     - Register in `growth.GrowthEngine` with biometric profile (height, weight, adult_height_age) and exact potential (93–96).
     - Calculate OVR and Composure from attributes seeded in GrowthEngine.
     - Ensure potentials strictly adhere to: Venjamin Valerio (96), Maverick Cantalejo (95), Yeshua Emmanuel Gocotano (93), Izyan Levin Bantol (95), James Bernard Rizon (94), Reid Randell Libatan (95), Ashle Zylle Baguio (95), Cliergy Jave Lanticse (94), Ezail Zamora (96), Earl Josh Hernando (94), Rich Lorenz Suico (94), Jhed Anthony Guinita (94). None are 99.

4. **Academy Regen Youth Intake**:
   - Observation: Python uses 54 first names (`ACADEMY_FIRST`), 54 last names (`ACADEMY_LAST`), 11 positions (`ACADEMY_POS`), and 4 personalities (`REGEN_PERSONALITIES`).
   - Deduction: Go implementation must provide identical constant slices. `RunYouthIntake` must check squad size `< 34`, roll 20% golden generation chance, draw valid attributes, assign `AC_{club_id}_{name}_{random}` ID, clamp market value, and register in GrowthEngine.

5. **Missing Code in `backend_go/pkg/datamanager`**:
   - Observation: The directory is empty, causing `go test` to fail.
   - Deduction: Complete implementations of `datamanager.go`, `prodigies.go`, `youth_intake.go`, and `datamanager_test.go` are required to fulfill Requirement R2 and Chunk 1 completion.

---

## 3. Features Discovered

| # | Category | Feature | Description | Inputs | Outputs | Error Behavior | Discovered Via |
|---|----------|---------|-------------|--------|---------|----------------|----------------|
| 1 | Ingestion | `LoadDataset` | Reads `dataset.json`, deserializes 96 clubs and 2,294 players into Go memory | `jsonPath string` | Populates `Clubs`, `ClubsList`, `Leagues`, `Metadata` | Returns `os.ErrNotExist` / `error` if file missing or corrupt JSON | `data_manager.py:200-240`, `dataset.json` |
| 2 | Ingestion | Season Ledger Reset | On fresh career kickoff, resets all player match and career stats to 0 | Ingested `Player` list | Resets `Goals`, `Assists`, `Appearances`, `CareerGoals`, etc. to 0 | None (in-memory mutation) | `data_manager.py:224-240` |
| 3 | Deduplication | Global Squad Deduplication | Eliminates duplicate players across clubs and within the same squad; merges career stats via `max()` | In-memory clubs and squad lists | Returns count of removed duplicates; squads updated to 0 duplicates | None; safe fallback to highest `(OVR, appearances)` | `data_manager.py:241-308` |
| 4 | Deduplication | Preferred Transfer Homes | Preserves 2026-27 updated transfer destinations for 12 key European players | Lowercase player name | Target club ID (e.g. `khvicha kvaratskhelia` -> `FL1-PSG`) | None; falls back to elite club check or highest OVR | `data_manager.py:16-29` |
| 5 | Wonderkids | Elite Prodigy Registry | 12 canonical outfield franchise prodigies with explicit biometrics, starting at age 14 | Wonderkid config array (12 entries) | 12 initialized `Player` structs with `WK_` IDs in `Wonderkids` slice | Errors if club ID not found | `data_manager.py:33-166, 309-415` |
| 6 | Wonderkids | Jhed Anthony Guinita Relocation | Moves Jhed Anthony Guinita from Marseille (`FL1-OM`) to Tottenham Hotspur (`EPL-TOT`) | Squad search across clubs | Player moved to `EPL-TOT` squad[0], removed from `FL1-OM` | None; instantiates player if missing | `data_manager.py:327-338` |
| 7 | Wonderkids | Stable Wonderkid ID | Generates deterministic identifier `WK_{FullNameWithUnderscores}` | `full_name string` | `string` (e.g. `WK_Venjamin_Valerio`) | Replaces spaces with underscores | `data_manager.py:180-182` |
| 8 | Wonderkids | Middle School Education Binding | Sets education to `"middle_school"`, `EducationPending=false`, calculates deterministic `SchoolWant` | Wonderkid player struct | Fields set on player; governs availability during exam weeks | None | `data_manager.py:372-374`, `models.py:25-36` |
| 9 | Wonderkids | Biometric Growth Engine Registration | Enrolls wonderkid in `GrowthEngine` with adult height age, height, weight, baseline OVR, and potential | Player ID, biometrics, potential (93–96) | `BiometricProfile` and `TechnicalAttributes` created in GrowthEngine | Potential strictly clamped [93, 96], never 99 | `data_manager.py:377-387`, `growth.py:116-158` |
| 10 | Wonderkids | Market Baseline Snapping | Snaps all player valuations to age/OVR anchor using `BaselineValue` and `ClampValue` | Squad players | `MarketValueEur` recalculated and clamped [300k, 500M] | Valuations cannot be negative or runaway | `data_manager.py:417-423`, `models.py:202-234` |
| 11 | Wonderkids | Prodigy Draft/Draw Distribution | Distributes the 12 wonderkids across the 12 elite clubs (1 kid per club) | `homes map[string]string` | Updated squad memberships; wonderkid moved to index 0 | Falls back to `DefaultProdigyHomes()` if invalid | `data_manager.py:424-451` |
| 12 | Wonderkids | Adopt U-14 Baselines | Resets franchise wonderkids back to age 14, middle school status, and growth profiles | None | Boolean `changed` indicating whether modifications were made | Skips non-wonderkids | `data_manager.py:468-498` |
| 13 | Youth Intake | Academy Regen Generator | Generates 2–4 academy graduates per club with realistic positions, ages, and personalities | `clubs []*Club`, optional `count *int`, `ge *GrowthEngine` | Slice of newly generated `(Player, Club)` pairs | Skips club if `len(Squad) >= 34` | `data_manager.py:530-616` |
| 14 | Youth Intake | Golden Generation Intake | 20% chance per club gives 1 elite graduate with OVR 72–78 and potential 90–95 | Random probability `< 0.20` | Elite academy graduate registered in GrowthEngine | Standard grad: OVR 58–72, potential 75–92 | `data_manager.py:550, 571-577` |
| 15 | Youth Intake | Academy Name Pool Deduplication | Generates unique names from 54 first and 54 last names; prevents collisions | Name pool, `taken` set | Unique full name; appends number `10-99` if collision persists after 40 tries | Collision resolved via suffix | `data_manager.py:557-564` |
| 16 | Domain Query | Elite Clubs Filter | Filters and returns the 12 European Super League elite clubs | None | `[]*Club` containing the 12 elite clubs | Excludes clubs not in dataset | `data_manager.py:500-504` |

---

## 4. Edge Cases

| # | Feature | Input | Observed Behavior |
|---|---------|-------|-------------------|
| 1 | Ingestion | Missing or invalid `dataset.json` path | Returns descriptive `os.ErrNotExist` / `fmt.Errorf` without panicking. |
| 2 | Ingestion | Extra or unknown JSON fields | Go `encoding/json` ignores unknown fields by default; all required fields must be populated with zero missing data. |
| 3 | Deduplication | Player duplicated in 2 clubs with PREFERRED_HOMES match | Kept in the club listed in `PREFERRED_HOMES` (e.g. Zubimendi kept in `EPL-ARS`, removed from `EPL-CHE`). Stats merged via `max()`. |
| 4 | Deduplication | Player duplicated between 1 elite club and 1 non-elite club (not in PREFERRED_HOMES) | Kept in the elite club (`elite_hits[0]`). Stats merged via `max()`. Non-elite copy removed. |
| 5 | Deduplication | Player duplicated across 2 non-elite clubs (or 2 elite clubs) | Kept in club of player with higher `(ovr, appearances)`. If identical, first encountered is kept. |
| 6 | Deduplication | Player duplicated twice within the *same* club | Merges stats into one copy, removes second copy, decrements `club.squad_size` to match `len(squad)`. |
| 7 | Deduplication | Accented vs unaccented names (e.g. `Martín Zubimendi` vs `Martin Zubimendi`) | Handled via both entries registered in `PREFERRED_HOMES` and UTF-8 case folding (`strings.TrimSpace(strings.ToLower(name))`). |
| 8 | Wonderkids | Wonderkid already present in squad with standard `Pxxxxx` ID and age 18 in raw dataset | Converted in place: ID changed to `WK_...`, age snapped to 14, education set to `middle_school`, potential set to config value (93–96). |
| 9 | Wonderkids | Wonderkid listed at wrong club in raw JSON (Jhed Anthony Guinita at `FL1-OM`) | Relocated: excised from `FL1-OM`, inserted at index 0 of `EPL-TOT`. Both squad sizes updated. |
| 10 | Wonderkids | Potential bounds verification | Potentials range from 93 to 96 inclusive (Valerio 96, Zamora 96, Cantalejo 95, Bantol 95, Libatan 95, Baguio 95, Rizon 94, Lanticse 94, Hernando 94, Suico 94, Guinita 94, Gocotano 93). Never 99. |
| 11 | Wonderkids | CAM position classification | All franchise wonderkids, including CAMs (Cantalejo, Bantol, Baguio), have `Category = "FWD"` per specification. |
| 12 | Youth Intake | Club squad already has 34 players | Youth intake generates 0 players for that club; `len(squad)` never exceeds 34. |
| 13 | Youth Intake | Name pool collision after 40 random attempts | Suffix with two-digit random integer `f"{first} {last} {10..99}"` to guarantee global uniqueness. |
| 14 | Youth Intake | Golden generation roll on 0-length club list | Empty list returned; zero crashes or out-of-bounds panics. |
| 15 | Valuation | Wonderkid market value calculation at age 14 | Base formula `11M * (1.086^(OVR-65)) * 1.15 (age<=21) * 1.35 (wonderkid)`. For OVR 78: ~€49.9M. Clamped strictly between €300,000 and €500,000,000. |

---

## 5. Caveats

1. **Dependency Order for Implementation**:
   - `backend_go/pkg/datamanager` directly imports `backend_go/pkg/models` (for `Club`, `Player`, `BaselineValue`, `ClampValue`, `SchoolWantFor`, etc.) and `backend_go/pkg/growth` (for `GrowthEngine`).
   - Because `pkg/models` and `pkg/growth` are currently empty directories, compiling `pkg/datamanager` requires either stub/interface definitions or coordinated implementation alongside Explorer 1 (`pkg/models`) and Explorer 2 (`pkg/growth`).
2. **Squad Size Synchronization**:
   - In Python `data_manager.py:333-337`, when moving Jhed Anthony Guinita, `other_club.squad_size` was updated to 23, but `dest.squad_size` was not updated to 25 until `apply_prodigy_homes` was called. In Go, `SquadSize` must always equal `len(Squad)`.
3. **Random Seed Determinism in Tests**:
   - Tests validating youth intake and deduplication should support deterministic pseudo-random number generator seeding (`rand.New(rand.NewSource(seed))`) to avoid flaky test assertions.

---

## 6. Conclusion & Recommendations

The specifications for `backend_go/pkg/datamanager` are fully identified and verified against the reference Python implementation and `dataset.json`.

### Exact Go Code Structure Recommended:
```
backend_go/pkg/datamanager/
├── datamanager.go       # DataManager struct, LoadDataset, DedupePlayers, ResetMarketToBaseline
├── prodigies.go          # ELITE_PRODIGY_CONFIGS, PREFERRED_HOMES, InitializeEliteProdigies, AdoptU14Prodigies
├── youth_intake.go       # ACADEMY_FIRST, ACADEMY_LAST, ACADEMY_POS, REGEN_PERSONALITIES, RunYouthIntake
└── datamanager_test.go   # Full test suite covering 96 clubs, 2294 players, dedupe, 12 wonderkids, youth intake
```

### Essential Contract Definitions for Go:
```go
package datamanager

import (
	"football_sim/pkg/growth"
	"football_sim/pkg/models"
)

type EliteProdigyConfig struct {
	ClubID         string  `json:"club_id"`
	FullName       string  `json:"full_name"`
	Position       string  `json:"position"`
	Age            int     `json:"age"`
	HeightCM       float64 `json:"height_cm"`
	WeightKG       float64 `json:"weight_kg"`
	BaselineOVR    int     `json:"baseline_ovr"`
	Potential      int     `json:"potential"`
	AdultHeightAge int     `json:"adult_height_age"`
}

type DatasetMetadata struct {
	Season       string         `json:"season"`
	LeagueCounts map[string]int `json:"league_counts"`
	Totals       map[string]int `json:"totals"`
}

type DataManager struct {
	JSONPath     string
	GrowthEngine *growth.GrowthEngine
	Clubs        map[string]*models.Club
	ClubsList    []*models.Club
	Leagues      map[string][]*models.Club
	Wonderkids   []*models.Player
	Metadata     DatasetMetadata
	ProdigyHomes map[string]string
}
```

---

## 7. Verification Method

To independently verify these specifications against the running Python system and Go environment:

1. **Verify Raw Dataset Ingestion & Statistics**:
   ```pwsh
   python -c "import json; d = json.load(open('dataset.json', encoding='utf-8')); print(len(d['clubs']), sum(len(c['squad']) for c in d['clubs']))"
   ```
   *Expected*: `96 2294`

2. **Verify Python DataManager Ingestion, Wonderkids, and Deduplication**:
   ```pwsh
   python -c "from data_manager import DataManager; dm = DataManager(); print(len(dm.clubs_list), sum(len(c.squad) for c in dm.clubs_list), len(dm.wonderkids))"
   ```
   *Expected*:
   `[DataManager] Initialized 12 U-14 outfield franchise prodigies in GrowthEngine.`
   `96 2294 12`

3. **Verify Wonderkid Potential Invariant**:
   ```pwsh
   python -c "from data_manager import DataManager; dm = DataManager(); print([(w.full_name, w.age, dm.growth_engine.biometrics[w.player_id].potential) for w in dm.wonderkids])"
   ```
   *Expected*: All 12 wonderkids at age 14, potentials in range [93, 96], zero 99s.

4. **Verify Go Test Suite (Post-Implementation)**:
   ```pwsh
   cd backend_go
   go test -v ./pkg/datamanager/...
   ```
   *Expected*: PASS for all ingestion, deduplication, wonderkid, and academy intake tests.
