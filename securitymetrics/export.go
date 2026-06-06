package securitymetrics

import (
	"encoding/json"
	"os"
	"sync"
)

// FindingSink writes findings somewhere durable so they reach a SIEM/operator beyond the
// live /api/security view. It is what Engine.OnFinding feeds. JSONLSink (below) is the
// built-in file sink; a host can also compose its own (e.g. log + file) with MultiSink.
type FindingSink interface {
	Emit(f Finding)
}

// JSONLSink appends each finding as one JSON line to a file — the simplest SIEM-ingestible
// export (filebeat/promtail/fluentbit tail it). Append-only and mutex-guarded; failures are
// swallowed (export must never block or crash the agent).
type JSONLSink struct {
	mu sync.Mutex
	f  *os.File
}

// NewJSONLSink opens (creating, append mode) the JSONL findings file. Returns nil + error if
// the path can't be opened; the caller should treat a nil sink as "export disabled".
func NewJSONLSink(path string) (*JSONLSink, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o640)
	if err != nil {
		return nil, err
	}
	return &JSONLSink{f: f}, nil
}

func (s *JSONLSink) Emit(f Finding) {
	if s == nil || s.f == nil {
		return
	}
	b, err := json.Marshal(f)
	if err != nil {
		return
	}
	s.mu.Lock()
	_, _ = s.f.Write(append(b, '\n'))
	s.mu.Unlock()
}

// Close flushes and closes the underlying file.
func (s *JSONLSink) Close() error {
	if s == nil || s.f == nil {
		return nil
	}
	return s.f.Close()
}

// MultiSink fans a finding out to several sinks (e.g. a structured logger + a JSONL file).
type MultiSink []FindingSink

func (m MultiSink) Emit(f Finding) {
	for _, s := range m {
		if s != nil {
			s.Emit(f)
		}
	}
}

// SinkFunc adapts a plain function (e.g. a structured-log call) into a FindingSink.
type SinkFunc func(Finding)

func (fn SinkFunc) Emit(f Finding) { fn(f) }
