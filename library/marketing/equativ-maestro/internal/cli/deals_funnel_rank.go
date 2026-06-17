// Copyright 2026 Greg Stellato and contributors. Licensed under Apache-2.0. See LICENSE.

// pp:data-source live

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/mvanhorn/printing-press-library/library/marketing/equativ-maestro/internal/cliutil"
	"github.com/mvanhorn/printing-press-library/library/marketing/equativ-maestro/internal/store"

	"github.com/spf13/cobra"
)

// funnelRow is one deal's bid-funnel ranking result.
type funnelRow struct {
	DealID   string   `json:"dealId"`
	Name     string   `json:"name"`
	Eligible float64  `json:"eligible"`
	Bids     float64  `json:"bids"`
	Wins     float64  `json:"wins"`
	WinRate  *float64 `json:"win_rate"`
}

// winRate computes wins/eligible, guarding divide-by-zero. Returns (rate, true)
// normally, or (0, false) when eligible <= 0 so callers can render null.
func winRate(eligible, wins float64) (float64, bool) {
	if eligible <= 0 {
		return 0, false
	}
	return wins / eligible, true
}

// funnelStageProbe pulls (eligible, bids, wins) out of one breakdown row,
// searching candidate field names defensively because the exact stage labels
// inside a breakdown entry are not pinned by the live contract. Values are
// summed by the caller across all breakdown rows.
func funnelStageProbe(obj map[string]json.RawMessage) (eligible, bids, wins float64) {
	for _, key := range []string{"eligible", "auctions", "bidRequests", "bid_requests", "available_impressions", "availableImpressions"} {
		if v, ok := cliutil.ExtractNumber(obj, key); ok {
			eligible = v
			break
		}
	}
	for _, key := range []string{"bids", "bid", "bidResponses", "bid_responses"} {
		if v, ok := cliutil.ExtractNumber(obj, key); ok {
			bids = v
			break
		}
	}
	for _, key := range []string{"wins", "winningBids", "winning_bids", "impressions", "wonImpressions"} {
		if v, ok := cliutil.ExtractNumber(obj, key); ok {
			wins = v
			break
		}
	}
	return eligible, bids, wins
}

// parseFunnelStages extracts (eligible, bids, wins) from a troubleshooting
// response. The verified shape is {"breakdown":[ ... ]}: the funnel stages live
// in the breakdown array, which this sums across (an empty/absent breakdown
// yields 0/0/0 so the deal still ranks with nulls last). It also tolerates a
// bare array or a leading data/results envelope defensively.
func parseFunnelStages(data json.RawMessage) (eligible, bids, wins float64) {
	sumRows := func(arr []json.RawMessage) (float64, float64, float64) {
		var e, b, w float64
		for _, item := range arr {
			var row map[string]json.RawMessage
			if json.Unmarshal(item, &row) != nil {
				continue
			}
			re, rb, rw := funnelStageProbe(row)
			e, b, w = e+re, b+rb, w+rw
		}
		return e, b, w
	}

	var top map[string]json.RawMessage
	if json.Unmarshal(data, &top) == nil {
		// Verified shape: {"breakdown":[...]}.
		for _, key := range []string{"breakdown", "data", "results", "items", "rows"} {
			if inner, ok := top[key]; ok {
				var arr []json.RawMessage
				if json.Unmarshal(inner, &arr) == nil {
					return sumRows(arr)
				}
			}
		}
		// Single flat object as a last resort.
		return funnelStageProbe(top)
	}

	// Bare array response.
	var arr []json.RawMessage
	if json.Unmarshal(data, &arr) == nil {
		return sumRows(arr)
	}
	return 0, 0, 0
}

