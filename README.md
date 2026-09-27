# Top Five European Football Sim

A career football sim: 96 clubs across the Premier League, La Liga, Bundesliga, Serie A, and Ligue 1 on a shared 38-week calendar, with domestic cups, three UEFA competitions, national teams, and twelve named U-17 prodigies. Simulate fixtures or whole matchweeks, then inspect reports, standings, transfers, and awards.

Go is the server. React is the match centre. There is no Python runtime.

## What you play

- **Domestic leagues** — 20-team leagues play 380 fixtures (38 each, home and away once); 18-team leagues play 306 fixtures (34 each).
- **Domestic cups** — FA Cup, EFL Cup, Copa del Rey, DFB-Pokal, Coppa Italia, Coupe de France on the shared calendar.
- **Champions League** — 36-team Swiss/coefficient league phase (8 games each, even home/away), knockout play-offs for 9th–24th with top-8 byes, then two-legged knockouts and a one-off final. Extra time and penalties settle the decider; winners are never faked.
- **Europa / Conference League** — 20-team league phases feeding two-legged knockouts and one-off finals.
- **European Nations Cup** — five national squads drawn from the original club country of eligible players, with round-robin fixtures on five matchweeks, a final, standings, and season history. The dataset has no player nationality field, so original club country is the current eligibility rule.
- **Qualification** — league position and cup wins carry into next season's UCL (36), Europa (20), and Conference (20) fields.
- **Wonderkids** — twelve canonical U-17s (high school, WK_ IDs, 99 potential, never pinned — Jhed Guinita included). Puberty, exams, mentors, personalities, and match XP.
- **Matchday** — simulation-only fixtures with bookings, reds, rain, derbies, European nights, and detailed post-match reports.
- **Window** — SUMMER / WINTER / CLOSED transfer FSM with squad-need AI, contract-sensitive offers, wage and budget checks, and sporting destination logic.

The UI is a neutral football-world viewer: every club is AI-controlled, while selected clubs and fixtures only change inspection context. The primary navigation tabs are Home, Match Centre, Competitions, Tables, Clubs, Players, Transfers, News, History, and Wonderkids.

Legacy note: old 12-club 44-week Super League / Champions Cup / Super Cup saves are ignored for fresh careers (never silently converted) and remain only for isolated unit/match tests via `NewTournamentManager`. The retired live WebSocket match (`/ws/match`) returns HTTP 410 in normal server runs; the legacy live engine remains compiled in only for opt-in contract tests.

## Requirements

- Go toolchain matching `backend_go/go.mod` (Go 1.22+ declared, tested with Go 1.27.1)
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

Two processes. Vite proxies `/api` to Go on port 8000.

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
| `-port` | `8000` | HTTP port (falls forward if busy) |
| `-dataset` | walks up for `dataset.json` | Club and player universe |
| `-save` | `saves/career.json` (or `FOOTBALL_SIM_SAVE`) | Career snapshot |
| `-static` | walks up for `frontend/dist` | Built React app |

If `frontend/dist` is missing, the API still runs. Build the client, or use Vite as above. Generated frontend output is intentionally not committed.

## Tests

```powershell
cd backend_go
go test ./...
go vet ./...
go test -race -p=1 ./...
```

For the frontend:

```powershell
cd frontend
bun install
bun test
bun run build
```

`bun test` runs the frontend unit tests. `bun run build` runs `tsc` and builds production chunks with Rollup code-splitting. Pull requests run both backend and frontend checks automatically through GitHub Actions (`.github/workflows/ci.yml`): backend runs test, vet, and race detection; frontend runs install, test, and build.

## Architecture

### Boot & initialization

When the Go binary starts (`cmd/server/main.go`), it runs a strict bootstrap pipeline:

