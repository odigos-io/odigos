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

	// Create the eBPF map
	spec := &ebpf.MapSpec{
		Type: ebpf.PerfEventArray,
		Name: "traces",
	}

	m, err := ebpf.NewMap(spec)
	if err != nil {
		return nil, err
	}
	b.tracesMap = m

	// Start the FD server
	server := &unixfd.Server{
		SocketPath: unixfd.DefaultSocketPath,
		Logger:     b.logger,
		FDProvider: func() int {
			return m.FD()
		},
	}

	// Run server in background
	go func() {
		ctx := context.Background()
		if err := server.Run(ctx); err != nil {
			b.logger.Error(err, "unixfd server failed")
		}
	}()

	b.logger.Info("TracesMap created, FD server started",
		"socket", unixfd.DefaultSocketPath,
		"map_fd", m.FD())

	return b.tracesMap, nil
}
