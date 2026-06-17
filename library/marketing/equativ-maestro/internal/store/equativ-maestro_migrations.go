// Copyright 2026 Greg Stellato and contributors. Licensed under Apache-2.0. See LICENSE.

// Custom store tables for the hand-authored Maestro "transcendence" commands
// (deals drift, forecast sweep, deals apply). These tables are created lazily
// by EnsureMaestroTables so they do not depend on the generated migrate()
// pipeline and survive future regeneration of store.go. Each novel command
// calls EnsureMaestroTables() once before touching these tables.
package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// EnsureMaestroTables lazily creates the auxiliary tables used by the novel
// Maestro commands. Idempotent (CREATE TABLE IF NOT EXISTS), safe to call on
// every command invocation. Serialized through writeMu like other writes.
func (s *Store) EnsureMaestroTables() error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS deal_pacing_snapshots (
			deal_id TEXT,
			dealId TEXT,
			name TEXT,
			captured_at TEXT,
			pacing REAL,
			is_under_delivering INTEGER,
			is_active INTEGER
		)`,
		`CREATE INDEX IF NOT EXISTS idx_deal_pacing_snapshots_deal_id
			ON deal_pacing_snapshots(deal_id, captured_at)`,
		`CREATE TABLE IF NOT EXISTS forecast_sweep_runs (
			run_id TEXT PRIMARY KEY,
			created_at TEXT,
			geo TEXT,
			format TEXT,
			audience TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS forecast_sweep_cells (
			run_id TEXT,
			geo TEXT,
			format TEXT,
			audience TEXT,
			avails REAL,
			auctions REAL,
			raw TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_forecast_sweep_cells_run_id
			ON forecast_sweep_cells(run_id)`,
		`CREATE TABLE IF NOT EXISTS deals_apply_quota (
			day TEXT PRIMARY KEY,
			creates INTEGER,
			updates INTEGER
		)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("ensure maestro tables: %w", err)
		}
	}
	return nil
}

// PacingSnapshot is one recorded pacing reading for a deal.
type PacingSnapshot struct {
	DealID            string
	Name              string
	CapturedAt        string
	Pacing            float64
	IsUnderDelivering bool
	IsActive          bool
}