1. **CLI flag parsing** — `-host`, `-port`, `-dataset`, `-save`, `-static`.
2. **Asset resolution** — `dataset.json`, `saves/career.json`, `frontend/dist` (walking upward from the working directory when not given).
3. **Save inspection** — a legacy 12-club save is deleted with its seed (Top 5 world is enforced, never silently converted); a valid save is loaded into memory.
4. **Universe seed** — a new career generates a random 64-bit seed persisted to `saves/career.json.seed`; a loaded save reuses the pinned seed (legacy default `20260907`).
5. **Subsystem PRNG derivation** (`tournament.NewSubsystemRNG`) — independent deterministic streams for `development` (growth), `datamanager`, `matches` (world manager), `transfers`, and `live_match`.
6. **Universe construction** — DataManager loads and dedupes the dataset and registers the 12 wonderkids; EuropeanWorldManager builds 5 leagues, 6 cups, and 3 UEFA competitions; TransferEngine links managers, wage caps, and financial power.
7. **Restoration & validation** — an existing snapshot is restored via `RestoreCareer()` and checked with `ValidateWorldState()`; a fresh world assigns squad roles, board expectations, and loans.
8. **Server launch** — HTTP routes are registered and the static SPA fallback is armed.

### Weekly simulation lifecycle

Every matchweek progresses through one pipeline:

```
[ Matchweek N begins ]
  -> Availability (suspensions, injuries, high-school exams on MW 12, 13, 24, 25, 32, 33)
  -> Tactical selection (manager philosophy; 11 rigid slots, PositionFit: Natural/Good/Acceptable/Emergency)
  -> Matchday execution (instant Poisson engine, wave-partitioned slate)
  -> Result application (goals, assists, clean sheets, ratings; points/GD to standings; morale shifts)
  -> Weekly maintenance tick (match XP under the +5 cap, puberty cycles, training energy,
     manager job security, dugout conversations, transfer tick when a window is open)
  -> Persistence snapshot (atomic write to saves/career.json + saves/clubs/*.json)
```

### Concurrency & determinism

- **Coarse world lock** — `Server.worldMu sync.RWMutex` coordinates the tournament, growth, and transfer engines; subsystems keep their own fine-grained RWMutexes.
- **Macro-simulation lock** — `Sim Week / Month / Season / Continue` hold an execution lock and are rejected with HTTP 409 while a live fixture context is active.
- **Wave-based slate pool** — `slate_pool.go` partitions same-week fixtures into non-conflicting waves, so a club never plays two fixtures of one wave on state mutated by the first result.
- **Deterministic universe** — one master seed derives all subsystem PRNG streams, so a pinned seed replays identically across restarts.

## Domain model (`backend_go/pkg/models`)

### Player

The `Player` struct carries 90+ serialized attributes: identity and position (category GK/DEF/MID/FWD, CAM counts as FWD), OVR, age, market value and weekly wage, contract years, loyalty, wonderkid flags, season and career statistical ledgers, dressing-room state (morale, fitness, sharpness clamped 0–100), squad role, fatigue and rolling form, secondary positions and versatility, education and personality fields, mentorship, and loan buy clauses.

Key formulas:

- **Fatigue drop** — 0 below 3 consecutive starts; otherwise `min(4, 2 + (starts - 3))` subtracted from OVR (floor 40).
- **Form modifier** — `clamp(round((avgRating8 - 6.5) * 4), -8, +8)` from a rolling 8-match rating window.
- **Versatility** — `clamp(42 + 22*hasSecondary + floor(PositionXP/2) + 4*(age <= 21), 30, 92)`.
- **Bosman rule** — at `ContractYears <= 0` a player leaves on a free if they requested a transfer, are a snake personality, or have loyalty below 45; otherwise they re-sign for 2 years (loyalty < 70), 3 years (70–87), or 4 years (88+).

### Club identity & finances

Clubs are governed by a 9-dimensional identity profile (all 0–100): reputation, historical prestige, financial power, board patience, academy quality, recruitment ambition, youth preference, transfer aggressiveness, and selling tendency. Elite clubs have hand-tuned presets (e.g. `LAL-RMA`, `LAL-BAR`, `BUN-BAY`, `EPL-LIV`, `FL1-PSG`); other clubs derive defaults from team rating.

