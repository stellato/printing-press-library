// Copyright 2026 Greg Stellato and contributors. Licensed under Apache-2.0. See LICENSE.

// pp:data-source local

package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/mvanhorn/printing-press-library/library/marketing/equativ-maestro/internal/store"

	"github.com/spf13/cobra"
)

// reconcileFinding is one structural problem detected across the synced book.
type reconcileFinding struct {
	Type   string `json:"type"`
	Entity string `json:"entity"`
	ID     string `json:"id"`
	Detail string `json:"detail"`
}

// reconcileInput carries the synced collections reconcileFindings inspects.
// Each field may be empty/nil when that resource was never synced; the checks
// that depend on a missing collection are skipped (and reported as skipped).
type reconcileInput struct {
	deals       []json.RawMessage
	campaigns   []json.RawMessage
	lineItems   []json.RawMessage
	budgets     []json.RawMessage // campaignBudgets ∪ lineItemBudgets
	checkSpend  bool
	haveDeals   bool
	haveCamps   bool
	haveLines   bool
	haveBudgets bool
}

// reconcileFindings is the pure rules engine: given synced collections it
// returns the findings plus the list of checks skipped because their input
// wasn't synced. Extracted from the command so the rules are unit-testable
// without a store.
func reconcileFindings(in reconcileInput) (findings []reconcileFinding, skipped []string) {
	findings = make([]reconcileFinding, 0)
	skipped = make([]string, 0)

	// Deal-level checks (need deals synced).
	if !in.haveDeals {
		skipped = append(skipped, "deal_checks: deals not synced")
	} else {
		for _, rd := range in.deals {
			var d map[string]json.RawMessage
			if json.Unmarshal(rd, &d) != nil {
				continue
			}
			id := rawStringField(d, "dealId", "id")
			name := rawStringField(d, "name")
			active := rawBool(d["isActive"])
			if active && rawBool(d["isUnderDelivering"]) {
				findings = append(findings, reconcileFinding{
					Type: "active_under_delivering", Entity: "deal", ID: id,
					Detail: fmt.Sprintf("deal %q is active but under-delivering", nameOrID(name, id)),
				})
			}
			if active && rawBool(d["isDisabledByRegulator"]) {
				findings = append(findings, reconcileFinding{
					Type: "active_but_regulator_disabled", Entity: "deal", ID: id,
					Detail: fmt.Sprintf("deal %q is active but disabled by a regulator", nameOrID(name, id)),
				})
			}
		}
	}

	// Cross-resource checks (need lineitems AND campaigns synced).
	if !in.haveLines || !in.haveCamps {
		skipped = append(skipped, "orphan_line_item, campaign_no_line_items: require both lineitems and campaigns synced")
	} else {
		campaignIDs := map[string]bool{}
		for _, rc := range in.campaigns {
			var c map[string]json.RawMessage
			if json.Unmarshal(rc, &c) != nil {
				continue
			}
			if id := rawStringField(c, "id", "campaignId"); id != "" {
				campaignIDs[id] = true
			}
		}
		campaignHasLine := map[string]bool{}
		for _, rl := range in.lineItems {
			var l map[string]json.RawMessage
			if json.Unmarshal(rl, &l) != nil {
				continue
			}
			liID := rawStringField(l, "id", "lineItemId")
			campID := rawStringField(l, "campaignId", "insertionId", "campaign_id")
			if campID != "" {
				campaignHasLine[campID] = true
				if !campaignIDs[campID] {
					findings = append(findings, reconcileFinding{
						Type: "orphan_line_item", Entity: "lineitem", ID: liID,
						Detail: fmt.Sprintf("line item %q references campaign %q not present among synced campaigns", liID, campID),
					})
				}
			}
		}
		campaignIDsSorted := make([]string, 0, len(campaignIDs))
		for id := range campaignIDs {
			campaignIDsSorted = append(campaignIDsSorted, id)
		}
		sort.Strings(campaignIDsSorted)
		for _, id := range campaignIDsSorted {
			if !campaignHasLine[id] {
				findings = append(findings, reconcileFinding{
					Type: "campaign_no_line_items", Entity: "campaign", ID: id,
					Detail: fmt.Sprintf("campaign %q has zero line items", id),
				})
			}
		}
	}

	// Spend checks (need --spend AND budgets + report data).
	if in.checkSpend {
		if !in.haveBudgets {
			skipped = append(skipped, "budget_exhausted: --spend set but no budget data synced")
		} else {
			for _, rb := range in.budgets {
				var b map[string]json.RawMessage
				if json.Unmarshal(rb, &b) != nil {
					continue
				}
				id := rawStringField(b, "id", "budgetId")
				budget, hasBudget := rawFloatField(b, "budget", "overallBudget", "targetBudget", "amount")
				delivered, hasDelivered := rawFloatField(b, "delivered", "spend", "deliveredAmount", "spentAmount")
				if hasBudget && hasDelivered && budget > 0 && delivered >= budget {
					findings = append(findings, reconcileFinding{
						Type: "budget_exhausted", Entity: "budget", ID: id,
						Detail: fmt.Sprintf("budget %q delivered %.2f of %.2f (exhausted/exceeded)", id, delivered, budget),
					})
				}
			}
		}
	}

	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Type != findings[j].Type {
			return findings[i].Type < findings[j].Type
		}
		return findings[i].ID < findings[j].ID
	})
	return findings, skipped
}

