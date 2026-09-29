# Top Five European Football Sim — Agent Guidelines & Architecture

## Overview
A career football simulator featuring:
- **European world**: 96 clubs across 5 top-flight leagues (Premier League, La Liga, Bundesliga, Serie A, and Ligue 1), 6 domestic knockout cups, 3 UEFA competitions (Champions League Swiss phase, Europa League, Conference League), and a separate five-team European Nations Cup.
- **Wonderkids**: 12 canonical U-17 prodigies with biometric progression, puberty curves, academy/high-school tracks, mentors, and XP.
- **Matchday**: Simulation-only fixture resolution with reports, events, tactical analysis, and matchweek digests. The live WebSocket match is retired.
- **Transfers**: Transfer window FSM (Summer 12 weeks, Winter 4 weeks, Closed), AI bidding, wage caps derived from club financial power, expiring contracts, and Bosman free transfers.
- **Persistence**: Real-time snapshot saving to `saves/career.json` (with sharded club sidecars `saves/clubs/` and persistent universe master seed `saves/career.json.seed`).
- **Tech Stack**: Go backend (`backend_go`, Go 1.22+ declared by `go.mod`) and React 18 / TypeScript / Vite / Tailwind CSS frontend (`frontend`) managed exclusively with Bun.

---

## Architecture & Code Structure

```
football_sim/
├── backend_go/            # Go server, engine, domain models, tests
│   ├── cmd/server/        # Entrypoint (CLI flags: -host, -port, -dataset, -save, -static, -fmoe*)
│   ├── cmd/train/         # Offline FootballMoE trainer (the only weight writer)
│   ├── cmd/modeltool/     # FootballMoE inspect / verify / tensors / benchmark
│   ├── models/            # Exported .fmoe deployment models (immutable artifacts)
│   ├── pkg/footballai/    # FootballMoE network, typed requests, runtime brain, recorder
│   │   └── weights/       # Custom .fmoe / .fmckpt binary format (FBMOE001)
│   ├── pkg/models/        # Core entities (Player, Club, Standings, Valuations, Personality, Tactics)
│   ├── pkg/growth/        # Biometrics, puberty curves, progression, training, aging decline, +5 cap
│   ├── pkg/datamanager/   # Ingestion of dataset.json, squad deduplication, wonderkid init, youth intakes
│   ├── pkg/matchengine/   # Instant Poisson fixture simulation (legacy live code retained for contract tests)
│   ├── pkg/tournament/    # European World, 5 leagues, 6 cups, 3 UEFA comps, storylines, match importance
│   ├── pkg/transfers/     # AI bidding, transfer window FSM, wage caps, contracts, Bosman rules
│   ├── pkg/persistence/   # Career JSON serialization, sharded club saves, universe seed
│   ├── pkg/matchreport/   # Post-match statistics, reports, timeline summaries, tactical analytics
│   ├── pkg/managers/      # Manager profiles, tactical counter archetypes, board patience
│   └── pkg/server/        # HTTP REST endpoints and static SPA delivery
├── frontend/              # Single-page web application (React 18, Vite, TypeScript, Tailwind CSS, Bun)
│   ├── src/components/    # Feature screens grouped by domain: career/, clubs/, competitions/, layout/,
│   │                      # matches/, postmatch/, prematch/, transfers/, ui/, wonderkids/
│   ├── src/lib/           # Club crests, rigid tactical slots, qualification rules
│   ├── src/services/      # api.ts REST client contracts
│   └── package.json       # Dependencies, test scripts, and build tasks
├── dataset.json           # Database of 96 clubs and 2,395 squad players
├── saves/                 # Generated runtime career state; not committed
└── scripts/run.ps1        # Helper script for running dev & production servers
```

The full architecture breakdown (boot pipeline, weekly simulation lifecycle, domain formulas, calendar, competition math, persistence) lives in the **Architecture** section of `README.md`.

---

## Strict Domain Invariants & Rules

1. **Language & Runtime**:
   - The backend is written in pure Go using the toolchain version declared in `backend_go/go.mod` (Go 1.22+). There is **no Python runtime**.
   - The frontend is React 18 with TypeScript and Tailwind CSS and is managed exclusively with Bun. Do not add npm/yarn lockfiles to the repository.
2. **Wonderkids**:
   - Exactly 12 canonical wonderkids start at age 17 (U-17), in high school, with category `FWD` and IDs prefixed with `WK_`.
   - All 12 canonical wonderkids have exactly 99 potential, including when loading older careers.
   - Default clubs come from `dataset.json` and may be shuffled; no prodigy is pinned to a club (including Jhed Anthony Guinita).
   - High-school exam unavailability still applies during matchweeks 12, 13, 24, 25, 32, and 33 unless the player is on a football-first track.