- **Initial budget** — `clamp(20 + floor(financialPower*13/10) + floor(reputation*3/10), 25, 180)` millions; balance is twice the transfer budget.
- **Annual wage cap** — `clamp(€80M + financialPower*€1.4M + reputation*€0.4M, €120M, €320M)`, ratcheted up on load if the committed bill already exceeds it.
- **Wage gate** — a signing passes only if `committedBill + wage*52 <= wageCap`.

### Formations & rigid tactical slots

| Formation | 11 non-overlapping slots |
|---|---|
| 4-3-3 (default) | GK, LB, LCB, RCB, RB, LCM, CM, RCM, LW, ST, RW |
| 4-3-3 Attack | GK, LB, LCB, RCB, RB, LCM, RCM, CAM, LW, ST, RW |
| 4-2-3-1 | GK, LB, LCB, RCB, RB, LDM, RDM, LW, CAM, RW, ST |
| 4-4-2 | GK, LB, LCB, RCB, RB, LM, LCM, RCM, RM, LST, RST |

Position fit hierarchy: **Natural** (+3000 selection score), **Good** (+2400, direct compatible lane), **Acceptable** (+1600, adjacent category role), **Emergency** (+0). Backend slot assignment (`models.AssignPlayersToFormation` / `Club.GetStartingElevenSlotsForFormation`, with `models.NormalizeFormation`, `models.FormationSlots`, `models.PlayersFromStartingSlots`, and `models.PositionFitForPlayer`) and the frontend pitch (`frontend/src/lib/tactics.ts`) are kept in exact sync.

### Personalities

Five archetypes: `dedicated_pro` (extra training, +10 leadership, mentor efficiency bonus), `flamboyant_star` (spotlight and sponsorships), `academic_dual` (+6 leadership, disciplined compliance), `big_game_performer` (clutch in derbies and European nights), and `snake` (conditional loyalty; leaves when unhappy).

### Standings tiebreakers

`StandingsTable.Less` sorts deterministically on five levels: points, goal difference, goals for, team rating, then club name.

### Valuation & wages

- **Baseline anchor** — `€11M * 1.086^max(0, OVR-65)`, with a 1.15x youth multiplier (age <= 21), 1.35x wonderkid multiplier, and 0.75^(age-32) veteran depreciation; anchor floor €500k.
- **Dynamic corridor** — market value is clamped to `[0.35 * anchor, 3.0 * anchor]`, then to the absolute bounds **€300k – €500M**.
- **Baseline weekly wage** — `€2,000 * 1.23^max(0, OVR-60)`, always bounded by the club's structural wage cap.

## The 12 canonical wonderkids

All twelve start at age 17, in high school, category FWD, with `WK_` IDs and **99 potential**. Default homes come from the dataset and may be shuffled at career creation; no prodigy is pinned to a club.

