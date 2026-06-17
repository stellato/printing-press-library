// Copyright 2026 Greg Stellato and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/mvanhorn/printing-press-library/library/marketing/equativ-maestro/internal/cliutil"
)

func TestWinRate(t *testing.T) {
	tests := []struct {
		name     string
		eligible float64
		wins     float64
		wantRate float64
		wantOK   bool
	}{
		{name: "normal", eligible: 100, wins: 25, wantRate: 0.25, wantOK: true},
		{name: "all win", eligible: 10, wins: 10, wantRate: 1.0, wantOK: true},
		{name: "zero eligible => null", eligible: 0, wins: 5, wantOK: false},
		{name: "negative eligible => null", eligible: -1, wins: 0, wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rate, ok := winRate(tt.eligible, tt.wins)
			if ok != tt.wantOK {
				t.Fatalf("winRate ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && rate != tt.wantRate {
				t.Errorf("winRate = %v, want %v", rate, tt.wantRate)
			}
		})
	}
}

func TestRankFunnelRows_WorstFirstNullsLast(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	rows := []funnelRow{
		{DealID: "good", WinRate: f(0.9)},
		{DealID: "null", WinRate: nil},
		{DealID: "worst", WinRate: f(0.1)},
		{DealID: "mid", WinRate: f(0.5)},
	}
	rankFunnelRows(rows)
	want := []string{"worst", "mid", "good", "null"}
	for i, w := range want {
		if rows[i].DealID != w {
			t.Errorf("position %d = %q, want %q", i, rows[i].DealID, w)
		}
	}
}

func TestParseFunnelStages(t *testing.T) {
	// Verified shape: {"breakdown":[ ... ]}, summed across rows.
	e, b, w := parseFunnelStages(json.RawMessage(
		`{"breakdown":[{"eligible":600,"bids":240,"wins":70},{"eligible":400,"bids":160,"wins":50}]}`,
	))
	if e != 1000 || b != 400 || w != 120 {
		t.Errorf("breakdown sum: got eligible=%v bids=%v wins=%v", e, b, w)
	}

	// Empty breakdown => 0/0/0 (deal still ranks, nulls last).
	e0, b0, w0 := parseFunnelStages(json.RawMessage(`{"breakdown":[]}`))
	if e0 != 0 || b0 != 0 || w0 != 0 {
		t.Errorf("empty breakdown should be zero, got eligible=%v bids=%v wins=%v", e0, b0, w0)
	}

	// Aliased stage names inside breakdown.
	e2, _, w2 := parseFunnelStages(json.RawMessage(`{"breakdown":[{"bidRequests":50,"winningBids":5}]}`))
	if e2 != 50 || w2 != 5 {
		t.Errorf("aliased: got eligible=%v wins=%v", e2, w2)
	}

	// Bare array (defensive fallback) is still summed.
	e3, _, w3 := parseFunnelStages(json.RawMessage(`[{"eligible":10,"wins":3},{"eligible":5,"wins":1}]`))
	if e3 != 15 || w3 != 4 {
		t.Errorf("bare array: got eligible=%v wins=%v", e3, w3)
	}
}

// TestFunnelFanoutExcludesErrors verifies that errored fetches are surfaced as
// FanoutErrors (and thus excluded from the ranked results) — mirroring the
// command's use of cliutil.FanoutRun.
func TestFunnelFanoutExcludesErrors(t *testing.T) {
	type ref struct{ id string }
	sources := []ref{{"ok1"}, {"boom"}, {"ok2"}}
	results, errs := cliutil.FanoutRun(
		context.Background(),
		sources,
		func(r ref) string { return r.id },
		func(_ context.Context, r ref) (funnelRow, error) {
			if r.id == "boom" {
				return funnelRow{}, errors.New("fetch failed")
			}
			rate := 0.5
			return funnelRow{DealID: r.id, Eligible: 10, Wins: 5, WinRate: &rate}, nil
		},
	)
	if len(results) != 2 {
		t.Fatalf("expected 2 successful results (errored excluded), got %d", len(results))
	}
	if len(errs) != 1 {
		t.Fatalf("expected 1 fetch failure recorded, got %d", len(errs))
	}
	if errs[0].Source != "boom" {
		t.Errorf("expected failure source 'boom', got %q", errs[0].Source)
	}
	// The errored deal must not appear in the ranked rows.
	rows := make([]funnelRow, 0, len(results))
	for _, r := range results {
		rows = append(rows, r.Value)
		if r.Value.DealID == "boom" {
			t.Errorf("errored deal 'boom' leaked into results")
		}
	}
	rankFunnelRows(rows)
	if len(rows) != 2 {
		t.Errorf("expected 2 ranked rows, got %d", len(rows))
	}
}
