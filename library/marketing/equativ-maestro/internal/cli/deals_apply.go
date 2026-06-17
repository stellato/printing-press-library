// Copyright 2026 Greg Stellato and contributors. Licensed under Apache-2.0. See LICENSE.

// pp:data-source live

package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mvanhorn/printing-press-library/library/marketing/equativ-maestro/internal/store"

	"github.com/spf13/cobra"
)

const (
	dealApplyCreateCap = 15
	dealApplyUpdateCap = 500
)

// dealOp is one staged create/update operation parsed from the input file.
type dealOp struct {
	Op   string          `json:"op"`
	Deal json.RawMessage `json:"deal"`
}

// opResult is the per-op outcome reported in the envelope.
type opResult struct {
	Op      string `json:"op"`
	Valid   bool   `json:"valid"`
	Reason  string `json:"reason,omitempty"`
	Applied bool   `json:"applied"`
	Error   string `json:"error,omitempty"`
}

// validateDealOp checks one op's local validity. create requires
// name+currency+price+pricingModel; update requires dealId or id. Returns
// (valid, reason). Reason is empty when valid.
func validateDealOp(op dealOp) (bool, string) {
	switch strings.ToLower(strings.TrimSpace(op.Op)) {
	case "create":
		var d map[string]json.RawMessage
		if len(op.Deal) == 0 || json.Unmarshal(op.Deal, &d) != nil {
			return false, "create op has no deal object"
		}
		var missing []string
		if rawStringField(d, "name") == "" {
			missing = append(missing, "name")
		}
		if rawStringField(d, "currency") == "" {
			missing = append(missing, "currency")
		}
		if _, ok := rawFloatField(d, "price"); !ok {
			missing = append(missing, "price")
		}
		if _, ok := rawIntField(d, "pricingModel"); !ok {
			missing = append(missing, "pricingModel")
		}
		if len(missing) > 0 {
			return false, "create missing required field(s): " + strings.Join(missing, ", ")
		}
		return true, ""
	case "update":
		var d map[string]json.RawMessage
		if len(op.Deal) == 0 || json.Unmarshal(op.Deal, &d) != nil {
			return false, "update op has no deal object"
		}
		if rawStringField(d, "dealId", "id") == "" {
			return false, "update requires dealId or id"
		}
		return true, ""
	case "":
		return false, "missing op (expected \"create\" or \"update\")"
	default:
		return false, fmt.Sprintf("unknown op %q (expected \"create\" or \"update\")", op.Op)
	}
}

// quotaPlan describes how many creates/updates a batch of valid ops would use
// against the day's remaining cap, and which ops (by zero-based index) would be
// skipped because they exceed the cap.
type quotaPlan struct {
	WouldCreate   int
	WouldUpdate   int
	SkippedForCap []int
	CreatesRemain int
	UpdatesRemain int
	AllowedCreate map[int]bool
	AllowedUpdate map[int]bool
}

// planQuota walks ops in order and decides which valid create/update ops fit
// under the remaining daily caps. usedCreates/usedUpdates are the day's
// already-consumed counts. Ops are gated in input order so the earliest ops
// win the remaining quota — matching the commit path's behavior.
func planQuota(ops []dealOp, valid []bool, usedCreates, usedUpdates int) quotaPlan {
	plan := quotaPlan{
		AllowedCreate: map[int]bool{},
		AllowedUpdate: map[int]bool{},
	}
	createsLeft := dealApplyCreateCap - usedCreates
	updatesLeft := dealApplyUpdateCap - usedUpdates
	if createsLeft < 0 {
		createsLeft = 0
	}
	if updatesLeft < 0 {
		updatesLeft = 0
	}
	for i, op := range ops {
		if i >= len(valid) || !valid[i] {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(op.Op)) {
		case "create":
			if plan.WouldCreate < createsLeft {
				plan.WouldCreate++
				plan.AllowedCreate[i] = true
			} else {
				plan.SkippedForCap = append(plan.SkippedForCap, i)
			}
		case "update":
			if plan.WouldUpdate < updatesLeft {
				plan.WouldUpdate++
				plan.AllowedUpdate[i] = true
			} else {
				plan.SkippedForCap = append(plan.SkippedForCap, i)
			}
		}
	}
	plan.CreatesRemain = createsLeft - plan.WouldCreate
	plan.UpdatesRemain = updatesLeft - plan.WouldUpdate
	return plan
}

