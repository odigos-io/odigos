package loaders

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/computed"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/graph/status"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type loadersKeyType string

const loadersKey loadersKeyType = "dataloaders"

type PodContainerId struct {
	Namespace     string
	PodName       string
	ContainerName string
}

type WorkloadContainerId struct {
	Namespace     string
	Kind          k8sconsts.WorkloadKind
	Name          string
	ContainerName string
}

// workloadIdSnapshot is an immutable snapshot of workload identity data.
// Published atomically so readers never need a lock.
type workloadIdSnapshot struct {
	workloadIds     []model.K8sWorkloadID
	workloadIdsMap  map[k8sconsts.PodWorkload]struct{}
	nsToWorkloadIds map[string][]model.K8sWorkloadID
}

type Loaders struct {
	logger logr.Logger

	k8sCacheClient client.Client

	configOnce sync.Once
	configErr  error

	// defaultFilterOnce guards the SetFilters(nil) path which is the only
	// concurrent pattern (namespace field fan-out). Non-nil filter paths are
	// already externally serialized by heavyWorkloadQueryMu.
	defaultFilterOnce sync.Once
	defaultFilterErr  error

	workloadFilter      *WorkloadFilter
	odigosConfiguration *common.OdigosConfiguration

	// list of all the (non-ignored) namespaces in the cluster.
	namespacesMutex   sync.Mutex
	namespacesFetched bool
	namespaces        []string

	// Atomically published snapshot; readers are lock-free.
	workloadSnap atomic.Pointer[workloadIdSnapshot]

	instrumentationConfigMutex    sync.Mutex
	instrumentationConfigsFetched bool
	instrumentationConfigs        map[model.K8sWorkloadID]*v1alpha1.InstrumentationConfig

	sourcesMutex    sync.Mutex
	sourcesFetched  bool
	workloadSources map[model.K8sWorkloadID]*v1alpha1.Source
	nsSources       map[string]*v1alpha1.Source

	workloadManifestsMutex   sync.Mutex
	workloadManifestsFetched bool
	workloadManifests        map[model.K8sWorkloadID]*computed.CachedWorkloadManifest

	workloadPodsMutex   sync.Mutex
	workloadPodsFetched bool
	workloadPods        map[model.K8sWorkloadID][]computed.CachedPod

	instrumentationInstancesMutex               sync.Mutex
	instrumentationInstancesFetched             bool
	instrumentationInstancesByPodContainer      map[PodContainerId][]*v1alpha1.InstrumentationInstance
	instrumentationInstancesByWorkloadContainer map[WorkloadContainerId][]*v1alpha1.InstrumentationInstance
}

func WithLoaders(ctx context.Context, loaders *Loaders) context.Context {
	return context.WithValue(ctx, loadersKey, loaders)
}

func For(ctx context.Context) *Loaders {
	return ctx.Value(loadersKey).(*Loaders)
}

func NewLoaders(logger logr.Logger, k8sCacheClient client.Client) *Loaders {
	return &Loaders{
		logger:         logger,
		k8sCacheClient: k8sCacheClient,
	}
}

func (l *Loaders) GetWorkloadIds() []model.K8sWorkloadID {
	if snap := l.workloadSnap.Load(); snap != nil {
		return snap.workloadIds
	}
	return nil
}

func (l *Loaders) GetWorkloadIdsInNamespace(ns string) []model.K8sWorkloadID {
	if snap := l.workloadSnap.Load(); snap != nil {
		return snap.nsToWorkloadIds[ns]
	}
	return nil
}

func (l *Loaders) loadNamespaces(ctx context.Context) error {
	if l.namespacesFetched {
		return nil
	}
	namespaces, err := fetchNamespaces(ctx, l.k8sCacheClient)
	if err != nil {
		return err
	}

	filteredNamespaces := make([]string, 0, len(namespaces.Items))
	for _, namespace := range namespaces.Items {
		if _, ok := l.workloadFilter.IgnoredNamespaces[namespace.Name]; !ok {
			filteredNamespaces = append(filteredNamespaces, namespace.Name)
		}
	}
	l.namespaces = filteredNamespaces
	l.namespacesFetched = true
	return nil
}

