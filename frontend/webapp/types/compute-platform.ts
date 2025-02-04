import type { ActionData } from './actions';
import type { K8sActualSource } from './sources';
import type { ActualDestination } from './destinations';
import type { InstrumentationRuleSpec } from './instrumentation-rules';

export interface TokenPayload {
  token: string;
  name: string;
  issuedAt: number;
  expiresAt: number;
}

export interface K8sActualNamespace {
  name: string;
  selected: boolean;
  k8sActualSources?: K8sActualSource[];
}

interface PaginatedData<T = any> {
  nextPage: string;
  items: T[];
}

export interface ComputePlatform {
  computePlatform: {
    computePlatformType?: string;
    apiTokens?: TokenPayload[];
    k8sActualNamespaces?: K8sActualNamespace[];
    k8sActualNamespace?: K8sActualNamespace;
    sources?: PaginatedData<K8sActualSource>;
    destinations?: ActualDestination[];
    actions?: ActionData[]; // should map from "ActionData" to "ActionDataParsed" in get-query
    instrumentationRules?: InstrumentationRuleSpec[]; // should map from "InstrumentationRuleSpec" to "InstrumentationRuleSpecMapped" in get-query
  };
}
