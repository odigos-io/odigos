// Package netmetrics wires the shared github.com/odigos-io/odigos/netmetrics (network map)
// and securitymetrics (security) engines into odiglet for the k8s case. The engines are
// env-agnostic; the only k8s-specific part is identity resolution — turning a flow's local
// PID and its peer IP into the SAME service.name that traces use. That mapping is sourced
// from caches odiglet already watches (the node-scoped Pod cache + InstrumentationConfig),
// so no new informers, no new container, no new binary are introduced.
package netmetrics

import (
	"bufio"
	"context"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/odigos-io/odigos/netmetrics"
)

// podIndexTTL bounds how often the pod IP/container-id index is rebuilt from the cache. Flow
// resolution calls PIDToService/PeerToService once per flow (hundreds per snapshot, every couple
// of seconds); rebuilding the index at most once per TTL turns that into a single cache List per
// TTL instead of one List+deep-copy per flow. Pods on a node change on the order of seconds, so a
// 1s index is effectively current.
const podIndexTTL = 1 * time.Second

// K8sIdentity resolves a flow endpoint to a k8s service identity. It backs the two closures
// injected into the shared netmetrics ServiceResolver:
//   - PIDToService: the flow's LOCAL endpoint (a pod process on this node, found via /proc by
//     the EndpointResolver) -> pod -> workload -> InstrumentationConfig.Spec.ServiceName.
//   - PeerToService: the flow's PEER IP -> pod (by pod IP, node-scoped cache) -> same path.
//
// Both resolve to the authoritative InstrumentationConfig.Spec.ServiceName when the workload
// is sourced (so flows and traces share one service.name); otherwise the workload/pod name.
type K8sIdentity struct {
	ctx    context.Context
	client client.Client // the manager's cached client (node-scoped Pods + InstrumentationConfig)

	// pod index, rebuilt at most once per podIndexTTL (see refreshLocked). Guards the maps and
	// the build timestamp so concurrent resolver goroutines (the /api/network handler and the
	// security source poll) share one index without racing.
	mu       sync.Mutex
	builtAt  time.Time
	ipToPod  map[string]*corev1.Pod // pod IP -> pod (host-network pods excluded; their IP is the node IP)
	cidToPod map[string]*corev1.Pod // runtime container id -> pod
}

// NewK8sIdentity builds the resolver over the controller-runtime cached client.
func NewK8sIdentity(ctx context.Context, c client.Client) *K8sIdentity {
	return &K8sIdentity{ctx: ctx, client: c}
}

// PIDToService implements netmetrics.PIDToService for k8s: PID -> container id (from
// /proc/<pid>/cgroup) -> pod -> service identity.
func (k *K8sIdentity) PIDToService(pid int) (netmetrics.Service, bool) {
	cid := containerIDForPID(pid)
	if cid == "" {
		return netmetrics.Service{}, false
	}
	pod := k.podByContainerID(cid)
	if pod == nil {
		return netmetrics.Service{}, false
	}
	return k.serviceForPod(pod)
}

// PeerToService implements netmetrics.PeerToService for k8s: peer IP -> pod (by pod IP) ->
// service identity. Node-scoped: a peer pod on THIS node resolves; cross-node peers fall back
// to the raw IP (a cluster-scoped EndpointSlice index is the multi-node fast-follow).
func (k *K8sIdentity) PeerToService(ip string) (netmetrics.Service, bool) {
	pod := k.podByIP(ip)
	if pod == nil {
		return netmetrics.Service{}, false
	}
	return k.serviceForPod(pod)
}