// if the instrumentation configs are not fetched yet, fetch them and cache them.
// this function assumes that the instrumentation config mutex is already locked.
func (l *Loaders) loadInstrumentationConfigs(ctx context.Context) error {
	if l.instrumentationConfigsFetched {
		return nil
	}
	instrumentationConfigs, err := fetchInstrumentationConfigs(ctx, l.workloadFilter, l.k8sCacheClient)
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
	workloadSources, namespaceSources, err := fetchSources(ctx, l.workloadFilter, l.k8sCacheClient)
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

	var workloadManifests map[model.K8sWorkloadID]*computed.CachedWorkloadManifest
	var err error

	// When workload IDs are already known (e.g. markedForInstrumentation path
	// or workloadsByIds), use targeted per-workload Get operations instead of
	// 6+ cluster-wide List operations. This dramatically reduces memory from
	// deep-copying thousands of unrelated K8s objects from the informer cache.
	if snap := l.workloadSnap.Load(); snap != nil {
		workloadManifests, err = fetchWorkloadManifestsByIds(ctx, l.logger, snap.workloadIds, l.k8sCacheClient)
	} else {
		workloadManifests, err = fetchWorkloadManifests(ctx, l.logger, l.workloadFilter, l.k8sCacheClient)
	}
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

	// if this is a single workload query, we need to fetch the workload manifest
	// to get the selector to fetch just this one pod.
	var singleWorkloadManifest *computed.CachedWorkloadManifest
	if l.workloadFilter.SingleWorkload != nil {
		var err error
		singleWorkloadManifest, err = l.GetWorkloadManifest(ctx, model.K8sWorkloadID{
			Namespace: l.workloadFilter.SingleWorkload.Namespace,
			Kind:      model.K8sResourceKind(l.workloadFilter.SingleWorkload.WorkloadKind),
			Name:      l.workloadFilter.SingleWorkload.WorkloadName,
		})
		if err != nil {
			return err
		}
	}

	var workloadIdsMap map[k8sconsts.PodWorkload]struct{}
	if snap := l.workloadSnap.Load(); snap != nil {
		workloadIdsMap = snap.workloadIdsMap
	}
	workloadPods, err := fetchWorkloadPods(ctx, l.logger, l.workloadFilter, singleWorkloadManifest, workloadIdsMap, l.k8sCacheClient)
	if err != nil {
		return err
	}

	automaticRolloutEnabled := true // default
	if l.odigosConfiguration.Rollout != nil && l.odigosConfiguration.Rollout.AutomaticRolloutDisabled != nil && *l.odigosConfiguration.Rollout.AutomaticRolloutDisabled {
		automaticRolloutEnabled = false
	}

	cachePods := make(map[model.K8sWorkloadID][]computed.CachedPod)
	l.instrumentationConfigMutex.Lock()
	defer l.instrumentationConfigMutex.Unlock()
	if err := l.loadInstrumentationConfigs(ctx); err != nil {
		return err
	}
	for sourceId, pods := range workloadPods {
		cachePods[sourceId] = make([]computed.CachedPod, 0, len(pods))
		for _, pod := range pods {
			ic := l.instrumentationConfigs[sourceId]
			agentInjected, agentInjectedStatus := status.CalculatePodAgentInjectedStatus(pod, ic, automaticRolloutEnabled)
			var startedPostAgentMetaHashChange *bool
			if ic != nil && ic.Spec.AgentsMetaHashChangedTime != nil {
				posStartTimeAfterAgentMetaHashChange := pod.CreationTimestamp.After(ic.Spec.AgentsMetaHashChangedTime.Time)
				startedPostAgentMetaHashChange = &posStartTimeAfterAgentMetaHashChange
			}
			containers := make([]computed.ComputedPodContainer, 0, len(pod.Spec.Containers))
			for _, container := range pod.Spec.Containers {
				otelDistroName := getEnvValueFromManifest(container.Env, k8sconsts.OdigosEnvVarDistroName)
				isExpectingInstrumentationInstances := isDistroExpectingInstrumentationInstances(otelDistroName)
				odigosInstrumentationDeviceName := getOdigosInstrumentationDeviceName(container.Resources.Requests)

				containerStatus := getContainerStatus(pod, container.Name)
				var ready, started *bool
				var isCrashLoop bool
				var restartCount *int
				var runningStartedTime, waitingReasonEnum, waitingMessage *string
				if containerStatus != nil {
					restartCountInt := int(containerStatus.RestartCount)
					restartCount = &restartCountInt
					ready = &containerStatus.Ready
					started = containerStatus.Started
					if containerStatus.State.Waiting != nil {
						isCrashLoop = containerStatus.State.Waiting.Reason == "CrashLoopBackOff"
						waitingReasonEnum = &containerStatus.State.Waiting.Reason
						waitingMessage = &containerStatus.State.Waiting.Message
					}
					if containerStatus.State.Running != nil {
						runningStartedTimeStr := containerStatus.State.Running.StartedAt.Format(time.RFC3339)
						runningStartedTime = &runningStartedTimeStr
					}
				}

				containers = append(containers, computed.ComputedPodContainer{
					ContainerName:                     container.Name,
					OtelDistroName:                    otelDistroName,
					ExpectingInstrumentationInstances: isExpectingInstrumentationInstances,
					OdigosInstrumentationDeviceName:   odigosInstrumentationDeviceName,
					Ready:                             ready,
					IsReady:                           ready != nil && *ready,
					Started:                           started,
					IsCrashLoop:                       isCrashLoop,
					RestartCount:                      restartCount,
					RunningStartedTime:                runningStartedTime,
					WaitingReasonEnum:                 waitingReasonEnum,
					WaitingMessage:                    waitingMessage,
				})
			}
			cachedPod := computed.CachedPod{
				PodNamespace:                   pod.Namespace,
				PodName:                        pod.Name,
				PodNodeName:                    pod.Spec.NodeName,
				PodStartTime:                   pod.CreationTimestamp.Format(time.RFC3339),
				StartedPostAgentMetaHashChange: startedPostAgentMetaHashChange,
				AgentInjected:                  agentInjected,
				AgentInjectedStatus:            agentInjectedStatus,
				Containers:                     containers,
			}

			cachePods[sourceId] = append(cachePods[sourceId], cachedPod)
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
	byPodContainer, byWorkloadContainer, err := fetchInstrumentationInstances(ctx, l.logger, l.workloadFilter, l.k8sCacheClient)
	if err != nil {
		return err
	}
	l.instrumentationInstancesByPodContainer = byPodContainer
	l.instrumentationInstancesByWorkloadContainer = byWorkloadContainer
	l.instrumentationInstancesFetched = true
	return nil
}

// LoadConfig loads the odigos configuration and sets up ignored namespaces.
// Cached after first call via sync.Once — safe to call concurrently.
func (l *Loaders) LoadConfig(ctx context.Context) error {
	l.configOnce.Do(func() {
		odigosns := env.GetCurrentNamespace()
		var odigosConfigurationConfigMap corev1.ConfigMap
		err := l.k8sCacheClient.Get(ctx, client.ObjectKey{
			Namespace: odigosns,
			Name:      consts.OdigosEffectiveConfigName,
		}, &odigosConfigurationConfigMap)
		if err != nil {
			l.configErr = err
			return
		}

		if err := yaml.Unmarshal([]byte(odigosConfigurationConfigMap.Data[consts.OdigosConfigurationFileName]), &l.odigosConfiguration); err != nil {
			l.configErr = err
			return
		}
		ignoredNamespacesMap := make(map[string]struct{})
		for _, namespace := range l.odigosConfiguration.IgnoredNamespaces {
			ignoredNamespacesMap[namespace] = struct{}{}
		}

		l.workloadFilter = &WorkloadFilter{
			ClusterWide:       &WorkloadFilterClusterWide{},
			NamespaceString:   "",
			IgnoredNamespaces: ignoredNamespacesMap,
		}
	})
	return l.configErr
}

// publishSnapshot builds an immutable workloadIdSnapshot from ids and
// atomically stores it so readers never need a lock.
func (l *Loaders) publishSnapshot(ids []model.K8sWorkloadID) {
	idsMap := make(map[k8sconsts.PodWorkload]struct{}, len(ids))
	nsMap := make(map[string][]model.K8sWorkloadID)
	for _, id := range ids {
		nsMap[id.Namespace] = append(nsMap[id.Namespace], id)
		idsMap[k8sconsts.PodWorkload{
			Namespace: id.Namespace,
			Kind:      k8sconsts.WorkloadKind(id.Kind),
			Name:      id.Name,
		}] = struct{}{}
	}
	l.workloadSnap.Store(&workloadIdSnapshot{
		workloadIds:     ids,
		workloadIdsMap:  idsMap,
		nsToWorkloadIds: nsMap,
	})
}

func (l *Loaders) SetFilters(ctx context.Context, filter *model.WorkloadFilter) error {
	if err := l.LoadConfig(ctx); err != nil {
		return err
	}

	isDefaultFilter := filter == nil || (filter.Namespace == nil && filter.Kind == nil &&
		filter.Name == nil && filter.MarkedForInstrumentation == nil)

	if isDefaultFilter {
		l.defaultFilterOnce.Do(func() {
			l.defaultFilterErr = l.doSetFilters(ctx, filter)
		})
		return l.defaultFilterErr
	}

	return l.doSetFilters(ctx, filter)
}

// doSetFilters performs the actual filter setup and workload ID loading.
// For nil/default filters this is called via sync.Once (concurrent safe).
// For non-nil filters this is called directly (externally serialized by
// heavyWorkloadQueryMu in the resolver layer).
func (l *Loaders) doSetFilters(ctx context.Context, filter *model.WorkloadFilter) error {
	ignoredNamespacesMap := l.workloadFilter.IgnoredNamespaces

	if filter != nil && filter.Namespace != nil {
		if _, ok := ignoredNamespacesMap[*filter.Namespace]; ok {
			return fmt.Errorf("namespace %s is configured to be ignored by odigos", *filter.Namespace)
		}
	}

	if filter != nil && filter.Kind != nil && filter.Name != nil && filter.Namespace != nil &&
		*filter.Kind != "" && *filter.Name != "" && *filter.Namespace != "" {

		l.workloadFilter = &WorkloadFilter{
			SingleWorkload: &WorkloadFilterSingleWorkload{
				WorkloadKind: k8sconsts.WorkloadKind(*filter.Kind),
				WorkloadName: *filter.Name,
				Namespace:    *filter.Namespace,
			},
			NamespaceString:   *filter.Namespace,
			IgnoredNamespaces: ignoredNamespacesMap,
		}
	} else if filter != nil && filter.Namespace != nil && *filter.Namespace != "" {
		l.workloadFilter = &WorkloadFilter{
			SingleNamespace: &WorkloadFilterSingleNamespace{
				Namespace: *filter.Namespace,
			},
			NamespaceString:   *filter.Namespace,
			IgnoredNamespaces: ignoredNamespacesMap,
		}
	} else {
		l.workloadFilter = &WorkloadFilter{
			ClusterWide:       &WorkloadFilterClusterWide{},
			NamespaceString:   "",
			IgnoredNamespaces: ignoredNamespacesMap,
		}
	}

	filterMarkedForInstrumentation := filter != nil && filter.MarkedForInstrumentation != nil && *filter.MarkedForInstrumentation

	var ids []model.K8sWorkloadID
	if filterMarkedForInstrumentation {
		l.instrumentationConfigMutex.Lock()
		defer l.instrumentationConfigMutex.Unlock()
		if err := l.loadInstrumentationConfigs(ctx); err != nil {
			return err
		}
		ids = make([]model.K8sWorkloadID, 0, len(l.instrumentationConfigs))
		for sourceId := range l.instrumentationConfigs {
			ids = append(ids, sourceId)
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

		allWorkloads := make(map[model.K8sWorkloadID]struct{})
		for workloadId, source := range l.workloadSources {
			if source.Spec.MatchWorkloadNameAsRegex {
				allWorkloads[workloadId] = struct{}{}
			}
		}
		for workloadId := range l.workloadManifests {
			allWorkloads[workloadId] = struct{}{}
		}
		ids = make([]model.K8sWorkloadID, 0, len(allWorkloads))
		for sourceId := range allWorkloads {
			ids = append(ids, sourceId)
		}
	}

	l.publishSnapshot(ids)
	return nil
}

func (l *Loaders) SetWorkloadIdsDirect(ctx context.Context, ids []model.K8sWorkloadID) error {
	if err := l.LoadConfig(ctx); err != nil {
		return err
	}
	l.publishSnapshot(ids)
	return nil
}

func getEnvValueFromManifest(envVarManifest []corev1.EnvVar, envVarName string) *string {
	if envVarManifest == nil {
		return nil
	}
	for _, envVar := range envVarManifest {
		if envVar.Name == envVarName {
			return &envVar.Value
		}
	}
	return nil
}

func getOdigosInstrumentationDeviceName(resources corev1.ResourceList) *string {
	for resourceName := range resources {
		resourceNameStr := string(resourceName)
		if strings.HasPrefix(resourceNameStr, common.OdigosResourceNamespace) {
			return &resourceNameStr
		}
	}
	return nil
}

// it would be better to set this on the distro manifest itself,
// but ui is not aware of the enterprise distros, so doing it manually for now.
func isDistroExpectingInstrumentationInstances(otelDistroName *string) bool {
	if otelDistroName == nil {
		return false
	}

	switch *otelDistroName {
	case "golang-community",
		"nodejs-community",
		"python-community",
		"golang-enterprise",
		"java-ebpf-instrumentations",
		"java-enterprise",
		"mysql-enterprise",
		"nodejs-enterprise",
		"python-enterprise":
		return true
	default:
		return false
	}
}

func getContainerStatus(pod *corev1.Pod, containerName string) *corev1.ContainerStatus {
	for i := range pod.Status.ContainerStatuses {
		containerStatus := &pod.Status.ContainerStatuses[i]
		if containerStatus.Name == containerName {
			return containerStatus
		}
	}
	for i := range pod.Status.InitContainerStatuses {
		containerStatus := &pod.Status.InitContainerStatuses[i]
		if containerStatus.Name == containerName {
			return containerStatus
		}
	}
	return nil
}