| Name | ID | Default club | Pos | OVR | Potential | Adult height age | Height / Weight |
|---|---|---|---|---|---|---|---|
| Venjamin Valerio | `WK_Venjamin_Valerio` | LAL-BAR | ST | 78 | 99 | 19 | 173 cm / 61 kg |
| Maverick Cantalejo | `WK_Maverick_Cantalejo` | LAL-RMA | CAM | 77 | 99 | 18 | 169 cm / 57 kg |
| Yeshua Emmanuel Gocotano | `WK_Yeshua_Emmanuel_Gocotano` | LAL-ATM | CF | 75 | 99 | 19 | 171 cm / 60 kg |
| Izyan Levin Bantol | `WK_Izyan_Levin_Bantol` | EPL-ARS | CAM | 76 | 99 | 18 | 170 cm / 58 kg |
| James Bernard Rizon | `WK_James_Bernard_Rizon` | EPL-LIV | RW | 76 | 99 | 20 | 178 cm / 66 kg |
| Reid Randell Libatan | `WK_Reid_Randell_Libatan` | BUN-BAY | LW | 76 | 99 | 19 | 176 cm / 64 kg |
| Ashle Zylle Baguio | `WK_Ashle_Zylle_Baguio` | BUN-DOR | CAM | 75 | 99 | 19 | 168 cm / 57 kg |
| Cliergy Jave Lanticse | `WK_Cliergy_Jave_Lanticse` | SEA-INT | RW | 75 | 99 | 20 | 169 cm / 58 kg |
| Ezail Zamora | `WK_Ezail_Zamora` | SEA-NAP | ST | 77 | 99 | 19 | 172 cm / 61 kg |
| Earl Josh Hernando | `WK_Earl_Josh_Hernando` | SEA-MIL | LW | 75 | 99 | 19 | 170 cm / 59 kg |
| Rich Lorenz Suico | `WK_Rich_Lorenz_Suico` | FL1-PSG | LW | 75 | 99 | 18 | 170 cm / 59 kg |
| Jhed Anthony Guinita | `WK_Jhed_Anthony_Guinita` | EPL-TOT | CF | 75 | 99 | 19 | 174 cm / 62 kg |

Enrolled prodigies sit exams on matchweeks 12, 13, 24, 25, 32, and 33 unless on a football-first track. Academy youth intake graduates 2–4 players per club per season (34-man squad ceiling, 20% golden-generation chance with OVR 72–78 / potential 90–95).

## Season calendar

All club competitions share one 38-matchweek calendar:

```
MW 2      Domestic Cup Round 1
MW 3      European league phase MD1
MW 5      EFL Cup Round 1
MW 6      European league phase MD2
MW 9      European league phase MD3
MW 10     Domestic Cup Round 2
MW 12     European league phase MD4   (high-school mid-term exams)
MW 13     EFL Cup Round 2            (high-school mid-term exams)
MW 15     European league phase MD5
MW 18     European league phase MD6 & Domestic Cup Round 3
MW 20     EFL Cup Round 3
MW 21     Winter transfer window opens (4 weeks)
MW 23     European league phase MD7
MW 24-25  High-school winter exams
MW 26     European league phase MD8 (finale)
MW 27     Domestic Cup semi-finals
MW 28     European knockout play-offs leg 1 & EFL Cup semi-finals
MW 29     European knockout play-offs leg 2
MW 30-31  European Round of 16 (legs 1 & 2)
MW 32-33  European quarter-finals (legs 1 & 2)   (high-school spring finals)
MW 35     European semi-finals leg 1 & Domestic Cup finals weekend
MW 36     European semi-finals leg 2 & EFL Cup final
MW 38     Domestic league finale & European finals (UCL, UEL, UECL)
Off-season Awards ceremony & 12-week summer transfer window
```

## Competitions

### Champions League Swiss draw

The 36-team league phase is an exact Swiss draw (`world_manager.go`):

1. **Pot seeding** — 4 pots of 9 by UEFA coefficient and team rating.
2. **Pot-pair blocks** — every club meets exactly 2 opponents per pot (8 games), via seeded 9-cycles intra-pot and double-shifted matchings inter-pot; same-league clashes minimized.
3. **Weekly matchings** — the 144 pairings decompose into 8 weekly perfect matchings of 18 games using an MRV heuristic with deterministic retries.
4. **Balanced orientation** — Eulerian trails orient the 8-regular graph so every club plays exactly 4 home and 4 away games.

### Knockouts

Ranks 9–24 play two-legged knockout play-offs; ranks 1–8 bye to the Round of 16. Two-legged ties swap venues with **no away-goals rule**; level aggregates after leg 2 go to extra time and penalties. European finals are one-off matches in MW 38.

### Promotion and relegation

