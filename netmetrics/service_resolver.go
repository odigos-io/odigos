package netmetrics

// Service is the identity stamped onto a flow endpoint. Field names follow OTel
// semconv: Name -> service.name, Namespace -> service.namespace.
type Service struct {
	Name      string
	Namespace string
	// SourceKind/SourceName are Odigos-specific provenance (odigos.vm.source.*),
	// optional and only populated by producers that have them.
	SourceKind string
	SourceName string
	// Instrumented is true when this identity came from an enabled Odigos Source
	// (so it also has traces/profiles under the same service.name), false when it
	// is a bare process resolved only by comm/cmdline (an instrumentation candidate).
	// Set by the injected PIDToService; the comm fallback leaves it false.
	Instrumented bool

	// Workload provenance for the instrumentation surface: what it would take to
	// turn this discovered process into an Odigos Source. Populated by the injected
	// PIDToService from the workload report; empty for comm-only / external peers.
	Kind     string // "docker" | "systemd" | "process" — the Source Kind to create
	Eligible bool   // language/runtime matches an instrumentation target
	Runtime  string // detected language/runtime (e.g. "java", "python")
}

func (s Service) empty() bool { return s.Name == "" }

// PIDToService resolves a PID to its service identity. INJECTED by the host agent:
//   - VM agent: backed by the profileattrs PID->Source table.
//   - odiglet : backed by the k8s instrumentation manager / informer.
//
// This is the only per-environment dependency; the join logic is shared.
type PIDToService func(pid int) (Service, bool)

// PeerToService resolves a remote IP to a service identity (CIDR registry, k8s
// informer, DNS, etc.). INJECTED by the host agent. May return false.
type PeerToService func(ip string) (Service, bool)

// FlowIdentity is the resolved identity for one OBI flow.
type FlowIdentity struct {
	Local       Service // the endpoint owned by a process on this node
	Peer        Service // the other endpoint (may be empty / raw IP)
	ServerPort  int     // the local service's own port (stable; not the peer ephemeral)
	LocalIsSrc  bool    // whether the local endpoint was the flow's source
	PeerIsLocal bool    // whether the peer is itself a process on this node (intra-node)
}

// ServiceResolver joins an OBI flow (5-tuple) to service identity. It is constructed
// once and shared; the host agent injects its PID->service and peer->service lookups.
type ServiceResolver struct {
	endpoints *EndpointResolver
	pidToSvc  PIDToService
	peerToSvc PeerToService
}

// NewServiceResolver composes the shared EndpointResolver with host-injected lookups.
// peerToSvc may be nil (peer left unresolved).
func NewServiceResolver(endpoints *EndpointResolver, pidToSvc PIDToService, peerToSvc PeerToService) *ServiceResolver {
	return &ServiceResolver{endpoints: endpoints, pidToSvc: pidToSvc, peerToSvc: peerToSvc}
}

// localService resolves a local endpoint to a Service: endpoint -> PID -> service.
// Falls back to a comm-named Service when the PID is local but the producer has no
// mapping (so the flow is still attributable, just coarsely).
func (s *ServiceResolver) localService(ip string, port int) (Service, bool) {
	ep, ok := s.endpoints.Lookup(ip, port)
	if !ok {
		return Service{}, false
	}
	if s.pidToSvc != nil {
		if svc, ok := s.pidToSvc(ep.PID); ok && !svc.empty() {
			return svc, true
		}
	}
	if ep.Comm != "" {
		return Service{Name: ep.Comm}, true // local but unmapped: name by process
	}
	return Service{}, false
}

// Resolve attributes a flow. It decides which endpoint is local (owned by a process
// here), resolves it to a Service, and resolves the peer via the injected lookup.
// Returns ok=false only if neither endpoint is local.
func (s *ServiceResolver) Resolve(srcIP string, srcPort int, dstIP string, dstPort int) (FlowIdentity, bool) {
	var fi FlowIdentity

	if svc, ok := s.localService(srcIP, srcPort); ok {
		fi.Local, fi.LocalIsSrc = svc, true
	} else if svc, ok := s.localService(dstIP, dstPort); ok {
		fi.Local, fi.LocalIsSrc = svc, false
	} else {
		return FlowIdentity{}, false
	}

	// peer = the other side; server_port = the local service's own port (stable,
	// not the peer's ephemeral port — which would explode cardinality).
	peerIP, peerPort := dstIP, dstPort
	fi.ServerPort = srcPort
	if !fi.LocalIsSrc {
		peerIP, peerPort = srcIP, srcPort
		fi.ServerPort = dstPort
	}

	// peer may itself be a local service (intra-node), else the injected registry.
	if svc, ok := s.localService(peerIP, peerPort); ok {
		fi.Peer = svc
		fi.PeerIsLocal = true
	} else if s.peerToSvc != nil {
		if svc, ok := s.peerToSvc(peerIP); ok {
			fi.Peer = svc
		}
	}
	if fi.Peer.empty() {
		fi.Peer = Service{Name: peerIP} // last resort: raw IP
	}
	return fi, true
}

// Endpoints exposes the underlying resolver (for Refresh scheduling / Size metrics).
func (s *ServiceResolver) Endpoints() *EndpointResolver { return s.endpoints }
