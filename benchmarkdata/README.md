# FootballMoE benchmarkdata v1

This directory is intended to live at the repository root as `benchmarkdata/`.

## Never train on this folder

This benchmark is deliberately separate from `trainingdata/`.
The training command should refuse to consume benchmark files.

The benchmark uses a different random seed, different latent equations, and deliberately oversamples difficult/boundary cases. This avoids merely testing whether the model memorized the bootstrap teacher.

## Files

- `match_holdout.jsonl` — 6,000 match cases; ~32% are deliberately difficult distribution-shift cases.
- `player_holdout.jsonl` — 6,000 injury/rotation/development/decline cases; ~35% edge cases.
- `economy_club_holdout.jsonl` — 6,000 valuation/negotiation/contract/board cases; ~34% edge cases.
- `router_challenge.jsonl` — 3,000 semantic routing challenges. It checks required/allowed experts rather than forcing exact arbitrary router probabilities.
- `stress_cases.jsonl` — 1,000 numerical-stability inputs.
- `determinism_cases.jsonl` — 1,000 fixed requests for repeatability/hash checks.
- `simulation_sanity.json` — long-run deterministic and domain-invariant checks.
- `manifest.json` — immutable benchmark manifest.

Total JSONL cases: 23,000.

## Metrics

Use task-specific metrics rather than one universal score.

Recommended:
- Match: expected-goal MAE, goal-difference MAE, result log loss, calibration.
- Injury: Brier score, log loss, calibration error.
- Rotation: rest-value MAE plus lineup/ranking quality.
- Development/decline: MAE and error by age band.
- Valuation: log-MAE, median absolute percentage error.
- Negotiation: Brier/log loss for accept/walk-away; MAE for counteroffer ratio.
- Contract: Brier/log loss for renew; wage log-MAE.
- Board patience: Brier/log loss and calibration.
- Router: required-expert-in-top2 accuracy, forbidden-top1 violations, utilization, entropy, collapse checks.

## Candidate vs baseline

Always benchmark:
1. candidate `.fmoe`;
2. previous released `.fmoe`;
3. existing deterministic heuristic where available.

Do not collapse all results into a single "model score." A model can improve match prediction while breaking transfers.

## Real historical football data

For match realism/calibration, supplement this fixed pack with a separately versioned real-match benchmark.
OpenFootball provides public-domain match results in JSON for major leagues. Keep real data in a versioned folder such as `benchmarkdata/real/openfootball-v1/`, and never silently refresh it after a benchmark release.

StatsBomb Open Data can later be used for richer event/lineup/set-piece evaluation, subject to its attribution terms.

## Immutability

Once benchmarkdata v1 is committed:
- do not edit cases because a model performs badly;
- create benchmarkdata v2 instead;
- retain v1 for regression history.

That makes v6 vs v7 vs v20 comparisons meaningful.
