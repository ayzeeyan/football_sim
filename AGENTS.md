# Super League Football Sim — Agent Guidelines & Architecture

## Overview
A career football simulator featuring:
- **European world**: 96 clubs across the Premier League, La Liga, Bundesliga, Serie A, and Ligue 1, plus domestic cups and three UEFA competitions on a shared calendar.
- **Wonderkids**: 12 canonical U-17 prodigies with biometric progression, puberty curves, academy/high-school tracks, mentors, and XP.
- **Matchday**: Live 60 FPS WebSocket pitch simulation engine with ball physics and tactical controls, plus instant simulation for the remainder of the slate.
- **Transfers**: Transfer window, AI bidding, wage budgets, and contract negotiations.
- **Persistence**: Real-time snapshot saving to `saves/career.json`.
- **Tech Stack**: Go backend (`backend_go`, version declared by `go.mod`) and React 18 / TypeScript / Vite / Tailwind CSS frontend (`frontend`).

---

## Architecture & Code Structure

```
football_sim/
├── backend_go/            # Go server, engine, domain models, tests
│   ├── cmd/server/        # Entrypoint (CLI flags: -host, -port, -dataset, -save, -static)
│   ├── pkg/models/        # Core entities (Player, Club, Standings, Valuations, Personality)
│   ├── pkg/growth/        # Biometrics, puberty curves, progression, training, aging decline
│   ├── pkg/datamanager/   # Ingestion of dataset.json, squad deduplication, wonderkid setup
│   ├── pkg/matchengine/   # 60 FPS WebSocket tick simulation & instant slate resolution
│   ├── pkg/tournament/    # Super League, Champions Cup, Super Cup calendar & fixtures
│   ├── pkg/transfers/     # AI bidding, transfer window, wage budgets
│   ├── pkg/persistence/   # Career JSON serialization/deserialization (saves/career.json)
│   ├── pkg/matchreport/   # Post-match statistics, reports, and timeline summaries
│   ├── pkg/managers/      # Manager profiles and tactical tendencies
│   └── pkg/server/        # HTTP REST endpoints & /ws/match WebSocket handler
├── frontend/              # Single-page web application
│   ├── src/components/    # Matchday, League, Wonderkids, Squads, Transfers, Inbox, History tabs
│   ├── src/services/      # api.ts and matchSocket.ts client contracts
│   └── package.json       # React 18, Vite, TypeScript, Tailwind CSS
├── dataset.json           # Database of 96 clubs and 2,401 players
├── saves/                 # Generated runtime career state; not committed
└── scripts/run.ps1        # Helper script for running dev & production servers
```

---

## Strict Domain Invariants & Rules

1. **Language & Runtime**:
   - The backend is written in pure Go using the toolchain version declared in `backend_go/go.mod`. There is **no Python runtime**.
   - The frontend is React 18 with TypeScript and Tailwind CSS and is managed with Bun. Do not add npm/yarn lockfiles to the repository.
2. **Wonderkids**:
   - Exactly 12 canonical wonderkids start at age 17 (U-17), in high school, with category `FWD` and IDs prefixed with `WK_`.
   - Wonderkid potentials must stay strictly within the `[93, 96]` range (never 99).
   - Default clubs come from `dataset.json` and may be shuffled; no prodigy is pinned to a club (including Jhed Anthony Guinita).
   - High-school exam unavailability still applies during matchweeks 12, 13, 24, 25, 32, and 33 unless the player is on a football-first track.
3. **Data Integrity & Economics**:
   - Exactly 0 duplicate players across and within club squads at all times.
   - Player valuations adhere strictly to valuation clamping (€300k minimum floor, €500M maximum ceiling; dynamic corridor [0.35 * anchor, 3.0 * anchor]).
   - Career save files and sharded club save files are runtime state and must never be committed.
   - Starting XIs use rigid, unique tactical slots (`GK`, `LB`, `LCB`, `RCB`, `RB`, `LCM`, `CM`, `RCM`, `LW`, `ST`, `RW`). Keep the backend slot assignment and frontend formation-pitch coordinates in sync; never infer pitch placement solely from a repeated natural-position label.
   - Match events must retain player and club identifiers in addition to display text. Attribution must use the side responsible at the time of the event, particularly when possession changes during resolution.
   - Transfer-window budgets must not exceed the club's available balance. Season and macro simulation paths must preserve the same window lifecycle boundaries.
4. **Gameplay Contracts**:
   - The Super League season has 38 matchweeks. Fixture labels, macro simulation, watch-live routing, standings, and historical views must use the scheduled fixture identity rather than an assumed opponent.
   - When resolving a same-week slate, calculate each non-conflicting fixture wave before applying it. A club must not play two fixtures from the same wave using state already mutated by the first result.
   - The Ballon d'Or score includes an explicit +10 bonus for players at the Super League champion.
   - Club crests are real assets mapped for all 96 dataset clubs in `frontend/src/lib/clubLogos.ts`; do not replace them with placeholders or introduce a fallback that masks missing mappings.
5. **Development & Verification**:
   - Run backend tests with: `cd backend_go && go test ./...`
   - Run backend static checks with: `cd backend_go && go vet ./...`
   - Run frontend verification with: `cd frontend && bun test && bun run build`
   - Pull requests must pass `.github/workflows/ci.yml` before merge.