The dataset contains only the five top flights — there is no second division — so the world is a closed pyramid: the five leagues form a prestige ladder (Premier League → La Liga → Bundesliga → Serie A → Ligue 1) and each adjacent pair exchanges boundary clubs at the season transition. The bottom three of the stronger league are relegated into the weaker league; the top three of the weaker league are promoted into the stronger one. League sizes never change, all 96 dataset clubs (and their crest mappings) are preserved, and the exchange is a deterministic function of the final tables. Ligue 1 is the base of the pyramid: its relegation places are a survival battle with no lower tier to drop into, and the season-transition news says so explicitly.

### Qualification quotas

Next season's fields: UCL 36 (8 England, 8 Spain, 8 Italy, 6 Germany, 6 France, plus the titleholder), Europa 20 (domestic cup winners plus top remaining league positions), Conference 20 (next-highest league finishers).

### Coefficients & prize money

- **Coefficient points** — league-phase win +2 / draw +1; knockout progression +1 (play-off) to +8 (final); champion bonus +4. Domestic results never award points.
- **Prize money** — domestic league finish scaled by prestige and position; domestic cup winner `(5 + floor(prestige/5))` millions with 40% runner-up share; European participation UCL €12M / UEL €6M / UECL €4M, plus rank prizes and per-round knockout payments up to UCL final €5M with winner bonuses. 50% of prize money reinvests into the transfer budget; European intake is tracked in a dedicated `EuropeanRevenue` ledger per season.

### Awards

Ballon d'Or score: `OVR*1.2 + goals*3.5 + assists*2.5 + apps*0.4 + (avgRating-6.5)*8 + teamBonus`, with an explicit **+10** bonus for players at the domestic league champion club and **+12** for UCL winners. Golden Boy (best U-21), Player of the Season (avg rating), Golden Boot (goals; tiebreak assists, OVR, ID), Playmaker, Young Player, Golden Glove, and league scoring awards round out the ceremony.

### Match importance & storylines

`match_importance.go` grades every fixture as Routine, Important, Big Match, Derby (El Clásico, North London, Madonnina, Klassiker, and heat-based rivalries), Cup Final, or Six-Pointer (late-season clashes within 3 points). `storylines.go` generates factual headlines from live tables: win streaks, title races, relegation scraps, scoring races, and European runs.

## Match engine & reports

Instant simulation (`matchengine/instant.go`) models 90 minutes with a multi-factor Poisson model:

- **Goal expectancy** — home/away lambdas from team ratings, morale bonuses (±0.8), home edge, and a snow factor (0.95).
- **Tactical counters** — high press beats possession, possession beats low block, low block beats free-flowing, free-flowing beats high press; the winning matchup adds +0.16 home edge, losing −0.12.
- **Weather** — rain cuts shooting accuracy 4% and passing 5%; wind cuts shooting 8%; snow cuts goal expectancy 5%.
- **Discipline** — 7% straight-red chance (+15% in high-heat derbies), second yellows auto-dismiss, and 12 referees with strict (1.2x), lenient (0.8x), or balanced card climates.
- **Penalties & VAR** — 22% in-play penalty chance with OVR-scaled conversion; 5% of goals trigger VAR reviews, 30% of which disallow.

Every fixture compiles a `MatchReport` (`matchreport/`): a timeline of events retaining flat player/club IDs, per-player rows with exact minutes and 6.0–10.0 ratings, clean sheets for GK/defenders with 60+ minutes and no goals conceded, possession-split pass totals and accuracy, xG from shot location and pressure, and tactical analytics (territory per third, counter efficiency).

## Growth & development (`backend_go/pkg/growth`)

