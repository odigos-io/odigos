package odigossymbolizeprocessor

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// moduleRef is the subset of an OTLP profile Mapping needed to symbolize a frame.
type moduleRef struct {
	Name        string
	MemoryStart uint64
	FileOffset  uint64
	BuildID     string
}

// frameRequest is one frame to symbolize, sent to the symbolize server.
type frameRequest struct {
	pid  int64
	mod  moduleRef
	addr uint64
}

// frameResult is the server's answer for one frame.
type frameResult struct {
	name   string
	source string
	ok     bool
}

// resolver turns native frames into names. The collector implementation is a thin
// client that batches a profile's frames and calls the node-local symbolize server
// over a unix socket — the ELF binary analysis (and its memory/CPU peaks) happen
// in that separate process, never in this throughput-critical pipeline. A server
// that's down or slow degrades gracefully: frames come back unresolved and stay
// module+offset, resolving on a later batch once the server has parsed the binary.
type resolver interface {
	resolveBatch(reqs []frameRequest) []frameResult
	close()
}

// --- JSON wire contract (matches profiles/symbolizeserver) ------------------

type wireFrame struct {
	PID         int64  `json:"pid"`
	Module      string `json:"module"`
	MemoryStart uint64 `json:"memoryStart"`
	FileOffset  uint64 `json:"fileOffset"`
	BuildID     string `json:"buildID"`
	Addr        uint64 `json:"addr"`
}
type wireResolved struct {
	Name   string `json:"name"`
	Source string `json:"source"`
}
type wireRequest struct {
	Frames []wireFrame `json:"frames"`
}
type wireResponse struct {
	Frames []wireResolved `json:"frames"`
}

// rpcResolver calls the symbolize server over its unix socket.
type rpcResolver struct {
	log    *zap.Logger
	url    string
	client *http.Client
}

func newResolver(cfg *Config, logger *zap.Logger) resolver {
	socket := defaultServerEndpoint
	if cfg != nil && cfg.ServerEndpoint != "" {
		socket = cfg.ServerEndpoint
	}
	return &rpcResolver{
		log: logger,
		url: "http://unix/symbolize",
		client: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					var d net.Dialer
					return d.DialContext(ctx, "unix", socket)
				},
			},
		},
	}
}

func (r *rpcResolver) resolveBatch(reqs []frameRequest) []frameResult {
	out := make([]frameResult, len(reqs))
	if len(reqs) == 0 {
		return out
	}
	body := wireRequest{Frames: make([]wireFrame, len(reqs))}
	for i, req := range reqs {
		body.Frames[i] = wireFrame{
			PID: req.pid, Module: req.mod.Name, MemoryStart: req.mod.MemoryStart,
			FileOffset: req.mod.FileOffset, BuildID: req.mod.BuildID, Addr: req.addr,
		}
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return out
	}
	resp, err := r.client.Post(r.url, "application/json", bytes.NewReader(buf))
	if err != nil {
		// server down / not ready — degrade gracefully, retry next batch
		r.log.Debug("symbolize server unreachable; frames stay raw", zap.Error(err))
		return out
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return out
	}
	var decoded wireResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return out
	}
	for i := range out {
		if i >= len(decoded.Frames) {
			break
		}
		if decoded.Frames[i].Name != "" {
			out[i] = frameResult{name: decoded.Frames[i].Name, source: decoded.Frames[i].Source, ok: true}
		}
	}
	return out
}

func (r *rpcResolver) close() { r.client.CloseIdleConnections() }