// RecordPacingSnapshot inserts one pacing snapshot row. capturedAt should be an
// RFC3339 / sortable timestamp string so LatestTwoPacingSnapshots can order by
// it lexically.
func (s *Store) RecordPacingSnapshot(snap PacingSnapshot) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_, err := s.db.Exec(
		`INSERT INTO deal_pacing_snapshots
			(deal_id, dealId, name, captured_at, pacing, is_under_delivering, is_active)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		snap.DealID, snap.DealID, snap.Name, snap.CapturedAt, snap.Pacing,
		boolToInt(snap.IsUnderDelivering), boolToInt(snap.IsActive),
	)
	if err != nil {
		return fmt.Errorf("record pacing snapshot: %w", err)
	}
	return nil
}

// LatestTwoPacingSnapshots returns up to the two most recent snapshots for a
// deal, ordered newest-first. The first element (index 0) is the current
// reading; the second (index 1), when present, is the prior baseline. Returns
// an empty slice when the deal has no snapshots. NULL-safe scans guard against
// rows written by older schema versions.
func (s *Store) LatestTwoPacingSnapshots(dealID string) ([]PacingSnapshot, error) {
	rows, err := s.db.Query(
		`SELECT COALESCE(deal_id, ''), COALESCE(name, ''), COALESCE(captured_at, ''),
		        COALESCE(pacing, 0), COALESCE(is_under_delivering, 0), COALESCE(is_active, 0)
		   FROM deal_pacing_snapshots
		  WHERE deal_id = ?
		  ORDER BY captured_at DESC, rowid DESC
		  LIMIT 2`,
		dealID,
	)
	if err != nil {
		return nil, fmt.Errorf("latest pacing snapshots: %w", err)
	}
	defer rows.Close()

	out := make([]PacingSnapshot, 0, 2)
	for rows.Next() {
		var snap PacingSnapshot
		var under, active int
		if err := rows.Scan(&snap.DealID, &snap.Name, &snap.CapturedAt, &snap.Pacing, &under, &active); err != nil {
			return nil, fmt.Errorf("scan pacing snapshot: %w", err)
		}
		snap.IsUnderDelivering = under != 0
		snap.IsActive = active != 0
		out = append(out, snap)
	}
	return out, rows.Err()
}

// LatestSnapshotOnOrBefore returns the most recent snapshot for a deal whose
// captured_at is on or before the given date (YYYY-MM-DD, inclusive of the whole
// day). When no snapshot exists on or before that date it falls back to the
// deal's oldest snapshot. Returns (nil, nil) when the deal has no snapshots at
// all. This backs `deals drift --since`: it selects the drift baseline by date
// instead of "the second-most-recent snapshot". NULL-safe scans guard against
// rows written by older schema versions.
func (s *Store) LatestSnapshotOnOrBefore(dealID, date string) (*PacingSnapshot, error) {
	scan := func(rows *sql.Rows) (*PacingSnapshot, error) {
		defer rows.Close()
		if !rows.Next() {
			return nil, rows.Err()
		}
		var snap PacingSnapshot
		var under, active int
		if err := rows.Scan(&snap.DealID, &snap.Name, &snap.CapturedAt, &snap.Pacing, &under, &active); err != nil {
			return nil, fmt.Errorf("scan pacing snapshot: %w", err)
		}
		snap.IsUnderDelivering = under != 0
		snap.IsActive = active != 0
		return &snap, rows.Err()
	}

	// captured_at is stored as an RFC3339 timestamp; "<date>T23:59:59.999999999Z"
	// is an inclusive upper bound on the whole day that sorts correctly lexically.
	upper := date + "T23:59:59.999999999Z"
	rows, err := s.db.Query(
		`SELECT COALESCE(deal_id, ''), COALESCE(name, ''), COALESCE(captured_at, ''),
		        COALESCE(pacing, 0), COALESCE(is_under_delivering, 0), COALESCE(is_active, 0)
		   FROM deal_pacing_snapshots
		  WHERE deal_id = ? AND captured_at <= ?
		  ORDER BY captured_at DESC, rowid DESC
		  LIMIT 1`,
		dealID, upper,
	)
	if err != nil {
		return nil, fmt.Errorf("latest snapshot on or before: %w", err)
	}
	snap, err := scan(rows)
	if err != nil {
		return nil, err
	}
	if snap != nil {
		return snap, nil
	}

	// Nothing on or before the date: fall back to the oldest snapshot.
	oldest, err := s.db.Query(
		`SELECT COALESCE(deal_id, ''), COALESCE(name, ''), COALESCE(captured_at, ''),
		        COALESCE(pacing, 0), COALESCE(is_under_delivering, 0), COALESCE(is_active, 0)
		   FROM deal_pacing_snapshots
		  WHERE deal_id = ?
		  ORDER BY captured_at ASC, rowid ASC
		  LIMIT 1`,
		dealID,
	)
	if err != nil {
		return nil, fmt.Errorf("oldest pacing snapshot: %w", err)
	}
	return scan(oldest)
}

// ForecastRun is a persisted forecast-sweep run header plus its cells.
type ForecastRun struct {
	RunID     string
	CreatedAt string
	Geo       string
	Format    string
	Audience  string
	Cells     []ForecastCell
}

// ForecastCell is one targeting-matrix cell within a forecast run. Avails and
// Auctions are summed from the inventoryInsights response (impressions and
// auctions respectively); the response carries no CPM, so none is stored.
type ForecastCell struct {
	Geo      string
	Format   string
	Audience string
	Avails   float64
	Auctions float64
	Raw      string
}

// SaveForecastRun persists a run header and all of its cells in one
// transaction. Re-saving the same run_id replaces its prior cells.
func (s *Store) SaveForecastRun(run ForecastRun) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("save forecast run: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(
		`INSERT INTO forecast_sweep_runs (run_id, created_at, geo, format, audience)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(run_id) DO UPDATE SET
			created_at = excluded.created_at,
			geo = excluded.geo,
			format = excluded.format,
			audience = excluded.audience`,
		run.RunID, run.CreatedAt, run.Geo, run.Format, run.Audience,
	); err != nil {
		return fmt.Errorf("save forecast run header: %w", err)
	}

	if _, err := tx.Exec(`DELETE FROM forecast_sweep_cells WHERE run_id = ?`, run.RunID); err != nil {
		return fmt.Errorf("clear forecast cells: %w", err)
	}

	for _, cell := range run.Cells {
		if _, err := tx.Exec(
			`INSERT INTO forecast_sweep_cells
				(run_id, geo, format, audience, avails, auctions, raw)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			run.RunID, cell.Geo, cell.Format, cell.Audience, cell.Avails, cell.Auctions, cell.Raw,
		); err != nil {
			return fmt.Errorf("save forecast cell: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("save forecast run: commit: %w", err)
	}
	return nil
}