- **Puberty** — stages Early (<= 14), Mid (15–16), Late (17+), Adult; annual height velocity caps 2.6 / 2.4 / 1.8 cm; 24% spurt chance per cycle; weight gain capped at 5 kg lifetime; adult-height cessation age (18–20) derived deterministically per player.
- **Match XP** — `rating*2.2 + goals*5 + assists*3`, scaled by age (1.05x at <= 16 down to 0.5x at 25+) and mentor multipliers (up to +0.25, +0.05 extra for dedicated pros).
- **Mentorship clinics** — weekly 32% proc (45% for dedicated pros) transferring composure, archetype-specific drills, and direct XP.
- **Training** — 3 energy units per week for hypertrophy, technical, or tactical regimens.
- **Hard cap** — no player gains more than **+5 OVR per season** (`season_cap.go` rolls back breaches).
- **Aging** — decline of −1 OVR/yr at 30–33, −2 at 34–35, −3 at 36+; retirement probability 10% at 36 rising to mandatory at 42.

## Transfer market (`backend_go/pkg/transfers`)

- **Window FSM** — `CLOSED` -> `SUMMER` (12 weeks) -> `CLOSED` -> `WINTER` (4 weeks from MW 21) -> `CLOSED`; the next season cannot initialize while window weeks remain.
- **Negotiation pipeline** — INQUIRY (20%) -> COUNTER_OFFER (40%) -> HIJACK_CHECK (60%) -> TERMS_MEDICAL (80%) -> COMPLETED (100%); any failure collapses the deal.
- **Destination scoring** — players weigh reputation and team-rating deltas, league prestige, crucial/star protection penalties, and morale; unhappy players (morale < 45) get a +4 boost.
- **Contracts** — annual decrement; Bosman free agency for transfer-requested, snake, or low-loyalty players; renewal length from loyalty tier; wages re-anchored via `WageForOVR` under the structural cap.

## Persistence (`backend_go/pkg/persistence`)

- **Master manifest** — `saves/career.json` holds the `CareerSnapshot` (current save version 9): tournament state, inbox, calendar, milestones, club index.
- **Report retention** — finished matches keep full reports for the last 3 matchweeks, compacted reports (no heatmaps/xG-flow samples) for the following 5, and beyond that an archival summary (score, scorers with player+club IDs, MOTM, key stats) so a 38-week career save stays bounded.
- **Sharded sidecars** — `saves/clubs/{club_id}.json` store each club's squad independently to avoid monolithic I/O.
- **Universe seed** — `saves/career.json.seed` pins the 64-bit master seed (legacy default `20260907`).
- **Atomic writes** — snapshots write to a `.tmp` file, verify, then rename over the target; crash corruption cannot leave a half-written save.
- **Validation** — `ValidateCareerSnapshot` checks version, matchweek/window boundaries, finances (`TransferBudget <= Balance`), zero duplicate players, valuation corridor compliance, and knockout bracket continuity.
- **Legacy mapping** — `ProdigyMap` remaps old player IDs to canonical `WK_` IDs on load.

## Frontend (`frontend/`)

Single-page React 18 + TypeScript app, styled with Tailwind CSS, built with Vite, managed exclusively with Bun. All primary screens are lazy-loaded via `React.lazy` behind an accessible loading fallback.

Tabs (defined in `src/lib/constants.ts`): Home, Match Centre, Competitions, Tables, Clubs, Players, Transfers, News, History, Wonderkids.

```
src/
  components/
    career/        New career modal, inbox, awards ceremony
    clubs/         Squad, club identity, player sheet, players tab
    competitions/  Standings, competition hub, home dashboard, history
    layout/        TopBar, global search, toasts, awards modal
    matches/       Formation pitch, match cards, simulation centre
    postmatch/     Match detail modal, broadcast, events, insights, digest
    prematch/      Calendar strip, match browser, fixture actions, insights
    transfers/     Transfer market tab
    ui/            Reusable primitives (Card, Badge, Button, ClubCrest)
    wonderkids/    Wonderkid lab, prodigy radar and watch
  hooks/           useAsyncData, useClubs, useToast
  lib/             Club crests (all 96 clubs), tactics slots, qualification rules, formatters
  services/        api.ts — typed REST client
  audio/           webAudio.ts — procedural Web Audio synthesis (no audio files)
  types/           Master TypeScript domain models and API contracts
```

