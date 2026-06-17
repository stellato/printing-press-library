// Copyright 2026 Greg Stellato and contributors. Licensed under Apache-2.0. See LICENSE.

package store

import (
	"path/filepath"
	"testing"
)

func openTempStore(t *testing.T) *Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "data.db")
	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestEnsureMaestroTablesIdempotent(t *testing.T) {
	s := openTempStore(t)
	if err := s.EnsureMaestroTables(); err != nil {
		t.Fatalf("EnsureMaestroTables (1st): %v", err)
	}
	// Second call must be a no-op, not an error.
	if err := s.EnsureMaestroTables(); err != nil {
		t.Fatalf("EnsureMaestroTables (2nd): %v", err)
	}
}

func TestPacingSnapshotRoundTrip(t *testing.T) {
	s := openTempStore(t)
	if err := s.EnsureMaestroTables(); err != nil {
		t.Fatalf("EnsureMaestroTables: %v", err)
	}

	if err := s.RecordPacingSnapshot(PacingSnapshot{
		DealID: "D1", Name: "Deal One", CapturedAt: "2026-06-16T10:00:00Z",
		Pacing: 80, IsUnderDelivering: true, IsActive: true,
	}); err != nil {
		t.Fatalf("RecordPacingSnapshot prior: %v", err)
	}
	if err := s.RecordPacingSnapshot(PacingSnapshot{
		DealID: "D1", Name: "Deal One", CapturedAt: "2026-06-17T10:00:00Z",
		Pacing: 55, IsUnderDelivering: true, IsActive: true,
	}); err != nil {
		t.Fatalf("RecordPacingSnapshot current: %v", err)
	}

	snaps, err := s.LatestTwoPacingSnapshots("D1")
	if err != nil {
		t.Fatalf("LatestTwoPacingSnapshots: %v", err)
	}
	if len(snaps) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(snaps))
	}
	// Newest first.
	if snaps[0].Pacing != 55 {
		t.Errorf("expected current pacing 55, got %v", snaps[0].Pacing)
	}
	if snaps[1].Pacing != 80 {
		t.Errorf("expected prior pacing 80, got %v", snaps[1].Pacing)
	}
	if snaps[0].Name != "Deal One" {
		t.Errorf("expected name round-trip, got %q", snaps[0].Name)
	}

	// A deal with no snapshots yields an empty slice, no error.
	none, err := s.LatestTwoPacingSnapshots("missing")
	if err != nil {
		t.Fatalf("LatestTwoPacingSnapshots(missing): %v", err)
	}
	if len(none) != 0 {
		t.Errorf("expected 0 snapshots for missing deal, got %d", len(none))
	}
}

func TestLatestSnapshotOnOrBefore(t *testing.T) {
	s := openTempStore(t)
	if err := s.EnsureMaestroTables(); err != nil {
		t.Fatalf("EnsureMaestroTables: %v", err)
	}

	// Three snapshots across three days.
	for _, snap := range []PacingSnapshot{
		{DealID: "D1", Name: "Deal One", CapturedAt: "2026-06-10T09:00:00Z", Pacing: 70},
		{DealID: "D1", Name: "Deal One", CapturedAt: "2026-06-14T09:00:00Z", Pacing: 85},
		{DealID: "D1", Name: "Deal One", CapturedAt: "2026-06-17T09:00:00Z", Pacing: 60},
	} {
		if err := s.RecordPacingSnapshot(snap); err != nil {
			t.Fatalf("RecordPacingSnapshot: %v", err)
		}
	}

	// On-or-before a date that has a same-day snapshot: that day is included
	// (inclusive whole-day upper bound).
	got, err := s.LatestSnapshotOnOrBefore("D1", "2026-06-14")
	if err != nil {
		t.Fatalf("LatestSnapshotOnOrBefore(2026-06-14): %v", err)
	}
	if got == nil || got.Pacing != 85 {
		t.Fatalf("expected the 06-14 snapshot (pacing 85), got %+v", got)
	}

	// A date between snapshots picks the latest snapshot before it.
	got, err = s.LatestSnapshotOnOrBefore("D1", "2026-06-15")
	if err != nil {
		t.Fatalf("LatestSnapshotOnOrBefore(2026-06-15): %v", err)
	}
	if got == nil || got.Pacing != 85 {
		t.Fatalf("expected the 06-14 snapshot for 06-15, got %+v", got)
	}

	// A date before every snapshot falls back to the oldest snapshot.
	got, err = s.LatestSnapshotOnOrBefore("D1", "2026-06-01")
	if err != nil {
		t.Fatalf("LatestSnapshotOnOrBefore(2026-06-01): %v", err)
	}
	if got == nil || got.Pacing != 70 {
		t.Fatalf("expected fallback to oldest snapshot (pacing 70), got %+v", got)
	}

	// A deal with no snapshots yields (nil, nil).
	got, err = s.LatestSnapshotOnOrBefore("missing", "2026-06-17")
	if err != nil {
		t.Fatalf("LatestSnapshotOnOrBefore(missing): %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for a deal with no snapshots, got %+v", got)
	}
}

