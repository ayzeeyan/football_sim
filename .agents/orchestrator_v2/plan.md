# Orchestration Plan — Football Sim Game Balance & Core Systems Overhaul

## 1. Survey Phase
- Spawn 3 Explorers in parallel to inspect the existing codebase against the requirements in ORIGINAL_REQUEST.md:
  - Explorer 1: R1 Wonderkid Growth Curve & Biometric Progression (pkg/growth, pkg/models, tests)
  - Explorer 2: R2 Calendar (44 matchweeks, quadruple round-robin, slates, cups) & R3 Tactical Lineups/Coordinates (pkg/tournament, pkg/matchengine, frontend pitch/radar)
  - Explorer 3: R4 12-Week Off-Season Transfer Window, Club Warchests (€50M-€250M), Single-Transfer Lock, Wonderkid Loan Return Rules (pkg/transfers, persistence, frontend transfers)
- Synthesize survey findings into PROJECT.md:
  - Architecture overview
  - Feature Inventory mapped to Milestones
  - Interface contracts and code layout

## 2. Milestone Execution (Project Pattern)
For each milestone:
  - Explorer phase: refine design & exact changes required
  - Worker phase: implement changes, run unit tests
  - Reviewer phase (2 Reviewers): verify correctness, test passing, non-regression
  - Challenger phase (2 Challengers): stress test edge cases
  - Forensic Auditor phase: audit against cheating/dummy code
  - Gate check: all APPROVE, CLEAN audit

### Milestones:
- Milestone 1 (M1): R1 Wonderkid Growth Curve Rebalance (+2 to +4 OVR/season, multi-year curve, [93, 96] potential cap)
- Milestone 2 (M2): R2 44-Matchweek Calendar & Quadruple Round-Robin (6 matches/week, ~55-60 games/season, calendar strips, standings)
- Milestone 3 (M3): R3 Position-Driven Tactical Lineups & Pitch Coordinates (CAM central attack, CDM deep, CM channels, wing spacing, ST/CF attack lead, eliminate left-wing bias)
- Milestone 4 (M4): R4 12-Week Transfer Window, Club Warchests & Wonderkid Rules (12 weekly stages, €50M-€250M warchests, single-transfer lock, 12-club restriction & original club return)
- Milestone 5 (M5): Full System Integration, E2E Testing, and Build Verification (`go test ./...` and `bun run build`)

## 3. Final Synthesis & Handoff
- Verify all acceptance criteria
- Generate handoff.md
- Report completion to Sentinel
