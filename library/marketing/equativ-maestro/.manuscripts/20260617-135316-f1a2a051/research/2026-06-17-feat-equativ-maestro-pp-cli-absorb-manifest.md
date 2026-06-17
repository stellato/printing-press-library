# Equativ Maestro CLI — Absorb Manifest

## Competitive landscape
- **Official Maestro MCP server** (`maestro.mcp.eqtv.io`, HTTP, OAuth/Bearer) — the only existing programmatic/agent tool. Capabilities: deal search/retrieve/create/update + budgets + delivery troubleshooting + inventory estimate; campaign/line-item/advertiser/creative management; media-planning targeting (geo/audience/reference/video); reporting (delivery/pacing/metrics, custom RTB reports, comparison, dimensions/metrics).
- **Demand/Auction-Package REST API** (`buyerconnectapis.smartadserver.com`, verified; `demand-api.eqtv.io` documented) — 110 endpoints. No CLI exists.
- No npm/PyPI/community wrapper, no Claude plugin, no competing CLI. (Equativ's GitHub is mobile Display-SDK samples only.)

**Thesis:** Match every capability of the official Maestro MCP server, then beat it with what an MCP API-mirror structurally cannot do — a local SQLite store, bulk/scripted operations, offline cross-entity analytics, and forecast/pacing intelligence.

## Absorbed (match or beat everything the MCP server + API expose)

| # | Feature | Best Source | Our Implementation | Added Value |
|---|---------|-------------|--------------------|-------------|
| 1 | Search deals (by buyer/publisher/partner/country/name/pricing) | Maestro MCP `search deals` | `equativ-maestro-pp-cli deals list --buyer-ids … --extended-search …` (generated endpoint) | Offline FTS over synced deals, `--json`/`--select`/`--csv`, composes with jq |
| 2 | Retrieve deal by internal/public ID | Maestro MCP `retrieve deal` | `(generated endpoint) deals getDeal` | Local cache, typed output, `--select` field narrowing |
| 3 | Create deal (pricing + targeting) | Maestro MCP `create deal` | `(generated endpoint) deals createDeal` | `--dry-run`, `--stdin`, scriptable, typed exit codes |
| 4 | Update deal (name/price/targeting/status) | Maestro MCP `update deal` | `(generated endpoint) deals updateDeal` | Bulk update from file, `--dry-run`, idempotent |
| 5 | Delete deal | Auction Package API | `(generated endpoint) deals deleteDeal` | `--dry-run`, confirmation, typed exits |
| 6 | Deal count breakdown | Auction Package API | `(generated endpoint) deals dealsCountBreakdown` | Offline, agent-native JSON |
| 7 | Deal capping timeframes | Auction Package API | `(generated endpoint) deals dealCappingTimeframes` | Reference cache |
| 8 | Deal KPI trackers | Demand API | `(generated endpoint) deals getDealKpiTrackers` | Local persistence for trend tracking |
| 9 | Manage deal budgets (caps + pacing) | Maestro MCP `deal budgets` | `(generated endpoint) deals kpiTrackers / budgets` | Scriptable bulk capping |
| 10 | Estimate available inventory (avails forecast) | Maestro MCP `estimate inventory` | `(generated endpoint) report inventoryInsights` | Persist forecasts, diff over time |
| 11 | Deal-targeting inventory insights | Demand API `deal/inventoryinsights` | `(generated endpoint) deal inventoryinsights` | Offline forecast snapshots |
| 12 | Filter suggestions (targeting autocomplete) | Demand API `filter-suggestions` | `(generated endpoint) filter-suggestions` | Scripted targeting build |
| 13 | Delivery troubleshooting (bid funnel) | Maestro MCP `troubleshoot delivery` | `(generated endpoint) report troubleshooting` | Offline funnel snapshots, alerts |
| 14 | Real-time delivery metrics (deals/campaigns/line items) | Maestro MCP `delivery metrics` | `(generated endpoint) report {deals,campaigns,lineItems} delivery/metrics` | Local store → offline pacing math |
| 15 | Pacing vs targets | Maestro MCP `track pacing` | `(generated endpoint) report */pacing` | Persisted pacing history |
| 16 | Custom RTB reports (any dimension/metric) | Maestro MCP `custom reports` | `(generated endpoint) report (POST)` | Scriptable report jobs, `--csv` export |
| 17 | Period-over-period comparison | Maestro MCP `comparison reports` | `(generated endpoint) report compare` | Offline diffing |
| 18 | Instant insights | Demand API `report/instantInsights` | `(generated endpoint) report instantInsights` | Cached insight snapshots |
| 19 | Publisher / domain-or-app reports | Demand API | `(generated endpoint) report {publishers,domainsOrApp}` | Bulk export |
| 20 | Report dimensions / metrics / field categories | Maestro MCP `available dimensions` | `(generated endpoint) report dimensions/metrics`, `fieldsCategory` | Reference cache for report building |
| 21 | Async / scheduled reports CRUD + executions + preview | Demand API `async-report/*` | `(generated endpoint) async-report …` | Manage scheduled reports from CLI |
| 22 | Campaigns CRUD + history | Maestro MCP `campaigns` | `(generated endpoint) selfServe campaigns …` | Local store, bulk ops, `--dry-run` |
| 23 | Line items CRUD + history | Maestro MCP `line items` | `(generated endpoint) selfServe lineitems …` | Bulk, offline join to campaigns |
| 24 | Advertisers CRUD | Maestro MCP `advertisers` | `(generated endpoint) selfServe advertisers …` | Local store |
| 25 | Creatives CRUD + banner create | Maestro MCP `creatives` | `(generated endpoint) selfServe creatives …` | Bulk, list-by-advertiser |
| 26 | Campaign / line-item budgets CRUD | Maestro MCP `budgets` | `(generated endpoint) selfServe campaignBudgets/lineItemBudgets …` | Scriptable |
| 27 | Trackers, native assets, IAB categories, TCF | Demand API `selfServe/*` | `(generated endpoint) selfServe {trackers,nativeAssets,iabCategories,tcfPurposes,tcfVendors}` | Reference + management |
| 28 | Geo reference (countries/states/cities/regions/DMAs) | Maestro MCP `geo search` | `(generated endpoint) {countries,cities,states,regions,dmas}` | Offline targeting lookups, FTS |
| 29 | Audience/semantic segment providers + types | Maestro MCP `audience search` | `(generated endpoint) {audienceSegmentProviders,semanticSegmentProviders,semanticSegmentTypes}` | Local segment catalog |
| 30 | Platform / inventory / impression / banner reference | Maestro MCP `reference data` | `(generated endpoint) {platforms,inventoryTypes,impressionTypes,creativeBannerSizes,partners,buyers,publishers,dsps}` | Offline reference cache |
| 31 | Video targeting reference | Maestro MCP `video` | `(generated endpoint) {videoPlacementTypes,videoPlayerSizeBuckets,videoCompletionTargets,videoDurationBuckets,viewabilityTargets}` | Reference cache |
| 32 | SPO / sustainability options | Maestro MCP `SPO options` | `(generated endpoint) {spoTargeting,spoTrustedPublishers}` | Reference cache |
| 33 | OAuth2 client-credentials auth + token caching | Maestro MCP `OAuth/Bearer` | `equativ-maestro-pp-cli auth login` + cached token (10-min TTL refresh) | No manual token juggling; works headless |
| 34 | Offline sync + full-text search + SQL | (none — MCP has no store) | `equativ-maestro-pp-cli sync` / `search` / `sql` | The entire local-store layer MCP cannot offer |

> Generated endpoint commands above are emitted by the Printing Press from the 110-op enriched OpenAPI spec; the framework supplies `sync`/`search`/`sql`/`doctor`/`--json`/`--select`/`--dry-run`/typed exits for free.

## Transcendence (built in Phase 3 — hand-code, 5 features)

| # | Feature | Command | Score | Buildability | Why Only We Can Do This | Long Description |
|---|---------|---------|-------|--------------|------------------------|------------------|
| 1 | Pacing drift sweep | `deals drift [--since <date>] [--threshold <pct>]` | 8/10 | hand-code | Diffs each deal's latest delivery/pacing snapshot vs the prior synced snapshot in local SQLite; the SPA/MCP keep no history | Use for "what changed since last week" — the SAME deal's pacing across two synced points. For point-in-time orphans/mismatches use `reconcile`. |
| 2 | Forecast permutation sweep | `forecast sweep --geo … --format … --audience … [--diff <run-id>]` | 9/10 | hand-code | Expands a targeting matrix to the cartesian set, calls inventoryInsights once per cell with one cached token + throttle (respects auth 10/5min), persists every avails result; SPA/MCP do one forecast per call | Use to GENERATE the avails grid (live), and with `--diff` to compare two stored runs locally. Only command that runs forecasts. |
| 3 | Cross-entity reconciliation | `reconcile [--spend]` | 8/10 | hand-code | Four-way local join deals↔campaigns↔line items↔latest reports to surface orphans + budget/spend mismatch; no API endpoint spans them | Use for point-in-time structural integrity across entity types. For time-over-time pacing change on a single deal use `deals drift`. |
| 4 | Bulk deal writer + quota guard | `deals apply <file> [--commit]` | 7/10 | hand-code | Validates a batch against synced reference tables, dry-runs by default, counts against the 15-create/500-update daily caps, stops before the cap | none |
| 5 | Bid-funnel win-rate ranking | `deals funnel-rank` | 6/10 | hand-code | Loops the locally-stored deal set, pulls each deal's bid-funnel, computes win-rate, ranks worst-first; ranking needs the local deal set the MCP lacks | Use to rank ALL deals by funnel win-rate. One deal's raw funnel = absorbed `report troubleshooting`; pacing over time = `deals drift`. |

**Hand-code commitment:** 5 of 5 transcendence rows are `hand-code` (~90–150 LoC each + root.go wiring). Full audit trail (customer model, 12 candidates, 7 kills) in `2026-06-17-novel-features-brainstorm.md`.