func TestForecastRunRoundTrip(t *testing.T) {
	s := openTempStore(t)
	if err := s.EnsureMaestroTables(); err != nil {
		t.Fatalf("EnsureMaestroTables: %v", err)
	}

	run := ForecastRun{
		RunID: "run-1", CreatedAt: "2026-06-17T00:00:00Z",
		Geo: "250,276", Format: "1,2", Audience: "",
		Cells: []ForecastCell{
			{Geo: "250", Format: "1", Avails: 1000, Auctions: 2500, Raw: `{"impressions":1000}`},
			{Geo: "276", Format: "2", Avails: 500, Auctions: 1200, Raw: `{"impressions":500}`},
		},
	}
	if err := s.SaveForecastRun(run); err != nil {
		t.Fatalf("SaveForecastRun: %v", err)
	}

	got, err := s.GetForecastRun("run-1")
	if err != nil {
		t.Fatalf("GetForecastRun: %v", err)
	}
	if got == nil {
		t.Fatalf("expected run, got nil")
	}
	if got.Geo != "250,276" || len(got.Cells) != 2 {
		t.Fatalf("unexpected run: geo=%q cells=%d", got.Geo, len(got.Cells))
	}
	if got.Cells[0].Avails != 1000 || got.Cells[1].Auctions != 1200 {
		t.Errorf("cell round-trip mismatch: %+v", got.Cells)
	}

	// Re-save replaces cells rather than duplicating them.
	run.Cells = run.Cells[:1]
	if err := s.SaveForecastRun(run); err != nil {
		t.Fatalf("SaveForecastRun (replace): %v", err)
	}
	got, err = s.GetForecastRun("run-1")
	if err != nil {
		t.Fatalf("GetForecastRun (replace): %v", err)
	}
	if len(got.Cells) != 1 {
		t.Errorf("expected cells replaced to 1, got %d", len(got.Cells))
	}

	// Missing run yields (nil, nil).
	missing, err := s.GetForecastRun("nope")
	if err != nil {
		t.Fatalf("GetForecastRun(nope): %v", err)
	}
	if missing != nil {
		t.Errorf("expected nil for missing run, got %+v", missing)
	}
}

func TestDealApplyQuota(t *testing.T) {
	s := openTempStore(t)
	if err := s.EnsureMaestroTables(); err != nil {
		t.Fatalf("EnsureMaestroTables: %v", err)
	}

	// Empty day -> zero quota.
	q, err := s.GetDealApplyQuota("2026-06-17")
	if err != nil {
		t.Fatalf("GetDealApplyQuota: %v", err)
	}
	if q.Creates != 0 || q.Updates != 0 {
		t.Errorf("expected zero quota, got %+v", q)
	}

	q, err = s.IncDealApplyQuota("2026-06-17", 3, 10)
	if err != nil {
		t.Fatalf("IncDealApplyQuota: %v", err)
	}
	if q.Creates != 3 || q.Updates != 10 {
		t.Errorf("after first inc expected 3/10, got %+v", q)
	}
	q, err = s.IncDealApplyQuota("2026-06-17", 2, 5)
	if err != nil {
		t.Fatalf("IncDealApplyQuota (2): %v", err)
	}
	if q.Creates != 5 || q.Updates != 15 {
		t.Errorf("after second inc expected 5/15, got %+v", q)
	}
}

func TestRawString(t *testing.T) {
	if RawString(nil) != "" {
		t.Errorf("expected empty string for nil raw")
	}
	if got := RawString([]byte(`{"a":1}`)); got != `{"a":1}` {
		t.Errorf("RawString mismatch: %q", got)
	}
}
