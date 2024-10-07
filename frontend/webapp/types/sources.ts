import { Condition } from './common';

export enum SourceSortOptions {
  NAME = 'name',
  KIND = 'kind',
  NAMESPACE = 'namespace',
  LANGUAGE = 'language',
}

export enum K8SSourceTypes {
  DEPLOYMENT = 'deployment',
  STATEFUL_SET = 'statefulset',
  DAEMON_SET = 'daemonset',
}

export interface ManagedSource {
  kind: string;
  name: string;
  namespace: string;
  reported_name?: string;
  number_of_running_instances: number;
  instrumentation_config: Array<InstrumentationConfig>;
  instrumented_application_details: {
    instrumentation_options: Array<InstrumentationConfig>;
    conditions: Array<Condition>;
    languages: Array<{
      container_name: string;
      language: string;
    }> | null;
  };
}

export interface Namespace {
  name: string;
  selected: boolean;
  totalApps: number;
}

export interface SourceConfig {
  app_instrumentation_labeled: boolean;
  instances: number;
  instrumentation_effective: boolean;
  kind: string;
  name: string;
  ns_instrumentation_labeled: boolean;
  selected?: boolean;
}

export type NamespaceConfiguration = {
  selected_all: boolean;
  future_selected: boolean;
  objects: SourceConfig[];
};

// Type for the overall structure which has "default" as a key
export type SelectedSourcesConfiguration = {
  [key: string]: NamespaceConfiguration;
};

export interface SelectedSources {
  [key: string]: {
    objects: {
      name: string;
      selected: boolean;
      kind: string;
      app_instrumentation_labeled: boolean | null;
      ns_instrumentation_labeled: boolean | null;
      instrumentation_effective: boolean | null;
      instances: number;
    };
    selected_all: boolean;
    future_selected: boolean;
  };
}

export type SourceContainer = {
  containerName: string;
  language: string;
};

export type K8sActualSource = {
  name: string;
  kind: string;
  namespace: string;
  numberOfInstances: number;
  selected?: boolean;
  instrumentedApplicationDetails: {
    containers: Array<SourceContainer>;
    conditions: Array<Condition>;
  };
};
export interface InstrumentationConfig {
  optionKey: string;
  optionValueBoolean?: boolean;
  spanKind: string;
  instrumentationLibraries: InstrumentationConfigLibrary[];
}

export interface InstrumentationConfigLibrary {
  instrumentationLibraryName: string;
  language: string;
  selected?: boolean;
}

export type WorkloadId = {
  kind: string;
  name: string;
  namespace: string;
};
