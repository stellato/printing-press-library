# Equativ Maestro CLI — Shipcheck Report

## Verdict: ship-with-gaps

Grade A (94/100), all 6 shipcheck legs PASS. The "gaps" are live request/response
shapes that cannot be verified in-session (no API-User credentials available); they
are documented below and in the README's `## Known Gaps` block. None require code
changes beyond live validation, so promotion proceeds.

## Shipcheck legs (umbrella, --no-live-check)
| Leg | Result | Notes |
|-----|--------|-------|
| verify | PASS | mock-mode runtime checks green |
| validate-narrative | PASS | quickstart + recipes resolve under PRINTING_PRESS_VERIFY=1 |
| dogfood | PASS | novel_features_check 5/5 found; 0 issues |
| workflow-verify | PASS | |
| verify-skill | PASS | flag-names, flag-commands, positional-args, shell-var-quotes, unknown-command, canonical-sections all ✓ |
| scorecard | PASS | 94/100 Grade A |

## Scorecard 94/100
- 10/10: Output Modes, Auth, Error Handling, Terminal UX, README, Doctor, Agent Native,
  MCP Remote Transport, MCP Tool Design, MCP Surface Strategy, Local Cache, Breadth,
  Vision, Workflows, Insight, Path Validity, Auth Protocol, Sync Correctness.
- Type Fidelity 5/5, Dead Code 5/5.
- Soft: MCP Quality 8/10, Agent Workflow 9/10, Data Pipeline Integrity 7/10,
  Cache Freshness 5/10 (cache intentionally not enabled). → Phase 5.5 polish targets.
- N/A: MCP Desc Quality, MCP Token Efficiency, Live API Verification (no creds).

## Phase 3 transcendence: 5 planned, 5 built
deals drift, forecast sweep, reconcile, deals apply, deals funnel-rank — all
hand-coded, unit-tested (behavioral assertions: computeDrift/rankDriftRows,
reconcileFindings, expandCells/parseAvails, validateDealOp/planQuota, winRate/
rankFunnelRows), go build + go vet + go test green.

## Known Gaps (also in README `## Known Gaps`)
- inventoryInsights (forecast sweep), /deals create/update (deals apply),
  /report/troubleshooting (funnel-rank), /report/deals/pacing (drift) request/response
  shapes are APPROXIMATE — derived from the documented spec + a read-only browser-sniff,
  not verified against live writes/forecasts (no API-User credentials in-session).
  forecast sweep + deals apply default to dry-run / print the request body; responses
  parsed defensively. Validate against the live API before production use.
- `deals drift --since` parses but always compares the latest two snapshots.
- `reconcile` cross-resource checks run only when campaigns/lineitems/budgets are synced.
- Default base URL buyerconnectapis.smartadserver.com (verified); demand-api.eqtv.io
  available via EQUATIV_MAESTRO_BASE_URL.

## Hand-fixes applied
- Un-hid the `deals` parent command (generator hides resource commands on large surfaces;
  this one hosts the 3 novel deals commands). Retro candidate.

## Live testing
Skipped — API requires OAuth2 client-credentials and no API-User credentials were
available in-session. CLI verified against the spec, mocks, and unit tests.
