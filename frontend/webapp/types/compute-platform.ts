import type { K8sActualSource } from './sources';
import type { ActualDestination } from './destinations';
import type { ActionData, ActionDataParsed } from './actions';
import type { InstrumentationRuleSpec } from './instrumentation-rules';

export type K8sActualNamespace = {
  name: string;
  k8sActualSources?: K8sActualSource[];
};

interface ComputePlatformData {
  id: string;
  name: string;
  computePlatformType: string;
  k8sActualNamespace?: K8sActualNamespace;
  k8sActualNamespaces: K8sActualNamespace[];
  actions: ActionData[];
  k8sActualSources: K8sActualSource[];
  destinations: ActualDestination[];
  instrumentationRules: InstrumentationRuleSpec[];
}

export type ComputePlatform = {
  computePlatform: ComputePlatformData;
};

interface ComputePlatformDataMapped extends ComputePlatformData {
  actions: ActionDataParsed[];
  instrumentationRules: InstrumentationRuleSpec[];
}

export type ComputePlatformMapped = {
  computePlatform: ComputePlatformDataMapped;
};
