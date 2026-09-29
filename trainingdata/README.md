# FootballMoE bootstrap training data

This folder is intended to live at the repository root as `trainingdata/`.

## Files

- `router_seed.jsonl` — 5,600 contextual routing examples for the four experts: Match, Player, Economy, Club.
- `match_seed.jsonl` — 8,000 match examples with ratings, form, fitness, fatigue, tactics, absences and scored outcomes.
- `player_seed.jsonl` — 8,000 mixed examples for injury risk, rotation, development and decline.
- `economy_club_seed.jsonl` — 8,000 mixed examples for valuation, negotiation, contracts and board patience.
- `manifest.json` — dataset metadata.

## Important

These are **bootstrap teacher datasets**. They are intentionally generated from football-aware rules with noise and broad coverage. They are useful for:
1. validating the Go training pipeline;
2. getting the shared encoder, router and task heads to learn sensible behavior;
3. preventing cold-start routing collapse.

They are **not** a replacement for ground truth.

After FootballMoE is integrated, record real simulator decisions/outcomes into new JSONL files and fine-tune on those. Keep `label_source` so you can weight real outcomes more strongly than teacher data.

Recommended future label weights:
- real historical / observed outcome: 1.0
- simulator outcome from a stable release: 0.8
- bootstrap teacher data in this folder: 0.25 to 0.4

## Splits

Each row has `group_id` and `split`.
Use the supplied `train`, `val`, and `test` values instead of randomly re-splitting rows.

## JSONL

Each line is one independent JSON object:

`{"task":"...","features":{...},"target":{...}}`

This makes streaming batches in Go easy and avoids loading the entire dataset into RAM.

## Recommended first curriculum

1. Train `match_prediction` plus `router`.
2. Add `injury_risk` and `rotation`.
3. Add `valuation`.
4. Add development/decline.
5. Add negotiation/contracts/board patience.
6. Fine-tune jointly at a lower learning rate.

## Real-data expansion

For match calibration, add public-domain OpenFootball match results.
For richer event-level football data, StatsBomb Open Data is useful; follow its attribution terms.

Do not blend external data blindly with `football_sim` feature values. Map team/player names and normalize feature definitions first.
