import type { FetchedSource } from './sources';
import type { FetchedAction } from './actions';
import type { FetchedNamespace } from './namespace';
import type { InstrumentationRuleSpec } from './instrumentation-rules';
import type { Destination, TokenPayload } from '@odigos/ui-utils';

interface PaginatedData<T = any> {
  nextPage: string;
  items: T[];
}

export interface ComputePlatform {
  computePlatform: {
    computePlatformType?: string;
    apiTokens?: TokenPayload[];
    k8sActualNamespaces?: FetchedNamespace[];
    k8sActualNamespace?: FetchedNamespace;
    sources?: PaginatedData<FetchedSource>; // fetched is already "mapped", except for conditions (which are mapped by it's own component)
    destinations?: Destination[]; // fetched is already "mapped"
    actions?: FetchedAction[]; // should map from "FetchedAction" to "Action" in get-query
    instrumentationRules?: InstrumentationRuleSpec[]; // should map from "InstrumentationRuleSpec" to "InstrumentationRuleSpecMapped" in get-query
  };
}
