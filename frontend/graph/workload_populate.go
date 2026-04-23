package graph

import (
	"context"
	"sync"

	"github.com/99designs/gqlgen/graphql"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/frontend/graph/loaders"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/graph/status"
	"github.com/odigos-io/odigos/frontend/services"
	frontendcommon "github.com/odigos-io/odigos/frontend/services/common"
	sourceutils "github.com/odigos-io/odigos/k8sutils/pkg/source"
)

// Serializes heavy workload queries so concurrent requests don't double memory usage.
var heavyWorkloadQueryMu sync.Mutex

// isNamespacesWorkloadsSelected inspects the current query selection and returns
// true if the caller requested the `workloads` sub-field under `namespaces`.
// Used to gate the expensive cluster-wide workload load for the lightweight
// `GET_NAMESPACES` query which only needs namespace metadata.
func isNamespacesWorkloadsSelected(ctx context.Context) bool {
	for _, field := range graphql.CollectFieldsCtx(ctx, nil) {
		if field.Name == "workloads" {
			return true
		}
	}
	return false
}

// populateNamespaceQueryWorkloadFields pre-computes the subset of K8sWorkload
// fields needed by the `namespaces { workloads { ... } }` query path:
// markedForInstrumentation, dataStreamNames, and numberOfInstances.
//
// This is called sequentially from queryResolver.Namespaces (under
// heavyWorkloadQueryMu) so gqlgen's per-field goroutine fan-out across
// 10K+ workloads short-circuits on pre-populated values instead of each
// spawning mutex-acquiring resolver goroutines.
//
// Intentionally slim: does NOT trigger loadInstrumentationConfigs or
// loadWorkloadPods, keeping this path much cheaper than populateWorkloadFields.
func populateNamespaceQueryWorkloadFields(ctx context.Context, l *loaders.Loaders, w *model.K8sWorkload) {
	id := *w.ID

	if sources, err := l.GetSources(ctx, id); err == nil && sources != nil {
		enabled, reason, rerr := sourceutils.IsObjectInstrumentedBySource(ctx, sources, nil)
		if rerr == nil {
			var marked *bool
			if enabled {
				marked = &enabled
			} else if reason.Reason == string("WorkloadSourceDisabled") {
				marked = &enabled
			}
			w.MarkedForInstrumentation = &model.K8sWorkloadMarkedForInstrumentation{
				MarkedForInstrumentation: marked,
				DecisionEnum:             string(reason.Reason),
				Message:                  reason.Message,
			}
		}

		ptrNames := services.ExtractDataStreamsFromSource(sources.Workload, sources.Namespace)
		names := make([]string, len(ptrNames))
		for i, p := range ptrNames {
			names[i] = *p
		}
		w.DataStreamNames = names
	}

	if manifest, err := l.GetWorkloadManifest(ctx, id); err == nil && manifest != nil {
		count := int(manifest.AvailableReplicas)
		w.NumberOfInstances = &count
	}
}

