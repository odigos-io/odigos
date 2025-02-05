import { type FetchedSource } from './sources';
import { type FetchedAction } from './actions';
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
  k8sActualSources?: FetchedSource[];
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
    sources?: PaginatedData<FetchedSource>; // fetched is already "mapped", except for conditions (which are mapped by it's own component)
    destinations?: Destination[]; // fetched is already "mapped"
    actions?: FetchedAction[]; // should map from "FetchedAction" to "Action" in get-query
    instrumentationRules?: InstrumentationRuleSpec[]; // should map from "InstrumentationRuleSpec" to "InstrumentationRuleSpecMapped" in get-query
  };
}
