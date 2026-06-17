# Equativ Maestro — Browser-Sniff Discovery Report

**Backend:** chrome-MCP (drove user's logged-in Chrome session, fresh capture tab, closed after).
**Captured at:** 2026-06-17. **Session:** authenticated (the authenticated user (org redacted)).
**Approach:** Native CDP network monitor + Performance API for endpoint inventory; read-only authenticated `fetch()` (page-context, user's session token) for response schemas. App's HTTP client pre-bound `fetch`/`XHR` at bootstrap, bypassing page-context interceptors — native monitor + Performance API were the reliable capture path.

## 1. User Goal Flow
- **Goal:** Discover supply / forecast availability for targeting criteria, then browse & configure PMP deals.
- **Steps completed:** Media Planning (toggled CTV → fired `report/inventoryInsights`); Deals list (29 deals); Deal summary detail (`/deals/{id}`, `report/deals/metrics`, `buyers`); Reporting → Instant Insights (`report/instantInsights`, `report/dimensions`, `report/metrics`); Site lists (`domainLists/filters`).
- **Steps intentionally NOT taken:** No deal/campaign create/update/delete (live mutations in a production account avoided). Captured request *shapes* and read endpoints only.
- **Coverage:** Media-planning + deals + reporting + site-lists surfaces. Self-serve activation (campaigns/line-items/creatives) covered by the documented OpenAPI spec, not exercised live.

## 2. Hosts
| Host | Role | Calls |
|------|------|-------|
| `buyerconnectapis.smartadserver.com` | **Live API** (Maestro UI backend) | 92+ |
| `oauth.eqtv.io` / `login.eqtv.io` | OAuth token endpoint | auth |
| `demand-api.eqtv.io` | Documented API host (Scalar/OpenAPI; API-User path) | docs |
| pendo / datadog / mpulse / GA / statuspage | Telemetry (noise) | — |

## 3. Auth (verified)
- **Bearer JWT alone authenticates reads** — `GET /deals?limit=1` returned **200** with `Authorization: Bearer <jwt>` and NO `X-CompanyId`. `x-pagination-total-count: 29`.
- Token stored in browser localStorage (OIDC/MSAL style); minted via `oauth.eqtv.io` (app) — documented API-User flow is `POST https://login.eqtv.io/oauth2/token` (`grant_type=client_credentials`, `client_id`, `client_secret`) → Bearer JWT, TTL 600s.
- `X-CompanyId` header: optional for reads; likely required for some writes / multi-company users. Spec lists it as a separate (OR) apiKey scheme.
- **Runtime shape: standard HTTPS JSON.** No WAF/clearance/bot gate. Plain HTTP replay works — no resident browser needed.

## 4. Endpoints Discovered (live, ~50 unique paths on buyerconnectapis.smartadserver.com)
**Deals (the gap — NOT in public OpenAPI):**
`GET /deals` (`limit,offset,sortBy,sortAsc,isDealPlus,includeOnlyFavorites`), `GET /deals/{id}`, `GET /deals/{id}/kpiTrackers`, `GET /deals/cappingTimeframes`. (Help docs also document `POST /deals`, `PUT /deals`, `DELETE /deals/{id}`, `/deals/{id}/domainlists`, `/deals/countBreakdown`.)

**Media planning / forecasting:** `POST /report/inventoryInsights`.

**Reporting:** `POST /report/deals/metrics`, `GET /report/deals/delivery`, `GET /report/deals/pacing`, `POST /report/instantInsights`, `GET /report/dimensions`, `GET /report/metrics`, `GET /report/fieldRestrictionRules`, `GET /report/timeRestrictionRules`, `GET /fieldsCategory`.

**Reference data:** `/buyers`, `/partners`, `/dsps`, `/publishers`, `/companies`, `/company`, `/businessUnits`, `/countries`, `/currencies`, `/roles`, `/user`, `/impressionTypes`, `/inventoryTypes`, `/platforms`, `/identityProviders`, `/thirdPartyProviders`, `/creativeBannerSizes`, `/audienceSegmentProviders`, `/semanticSegmentProviders`, `/semanticSegmentTypes`, `/ctvCategory`, `/ctvChannel`, `/ctvDistributor`, `/ctvNetwork`, `/ctvOtt`, `/videoAdBreakTypes`, `/videoCompletionTargets`, `/videoDurationBuckets`, `/videoPlacementTypes`, `/videoPlayerSizeBuckets`, `/viewabilityTargets`, `/domainCategoryTypes`, `/domainCategoryUniverses`, `/domainLists/filters`, `/spoTargeting`, `/spoTrustedPublishers`, `/templates`, `/selfServe/noBidFilterReasons`, `/authorize`.

## 5. Deal schema (verified via GET /deals/{id}, 43 fields)
`id`(int, internal), `dealId`(str, public), `name`, `currency`, `beginDate`, `createdAt`, `updatedAt`, `price`(num), `pricingModel`(int), `grossPriceCpmInDealCurrency`(num), `typeId`, `priorityId`, `dealCriticalityId`, targeting: `buyerIds[]`, `countryCodes[]`, `countryIds[]`, `networkIds[]`, `impressionTypeIds[]`, `inventoryTypeIds[]`, `keywordIdsTargeting`, `audienceSegmentsTargeting`, `semanticSegmentsTargeting`, `domainCategorySegmentsTargeting`; costs: `customVendorCost`, `smartVendorCostRate`, `companyVendorCostRate`, `managementVendorCostRate`, `audienceSegmentsDataCost`, `semanticSegmentsDataCost`; flags: `isActive`, `isArchived`, `isCTV`, `isCookieless`, `isDealPlus`, `isDisabledByRegulator`, `isDspSyncEnabled`, `isEditable`, `isExclusive`, `isGreenPmp`, `isParentDeal`, `isUnderDelivering`, `createdBySupplyLegacyWorkflow`; `apiCompliance`(obj).

## 6. Relationship to documented OpenAPI (demand-api.eqtv.io, 102 ops)
- **Overlap:** `report/inventoryInsights`, `report/*`, reference data, `fieldsCategory`, `dimensions/metrics` — same paths, documented schemas apply.
- **Documented-only:** `selfServe/*` (campaigns, lineitems, creatives, advertisers, budgets, trackers, nativeAssets), `async-report/*`, `cities/states/regions/dmas`, `filter-suggestions` — the activation/self-serve side, not exercised by the media-planning UI.
- **Live-only (gap filled by sniff):** `/deals` CRUD + rich targeting reference (`ctv*`, `video*`, `spo*`, segment providers, `domainLists`, `currencies`, `identityProviders`, `templates`, `user`, `company`).

## 7. Generation plan
- **Primary spec:** documented Demand API OpenAPI (full schemas). Add `servers`. 
- **Enrichment:** deals CRUD + extra reference endpoints (with verified Deal schema).
- **Base URL:** configurable; default `https://demand-api.eqtv.io` (API-User host), alt `buyerconnectapis.smartadserver.com` (verified app host).
- **Auth:** OAuth2 client-credentials → Bearer (+ optional `X-CompanyId`).

## 8. Authentication Context
Authenticated chrome-MCP session (user's own Chrome). Token used in-page for read-only schema capture; **never persisted, returned, or written to any artifact** (harness credential filter active; only redacted field-shapes captured). Session state excluded from manuscript archiving.

## 9. Rate Limiting
No 429s. Auth rate limit (docs): 10/5min. Deal updates 500/day; deal creations 15/day.