// serviceForPod maps a Pod to its service identity. Sourced (instrumented) workloads resolve
// to InstrumentationConfig.Spec.ServiceName — the IDENTICAL name their traces carry; otherwise
// the workload name (or the bare pod name for unmanaged pods).
func (k *K8sIdentity) serviceForPod(pod *corev1.Pod) (netmetrics.Service, bool) {
	pw, err := workload.PodWorkloadObject(pod)
	if err != nil || pw == nil {
		// Unmanaged pod (no workload owner): name it by the pod, not instrumented.
		return netmetrics.Service{Name: pod.Name, Namespace: pod.Namespace}, true
	}

	name := pw.Name
	instrumented := false
	var ic odigosv1.InstrumentationConfig
	key := client.ObjectKey{
		Namespace: pw.Namespace,
		Name:      workload.CalculateWorkloadRuntimeObjectName(pw.Name, pw.Kind),
	}
	if err := k.client.Get(k.ctx, key, &ic); err == nil {
		// The workload is sourced/instrumented; use the authoritative service.name traces use.
		instrumented = true
		if ic.Spec.ServiceName != "" {
			name = ic.Spec.ServiceName
		}
	}
	return netmetrics.Service{
		Name:         name,
		Namespace:    pw.Namespace,
		Instrumented: instrumented,
		Kind:         string(pw.Kind),
		Eligible:     instrumented, // sourced ⇒ a trace-able workload (refine with runtime detection later)
	}, true
}

// podByIP finds the pod owning an IP via the TTL-cached index.
func (k *K8sIdentity) podByIP(ip string) *corev1.Pod {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.refreshLocked()
	return k.ipToPod[ip]
}

// podByContainerID finds the pod whose container has the given runtime container id, via the
// TTL-cached index.
func (k *K8sIdentity) podByContainerID(cid string) *corev1.Pod {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.refreshLocked()
	return k.cidToPod[cid]
}

// refreshLocked rebuilds the pod IP/container-id index from the node-scoped cache when it is
// older than podIndexTTL. Caller must hold k.mu. On a List error the previous index is kept
// (better a slightly stale map than no resolution). The cache List is in-memory (no API call),
// so holding the lock across it is cheap at node scale.
func (k *K8sIdentity) refreshLocked() {
	if k.ipToPod != nil && time.Since(k.builtAt) < podIndexTTL {
		return
	}
	var pods corev1.PodList
	if err := k.client.List(k.ctx, &pods); err != nil {
		return
	}
	ipIdx := make(map[string]*corev1.Pod, len(pods.Items))
	cidIdx := make(map[string]*corev1.Pod, len(pods.Items))
	for i := range pods.Items {
		p := &pods.Items[i]
		// Host-network pods report the node IP as their pod IP; indexing them would alias every
		// host-network pod (and the node itself) to one arbitrary pod. Skip them in the IP index —
		// such peers fall back to the raw IP / host classification. Their containers are still
		// indexed by container id for local PID resolution.
		if !p.Spec.HostNetwork {
			if p.Status.PodIP != "" {
				ipIdx[p.Status.PodIP] = p
			}
			for _, pip := range p.Status.PodIPs {
				if pip.IP != "" {
					ipIdx[pip.IP] = p
				}
			}
		}
		for _, cs := range p.Status.ContainerStatuses {
			if id := stripContainerScheme(cs.ContainerID); id != "" {
				cidIdx[id] = p
			}
		}
	}
	k.ipToPod = ipIdx
	k.cidToPod = cidIdx
	k.builtAt = time.Now()
}

// containerIDRe matches a 64-hex container id anywhere in a cgroup path (handles both the
// cgroupfs layout `/kubepods/.../pod<uid>/<id>` and the systemd layout
// `...cri-containerd-<id>.scope`).
var containerIDRe = regexp.MustCompile(`[0-9a-f]{64}`)

// containerIDForPID reads /proc/<pid>/cgroup and extracts the container id.
func containerIDForPID(pid int) string {
	f, err := os.Open("/proc/" + strconv.Itoa(pid) + "/cgroup")
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if m := containerIDRe.FindString(sc.Text()); m != "" {
			return m
		}
	}
	return ""
}

// stripContainerScheme turns "containerd://<id>" / "docker://<id>" into the bare id.
func stripContainerScheme(s string) string {
	if i := strings.Index(s, "://"); i >= 0 {
		return s[i+3:]
	}
	return s
}
