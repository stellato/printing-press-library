---
name: pp-equativ-maestro
description: "CLI for Maestro by Equativ — manage PMP deals, run avails forecasts, pull delivery/pacing reports, and reconcile a curation book, with a local store and offline analytics the web UI and official MCP server can't match. Trigger phrases: `list my Maestro deals`, `forecast avails for this targeting`, `which Maestro deals are off pace`, `reconcile my curation book`, `create a Maestro PMP deal`, `use equativ-maestro`, `run equativ-maestro`."
author: "Greg Stellato"
license: "Apache-2.0"
argument-hint: "<command> [args] | install cli|mcp"
allowed-tools: "Read Bash"
metadata:
  openclaw:
    requires:
      bins:
        - equativ-maestro-pp-cli
    install:
      - kind: go
        bins: [equativ-maestro-pp-cli]
        module: github.com/mvanhorn/printing-press-library/library/marketing/equativ-maestro/cmd/equativ-maestro-pp-cli
---

# Maestro by Equativ — Printing Press CLI

## Prerequisites: Install the CLI

This skill drives the `equativ-maestro-pp-cli` binary. **You must verify the CLI is installed before invoking any command from this skill.** If it is missing, install it first:

1. Install via the Printing Press installer. It defaults binaries to `$HOME/.local/bin` on macOS/Linux and `%LOCALAPPDATA%\Programs\PrintingPress\bin` on Windows:
   ```bash
   npx -y @mvanhorn/printing-press-library install equativ-maestro --cli-only
   ```
2. Verify: `equativ-maestro-pp-cli --version`
3. Ensure the reported install directory is on `$PATH` for the agent/runtime that will invoke this skill.

If the `npx` install fails (no Node, offline, etc.), fall back to a direct Go install (requires Go 1.26.4 or newer). This installs into `$GOPATH/bin` (default `$HOME/go/bin`), so add that directory to `$PATH` instead:

```bash
go install github.com/mvanhorn/printing-press-library/library/marketing/equativ-maestro/cmd/equativ-maestro-pp-cli@latest
```

If `--version` reports "command not found" after install, the runtime cannot see the binary directory on `$PATH`. Do not proceed with skill commands until verification succeeds.

A command-line interface for Equativ's Maestro curation platform. It mirrors the full deal, media-planning, reporting, and activation surface, then adds what a stateless API mirror can't: a local SQLite store with sync and full-text search, pacing-drift detection (deals drift), avails permutation sweeps (forecast sweep), cross-entity reconciliation (reconcile), and quota-aware bulk deal writes (deals apply).

## When to Use This CLI

Use this CLI to script Equativ Maestro: bulk deal operations, avails forecasting across many targeting permutations, pulling delivery/pacing reports for offline analysis, and reconciling a curation book. It shines for agency traders, media planners, and ad-ops who live in spreadsheets and want the Maestro surface as composable commands with a local database.

## Anti-triggers

Do not use this CLI for:
- Do not use this CLI for ad creative rendering or the mobile Display SDK — that is a separate Equativ product.
- Do not use it for publisher/supply-side monetization (the EMP supply API) — this is the demand/buyer side only.
- Do not use it as a real-time bidder; it manages deals/campaigns/reports, it does not bid.

## Unique Capabilities

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

## Command Reference

**async-report** — AsyncReport

- `equativ-maestro-pp-cli async-report create` — Create a new async report
- `equativ-maestro-pp-cli async-report delete` — Deletes an async report
- `equativ-maestro-pp-cli async-report get` — Get a single async report by id
- `equativ-maestro-pp-cli async-report list` — Get all async reports, with the ability to filter by id
- `equativ-maestro-pp-cli async-report update` — Updates an async report

**async-report-gcp** — Manage async report gcp

- `equativ-maestro-pp-cli async-report-gcp create` — Create a new async report
- `equativ-maestro-pp-cli async-report-gcp get` — Get a single async report by id
- `equativ-maestro-pp-cli async-report-gcp update` — Updates an async report

