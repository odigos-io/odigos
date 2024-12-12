interface EntityProperty {
  name: string;
  value: string;
  status?: string;
  explain?: string;
}

interface InstrumentationLabelsAnalyze {
  instrumented: EntityProperty;
  workload?: EntityProperty;
  namespace?: EntityProperty;
  instrumentedText?: EntityProperty;
}

interface InstrumentationConfigAnalyze {
  created: EntityProperty;
  createTime?: EntityProperty;
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

interface InstrumentedApplicationAnalyze {
  created: EntityProperty;
  createTime?: EntityProperty;
  containers: ContainerRuntimeInfoAnalyze[];
}

interface ContainerWorkloadManifestAnalyze {
  containerName: EntityProperty;
  devices: EntityProperty;
  originalEnv: EntityProperty[];
}

interface InstrumentationDeviceAnalyze {
  statusText: EntityProperty;
  containers: ContainerWorkloadManifestAnalyze[];
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
  labels: InstrumentationLabelsAnalyze;

  instrumentationConfig: InstrumentationConfigAnalyze;
  runtimeInfo?: RuntimeInfoAnalyze;
  instrumentedApplication: InstrumentedApplicationAnalyze;
  instrumentationDevice: InstrumentationDeviceAnalyze;

  totalPods: number;
  podsPhasesCount: string;
  pods: PodAnalyze[];
}

export interface DescribeSource {
  describeSource: SourceAnalyze;
}