// populateWorkloadFields pre-computes all resolver fields for a workload
// sequentially. This is called from the Workloads batch resolver to avoid
// gqlgen spawning a goroutine per field per workload.
// Errors are logged but don't fail the entire batch — individual workloads
// get nil/zero values for fields that fail to resolve.
func (r *queryResolver) populateWorkloadFields(ctx context.Context, l *loaders.Loaders, w *model.K8sWorkload) {
	id := *w.ID

	ic, _ := l.GetInstrumentationConfig(ctx, id)

	// serviceName
	if ic != nil {
		w.ServiceName = &ic.Spec.ServiceName
	}

	// rollbackOccurred
	if ic != nil {
		w.RollbackOccurred = ic.Status.RollbackOccurred
	}

	// markedForInstrumentation
	if sources, err := l.GetSources(ctx, id); err == nil {
		enabled, reason, err := sourceutils.IsObjectInstrumentedBySource(ctx, sources, nil)
		if err == nil {
			var marked *bool
			if enabled {
				marked = &enabled
			} else if reason.Reason == string("WorkloadSourceDisabled") {
				marked = &enabled
			}
			w.MarkedForInstrumentation = &model.K8sWorkloadMarkedForInstrumentation{
				MarkedForInstrumentation: marked,
				DecisionEnum:             string(reason.Reason),
				Message:                  reason.Message,
			}
		}
	}

	// dataStreamNames
	if sources, err := l.GetSources(ctx, id); err == nil {
		ptrNames := services.ExtractDataStreamsFromSource(sources.Workload, sources.Namespace)
		names := make([]string, len(ptrNames))
		for i, p := range ptrNames {
			names[i] = *p
		}
		w.DataStreamNames = names
	}

	// numberOfInstances
	if manifest, err := l.GetWorkloadManifest(ctx, id); err == nil && manifest != nil {
		count := int(manifest.AvailableReplicas)
		w.NumberOfInstances = &count
	}

	// runtimeInfo
	if ic != nil {
		completed := len(ic.Status.RuntimeDetailsByContainer) > 0
		uniqueLanguages := make(map[common.ProgrammingLanguage]struct{})
		containers := make([]*model.K8sWorkloadRuntimeInfoContainer, len(ic.Status.RuntimeDetailsByContainer))
		for i, container := range ic.Status.RuntimeDetailsByContainer {
			containers[i] = runtimeDetailsContainersToModel(&container)
			_, ignored := l.GetIgnoredContainers()[container.ContainerName]
			if container.Language != common.UnknownProgrammingLanguage && !ignored {
				uniqueLanguages[container.Language] = struct{}{}
			}
		}
		var detectedLanguages []model.ProgrammingLanguage
		if completed {
			detectedLanguages = make([]model.ProgrammingLanguage, 0, len(uniqueLanguages))
			for language := range uniqueLanguages {
				detectedLanguages = append(detectedLanguages, model.ProgrammingLanguage(language))
			}
		}
		w.RuntimeInfo = &model.K8sWorkloadRuntimeInfo{
			Completed:         completed,
			CompletedStatus:   status.CalculateRuntimeInspectionStatus(ic),
			DetectedLanguages: detectedLanguages,
			Containers:        containers,
		}
	}

	// containers
	if ic != nil {
		containerByName := make(map[string]*model.K8sWorkloadContainer)
		for i := range ic.Spec.Containers {
			container := &ic.Spec.Containers[i]
			if _, ok := containerByName[container.ContainerName]; !ok {
				containerByName[container.ContainerName] = &model.K8sWorkloadContainer{
					ContainerName: container.ContainerName,
				}
			}
			containerByName[container.ContainerName].AgentEnabled = agentEnabledContainersToModel(container)
			containerByName[container.ContainerName].AgentConfig = containerAgentConfigToAgentConfigModel(container)
		}
		for _, container := range ic.Status.RuntimeDetailsByContainer {
			if _, ok := containerByName[container.ContainerName]; !ok {
				containerByName[container.ContainerName] = &model.K8sWorkloadContainer{
					ContainerName: container.ContainerName,
				}
			}
			containerByName[container.ContainerName].RuntimeInfo = runtimeDetailsContainersToModel(&container)
		}
		for _, container := range ic.Spec.ContainersOverrides {
			if _, ok := containerByName[container.ContainerName]; !ok {
				containerByName[container.ContainerName] = &model.K8sWorkloadContainer{
					ContainerName: container.ContainerName,
				}
			}
			overrides := &model.K8sWorkloadContainerOverrides{
				ContainerName: container.ContainerName,
			}
			if container.RuntimeInfo != nil {
				overrides.RuntimeInfo = runtimeDetailsContainersToModel(container.RuntimeInfo)
			}
			containerByName[container.ContainerName].Overrides = overrides
		}
		w.Containers = make([]*model.K8sWorkloadContainer, 0, len(containerByName))
		for _, container := range containerByName {
			w.Containers = append(w.Containers, container)
		}
	}

	// Pod-dependent fields: conditions, workloadOdigosHealthStatus, podsAgentInjectionStatus.
	// Pre-computed here (not left for lazy resolvers) to avoid 30K goroutines.
	// CachedPods are loaded once and shared across all workloads via the Loaders cache.
	pods, _ := l.GetWorkloadPods(ctx, id)

	var runtimeDetection, agentInjectionEnabled, rolloutStatus, agentInjected, processesHealth, expectingTelemetry *model.DesiredConditionStatus

	if ic != nil {
		runtimeDetection = status.CalculateRuntimeInspectionStatus(ic)
		agentInjectionEnabled = status.CalculateAgentInjectionEnabledStatus(ic)
		rolloutStatus = status.CalculateRolloutStatus(ic)
	}
	agentInjected = status.CalculateAgentInjectedStatus(ic, pods)
	containerNames := getContainerNamesWithOptionalPodManifestInjection(ic)
	processesHealth, _ = aggregateProcessesHealthForWorkload(ctx, &id, containerNames)

	var totalDataSent *int
	if workloadMetrics, ok := r.MetricsConsumer.GetSingleSourceMetrics(frontendcommon.SourceID{
		Namespace: id.Namespace,
		Kind:      k8sconsts.WorkloadKind(id.Kind),
		Name:      id.Name,
	}); ok {
		tds := int(workloadMetrics.TotalDataSent())
		totalDataSent = &tds
	}
	telemetryMetrics := status.CalculateExpectingTelemetryStatus(ic, pods, totalDataSent)
	expectingTelemetry = telemetryMetrics.TelemetryObservedStatus

	if ic != nil {
		w.Conditions = &model.K8sWorkloadConditions{
			RuntimeDetection:      runtimeDetection,
			AgentInjectionEnabled: agentInjectionEnabled,
			Rollout:               rolloutStatus,
			AgentInjected:         agentInjected,
			ProcessesAgentHealth:  processesHealth,
			ExpectingTelemetry:    expectingTelemetry,
		}
	}

	w.PodsAgentInjectionStatus = agentInjected

	healthConditions := make([]*model.DesiredConditionStatus, 0, 6)
	if ic != nil {
		healthConditions = append(healthConditions, runtimeDetection, agentInjectionEnabled, rolloutStatus)
	} else {
		reasonStr := string(status.WorkloadOdigosHealthStatusReasonDisabled)
		healthConditions = append(healthConditions, &model.DesiredConditionStatus{
			Name: status.WorkloadOdigosHealthStatus, Status: model.DesiredStateProgressDisabled,
			ReasonEnum: &reasonStr, Message: "workload is not marked for instrumentation",
		})
	}
	healthConditions = append(healthConditions, agentInjected, processesHealth, expectingTelemetry)

	mostSevere := aggregateConditionsBySeverity(healthConditions)
	if mostSevere == nil {
		mostSevere = &model.DesiredConditionStatus{Name: status.WorkloadOdigosHealthStatus, Status: model.DesiredStateProgressUnknown}
	} else if mostSevere.Status == model.DesiredStateProgressSuccess {
		reasonStr := string(status.WorkloadOdigosHealthStatusReasonSuccessAndEmittingTelemetry)
		mostSevere = &model.DesiredConditionStatus{
			Name: status.WorkloadOdigosHealthStatus, Status: model.DesiredStateProgressSuccess,
			ReasonEnum: &reasonStr, Message: "source is instrumented, healthy and telemetry has been observed",
		}
	}
	w.WorkloadOdigosHealthStatus = mostSevere
}
