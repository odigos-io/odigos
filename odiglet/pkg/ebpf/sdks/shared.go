package sdks

import (
	"fmt"
	"sync"

	"github.com/odigos-io/odigos/common/consts"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// sharedTracesGRPCConn returns a process-wide gRPC ClientConn to the local
// data-collection OTLP endpoint, dialed lazily on first call and reused by
// every per-PID OTLP exporter created by the eBPF SDK factories in this
// package.
//
// Why shared: each instrumented PID previously created its own otlptracegrpc
// client, which opened a dedicated TCP connection and spawned ~3 background
// goroutines (resolver watcher, CallbackSerializer, subConn transport) plus
// ~200-500 KB of transport state. On dense nodes with hundreds of
// instrumented processes this was a dominant driver of odiglet's memory
// footprint and goroutine count. Passing one *grpc.ClientConn to each
// exporter via otlptracegrpc.WithGRPCConn collapses all of that to a single
// TCP connection and ~3 goroutines total.
//
// Lifecycle: grpc.NewClient is lazy and does not block on dial, so the first
// call does not stall even if data-collection is not yet ready. The conn is
// never explicitly closed — odiglet owns it for the lifetime of the process
// and the OS reclaims the socket on exit. This is consistent with
// otlptracegrpc.WithGRPCConn's documented contract: the OTLP exporter
// Shutdown path explicitly does not close a conn provided via WithGRPCConn,
// so per-PID Close never tears down this shared transport.
func sharedTracesGRPCConn() (*grpc.ClientConn, error) {
	sharedTracesConnOnce.Do(func() {
		// data-collection runs as a sidecar container in the same pod as
		// odiglet, so localhost:<OTLPPort> is the in-pod loopback path.
		addr := fmt.Sprintf("localhost:%d", consts.OTLPPort)
		conn, err := grpc.NewClient(
			addr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			sharedTracesConnErr = fmt.Errorf("failed to dial shared data-collection otlp conn at %s: %w", addr, err)
			return
		}
		sharedTracesConn = conn
	})
	return sharedTracesConn, sharedTracesConnErr
}

var (
	sharedTracesConnOnce sync.Once
	sharedTracesConn     *grpc.ClientConn
	sharedTracesConnErr  error
)
