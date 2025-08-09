package loaders

import (
	"context"
	"fmt"
	"sync"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
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
	odigosConfiguration     *common.OdigosConfiguration
	ignoredNamespacesMap    map[string]struct{}

	// the value we use for the namespace in the quires to api server.
	// for all namespace, this will be empty string.
	// for a namespace query, or a query for specific source, this will be the namespace name.
	queryNamespace string

	workloadIds []model.K8sWorkloadID

	instrumentationConfigMutex    sync.Mutex
	instrumentationConfigsFetched bool
	instrumentationConfigs        map[model.K8sWorkloadID]*v1alpha1.InstrumentationConfig

	sourcesMutex    sync.Mutex
	sourcesFetched  bool
	workloadSources map[model.K8sWorkloadID]*v1alpha1.Source
	nsSources       map[string]*v1alpha1.Source

	workloadManifestsMutex   sync.Mutex
	workloadManifestsFetched bool
	workloadManifests        map[model.K8sWorkloadID]*WorkloadManifest

	workloadPodsMutex   sync.Mutex
	workloadPodsFetched bool
	workloadPods        map[model.K8sWorkloadID][]CachedPod

	instrumentationInstancesMutex   sync.Mutex
	instrumentationInstancesFetched bool
	instrumentationInstances        map[PodId]*v1alpha1.InstrumentationInstance
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

// if the instrumentation configs are not fetched yet, fetch them and cache them.
// this function assumes that the instrumentation config mutex is already locked.
func (l *Loaders) loadInstrumentationConfigs(ctx context.Context) error {
	if l.instrumentationConfigsFetched {
		return nil
	}
	instrumentationConfigs, err := l.fetchInstrumentationConfigs(ctx)
	if err != nil {
		return err
	}
	l.instrumentationConfigs = instrumentationConfigs
	l.instrumentationConfigsFetched = true
	return nil
}

func (l *Loaders) loadSources(ctx context.Context) error {
	if l.sourcesFetched {
		return nil
	}
	workloadSources, namespaceSources, err := l.fetchSources(ctx)
	if err != nil {
		return err
	}
	l.workloadSources = workloadSources
	l.nsSources = namespaceSources
	l.sourcesFetched = true
	return nil
}

func (l *Loaders) loadWorkloadManifests(ctx context.Context) error {
	if l.workloadManifestsFetched {
		return nil
	}
	workloadManifests, err := l.fetchWorkloadManifests(ctx)
	if err != nil {
		return err
	}
	l.workloadManifests = workloadManifests
	l.workloadManifestsFetched = true
	return nil
}

// if the workload pods are not fetched yet, fetch them and cache them.
// also compute additional values for each pod as needed.
func (l *Loaders) loadWorkloadPods(ctx context.Context) error {
	if l.workloadPodsFetched {
		return nil
	}

	workloadPods, err := l.fetchWorkloadPods(ctx)
	if err != nil {
		return err
	}

	cachePods := make(map[model.K8sWorkloadID][]CachedPod)
	l.instrumentationConfigMutex.Lock()
	defer l.instrumentationConfigMutex.Unlock()
	if err := l.loadInstrumentationConfigs(ctx); err != nil {
		return err
	}
	for sourceId, pods := range workloadPods {
		cachePods[sourceId] = make([]CachedPod, 0, len(pods))
		for _, pod := range pods {
			cachePods[sourceId] = append(cachePods[sourceId], CachedPod{
				Pod:               pod,
				ComputedPodValues: NewComputedPodValues(pod, l.instrumentationConfigs[sourceId]),
			})
		}
	}

	l.workloadPods = cachePods
	l.workloadPodsFetched = true

	return nil
}

func (l *Loaders) loadInstrumentationInstances(ctx context.Context) error {

	if l.instrumentationInstancesFetched {
		return nil
	}
	instrumentationInstances, err := l.fetchInstrumentationInstances(ctx)
	if err != nil {
		return err
	}
	l.instrumentationInstances = instrumentationInstances
	l.instrumentationInstancesFetched = true
	return nil
}

func (l *Loaders) SetFilters(ctx context.Context, filter *model.WorkloadFilter) error {

	// fetch odigos configuration for each request.
	odigosns := env.GetCurrentNamespace()
	configMap, err := kube.DefaultClient.CoreV1().ConfigMaps(odigosns).Get(ctx, consts.OdigosEffectiveConfigName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal([]byte(configMap.Data[consts.OdigosConfigurationFileName]), &l.odigosConfiguration); err != nil {
		return err
	}
	ignoredNamespacesMap := make(map[string]struct{})
	for _, namespace := range l.odigosConfiguration.IgnoredNamespaces {
		ignoredNamespacesMap[namespace] = struct{}{}
	}
	l.ignoredNamespacesMap = ignoredNamespacesMap

	// check if it's a namespace query for ignored namespaces.
	if filter != nil && filter.Namespace != nil {
		if _, ok := l.ignoredNamespacesMap[*filter.Namespace]; ok {
			return fmt.Errorf("namespace %s is configured to be ignored by odigos", *filter.Namespace)
		}
	}

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
		if err := l.loadInstrumentationConfigs(ctx); err != nil {
			return err
		}
		l.workloadIds = make([]model.K8sWorkloadID, 0, len(l.instrumentationConfigs))
		for sourceId := range l.instrumentationConfigs {
			l.workloadIds = append(l.workloadIds, sourceId)
		}
	} else {
		l.sourcesMutex.Lock()
		defer l.sourcesMutex.Unlock()
		l.workloadManifestsMutex.Lock()
		defer l.workloadManifestsMutex.Unlock()

		if err := l.loadSources(ctx); err != nil {
			return err
		}
		if err := l.loadWorkloadManifests(ctx); err != nil {
			return err
		}

		// calculate the source ids from the workload sources and manifests.
		// we can have workloads without sources, and sources without workloads.
		allWorkloads := make(map[model.K8sWorkloadID]struct{})
		for workloadId := range l.workloadSources {
			allWorkloads[workloadId] = struct{}{}
		}
		for workloadId := range l.workloadManifests {
			allWorkloads[workloadId] = struct{}{}
		}
		l.workloadIds = make([]model.K8sWorkloadID, 0, len(allWorkloads))
		for sourceId := range allWorkloads {
			l.workloadIds = append(l.workloadIds, sourceId)
		}
	}

	return nil
}
