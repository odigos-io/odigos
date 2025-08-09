package loaders

import (
	"context"
	"sync"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
	corev1 "k8s.io/api/core/v1"
)

type loadersKeyType string

const loadersKey loadersKeyType = "dataloaders"

type PodId struct {
	Namespace string
	PodName   string
}

type Loaders struct {
	mu sync.Mutex

	filter                  *model.WorkloadFilter
	isFilterSingleWorkload  bool
	isFilterSingleNamespace bool

	// the value we use for the namespace in the quires to api server.
	// for all namespace, this will be empty string.
	// for a namespace query, or a query for specific source, this will be the namespace name.
	queryNamespace string

	workloadIds []model.K8sWorkloadID

	instrumentationConfigMutex sync.Mutex
	instrumentationConfigs     map[model.K8sWorkloadID]*v1alpha1.InstrumentationConfig

	sourcesMutex    sync.Mutex
	workloadSources map[model.K8sWorkloadID]*v1alpha1.Source
	nsSources       map[string]*v1alpha1.Source

	workloadManifestsMutex sync.Mutex
	workloadManifests      map[model.K8sWorkloadID]*WorkloadManifest

	workloadPodsMutex sync.Mutex
	workloadPods      map[model.K8sWorkloadID][]*corev1.Pod

	instrumentationInstancesMutex sync.Mutex
	instrumentationInstances      map[PodId]*v1alpha1.InstrumentationInstance
}

func WithLoaders(ctx context.Context, loaders *Loaders) context.Context {
	return context.WithValue(ctx, loadersKey, loaders)
}

func For(ctx context.Context) *Loaders {
	return ctx.Value(loadersKey).(*Loaders)
}

func NewLoaders() *Loaders {
	return &Loaders{}
}

func (l *Loaders) GetWorkloadIds() []model.K8sWorkloadID {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.workloadIds
}

func (l *Loaders) GetInstrumentationConfig(ctx context.Context, workload model.K8sWorkloadID) (*v1alpha1.InstrumentationConfig, error) {
	l.instrumentationConfigMutex.Lock()
	defer l.instrumentationConfigMutex.Unlock()

	// if we did not fecth the instrumentation configs yet, do it now.
	if len(l.instrumentationConfigs) == 0 {
		instrumentationConfigs, err := l.fetchInstrumentationConfigs(ctx)
		if err != nil {
			return nil, err
		}
		l.instrumentationConfigs = instrumentationConfigs
	}
	return l.instrumentationConfigs[workload], nil
}

func (l *Loaders) GetSources(ctx context.Context, sourceId model.K8sWorkloadID) (*v1alpha1.WorkloadSources, error) {
	l.sourcesMutex.Lock()
	defer l.sourcesMutex.Unlock()

	// if we did not fetch the sources yet, do it now.
	if len(l.workloadSources) == 0 || len(l.nsSources) == 0 {
		workloadSources, namespaceSources, err := l.fetchSources(ctx)
		if err != nil {
			return nil, err
		}
		l.workloadSources = workloadSources
		l.nsSources = namespaceSources
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

	if len(l.workloadManifests) == 0 {
		workloadManifests, err := l.fetchWorkloadManifests(ctx)
		if err != nil {
			return nil, err
		}
		l.workloadManifests = workloadManifests
	}
	return l.workloadManifests[sourceId], nil
}

func (l *Loaders) GetWorkloadPods(ctx context.Context, sourceId model.K8sWorkloadID) ([]*corev1.Pod, error) {

	l.workloadPodsMutex.Lock()
	defer l.workloadPodsMutex.Unlock()

	if len(l.workloadPods) == 0 {
		workloadPods, err := l.fetchWorkloadPods(ctx)
		if err != nil {
			return nil, err
		}
		l.workloadPods = workloadPods
	}
	return l.workloadPods[sourceId], nil
}

func (l *Loaders) GetInstrumentationInstance(ctx context.Context, podId PodId) (*v1alpha1.InstrumentationInstance, error) {

	l.instrumentationInstancesMutex.Lock()
	defer l.instrumentationInstancesMutex.Unlock()

	if len(l.instrumentationInstances) == 0 {
		instrumentationInstances, err := l.fetchInstrumentationInstances(ctx)
		if err != nil {
			return nil, err
		}
		l.instrumentationInstances = instrumentationInstances
	}

	return l.instrumentationInstances[podId], nil
}

func (l *Loaders) SetFilters(ctx context.Context, filter *model.WorkloadFilter) error {

	l.filter = filter

	// it's a single workload filter if all the workload fields are set and not empty.
	l.isFilterSingleWorkload = filter != nil && filter.WorkloadKind != nil && filter.WorkloadName != nil && filter.Namespace != nil &&
		*filter.WorkloadKind != "" && *filter.WorkloadName != "" && *filter.Namespace != ""

	// it's a single namespace filter if the namespace field is set and not empty.
	l.isFilterSingleNamespace = filter != nil && filter.Namespace != nil && *filter.Namespace != ""

	if filter != nil {
		if filter.Namespace != nil {
			l.queryNamespace = *filter.Namespace
		}
	}

	filterMarkedForInstrumentation := filter != nil && filter.MarkedForInstrumentation != nil && *filter.MarkedForInstrumentation

	if filterMarkedForInstrumentation {
		l.instrumentationConfigMutex.Lock()
		defer l.instrumentationConfigMutex.Unlock()
		configById, err := l.fetchInstrumentationConfigs(ctx)
		if err != nil {
			return err
		}
		l.instrumentationConfigs = configById
		l.workloadIds = make([]model.K8sWorkloadID, 0, len(configById))
		for sourceId := range configById {
			l.workloadIds = append(l.workloadIds, sourceId)
		}
	} else {
		l.sourcesMutex.Lock()
		defer l.sourcesMutex.Unlock()
		l.workloadManifestsMutex.Lock()
		defer l.workloadManifestsMutex.Unlock()

		// fetch all sources (both those marked for instrumentation and those not)
		// this is to allow the user to review and instrument potential sources.
		workloadSources, namespaceSources, err := l.fetchSources(ctx)
		if err != nil {
			return err
		}
		l.workloadSources = workloadSources
		l.nsSources = namespaceSources

		workloadManifests, err := l.fetchWorkloadManifests(ctx)
		if err != nil {
			return err
		}
		l.workloadManifests = workloadManifests

		// calculate the source ids from the workload sources and manifests.
		// we can have workloads without sources, and sources without workloads.
		allWorkloads := make(map[model.K8sWorkloadID]struct{})
		for workloadId := range workloadSources {
			allWorkloads[workloadId] = struct{}{}
		}
		for workloadId := range workloadManifests {
			allWorkloads[workloadId] = struct{}{}
		}
		l.workloadIds = make([]model.K8sWorkloadID, 0, len(allWorkloads))
		for sourceId := range allWorkloads {
			l.workloadIds = append(l.workloadIds, sourceId)
		}
	}

	return nil
}
