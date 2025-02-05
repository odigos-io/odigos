import { type FetchedAction } from './actions';
import { type K8sActualSource } from './sources';
import { type Destination } from '@odigos/ui-containers';
import { type InstrumentationRuleSpec } from './instrumentation-rules';

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
    destinations?: Destination[]; // fetched is already "mapped"
    actions?: FetchedAction[]; // should map from "FetchedAction" to "Action" in get-query
    instrumentationRules?: InstrumentationRuleSpec[]; // should map from "InstrumentationRuleSpec" to "InstrumentationRuleSpecMapped" in get-query
  };
}
