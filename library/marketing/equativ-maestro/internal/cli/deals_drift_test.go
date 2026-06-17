// Copyright 2026 Greg Stellato and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/mvanhorn/printing-press-library/library/marketing/equativ-maestro/internal/store"
)

func TestComputeDrift(t *testing.T) {
	tests := []struct {
		name         string
		snaps        []store.PacingSnapshot
		wantDrift    float64
		wantBaseline bool
	}{
		{
			name:         "two snapshots: drift is current minus prior",
			snaps:        []store.PacingSnapshot{{Pacing: 55}, {Pacing: 80}},
			wantDrift:    -25, // newest-first: 55 - 80
			wantBaseline: false,
		},
		{
			name:         "two snapshots: positive drift",
			snaps:        []store.PacingSnapshot{{Pacing: 90}, {Pacing: 70}},
			wantDrift:    20,
			wantBaseline: false,
		},
		{
			name:         "one snapshot: baseline only",
			snaps:        []store.PacingSnapshot{{Pacing: 80}},
			wantBaseline: true,
		},
		{
			name:         "no snapshots: baseline only",
			snaps:        nil,
			wantBaseline: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			drift, baseline := computeDrift(tt.snaps)
			if baseline != tt.wantBaseline {
				t.Fatalf("baselineOnly = %v, want %v", baseline, tt.wantBaseline)
			}
			if !baseline && drift != tt.wantDrift {
				t.Errorf("drift = %v, want %v", drift, tt.wantDrift)
			}
		})
	}
}

func TestRankDriftRows(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	rows := []driftRow{
		{DealID: "small", Drift: f(-5)},
		{DealID: "baseline", Drift: nil, BaselineOnly: true},
		{DealID: "big", Drift: f(-40)},
		{DealID: "mid", Drift: f(12)},
	}
	rankDriftRows(rows)
	// Expect: big (40) > mid (12) > small (5) > baseline (nil, last).
	wantOrder := []string{"big", "mid", "small", "baseline"}
	for i, w := range wantOrder {
		if rows[i].DealID != w {
			t.Errorf("position %d: got %q, want %q (order: %+v)", i, rows[i].DealID, w, dealIDs(rows))
		}
	}
}

func dealIDs(rows []driftRow) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.DealID
	}
	return out
}

