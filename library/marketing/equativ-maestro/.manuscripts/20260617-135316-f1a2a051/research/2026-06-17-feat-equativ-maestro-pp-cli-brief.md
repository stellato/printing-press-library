# Equativ Maestro CLI Brief

## API Identity
- **Domain:** Programmatic advertising / media buying (curation platform). Maestro by Equativ is an end-to-end curation platform connecting media buyers (agencies, advertisers) with publisher supply via the Equativ Monetization Platform.
- **Users:** Programmatic traders, media planners, agency buyers, advertiser in-house teams.
- **Data profile:** Campaigns, line items, creatives, advertisers, PMP deals, inventory/supply, audience & semantic segments, reports (delivery/pacing/metrics/troubleshooting/insights), reference data (countries, cities, regions, DMAs, platforms, partners, inventory types).
- **Backend:** Documented **Demand API** — OpenAPI 3.0.3, base `https://demand-api.eqtv.io`, Scalar docs at `https://demand-api.eqtv.io/help/`, spec at `https://demand-api.eqtv.io/openapi/Demand(demand-api.eqtv.io)` (1.13 MB, 102 operations, 83 schemas).

## Auth
- **4 schemes (OR):** `oAuth Login` (authorizationCode, client_id `Maestro`, login `https://oauth.smartadserverapis.com/login`), `oAuth API User` (clientCredentials, token `https://oauth.smartadserverapis.com/oauth2/token`), `X-CompanyId` (apiKey header), `bearerAuth` (HTTP bearer JWT).
- **Documented machine path (help center):** API User → `POST https://login.eqtv.io/oauth2/token` (`application/x-www-form-urlencoded`, `grant_type=client_credentials&client_id=…&client_secret=…`) → Bearer JWT (TTL 600s) → `Authorization: Bearer <token>` + `X-CompanyId`.
- **User's situation:** Has a browser login (interactive OAuth Login). Unsure about API User (client-credentials) creds. Wants Chrome-MCP browser-sniff to capture the live auth + endpoints. → Sniff will reveal the real token source + X-CompanyId; CLI auth will support client-credentials (primary) and a captured-bearer fallback.
- **Rate limits:** auth 10 / 5 min; deal updates 500/day; deal creations 15/day.

## Reachability Risk
- **Low.** Documented OpenAPI live and fetchable (200). API host `demand-api.eqtv.io` reachable. Auth required (expected). Bot-protection: app is an Angular SPA behind OAuth login; API is standard HTTPS JSON.

## Top Workflows
1. **Plan supply / forecast availability** — set targeting (geo, device, format, audience/contextual), get availability estimates (`POST /report/inventoryInsights`, `POST /deal/inventoryinsights`, `POST /filter-suggestions`).
2. **Create & manage PMP deals** — discover supply → create deal → get deal ID → traffic in DSP. (`/deals` CRUD — internal API, to be browser-sniffed.)
3. **Activate campaigns** — self-serve campaigns/line-items/creatives/advertisers CRUD + budgets + trackers (`/selfServe/*`).
4. **Report & troubleshoot** — delivery/pacing/metrics for campaigns/deals/line-items, instant insights, troubleshooting, async/scheduled reports (`/report/*`, `/async-report/*`).
5. **Look up reference data** — countries, cities, regions, DMAs, platforms, partners, inventory types, IAB categories, TCF purposes/vendors.

## Table Stakes
- List/get/create/update/delete for campaigns, line items, creatives, advertisers, deals.
- Run inventory-insights forecasts with targeting filters.
- Pull delivery/pacing/metrics reports; export.
- Reference-data lookups for building targeting.
- OAuth2 client-credentials auth with token caching/refresh.

## Data Layer
- **Primary entities:** campaigns, lineItems, creatives, advertisers, deals, asyncReports, + reference tables (countries, cities, states, regions, dmas, platforms, partners, inventoryTypes, businessUnits, creativeBannerSizes).
- **Sync cursor:** `createdOrModifiedSince` (UTC) on list endpoints; pagination via `limit`/`offset`, `X-Pagination-Total-Count`, RFC5988 `Link`.
- **FTS/search:** local full-text over deals/campaigns/line-items names; `extendedSearch` upstream.

## Source Priority
- Single product (Maestro), two surfaces:
  - **Primary spec:** documented Demand API OpenAPI (102 ops) — reporting, self-serve activation, forecasting, reference data.
  - **Enrichment (browser-sniff, chrome-MCP):** internal `/deals` CRUD (PMP creation) + any media-planning UI endpoints not in the public spec + confirm live auth headers.
- Auth: both surfaces share the same Equativ OAuth2 bearer + X-CompanyId.

## User Vision
- User chose "the Maestro app itself," has a browser login, and explicitly requested Chrome-MCP click-and-sniff of the live media-planning session. Deal-creation coverage is a priority (chose browser-sniff for the /deals gap).

## Product Thesis
- **Name:** Maestro CLI (`equativ-maestro-pp-cli`).
- **Why it should exist:** No CLI exists for Equativ Maestro. Traders live in a slow SPA for forecasting, deal creation, and pulling reports. A scriptable, agent-native CLI with a local store enables: bulk deal/campaign ops, scripted forecasts across targeting permutations, offline report diffing/pacing alerts, and reference-data lookups — none of which the web UI makes fast.

## Build Priorities
1. Data layer for campaigns, line items, creatives, advertisers, deals + reference tables; sync via `createdOrModifiedSince`.
2. Absorb the full documented Demand API (102 ops): reporting, self-serve activation, forecasting, reference data — all with `--json`, `--select`, `--dry-run`, typed exits.
3. Browser-sniff (chrome-MCP) the `/deals` CRUD + live auth; merge into the spec.
4. Transcendence: cross-entity offline analytics the SPA can't do (pacing drift, forecast permutation sweeps, deal/campaign reconciliation, targeting-coverage diffs).
