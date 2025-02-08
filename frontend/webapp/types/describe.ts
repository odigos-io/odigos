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

export interface DescribeSource {
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
