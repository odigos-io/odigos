package graph

import _ "embed"

// WorkloadsHTML is the standalone Workloads dashboard page served at
// /workloads. It's a committed source file, so embedding it here ships it with
// the frontend Go module and any importer (the enterprise UI) gets it without
// vendoring its own copy.
//
//go:embed workloads.html
var WorkloadsHTML []byte
