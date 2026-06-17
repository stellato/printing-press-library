// Copyright 2026 Greg Stellato and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"testing"
)

func mkOp(op, deal string) dealOp {
	return dealOp{Op: op, Deal: json.RawMessage(deal)}
}

func TestValidateDealOp(t *testing.T) {
	tests := []struct {
		name      string
		op        dealOp
		wantValid bool
	}{
		{
			name:      "valid create",
			op:        mkOp("create", `{"name":"My Deal","currency":"USD","price":2.5,"pricingModel":1}`),
			wantValid: true,
		},
		{
			name:      "create missing name => invalid",
			op:        mkOp("create", `{"currency":"USD","price":2.5,"pricingModel":1}`),
			wantValid: false,
		},
		{
			name:      "create missing price => invalid",
			op:        mkOp("create", `{"name":"X","currency":"USD","pricingModel":1}`),
			wantValid: false,
		},
		{
			name:      "valid update with dealId",
			op:        mkOp("update", `{"dealId":"ABC","price":3.0}`),
			wantValid: true,
		},
		{
			name:      "valid update with id",
			op:        mkOp("update", `{"id":123}`),
			wantValid: true,
		},
		{
			name:      "update without dealId or id => invalid",
			op:        mkOp("update", `{"price":3.0}`),
			wantValid: false,
		},
		{
			name:      "unknown op => invalid",
			op:        mkOp("delete", `{"id":1}`),
			wantValid: false,
		},
		{
			name:      "missing op => invalid",
			op:        mkOp("", `{"name":"X"}`),
			wantValid: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, reason := validateDealOp(tt.op)
			if valid != tt.wantValid {
				t.Errorf("validateDealOp valid = %v, want %v (reason: %q)", valid, tt.wantValid, reason)
			}
			if !valid && reason == "" {
				t.Errorf("expected a non-empty reason for invalid op")
			}
		})
	}
}

func TestPlanQuota_CreateCapStopsAtFifteen(t *testing.T) {
	// 20 create ops, all valid, no prior usage => only 15 fit under the cap.
	ops := make([]dealOp, 20)
	valid := make([]bool, 20)
	for i := range ops {
		ops[i] = mkOp("create", `{"name":"X","currency":"USD","price":1,"pricingModel":1}`)
		valid[i] = true
	}
	plan := planQuota(ops, valid, 0, 0)
	if plan.WouldCreate != dealApplyCreateCap {
		t.Fatalf("expected WouldCreate to stop at %d, got %d", dealApplyCreateCap, plan.WouldCreate)
	}
	if len(plan.SkippedForCap) != 5 {
		t.Errorf("expected 5 ops skipped for cap, got %d", len(plan.SkippedForCap))
	}
	// The first 15 should be allowed, the last 5 skipped.
	for i := 0; i < dealApplyCreateCap; i++ {
		if !plan.AllowedCreate[i] {
			t.Errorf("op %d should be allowed under the cap", i)
		}
	}
	for i := dealApplyCreateCap; i < 20; i++ {
		if plan.AllowedCreate[i] {
			t.Errorf("op %d should be skipped (over cap)", i)
		}
	}
}

func TestPlanQuota_RespectsPriorUsage(t *testing.T) {
	// 5 create ops, but 13 already used today => only 2 fit (cap 15).
	ops := make([]dealOp, 5)
	valid := make([]bool, 5)
	for i := range ops {
		ops[i] = mkOp("create", `{"name":"X","currency":"USD","price":1,"pricingModel":1}`)
		valid[i] = true
	}
	plan := planQuota(ops, valid, 13, 0)
	if plan.WouldCreate != 2 {
		t.Fatalf("expected 2 creates to fit (15-13), got %d", plan.WouldCreate)
	}
	if len(plan.SkippedForCap) != 3 {
		t.Errorf("expected 3 skipped, got %d", len(plan.SkippedForCap))
	}
	if plan.CreatesRemain != 0 {
		t.Errorf("expected 0 creates remaining, got %d", plan.CreatesRemain)
	}
}

func TestPlanQuota_InvalidOpsDoNotConsumeQuota(t *testing.T) {
	ops := []dealOp{
		mkOp("create", `{"name":"ok","currency":"USD","price":1,"pricingModel":1}`),
		mkOp("create", `{"currency":"USD"}`), // invalid, missing fields
		mkOp("update", `{"dealId":"D1"}`),
	}
	valid := []bool{true, false, true}
	plan := planQuota(ops, valid, 0, 0)
	if plan.WouldCreate != 1 {
		t.Errorf("expected 1 valid create counted, got %d", plan.WouldCreate)
	}
	if plan.WouldUpdate != 1 {
		t.Errorf("expected 1 valid update counted, got %d", plan.WouldUpdate)
	}
}

func TestPlanQuota_UpdateCap(t *testing.T) {
	// One update past the cap is skipped.
	ops := make([]dealOp, 2)
	valid := make([]bool, 2)
	for i := range ops {
		ops[i] = mkOp("update", `{"dealId":"D"}`)
		valid[i] = true
	}
	// 499 already used => only 1 of 2 fits.
	plan := planQuota(ops, valid, 0, dealApplyUpdateCap-1)
	if plan.WouldUpdate != 1 {
		t.Fatalf("expected 1 update to fit, got %d", plan.WouldUpdate)
	}
	if len(plan.SkippedForCap) != 1 {
		t.Errorf("expected 1 update skipped, got %d", len(plan.SkippedForCap))
	}
}
