// Copyright 2026 Greg Stellato and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"testing"
)

func TestExpandCells_Cartesian(t *testing.T) {
	cells, truncated := expandCells("250,276", "1,2", "", 0)
	if truncated {
		t.Errorf("did not expect truncation")
	}
	if len(cells) != 4 {
		t.Fatalf("expected 4 cells (2 geos x 2 devices x 1 audience), got %d: %+v", len(cells), cells)
	}
	// Verify the exact set, geo-major then device order.
	want := []sweepCell{
		{Geo: "250", Device: "1"},
		{Geo: "250", Device: "2"},
		{Geo: "276", Device: "1"},
		{Geo: "276", Device: "2"},
	}
	for i, w := range want {
		if cells[i] != w {
			t.Errorf("cell %d = %+v, want %+v", i, cells[i], w)
		}
	}
}

func TestExpandCells_WithAudience(t *testing.T) {
	cells, _ := expandCells("250", "1", "10,20,30", 0)
	if len(cells) != 3 {
		t.Fatalf("expected 3 cells (1x1x3), got %d", len(cells))
	}
}

func TestExpandCells_GeoOnly(t *testing.T) {
	// Only --geo given: sweep geos, one cell each (device/audience collapse).
	cells, _ := expandCells("250,276,840", "", "", 0)
	if len(cells) != 3 {
		t.Fatalf("expected 3 geo-only cells, got %d: %+v", len(cells), cells)
	}
	for _, c := range cells {
		if c.Device != "" || c.Audience != "" {
			t.Errorf("geo-only cell should have empty device/audience, got %+v", c)
		}
	}
}

func TestExpandCells_Truncation(t *testing.T) {
	cells, truncated := expandCells("250,276,840", "1,2", "", 4)
	if !truncated {
		t.Errorf("expected truncation at max-cells=4")
	}
	if len(cells) != 4 {
		t.Fatalf("expected exactly 4 cells after truncation, got %d", len(cells))
	}
}

func TestExpandCells_NoAxes_OneUnconstrainedCell(t *testing.T) {
	// No axes at all: exactly one fully-unconstrained cell (total avails).
	cells, truncated := expandCells("", "", "", 0)
	if truncated {
		t.Errorf("did not expect truncation for a single cell")
	}
	if len(cells) != 1 {
		t.Fatalf("expected 1 unconstrained cell, got %d: %+v", len(cells), cells)
	}
	if cells[0] != (sweepCell{}) {
		t.Errorf("expected an empty (unconstrained) cell, got %+v", cells[0])
	}
}

func TestParseAvails(t *testing.T) {
	tests := []struct {
		name         string
		raw          string
		wantAvail    float64
		wantAuctions float64
	}{
		{
			name:         "verified array of per-day rows is summed",
			raw:          `[{"impressions":1000,"auctions":4000,"day":"1718000000000"},{"impressions":500,"auctions":2000,"day":"1718086400000"}]`,
			wantAvail:    1500,
			wantAuctions: 6000,
		},
		{
			name:         "empty array => zero",
			raw:          `[]`,
			wantAvail:    0,
			wantAuctions: 0,
		},
		{
			name:         "string-encoded numbers",
			raw:          `[{"impressions":"250","auctions":"900"}]`,
			wantAvail:    250,
			wantAuctions: 900,
		},
		{
			name:         "enveloped data array (defensive)",
			raw:          `{"data":[{"impressions":42,"auctions":100}]}`,
			wantAvail:    42,
			wantAuctions: 100,
		},
		{
			name:         "single object (defensive)",
			raw:          `{"impressions":7,"auctions":9}`,
			wantAvail:    7,
			wantAuctions: 9,
		},
		{
			name:         "missing fields => zero",
			raw:          `[{"foo":"bar"}]`,
			wantAvail:    0,
			wantAuctions: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, au := parseAvails(json.RawMessage(tt.raw))
			if a != tt.wantAvail {
				t.Errorf("avails = %v, want %v", a, tt.wantAvail)
			}
			if au != tt.wantAuctions {
				t.Errorf("auctions = %v, want %v", au, tt.wantAuctions)
			}
		})
	}
}

func TestBuildSweepBody_NestedFiltersAndContract(t *testing.T) {
	// geo + device => one cell, AND of two numeric filters in a nested array.
	body := buildSweepBody(sweepCell{Geo: "250", Device: "2"}, "2026-06-10", "2026-06-16")

	if body["useCaseId"] != "ForecastDmkp" {
		t.Errorf("expected useCaseId ForecastDmkp, got %v", body["useCaseId"])
	}
	if body["timezone"] != "UTC" || body["currency"] != "EUR" || body["periodicity"] != "day" {
		t.Errorf("contract scalars wrong: %+v", body)
	}
	if body["startDate"] != "2026-06-10" || body["endDate"] != "2026-06-16" {
		t.Errorf("dates not set on body: %+v", body)
	}

	// Round-trip through JSON so we assert the exact nested structure and that
	// values are numbers (not strings).
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	filters, ok := decoded["filters"].([]any)
	if !ok || len(filters) != 1 {
		t.Fatalf("expected filters to be a 1-element nested array, got %v", decoded["filters"])
	}
	inner, ok := filters[0].([]any)
	if !ok || len(inner) != 2 {
		t.Fatalf("expected inner AND array of 2 filters, got %v", filters[0])
	}
	f0, _ := inner[0].(map[string]any)
	if f0["field"] != "countryId" || f0["operator"] != "IN" {
		t.Errorf("first filter should be countryId IN, got %v", f0)
	}
	vals, ok := f0["values"].([]any)
	if !ok || len(vals) != 1 {
		t.Fatalf("expected values array of 1, got %v", f0["values"])
	}
	// json numbers decode to float64 — a string would decode to string.
	if _, isNum := vals[0].(float64); !isNum {
		t.Errorf("expected numeric value in values array, got %T (%v)", vals[0], vals[0])
	}
	f1, _ := inner[1].(map[string]any)
	if f1["field"] != "deviceTypeId" {
		t.Errorf("second filter should be deviceTypeId, got %v", f1)
	}
}

func TestBuildSweepBody_NoAxes_EmptyFilters(t *testing.T) {
	// Unconstrained cell => filters is an empty array (total avails).
	body := buildSweepBody(sweepCell{}, "2026-06-10", "2026-06-16")
	raw, _ := json.Marshal(body)
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	filters, ok := decoded["filters"].([]any)
	if !ok {
		t.Fatalf("expected filters array, got %T", decoded["filters"])
	}
	if len(filters) != 0 {
		t.Errorf("expected empty filters for an unconstrained cell, got %v", filters)
	}
}

func TestBuildSweepBody_AudienceFilter(t *testing.T) {
	body := buildSweepBody(sweepCell{Geo: "250", Audience: "555"}, "a", "b")
	raw, _ := json.Marshal(body)
	var decoded map[string]any
	_ = json.Unmarshal(raw, &decoded)
	inner := decoded["filters"].([]any)[0].([]any)
	if len(inner) != 2 {
		t.Fatalf("expected geo + audience filters, got %v", inner)
	}
	last := inner[1].(map[string]any)
	if last["field"] != "audienceSegmentId" {
		t.Errorf("expected audienceSegmentId filter, got %v", last)
	}
}
