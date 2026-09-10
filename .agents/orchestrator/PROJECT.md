# Project: Football Sim Backend Rewrite (Chunk 1)

## Architecture
Chunk 1 implements the core foundation of the Football Sim backend in Go (`backend_go`):
- `pkg/models`: Pure domain entities (`Player`, `Club`, `Standings`), valuation math (`BaselineValue`, `ClampValue`, `FormatCurrency`), and personality archetypes (`PersonalityFor`, `SchoolWantFor`). Zero internal package dependencies. [COMPLETED & VERIFIED]
- `pkg/growth`: Biometric profile and growth engine (`GrowthEngine`, `BiometricProfile`, `TechnicalAttributes`), puberty progression, match XP, senior mentorship, training cycles, aging decline curves (30+), and youth development (<25). Thread-safe with `sync.RWMutex`. Zero internal package dependencies. [COMPLETED & VERIFIED]
- `pkg/datamanager`: Dataset ingestion (`dataset.json` with 96 clubs, 2,294 players), strict squad deduplication (0 duplicates invariant under distinct structs and pointer aliasing), 12 canonical U-14 wonderkids setup (age 14, middle school, potentials 93-96, relocation of Jhed Anthony Guinita to Tottenham `EPL-TOT`), and academy regen youth intake. Depends on `pkg/models` and `pkg/growth`. [COMPLETED & VERIFIED]

## Feature Inventory
| # | Feature | Description | Milestone | Source | Status |
|---|---------|-------------|-----------|--------|--------|
| 1 | Go Player Model | High-performance Player struct with availability, education, career stats, effective OVR fatigue drops | M1 | models.py | DONE |
| 2 | Go Club Model | Club struct with standings stats, 4-3-3 starting XI (wonderkid prioritization, fatigue rotation), bench selection, morale | M1 | models.py | DONE |
| 3 | Standings & Tiebreakers | StandingsTable sort: Points desc > GD desc > GF desc > Team Rating desc > Name asc | M1 | models.py, league.py | DONE |
| 4 | Valuation Curves | BaselineValue with youth boost, wonderkid boost, veteran depreciation past 32 | M1 | models.py | DONE |
| 5 | Valuation Clamping | ClampValue dynamic corridor [0.35 * anchor, 3.0 * anchor] and absolute bounds [€300k, €500M] | M1 | models.py | DONE |
| 6 | Currency Formatting | FormatCurrency (€K, €M, €B, €T) and FormatWage | M1 | models.py | DONE |
| 7 | Personality Archetypes | 4 archetypes (dedicated_pro, flamboyant_star, academic_dual, big_game_performer), canonical assignments, hash fallback | M1 | models.py | DONE |
| 8 | School & Exam Logic | Deterministic SchoolWant hash, ExamWeeks unavailable checks (weeks 12, 13, 24, 25, 32, 33) | M1 | models.py | DONE |
| 9 | BiometricProfile | Height, weight, puberty stage, growth velocity, adult height age, level XP target (145 * 1.18^lvl) | M2 | growth.py | DONE |
| 10 | TechnicalAttributes | Hexagonal matrix (pace, shooting, passing, dribbling, defending, physicality) + 7 sub-attributes | M2 | growth.py | DONE |
| 11 | GrowthEngine Core | Thread-safe engine (`sync.RWMutex`), attribute seeding, position-weighted OVR, nudge to OVR | M2 | growth.py | DONE |
| 12 | Puberty Simulation | Annual height/weight growth caps, aerial reach & stamina gains, transition to adult frame | M2 | growth.py | DONE |
| 13 | Match XP & Mentorship | Age multiplier, mentor multiplier, personality synergy, composure gains | M2 | growth.py | DONE |
| 14 | Training Cycles | Energy pool (3), hypertrophy, technical, and tactical regimens | M2 | growth.py | DONE |
| 15 | Aging Decline (30+) | Physical attribute decay (-1 for 30-33, -2 for 34-35, -3 for 36+) down to floor 35; OVR seasonal decay down to 55 | M2 | growth.py, tournament.py | DONE |
| 16 | Youth Development (<25) | Appearance-scaled growth (+1, +2, +3), bounded strictly by potential ceiling (never 99) | M2 | growth.py, tournament.py | DONE |
| 17 | dataset.json Ingestion | Lossless deserialization of 96 clubs and 2,294 players into native Go memory | M3 | data_manager.py, dataset.json | DONE |
| 18 | Season Ledger Reset | On fresh career kickoff, reset all player match/career stats to 0 | M3 | data_manager.py | DONE |
| 19 | Strict Squad Deduplication | Enforce 0 duplicate players across and within clubs; slot-based sweep handles pointer aliasing & distinct structs | M3 | data_manager.py | DONE |
| 20 | 12 Canonical Wonderkids | Setup at age 14, middle school, category FWD, stable WK_ IDs, exact potentials [93, 96] in GrowthEngine | M3 | data_manager.py | DONE |
| 21 | Wonderkid Relocation | Move Jhed Anthony Guinita from Marseille (FL1-OM) to Tottenham Hotspur (EPL-TOT) | M3 | data_manager.py | DONE |
| 22 | Market Baseline Snapping | Recalculate and clamp market values for all squad players upon load | M3 | data_manager.py | DONE |
| 23 | Academy Youth Intake | 2-4 graduates per club, squad cap 34, 20% golden generation (OVR 72-78, pot 90-95), unique name pool | M3 | data_manager.py | DONE |
| 24 | Full Test Suite Verification | `cd backend_go && go test -v ./...` passes 100% with 0 compiler warnings and 0 runtime panics (87/87 pass) | M4 | ORIGINAL_REQUEST.md | DONE |

## Milestones
| # | Name | Scope | Dependencies | Status |
|---|------|-------|-------------|--------|
| M1 | Core Domain Models & Valuation Math | `pkg/models` (Player, Club, Standings, Valuation, Personality, unit tests) | none | DONE |
| M2 | Growth Engine & Biometrics | `pkg/growth` (BiometricProfile, TechnicalAttributes, GrowthEngine, Puberty, Aging, Youth, unit tests) | none | DONE |
| M3 | Data Ingestion & Canonical Wonderkids | `pkg/datamanager` (Dataset ingestion, deduplication, 12 wonderkids, youth intake, unit tests) | M1, M2 | DONE |
| M4 | Final Full Verification & Forensic Audit | Full test suite `go test -v ./...`, Reviewers, Challengers, Forensic Auditor | M1, M2, M3 | DONE |

## Code Layout
```
backend_go/
├── go.mod
├── pkg/
│   ├── models/
│   │   ├── constants.go
│   │   ├── constants_test.go
│   │   ├── player.go
│   │   ├── player_test.go
│   │   ├── club.go
│   │   ├── club_test.go
│   │   ├── standings.go
│   │   ├── standings_test.go
│   │   ├── valuation.go
│   │   ├── valuation_test.go
│   │   ├── personality.go
│   │   ├── personality_test.go
│   │   └── challenger_stress_test.go
│   ├── growth/
│   │   ├── biometrics.go
│   │   ├── engine.go
│   │   ├── puberty.go
│   │   ├── progression.go
│   │   ├── aging.go
│   │   ├── growth_test.go
│   │   └── challenger_stress_test.go
│   └── datamanager/
│       ├── datamanager.go
│       ├── prodigies.go
│       ├── youth_intake.go
│       ├── datamanager_test.go
│       └── challenger_stress_test.go
```
