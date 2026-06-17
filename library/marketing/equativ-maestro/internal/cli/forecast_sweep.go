// Copyright 2026 Greg Stellato and contributors. Licensed under Apache-2.0. See LICENSE.

// pp:data-source live

package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mvanhorn/printing-press-library/library/marketing/equativ-maestro/internal/cliutil"
	"github.com/mvanhorn/printing-press-library/library/marketing/equativ-maestro/internal/store"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

// sweepCell is one targeting-matrix coordinate before it is forecast. Each
// dimension holds a numeric Equativ id (countryId / deviceTypeId /
// audienceSegmentId); an empty string means the dimension is not constrained
// for this cell.
type sweepCell struct {
	Geo      string
	Device   string
	Audience string
}

// expandCells builds the cartesian product of geo × device × audience. Each
// argument is a CSV string of numeric ids; empty axes collapse to a single
// unconstrained value so they don't multiply the matrix. maxCells caps the
// result (0 = no cap); the returned bool reports whether truncation occurred.
// Order is deterministic (geo-major, then device, then audience). When every
// axis is empty the result is a single fully-unconstrained cell (total avails).
func expandCells(geoCSV, deviceCSV, audienceCSV string, maxCells int) ([]sweepCell, bool) {
	geos := splitCSV(geoCSV)
	devices := splitCSV(deviceCSV)
	audiences := splitCSV(audienceCSV)
	if len(geos) == 0 {
		geos = []string{""}
	}
	if len(devices) == 0 {
		devices = []string{""}
	}
	if len(audiences) == 0 {
		audiences = []string{""}
	}

	cells := make([]sweepCell, 0)
	truncated := false
	for _, g := range geos {
		for _, d := range devices {
			for _, a := range audiences {
				if maxCells > 0 && len(cells) >= maxCells {
					truncated = true
					return cells, truncated
				}
				cells = append(cells, sweepCell{Geo: g, Device: d, Audience: a})
			}
		}
	}
	return cells, truncated
}