Club crests are real, verified assets mapped for all 96 clubs in `src/lib/clubLogos.ts`; player portraits derive deterministic palettes from name hashes.

## Career systems

- **Prizes** — participation, league-phase rank, per-round knockout, and runner-up/winner payments. European intake is tracked in its own season ledger (shown on the club panel), separate from domestic prize money.
- **Awards** — Ballon d'Or, Golden Boot, Playmaker, Golden Boy, Young Player, Golden Glove, league scoring awards, and competition-specific scoring awards use season performance and team achievement.
- **Coefficients** — UEFA-style points from European results (league-phase wins/draws plus knockout progress) seed future Swiss pots, shown in the Competition Hub.
- **Wages** — annual caps derived from financial power and reputation bind every signing; valuation clamps (€300k–€500M) apply independently.
- **Tactics & Lineups** — rigid 11-slot tactical assignments across supported formations (4-3-3, 4-3-3 Attack, 4-2-3-1, 4-4-2) kept in exact sync with pitch visualizers (`tactics.go`, `tactics.ts`).
- **Loans** — summer and winter waves for prospects (wonderkids stay); solid destinations may hold a clamped buy clause that converts to a permanent move at season end when affordable and accepted.
- **Development** — seasonal growth rewards minutes played (full-90 equivalents, never above raw appearances) inside the +5 annual OVR cap and potential ceilings.
- **Post-Match Broadcast** — interactive post-match broadcast modal (`PostMatchBroadcast.tsx`) providing full-time match summaries, team stats, player ratings, chronological events, table impact, and competition context.
- **Storylines & Importance** — factual narrative headlines (`storylines.go`) tracking streaks, title races, and European runs, alongside contextual fixture stakes (`match_importance.go`: Routine, Important, Big Match, Derby, Cup Final, Six-Pointer).
- **Conversations** — lightweight inbox replies for minutes, contracts, loans, form, and European nights; choices nudge morale, loyalty, or transfer requests only.

## Career save

The career save is `saves/career.json`. Fixture and slate simulations, transfers, and a clean shutdown write it. Starting a new career replaces that file, with universe seed pinned in `saves/career.json.seed` and sharded club sidecars in `saves/clubs/`.

Runtime saves and their `saves/clubs/` sidecars are intentionally ignored by Git so playing the game does not dirty the repository. Copy the save directory elsewhere if you want a manual backup before a reset.

## Layout

```
backend_go/          Go module — server, engine, tests
  cmd/server/        Process entrypoint (CLI flags: -host, -port, -dataset, -save, -static)
  pkg/datamanager/   Ingestion of dataset.json, squad deduplication, wonderkid init, youth intakes
  pkg/growth/        Biometrics, puberty curves, progression, training, aging, +5 annual cap
  pkg/managers/      Manager profiles, tactical counter archetypes, board patience
  pkg/matchengine/   Instant Poisson slate engine (legacy live code retained for tests)
  pkg/matchreport/   Post-match statistics, reports, timeline summaries, tactical analytics
  pkg/models/        Core domain entities, rigid 11-slot formations, valuations, wage caps
  pkg/persistence/   Career JSON snapshots, sharded club saves, universe seed
  pkg/server/        HTTP REST router and static SPA fallback
  pkg/tournament/    Top Five European world, 6 cups, 3 UEFA comps, calendar, storylines, importance
  pkg/transfers/     Transfer window FSM, AI bidding, contracts, Bosman free transfers
frontend/            React 18 + TypeScript + Vite + Tailwind CSS + Bun
  src/components/    Feature screens grouped by domain (see Frontend section above)
  src/lib/           Club crests (all 96 clubs), tactical pitch slots, qualification rules
  src/services/      REST API client
dataset.json         96 clubs, 2,395 squad players; 12 canonical U-17 wonderkids
saves/career.json    Runtime career manifest (generated, not committed)
scripts/run.ps1      One-command play / -Dev launcher
.skills/             Agent skill packs (frontend-design); licensed under the root MIT LICENSE
```

