package podsinjection

import (
	"sync"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/odigos-io/odigos/api/k8sconsts"
)

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

func (p *PodsTracker) SetPodWorkload(req ctrl.Request, pw k8sconsts.PodWorkload) {
	p.Mutex.Lock()
	defer p.Mutex.Unlock()
	p.podToWorkloadMap[req] = pw
}

func (p *PodsTracker) DeletePodWorkload(req ctrl.Request) {
	p.Mutex.Lock()
	defer p.Mutex.Unlock()
	delete(p.podToWorkloadMap, req)
}
