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
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/prometheus/procfs"
)

var sockInodeRe = regexp.MustCompile(`socket:\[(\d+)\]`)

// TCP socket states in /proc/net/tcp (hex in the file; procfs parses to these values).
const (
	tcpEstablished uint64 = 0x01
	tcpListen      uint64 = 0x0A
)

// Endpoint identifies the process owning a local socket endpoint.
type Endpoint struct {
	PID  int
	Comm string
	Cmd  string

	// boundKey is the table key this Endpoint was stored under; used internally so a
	// concrete-IP listen key is not overwritten by a bare wildcard key for the same port.
	boundKey string
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
//
// It reads the socket table of EVERY network namespace present (one representative pid
// per netns), so processes inside containers — which live in their own netns and whose
// sockets are absent from the host's /proc/net/tcp — resolve too. Socket inodes are
// global, so the inode->endpoint map merges cleanly across namespaces.
//
// Wildcard listen sockets (0.0.0.0:port / [::]:port) are also indexed under each
// concrete local IP observed in their namespace, so a server reachable as
// containerIP:port resolves to its listening process even when many containers listen
// on the same port in different namespaces (which would otherwise collide on the bare
// "0.0.0.0:port" key).
func (r *EndpointResolver) Refresh() error {
	procs, err := r.fs.AllProcs()
	if err != nil {
		return err
	}

	// One representative pid per distinct network namespace (one of them is the host
	// netns). Reading a netns's /proc/<pid>/net/tcp{,6} is dominated by the kernel
	// regenerating that namespace's socket table.
	repPids := []int{}
	seenNetns := map[string]struct{}{}
	for _, p := range procs {
		ns, err := os.Readlink("/proc/" + strconv.Itoa(p.PID) + "/ns/net")
		if err != nil {
			continue
		}
		if _, dup := seenNetns[ns]; dup {
			continue
		}
		seenNetns[ns] = struct{}{}
		repPids = append(repPids, p.PID)
	}

	// indexNS reads one namespace and returns its inode -> [ip:port keys] fragment.
	// For TCP only LISTEN/ESTABLISHED sockets are kept (skip TIME_WAIT churn → fast);
	// UDP has no such states so all are kept. Wildcard listeners are synthesized under
	// each concrete local IP observed in the namespace (so containerIP:port resolves
	// even though the listen socket is bound to 0.0.0.0).
	indexNS := func(fs procfs.FS) map[uint64][]string {
		frag := map[uint64][]string{}
		ingest := func(lines procfs.NetTCP, tcpStates bool) {
			keep := func(st uint64) bool { return !tcpStates || st == tcpListen || st == tcpEstablished }
			var nsIPs []string
			seen := map[string]struct{}{}
			for _, l := range lines {
				if !keep(l.St) || l.LocalAddr.IsUnspecified() {
					continue
				}
				s := l.LocalAddr.String()
				if _, dup := seen[s]; !dup {
					seen[s] = struct{}{}
					nsIPs = append(nsIPs, s)
				}
			}
			for _, l := range lines {
				if !keep(l.St) {
					continue
				}
				port := strconv.FormatUint(l.LocalPort, 10)
				if l.LocalAddr.IsUnspecified() {
					frag[l.Inode] = append(frag[l.Inode], net.JoinHostPort(l.LocalAddr.String(), port))
					for _, ip := range nsIPs {
						frag[l.Inode] = append(frag[l.Inode], net.JoinHostPort(ip, port))
					}
				} else {
					frag[l.Inode] = append(frag[l.Inode], net.JoinHostPort(l.LocalAddr.String(), port))
				}
			}
		}
		if t, err := fs.NetTCP(); err == nil {
			ingest(t, true)
		}
		if t, err := fs.NetTCP6(); err == nil {
			ingest(t, true)
		}
		if u, err := fs.NetUDP(); err == nil {
			ingest(procfs.NetTCP(u), false)
		}
		if u, err := fs.NetUDP6(); err == nil {
			ingest(procfs.NetTCP(u), false)
		}
		return frag
	}

	// inode -> one or more "ip:port" keys, read serially per namespace. Concurrent reads
	// of different namespaces' /proc/net/tcp contend heavily on kernel locks and can be
	// far slower than serial, so namespaces are read one at a time. (Socket inodes are
	// global, so per-namespace fragments merge cleanly.) Under extreme container/socket
	// counts this is the /proc approach's cost ceiling; the eBPF socktrack path removes it.
	inodeEP := map[uint64][]string{}
	for _, pid := range repPids {
		fs, err := procfs.NewFS("/proc/" + strconv.Itoa(pid))
		if err != nil {
			continue
		}
		for inode, keys := range indexNS(fs) {
			inodeEP[inode] = append(inodeEP[inode], keys...)
		}
	}

	// socket inode -> owning pid, by walking every pid's fds (visible host-wide via hostPID).
	tbl := make(map[string]Endpoint, len(inodeEP))
	for _, p := range procs {
		targets, err := p.FileDescriptorTargets()
		if err != nil {
			continue // process exited or no permission
		}
		var comm, cmd string
		for _, t := range targets {
			m := sockInodeRe.FindStringSubmatch(t)
			if m == nil {
				continue
			}
			inode, _ := strconv.ParseUint(m[1], 10, 64)
			keys, ok := inodeEP[inode]
			if !ok {
				continue
			}
			if comm == "" { // resolve process metadata lazily, only for procs that own a socket
				comm, _ = p.Comm()
				parts, _ := p.CmdLine()
				cmd = joinArgs(parts)
			}
			ep := Endpoint{PID: p.PID, Comm: comm, Cmd: cmd}
			for _, k := range keys {
				// concrete-IP keys win over bare wildcard keys on collision.
				if existing, exists := tbl[k]; exists && isWildcardKey(k) && !isWildcardKey(existing.boundKey) {
					continue
				}
				e := ep
				e.boundKey = k
				tbl[k] = e
			}
		}
	}
	r.mu.Lock()
	r.table = tbl
	r.mu.Unlock()
	return nil
}

func isWildcardKey(k string) bool {
	return strings.HasPrefix(k, "0.0.0.0:") || strings.HasPrefix(k, "[::]:") || strings.HasPrefix(k, ":::")
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
