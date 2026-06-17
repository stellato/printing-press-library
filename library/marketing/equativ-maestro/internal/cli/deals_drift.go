// Copyright 2026 Greg Stellato and contributors. Licensed under Apache-2.0. See LICENSE.

// pp:data-source live

package cli

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/mvanhorn/printing-press-library/library/marketing/equativ-maestro/internal/cliutil"
	"github.com/mvanhorn/printing-press-library/library/marketing/equativ-maestro/internal/store"

	"github.com/spf13/cobra"
)

// driftRow is one deal's pacing-drift result in the output envelope.
type driftRow struct {
	DealID       string   `json:"dealId"`
	Name         string   `json:"name"`
	Pacing       float64  `json:"pacing"`
	PriorPacing  *float64 `json:"prior_pacing"`
	Drift        *float64 `json:"drift"`
	BaselineOnly bool     `json:"baseline_only"`
}

// computeDrift returns (drift, baselineOnly). When fewer than two snapshots
// exist the deal is baseline-only and drift is undefined (returns 0, true).
// Otherwise drift = current - prior. snaps is expected newest-first (the order
// LatestTwoPacingSnapshots returns).
func computeDrift(snaps []store.PacingSnapshot) (drift float64, baselineOnly bool) {
	if len(snaps) < 2 {
		return 0, true
	}
	return snaps[0].Pacing - snaps[1].Pacing, false
}

// driftBaseline selects the prior-pacing reading the current run is compared
// against. It MUST be called BEFORE the current snapshot is recorded, so the
// just-captured row can never be picked as its own baseline. Without --since
// the baseline is the most-recent EXISTING snapshot (the reading before the one
// about to be recorded). With --since=<date> the baseline is the latest
// existing snapshot captured on or before that date. Returns (priorPacing,
// havePrior, err); havePrior is false when the deal has no usable baseline yet
// (baseline-only).
//
// since is assumed already validated as YYYY-MM-DD by the caller.
func driftBaseline(db *store.Store, dealID, since string) (priorPacing float64, havePrior bool, err error) {
	if since == "" {
		snaps, err := db.LatestTwoPacingSnapshots(dealID)
		if err != nil {
			return 0, false, err
		}
		if len(snaps) < 1 {
			return 0, false, nil
		}
		return snaps[0].Pacing, true, nil
	}

	base, err := db.LatestSnapshotOnOrBefore(dealID, since)
	if err != nil {
		return 0, false, err
	}
	if base == nil {
		return 0, false, nil
	}
	return base.Pacing, true, nil
}

// rankDriftRows sorts rows by absolute drift descending; baseline-only rows
// (no drift yet) sort to the end. Stable on dealId for deterministic output.
func rankDriftRows(rows []driftRow) {
	sort.SliceStable(rows, func(i, j int) bool {
		di, dj := rows[i].Drift, rows[j].Drift
		switch {
		case di == nil && dj == nil:
			return rows[i].DealID < rows[j].DealID
		case di == nil:
			return false
		case dj == nil:
			return true
		}
		ai, aj := math.Abs(*di), math.Abs(*dj)
		if ai == aj {
			return rows[i].DealID < rows[j].DealID
		}
		return ai > aj
	})
}

// parsePacingValue extracts a pacing reading for a single deal object from a
// pacing report response, falling back to the deal's isUnderDelivering flag
// when no numeric field is present. Searches a small set of likely field names
// defensively because the pacing report shape is not pinned by the spec.
func parsePacingValue(raw json.RawMessage, isUnderDelivering bool) float64 {
	var obj map[string]json.RawMessage
	if json.Unmarshal(raw, &obj) == nil {
		for _, key := range []string{"pacing", "pace", "delivery", "deliveryRate", "delivery_rate", "deliveryPacing", "pacingRate"} {
			if v, ok := cliutil.ExtractNumber(obj, key); ok {
				return v
			}
		}
	}
	if isUnderDelivering {
		return 0
	}
	return 100
}

