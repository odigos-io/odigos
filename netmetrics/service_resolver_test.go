package netmetrics

import "testing"

// fakeEndpoints lets us test ServiceResolver join logic without /proc.
func newTestResolver(local map[string]Endpoint, pidSvc map[int]Service, peerSvc map[string]Service) *ServiceResolver {
	er := &EndpointResolver{table: map[string]Endpoint{}}
	for k, v := range local {
		er.table[k] = v
	}
	pidToSvc := func(pid int) (Service, bool) { s, ok := pidSvc[pid]; return s, ok }
	peerToSvc := func(ip string) (Service, bool) { s, ok := peerSvc[ip]; return s, ok }
	return NewServiceResolver(er, pidToSvc, peerToSvc)
}

func TestResolve_LocalIsDst_ServerPortStable(t *testing.T) {
	// inventory listens on 172.17.0.1:18080 (pid 100), peer frontend is 172.17.0.3 via registry.
	r := newTestResolver(
		map[string]Endpoint{"172.17.0.1:18080": {PID: 100, Comm: "python3"}},
		map[int]Service{100: {Name: "inventory"}},
		map[string]Service{"172.17.0.3": {Name: "frontend"}},
	)
	// request flow: frontend(src ephemeral) -> inventory(dst :18080)
	fi, ok := r.Resolve("172.17.0.3", 40000, "172.17.0.1", 18080)
	if !ok {
		t.Fatal("expected resolve ok")
	}
	if fi.Local.Name != "inventory" || fi.Peer.Name != "frontend" {
		t.Fatalf("got local=%q peer=%q", fi.Local.Name, fi.Peer.Name)
	}
	if fi.ServerPort != 18080 {
		t.Fatalf("server_port should be the local service port 18080, got %d", fi.ServerPort)
	}
	if fi.LocalIsSrc {
		t.Fatal("local should be dst")
	}
}

func TestResolve_ResponseDirection_SameServerPort(t *testing.T) {
	r := newTestResolver(
		map[string]Endpoint{"172.17.0.1:18080": {PID: 100, Comm: "python3"}},
		map[int]Service{100: {Name: "inventory"}},
		map[string]Service{"172.17.0.3": {Name: "frontend"}},
	)
	// response flow: inventory(src :18080) -> frontend(dst ephemeral)
	fi, ok := r.Resolve("172.17.0.1", 18080, "172.17.0.3", 40000)
	if !ok {
		t.Fatal("expected resolve ok")
	}
	if fi.Local.Name != "inventory" || fi.Peer.Name != "frontend" {
		t.Fatalf("got local=%q peer=%q", fi.Local.Name, fi.Peer.Name)
	}
	if fi.ServerPort != 18080 {
		t.Fatalf("server_port must stay 18080 across directions, got %d", fi.ServerPort)
	}
	if !fi.LocalIsSrc {
		t.Fatal("local should be src")
	}
}

func TestResolve_CommFallback(t *testing.T) {
	// local PID has no service mapping -> fall back to comm name.
	r := newTestResolver(
		map[string]Endpoint{"10.0.0.5:9000": {PID: 7, Comm: "redis-server"}},
		map[int]Service{}, // no mapping
		nil,
	)
	fi, ok := r.Resolve("10.0.0.5", 9000, "8.8.8.8", 53)
	if !ok || fi.Local.Name != "redis-server" {
		t.Fatalf("expected comm fallback redis-server, got ok=%v name=%q", ok, fi.Local.Name)
	}
	if fi.Peer.Name != "8.8.8.8" {
		t.Fatalf("unresolved peer should be raw IP, got %q", fi.Peer.Name)
	}
}

func TestResolve_NoLocalEndpoint(t *testing.T) {
	r := newTestResolver(map[string]Endpoint{}, map[int]Service{}, nil)
	if _, ok := r.Resolve("1.2.3.4", 1, "5.6.7.8", 2); ok {
		t.Fatal("expected not ok when neither endpoint is local")
	}
}
