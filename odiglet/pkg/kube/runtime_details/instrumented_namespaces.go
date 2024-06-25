package runtime_details

import (
	"sync"
)

// instrumentedNamespaces contains the namespaces that have been labeled for instrumentation.
// This is used to skip the runtime details inspection for workloads which are not labeled for instrumentation.
// We need to make sure that their namespaces are not labeled for instrumentation as well before we can filter the reconcile request.
// It is updated by the NamespacesReconciler, and used by the WorkloadEnabledPredicate.
type instrumentedNamespaces struct {
	m map[string]struct{}
	mu sync.RWMutex
}

func newInstrumentedNamespaces() *instrumentedNamespaces {
	return &instrumentedNamespaces{
		m: make(map[string]struct{}),
	}
}

func (i *instrumentedNamespaces) add(namespace string) {
	i.mu.Lock()
	i.m[namespace] = struct{}{}
	i.mu.Unlock()
}

func (i *instrumentedNamespaces) remove(namespace string) {
	i.mu.Lock()
	delete(i.m, namespace)
	i.mu.Unlock()
}

func (i *instrumentedNamespaces) contains(namespace string) bool {
	i.mu.RLock()
	_, ok := i.m[namespace]
	i.mu.RUnlock()
	return ok
}