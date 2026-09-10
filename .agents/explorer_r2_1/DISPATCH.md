# Dispatch: Explorer Round 2 - Fix Strategy for Pointer-Aliased Deduplication Defect

Authoritative Request: c:\Users\Izyan\Downloads\General\football_sim\.agents\ORIGINAL_REQUEST.md
Working Directory: c:\Users\Izyan\Downloads\General\football_sim\.agents\explorer_r2_1
Project Spec: c:\Users\Izyan\Downloads\General\football_sim\.agents\orchestrator\PROJECT.md
Gate Status: c:\Users\Izyan\Downloads\General\football_sim\.agents\orchestrator\GATE_STATUS.md

## Full Audit & Review Evidence (MANDATORY TO READ):
- Forensic Auditor Report: c:\Users\Izyan\Downloads\General\football_sim\.agents\auditor_chunk1\handoff.md
- Challenger 2 Report: c:\Users\Izyan\Downloads\General\football_sim\.agents\challenger_chunk1_2\handoff.md
- Reviewer 2 Report: c:\Users\Izyan\Downloads\General\football_sim\.agents\reviewer_chunk1_2\handoff.md

## Failure Details
Running `cd backend_go && go test -v ./...` failed on:
- `TestChallenger_DuplicateInjection_IdenticalPointerAttack`
- `TestChallenger_DuplicateInjection_CrossClubSharedPointerAttack`

Root cause:
In `backend_go/pkg/datamanager/datamanager.go:321`, `DedupePlayers()` checks:
```go
for _, cp := range copies {
    if cp.player == keepPlayer {
        continue
    }
    ...
}
```
When duplicate players share the exact same struct pointer in memory (pointer aliasing across clubs or within a squad), `cp.player == keepPlayer` is true on all iterations, so 0 duplicates are removed.

## Objective
Analyze the code in `backend_go/pkg/datamanager/datamanager.go` and provide a concrete, fully verified fix strategy for Worker.
Investigate:
1. Exact fix to `DedupePlayers()` that ensures 100% duplicate elimination even under pointer aliasing.
2. How to ensure stats merging (`max()`), squad size synchronization (`SquadSize = len(Squad)`), and canonical player retention work correctly.
3. Verify that the fix strategy addresses all findings from the Forensic Auditor, Challenger 2, and Reviewer 2.

Do NOT implement the fix (you are read-only). Write your report and recommended fix in `handoff.md` in your working directory.
