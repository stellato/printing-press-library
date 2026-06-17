// Copyright 2026 Greg Stellato and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func rawList(objs ...string) []json.RawMessage {
	out := make([]json.RawMessage, 0, len(objs))
	for _, o := range objs {
		out = append(out, json.RawMessage(o))
	}
	return out
}

func findingTypes(fs []reconcileFinding) map[string]int {
	m := map[string]int{}
	for _, f := range fs {
		m[f.Type]++
	}
	return m
}

func TestReconcileFindings_ActiveUnderDelivering(t *testing.T) {
	in := reconcileInput{
		haveDeals: true,
		deals: rawList(
			`{"dealId":"D1","name":"Bad","isActive":true,"isUnderDelivering":true}`,
			`{"dealId":"D2","name":"Fine","isActive":true,"isUnderDelivering":false}`,
			`{"dealId":"D3","name":"Inactive","isActive":false,"isUnderDelivering":true}`,
		),
	}
	findings, _ := reconcileFindings(in)
	types := findingTypes(findings)
	if types["active_under_delivering"] != 1 {
		t.Fatalf("expected exactly 1 active_under_delivering finding, got %d (%+v)", types["active_under_delivering"], findings)
	}
	if findings[0].ID != "D1" {
		t.Errorf("expected finding on D1, got %q", findings[0].ID)
	}
}

func TestReconcileFindings_RegulatorDisabled(t *testing.T) {
	in := reconcileInput{
		haveDeals: true,
		deals: rawList(
			`{"dealId":"D9","name":"Blocked","isActive":true,"isDisabledByRegulator":true}`,
		),
	}
	findings, _ := reconcileFindings(in)
	if findingTypes(findings)["active_but_regulator_disabled"] != 1 {
		t.Fatalf("expected regulator-disabled finding, got %+v", findings)
	}
}

func TestReconcileFindings_CleanData(t *testing.T) {
	in := reconcileInput{
		haveDeals: true,
		deals: rawList(
			`{"dealId":"D1","name":"Healthy","isActive":true,"isUnderDelivering":false}`,
		),
	}
	findings, _ := reconcileFindings(in)
	if len(findings) != 0 {
		t.Fatalf("expected no findings on clean data, got %+v", findings)
	}
}

func TestReconcileFindings_OrphanLineItemAndEmptyCampaign(t *testing.T) {
	in := reconcileInput{
		haveDeals: true,
		haveCamps: true,
		haveLines: true,
		deals:     rawList(`{"dealId":"D1","isActive":true}`),
		campaigns: rawList(`{"id":"C1"}`, `{"id":"C2"}`),
		lineItems: rawList(
			`{"id":"L1","campaignId":"C1"}`,   // belongs to C1 (ok)
			`{"id":"L2","campaignId":"C404"}`, // orphan
		),
	}
	findings, skipped := reconcileFindings(in)
	types := findingTypes(findings)
	if types["orphan_line_item"] != 1 {
		t.Errorf("expected 1 orphan_line_item, got %d (%+v)", types["orphan_line_item"], findings)
	}
	// C2 has no line items.
	if types["campaign_no_line_items"] != 1 {
		t.Errorf("expected 1 campaign_no_line_items (C2), got %d (%+v)", types["campaign_no_line_items"], findings)
	}
	if len(skipped) != 0 {
		t.Errorf("expected no skipped checks when both synced, got %+v", skipped)
	}
}

func TestReconcileFindings_SkipsWhenNotSynced(t *testing.T) {
	in := reconcileInput{
		haveDeals: false, // deals not synced
		haveCamps: false,
		haveLines: false,
	}
	findings, skipped := reconcileFindings(in)
	if len(findings) != 0 {
		t.Errorf("expected no findings with nothing synced, got %+v", findings)
	}
	if len(skipped) < 2 {
		t.Errorf("expected skipped checks reported (deals + cross-resource), got %+v", skipped)
	}
}

func TestReconcileFindings_SpendCheck(t *testing.T) {
	// --spend with budgets synced: one exhausted budget.
	in := reconcileInput{
		haveDeals:   true,
		deals:       rawList(`{"dealId":"D1","isActive":true}`),
		checkSpend:  true,
		haveBudgets: true,
		budgets: rawList(
			`{"id":"B1","budget":1000,"delivered":1000}`, // exhausted
			`{"id":"B2","budget":1000,"delivered":400}`,  // fine
		),
	}
	findings, _ := reconcileFindings(in)
	if findingTypes(findings)["budget_exhausted"] != 1 {
		t.Fatalf("expected 1 budget_exhausted finding, got %+v", findings)
	}

	// --spend but no budgets synced: reported as skipped, not a finding.
	in2 := reconcileInput{haveDeals: true, deals: rawList(`{"dealId":"D1","isActive":true}`), checkSpend: true, haveBudgets: false}
	_, skipped := reconcileFindings(in2)
	foundSkip := false
	for _, s := range skipped {
		if strings.Contains(s, "budget_exhausted") {
			foundSkip = true
		}
	}
	if !foundSkip {
		t.Errorf("expected budget_exhausted reported as skipped when no budgets synced, got %+v", skipped)
	}
}