3. **Data Integrity & Economics**:
   - Exactly 0 duplicate players across and within club squads at all times.
   - Player valuations adhere strictly to valuation clamping (€300k minimum floor, €500M maximum ceiling; dynamic corridor [0.35 * anchor, 3.0 * anchor]).
   - Structural annual wage caps derived from club financial power and reputation bind all player signings; transfer-window budgets must not exceed the club's available balance.
   - Career save files (`saves/career.json`), sharded club save files (`saves/clubs/*.json`), and the universe seed (`saves/career.json.seed`) are runtime state and must never be committed.
   - Starting XIs use rigid, unique tactical slots (`GK`, `LB`, `LCB`, `RCB`, `RB`, `LCM`, `CM`, `RCM`, `LW`, `ST`, `RW` in 4-3-3, or corresponding unique slots in 4-3-3 Attack, 4-2-3-1, and 4-4-2). Keep backend slot assignments (`models.AssignPlayersToFormation` / `Club.GetStartingElevenSlotsForFormation`) and frontend formation-pitch coordinates (`frontend/src/lib/tactics.ts`) in sync; never infer pitch placement solely from a repeated natural-position label.
   - Match events must retain player and club identifiers in addition to display text. Attribution must use the side responsible at the time of the event, particularly when possession changes during resolution.
   - Transfer-window FSM (`CLOSED` ↔ `SUMMER` ↔ `WINTER`) must complete all scheduled window weeks before season transition initializes; macro simulation paths must preserve the same window lifecycle boundaries.
4. **Gameplay Contracts**:
   - The shared European World club season has 38 matchweeks across 5 domestic leagues, 6 domestic cups, and 3 UEFA competitions. The national competition has its own schedule and must never mutate club fixtures or club statistics. Fixture labels, simulation routing, standings, and historical views must use the scheduled fixture identity rather than an assumed opponent.
   - When resolving a same-week slate, calculate each non-conflicting fixture wave before applying it (`slate_pool.go`). A club must not play two fixtures from the same wave using state already mutated by the first result.
   - Two-legged knockout ties are resolved on aggregate scorelines across swapped venues with no away goals rule; level aggregate after Leg 2 goes to extra time and penalties.
   - The Ballon d'Or score includes an explicit +10 bonus for players at the domestic league champion club.
   - Club crests are real assets mapped for all 96 dataset clubs in `frontend/src/lib/clubLogos.ts`; do not replace them with placeholders or introduce a fallback that masks missing mappings.
   - The UI is a neutral football-world viewer: every club, player, transfer, squad, and training decision is machine-selected by the AI world. Selecting a club or fixture provides inspection context only and never implies human club or player management; no endpoint may accept viewer-directed control of AI entities. The viewer's only verbs are simulate (advance the world) and inspect (read state).
5. **FootballMoE Neural Layer** (`pkg/footballai`):
   - Training is offline-only. `cmd/train` is the only component that modifies model weights; the game runtime loads an immutable `.fmoe` and must never update, mutate, or self-train weights during a career. No live weight updates, no online RL, no hidden continual-learning loop.
   - The network predicts probabilities and expectations; the simulation's deterministic RNG and rules resolve actual outcomes. FootballMoE must never roll events itself.
   - All gameplay-affecting AI features sit behind `AIConfig` flags (`-fmoe` + `-fmoe-features`), default off. The default simulation path must remain byte-identical when the brain is disabled.
   - Career saves persist `AIModelInfo` (format/model version + model hash). A save pinned to a different model hash keeps its behavior: the mismatch is logged and AI features stay disabled for the session — never a silent behavior change.
   - The runtime outcome recorder (`-fmoe-record`) is observational only: it records (pre-state, later outcome) pairs as JSONL for future offline training and must not alter any simulation result.
   - Inference must be deterministic: same `.fmoe` + same request + same world state = same output, no hidden neural RNG.
   - The `.fmoe` format (`FBMOE001`, little-endian, checksummed) carries architecture metadata, feature-schema version, and normalization statistics. Loaders must reject incompatible weights and schema mismatches with an error, never fall back silently.
   - `trainingdata/` bootstrap packs (`label_source = bootstrap_teacher_v1`) are warm-start data, not ground truth. Do not delete or overwrite them; weight label sources (bootstrap < simulation < historical/human) instead.
   - Every live integration site (match xG hint in `computeSlateFixture`, injury chance in `apply.go`, bid valuation in `transfers`) must stay bounded inside its corridor, fail open on any model error, and reject implausible outputs outright. The corridors and gates live in `pkg/tournament/footballai_bridge.go`, `pkg/transfers/transfers.go`, and `matchengine.BlendLambdaWithHint`.
   - Before enabling a flag in any release, run `cmd/abtest`: the AI-enabled world must stay inside the baseline realism bands (goals per match 2.2-3.4, home advantage intact). A flag that drifts outside the bands does not ship.
6. **Development & Verification**:
   - Run backend tests with: `cd backend_go && go test ./...`
   - Run backend static checks with: `cd backend_go && go vet ./...`
   - Run frontend verification with: `cd frontend && bun test && bun run build`
   - Pull requests must pass `.github/workflows/ci.yml` before merge.