**businessunits** — BusinessUnits

- `equativ-maestro-pp-cli businessunits get` — Returns the RTB+ business unit with the given id.
- `equativ-maestro-pp-cli businessunits get-all` — Returns all the RTB+ business units.

**cities** — Cities

- `equativ-maestro-pp-cli cities get` — Returns the city with the given id.
- `equativ-maestro-pp-cli cities get-all` — Returns all the cities.

**countries** — Countries

- `equativ-maestro-pp-cli countries get` — Returns the country with the given id.
- `equativ-maestro-pp-cli countries get-all` — Returns all the countries.
- `equativ-maestro-pp-cli countries get-by-iso-code` — Returns the country with the given ISO 3166 code.

**creative-banner-sizes** — CreativeBannerSizes

- `equativ-maestro-pp-cli creative-banner-sizes` — The list is filtered by the bellow parameters

**deal** — Deals

- `equativ-maestro-pp-cli deal` — Create a report scoped to the given deal's targeting.

**deals** — Deals

- `equativ-maestro-pp-cli deals capping-timeframes` — List deal frequency-capping timeframes.
- `equativ-maestro-pp-cli deals count-breakdown` — Get deal count breakdown (active/archived/etc).
- `equativ-maestro-pp-cli deals create` — Create a PMP deal (write; body shape approximate, verify live).
- `equativ-maestro-pp-cli deals delete` — Delete a PMP deal.
- `equativ-maestro-pp-cli deals get` — Get a PMP deal by internal id.
- `equativ-maestro-pp-cli deals list` — List PMP deals.
- `equativ-maestro-pp-cli deals list-kpitrackers` — Gets KPI tracking rule definitions.
- `equativ-maestro-pp-cli deals update` — Update a PMP deal (write; body shape approximate, verify live).
- `equativ-maestro-pp-cli deals update-kpitrackers` — Updates a KPI Tracker.

**dmas** — Dmas

- `equativ-maestro-pp-cli dmas get` — Returns the DMA with the given id.
- `equativ-maestro-pp-cli dmas get-all` — Returns all the DMAs.

**fields-category** — FieldsCategory

- `equativ-maestro-pp-cli fields-category` — Get all available fieldsCategory

**filter-suggestions** — FilterSuggestions

- `equativ-maestro-pp-cli filter-suggestions` — Create

**inventory-types** — InventoryTypes

- `equativ-maestro-pp-cli inventory-types get` — Get
- `equativ-maestro-pp-cli inventory-types get-all` — Get all

**partners** — Partners

- `equativ-maestro-pp-cli partners get` — Returns the RTB+ partner with the given id
- `equativ-maestro-pp-cli partners list` — Returns all the RTB+ Partners

**platforms** — Platforms

- `equativ-maestro-pp-cli platforms get` — Returns the platform with the given id.
- `equativ-maestro-pp-cli platforms get-all` — Returns all the platforms.

**regions** — Regions

- `equativ-maestro-pp-cli regions get` — Returns the region with the given id.
- `equativ-maestro-pp-cli regions get-all` — Returns all the regions.

**report** — Report

