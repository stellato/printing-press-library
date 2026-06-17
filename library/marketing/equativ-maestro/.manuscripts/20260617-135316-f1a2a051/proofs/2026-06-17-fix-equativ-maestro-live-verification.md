# Live API Verification (post-publish, via browser session)

The operator has a Maestro browser login but no admin/API-User access. Using their
live authenticated session (read-only fetches against buyerconnectapis.smartadserver.com),
the 3 novel commands' request/response contracts were verified and corrected:

## forecast sweep (POST /report/inventoryInsights) — VERIFIED
- Working body: `{startDate,endDate, metrics:["impressions","auctions"], dimensions:[], filters:[[{field,operator:"IN",values:[<id>]}]], useCaseId:"ForecastDmkp", timezone:"UTC", currency:"EUR", periodicity:"day"}` → HTTP 200.
- Response: `[{impressions,auctions,day:<epoch_ms>}]`. avails = Σ impressions. FR(countryId 250)=12.5M vs unfiltered=9.15B (filter confirmed working).
- Fixes: was `metrics:["avails"]` (403 invalid field); flags changed to numeric-ID `--geo`(countryId)/`--device`(deviceTypeId)/`--audience`(audienceSegmentId); removed `--format` (no verified field); dropped median_cpm (not in response), added auctions.

## deals drift (GET /report/deals/pacing) — VERIFIED
- `?ids=<internal id>&timezone=` → 200. Was passing public `dealId` → HTTP 400 "ids not valid". Fixed to internal `id`. Implemented `--since` baseline (was a no-op).

## deals funnel-rank (GET /report/troubleshooting) — VERIFIED
- `?RuleId=<internal id>&StartDate&EndDate&Timezone=UTC` → 200, response `{breakdown:[...]}`. Was passing `ids=<dealId>` → HTTP 403 "Use RuleId instead". Fixed.

## Greptile review fixes
- P1: help guard now gated on isTerminal (piped invocations run instead of printing help).
- P2: deals drift --since implemented (was registered-but-discarded).

## MCPB
- Force-added cmd/equativ-maestro-pp-mcp/ (the `.gitignore *-pp-mcp` rule silently ignored the MCP binary source dir).

Build/vet/test green; verify-skill + validate-narrative pass. deals apply create body remains approximate (no deals created during read-only discovery; dry-run by default).
