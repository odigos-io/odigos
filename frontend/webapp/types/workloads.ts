import { type DesiredConditionStatus, type SourceContainer, type WorkloadId } from '@odigos/ui-kit/types';

// Mirrors the GraphQL `K8sWorkloadConditions` shape. Stays a webapp-local type
// because the ui-kit's `Source.conditions` is a flat `Condition[]` array, so
// this shape needs to be flattened during the workload→source mapping.
export interface K8sWorkloadConditions {
  runtimeDetection: DesiredConditionStatus | null;
  agentInjectionEnabled: DesiredConditionStatus | null;
  rollout: DesiredConditionStatus | null;
  agentInjected: DesiredConditionStatus | null;
  processesAgentHealth: DesiredConditionStatus | null;
  expectingTelemetry: DesiredConditionStatus | null;
}

// The shape of `K8sWorkload` we actually consume. The container shape mirrors
// the ui-kit's `SourceContainer` exactly so we can pass it through the mapper
// without any per-field copying.
export interface WorkloadResponse {
  id: WorkloadId;
  serviceName: string | null;
  dataStreamNames: string[];
  numberOfInstances: number | null;
  markedForInstrumentation: { markedForInstrumentation: boolean | null };
  runtimeInfo: { detectedLanguages: string[] | null } | null;
  containers: SourceContainer[] | null;
  conditions: K8sWorkloadConditions | null;
  workloadOdigosHealthStatus: DesiredConditionStatus | null;
  podsAgentInjectionStatus: DesiredConditionStatus | null;
  rollbackOccurred: boolean;
}
