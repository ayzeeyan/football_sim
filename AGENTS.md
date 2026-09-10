# Super League Football Sim — Agent Guidelines & Architecture

## Overview
A career football simulator featuring:
- **Super League**: 12 elite clubs in a 44-week quadruple round-robin format (four cycles, two home and two away meetings per opponent).
- **Tournaments**: Champions Cup (2 groups of 6, two-legged knockouts, one-off final) and Super Cup (bracket play-in for seeds 5–12, top 4 byes).
- **Wonderkids**: 12 canonical U-14 prodigies with biometric progression, puberty curves, academic exams, mentors, and XP.
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
   - Exactly 12 canonical wonderkids start at age 14, in middle school, with category `FWD` and IDs prefixed with `WK_`.
   - Wonderkid potentials must stay strictly within the `[93, 96]` range (never 99).
   - Jhed Anthony Guinita belongs to Tottenham Hotspur (`EPL-TOT`).
   - Wonderkids have exam unavailability during matchweeks 12, 13, 24, 25, 32, and 33.
3. **Data Integrity & Economics**:
   - Exactly 0 duplicate players across and within club squads at all times.
   - Player valuations adhere strictly to valuation clamping (€300k minimum floor, €500M maximum ceiling; dynamic corridor [0.35 * anchor, 3.0 * anchor]).
   - Career save files and sharded club save files are runtime state and must never be committed.
4. **Development & Verification**:
   - Run backend tests with: `cd backend_go && go test ./...`
   - Run backend static checks with: `cd backend_go && go vet ./...`
   - Run frontend verification with: `cd frontend && bun install && bun run build`
   - Pull requests must pass `.github/workflows/ci.yml` before merge.
