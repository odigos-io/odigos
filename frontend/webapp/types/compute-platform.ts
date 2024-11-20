import { ActionData } from './actions';
import { K8sActualSource } from './sources';
import { ActualDestination } from './destinations';
import { InstrumentationRuleSpec } from './instrumentation-rules';

export type K8sActualNamespace = {
  name: string;
  k8sActualSources?: K8sActualSource[];
};

type ComputePlatformData = {
  id: string;
  name: string;
  computePlatformType: string;
  k8sActualNamespace?: K8sActualNamespace;
  k8sActualNamespaces: K8sActualNamespace[];
  actions: ActionData[];
  k8sActualSources: K8sActualSource[];
  destinations: ActualDestination[];
  instrumentationRules: InstrumentationRuleSpec[];
};

export type ComputePlatform = {
  computePlatform: ComputePlatformData;
};