// rankFunnelRows sorts ascending by win_rate (worst first); rows with a null
// win_rate (undefined, divide-by-zero) sort to the end. Stable on dealId.
func rankFunnelRows(rows []funnelRow) {
	sort.SliceStable(rows, func(i, j int) bool {
		ri, rj := rows[i].WinRate, rows[j].WinRate
		switch {
		case ri == nil && rj == nil:
			return rows[i].DealID < rows[j].DealID
		case ri == nil:
			return false
		case rj == nil:
			return true
		}
		if *ri == *rj {
			return rows[i].DealID < rows[j].DealID
		}
		return *ri < *rj
	})
}

func newNovelDealsFunnelRankCmd(flags *rootFlags) *cobra.Command {
	var flagLimit int
	var flagMaxScan int
	var dbPath string

	cmd := &cobra.Command{
		Use:   "funnel-rank",
		Short: "Rank every deal by its bid-funnel win rate (eligible to bid to win) so the worst performers surface first.",
		Long: `Rank active deals by bid-funnel win rate (wins / eligible), worst performers first.

For each active synced deal this calls the troubleshooting report (keyed by the
deal's internal id as RuleId, over the last 30 days) and sums eligible / bids /
wins from the response breakdown array. win_rate is wins/eligible, or null when
eligible is zero (nulls rank last). Deals whose fetch fails are excluded from the
ranking and reported under fetch_failures.

This is a live command: it calls the troubleshooting report per deal, so
--data-source local is rejected.`,
		Example: `  # Rank the 20 worst deals by win rate
  equativ-maestro-pp-cli deals funnel-rank --json

  # Scan up to 100 deals, return the 10 worst
  equativ-maestro-pp-cli deals funnel-rank --max-scan 100 --limit 10 --json`,
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 && isTerminal(cmd.OutOrStdout()) {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				fmt.Fprintln(cmd.ErrOrStderr(), "would rank active deals by bid-funnel win rate (worst first)")
				return nil
			}
			if flags.dataSource == "local" {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("deals funnel-rank is a live command and cannot run with --data-source local; use live or auto"))
			}

			if dbPath == "" {
				dbPath = defaultDBPath("equativ-maestro-pp-cli")
			}
			if _, err := os.Stat(dbPath); os.IsNotExist(err) {
				fmt.Fprintf(cmd.ErrOrStderr(), "no local mirror at %s\nrun: equativ-maestro-pp-cli sync --resources deals --db %s\n", dbPath, dbPath)
				if flags.asJSON || flags.agent {
					fmt.Fprintln(cmd.OutOrStdout(), "[]")
				}
				return nil
			}

			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()

			db, err := store.OpenWithContext(ctx, dbPath)
			if err != nil {
				return fmt.Errorf("opening local database: %w", err)
			}
			defer db.Close()
			maybeEmitSyncHints(cmd, db, "deals", flags.maxAge)

			rawDeals, err := db.List("deals", 0)
			if err != nil {
				return fmt.Errorf("listing synced deals: %w", err)
			}

			type dealRef struct {
				dealID     string // public dealId — reported in output
				internalID string // internal numeric id — the troubleshooting RuleId
				name       string
			}
			active := make([]dealRef, 0, len(rawDeals))
			for _, rd := range rawDeals {
				var d map[string]json.RawMessage
				if json.Unmarshal(rd, &d) != nil {
					continue
				}
				if !rawBool(d["isActive"]) {
					continue
				}
				active = append(active, dealRef{
					dealID:     rawStringField(d, "dealId", "id"),
					internalID: rawStringField(d, "id"),
					name:       rawStringField(d, "name"),
				})
			}

			maxScan := flagMaxScan
			if cliutil.IsDogfoodEnv() {
				maxScan = 1
			}
			if maxScan > 0 && len(active) > maxScan {
				fmt.Fprintf(cmd.ErrOrStderr(), "note: scanning the first %d of %d active deals (--max-scan)\n", maxScan, len(active))
				active = active[:maxScan]
			}

			if len(active) == 0 {
				return emitFunnelEnvelope(cmd, flags, make([]funnelRow, 0), 0, make([]map[string]string, 0))
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			// Default the troubleshooting window to the last 30 days. This is
			// the printed CLI, not the workflow runtime, so time.Now() is fine.
			now := time.Now().UTC()
			startDate := now.AddDate(0, 0, -30).Format("2006-01-02")
			endDate := now.Format("2006-01-02")

			// Bounded fan-out over the troubleshooting endpoint per deal. RuleId
			// is single-valued (one request per deal); passing a DealId returns
			// HTTP 403 "Use RuleId instead". Param names are capitalized.
			results, fanoutErrs := cliutil.FanoutRun(
				ctx,
				active,
				func(d dealRef) string { return d.dealID },
				func(ctx context.Context, d dealRef) (funnelRow, error) {
					data, gerr := c.Get(ctx, "/report/troubleshooting", map[string]string{
						"RuleId":    d.internalID,
						"StartDate": startDate,
						"EndDate":   endDate,
						"Timezone":  "UTC",
					})
					if gerr != nil {
						return funnelRow{}, gerr
					}
					eligible, bids, wins := parseFunnelStages(data)
					row := funnelRow{DealID: d.dealID, Name: d.name, Eligible: eligible, Bids: bids, Wins: wins}
					if rate, ok := winRate(eligible, wins); ok {
						row.WinRate = &rate
					}
					return row, nil
				},
				cliutil.WithConcurrency(5),
			)

			rows := make([]funnelRow, 0, len(results))
			for _, r := range results {
				rows = append(rows, r.Value)
			}
			rankFunnelRows(rows)
			if flagLimit > 0 && len(rows) > flagLimit {
				rows = rows[:flagLimit]
			}

			fetchFailures := make([]map[string]string, 0, len(fanoutErrs))
			for _, fe := range fanoutErrs {
				fetchFailures = append(fetchFailures, map[string]string{"dealId": fe.Source, "error": fe.Err.Error()})
			}
			if len(fetchFailures) > 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: %d of %d deal funnel fetches failed\n", len(fetchFailures), len(active))
			}

			return emitFunnelEnvelope(cmd, flags, rows, len(active), fetchFailures)
		},
	}
	cmd.Flags().IntVar(&flagLimit, "limit", 20, "Maximum deals to return in the ranking (worst first)")
	cmd.Flags().IntVar(&flagMaxScan, "max-scan", 50, "Maximum deals to fetch funnels for")
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to the local SQLite mirror (default: per-user data dir)")
	return cmd
}

