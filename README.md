# Top Five European Football Sim

A career football sim: 96 clubs across the Premier League, La Liga, Bundesliga, Serie A, and Ligue 1 on a shared 38-week calendar, with domestic cups and three UEFA competitions plus twelve named U-17 prodigies. You watch one match live on a 60 FPS pitch; the rest of the slate can be simmed instantly. Results stand.

Go is the server. React is the match centre. There is no Python runtime.

## What you play

- **Domestic leagues** — 20-team leagues play 380 fixtures (38 each, home and away once); 18-team leagues play 306 fixtures (34 each).
- **Domestic cups** — FA Cup, EFL Cup, Copa del Rey, DFB-Pokal, Coppa Italia, Coupe de France on the shared calendar.
- **Champions League** — 36-team Swiss/coefficient league phase (8 games each, even home/away), knockout play-offs for 9th–24th with top-8 byes, then two-legged knockouts and a one-off final. Extra time and penalties settle the decider; winners are never faked.
- **Europa / Conference League** — 20-team league phases feeding two-legged knockouts and one-off finals.
- **Qualification** — league position and cup wins carry into next season's UCL (36), Europa (20), and Conference (20) fields.
- **Wonderkids** — twelve canonical U-17s (high school, WK_ IDs, potential 93–96, never pinned — Jhed Guinita included). Puberty, exams, mentors, personalities, and match XP.
- **Matchday** — live WebSocket pitch with bookings, reds, rain, derbies, and European nights. Instant sim for everything else on the slate.
- **Window** — SUMMER / WINTER / CLOSED transfer FSM with AI bidding, wage caps from financial power, and sporting destination logic.

The UI tabs are Match, League, Wonderkids, Squads, Transfers, Inbox, History, and Competitions. The League tab shows your watched club's domestic league; Europe lives in the Competition Hub.

Legacy note: old 12-club 44-week Super League / Champions Cup / Super Cup saves are ignored for fresh careers (never silently converted) and remain only for isolated unit/match tests via `NewTournamentManager`.

## Requirements

- Go toolchain matching `backend_go/go.mod` (currently Go 1.27.1)
- Bun (for the React client)
- Windows, macOS, or Linux

## Run it

From the repo root:

```powershell
.\scripts\run.ps1
```

That installs frontend deps if needed, builds the client, and starts the Go server. Open the URL it prints — default **http://localhost:8000**.

If port 8000 is taken, the server walks to the next free port and logs it. The server binds to loopback by default; use `-host 0.0.0.0` only when you intentionally want LAN access.

Equivalent by hand:

```powershell
cd frontend
bun install
bun run build
cd ..\backend_go
go run ./cmd/server -dataset ..\dataset.json -static ..\frontend\dist -save ..\saves\career.json
```

## Develop

Two processes. Vite proxies `/api` and `/ws` to Go on port 8000.

Terminal 1:

```powershell
cd backend_go
go run ./cmd/server
```

Terminal 2:

```powershell
cd frontend
bun install
bun run dev
```

Open **http://localhost:5173**.

Or:

```powershell
.\scripts\run.ps1 -Dev
```

that only starts Vite; you still need the Go server in another terminal.

### Server flags

| Flag | Default | Meaning |
|---|---|---|
| `-host` | `127.0.0.1` | Bind address. Use `0.0.0.0` explicitly for LAN access |
| `-port` | `8000` | HTTP and WebSocket port (falls forward if busy) |
| `-dataset` | walks up for `dataset.json` | Club and player universe |
| `-save` | `saves/career.json` (or `FOOTBALL_SIM_SAVE`) | Career snapshot |
| `-static` | walks up for `frontend/dist` | Built React app |

If `frontend/dist` is missing, the API still runs. Build the client, or use Vite as above. Generated frontend output is intentionally not committed.

## Tests

```powershell
cd backend_go
go test ./...
go vet ./...
```

For the frontend:

```powershell
cd frontend
bun install
bun run build
```

Pull requests run both backend and frontend checks automatically through GitHub Actions.

## Career systems

- **Prizes** — participation, league-phase rank, per-round knockout, and runner-up/winner payments. European intake is tracked in its own season ledger (shown on the club panel), separate from domestic prize money.
- **Coefficients** — UEFA-style points from European results (league-phase wins/draws plus knockout progress) seed future Swiss pots, shown in the Competition Hub.
- **Wages** — annual caps derived from financial power bind every signing; valuation clamps (€300k–€500M) apply independently.
- **Loans** — summer and winter waves for prospects (wonderkids stay); solid destinations may hold a clamped buy clause that converts to a permanent move at season end when affordable and accepted.
- **Development** — seasonal growth rewards minutes played (full-90 equivalents, never above raw appearances) inside the +5 annual OVR cap and potential ceilings.
- **Conversations** — lightweight inbox replies for minutes, contracts, loans, form, and European nights; choices nudge morale, loyalty, or transfer requests only.

## Career save

The live save is `saves/career.json`. Full time, slate sims, transfers, and a clean shutdown all write it. Starting a new career from the UI replaces that file.

Runtime saves and their `saves/clubs/` sidecars are intentionally ignored by Git so playing the game does not dirty the repository. Copy the save directory elsewhere if you want a manual backup before a reset.

## Layout

```
backend_go/          Go module — server, engine, tests
  cmd/server/        Process entry
  pkg/server/        HTTP + /ws/match
  pkg/tournament/    Top Five world, cups, calendar (legacy Super League kept for tests)
  pkg/matchengine/   Live and instant matches
  pkg/growth/        Biometrics, puberty, training
  pkg/transfers/     Window and bids
  pkg/persistence/   Career snapshot
frontend/            React + Vite + Tailwind
  src/services/      api.ts and matchSocket.ts (the contract)
dataset.json         96 clubs, ~2,395 players; 12 canonical U-17 wonderkids
saves/career.json    Runtime career manifest (generated, not committed)
scripts/run.ps1      One-command play / -Dev
```

## API

Same origin as the page when you use the built client. Vite dev proxies these:

- `GET /api/health` — `{ "backend": "go", "status": "ok" }`
- `GET /api/clubs`, `/api/fixtures`, `/api/competitions`, `/api/competitions/{id}`
- `GET /api/super-league` (compatibility: watched club's domestic league in world careers), `/api/ucl`, `/api/super-cup` (legacy boards)
- `POST /api/fixtures/{id}/simulate`, `POST /api/fixtures/simulate-remaining`
- `POST /api/career/new` — `{ "shuffle", "homes" }`
- `GET /api/prodigies`, training and position-path POSTs
- `GET /api/transfers`, bid / advance
- `WS /ws/match` — live tick stream (`set_clubs`, `kickoff`, `pause`, `set_speed`, `reset`)