// splitCSV trims and drops empties from a comma-separated list.
func splitCSV(s string) []string {
	out := make([]string, 0)
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// parseAvails sums (avails, auctions) across an inventoryInsights response. The
// verified shape is a JSON array of {"impressions":<num>,"auctions":<num>,
// "day":"<epoch_ms>"} rows; avails is the sum of impressions and auctions is the
// sum of auctions across the array. It still descends into a data/results
// envelope and tolerates a single object defensively, because only the array
// shape is pinned by the live contract.
func parseAvails(data json.RawMessage) (avails float64, auctions float64) {
	addRow := func(raw json.RawMessage) {
		var obj map[string]json.RawMessage
		if json.Unmarshal(raw, &obj) != nil {
			return
		}
		for _, key := range []string{"impressions", "avails", "availableImpressions", "available_impressions"} {
			if v, ok := cliutil.ExtractNumber(obj, key); ok {
				avails += v
				break
			}
		}
		for _, key := range []string{"auctions", "auctionCount", "auction_count", "bidRequests", "bid_requests"} {
			if v, ok := cliutil.ExtractNumber(obj, key); ok {
				auctions += v
				break
			}
		}
	}

	// Verified shape: a bare array of per-day rows.
	var arr []json.RawMessage
	if json.Unmarshal(data, &arr) == nil {
		for _, row := range arr {
			addRow(row)
		}
		return avails, auctions
	}

	// Defensive fallbacks: an envelope wrapping the array, or a single object.
	var top map[string]json.RawMessage
	if json.Unmarshal(data, &top) == nil {
		for _, key := range []string{"data", "results", "items", "rows"} {
			if inner, ok := top[key]; ok {
				var innerArr []json.RawMessage
				if json.Unmarshal(inner, &innerArr) == nil {
					for _, row := range innerArr {
						addRow(row)
					}
					return avails, auctions
				}
			}
		}
		addRow(data)
	}
	return avails, auctions
}

// buildSweepBody constructs the verified inventoryInsights request body for one
// cell. filters is a NESTED array [[f1,f2,...]] where the inner array is the AND
// of that cell's dimension filters; each filter is
// {"field":<name>,"operator":"IN","values":[<numericId>]} with numeric values.
// A cell with no constrained dimensions yields filters: [] (total avails).
func buildSweepBody(cell sweepCell, startDate, endDate string) map[string]any {
	inner := make([]map[string]any, 0, 3)
	addFilter := func(field, value string) {
		if value == "" {
			return
		}
		inner = append(inner, map[string]any{
			"field":    field,
			"operator": "IN",
			"values":   []any{numericValue(value)},
		})
	}
	addFilter("countryId", cell.Geo)
	addFilter("deviceTypeId", cell.Device)
	addFilter("audienceSegmentId", cell.Audience)

	filters := make([]any, 0, 1)
	if len(inner) > 0 {
		filters = append(filters, inner)
	}

	return map[string]any{
		"startDate":   startDate,
		"endDate":     endDate,
		"metrics":     []string{"impressions", "auctions"},
		"dimensions":  []any{},
		"filters":     filters,
		"useCaseId":   "ForecastDmkp",
		"timezone":    "UTC",
		"currency":    "EUR",
		"periodicity": "day",
	}
}

// numericValue renders an all-digit id as a JSON number so the request body
// carries numeric values (not strings) as the contract requires. A non-numeric
// token (shouldn't happen for id flags) is passed through as a string so the
// value is never silently dropped.
func numericValue(s string) any {
	if isAllDigits(s) {
		var n json.Number = json.Number(s)
		if i, err := n.Int64(); err == nil {
			return i
		}
	}
	return s
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func newNovelForecastSweepCmd(flags *rootFlags) *cobra.Command {
	var flagGeo, flagDevice, flagAudience string
	var flagStart, flagEnd, flagDiff string
	var flagMaxCells int
	var dbPath string

	cmd := &cobra.Command{
		Use:   "sweep",
		Short: "Run an avails forecast across a whole targeting matrix (geo x device x audience) in one command and get the grid.",
		Long: `Run an avails forecast across the cartesian product of geo × device × audience
and return the grid of results.

Each cell POSTs /report/inventoryInsights with useCaseId "ForecastDmkp" and sums
impressions (avails) and auctions across the returned per-day rows. Axes take
numeric Equativ ids (not codes):
  --geo       country IDs        (from 'equativ-maestro-pp-cli countries')
  --device    device type IDs    (optional dimension)
  --audience  audience segment IDs (optional dimension)

If only --geo is given the sweep runs over geos; if no axis is given a single
unconstrained cell is forecast (total avails). Use --dry-run to print the exact
request body for each cell without calling the API.

This is a live command: it POSTs an avails forecast per cell, so --data-source
local is rejected (except a pure --diff replay of a stored run, which reads the
local mirror).`,
		Example: `  # Forecast France (countryId 250)
  equativ-maestro-pp-cli forecast sweep --geo 250 --json

  # Forecast FR/DE × two device types (4 cells)
  equativ-maestro-pp-cli forecast sweep --geo 250,276 --device 1,2 --json

  # Preview the per-cell request body without calling the API
  equativ-maestro-pp-cli forecast sweep --geo 250 --dry-run

  # Replay a stored run and diff against it
  equativ-maestro-pp-cli forecast sweep --diff <run_id> --geo 250,276`,
		Annotations: map[string]string{"mcp:read-only": "false"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 && isTerminal(cmd.OutOrStdout()) {
				return cmd.Help()
			}

			haveAxes := strings.TrimSpace(flagGeo) != "" || strings.TrimSpace(flagDevice) != "" || strings.TrimSpace(flagAudience) != ""

			// Pure --diff replay (no new axes) only reads the stored run.
			diffReplayOnly := flagDiff != "" && !haveAxes

			if dryRunOK(flags) {
				cells, truncated := expandCells(flagGeo, flagDevice, flagAudience, effectiveMaxCells(flagMaxCells))
				start, end := sweepDates(flagStart, flagEnd)
				fmt.Fprintf(cmd.ErrOrStderr(), "would forecast %d cell(s) over %s..%s\n", len(cells), start, end)
				if truncated {
					fmt.Fprintf(cmd.ErrOrStderr(), "note: matrix truncated to --max-cells=%d\n", effectiveMaxCells(flagMaxCells))
				}
				for _, c := range cells {
					body := buildSweepBody(c, start, end)
					raw, _ := json.Marshal(body)
					fmt.Fprintf(cmd.OutOrStdout(), "%s\n", raw)
				}
				return nil
			}

			if flags.dataSource == "local" && !diffReplayOnly {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("forecast sweep is a live command and cannot run with --data-source local; use live or auto"))
			}

			if dbPath == "" {
				dbPath = defaultDBPath("equativ-maestro-pp-cli")
			}

			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()

			db, err := store.OpenWithContext(ctx, dbPath)
			if err != nil {
				return fmt.Errorf("opening local database: %w", err)
			}
			defer db.Close()
			if err := db.EnsureMaestroTables(); err != nil {
				return err
			}

			// Load the prior run for --diff (and for replay-only mode).
			var prior *store.ForecastRun
			if flagDiff != "" {
				prior, err = db.GetForecastRun(flagDiff)
				if err != nil {
					return fmt.Errorf("loading prior run %q: %w", flagDiff, err)
				}
				if prior == nil {
					return notFoundErr(fmt.Errorf("prior run %q not found in local store", flagDiff))
				}
				if diffReplayOnly {
					return emitSweepEnvelope(cmd, flags, prior, nil, false, nil)
				}
			}

			cells, truncated := expandCells(flagGeo, flagDevice, flagAudience, effectiveMaxCells(flagMaxCells))
			// expandCells always returns at least one cell (the unconstrained
			// total when no axis is given), so there is no empty-matrix guard.
			if truncated {
				fmt.Fprintf(cmd.ErrOrStderr(), "note: matrix truncated to --max-cells=%d\n", effectiveMaxCells(flagMaxCells))
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}
			start, end := sweepDates(flagStart, flagEnd)

			run := store.ForecastRun{
				RunID:     uuid.NewString(),
				CreatedAt: time.Now().UTC().Format(time.RFC3339),
				Geo:       flagGeo,
				Format:    flagDevice,
				Audience:  flagAudience,
				Cells:     make([]store.ForecastCell, 0, len(cells)),
			}
			fetchFailures := make([]map[string]string, 0)

			for i, cell := range cells {
				body := buildSweepBody(cell, start, end)
				data, _, postErr := c.Post(ctx, "/report/inventoryInsights", body)
				if postErr != nil {
					// Do NOT append failed cells to run.Cells: a stored cell with
					// avails=0 would later be picked up as a 0-baseline by --diff
					// and report a false positive delta. The failure is captured
					// in fetchFailures instead.
					fetchFailures = append(fetchFailures, map[string]string{
						"geo": cell.Geo, "device": cell.Device, "audience": cell.Audience,
						"error": postErr.Error(),
					})
				} else {
					sc := store.ForecastCell{Geo: cell.Geo, Format: cell.Device, Audience: cell.Audience}
					sc.Avails, sc.Auctions = parseAvails(data)
					sc.Raw = store.RawString(data)
					run.Cells = append(run.Cells, sc)
				}

				// Throttle ~1s between cells; skip under dogfood to stay under
				// the per-command timeout.
				if i < len(cells)-1 && !cliutil.IsDogfoodEnv() {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(time.Second):
					}
				}
			}

			if err := db.SaveForecastRun(run); err != nil {
				return fmt.Errorf("saving forecast run: %w", err)
			}
			if len(fetchFailures) > 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: %d of %d forecast cells failed to fetch\n", len(fetchFailures), len(cells))
			}
			return emitSweepEnvelope(cmd, flags, &run, prior, truncated, fetchFailures)
		},
	}

	cmd.Flags().StringVar(&flagGeo, "geo", "", "Targeting geos as CSV of numeric country IDs (e.g. 250,276; from 'countries')")
	cmd.Flags().StringVar(&flagDevice, "device", "", "Device types as CSV of numeric device type IDs (optional dimension)")
	cmd.Flags().StringVar(&flagAudience, "audience", "", "Audience segments as CSV of numeric audience segment IDs (optional dimension)")
	cmd.Flags().StringVar(&flagStart, "start", "", "Forecast window start date YYYY-MM-DD (default: 7 days ago)")
	cmd.Flags().StringVar(&flagEnd, "end", "", "Forecast window end date YYYY-MM-DD (default: today)")
	cmd.Flags().StringVar(&flagDiff, "diff", "", "Prior run_id to compare against (adds avails_delta per cell)")
	cmd.Flags().IntVar(&flagMaxCells, "max-cells", 24, "Maximum matrix cells to forecast (truncates with a note)")
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to the local SQLite mirror (default: per-user data dir)")
	return cmd
}