// emitFunnelEnvelope renders the funnel ranking as JSON or a human table.
func emitFunnelEnvelope(cmd *cobra.Command, flags *rootFlags, rows []funnelRow, scanned int, fetchFailures []map[string]string) error {
	if fetchFailures == nil {
		fetchFailures = make([]map[string]string, 0)
	}
	out := cmd.OutOrStdout()
	if flags.asJSON || (!isTerminal(out) && !flags.csv && !flags.quiet && !flags.plain) {
		envelope := map[string]any{
			"deals":          rows,
			"scanned_deals":  scanned,
			"fetch_failures": fetchFailures,
		}
		raw, err := json.Marshal(envelope)
		if err != nil {
			return err
		}
		return printOutput(out, raw, true)
	}
	if wantsHumanTable(out, flags) {
		items := make([]map[string]any, 0, len(rows))
		for _, r := range rows {
			m := map[string]any{"dealId": r.DealID, "name": r.Name, "eligible": r.Eligible, "bids": r.Bids, "wins": r.Wins}
			if r.WinRate != nil {
				m["win_rate"] = *r.WinRate
			} else {
				m["win_rate"] = "null"
			}
			items = append(items, m)
		}
		if len(items) == 0 {
			fmt.Fprintln(out, "no active deals to rank")
			return nil
		}
		return printAutoTable(out, items)
	}
	raw, _ := json.Marshal(map[string]any{"deals": rows, "scanned_deals": scanned})
	return printOutputWithFlags(out, raw, flags)
}