- `equativ-maestro-pp-cli report create` — Generates an inventory insights report Report includes given metrics and dimensions on specified period
- `equativ-maestro-pp-cli report create-compare` — Report includes given metrics and dimensions on specified periods, optional parameters allows to filter and sort result.
- `equativ-maestro-pp-cli report create-deals` — Generates a deal metrics report Report includes StartDate, EndDate, Timezone
- `equativ-maestro-pp-cli report create-domainsorapp` — Creates and returns a CSV report of the last day avails and average CPM if (authorized)
- `equativ-maestro-pp-cli report create-fieldrestrictionrules` — Validate params For MicroService calls if needed
- `equativ-maestro-pp-cli report create-instantinsights` — Generates a report with your data Report includes given metrics and dimensions on specified period
- `equativ-maestro-pp-cli report create-inventoryinsights` — Generates an inventory insights report Report includes given metrics and dimensions on specified period
- `equativ-maestro-pp-cli report create-publishers` — Gets the list of publisher ids Report includes StartDate, EndDate, DomainNameOrOutputBundleId
- `equativ-maestro-pp-cli report list` — Only returns dimension attributes available for the authenticated user
- `equativ-maestro-pp-cli report list-campaigns` — List campaigns
- `equativ-maestro-pp-cli report list-campaigns-2` — List campaigns 2
- `equativ-maestro-pp-cli report list-campaigns-3` — List campaigns 3
- `equativ-maestro-pp-cli report list-deals` — List deals
- `equativ-maestro-pp-cli report list-deals-2` — List deals 2
- `equativ-maestro-pp-cli report list-dimensions` — Only returns dimensions available for the authenticated user
- `equativ-maestro-pp-cli report list-fieldrestrictionrules` — Get the list of restriction rules
- `equativ-maestro-pp-cli report list-lineitems` — Generates line items report based on budgets
- `equativ-maestro-pp-cli report list-lineitems-2` — List lineitems 2
- `equativ-maestro-pp-cli report list-lineitems-3` — List lineitems 3
- `equativ-maestro-pp-cli report list-metrics` — Only returns metrics available for the authenticated user
- `equativ-maestro-pp-cli report list-troubleshooting` — List troubleshooting

**self-serve** — Manage self serve

- `equativ-maestro-pp-cli self-serve create` — Create new Advertiser resource
- `equativ-maestro-pp-cli self-serve create-selfserve` — Create new CampaignBudget resource
- `equativ-maestro-pp-cli self-serve create-selfserve-2` — Create new Campaign resource
- `equativ-maestro-pp-cli self-serve create-selfserve-3` — Create new Creative resource
- `equativ-maestro-pp-cli self-serve create-selfserve-4` — Create new LineItemBudget resource
- `equativ-maestro-pp-cli self-serve create-selfserve-5` — Create new LineItem resource
- `equativ-maestro-pp-cli self-serve create-selfserve-6` — Create new Native Asset resource
- `equativ-maestro-pp-cli self-serve create-selfserve-7` — Create new Tracker resource
- `equativ-maestro-pp-cli self-serve delete` — Delete Advertiser resource
- `equativ-maestro-pp-cli self-serve delete-selfserve` — Delete CampaignBudget resource
- `equativ-maestro-pp-cli self-serve delete-selfserve-2` — Delete Campaign resource
- `equativ-maestro-pp-cli self-serve delete-selfserve-3` — Delete Creative resource
- `equativ-maestro-pp-cli self-serve delete-selfserve-4` — Delete LineItemBudget resource
- `equativ-maestro-pp-cli self-serve delete-selfserve-5` — Delete LineItem resource
- `equativ-maestro-pp-cli self-serve delete-selfserve-6` — Delete Native Asset resource
- `equativ-maestro-pp-cli self-serve delete-selfserve-7` — Delete Tracker resource
- `equativ-maestro-pp-cli self-serve get` — Get a single Advertiser resource
- `equativ-maestro-pp-cli self-serve get-selfserve` — Get a single CampaignBudget resource
- `equativ-maestro-pp-cli self-serve get-selfserve-2` — Get a single Campaign resource
- `equativ-maestro-pp-cli self-serve get-selfserve-3` — Get a single Creative resource
- `equativ-maestro-pp-cli self-serve get-selfserve-4` — Get a single LineItemBudget resource
- `equativ-maestro-pp-cli self-serve get-selfserve-5` — Get a single LineItem resource
- `equativ-maestro-pp-cli self-serve get-selfserve-6` — Get a single Native Asset resource
- `equativ-maestro-pp-cli self-serve get-selfserve-7` — Get a single Tracker resource
- `equativ-maestro-pp-cli self-serve get-selfserve-8` — Get a single CampaignHistory resource
- `equativ-maestro-pp-cli self-serve get-selfserve-9` — Get a single LineItemHistory resource
- `equativ-maestro-pp-cli self-serve list` — Get all Advertiser resources
- `equativ-maestro-pp-cli self-serve list-selfserve` — Get all CampaignBudget resources
- `equativ-maestro-pp-cli self-serve list-selfserve-10` — Get all Tracker resources
- `equativ-maestro-pp-cli self-serve list-selfserve-2` — Get all Campaign resources
- `equativ-maestro-pp-cli self-serve list-selfserve-3` — Get all Creative resources
- `equativ-maestro-pp-cli self-serve list-selfserve-4` — Get all IabCategory resources
- `equativ-maestro-pp-cli self-serve list-selfserve-5` — Get all LineItemBudget resources
- `equativ-maestro-pp-cli self-serve list-selfserve-6` — Get all LineItem resources
- `equativ-maestro-pp-cli self-serve list-selfserve-7` — Get all Native Asset resources
- `equativ-maestro-pp-cli self-serve list-selfserve-8` — Get all TcfPurpose resources
- `equativ-maestro-pp-cli self-serve list-selfserve-9` — Get all TcfVendor resources
- `equativ-maestro-pp-cli self-serve update` — Update Advertiser resource
- `equativ-maestro-pp-cli self-serve update-selfserve` — Update CampaignBudget resource
- `equativ-maestro-pp-cli self-serve update-selfserve-2` — Update Campaign resource
- `equativ-maestro-pp-cli self-serve update-selfserve-3` — Update Creative resource
- `equativ-maestro-pp-cli self-serve update-selfserve-4` — Update LineItemBudget resource
- `equativ-maestro-pp-cli self-serve update-selfserve-5` — Update LineItem resource
- `equativ-maestro-pp-cli self-serve update-selfserve-6` — Update Native Asset resource
- `equativ-maestro-pp-cli self-serve update-selfserve-7` — Update Tracker resource

