# Dispatch: Worker Round 2 (Pointer-Aliased Deduplication Remediation)

Authoritative Request: c:\Users\Izyan\Downloads\General\football_sim\.agents\ORIGINAL_REQUEST.md
Working Directory: c:\Users\Izyan\Downloads\General\football_sim\.agents\worker_r2
Project Spec: c:\Users\Izyan\Downloads\General\football_sim\.agents\orchestrator\PROJECT.md

## Reports to Read:
- Explorer R2.2 Handoff & Patch: c:\Users\Izyan\Downloads\General\football_sim\.agents\explorer_r2_2\handoff.md
- Explorer R2.1 Handoff: c:\Users\Izyan\Downloads\General\football_sim\.agents\explorer_r2_1\handoff.md
- Explorer R2.3 Handoff: c:\Users\Izyan\Downloads\General\football_sim\.agents\explorer_r2_3\handoff.md
- Forensic Auditor Report: c:\Users\Izyan\Downloads\General\football_sim\.agents\auditor_chunk1\handoff.md
- Challenger 2 Report: c:\Users\Izyan\Downloads\General\football_sim\.agents\challenger_chunk1_2\handoff.md

## Exclusive File Ownership
You exclusively own and may edit files in:
`backend_go/pkg/datamanager/`
Do NOT touch any files outside this directory.

## Objective
Remediate the pointer-aliasing deduplication failure in `backend_go/pkg/datamanager/datamanager.go` and `prodigies.go`.

Apply the slot-based universal squad sweep strategy agreed upon by all three Explorers:
1. In `DedupePlayers()`:
   - For each duplicate name group:
     - Merge stats via `maxInt()` for distinct player copies (`cp.player != keepPlayer`).
     - Sweep clubs in `dm.ClubsList`. Match players using `(p == keepPlayer) || strings.EqualFold(strings.TrimSpace(p.FullName), name)`.
     - In `keepClub`: retain strictly the FIRST matching occurrence as `keepPlayer`. For any subsequent matching occurrence in `keepClub`, merge stats, increment `removed++`, and drop from squad.
     - In all other clubs: drop all matching occurrences, merge stats, and increment `removed++`.
     - For all modified clubs, set `club.Squad = newSquad` and `club.SquadSize = len(club.Squad)`.
2. In `InitializeEliteProdigies()` (`prodigies.go:304-320`):
   - When cleaning up lingering wonderkid copies, use a similar slot-based retention (`keptProdigy := false` in home club, remove all other copies across clubs) to ensure identical pointers are also cleaned up.
3. Test and verify:
   - Run `cd backend_go && go test -v ./pkg/datamanager/...`
   - Run `cd backend_go && go test -v ./...`
   Verify that 100% of all tests pass (including `TestChallenger_DuplicateInjection_IdenticalPointerAttack` and `TestChallenger_DuplicateInjection_CrossClubSharedPointerAttack`), with 0 compiler warnings, 0 linter issues (`go vet`), and 0 runtime panics.

MANDATORY INTEGRITY WARNING:
DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. A auditor will independently verify your work. Integrity violations WILL be detected and your work WILL be rejected.

Write your completion report in `c:\Users\Izyan\Downloads\General\football_sim\.agents\worker_r2\handoff.md` with verbatim test execution output.

## 2026-09-07T07:16:40Z
You are Worker Round 2 for the Football Sim Go backend rewrite Chunk 1.
MANDATORY: Read the authoritative user request first:
c:\Users\Izyan\Downloads\General\football_sim\.agents\ORIGINAL_REQUEST.md
Working Directory: c:\Users\Izyan\Downloads\General\football_sim\.agents\worker_r2
Exclusive File Ownership: backend_go/pkg/datamanager/
Task: Apply the slot-based deduplication remediation to datamanager.go and prodigies.go in backend_go/pkg/datamanager/. Ensure pointer aliasing / shared struct references across clubs and within squads are properly deduplicated (retaining strictly 1 canonical copy in the canonical club and purging all others, while merging stats via max and syncing squad sizes). Run go test -v ./pkg/datamanager/... and go test -v ./... in backend_go.

