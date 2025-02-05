interface EntityProperty {
  name: string;
  value: string;
  status?: string;
  explain?: string;
}

interface InstrumentationSourcesAnalyze {
  instrumented: EntityProperty;
  workload?: EntityProperty;
  namespace?: EntityProperty;
  instrumentedText?: EntityProperty;
}

interface ContainerRuntimeInfoAnalyze {
  containerName: EntityProperty;
  language: EntityProperty;
  runtimeVersion: EntityProperty;
  envVars: EntityProperty[];
}

interface RuntimeInfoAnalyze {
  generation: EntityProperty;
  containers: ContainerRuntimeInfoAnalyze[];
}

interface ContainerAgentConfigAnalyze {
  containerName: EntityProperty;
  agentEnabled: EntityProperty;
  reason?: EntityProperty;
  message?: EntityProperty;
  otelDistroName?: EntityProperty;
}

interface OtelAgentsAnalyze {
  created: EntityProperty;
  createTime?: EntityProperty;
  containers: ContainerAgentConfigAnalyze[];
}

interface InstrumentationInstanceAnalyze {
  healthy: EntityProperty;
  message?: EntityProperty;
  identifyingAttributes: EntityProperty[];
}

interface PodContainerAnalyze {
  containerName: EntityProperty;
  actualDevices: EntityProperty;
  instrumentationInstances: InstrumentationInstanceAnalyze[];
}

interface PodAnalyze {
  podName: EntityProperty;
  nodeName: EntityProperty;
  phase: EntityProperty;
  containers: PodContainerAnalyze[];
}

interface SourceAnalyze {
  name: EntityProperty;
  kind: EntityProperty;
  namespace: EntityProperty;

  sourceObjects?: InstrumentationSourcesAnalyze;
  runtimeInfo?: RuntimeInfoAnalyze;
  otelAgents?: OtelAgentsAnalyze;

  totalPods: number;
  podsPhasesCount: string;
  pods: PodAnalyze[];
}

interface ClusterCollectorAnalyze {
  enabled: EntityProperty;
  collectorGroup: EntityProperty;
  deployed?: EntityProperty;
  deployedError?: EntityProperty;
  collectorReady?: EntityProperty;
  deploymentCreated: EntityProperty;
  expectedReplicas?: EntityProperty;
  healthyReplicas?: EntityProperty;
  failedReplicas?: EntityProperty;
  failedReplicasReason?: EntityProperty;
}

interface NodeCollectorAnalyze {
  enabled: EntityProperty;
  collectorGroup: EntityProperty;
  deployed?: EntityProperty;
  deployedError?: EntityProperty;
  collectorReady?: EntityProperty;
  daemonSet: EntityProperty;
  desiredNodes?: EntityProperty;
  currentNodes?: EntityProperty;
  updatedNodes?: EntityProperty;
  availableNodes?: EntityProperty;
}

interface OdigosAnalyze {
  odigosVersion: EntityProperty;
  kubernetesVersion: EntityProperty;
  tier: EntityProperty;
  installationMethod: EntityProperty;
  numberOfDestinations: number;
  numberOfSources: number;
  clusterCollector: ClusterCollectorAnalyze;
  nodeCollector: NodeCollectorAnalyze;
  isSettled: boolean;
  hasErrors: boolean;
}

export interface DescribeSource {
  describeSource: SourceAnalyze;
}

export interface DescribeOdigos {
  describeOdigos: OdigosAnalyze;
}
