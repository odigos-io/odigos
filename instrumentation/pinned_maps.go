package instrumentation

import (
	"os"
	"path/filepath"

	"github.com/cilium/ebpf"
	"github.com/go-logr/logr"
	"golang.org/x/sys/unix"
)

// BPFFsPath is the system path to the BPF file-system.
const bpfFsPath = "/sys/fs/bpf"

type bpfFsMapsManager struct {
	logger            logr.Logger
	mountedFs         bool
	odigletDirCreated bool
	tracesMap         *ebpf.Map
}

func (b *bpfFsMapsManager) TracesMap() (*ebpf.Map, error) {
	if b.tracesMap != nil {
		return b.tracesMap, nil
	}

	if !b.mountedFs && !isBPFFSMounted() {
		if err := mountBpfFs(); err != nil {
			return nil, err
		}

		b.logger.Info("Mounted BPF file-system")
		b.mountedFs = true
	}

	if !b.odigletDirCreated {
		err := os.Mkdir(filepath.Join(bpfFsPath, "odiglet"), 0o755)
		if err != nil {
			return nil, err
		}
		b.odigletDirCreated = true
	}

	spec := &ebpf.MapSpec{
		Type:    ebpf.PerfEventArray,
		Name:    "traces",
		Pinning: ebpf.PinByName,
	}

	m, err := ebpf.NewMapWithOptions(spec, ebpf.MapOptions{
		PinPath: filepath.Join(bpfFsPath, "odiglet"),
	})
	if err != nil {
		return nil, err
	}
	b.tracesMap = m
	return b.tracesMap, nil
}

// mountBpfFs mounts the BPF file-system for the given target.
func mountBpfFs() error {
	// Directory does not exist, create it and mount
	if err := os.MkdirAll(bpfFsPath, 0o755); err != nil {
		return err
	}

	err := unix.Mount(bpfFsPath, bpfFsPath, "bpf", 0, "")
	if err != nil {
		return err
	}

	return nil
}

func isBPFFSMounted() bool {
	var stat unix.Statfs_t
	err := unix.Statfs(bpfFsPath, &stat)
	if err != nil {
		return false
	}

	return stat.Type == unix.BPF_FS_MAGIC
}

// Cleanup removes the BPF file-system if it was mounted, or the odiglet directory if it was created.
func (b *bpfFsMapsManager) Cleanup() error {
	if b.odigletDirCreated {
		if err := os.Remove(filepath.Join(bpfFsPath, "odiglet")); err != nil {
			return err
		}
		b.odigletDirCreated = false
	}

	return nil
}
