# Survey Task 3: Data Ingestion, Deduplication, Canonical Wonderkids & Academy Intake

Authoritative Request: c:\Users\Izyan\Downloads\General\football_sim\.agents\ORIGINAL_REQUEST.md
Working Directory: c:\Users\Izyan\Downloads\General\football_sim\.agents\spec_miner_survey_3

## Objective
Investigate `data_manager.py`, `dataset.json`, and existing Go code in `backend_go/pkg/datamanager`.
Extract exact specifications for:
1. Ingesting `dataset.json` (96 clubs, 2,294 players) into native Go memory without data loss.
2. Strict squad deduplication rules (0 duplicate players across clubs and within clubs).
3. 12 canonical U-14 wonderkids: exact names, IDs (WK_ IDs), starting age 14, middle school status, and exact potentials in range 93-96 (never 99).
4. Academy regen youth intake mechanics.
5. Current state of `backend_go/pkg/datamanager`, what is implemented, what is missing or failing, and what tests exist or are needed.

Produce a comprehensive specification and gap report in `handoff.md` in your working directory.

## 2026-09-07T06:44:51Z
Received dispatch assignment for Spec Miner 3 for Football Sim Go backend rewrite Chunk 1.
Parent ID: 7396780a-6b1b-46cf-b464-6d10f96706f4
Working Directory: c:\Users\Izyan\Downloads\General\football_sim\.agents\spec_miner_survey_3
Authoritative User Request: c:\Users\Izyan\Downloads\General\football_sim\.agents\ORIGINAL_REQUEST.md