func newNovelDealsApplyCmd(flags *rootFlags) *cobra.Command {
	var flagCommit bool
	var dbPath string

	cmd := &cobra.Command{
		Use:   "apply <file>",
		Short: "Stage a batch of deal creates/updates from a file, validate locally, dry-run by default",
		Long: `Apply a batch of deal create/update operations from a JSON file.

The file is a JSON array of operations, each:
  {"op": "create"|"update", "deal": { ...DealInput... }}

create requires name + currency + price + pricingModel; update requires
dealId or id. Operations are validated locally first. Daily rate caps apply:
15 creates/day and 500 updates/day; ops beyond the cap are skipped (and named).

By default this is a DRY RUN — it validates and previews quota usage but does
NOT call the API. Pass --commit to actually create/update deals.

NOTE: the deal create/update request body shape is APPROXIMATE — validate
against live credentials before committing in production.`,
		Example: `  # Validate and preview (no API calls)
  equativ-maestro-pp-cli deals apply ops.json --json

  # Actually create/update within the daily caps
  equativ-maestro-pp-cli deals apply ops.json --commit --json`,
		Annotations: map[string]string{"mcp:read-only": "false"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				n := 0
				if len(args) > 0 {
					if data, err := os.ReadFile(args[0]); err == nil {
						var ops []dealOp
						_ = json.Unmarshal(data, &ops)
						n = len(ops)
					}
				}
				fmt.Fprintf(cmd.ErrOrStderr(), "would apply %d op(s) from file\n", n)
				return nil
			}
			if flags.dataSource == "local" {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("deals apply is a live command and cannot run with --data-source local; use live or auto"))
			}
			if len(args) == 0 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("a path to an ops JSON file is required"))
			}

			data, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("reading ops file: %w", err)
			}
			var ops []dealOp
			if err := json.Unmarshal(data, &ops); err != nil {
				return usageErr(fmt.Errorf("parsing ops file: %w (expected a JSON array of {op, deal})", err))
			}
			if len(ops) == 0 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("ops file contained no operations"))
			}

			if dbPath == "" {
				dbPath = defaultDBPath("equativ-maestro-pp-cli")
			}
			db, err := store.OpenWithContext(cmd.Context(), dbPath)
			if err != nil {
				return fmt.Errorf("opening local database: %w", err)
			}
			defer db.Close()
			if err := db.EnsureMaestroTables(); err != nil {
				return err
			}

			valid := make([]bool, len(ops))
			results := make([]opResult, len(ops))
			for i, op := range ops {
				ok, reason := validateDealOp(op)
				valid[i] = ok
				results[i] = opResult{Op: op.Op, Valid: ok, Reason: reason}
			}

			today := time.Now().UTC().Format("2006-01-02")
			q, err := db.GetDealApplyQuota(today)
			if err != nil {
				return err
			}
			plan := planQuota(ops, valid, q.Creates, q.Updates)

			if !flagCommit {
				// Dry run (default): preview only, no API and no quota mutation.
				return emitApplyEnvelope(cmd, flags, true, results, q.Creates, q.Updates, plan)
			}

			// Commit path.
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			committedCreates, committedUpdates := 0, 0
			for i, op := range ops {
				if !valid[i] {
					continue
				}
				lop := strings.ToLower(strings.TrimSpace(op.Op))
				if (lop == "create" && !plan.AllowedCreate[i]) || (lop == "update" && !plan.AllowedUpdate[i]) {
					results[i].Reason = appendReason(results[i].Reason, "skipped: daily cap reached")
					fmt.Fprintf(cmd.ErrOrStderr(), "skipping op %d (%s): daily cap reached\n", i, lop)
					continue
				}

				var body map[string]any
				if json.Unmarshal(op.Deal, &body) != nil {
					results[i].Error = "deal object is not a JSON object"
					continue
				}
				switch lop {
				case "create":
					if _, _, perr := c.Post(ctx, "/deals", body); perr != nil {
						results[i].Error = perr.Error()
						continue
					}
					results[i].Applied = true
					committedCreates++
				case "update":
					if _, _, perr := c.Put(ctx, "/deals", body); perr != nil {
						results[i].Error = perr.Error()
						continue
					}
					results[i].Applied = true
					committedUpdates++
				}
			}

			if committedCreates > 0 || committedUpdates > 0 {
				if _, qerr := db.IncDealApplyQuota(today, committedCreates, committedUpdates); qerr != nil {
					// The live API writes already happened. Surface which ops
					// committed before failing so the user does not blindly
					// re-run the whole batch and duplicate creates.
					fmt.Fprintf(cmd.ErrOrStderr(),
						"warning: %d create(s) and %d update(s) were committed via the API, but recording daily quota usage failed: %v\nThe results below show which ops applied — do NOT re-run the whole batch.\n",
						committedCreates, committedUpdates, qerr)
					failPlan := quotaPlan{
						WouldCreate:   committedCreates,
						WouldUpdate:   committedUpdates,
						SkippedForCap: plan.SkippedForCap,
						CreatesRemain: dealApplyCreateCap - (q.Creates + committedCreates),
						UpdatesRemain: dealApplyUpdateCap - (q.Updates + committedUpdates),
					}
					_ = emitApplyEnvelope(cmd, flags, false, results, q.Creates+committedCreates, q.Updates+committedUpdates, failPlan)
					return qerr
				}
			}
			// Re-read so the envelope reflects post-commit usage.
			q, err = db.GetDealApplyQuota(today)
			if err != nil {
				return err
			}
			postPlan := quotaPlan{
				WouldCreate:   committedCreates,
				WouldUpdate:   committedUpdates,
				SkippedForCap: plan.SkippedForCap,
				CreatesRemain: dealApplyCreateCap - q.Creates,
				UpdatesRemain: dealApplyUpdateCap - q.Updates,
			}
			return emitApplyEnvelope(cmd, flags, false, results, q.Creates, q.Updates, postPlan)
		},
	}
	cmd.Flags().BoolVar(&flagCommit, "commit", false, "Actually create/update deals (default is a validate-only dry run)")
	cmd.Flags().StringVar(&dbPath, "db", "", "Path to the local SQLite mirror (default: per-user data dir)")
	return cmd
}

