// Package netmetrics provides shared, environment-agnostic enrichment of OBI
// network flows with process/service identity. It is imported by BOTH the VM
// agent and odiglet (k8s) so the resolution logic is written exactly once.
//
// Layering (no duplication):
//   - EndpointResolver  : local socket endpoint (ip:port) -> owning PID, from /proc
//     (github.com/prometheus/procfs — already an odigos dependency). Pure, env-agnostic.
//   - ServiceResolver   : joins a flow to identity, composing EndpointResolver with two
//     host-injected lookups (PID->service and peerIP->service). The VM agent injects its
//     PID->Source table; odiglet injects its k8s informer. Only the *source* of identity
//     differs per environment — the join logic here is shared.
//
// Once the OBI fork stamps process.pid on the flow (socktrack_map), EndpointResolver
// becomes optional: callers can pass the flow's PID directly to ServiceResolver.
package netmetrics

import (
	"net"
	"regexp"
	"strconv"
	"sync"

	"github.com/prometheus/procfs"
)

var sockInodeRe = regexp.MustCompile(`socket:\[(\d+)\]`)

// Endpoint identifies the process owning a local socket endpoint.
type Endpoint struct {
	PID  int
	Comm string
	Cmd  string
}

// EndpointResolver maps a local socket endpoint ("ip:port") to the owning process,
// by joining /proc/net/{tcp,tcp6,udp,udp6} (inode -> 5-tuple) with /proc/<pid>/fd
// (socket:[inode] -> pid). This is the Falco/libsinsp model and needs no eBPF.
type EndpointResolver struct {
	fs procfs.FS

	mu    sync.RWMutex
	table map[string]Endpoint // "ip:port" -> Endpoint
}

// NewEndpointResolver creates a resolver over the default /proc mount.
func NewEndpointResolver() (*EndpointResolver, error) {
	fs, err := procfs.NewDefaultFS()
	if err != nil {
		return nil, err
	}
	return &EndpointResolver{fs: fs, table: map[string]Endpoint{}}, nil
}

// Refresh rebuilds the endpoint->process table from /proc. Call periodically and/or
// on cache-miss. Long-lived flows resolve reliably; very short-lived sockets may be
// missed (best-effort) — the OBI socktrack path removes that race in production.
func (r *EndpointResolver) Refresh() error {
	inodeEP := map[uint64]string{} // socket inode -> "ip:port"
	add := func(lines procfs.NetTCP) {
		for _, l := range lines {
			inodeEP[l.Inode] = net.JoinHostPort(l.LocalAddr.String(), strconv.FormatUint(l.LocalPort, 10))
		}
	}
	if t, err := r.fs.NetTCP(); err == nil {
		add(t)
	}
	if t, err := r.fs.NetTCP6(); err == nil {
		add(t)
	}
	if u, err := r.fs.NetUDP(); err == nil {
		add(procfs.NetTCP(u))
	}
	if u, err := r.fs.NetUDP6(); err == nil {
		add(procfs.NetTCP(u))
	}

	procs, err := r.fs.AllProcs()
	if err != nil {
		return err
	}
	tbl := make(map[string]Endpoint, len(inodeEP))
	for _, p := range procs {
		targets, err := p.FileDescriptorTargets()
		if err != nil {
			continue // process exited or no permission
		}
		comm, _ := p.Comm()
		cmdParts, _ := p.CmdLine()
		cmd := joinArgs(cmdParts)
		for _, t := range targets {
			m := sockInodeRe.FindStringSubmatch(t)
			if m == nil {
				continue
			}
			inode, _ := strconv.ParseUint(m[1], 10, 64)
			if ep, ok := inodeEP[inode]; ok {
				tbl[ep] = Endpoint{PID: p.PID, Comm: comm, Cmd: cmd}
			}
		}
	}
	r.mu.Lock()
	r.table = tbl
	r.mu.Unlock()
	return nil
}

// Lookup returns the process owning the local endpoint ip:port, trying the exact
// endpoint and then wildcard listen sockets (0.0.0.0:port / [::]:port).
func (r *EndpointResolver) Lookup(ip string, port int) (Endpoint, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if ep, ok := r.table[net.JoinHostPort(ip, strconv.Itoa(port))]; ok {
		return ep, true
	}
	if ep, ok := r.table[net.JoinHostPort("0.0.0.0", strconv.Itoa(port))]; ok {
		return ep, true
	}
	if ep, ok := r.table[net.JoinHostPort("::", strconv.Itoa(port))]; ok {
		return ep, true
	}
	return Endpoint{}, false
}

// Size returns the number of resolved local endpoints (for observability).
func (r *EndpointResolver) Size() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.table)
}

// EndpointWithPort pairs an Endpoint with the local port it was bound to.
type EndpointWithPort struct {
	Endpoint
	Key  string // "ip:port"
	Port int
}

// Snapshot returns a copy of the current endpoint table. Used by callers that need
// to derive their own PID->service mapping from endpoint metadata (comm/cmdline/port).
func (r *EndpointResolver) Snapshot() []EndpointWithPort {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]EndpointWithPort, 0, len(r.table))
	for k, ep := range r.table {
		port := 0
		if i := lastColon(k); i >= 0 {
			port, _ = strconv.Atoi(k[i+1:])
		}
		out = append(out, EndpointWithPort{Endpoint: ep, Key: k, Port: port})
	}
	return out
}

func lastColon(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == ':' {
			return i
		}
	}
	return -1
}

func joinArgs(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += " "
		}
		out += p
	}
	return out
}