// indexPacingByDeal maps a pacing report response (array or {data:[...]}) to a
// per-internal-id raw object so each deal can parse its own pacing value. The
// pacing endpoint is keyed by the deal's internal numeric id (the `id` field,
// e.g. 5882525), NOT the public dealId, so this indexes by `id` first. Falls
// back to an empty map when the shape is unrecognized; callers then use the
// isUnderDelivering fallback.
func indexPacingByDeal(data json.RawMessage) map[string]json.RawMessage {
	out := map[string]json.RawMessage{}
	var arr []json.RawMessage
	if json.Unmarshal(data, &arr) != nil {
		// Try common envelopes.
		var env map[string]json.RawMessage
		if json.Unmarshal(data, &env) == nil {
			for _, key := range []string{"data", "results", "items", "deals"} {
				if inner, ok := env[key]; ok {
					if json.Unmarshal(inner, &arr) == nil {
						break
					}
				}
			}
		}
	}
	for _, item := range arr {
		var obj map[string]json.RawMessage
		if json.Unmarshal(item, &obj) != nil {
			continue
		}
		// Key by internal id first (what the pacing endpoint echoes back), then
		// fall back to dealId for defensiveness against shape drift.
		for _, key := range []string{"id", "dealId", "deal_id"} {
			if raw, ok := obj[key]; ok {
				var id string
				if json.Unmarshal(raw, &id) == nil && id != "" {
					out[id] = item
					break
				}
				var n json.Number
				if json.Unmarshal(raw, &n) == nil && n.String() != "" {
					out[n.String()] = item
					break
				}
			}
		}
	}
	return out
}

