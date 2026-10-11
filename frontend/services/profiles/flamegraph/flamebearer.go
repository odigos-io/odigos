package flamegraph

import (
	pyrofb "github.com/grafana/pyroscope/pkg/og/structs/flamebearer"
)

// FlamebearerProfile is the JSON blob served as SourceProfilingResult.profileJson.
// Core flamegraph fields are upstream Pyroscope FlamebearerProfile; Symbols is an Odigos-only extension.
// Symbols is an Odigos-only top-table convenience (self/total per function name); omit when empty via json omitempty.
type FlamebearerProfile struct {
	*pyrofb.FlamebearerProfile
	Symbols []SymbolStats `json:"symbols,omitempty"`
}

// Sample is one profile stack aggregate: frame names root-first and total value (e.g. sample count).
type Sample struct {
	Stack []string
	Value int64
}

const (
	otherName = "other"
)