// GetForecastRun loads a run header and its cells. Returns (nil, nil) when no
// run with that id exists.
func (s *Store) GetForecastRun(runID string) (*ForecastRun, error) {
	run := &ForecastRun{}
	var geo, format, audience sql.NullString
	var createdAt sql.NullString
	err := s.db.QueryRow(
		`SELECT run_id, COALESCE(created_at, ''), geo, format, audience
		   FROM forecast_sweep_runs WHERE run_id = ?`,
		runID,
	).Scan(&run.RunID, &createdAt, &geo, &format, &audience)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get forecast run: %w", err)
	}
	run.CreatedAt = createdAt.String
	run.Geo = geo.String
	run.Format = format.String
	run.Audience = audience.String

	rows, err := s.db.Query(
		`SELECT geo, format, audience, COALESCE(avails, 0), COALESCE(auctions, 0), COALESCE(raw, '')
		   FROM forecast_sweep_cells WHERE run_id = ?
		   ORDER BY rowid ASC`,
		runID,
	)
	if err != nil {
		return nil, fmt.Errorf("get forecast cells: %w", err)
	}
	defer rows.Close()
	run.Cells = make([]ForecastCell, 0)
	for rows.Next() {
		var cell ForecastCell
		var g, f, a sql.NullString
		if err := rows.Scan(&g, &f, &a, &cell.Avails, &cell.Auctions, &cell.Raw); err != nil {
			return nil, fmt.Errorf("scan forecast cell: %w", err)
		}
		cell.Geo = g.String
		cell.Format = f.String
		cell.Audience = a.String
		run.Cells = append(run.Cells, cell)
	}
	return run, rows.Err()
}

// DealApplyQuota is the recorded create/update count for one calendar day.
type DealApplyQuota struct {
	Day     string
	Creates int
	Updates int
}

// GetDealApplyQuota returns the recorded quota usage for a day. A day with no
// row yields a zero-valued quota (not an error).
func (s *Store) GetDealApplyQuota(day string) (DealApplyQuota, error) {
	q := DealApplyQuota{Day: day}
	var creates, updates sql.NullInt64
	err := s.db.QueryRow(
		`SELECT COALESCE(creates, 0), COALESCE(updates, 0) FROM deals_apply_quota WHERE day = ?`,
		day,
	).Scan(&creates, &updates)
	if err == sql.ErrNoRows {
		return q, nil
	}
	if err != nil {
		return q, fmt.Errorf("get deal apply quota: %w", err)
	}
	q.Creates = int(creates.Int64)
	q.Updates = int(updates.Int64)
	return q, nil
}

// IncDealApplyQuota atomically adds to the recorded create/update counts for a
// day and returns the new totals.
func (s *Store) IncDealApplyQuota(day string, creates, updates int) (DealApplyQuota, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_, err := s.db.Exec(
		`INSERT INTO deals_apply_quota (day, creates, updates)
		 VALUES (?, ?, ?)
		 ON CONFLICT(day) DO UPDATE SET
			creates = creates + excluded.creates,
			updates = updates + excluded.updates`,
		day, creates, updates,
	)
	if err != nil {
		return DealApplyQuota{}, fmt.Errorf("inc deal apply quota: %w", err)
	}
	q := DealApplyQuota{Day: day}
	var c, u sql.NullInt64
	if err := s.db.QueryRow(
		`SELECT COALESCE(creates, 0), COALESCE(updates, 0) FROM deals_apply_quota WHERE day = ?`,
		day,
	).Scan(&c, &u); err != nil {
		return DealApplyQuota{}, fmt.Errorf("read back deal apply quota: %w", err)
	}
	q.Creates = int(c.Int64)
	q.Updates = int(u.Int64)
	return q, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// RawString is a tiny helper so callers can store a json.RawMessage as TEXT,
// normalizing nil/empty to the empty string.
func RawString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	return string(raw)
}