func appendReason(existing, add string) string {
	if existing == "" {
		return add
	}
	return existing + "; " + add
}

// emitApplyEnvelope renders the apply result. dryRun controls the dry_run flag
// and the quota block's framing (would_use vs used).
func emitApplyEnvelope(cmd *cobra.Command, flags *rootFlags, dryRun bool, results []opResult, createsUsed, updatesUsed int, plan quotaPlan) error {
	quota := map[string]any{
		"creates_used":      createsUsed,
		"updates_used":      updatesUsed,
		"creates_remaining": clampNonNeg(dealApplyCreateCap - createsUsed),
		"updates_remaining": clampNonNeg(dealApplyUpdateCap - updatesUsed),
	}
	if dryRun {
		quota["would_use"] = map[string]int{"creates": plan.WouldCreate, "updates": plan.WouldUpdate}
		quota["remaining_after"] = map[string]int{
			"creates": clampNonNeg(plan.CreatesRemain),
			"updates": clampNonNeg(plan.UpdatesRemain),
		}
		if len(plan.SkippedForCap) > 0 {
			quota["skipped_for_cap"] = plan.SkippedForCap
		}
	}
	envelope := map[string]any{
		"dry_run": dryRun,
		"ops":     results,
		"quota":   quota,
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
		items := make([]map[string]any, 0, len(results))
		for i, r := range results {
			items = append(items, map[string]any{
				"index": i, "op": r.Op, "valid": r.Valid, "applied": r.Applied,
				"reason": r.Reason, "error": r.Error,
			})
		}
		if len(items) > 0 {
			if err := printAutoTable(out, items); err != nil {
				return err
			}
		}
		fmt.Fprintf(out, "\nquota: creates %d/%d, updates %d/%d (dry_run=%v)\n",
			createsUsed, dealApplyCreateCap, updatesUsed, dealApplyUpdateCap, dryRun)
		return nil
	}
	raw, _ := json.Marshal(envelope)
	return printOutputWithFlags(out, raw, flags)
}

func clampNonNeg(n int) int {
	if n < 0 {
		return 0
	}
	return n
}
