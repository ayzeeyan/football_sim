# Contributing

Thanks for helping build the sim. This document covers setup, the verification
gates every change must pass, and the conventions the project enforces.

## Setup

- Go toolchain matching `backend_go/go.mod` (Go 1.22+)
- Bun (the frontend is managed exclusively with Bun — never commit npm or yarn lockfiles)
- Clone, then:

```powershell
cd frontend
bun install
```

## Verification gates

Run all of these before opening a pull request. CI runs the same set.

```powershell
cd backend_go
go test ./...
go vet ./...
go test -race -p=1 ./...
gofmt -l .        # must print nothing

cd ../frontend
bun run lint      # Biome linter (CI enforces it)
bun test
bun run build
```

Or from the repo root: `make check`. CI also reports Go coverage and
enforces a 25% frontend line-coverage floor.

All gates must be green. Do not weaken or delete a failing test to make CI
pass — if a test encodes a wrong assumption, say so explicitly in your PR and
fix the test and the corresponding AGENTS.md rule together.

## Commit messages

Use conventional commits: `fix(backend):`, `feat(ui):`, `test(persistence):`,
`docs:`, `build:`, `refactor:`. Keep the subject line imperative and under
72 characters; explain the why in the body. Do not rewrite published history.

## Hard rules

Read `AGENTS.md` first — the "Strict Domain Invariants" section is binding:

- Determinism is sacred: seeded RNG only, no `time.Now()` or global `rand` in
  simulation paths, no map-iteration-order dependence, sort before candidate
  selection. Add a determinism test for anything that consumes randomness.
- Exactly 12 `WK_` wonderkids at 99 potential; zero duplicate players;
  valuation clamps (€300k floor, €500M ceiling, [0.35x, 3.0x] corridor).
- Rigid unique tactical slots stay in sync between `models/tactics.go` and
  `frontend/src/lib/tactics.ts`.
- Match events retain player and club IDs; two-legged ties aggregate with no
  away-goals rule; the transfer window FSM must complete before season
  transition.
- Never commit runtime state: `saves/career.json`, `saves/career.json.seed`,
  `saves/clubs/`, coverage artifacts, or built frontend output.
- No Python runtime. No new runtime dependency without discussing it first.
- If your change alters the save format, bump `SaveVersion` in
  `pkg/persistence` and add a migration plus a validation test.

## API changes

Routes live in the table-driven registry (`pkg/server/routes.go`). Every new
endpoint must: (1) be added to the registry, (2) appear in README.md's API
list, and (3) be covered by a route test. `TestRouteRegistryMatchesReadmeDocs`
fails the build if the docs and the registry drift.

## Frontend changes

- Keep components in their domain directory (`components/<domain>/`).
- Run `bun run lint`, `bun test`, and `bun run build`; all must pass.
- Format files you touch with `bunx biome format --write <files>`; do not
  reformat the whole tree in the same change.
- The UI is a neutral football-world viewer: every club is AI-controlled.
  Features that hand the user control of AI entities require an AGENTS.md
  amendment first — open an issue to discuss before implementing.
