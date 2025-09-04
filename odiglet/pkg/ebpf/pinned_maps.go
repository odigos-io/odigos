package ebpf

import (
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/cilium/ebpf"
	"github.com/go-logr/logr"
	"golang.org/x/sys/unix"
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

	if !b.mountedFs && !isBPFFSMounted() {
		if err := mountBpfFs(); err != nil {
			return nil, err
		}

		b.logger.Info("Mounted BPF file-system")
		b.mountedFs = true
	}

	odigletPath := filepath.Join(bpfFsPath, "odiglet")
	if _, err := os.Stat(odigletPath); os.IsNotExist(err) {
		err := os.Mkdir(odigletPath, 0o755)
		if err != nil {
			return nil, err
		}
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

	// --- BLOCKING SOCKET HANDOFF ---
	socketPath := "/var/exchange/exchange.sock"
	_ = os.Remove(socketPath)

	addr, err := net.ResolveUnixAddr("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("resolve unix addr: %w", err)
	}

	ul, err := net.ListenUnix("unix", addr)
	if err != nil {
		return nil, fmt.Errorf("listen unix: %w", err)
	}
	defer ul.Close()

	b.logger.Info("odiglet waiting for data-collection to connect", "socket", socketPath)

	conn, err := ul.AcceptUnix()
	if err != nil {
		return nil, fmt.Errorf("accept unix: %w", err)
	}
	defer conn.Close()

	if err := sendFD(conn, m.FD()); err != nil {
		return nil, fmt.Errorf("sendFD failed: %w", err)
	}
	b.logger.Info("✅ FD sent to data-collection")

	return b.tracesMap, nil
}

func sendFD(c *net.UnixConn, fd int) error {
	controlMessage := unix.UnixRights(fd)
	_, _, err := c.WriteMsgUnix([]byte("x"), controlMessage, nil)
	return err
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
