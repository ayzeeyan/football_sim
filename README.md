# Super League

A career football sim: twelve elite clubs, a 33-week triple round-robin, Champions Cup, Super Cup, and twelve named U-14 prodigies. You watch one match live on a 60 FPS pitch; the rest of the slate can be simmed instantly. Results stand.

Go is the server. React is the match centre. There is no Python runtime.

## What you play

- **Super League** — 12 clubs, 33 matchweeks, 33 games each. Opening series, return series, then a home stretch that is not a copy of weeks 1–11.
- **Champions Cup** — two groups of six, two-legged quarters and semis, one-off final.
- **Super Cup** — play-in for seeds 5–12, byes for the top four, one-night ties.
- **Wonderkids** — twelve canonical kids (age 14, middle school, potential 93–96). Puberty, exams, mentors, personalities, and match XP.
- **Matchday** — live WebSocket pitch with bookings, reds, rain, derbies, and European nights. Instant sim for everything else on the slate.
- **Window** — the transfer market opens when the season ends.

The UI tabs are Match, League, Wonderkids, Squads, Transfers, Inbox, and History.

## Requirements

- Go 1.22 or newer
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

## Career save

The live save is `saves/career.json`. Full time, slate sims, transfers, and a clean shutdown all write it. Starting a new career from the UI replaces that file.

Runtime saves and their `saves/clubs/` sidecars are intentionally ignored by Git so playing the game does not dirty the repository. Copy the save directory elsewhere if you want a manual backup before a reset.

## Layout

```
backend_go/          Go module — server, engine, tests
  cmd/server/        Process entry
  pkg/server/        HTTP + /ws/match
  pkg/tournament/    Super League, cups, calendar
  pkg/matchengine/   Live and instant matches
  pkg/growth/        Biometrics, puberty, training
  pkg/transfers/     Window and bids
  pkg/persistence/   Career snapshot
frontend/            React + Vite + Tailwind
  src/services/      api.ts and matchSocket.ts (the contract)
dataset.json         96 clubs, 2,401 players; 12 elite + 12 prodigies used in career
saves/career.json    Runtime career manifest (generated, not committed)
scripts/run.ps1      One-command play / -Dev
```

## API

Same origin as the page when you use the built client. Vite dev proxies these:

- `GET /api/health` — `{ "backend": "go", "status": "ok" }`
- `GET /api/clubs`, `/api/fixtures`, `/api/super-league`, `/api/ucl`, `/api/super-cup`
- `POST /api/fixtures/{id}/simulate`, `POST /api/fixtures/simulate-remaining`
- `POST /api/career/new` — `{ "shuffle", "homes" }`
- `GET /api/prodigies`, training and position-path POSTs
- `GET /api/transfers`, bid / advance
- `WS /ws/match` — live tick stream (`set_clubs`, `kickoff`, `pause`, `set_speed`, `reset`)
