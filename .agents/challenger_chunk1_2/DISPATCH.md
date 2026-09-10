# Dispatch: Challenger 2 (Data Ingestion, Deduplication & Wonderkids Stress Testing)

Authoritative Request: c:\Users\Izyan\Downloads\General\football_sim\.agents\ORIGINAL_REQUEST.md
Working Directory: c:\Users\Izyan\Downloads\General\football_sim\.agents\challenger_chunk1_2
Project Spec: c:\Users\Izyan\Downloads\General\football_sim\.agents\orchestrator\PROJECT.md

## Scope & Objective
Empirically challenge and stress test `backend_go/pkg/datamanager`:
1. Stress test deduplication: inject synthetic duplicate players across clubs and within clubs; verify `DedupePlayers` eliminates 100% of duplicates, leaving strictly 0 duplicates across the entire database.
2. Invariant verification: inspect all 96 clubs and all squad players in memory to guarantee 0 duplicate players.
3. Canonical wonderkids verification: assert that all 12 wonderkids start at age 14, in middle school, with `WK_` IDs, and potentials strictly in [93, 96] (never 99).
4. Relocation verification: assert that Jhed Anthony Guinita is at Tottenham Hotspur (`EPL-TOT`) and absent from `FL1-OM`.
5. Youth intake stress test: repeatedly trigger youth intake on clubs at squad size 32, 33, 34, 35; verify that `len(squad) <= 34` is never violated.

Output: Write your empirical findings, test code, and explicit verdict (`APPROVE` or `REJECT`) in `handoff.md`.

## 2026-09-07T07:04:48Z
You are Challenger 2 for the Football Sim Go backend rewrite Chunk 1.

MANDATORY: Read the authoritative user request first:
c:\Users\Izyan\Downloads\General\football_sim\.agents\ORIGINAL_REQUEST.md

Working Directory: c:\Users\Izyan\Downloads\General\football_sim\.agents\challenger_chunk1_2
Read instructions from: c:\Users\Izyan\Downloads\General\football_sim\.agents\challenger_chunk1_2\DISPATCH.md
Read Project Spec from: c:\Users\Izyan\Downloads\General\football_sim\.agents\orchestrator\PROJECT.md

Empirically challenge and stress-test `backend_go/pkg/datamanager`:
- Duplicate injection stress: inject duplicate players across clubs and within clubs; verify DedupePlayers removes 100% of duplicates leaving strictly 0 duplicates.
- Invariant check: verify that after loading and deduplication, 0 duplicate players exist across all 96 clubs.
- Wonderkid invariant: verify all 12 wonderkids are age 14, in middle school, with WK_ IDs and potentials [93, 96] (never 99).
- Relocation check: verify Jhed Anthony Guinita is at Tottenham Hotspur (EPL-TOT) and absent from Marseille (FL1-OM).
- Youth intake stress: test squad cap 34 under multiple intake rounds.

Write your empirical verification report in `c:\Users\Izyan\Downloads\General\football_sim\.agents\challenger_chunk1_2\handoff.md` with your explicit verdict: APPROVE or REJECT.
Update your `progress.md` frequently with a timestamp header for liveness.
When finished, send a message to orchestrator with your verdict.