func TestParsePacingValue(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		underDeli bool
		want      float64
	}{
		{name: "numeric pacing field", raw: `{"dealId":"D1","pacing":73.5}`, want: 73.5},
		{name: "delivery field fallback", raw: `{"deliveryRate":42}`, want: 42},
		{name: "string-encoded number", raw: `{"pacing":"61.2"}`, want: 61.2},
		{name: "absent -> under-delivering true => 0", raw: `{"dealId":"D1"}`, underDeli: true, want: 0},
		{name: "absent -> under-delivering false => 100", raw: `{"dealId":"D1"}`, underDeli: false, want: 100},
		{name: "empty raw -> fallback false => 100", raw: ``, underDeli: false, want: 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var raw json.RawMessage
			if tt.raw != "" {
				raw = json.RawMessage(tt.raw)
			}
			got := parsePacingValue(raw, tt.underDeli)
			if got != tt.want {
				t.Errorf("parsePacingValue = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIndexPacingByDeal(t *testing.T) {
	// Pacing rows are keyed by internal numeric id (what the endpoint echoes),
	// not the public dealId.
	data := json.RawMessage(`[{"id":5882525,"pacing":10},{"id":5882526,"pacing":20}]`)
	idx := indexPacingByDeal(data)
	if len(idx) != 2 {
		t.Fatalf("expected 2 indexed deals, got %d", len(idx))
	}
	if _, ok := idx["5882525"]; !ok {
		t.Errorf("internal id 5882525 missing from index (keys: %v)", keysOf(idx))
	}
	// Enveloped form, also keyed by internal id.
	env := json.RawMessage(`{"data":[{"id":5882527,"pacing":30}]}`)
	idx2 := indexPacingByDeal(env)
	if _, ok := idx2["5882527"]; !ok {
		t.Errorf("internal id 5882527 missing from enveloped index")
	}
}

func keysOf(m map[string]json.RawMessage) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// TestDriftBaseline_Default verifies the no-since baseline is the most-recent
// EXISTING snapshot. driftBaseline runs BEFORE the current reading is recorded,
// so it only ever sees prior snapshots.
func TestDriftBaseline_Default(t *testing.T) {
	db := openDriftTestStore(t)
	// Only the prior reading exists at baseline-selection time.
	if err := db.RecordPacingSnapshot(store.PacingSnapshot{DealID: "D1", CapturedAt: "2026-06-10T09:00:00Z", Pacing: 70}); err != nil {
		t.Fatalf("RecordPacingSnapshot: %v", err)
	}
	prior, have, err := driftBaseline(db, "D1", "")
	if err != nil {
		t.Fatalf("driftBaseline: %v", err)
	}
	if !have || prior != 70 {
		t.Fatalf("expected prior 70 from the most-recent existing snapshot, got prior=%v have=%v", prior, have)
	}

	// No existing snapshot => baseline-only (first run, before any record).
	db2 := openDriftTestStore(t)
	if _, have, err := driftBaseline(db2, "X", ""); err != nil || have {
		t.Fatalf("expected baseline-only with no prior snapshot, got have=%v err=%v", have, err)
	}
}

// TestDriftBaseline_SinceTodayWithPrior is the regression for the same-day bug:
// with a prior same-day snapshot and --since=today, the baseline must be that
// prior (not the about-to-be-recorded current row, which isn't recorded yet).
func TestDriftBaseline_SinceTodayWithPrior(t *testing.T) {
	db := openDriftTestStore(t)
	if err := db.RecordPacingSnapshot(store.PacingSnapshot{DealID: "D1", CapturedAt: "2026-06-18T10:00:00Z", Pacing: 42}); err != nil {
		t.Fatalf("RecordPacingSnapshot: %v", err)
	}
	prior, have, err := driftBaseline(db, "D1", "2026-06-18")
	if err != nil {
		t.Fatalf("driftBaseline(since today): %v", err)
	}
	if !have || prior != 42 {
		t.Fatalf("expected prior 42 from the same-day snapshot, got prior=%v have=%v", prior, have)
	}
}

// TestDriftBaseline_Since verifies --since selects the snapshot on or before the
// given date as the baseline, rather than the immediately-preceding snapshot.
func TestDriftBaseline_Since(t *testing.T) {
	db := openDriftTestStore(t)
	for _, s := range []store.PacingSnapshot{
		{DealID: "D1", CapturedAt: "2026-06-05T09:00:00Z", Pacing: 60},
		{DealID: "D1", CapturedAt: "2026-06-10T09:00:00Z", Pacing: 80},
		{DealID: "D1", CapturedAt: "2026-06-14T09:00:00Z", Pacing: 90},
		{DealID: "D1", CapturedAt: "2026-06-17T09:00:00Z", Pacing: 55}, // current
	} {
		if err := db.RecordPacingSnapshot(s); err != nil {
			t.Fatalf("RecordPacingSnapshot: %v", err)
		}
	}

	// Baseline against 2026-06-11 => the 06-10 snapshot (80), NOT the
	// second-most-recent (90).
	prior, have, err := driftBaseline(db, "D1", "2026-06-11")
	if err != nil {
		t.Fatalf("driftBaseline(since): %v", err)
	}
	if !have || prior != 80 {
		t.Fatalf("expected --since baseline 80 (06-10 snapshot), got prior=%v have=%v", prior, have)
	}

	// A --since before every snapshot falls back to the oldest (60).
	prior, have, err = driftBaseline(db, "D1", "2026-06-01")
	if err != nil {
		t.Fatalf("driftBaseline(since, old): %v", err)
	}
	if !have || prior != 60 {
		t.Fatalf("expected fallback to oldest snapshot 60, got prior=%v have=%v", prior, have)
	}
}

func openDriftTestStore(t *testing.T) *store.Store {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "data.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.EnsureMaestroTables(); err != nil {
		t.Fatalf("EnsureMaestroTables: %v", err)
	}
	return db
}