// effectiveMaxCells clamps to 1 under the dogfood matrix to keep the live run
// inside the per-command timeout.
func effectiveMaxCells(flagMaxCells int) int {
	if cliutil.IsDogfoodEnv() {
		return 1
	}
	return flagMaxCells
}

// sweepDates resolves the forecast window, defaulting to the last 7 days
// ending today. This is the printed CLI, not the workflow runtime, so
// time.Now() is acceptable here.
func sweepDates(start, end string) (string, string) {
	if end == "" {
		end = time.Now().UTC().Format("2006-01-02")
	}
	if start == "" {
		start = time.Now().UTC().AddDate(0, 0, -7).Format("2006-01-02")
	}
	return start, end
}

// emitSweepEnvelope renders the run grid as JSON or a human table. When prior
// is non-nil each cell gains an avails_delta vs the matching prior cell.
func emitSweepEnvelope(cmd *cobra.Command, flags *rootFlags, run *store.ForecastRun, prior *store.ForecastRun, truncated bool, fetchFailures []map[string]string) error {
	priorIdx := map[string]float64{}
	if prior != nil {
		for _, pc := range prior.Cells {
			priorIdx[cellKey(pc.Geo, pc.Format, pc.Audience)] = pc.Avails
		}
	}
	if fetchFailures == nil {
		fetchFailures = make([]map[string]string, 0)
	}

	type outCell struct {
		Geo         string   `json:"geo"`
		Device      string   `json:"device"`
		Audience    string   `json:"audience"`
		Avails      float64  `json:"avails"`
		Auctions    float64  `json:"auctions"`
		AvailsDelta *float64 `json:"avails_delta,omitempty"`
	}
	cells := make([]outCell, 0, len(run.Cells))
	for _, sc := range run.Cells {
		oc := outCell{Geo: sc.Geo, Device: sc.Format, Audience: sc.Audience, Avails: sc.Avails, Auctions: sc.Auctions}
		if prior != nil {
			if pa, ok := priorIdx[cellKey(sc.Geo, sc.Format, sc.Audience)]; ok {
				d := sc.Avails - pa
				oc.AvailsDelta = &d
			}
		}
		cells = append(cells, oc)
	}

	out := cmd.OutOrStdout()
	if flags.asJSON || (!isTerminal(out) && !flags.csv && !flags.quiet && !flags.plain) {
		envelope := map[string]any{
			"run_id":         run.RunID,
			"cells":          cells,
			"truncated":      truncated,
			"fetch_failures": fetchFailures,
		}
		raw, err := json.Marshal(envelope)
		if err != nil {
			return err
		}
		return printOutput(out, raw, true)
	}
	if wantsHumanTable(out, flags) {
		items := make([]map[string]any, 0, len(cells))
		for _, c := range cells {
			m := map[string]any{"geo": c.Geo, "device": c.Device, "audience": c.Audience, "avails": c.Avails, "auctions": c.Auctions}
			if c.AvailsDelta != nil {
				m["avails_delta"] = *c.AvailsDelta
			}
			items = append(items, m)
		}
		fmt.Fprintf(out, "run_id: %s\n", run.RunID)
		if len(items) == 0 {
			fmt.Fprintln(out, "no cells")
			return nil
		}
		return printAutoTable(out, items)
	}
	raw, _ := json.Marshal(map[string]any{"run_id": run.RunID, "cells": cells})
	return printOutputWithFlags(out, raw, flags)
}

func cellKey(geo, device, audience string) string {
	return geo + "|" + device + "|" + audience
}