func nameOrID(name, id string) string {
	if name != "" {
		return name
	}
	return id
}

func newNovelReconcileCmd(flags *rootFlags) *cobra.Command {
	var flagSpend bool
	var dbPath string

	cmd := &cobra.Command{
		Use:   "reconcile",
		Short: "Find structural problems across your book: deals with no line item, line items with zero delivery",
		Long: `Surface structural problems across the locally synced book at a point in time.

Checks (each runs only when its inputs were synced; skipped checks are reported):
  • active_under_delivering        active deals flagged under-delivering
  • active_but_regulator_disabled  active deals disabled by a regulator
  • orphan_line_item               line items whose campaign isn't synced
  • campaign_no_line_items         campaigns with zero line items
  • budget_exhausted (--spend)     budgets delivered at or beyond target

This is a local command and reads only the synced mirror; it never calls the API.
Run 'equativ-maestro-pp-cli sync' first to populate the mirror.`,
		Example: `  # Reconcile the synced book
  equativ-maestro-pp-cli reconcile --json

  # Also check budget vs delivery (when budgets are synced)
  equativ-maestro-pp-cli reconcile --spend --json`,
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Only show help for an interactive bare call; a piped/agent
			// invocation with no flags is a legitimate "run it" request.
			if len(args) == 0 && cmd.Flags().NFlag() == 0 && isTerminal(cmd.OutOrStdout()) {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				fmt.Fprintln(cmd.ErrOrStderr(), "would scan the synced book for structural problems")
				return nil
			}
			if flags.dataSource == "live" {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("reconcile is a local command and cannot run with --data-source live; use local or auto"))
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

			db, err := store.OpenWithContext(cmd.Context(), dbPath)
			if err != nil {
				return fmt.Errorf("opening local database: %w", err)
			}
			defer db.Close()
			maybeEmitSyncHints(cmd, db, "deals", flags.maxAge)

			in := reconcileInput{checkSpend: flagSpend}
			in.deals, in.haveDeals = listIfSynced(db, "deals")
			in.campaigns, in.haveCamps = listIfSynced(db, "campaigns")
			in.lineItems, in.haveLines = listIfSynced(db, "lineitems")
			campBudgets, haveCampB := listIfSynced(db, "campaignBudgets")
			liBudgets, haveLiB := listIfSynced(db, "lineItemBudgets")
			in.budgets = append(append([]json.RawMessage{}, campBudgets...), liBudgets...)
			in.haveBudgets = haveCampB || haveLiB

			findings, skipped := reconcileFindings(in)

			envelope := map[string]any{
				"findings": findings,
				"checked": map[string]int{
					"deals":     len(in.deals),
					"campaigns": len(in.campaigns),
					"lineitems": len(in.lineItems),
				},
				"skipped_checks": skipped,
			}
			if len(findings) == 0 {
				envelope["note"] = "no structural problems found in the synced data"
			}

			out := cmd.OutOrStdout()
			if flags.asJSON || (!isTerminal(out) && !flags.csv && !flags.quiet && !flags.plain) {
				raw, err := json.Marshal(envelope)
				if err != nil {
					return err
				}
				return printOutput(out, raw, true)
			}
			if wantsHumanTable(out, flags) {
				if len(findings) == 0 {
					fmt.Fprintln(out, "No structural problems found in the synced data.")
					if len(skipped) > 0 {
						fmt.Fprintf(out, "Skipped %d check group(s) due to unsynced data.\n", len(skipped))
					}
					return nil
				}
				items := make([]map[string]any, 0, len(findings))
				for _, f := range findings {
					items = append(items, map[string]any{"type": f.Type, "entity": f.Entity, "id": f.ID, "detail": f.Detail})
				}
				if err := printAutoTable(out, items); err != nil {
					return err
				}
				for _, s := range skipped {
					fmt.Fprintf(cmd.ErrOrStderr(), "skipped: %s\n", s)
				}
				return nil
			}
			raw, _ := json.Marshal(envelope)
			return printOutputWithFlags(out, raw, flags)
		},
	}
	cmd.Flags().BoolVar(&flagSpend, "spend", false, "Also check budget vs delivered spend (only when budget data is synced)")
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to the local SQLite mirror (default: per-user data dir)")
	return cmd
}

// listIfSynced returns the synced rows for a resource type plus whether the
// resource has been synced at all (distinct from "synced but empty"). A
// store/query error is treated as not-synced rather than fatal so reconcile
// degrades gracefully on resources this CLI never mirrors.
func listIfSynced(db *store.Store, resourceType string) ([]json.RawMessage, bool) {
	_, lastSynced, _, err := db.GetSyncState(resourceType)
	synced := err == nil && !lastSynced.IsZero()
	rows, listErr := db.List(resourceType, 0)
	if listErr != nil {
		return nil, synced
	}
	if len(rows) > 0 {
		// Presence of rows is itself proof the resource was synced, even if
		// sync_state bookkeeping is missing.
		synced = true
	}
	return rows, synced
}
