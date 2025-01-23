package model

import "go.opentelemetry.io/collector/pdata/ptrace"

type TracesPayload struct {
	TraceData ptrace.Traces
}