**states** — States

- `equativ-maestro-pp-cli states get` — Returns the state with the given id.
- `equativ-maestro-pp-cli states get-all` — Returns all the states.


### Finding the right command

When you know what you want to do but not which command does it, ask the CLI directly:

```bash
equativ-maestro-pp-cli which "<capability in your own words>"
```

`which` resolves a natural-language capability query to the best matching command from this CLI's curated feature index. Exit code `0` means at least one match; exit code `2` means no confident match — fall back to `--help` or use a narrower query.

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

## Auth Setup

Maestro uses OAuth2 client-credentials. Create an API User in Maestro to get a client_id and client_secret, set EQUATIV_CLIENT_ID and EQUATIV_CLIENT_SECRET, then run 'equativ-maestro-pp-cli auth login'. The CLI exchanges them at login.eqtv.io for a short-lived Bearer token and caches it, refreshing automatically before expiry. Set EQUATIV_MAESTRO_BASE_URL to override the API host (default buyerconnectapis.smartadserver.com).

Run `equativ-maestro-pp-cli doctor` to verify setup.

## Agent Mode

Add `--agent` to any command. Expands to: `--json --compact --no-input --no-color --yes`.

- **Pipeable** — JSON on stdout, errors on stderr
- **Filterable** — `--select` keeps a subset of fields. Dotted paths descend into nested structures; arrays traverse element-wise. Critical for keeping context small on verbose APIs:

  ```bash
  equativ-maestro-pp-cli async-report list --agent --select id,name,status
  ```
- **Previewable** — `--dry-run` shows the request without sending
- **Offline-friendly** — sync/search commands can use the local SQLite store when available
- **Non-interactive** — never prompts, every input is a flag
- **Explicit retries** — use `--idempotent` only when an already-existing create should count as success, and `--ignore-missing` only when a missing delete target should count as success

### Response envelope

