//go:build linux

// Package symbolizeserver runs the native (C/C++/Rust) symbolization engine as a
// node-local service. The collector calls it over a unix socket to symbolize the
// frames the eBPF profiler left as module+offset, so the heavy ELF work (and its
// memory/CPU peaks) lives here — in a process that can scale/OOM independently —
// instead of in the throughput-critical collector pipeline.
//
// On VMs the vm-agent runs this server; on k8s the odiglet DaemonSet does (both
// already have /proc access). The wire contract is JSON over a unix socket:
//
//	POST /symbolize  {"frames":[{pid,module,memoryStart,fileOffset,buildID,addr}]}
//	             ⇒   {"frames":[{name,source}]}   (empty name ⇒ unresolved / parse pending)
package symbolizeserver

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"time"

	"go.uber.org/zap"

	"github.com/odigos-io/odigos/profiles/symbolize"
)

// DefaultSocketPath is the well-known socket the collector dials by default.
const DefaultSocketPath = "/var/odigos/symbolize.sock"

// socketMode makes the socket reachable by the collector (a different uid/container).
const socketMode os.FileMode = 0o666

// Frame is one symbolization request — exactly what the collector has from the
// OTLP profile (the profiler supplies the build-id and offsets).
type Frame struct {
	PID         int    `json:"pid"`
	Module      string `json:"module"`      // mapping basename, e.g. "libCXOPSX00.so"
	MemoryStart uint64 `json:"memoryStart"` // OTLP Mapping.MemoryStart (0 if normalized)
	FileOffset  uint64 `json:"fileOffset"`  // OTLP Mapping.FileOffset
	BuildID     string `json:"buildID"`     // GNU build-id (hex), "" to skip verification
	Addr        uint64 `json:"addr"`        // frame instruction address
}

// Resolved is one symbolization result.
type Resolved struct {
	Name   string `json:"name"`   // demangled function name; "" when unresolved
	Source string `json:"source"` // "symtab" | "dynsym" | ""
}

type symbolizeRequest struct {
	Frames []Frame `json:"frames"`
}
type symbolizeResponse struct {
	Frames []Resolved `json:"frames"`
}

// Server symbolizes frames on behalf of collectors over a unix socket.
type Server struct {
	log        *zap.Logger
	socketPath string
	engine     *symbolize.Symbolizer
	httpSrv    *http.Server
	listener   net.Listener
}

// New builds a server bound to socketPath (DefaultSocketPath if empty). Engine
// options (cache sizes, workers, logger) are passed through to the symbolizer.
func New(socketPath string, log *zap.Logger, opts ...symbolize.Option) *Server {
	if socketPath == "" {
		socketPath = DefaultSocketPath
	}
	if log == nil {
		log = zap.NewNop()
	}
	s := &Server{
		log:        log,
		socketPath: socketPath,
		engine:     symbolize.New(append([]symbolize.Option{symbolize.WithLogger(log)}, opts...)...),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/symbolize", s.handleSymbolize)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	s.httpSrv = &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	return s
}

// Start binds the unix socket and serves until Close. A stale socket from a prior
// run is removed first; the socket is chmod'd so the collector can dial it.
func (s *Server) Start() error {
	_ = os.Remove(s.socketPath)
	if dir := dirOf(s.socketPath); dir != "" {
		_ = os.MkdirAll(dir, 0o755)
	}
	ln, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return err
	}
	if err := os.Chmod(s.socketPath, socketMode); err != nil {
		_ = ln.Close()
		return err
	}
	s.listener = ln
	s.log.Info("symbolize server listening", zap.String("socket", s.socketPath))
	go func() {
		if err := s.httpSrv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.log.Error("symbolize server stopped", zap.Error(err))
		}
	}()
	return nil
}

// Close stops the HTTP server and the symbolization engine.
func (s *Server) Close(ctx context.Context) {
	_ = s.httpSrv.Shutdown(ctx)
	s.engine.Close()
	_ = os.Remove(s.socketPath)
}

func (s *Server) handleSymbolize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req symbolizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	resp := symbolizeResponse{Frames: make([]Resolved, len(req.Frames))}
	for i, f := range req.Frames {
		frame, ok := s.engine.Resolve(f.PID, symbolize.Mapping{
			Name:        f.Module,
			MemoryStart: f.MemoryStart,
			FileOffset:  f.FileOffset,
			BuildID:     f.BuildID,
		}, f.Addr)
		if ok {
			resp.Frames[i] = Resolved{Name: frame.Name, Source: frame.Source}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[:i]
		}
	}
	return ""
}
