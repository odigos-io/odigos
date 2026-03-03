import { type DesiredConditionStatus, type DesiredStateProgress, type WorkloadId } from '@odigos/ui-kit/types';

export interface K8sWorkloadRuntimeInfo {
  language: string;
  runtimeVersion: string | null;
}

export interface K8sWorkloadAgentEnabled {
  agentEnabled: boolean;
  agentEnabledStatus: { message: string };
  otelDistroName: string | null;
}

export interface K8sWorkloadContainerResponse {
  containerName: string;
  runtimeInfo: K8sWorkloadRuntimeInfo | null;
  agentEnabled: K8sWorkloadAgentEnabled | null;
  overrides: { containerName: string } | null;
}

export interface K8sWorkloadConditions {
  runtimeDetection: DesiredConditionStatus | null;
  agentInjectionEnabled: DesiredConditionStatus | null;
  rollout: DesiredConditionStatus | null;
  agentInjected: DesiredConditionStatus | null;
  processesAgentHealth: DesiredConditionStatus | null;
  expectingTelemetry: DesiredConditionStatus | null;
}

export interface WorkloadResponse {
  id: WorkloadId;
  serviceName: string | null;
  dataStreamNames: string[];
  numberOfInstances: number | null;
  markedForInstrumentation: { markedForInstrumentation: boolean | null };
  runtimeInfo: { detectedLanguages: string[] | null } | null;
  containers: K8sWorkloadContainerResponse[] | null;
  conditions: K8sWorkloadConditions | null;
  workloadOdigosHealthStatus: DesiredConditionStatus | null;
  podsAgentInjectionStatus: DesiredConditionStatus | null;
  rollbackOccurred: boolean;
}
