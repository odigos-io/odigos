package podsinjectionstatus

import (
	"fmt"
	"sync"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/odigos-io/odigos/api/k8sconsts"
)

const (
	// protection from possible bugs where the memory is not being reclaimed when pods goes down.
	// there isn't a known scenario where it's expected to happen.
	maxPodsTrackerSize = 50000
)

// track each pod name+namespace to the workload it belongs to.
// when controller reports pod deleted, we need to update the pods injection status in ic.
// however, at this point we only have the pod name, and don't know which ic to fetch and sync from the event.
// this tracker is to discover this mapping at this point.
//
// It would be best to populate a finalaizer and prevent the pod from being deleted until the ic status is updated,
// but finalizers can also bring in stability concerns, since odigos has some issue and fails to remove it,
// this can cause pods to be stuck in terminating state.
//
// all pods should be tracked since at the beginning of instrumentor, it pulls all pods into the cache and uses
// them to build the initial pods injection status.
type PodsTracker struct {
	sync.Mutex
	podToWorkloadMap map[ctrl.Request]k8sconsts.PodWorkload
}

func NewPodsTracker() *PodsTracker {
	return &PodsTracker{
		podToWorkloadMap: make(map[ctrl.Request]k8sconsts.PodWorkload),
	}
}

func (p *PodsTracker) GetPodWorkload(req ctrl.Request) (k8sconsts.PodWorkload, bool) {
	p.Mutex.Lock()
	defer p.Mutex.Unlock()
	pw, ok := p.podToWorkloadMap[req]
	return pw, ok
}

func (p *PodsTracker) SetPodWorkload(req ctrl.Request, pw k8sconsts.PodWorkload) error {
	p.Mutex.Lock()
	defer p.Mutex.Unlock()
	if len(p.podToWorkloadMap) >= maxPodsTrackerSize {
		return fmt.Errorf("pods tracker is at max size: %d, skipping pod workload set", maxPodsTrackerSize)
	}
	p.podToWorkloadMap[req] = pw
	return nil
}

func (p *PodsTracker) DeletePodWorkload(req ctrl.Request) {
	p.Mutex.Lock()
	defer p.Mutex.Unlock()
	delete(p.podToWorkloadMap, req)
}
