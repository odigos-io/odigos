package securitymetrics

import (
	"context"
	"time"

	"github.com/odigos-io/odigos/netmetrics"
)

// TCPHealthFunc returns the current per-edge TCP health. The VM agent passes the enricher's
// ResolveTCPHealth; tests pass a fake. This is the only coupling between the TCP-health
// source and the network layer.
type TCPHealthFunc func() ([]netmetrics.TCPHealth, error)

// TCPHealthSource turns OBI's TCP-health stats (rtt / retransmits / failed-connections) into
// SecurityEvents — the substrate for scan, RST-storm, and latency detection. Like
// NetworkSource it polls and is source-agnostic from there on. It emits, per service→peer
// edge, the DELTA in retransmits/failed-conns since the last poll (so detectors see a rate,
// not an ever-growing cumulative) plus the current average RTT.
type TCPHealthSource struct {
	health   TCPHealthFunc
	interval time.Duration

	prev map[string]netmetrics.TCPHealth // edge key -> last reading (for deltas)
}

func NewTCPHealthSource(health TCPHealthFunc, interval time.Duration) *TCPHealthSource {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	return &TCPHealthSource{health: health, interval: interval, prev: map[string]netmetrics.TCPHealth{}}
}

func (s *TCPHealthSource) Name() string { return "tcp-health" }

func (s *TCPHealthSource) Events(ctx context.Context) <-chan SecurityEvent {
	out := make(chan SecurityEvent, 128)
	go func() {
		defer close(out)
		t := time.NewTicker(s.interval)
		defer t.Stop()
		s.emit(ctx, out)
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				s.emit(ctx, out)
			}
		}
	}()
	return out
}

func (s *TCPHealthSource) emit(ctx context.Context, out chan<- SecurityEvent) {
	rows, err := s.health()
	if err != nil {
		return
	}
	now := time.Now()
	for _, h := range rows {
		key := h.Service + "\x00" + h.Peer + "\x00" + h.ServerPort
		prev := s.prev[key]
		s.prev[key] = h
		// deltas since last poll; cumulative resets (counter < prev) clamp to 0.
		dRetx := h.Retransmits - prev.Retransmits
		if dRetx < 0 {
			dRetx = h.Retransmits
		}
		dFail := h.FailedConns - prev.FailedConns
		if dFail < 0 {
			dFail = h.FailedConns
		}
		ev := SecurityEvent{
			Time:    now,
			Source:  "tcp-health",
			Subject: Subject{Service: h.Service},
			Cat:     CategoryTCPHealth,
			Verb:    "stat",
			Object:  Object{PeerService: h.Peer},
			Attrs: map[string]any{
				"avg_rtt_ms":         h.AvgRttMs,
				"retransmits_delta":  dRetx,
				"failed_conns_delta": dFail,
				"server_port":        h.ServerPort,
			},
		}
		select {
		case out <- ev:
		case <-ctx.Done():
			return
		}
	}
}
