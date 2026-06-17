# Equativ Maestro CLI — Phase 3 Build Log

Manifest transcendence rows: 5 planned, 5 built. Phase 3 complete.

## Generated (Priority 0/1)
- 110-op enriched OpenAPI -> data layer + endpoint commands (deals CRUD, reporting, self-serve activation, media-planning forecasting, reference data).
- Framework: sync/search/sql/analytics/tail/doctor, OAuth2 client-credentials auth (EQUATIV_CLIENT_ID/SECRET -> login.eqtv.io, Bearer cache), thin MCP search+execute surface (>50 tools), configurable EQUATIV_MAESTRO_BASE_URL.

## Transcendence (Priority 2 — all 5 hand-built + unit-tested)
1. `deals drift` — pacing-drift sweep vs prior snapshot (custom deal_pacing_snapshots table); typed exit 3 on --threshold breach; baseline-on-first-run. data-source: live.
2. `forecast sweep` — cartesian geo×format×audience avails grid via POST /report/inventoryInsights, throttled, persisted (forecast_sweep_runs/cells); --diff vs prior run; --max-cells cap. data-source: live. (inventoryInsights body approximate; flagged in Long + dry-run prints body.)
3. `reconcile` — local joins across deals/campaigns/lineitems; surfaces active-under-delivering, regulator-disabled, orphan line items, empty campaigns; --spend budget checks; honestly reports skipped checks for unsynced resources. data-source: local.
4. `deals apply <file>` — staged bulk deal writes; local validation + dry-run default + quota ledger (deals_apply_quota, 15 creates/500 updates/day caps). data-source: live. (deal write body approximate; flagged.)
5. `deals funnel-rank` — parallel (FanoutRun, conc 5) per-deal bid-funnel win-rate ranking with partial-failure accounting (fetch_failures). data-source: live.

Store migrations: internal/store/equativ-maestro_migrations.go (snapshots/sweep/quota tables). Helpers: internal/cli/novel_helpers.go.

## Deferred / approximate (honest notes)
- inventoryInsights, troubleshooting, /report/deals/pacing request/response shapes parsed DEFENSIVELY (not captured during read-only sniff). Flagged in each command's Long. Verify with live API-User credentials before production write/forecast use.
- deals create/update body = approximate writable subset (DealInput); dry-run by default.
- `deals drift --since` flag parses but currently always uses latest-two snapshots (documented narrowing).

## Generator gaps found (retro candidates)
- Large-surface pattern marks resource commands Hidden:true; when a hidden resource (deals) hosts hand-built novel commands, they become undiscoverable in --help. Hand-fix: un-hid deals.go (Hidden:false). Generator should auto-unhide resources that host novel/extra commands.
