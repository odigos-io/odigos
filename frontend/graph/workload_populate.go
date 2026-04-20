package graph

import (
	"context"
	"sync"

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
