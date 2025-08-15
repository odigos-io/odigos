package loaders

import (
	"context"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/computed"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

func (l *Loaders) GetInstrumentationConfig(ctx context.Context, workload model.K8sWorkloadID) (*v1alpha1.InstrumentationConfig, error) {
	l.instrumentationConfigMutex.Lock()
	defer l.instrumentationConfigMutex.Unlock()

	if err := l.loadInstrumentationConfigs(ctx); err != nil {
		return nil, err
	}
	return l.instrumentationConfigs[workload], nil
}

func (l *Loaders) GetSources(ctx context.Context, sourceId model.K8sWorkloadID) (*v1alpha1.WorkloadSources, error) {
	l.sourcesMutex.Lock()
	defer l.sourcesMutex.Unlock()

	if err := l.loadSources(ctx); err != nil {
		return nil, err
	}

	// return both the workload and namespace sources for this one.
	return &v1alpha1.WorkloadSources{
		Workload:  l.workloadSources[sourceId],
		Namespace: l.nsSources[sourceId.Namespace],
	}, nil
}

func (l *Loaders) GetWorkloadManifest(ctx context.Context, sourceId model.K8sWorkloadID) (*WorkloadManifest, error) {
	l.workloadManifestsMutex.Lock()
	defer l.workloadManifestsMutex.Unlock()

	if err := l.loadWorkloadManifests(ctx); err != nil {
		return nil, err
	}
	return l.workloadManifests[sourceId], nil
}

func (l *Loaders) GetWorkloadPods(ctx context.Context, sourceId model.K8sWorkloadID) ([]computed.CachedPod, error) {

	l.workloadPodsMutex.Lock()
	defer l.workloadPodsMutex.Unlock()

	if err := l.loadWorkloadPods(ctx); err != nil {
		return nil, err
	}

	return l.workloadPods[sourceId], nil
}

func (l *Loaders) GetInstrumentationInstancesForContainer(ctx context.Context, containerId ContainerId) ([]*v1alpha1.InstrumentationInstance, error) {

	l.instrumentationInstancesMutex.Lock()
	defer l.instrumentationInstancesMutex.Unlock()

	if err := l.loadInstrumentationInstances(ctx); err != nil {
		return nil, err
	}
	return l.instrumentationInstancesByContainer[containerId], nil
}
