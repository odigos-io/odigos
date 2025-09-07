package ebpf

import (
	"context"

	"github.com/cilium/ebpf"
	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/common/unixfd"
)

// BPFFsPath is the system path to the BPF file-system.
const bpfFsPath = "/sys/fs/bpf"

type bpfFsMapsManager struct {
	logger    logr.Logger
	mountedFs bool
	tracesMap *ebpf.Map
}

func (b *bpfFsMapsManager) TracesMap() (*ebpf.Map, error) {
	if b.tracesMap != nil {
		return b.tracesMap, nil
	}

	spec := &ebpf.MapSpec{
		Type: ebpf.PerfEventArray,
		Name: "traces",
	}
	m, err := ebpf.NewMap(spec)
	if err != nil {
		return nil, err
	}
	b.tracesMap = m

	// --- Start the Unix FD server in background ---
	socketPath := "/var/exchange/exchange.sock"
	server := &unixfd.Server{
		SocketPath: socketPath,
		Logger:     b.logger,
		FDProvider: func() int { return m.FD() },
	}

	// Run server in goroutine so TracesMap() is non-blocking
	ctx := context.Background() // replace with odiglet-wide context if you have one
	go func() {
		if err := server.Run(ctx); err != nil {
			b.logger.Error(err, "unixfd server exited with error")
		}
	}()

	b.logger.Info("TracesMap created and unixfd server started",
		"socket", socketPath, "map_name", spec.Name)

	return b.tracesMap, nil
}