Commands that read from the local store or the API wrap output in a provenance envelope:

```json
{
  "meta": {"source": "live" | "local", "synced_at": "...", "reason": "..."},
  "results": <data>
}
```

Parse `.results` for data and `.meta.source` to know whether it's live or local. A human-readable `N results (live)` summary is printed to stderr only when stdout is a terminal AND no machine-format flag (`--json`, `--csv`, `--compact`, `--quiet`, `--plain`, `--select`) is set — piped/agent consumers and explicit-format runs get pure JSON on stdout.

## Agent Feedback

When you (or the agent) notice something off about this CLI, record it:

```
equativ-maestro-pp-cli feedback "the --since flag is inclusive but docs say exclusive"
equativ-maestro-pp-cli feedback --stdin < notes.txt
equativ-maestro-pp-cli feedback list --json --limit 10
```

Entries are stored locally at `~/.local/share/equativ-maestro-pp-cli/feedback.jsonl`. They are never POSTed unless `EQUATIV_MAESTRO_FEEDBACK_ENDPOINT` is set AND either `--send` is passed or `EQUATIV_MAESTRO_FEEDBACK_AUTO_SEND=true`. Default behavior is local-only.

Write what *surprised* you, not a bug report. Short, specific, one line: that is the part that compounds.

## Output Delivery

Every command accepts `--deliver <sink>`. The output goes to the named sink in addition to (or instead of) stdout, so agents can route command results without hand-piping. Three sinks are supported:

| Sink | Effect |
|------|--------|
| `stdout` | Default; write to stdout only |
| `file:<path>` | Atomically write output to `<path>` (tmp + rename) |
| `webhook:<url>` | POST the output body to the URL (`application/json` or `application/x-ndjson` when `--compact`) |

Unknown schemes are refused with a structured error naming the supported set. Webhook failures return non-zero and log the URL + HTTP status on stderr.

## Named Profiles

A profile is a saved set of flag values, reused across invocations. Use it when a scheduled agent calls the same command every run with the same configuration - HeyGen's "Beacon" pattern.

```
equativ-maestro-pp-cli profile save briefing --json
equativ-maestro-pp-cli --profile briefing async-report list
equativ-maestro-pp-cli profile list --json
equativ-maestro-pp-cli profile show briefing
equativ-maestro-pp-cli profile delete briefing --yes
```

Explicit flags always win over profile values; profile values win over defaults. `agent-context` lists all available profiles under `available_profiles` so introspecting agents discover them at runtime.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 2 | Usage error (wrong arguments) |
| 3 | Resource not found |
| 4 | Authentication required |
| 5 | API error (upstream issue) |
| 7 | Rate limited (wait and retry) |
| 10 | Config error |

## Argument Parsing

Parse `$ARGUMENTS`:

1. **Empty, `help`, or `--help`** → show `equativ-maestro-pp-cli --help` output
2. **Starts with `install`** → ends with `mcp` → MCP installation; otherwise → see Prerequisites above
3. **Anything else** → Direct Use (execute as CLI command with `--agent`)

## MCP Server Installation

1. Install the MCP server:
   ```bash
   go install github.com/mvanhorn/printing-press-library/library/marketing/equativ-maestro/cmd/equativ-maestro-pp-mcp@latest
   ```
2. Register with Claude Code:
   ```bash
   claude mcp add equativ-maestro-pp-mcp -- equativ-maestro-pp-mcp
   ```
3. Verify: `claude mcp list`

## Direct Use

1. Check if installed: `which equativ-maestro-pp-cli`
   If not found, offer to install (see Prerequisites at the top of this skill).
2. Match the user query to the best command from the Unique Capabilities and Command Reference above.
3. Execute with the `--agent` flag:
   ```bash
   equativ-maestro-pp-cli <command> [subcommand] [args] --agent
   ```
4. If ambiguous, drill into subcommand help: `equativ-maestro-pp-cli <command> --help`.
