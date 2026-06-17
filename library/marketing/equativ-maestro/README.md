# Maestro by Equativ CLI

**Every Maestro deal, avails forecast, and report — scriptable, with a local database and offline analytics the web UI and official MCP server can't match.**

A command-line interface for Equativ's Maestro curation platform. It mirrors the full deal, media-planning, reporting, and activation surface, then adds what a stateless API mirror can't: a local SQLite store with sync and full-text search, pacing-drift detection (deals drift), avails permutation sweeps (forecast sweep), cross-entity reconciliation (reconcile), and quota-aware bulk deal writes (deals apply).

## Install

The recommended path installs both the `equativ-maestro-pp-cli` binary and the `pp-equativ-maestro` agent skill (Claude Code, Codex, Cursor, Gemini CLI, GitHub Copilot, and other agents supported by the upstream [`skills`](https://github.com/vercel-labs/skills) CLI) in one shot:

```bash
npx -y @mvanhorn/printing-press-library install equativ-maestro
```

For CLI only (no skill):

```bash
npx -y @mvanhorn/printing-press-library install equativ-maestro --cli-only
```

For skill only — installs the skill into the same agents as the default command above, but skips the CLI binary (use this to update or reinstall just the skill):

```bash
npx -y @mvanhorn/printing-press-library install equativ-maestro --skill-only
```

To constrain the skill install to one or more specific agents (repeatable — agent names match the [`skills`](https://github.com/vercel-labs/skills) CLI):

```bash
npx -y @mvanhorn/printing-press-library install equativ-maestro --agent claude-code
npx -y @mvanhorn/printing-press-library install equativ-maestro --agent claude-code --agent codex
```

### Without Node (Go fallback)

If `npx` isn't available (no Node, offline), install the CLI directly via Go (requires Go 1.26.4 or newer):

```bash
go install github.com/mvanhorn/printing-press-library/library/marketing/equativ-maestro/cmd/equativ-maestro-pp-cli@latest
```

This installs the CLI only — no skill.

### Pre-built binary

Download a pre-built binary for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/equativ-maestro-current). On macOS, clear the Gatekeeper quarantine: `xattr -d com.apple.quarantine <binary>`. On Unix, mark it executable: `chmod +x <binary>`.

<!-- pp-hermes-install-anchor -->
## Install for Hermes

Install the CLI binary first. The installer writes binaries to a per-user managed bin directory by default: `$HOME/.local/bin` on macOS/Linux and `%LOCALAPPDATA%\Programs\PrintingPress\bin` on Windows.

```bash
npx -y @mvanhorn/printing-press-library install equativ-maestro --cli-only
```

Then install the focused Hermes skill.

From the Hermes CLI:

```bash
hermes skills install mvanhorn/printing-press-library/cli-skills/pp-equativ-maestro --force
```

Inside a Hermes chat session:

```bash
/skills install mvanhorn/printing-press-library/cli-skills/pp-equativ-maestro --force
```

Restart the Hermes session or gateway if the newly installed skill is not visible immediately.

## Install for OpenClaw
Install both the CLI binary and the focused OpenClaw skill. The installer defaults binaries to a per-user bin directory (`$HOME/.local/bin` on macOS/Linux, `%LOCALAPPDATA%\Programs\PrintingPress\bin` on Windows):

```bash
npx -y @mvanhorn/printing-press-library install equativ-maestro --agent openclaw
```

Restart the OpenClaw session or gateway if the newly installed skill is not visible immediately.

## Use with Claude Desktop

This CLI ships an [MCPB](https://github.com/modelcontextprotocol/mcpb) bundle — Claude Desktop's standard format for one-click MCP extension installs (no JSON config required).

To install:

1. Download the `.mcpb` for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/equativ-maestro-current).
2. Double-click the `.mcpb` file. Claude Desktop opens and walks you through the install.
3. Fill in `EQUATIV_CLIENT_ID` when Claude Desktop prompts you.

Requires Claude Desktop 1.0.0 or later. Pre-built bundles ship for macOS Apple Silicon (`darwin-arm64`) and Windows (`amd64`, `arm64`); for other platforms, use the manual config below.

<details>
<summary>Manual JSON config (advanced)</summary>

If you can't use the MCPB bundle (older Claude Desktop, unsupported platform), install the MCP binary and configure it manually.


```bash
go install github.com/mvanhorn/printing-press-library/library/marketing/equativ-maestro/cmd/equativ-maestro-pp-mcp@latest
```

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "equativ-maestro": {
      "command": "equativ-maestro-pp-mcp",
      "env": {
        "EQUATIV_CLIENT_ID": "<your-key>"
      }
    }
  }
}
```

</details>

## Authentication

Maestro uses OAuth2 client-credentials. Create an API User in Maestro to get a client_id and client_secret, set EQUATIV_CLIENT_ID and EQUATIV_CLIENT_SECRET, then run 'equativ-maestro-pp-cli auth login'. The CLI exchanges them at login.eqtv.io for a short-lived Bearer token and caches it, refreshing automatically before expiry. Set EQUATIV_MAESTRO_BASE_URL to override the API host (default buyerconnectapis.smartadserver.com).

## Quick Start

```bash
# Check config and reachability before anything else (works without credentials).
equativ-maestro-pp-cli doctor --dry-run

# Pull your deals into the local store so search and analytics work offline.
equativ-maestro-pp-cli sync --resources deals

# Full-text search your synced deals.
equativ-maestro-pp-cli search "display" --type deals

# See which deals drifted off pace since the last sync.
equativ-maestro-pp-cli deals drift --threshold 10

```

## Unique Features

These capabilities aren't available in any other tool for this API.

### Local state that compounds
- **`deals drift`** — See which PMP deals drifted off pace since your last sync — one ranked view across your whole deal book.

  _Reach for this to answer 'what changed since last week' across many deals at once instead of opening each deal's reporting tab._

  ```bash
  equativ-maestro-pp-cli deals drift --threshold 10 --agent
  ```
- **`reconcile`** — Find structural problems across your book: deals with no line item, line items with zero delivery, budgets that don't match delivered spend.

  _Reach for this before a client review to catch orphans and budget mismatches the separate web screens never surface together._

  ```bash
  equativ-maestro-pp-cli reconcile --spend --agent
  ```
- **`deals funnel-rank`** — Rank every deal by its bid-funnel win rate (eligible to bid to win) so the worst performers surface first.

  _Reach for this to triage which deals to troubleshoot first instead of opening funnels one by one._

  ```bash
  equativ-maestro-pp-cli deals funnel-rank --csv
  ```

### Orchestration the web UI can't do
- **`forecast sweep`** — Run an avails forecast across a whole targeting matrix (geo x format x audience) in one command and get the grid.

  _Reach for this when planning supply: see where avails actually exist across the whole targeting surface, not one cell at a time._

  ```bash
  equativ-maestro-pp-cli forecast sweep --geo 250,276 --agent
  ```
- **`deals apply`** — Stage a batch of deal creates/updates from a file, validate locally, dry-run by default, and commit within the daily rate caps.

  _Reach for this for bulk activation or teardown so a rate-limit doesn't leave a half-applied batch mid-flight._

  ```bash
  equativ-maestro-pp-cli deals apply deals.json
  ```

## Recipes


### Agent-friendly deal list, narrowed fields

```bash
equativ-maestro-pp-cli deals list --agent --select dealId,name,price,isActive,isCTV
```

Return only the fields an agent needs from the deal list instead of the full 43-field object.

### Forecast avails across a targeting grid

```bash
equativ-maestro-pp-cli forecast sweep --geo 250,276 --device 2 --agent --select cells.geo,cells.device,cells.avails,cells.auctions
```

One command runs the whole matrix of inventoryInsights forecasts and returns a compact grid of avails per targeting cell.

### Weekly pacing-drift triage

```bash
equativ-maestro-pp-cli deals drift --threshold 10 --csv
```

Rank deals whose pacing moved more than 10% since the last sync and pipe the CSV into your spreadsheet.

### Pre-review reconciliation

```bash
equativ-maestro-pp-cli reconcile --spend --agent
```

Surface orphaned deals/line items and budget-vs-spend mismatches across the synced book before a client review.

### Dry-run a bulk deal batch

```bash
equativ-maestro-pp-cli deals apply new-deals.json
```

Validate and preview every create/update locally; nothing is written until you add --commit.

## Usage

Run `equativ-maestro-pp-cli --help` for the full command reference and flag list.

## Commands

### async-report

AsyncReport

- **`equativ-maestro-pp-cli async-report create`** - Create a new async report
- **`equativ-maestro-pp-cli async-report delete`** - Deletes an async report
- **`equativ-maestro-pp-cli async-report get`** - Get a single async report by id
- **`equativ-maestro-pp-cli async-report list`** - Get all async reports, with the ability to filter by id
- **`equativ-maestro-pp-cli async-report update`** - Updates an async report

### async-report-gcp

Manage async report gcp

- **`equativ-maestro-pp-cli async-report-gcp create`** - Create a new async report
- **`equativ-maestro-pp-cli async-report-gcp get`** - Get a single async report by id
- **`equativ-maestro-pp-cli async-report-gcp update`** - Updates an async report

### businessunits

BusinessUnits

- **`equativ-maestro-pp-cli businessunits get`** - Returns the RTB+ business unit with the given id.
- **`equativ-maestro-pp-cli businessunits get-all`** - Returns all the RTB+ business units.

### cities

Cities

- **`equativ-maestro-pp-cli cities get`** - Returns the city with the given id.
- **`equativ-maestro-pp-cli cities get-all`** - Returns all the cities.

### countries

Countries

- **`equativ-maestro-pp-cli countries get`** - Returns the country with the given id.
- **`equativ-maestro-pp-cli countries get-all`** - Returns all the countries.
- **`equativ-maestro-pp-cli countries get-by-iso-code`** - Returns the country with the given ISO 3166 code.

### creative-banner-sizes

CreativeBannerSizes

- **`equativ-maestro-pp-cli creative-banner-sizes`** - ## Get the list of creative banner size
The list is filtered by the bellow parameters

### deal

Deals

- **`equativ-maestro-pp-cli deal`** - Create a report scoped to the given deal's targeting.
Targeting properties are converted to report filters before the query is
forwarded to the reporting engine.

### deals

Deals

- **`equativ-maestro-pp-cli deals capping-timeframes`** - List deal frequency-capping timeframes.
- **`equativ-maestro-pp-cli deals count-breakdown`** - Get deal count breakdown (active/archived/etc).
- **`equativ-maestro-pp-cli deals create`** - Create a PMP deal (write; body shape approximate, verify live).
- **`equativ-maestro-pp-cli deals delete`** - Delete a PMP deal.
- **`equativ-maestro-pp-cli deals get`** - Get a PMP deal by internal id.
- **`equativ-maestro-pp-cli deals list`** - List PMP deals.
- **`equativ-maestro-pp-cli deals list-kpitrackers`** - Gets KPI tracking rule definitions.
- **`equativ-maestro-pp-cli deals update`** - Update a PMP deal (write; body shape approximate, verify live).
- **`equativ-maestro-pp-cli deals update-kpitrackers`** - Updates a KPI Tracker.

### dmas

Dmas

- **`equativ-maestro-pp-cli dmas get`** - Returns the DMA with the given id.
- **`equativ-maestro-pp-cli dmas get-all`** - Returns all the DMAs.

### fields-category

FieldsCategory

- **`equativ-maestro-pp-cli fields-category`** - ## Get all available fieldsCategory

### filter-suggestions

FilterSuggestions

- **`equativ-maestro-pp-cli filter-suggestions`** - Create

### inventory-types

InventoryTypes

- **`equativ-maestro-pp-cli inventory-types get`** - Get
- **`equativ-maestro-pp-cli inventory-types get-all`** - Get all

### partners

Partners

- **`equativ-maestro-pp-cli partners get`** - Returns the RTB+ partner with the given id
- **`equativ-maestro-pp-cli partners list`** - Returns all the RTB+ Partners

### platforms

Platforms

- **`equativ-maestro-pp-cli platforms get`** - Returns the platform with the given id.
- **`equativ-maestro-pp-cli platforms get-all`** - Returns all the platforms.

### regions

Regions

- **`equativ-maestro-pp-cli regions get`** - Returns the region with the given id.
- **`equativ-maestro-pp-cli regions get-all`** - Returns all the regions.

### report

Report

- **`equativ-maestro-pp-cli report create`** - Generates an inventory insights report
Report includes given metrics and dimensions on specified period, optional parameters allows to filter and sort
result.
- **`equativ-maestro-pp-cli report create-compare`** - ## Generate a compare report on two periods with your data
Report includes given metrics and dimensions on specified periods, optional parameters allows to filter and sort
result.\
All parameters are specified in the request body.
- **`equativ-maestro-pp-cli report create-deals`** - Generates a deal metrics report
Report includes StartDate, EndDate, Timezone , UseCaseId and DealIds allows to filter and sort
result.
- **`equativ-maestro-pp-cli report create-domainsorapp`** - Creates and returns a CSV report of the last day avails and average CPM if (authorized),
with publishers and domains dimensions.
- **`equativ-maestro-pp-cli report create-fieldrestrictionrules`** - Validate params
For MicroService calls if needed
- **`equativ-maestro-pp-cli report create-instantinsights`** - Generates a report with your data
Report includes given metrics and dimensions on specified period, optional parameters allows to filter and sort
result.
- **`equativ-maestro-pp-cli report create-inventoryinsights`** - Generates an inventory insights report
Report includes given metrics and dimensions on specified period, optional parameters allows to filter and sort
result.
- **`equativ-maestro-pp-cli report create-publishers`** - Gets the list of publisher ids
Report includes StartDate, EndDate, DomainNameOrOutputBundleId, Timezone and Filters allows to filter and sort
result.
- **`equativ-maestro-pp-cli report list`** - ## Get all available dimension attributes
Only returns dimension attributes available for the authenticated user
- **`equativ-maestro-pp-cli report list-campaigns`** - List campaigns
- **`equativ-maestro-pp-cli report list-campaigns-2`** - List campaigns 2
- **`equativ-maestro-pp-cli report list-campaigns-3`** - List campaigns 3
- **`equativ-maestro-pp-cli report list-deals`** - List deals
- **`equativ-maestro-pp-cli report list-deals-2`** - List deals 2
- **`equativ-maestro-pp-cli report list-dimensions`** - ## Get all available dimensions
Only returns dimensions available for the authenticated user
- **`equativ-maestro-pp-cli report list-fieldrestrictionrules`** - Get the list of restriction rules
- **`equativ-maestro-pp-cli report list-lineitems`** - Generates line items report based on budgets
- **`equativ-maestro-pp-cli report list-lineitems-2`** - List lineitems 2
- **`equativ-maestro-pp-cli report list-lineitems-3`** - List lineitems 3
- **`equativ-maestro-pp-cli report list-metrics`** - ## Get all available metrics
Only returns metrics available for the authenticated user
- **`equativ-maestro-pp-cli report list-troubleshooting`** - List troubleshooting

### self-serve

Manage self serve

- **`equativ-maestro-pp-cli self-serve create`** - Create new Advertiser resource
- **`equativ-maestro-pp-cli self-serve create-selfserve`** - Create new CampaignBudget resource
- **`equativ-maestro-pp-cli self-serve create-selfserve-2`** - Create new Campaign resource
- **`equativ-maestro-pp-cli self-serve create-selfserve-3`** - Create new Creative resource
- **`equativ-maestro-pp-cli self-serve create-selfserve-4`** - Create new LineItemBudget resource
- **`equativ-maestro-pp-cli self-serve create-selfserve-5`** - Create new LineItem resource
- **`equativ-maestro-pp-cli self-serve create-selfserve-6`** - Create new Native Asset resource
- **`equativ-maestro-pp-cli self-serve create-selfserve-7`** - Create new Tracker resource
- **`equativ-maestro-pp-cli self-serve delete`** - Delete Advertiser resource
- **`equativ-maestro-pp-cli self-serve delete-selfserve`** - Delete CampaignBudget resource
- **`equativ-maestro-pp-cli self-serve delete-selfserve-2`** - Delete Campaign resource
- **`equativ-maestro-pp-cli self-serve delete-selfserve-3`** - Delete Creative resource
- **`equativ-maestro-pp-cli self-serve delete-selfserve-4`** - Delete LineItemBudget resource
- **`equativ-maestro-pp-cli self-serve delete-selfserve-5`** - Delete LineItem resource
- **`equativ-maestro-pp-cli self-serve delete-selfserve-6`** - Delete Native Asset resource
- **`equativ-maestro-pp-cli self-serve delete-selfserve-7`** - Delete Tracker resource
- **`equativ-maestro-pp-cli self-serve get`** - Get a single Advertiser resource
- **`equativ-maestro-pp-cli self-serve get-selfserve`** - Get a single CampaignBudget resource
- **`equativ-maestro-pp-cli self-serve get-selfserve-2`** - Get a single Campaign resource
- **`equativ-maestro-pp-cli self-serve get-selfserve-3`** - Get a single Creative resource
- **`equativ-maestro-pp-cli self-serve get-selfserve-4`** - Get a single LineItemBudget resource
- **`equativ-maestro-pp-cli self-serve get-selfserve-5`** - Get a single LineItem resource
- **`equativ-maestro-pp-cli self-serve get-selfserve-6`** - Get a single Native Asset resource
- **`equativ-maestro-pp-cli self-serve get-selfserve-7`** - Get a single Tracker resource
- **`equativ-maestro-pp-cli self-serve get-selfserve-8`** - Get a single CampaignHistory resource
- **`equativ-maestro-pp-cli self-serve get-selfserve-9`** - Get a single LineItemHistory resource
- **`equativ-maestro-pp-cli self-serve list`** - Get all Advertiser resources
- **`equativ-maestro-pp-cli self-serve list-selfserve`** - Get all CampaignBudget resources
- **`equativ-maestro-pp-cli self-serve list-selfserve-10`** - Get all Tracker resources
- **`equativ-maestro-pp-cli self-serve list-selfserve-2`** - Get all Campaign resources
- **`equativ-maestro-pp-cli self-serve list-selfserve-3`** - Get all Creative resources
- **`equativ-maestro-pp-cli self-serve list-selfserve-4`** - Get all IabCategory resources
- **`equativ-maestro-pp-cli self-serve list-selfserve-5`** - Get all LineItemBudget resources
- **`equativ-maestro-pp-cli self-serve list-selfserve-6`** - Get all LineItem resources
- **`equativ-maestro-pp-cli self-serve list-selfserve-7`** - Get all Native Asset resources
- **`equativ-maestro-pp-cli self-serve list-selfserve-8`** - Get all TcfPurpose resources
- **`equativ-maestro-pp-cli self-serve list-selfserve-9`** - Get all TcfVendor resources
- **`equativ-maestro-pp-cli self-serve update`** - Update Advertiser resource
- **`equativ-maestro-pp-cli self-serve update-selfserve`** - Update CampaignBudget resource
- **`equativ-maestro-pp-cli self-serve update-selfserve-2`** - Update Campaign resource
- **`equativ-maestro-pp-cli self-serve update-selfserve-3`** - Update Creative resource
- **`equativ-maestro-pp-cli self-serve update-selfserve-4`** - Update LineItemBudget resource
- **`equativ-maestro-pp-cli self-serve update-selfserve-5`** - Update LineItem resource
- **`equativ-maestro-pp-cli self-serve update-selfserve-6`** - Update Native Asset resource
- **`equativ-maestro-pp-cli self-serve update-selfserve-7`** - Update Tracker resource

### states

States

- **`equativ-maestro-pp-cli states get`** - Returns the state with the given id.
- **`equativ-maestro-pp-cli states get-all`** - Returns all the states.


## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
equativ-maestro-pp-cli async-report list

# JSON for scripting and agents
equativ-maestro-pp-cli async-report list --json

# Filter to specific fields
equativ-maestro-pp-cli async-report list --json --select id,name,status

# Dry run — show the request without sending
equativ-maestro-pp-cli async-report list --dry-run

# Agent mode — JSON + compact + no prompts in one flag
equativ-maestro-pp-cli async-report list --agent
```

## Agent Usage

This CLI is designed for AI agent consumption:

- **Non-interactive** - never prompts, every input is a flag
- **Pipeable** - `--json` output to stdout, errors to stderr
- **Filterable** - `--select id,name` returns only fields you need
- **Previewable** - `--dry-run` shows the request without sending
- **Explicit retries** - add `--idempotent` to create retries and `--ignore-missing` to delete retries when a no-op success is acceptable
- **Confirmable** - `--yes` for explicit confirmation of destructive actions
- **Piped input** - write commands can accept structured input when their help lists `--stdin`
- **Offline-friendly** - sync/search commands can use the local SQLite store when available
- **Agent-safe by default** - no colors or formatting unless `--human-friendly` is set

Exit codes: `0` success, `2` usage error, `3` not found, `4` auth error, `5` API error, `7` rate limited, `10` config error.

## Health Check

```bash
equativ-maestro-pp-cli doctor
```

Verifies configuration, credentials, and connectivity to the API.

## Configuration

Config file: `~/.config/equativ-maestro-pp-cli/config.toml`

Static request headers can be configured under `headers`; per-command header overrides take precedence.

Environment variables:

| Name | Kind | Required | Description |
| --- | --- | --- | --- |
| `EQUATIV_CLIENT_ID` | per_call | Yes | Set to your API credential. |
| `EQUATIV_CLIENT_SECRET` | per_call | Yes | Set to your API credential. |

### agentcookie (optional)

If you use agentcookie to sync secrets across machines, this CLI auto-adopts agentcookie-managed credentials with no extra setup. When the daemon writes to this CLI's config, `equativ-maestro-pp-cli doctor` reports `agentcookie: detected` and `auth-status` labels the source as `agentcookie`. Skip this section if you don't use agentcookie - the CLI works the same as any other.

## Troubleshooting
**Authentication errors (exit code 4)**
- Run `equativ-maestro-pp-cli doctor` to check credentials
- Verify the environment variable is set: `echo $EQUATIV_CLIENT_ID`
**Not found errors (exit code 3)**
- Check the resource ID is correct
- Run the `list` command to see available items

### API-specific
- **401 Unauthorized on every call** — Run 'equativ-maestro-pp-cli auth login' to mint a fresh token; the Bearer token is short-lived (per the server's expires_in) and the CLI auto-refreshes it before expiry.
- **429 / 'deal creations limit reached'** — The API caps deal creations at 15/day and updates at 500/day. Use 'deals apply' which counts against the cap and stops before breaching it.
- **Some reference or activation commands 404** — Set EQUATIV_MAESTRO_BASE_URL=https://demand-api.eqtv.io if your API User is provisioned on the documented Demand API host instead of the default live host.
- **deals drift returns nothing** — drift needs at least two syncs to compare. Run 'sync --resources deals' now and again later, then re-run drift.

## Known Gaps

This CLI was generated from Equativ's documented Demand API spec, with the
media-planning surfaces **verified live against the production Maestro API**
during a read-only discovery pass:

- **`forecast sweep`** — the `POST /report/inventoryInsights` request body
  (`metrics:["impressions","auctions"]`, nested `filters:[[{field,operator:"IN",values}]]`,
  `useCaseId:"ForecastDmkp"`) and the `[{impressions,auctions,day}]` response are
  **verified live**. `--geo`/`--device`/`--audience` take **numeric IDs**
  (`countryId`/`deviceTypeId`/`audienceSegmentId` — get country IDs from
  `equativ-maestro-pp-cli countries`). avails = summed impressions.
- **`deals drift`** — pacing is fetched by the deal's **internal id** via
  `GET /report/deals/pacing` (verified). The response envelope is confirmed; the
  exact per-row pacing field label is parsed defensively.
- **`deals funnel-rank`** — uses `GET /report/troubleshooting?RuleId=<internal id>`
  returning `{breakdown:[...]}` (verified). Per-funnel-stage labels are parsed
  defensively.
- **`deals apply` (create/update)** — the deal write body is an approximate
  writable subset (`DealInput`) and was **not** exercised live (no deals were
  created during discovery). It is **dry-run by default**; only `--commit` writes.
  Confirm the create/update payload against the live API before committing real deals.
- **`reconcile`** cross-resource checks (orphan line items, empty campaigns, `--spend`)
  run only when `campaigns`/`lineitems`/budgets are synced; otherwise they self-report
  as skipped rather than failing.
- **Two API hosts** — default base URL is the verified `buyerconnectapis.smartadserver.com`;
  some activation endpoints may live on `demand-api.eqtv.io`. Override with
  `EQUATIV_MAESTRO_BASE_URL`.
