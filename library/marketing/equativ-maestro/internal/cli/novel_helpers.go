// Copyright 2026 Greg Stellato and contributors. Licensed under Apache-2.0. See LICENSE.

// Shared helpers for the hand-authored Maestro "transcendence" commands
// (deals drift, reconcile, forecast sweep, deals apply, deals funnel-rank).
// These decode loosely-typed deal/report JSON defensively so a missing or
// differently-typed field never drops a row.
package cli

import (
	"encoding/json"
	"strconv"
)

// rawBool decodes a json.RawMessage as a boolean, accepting true/false, 1/0,
// and the strings "true"/"1". Returns false for anything unrecognized.
func rawBool(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var b bool
	if json.Unmarshal(raw, &b) == nil {
		return b
	}
	var n float64
	if json.Unmarshal(raw, &n) == nil {
		return n != 0
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s == "true" || s == "1"
	}
	return false
}

// rawStringField returns the first present, non-empty string value among keys.
// A numeric value is rendered to its string form so an int dealId still yields
// a usable identifier.
func rawStringField(obj map[string]json.RawMessage, keys ...string) string {
	for _, k := range keys {
		raw, ok := obj[k]
		if !ok || len(raw) == 0 {
			continue
		}
		var s string
		if json.Unmarshal(raw, &s) == nil && s != "" {
			return s
		}
		var n json.Number
		if json.Unmarshal(raw, &n) == nil && n.String() != "" {
			return n.String()
		}
	}
	return ""
}

// rawFloatField returns the first present numeric field among keys, accepting
// JSON numbers and numeric strings.
func rawFloatField(obj map[string]json.RawMessage, keys ...string) (float64, bool) {
	for _, k := range keys {
		raw, ok := obj[k]
		if !ok || len(raw) == 0 {
			continue
		}
		var f float64
		if json.Unmarshal(raw, &f) == nil {
			return f, true
		}
		var s string
		if json.Unmarshal(raw, &s) == nil {
			if v, err := strconv.ParseFloat(s, 64); err == nil {
				return v, true
			}
		}
	}
	return 0, false
}

// rawIntField returns the first present integer-valued field among keys.
func rawIntField(obj map[string]json.RawMessage, keys ...string) (int64, bool) {
	for _, k := range keys {
		raw, ok := obj[k]
		if !ok || len(raw) == 0 {
			continue
		}
		var n int64
		if json.Unmarshal(raw, &n) == nil {
			return n, true
		}
		var s string
		if json.Unmarshal(raw, &s) == nil {
			if v, err := strconv.ParseInt(s, 10, 64); err == nil {
				return v, true
			}
		}
	}
	return 0, false
}