func newNovelDealsDriftCmd(flags *rootFlags) *cobra.Command {
	var flagThreshold float64
	var flagSince string
	var dbPath string

	cmd := &cobra.Command{
		Use:   "drift",
		Short: "See which PMP deals drifted off pace since your last sync — one ranked view across your whole deal book.",
		Long: `Rank active deals by how much their pacing changed since the prior snapshot.

Each run fetches current pacing for every active synced deal, records a snapshot,
and compares it against that deal's previous snapshot to compute drift
(current pacing minus prior pacing). Deals with only one snapshot are reported as
"baseline" — run drift again later to see movement.

Pass --since <YYYY-MM-DD> to compare against the latest snapshot captured on or
before that date instead of the immediately preceding one (when no snapshot is
on/before the date, the deal's oldest snapshot is used).

This is a live command: it calls the pacing report and records snapshots, so
--data-source local is rejected. Pacing is fetched by the deal's internal numeric
id (the public dealId is not accepted by the pacing endpoint). The pacing field is
parsed defensively and falls back to the deal's isUnderDelivering flag when the
report omits a numeric pacing value.`,
		Example: `  # Record a baseline / show drift since last run
  equativ-maestro-pp-cli deals drift

  # Compare current pacing against the snapshot on or before a date
  equativ-maestro-pp-cli deals drift --since 2026-06-10 --json

  # Exit non-zero (code 3) if any deal drifted 20% or more
  equativ-maestro-pp-cli deals drift --threshold 20 --json`,
		Annotations: map[string]string{"mcp:read-only": "true", "pp:typed-exit-codes": "0,3"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 && isTerminal(cmd.OutOrStdout()) {
				return cmd.Help()
			}
			if flagSince != "" {
				if _, err := time.Parse("2006-01-02", flagSince); err != nil {
					_ = cmd.Usage()
					return usageErr(fmt.Errorf("--since must be a date in YYYY-MM-DD form: %w", err))
				}
			}
			if dryRunOK(flags) {
				fmt.Fprintln(cmd.ErrOrStderr(), "would rank active deals by pacing drift since the prior snapshot")
				return nil
			}
			if flags.dataSource == "local" {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("deals drift is a live command and cannot run with --data-source local; use live or auto"))
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
			if err := db.EnsureMaestroTables(); err != nil {
				return err
			}
			maybeEmitSyncHints(cmd, db, "deals", flags.maxAge)

			rawDeals, err := db.List("deals", 0)
			if err != nil {
				return fmt.Errorf("listing synced deals: %w", err)
			}

			type activeDeal struct {
				dealID            string // public dealId — stable key for snapshot history
				internalID        string // internal numeric id — what the pacing endpoint expects
				name              string
				isUnderDelivering bool
			}
			active := make([]activeDeal, 0, len(rawDeals))
			for _, rd := range rawDeals {
				var d map[string]json.RawMessage
				if json.Unmarshal(rd, &d) != nil {
					continue
				}
				if !rawBool(d["isActive"]) {
					continue
				}
				active = append(active, activeDeal{
					dealID:            rawStringField(d, "dealId", "id"),
					internalID:        rawStringField(d, "id"),
					name:              rawStringField(d, "name"),
					isUnderDelivering: rawBool(d["isUnderDelivering"]),
				})
			}

			if len(active) == 0 {
				return emitDriftEnvelope(cmd, flags, make([]driftRow, 0), "no active deals in the local mirror", false)
			}

			// Batch all internal ids into one pacing request. The pacing
			// endpoint is keyed by the deal's internal numeric id; passing the
			// public dealId returns HTTP 400 "ids not valid".
			ids := make([]string, 0, len(active))
			for _, a := range active {
				if a.internalID != "" {
					ids = append(ids, a.internalID)
				}
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			params := map[string]string{"ids": strings.Join(ids, ","), "timezone": "UTC"}
			pacingData, err := c.GetNoCache(ctx, "/report/deals/pacing", params)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			pacingByDeal := indexPacingByDeal(pacingData)

			capturedAt := time.Now().UTC().Format(time.RFC3339Nano)
			rows := make([]driftRow, 0, len(active))
			anyDrift := false
			anyPrior := false
			thresholdBreached := false

			for _, a := range active {
				// Pacing is keyed by internal id in the report response.
				pacing := parsePacingValue(pacingByDeal[a.internalID], a.isUnderDelivering)

				// Select the baseline from EXISTING snapshots BEFORE recording
				// the current reading, so the just-captured row can never be
				// picked as its own --since baseline.
				priorPacing, havePrior, err := driftBaseline(db, a.dealID, flagSince)
				if err != nil {
					return fmt.Errorf("loading snapshots for %s: %w", a.dealID, err)
				}

				if err := db.RecordPacingSnapshot(store.PacingSnapshot{
					DealID:            a.dealID,
					Name:              a.name,
					CapturedAt:        capturedAt,
					Pacing:            pacing,
					IsUnderDelivering: a.isUnderDelivering,
					IsActive:          true,
				}); err != nil {
					return fmt.Errorf("recording snapshot for %s: %w", a.dealID, err)
				}
				row := driftRow{
					DealID:       a.dealID,
					Name:         a.name,
					Pacing:       pacing,
					BaselineOnly: !havePrior,
				}
				if havePrior {
					anyDrift = true
					anyPrior = true
					prior := priorPacing
					d := pacing - priorPacing
					row.PriorPacing = &prior
					row.Drift = &d
					if flagThreshold > 0 && math.Abs(d) >= flagThreshold {
						thresholdBreached = true
					}
				}
				rows = append(rows, row)
			}

			rankDriftRows(rows)

			note := ""
			if !anyPrior {
				note = "baseline recorded; run again later to see drift"
				fmt.Fprintln(cmd.ErrOrStderr(), note)
			}

			if err := emitDriftEnvelope(cmd, flags, rows, note, !anyDrift); err != nil {
				return err
			}

			if thresholdBreached {
				return &cliError{code: 3, err: fmt.Errorf("one or more deals drifted at or beyond the --threshold of %.2f", flagThreshold)}
			}
			return nil
		},
	}

	cmd.Flags().Float64Var(&flagThreshold, "threshold", 0, "Drift magnitude (in pacing %) that triggers a non-zero exit (code 3); 0 never exits non-zero")
	cmd.Flags().StringVar(&flagSince, "since", "", "Baseline date (YYYY-MM-DD): drift is measured against the latest snapshot on or before this date instead of the previous snapshot")
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to the local SQLite mirror (default: per-user data dir)")
	return cmd
}

// emitDriftEnvelope writes the drift result as JSON (machine modes / piped) or a
// human table. allBaseline adds a note when every row is baseline-only.
func emitDriftEnvelope(cmd *cobra.Command, flags *rootFlags, rows []driftRow, note string, allBaseline bool) error {
	out := cmd.OutOrStdout()
	if flags.asJSON || (!isTerminal(out) && !flags.csv && !flags.quiet && !flags.plain) {
		envelope := map[string]any{"deals": rows}
		if note != "" {
			envelope["note"] = note
		}
		if allBaseline && note == "" {
			envelope["note"] = "all deals are baseline-only; run again later to see drift"
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
			m := map[string]any{"dealId": r.DealID, "name": r.Name, "pacing": r.Pacing, "baseline_only": r.BaselineOnly}
			if r.Drift != nil {
				m["drift"] = *r.Drift
			}
			if r.PriorPacing != nil {
				m["prior_pacing"] = *r.PriorPacing
			}
			items = append(items, m)
		}
		if len(items) == 0 {
			fmt.Fprintln(out, "no active deals to rank")
			return nil
		}
		return printAutoTable(out, items)
	}
	raw, _ := json.Marshal(map[string]any{"deals": rows})
	return printOutputWithFlags(out, raw, flags)
}
