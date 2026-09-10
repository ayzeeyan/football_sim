# Original User Request

## 2026-09-09T16:04:01Z

Overhaul the Football Sim game balance and core systems: rebalance wonderkid progression velocity to a realistic multi-year curve (+2 to +4 OVR/season), expand the season calendar to a 44-matchweek quadruple round-robin plus cups (~55-60 games/season with 6 matches every league week), implement position-driven tactical formations and pitch coordinates (e.g. CAM centrally attacking, CDM deep, proper flank distribution), and restructure the off-season transfer window into a 12-week market with club warchests, single-transfer locks, and 12-club wonderkid loan return rules.

Working directory: c:\Users\Izyan\General\football_sim
Integrity mode: development

## Requirements

### R1. Wonderkid Growth Curve Rebalance
- Rebalance the match XP formulas, level-up XP scaling, and appearance progression in `pkg/growth` so U-14 wonderkids (starting age 14, OVR 72–75) gain an average of +2 to +4 OVR per full season under consistent playing time.
- Wonderkids must develop along a multi-year trajectory (reaching ~79–82 OVR by age 16, ~85–88 OVR by age 18, and approaching their canonical 93–96 ceiling in their early 20s), eliminating single-season leaps into world-class ratings.

### R2. 44-Matchweek Calendar & Seasonal Game Volume (~60 Games)
- Expand the Super League schedule to 44 matchweeks (quadruple round-robin: 4 full cycles of 11 rounds across the 12 clubs).
- Guarantee that every league matchweek slate contains all 6 league fixtures (12 clubs playing each week), with no dropped or truncated matches.
- Together with Champions Cup and Super Cup fixtures, ensure elite clubs play ~55–60 total matches per season, and update calendar strips and standings calculations accordingly.

### R3. Position-Driven Tactical Lineup & Pitch Coordinates
- Implement position-aware formation logic and pitch coordinate assignment in `pkg/models` and `pkg/matchengine` that reflects each player's natural position:
  - Central Attacking Midfielders (`CAM`) positioned centrally in attacking midfield between the central midfielders and the striker.
  - Defensive Midfielders (`CDM`) positioned in deep central midfield.
  - Central Midfielders (`CM`) positioned in central midfield channels.
  - Flank players (`LB`, `LWB`, `RB`, `RWB`, `LM`, `RM`, `LW`, `RW`) positioned appropriately along the wings.
  - Strikers / Center Forwards (`ST`, `CF`) positioned centrally leading the attack.
- Ensure the live pitch canvas, lineup HUDs, and dugout views accurately render players according to their assigned tactical coordinates, eliminating generic left-wing clustering.

### R4. 12-Week Off-Season Transfer Window, Club Warchests & Wonderkid Rules
- Restructure the off-season transfer window into 12 weekly stages (Weeks 1 to 12) advancing week-by-week rather than daily ticks.
- Introduce an explicit, persistent transfer warchest/budget for each of the 12 clubs (initialized between €50M and €250M based on club stature).
- Ensure player purchases deduct from the buyer's warchest and add to the seller's warchest; prevent AI clubs from making bids that exceed their available balance.
- Enforce a strict single-transfer rule per window: once a player completes a transfer in a window, they cannot be transferred again in that same window.
- Restrict the 12 canonical wonderkids (`WK_` IDs) so they may only transfer between the 12 Super League clubs, and at the end of the season / start of the new campaign, automatically return them to their canonical original parent clubs.

## Acceptance Criteria

### Growth Tuning
- [ ] Simulating a full 44-week season with regular starts results in a wonderkid gaining +2 to +4 OVR (never exceeding +5 OVR in a single season).
- [ ] Wonderkid potential caps remain strictly in the [93, 96] range.

### Calendar & Match Counts
- [ ] Super League season consists of exactly 44 matchweeks, with each club playing 44 league games.
- [ ] Every league matchweek slate contains exactly 6 matches.
- [ ] Across league and cup tournaments, deep-run clubs record ~55–60 total matches in season history.

### Pitch Positioning & Lineups
- [ ] Starters have coordinates assigned based on their natural position (CAM at central attacking midfield, CDM deep central, correct wing/center spacing).
- [ ] Live match radar and canvas display players in their natural tactical formation without left-wing bias.

### Transfer Window & Financial Integrity
- [ ] Off-season transfer window advances through 12 weekly phases.
- [ ] Zero players transfer more than once within the same transfer window.
- [ ] Club warchests update on every completed transaction and prevent negative warchest balances.
- [ ] Wonderkids only transfer within the 12 Super League clubs, and any transferred wonderkid returns to their original club upon season reset.

### Verification & Stability
- [ ] `cd backend_go && go test ./...` passes 100% with 0 panics and 0 errors.
- [ ] `cd frontend && bun run build` (or `npm run build`) builds cleanly with 0 TypeScript compilation errors.
