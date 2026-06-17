# Equativ Maestro CLI — Novel Features Brainstorm (subagent audit trail)

## Customer model
- **Priya — programmatic trader (agency desk):** owns ~40 live PMP deals. Monday pacing sweep deal-by-deal in the SPA; no memory of last week's numbers, so drift is invisible. Frustration: no single "which deals drifted since last week" view.
- **Marcus — media planner (avails/supply):** runs one inventory-insights forecast per click; needs grids of avails across geo × format × audience. Frustration: sweeping a targeting matrix = dozens of sequential manual runs; can't see the supply surface at once.
- **Dana — agency curation manager:** reconciles planned (deals/line-items/advertisers) vs delivering (reports) by hand across separate SPA screens. Frustration: no cross-entity reconciliation; orphans/mismatches found by accident.
- **Sam — activation/ad-ops:** bulk deal/line-item setup + teardown; hits 15 creates/500 updates per day caps with no warning until a write fails mid-batch. Frustration: no stage/validate/dry-run before commit; no quota visibility.

## Survivors (transcendence — Phase 3 hand-build)
| # | Feature | Command | Score | Buildability | Persona | Long Description |
|---|---------|---------|-------|--------------|---------|------------------|
| 1 | Pacing drift sweep | `deals drift [--since][--threshold]` | 8/10 | hand-code | Priya | "what changed since last week" — same deal across two synced times; for structural orphans use `reconcile` |
| 2 | Forecast permutation sweep | `forecast sweep --geo --format --audience [--diff <run-id>]` | 9/10 | hand-code | Marcus | GENERATES the avails grid (live), `--diff` compares two stored runs; only command that runs forecasts |
| 3 | Cross-entity reconciliation | `reconcile [--spend]` | 8/10 | hand-code | Dana | point-in-time structural integrity (orphans, budget≠spend); for time-over-time use `deals drift` |
| 4 | Bulk deal writer + quota guard | `deals apply <file> [--commit]` | 7/10 | hand-code | Sam | none |
| 5 | Bid-funnel win-rate ranking | `deals funnel-rank` | 6/10 | hand-code | Priya | rank ALL deals by funnel win-rate; one deal's raw funnel = absorbed troubleshooting report |

## Killed candidates
- `doctor --book` → folds into `reconcile`+`deals drift`.
- `search --type segment` → covered by reference sync + framework FTS.
- `deals drift --since/--threshold` (standalone) → flags on `deals drift`.
- `reconcile --spend` (standalone) → a check inside `reconcile`.
- `doctor --refdata` → framework `doctor`/`sync` concern.
- `forecast promote` → speculative, no research backing.
- `forecast diff` (standalone) → ships as `forecast sweep --diff`.

Full three-pass reasoning retained from subagent run (customer model → 12 candidates → adversarial cut to 5). All survivors get power from local SQLite snapshot history, cross-entity joins, throttled multi-call orchestration, or quota accounting — none possible in the stateless official Maestro MCP server.