## API

Same origin as the page when you use the built client. Vite dev proxies these:

- `GET /api/health` — `{ "backend": "go", "status": "ok" }`
- `GET /api/stats` — Diagnostics snapshot (`DiagnosticsSnapshot`)
- `GET /api/clubs`, `/api/clubs/{club_id}/squad`, `/api/clubs/{club_id}/xi`, `/api/clubs/{club_id}/history`, `/api/clubs/{club_id}/profile`, `/api/clubs/{club_id}/fixtures`, `/api/clubs/{club_id}/transfers`
- `GET /api/clubs/{club_id}/scouting?limit=` — deterministic AI recruitment shortlist (consistency, ceiling, form, value trend, risk); observational only
- `GET /api/h2h/{club_a}/{club_b}`, `/api/players/{player_id}`, `/api/search?q=...`
- `GET /api/prodigies`, `/api/prodigies/watch`, `/api/wonderkids` (legacy alias)
- `GET /api/prodigies/{player_id}/timeline`, `/api/growth/milestones`, `/api/training/status`, `/api/training/projection/{player_id}`, `/api/nxgn50`
- `POST /api/prodigies/{player_id}/train`, `POST /api/prodigies/{player_id}/position-path`, `POST /api/prodigies/{player_id}/school-track`
- `GET /api/calendar`, `/api/fixtures`, `/api/fixtures/{fixture_id}`, `/api/competitions`, `/api/competitions/{competition_id}`
- `GET /api/competitions/nations-cup` — national squads, results, table, and history
- `GET /api/competitions/nations-cup/fixtures/{fixture_id}`, `POST /api/competitions/nations-cup/fixtures/{fixture_id}/simulate` — national-team fixtures (never mutate club data)
- `GET /api/openapi.json` — OpenAPI 3.1 spec generated from the server's route table
- `GET /api/super-league` (compatibility: selected domestic-league view in world careers; optional `?league=` accepts a league name or competition ID and falls back to the default view), `/api/ucl`, `/api/ucl/fixtures`, `/api/super-cup`
- `POST /api/fixtures/{fixture_id}/simulate`, `POST /api/fixtures/simulate-remaining`
- `GET /api/fixtures/{fixture_id}/whatif?seed=` — read-only sandbox: alternative scoreline, xG, and hypothetical table movement under a scratch seed; the recorded result always stands
- `POST /api/sim/continue`, `POST /api/sim/week`, `POST /api/sim/month`, `POST /api/sim/season`
- `GET /api/world/dashboard`, `/api/scoring-race`, `/api/trophies`, `/api/records`
- `GET /api/season/awards`, `/api/season/awards/ceremony`, `/api/season/history`, `/api/season/stats`, `/api/season/stats/advanced`
- `POST /api/season/reset`, `POST /api/season/restart`
- `GET /api/career/default-homes`, `/api/career/preview-shuffle`, `POST /api/career/new` — `{ "shuffle", "homes" }`
- `GET /api/favourite`, `POST /api/favourite` (observational viewing preference), `GET /api/week/watch`
- `GET /api/watchlist`, `POST /api/watchlist` — multi-entity watchlist (clubs, players, competitions; observational only, feeds weekly watch digests)
- `GET /api/transfers`, `/api/transfers/records`, `POST /api/transfers/bid`, `POST /api/transfers/advance`
- `GET /api/inbox`, `POST /api/inbox/read`, `POST /api/inbox/reply`
- `GET /api/export/standings?league=`, `/api/export/squad?club_id=`, `/api/export/fixtures`, `/api/export/transfers` — user-initiated CSV downloads of resolved state

The retired `/ws/match` endpoint returns HTTP 410 in normal server runs.
